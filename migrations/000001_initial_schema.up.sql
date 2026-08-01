CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE countries (
    code char(2) PRIMARY KEY,
    name text NOT NULL,
    flag_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT countries_code_uppercase CHECK (code = upper(code))
);

CREATE TABLE leagues (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    external_id text NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    type text NOT NULL DEFAULT 'league',
    country_code char(2) REFERENCES countries(code) ON UPDATE CASCADE ON DELETE SET NULL,
    logo_url text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id),
    UNIQUE (slug),
    CONSTRAINT leagues_type_valid CHECK (type IN ('league', 'cup', 'friendly'))
);

CREATE TABLE seasons (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    league_id uuid NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    provider text NOT NULL,
    external_id text NOT NULL,
    name text NOT NULL,
    starts_on date NOT NULL,
    ends_on date NOT NULL,
    is_current boolean NOT NULL DEFAULT false,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id),
    CONSTRAINT seasons_date_order CHECK (ends_on >= starts_on)
);
CREATE UNIQUE INDEX seasons_one_current_per_league_idx ON seasons (league_id) WHERE is_current;

CREATE TABLE venues (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    external_id text NOT NULL,
    name text NOT NULL,
    city text,
    country_code char(2) REFERENCES countries(code) ON UPDATE CASCADE ON DELETE SET NULL,
    address text,
    latitude double precision,
    longitude double precision,
    capacity integer,
    surface text,
    image_url text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id),
    CONSTRAINT venues_capacity_positive CHECK (capacity IS NULL OR capacity >= 0),
    CONSTRAINT venues_latitude_valid CHECK (latitude IS NULL OR latitude BETWEEN -90 AND 90),
    CONSTRAINT venues_longitude_valid CHECK (longitude IS NULL OR longitude BETWEEN -180 AND 180)
);

CREATE TABLE teams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    external_id text NOT NULL,
    name text NOT NULL,
    short_name text,
    code text,
    country_code char(2) REFERENCES countries(code) ON UPDATE CASCADE ON DELETE SET NULL,
    founded_year integer,
    national boolean NOT NULL DEFAULT false,
    logo_url text,
    venue_id uuid REFERENCES venues(id) ON DELETE SET NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id),
    CONSTRAINT teams_founded_year_valid CHECK (founded_year IS NULL OR founded_year BETWEEN 1800 AND 2200)
);
CREATE INDEX teams_country_name_idx ON teams (country_code, name, id);

CREATE TABLE people (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    external_id text NOT NULL,
    display_name text NOT NULL,
    first_name text,
    last_name text,
    birth_date date,
    country_code char(2) REFERENCES countries(code) ON UPDATE CASCADE ON DELETE SET NULL,
    photo_url text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id)
);
CREATE INDEX people_display_name_idx ON people (display_name, id);

CREATE TABLE players (
    person_id uuid PRIMARY KEY REFERENCES people(id) ON DELETE CASCADE,
    position text,
    preferred_foot text,
    height_cm smallint,
    weight_kg numeric(5,2),
    CONSTRAINT players_foot_valid CHECK (preferred_foot IS NULL OR preferred_foot IN ('left', 'right', 'both')),
    CONSTRAINT players_height_valid CHECK (height_cm IS NULL OR height_cm BETWEEN 100 AND 250),
    CONSTRAINT players_weight_valid CHECK (weight_kg IS NULL OR weight_kg BETWEEN 30 AND 250)
);

CREATE TABLE coaches (
    person_id uuid PRIMARY KEY REFERENCES people(id) ON DELETE CASCADE
);

CREATE TABLE team_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    person_id uuid NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    role text NOT NULL,
    shirt_number smallint,
    starts_on date,
    ends_on date,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT team_memberships_role_valid CHECK (role IN ('player', 'head_coach', 'assistant_coach', 'staff')),
    CONSTRAINT team_memberships_shirt_valid CHECK (shirt_number IS NULL OR shirt_number BETWEEN 0 AND 99),
    CONSTRAINT team_memberships_date_order CHECK (ends_on IS NULL OR starts_on IS NULL OR ends_on >= starts_on)
);
CREATE INDEX team_memberships_active_team_idx ON team_memberships (team_id, role, person_id) WHERE ends_on IS NULL;
CREATE INDEX team_memberships_person_idx ON team_memberships (person_id, starts_on DESC);

CREATE TABLE matches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider text NOT NULL,
    external_id text NOT NULL,
    league_id uuid NOT NULL REFERENCES leagues(id),
    season_id uuid NOT NULL REFERENCES seasons(id),
    stage text,
    round text,
    leg smallint NOT NULL DEFAULT 1,
    kickoff_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'scheduled',
    period text,
    elapsed_minute smallint,
    venue_id uuid REFERENCES venues(id) ON DELETE SET NULL,
    home_team_id uuid NOT NULL REFERENCES teams(id),
    away_team_id uuid NOT NULL REFERENCES teams(id),
    home_score smallint,
    away_score smallint,
    home_half_time_score smallint,
    away_half_time_score smallint,
    winner_team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
    version bigint NOT NULL DEFAULT 1,
    source_hash bytea NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, external_id),
    CONSTRAINT matches_teams_differ CHECK (home_team_id <> away_team_id),
    CONSTRAINT matches_leg_positive CHECK (leg > 0),
    CONSTRAINT matches_elapsed_valid CHECK (elapsed_minute IS NULL OR elapsed_minute BETWEEN 0 AND 200),
    CONSTRAINT matches_scores_valid CHECK (
        (home_score IS NULL OR home_score >= 0) AND
        (away_score IS NULL OR away_score >= 0) AND
        (home_half_time_score IS NULL OR home_half_time_score >= 0) AND
        (away_half_time_score IS NULL OR away_half_time_score >= 0)
    ),
    CONSTRAINT matches_winner_participates CHECK (winner_team_id IS NULL OR winner_team_id IN (home_team_id, away_team_id)),
    CONSTRAINT matches_status_valid CHECK (status IN ('scheduled', 'postponed', 'cancelled', 'live', 'suspended', 'finished', 'abandoned', 'awarded'))
);
CREATE INDEX matches_feed_idx ON matches (kickoff_at, id);
CREATE INDEX matches_league_feed_idx ON matches (league_id, kickoff_at, id);
CREATE INDEX matches_season_feed_idx ON matches (season_id, kickoff_at, id);
CREATE INDEX matches_home_team_feed_idx ON matches (home_team_id, kickoff_at DESC);
CREATE INDEX matches_away_team_feed_idx ON matches (away_team_id, kickoff_at DESC);
CREATE INDEX matches_live_idx ON matches (kickoff_at, id) WHERE status IN ('live', 'suspended');

CREATE TABLE match_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    provider text NOT NULL,
    external_id text,
    sequence integer NOT NULL,
    period text,
    minute smallint,
    stoppage_minute smallint,
    type text NOT NULL,
    team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
    primary_person_id uuid REFERENCES people(id) ON DELETE SET NULL,
    secondary_person_id uuid REFERENCES people(id) ON DELETE SET NULL,
    detail text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (match_id, sequence),
    CONSTRAINT match_events_sequence_positive CHECK (sequence > 0),
    CONSTRAINT match_events_minute_valid CHECK (minute IS NULL OR minute BETWEEN 0 AND 200),
    CONSTRAINT match_events_stoppage_valid CHECK (stoppage_minute IS NULL OR stoppage_minute BETWEEN 0 AND 99)
);
CREATE UNIQUE INDEX match_events_provider_external_idx ON match_events (match_id, provider, external_id) WHERE external_id IS NOT NULL;
CREATE INDEX match_events_match_sequence_idx ON match_events (match_id, sequence);

CREATE TABLE match_lineups (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    person_id uuid NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    position text,
    grid_position text,
    shirt_number smallint,
    is_starter boolean NOT NULL DEFAULT false,
    is_captain boolean NOT NULL DEFAULT false,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (match_id, team_id, person_id),
    CONSTRAINT match_lineups_shirt_valid CHECK (shirt_number IS NULL OR shirt_number BETWEEN 0 AND 99)
);

CREATE TABLE standings (
    season_id uuid NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    group_name text NOT NULL DEFAULT '',
    position smallint NOT NULL,
    played smallint NOT NULL DEFAULT 0,
    won smallint NOT NULL DEFAULT 0,
    drawn smallint NOT NULL DEFAULT 0,
    lost smallint NOT NULL DEFAULT 0,
    goals_for smallint NOT NULL DEFAULT 0,
    goals_against smallint NOT NULL DEFAULT 0,
    points smallint NOT NULL DEFAULT 0,
    form text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (season_id, team_id, group_name),
    CONSTRAINT standings_position_positive CHECK (position > 0),
    CONSTRAINT standings_counts_positive CHECK (played >= 0 AND won >= 0 AND drawn >= 0 AND lost >= 0 AND goals_for >= 0 AND goals_against >= 0)
);
CREATE INDEX standings_order_idx ON standings (season_id, group_name, position);

CREATE TABLE ingestion_cursors (
    provider text NOT NULL,
    stream text NOT NULL,
    cursor text,
    last_success_at timestamptz,
    last_error text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, stream)
);

CREATE TABLE outbox_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    attempts integer NOT NULL DEFAULT 0,
    last_error text
);
CREATE INDEX outbox_events_unpublished_idx ON outbox_events (occurred_at, id) WHERE published_at IS NULL;

CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER countries_touch_updated_at BEFORE UPDATE ON countries FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER leagues_touch_updated_at BEFORE UPDATE ON leagues FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER seasons_touch_updated_at BEFORE UPDATE ON seasons FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER venues_touch_updated_at BEFORE UPDATE ON venues FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER teams_touch_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER people_touch_updated_at BEFORE UPDATE ON people FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER team_memberships_touch_updated_at BEFORE UPDATE ON team_memberships FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
