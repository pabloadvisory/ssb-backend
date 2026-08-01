CREATE TABLE external_ids (
    entity_type text NOT NULL,
    entity_id uuid NOT NULL,
    provider text NOT NULL,
    external_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, entity_type, external_id),
    UNIQUE (entity_type, entity_id, provider),
    CONSTRAINT external_ids_entity_type_valid CHECK (
        entity_type IN ('league', 'season', 'venue', 'team', 'person', 'match', 'match_event')
    ),
    CONSTRAINT external_ids_provider_not_blank CHECK (btrim(provider) <> ''),
    CONSTRAINT external_ids_external_id_not_blank CHECK (btrim(external_id) <> '')
);
CREATE INDEX external_ids_entity_idx ON external_ids (entity_type, entity_id);

INSERT INTO external_ids (entity_type, entity_id, provider, external_id)
SELECT 'league', id, provider, external_id FROM leagues
UNION ALL SELECT 'season', id, provider, external_id FROM seasons
UNION ALL SELECT 'venue', id, provider, external_id FROM venues
UNION ALL SELECT 'team', id, provider, external_id FROM teams
UNION ALL SELECT 'person', id, provider, external_id FROM people
UNION ALL SELECT 'match', id, provider, external_id FROM matches
UNION ALL SELECT 'match_event', id, provider, external_id FROM match_events
WHERE external_id IS NOT NULL AND btrim(external_id) <> '';

CREATE TRIGGER external_ids_touch_updated_at
    BEFORE UPDATE ON external_ids FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE push_endpoint_tokens (
    endpoint_id uuid PRIMARY KEY REFERENCES push_endpoints(id) ON DELETE CASCADE,
    token text NOT NULL,
    token_hash bytea NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO push_endpoint_tokens (endpoint_id, token, token_hash)
SELECT id, token, token_hash FROM push_endpoints;
CREATE TRIGGER push_endpoint_tokens_touch_updated_at
    BEFORE UPDATE ON push_endpoint_tokens FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE push_endpoints
    DROP CONSTRAINT push_endpoints_token_hash_key,
    DROP COLUMN token,
    DROP COLUMN token_hash;

ALTER TABLE seasons
    ADD CONSTRAINT seasons_id_league_unique UNIQUE (id, league_id);
ALTER TABLE matches
    ADD CONSTRAINT matches_season_league_fk
    FOREIGN KEY (season_id, league_id) REFERENCES seasons (id, league_id);

ALTER TABLE leagues
    ADD COLUMN gender text NOT NULL DEFAULT 'men',
    ADD CONSTRAINT leagues_gender_valid CHECK (gender IN ('men', 'women', 'mixed'));

ALTER TABLE venues
    ADD COLUMN timezone text,
    ADD CONSTRAINT venues_timezone_not_blank CHECK (timezone IS NULL OR btrim(timezone) <> '');

ALTER TABLE players
    ADD COLUMN detailed_position text,
    ADD CONSTRAINT players_position_valid CHECK (
        position IS NULL OR position IN ('goalkeeper', 'defender', 'midfielder', 'forward')
    );

ALTER TABLE matches
    ADD COLUMN group_name text,
    ADD COLUMN round_sort integer,
    ADD COLUMN home_extra_time_score smallint,
    ADD COLUMN away_extra_time_score smallint,
    ADD COLUMN home_penalty_score smallint,
    ADD COLUMN away_penalty_score smallint,
    ADD COLUMN attendance integer,
    ADD COLUMN first_leg_match_id uuid REFERENCES matches(id) ON DELETE SET NULL,
    ADD CONSTRAINT matches_round_sort_positive CHECK (round_sort IS NULL OR round_sort > 0),
    ADD CONSTRAINT matches_attendance_positive CHECK (attendance IS NULL OR attendance >= 0),
    ADD CONSTRAINT matches_extra_time_scores_valid CHECK (
        (home_extra_time_score IS NULL OR home_extra_time_score >= 0) AND
        (away_extra_time_score IS NULL OR away_extra_time_score >= 0)
    ),
    ADD CONSTRAINT matches_penalty_scores_valid CHECK (
        (home_penalty_score IS NULL OR home_penalty_score >= 0) AND
        (away_penalty_score IS NULL OR away_penalty_score >= 0)
    ),
    ADD CONSTRAINT matches_first_leg_different CHECK (first_leg_match_id IS NULL OR first_leg_match_id <> id),
    ADD CONSTRAINT matches_period_valid CHECK (
        period IS NULL OR period IN (
            'first_half', 'half_time', 'second_half',
            'extra_time_first_half', 'extra_time_half_time', 'extra_time_second_half',
            'penalties', 'full_time'
        )
    );
CREATE INDEX matches_group_round_idx
    ON matches (season_id, group_name, round_sort, kickoff_at, id);
CREATE INDEX matches_first_leg_idx
    ON matches (first_leg_match_id) WHERE first_leg_match_id IS NOT NULL;

ALTER TABLE match_events
    ADD COLUMN home_score smallint,
    ADD COLUMN away_score smallint,
    ADD CONSTRAINT match_events_scores_valid CHECK (
        (home_score IS NULL OR home_score >= 0) AND
        (away_score IS NULL OR away_score >= 0)
    ),
    ADD CONSTRAINT match_events_type_valid CHECK (
        type IN (
            'kickoff', 'half_time', 'second_half_started',
            'extra_time_started', 'extra_time_half_time', 'penalties_started', 'full_time',
            'goal', 'own_goal', 'penalty_goal', 'penalty_missed',
            'yellow_card', 'second_yellow', 'red_card', 'substitution', 'var_decision',
            'match_suspended', 'match_resumed', 'match_cancelled'
        )
    );
COMMENT ON COLUMN match_events.primary_person_id IS
    'Primary actor. For substitutions this is the player entering the match.';
COMMENT ON COLUMN match_events.secondary_person_id IS
    'Secondary actor. For substitutions this is the player leaving the match.';

CREATE TABLE season_teams (
    season_id uuid NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    promoted boolean NOT NULL DEFAULT false,
    relegated boolean NOT NULL DEFAULT false,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (season_id, team_id),
    CONSTRAINT season_teams_movement_exclusive CHECK (NOT (promoted AND relegated))
);
CREATE INDEX season_teams_team_idx ON season_teams (team_id, season_id);
CREATE TRIGGER season_teams_touch_updated_at
    BEFORE UPDATE ON season_teams FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

INSERT INTO season_teams (season_id, team_id)
SELECT DISTINCT season_id, home_team_id FROM matches
UNION
SELECT DISTINCT season_id, away_team_id FROM matches
UNION
SELECT DISTINCT season_id, team_id FROM standings
ON CONFLICT (season_id, team_id) DO NOTHING;

CREATE TABLE match_team_info (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    formation text,
    coach_id uuid REFERENCES coaches(person_id) ON DELETE SET NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, team_id),
    CONSTRAINT match_team_info_formation_not_blank CHECK (formation IS NULL OR btrim(formation) <> '')
);
CREATE TRIGGER match_team_info_touch_updated_at
    BEFORE UPDATE ON match_team_info FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE match_officials (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    person_id uuid NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    role text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, person_id, role),
    CONSTRAINT match_officials_role_valid CHECK (
        role IN ('referee', 'assistant_referee', 'fourth_official', 'var', 'assistant_var')
    )
);
CREATE INDEX match_officials_person_idx ON match_officials (person_id, match_id);

CREATE TABLE player_match_statistics (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    person_id uuid NOT NULL REFERENCES players(person_id) ON DELETE CASCADE,
    started boolean NOT NULL DEFAULT false,
    minutes_played smallint NOT NULL DEFAULT 0,
    goals smallint NOT NULL DEFAULT 0,
    assists smallint NOT NULL DEFAULT 0,
    shots smallint NOT NULL DEFAULT 0,
    shots_on_target smallint NOT NULL DEFAULT 0,
    passes smallint NOT NULL DEFAULT 0,
    tackles smallint NOT NULL DEFAULT 0,
    saves smallint NOT NULL DEFAULT 0,
    yellow_cards smallint NOT NULL DEFAULT 0,
    red_cards smallint NOT NULL DEFAULT 0,
    rating numeric(4,2),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, person_id),
    CONSTRAINT player_match_statistics_minutes_valid CHECK (minutes_played BETWEEN 0 AND 200),
    CONSTRAINT player_match_statistics_counts_valid CHECK (
        goals >= 0 AND assists >= 0 AND shots >= 0 AND shots_on_target >= 0 AND
        passes >= 0 AND tackles >= 0 AND saves >= 0 AND yellow_cards >= 0 AND red_cards >= 0
    ),
    CONSTRAINT player_match_statistics_rating_valid CHECK (rating IS NULL OR rating BETWEEN 0 AND 10)
);
CREATE INDEX player_match_statistics_team_idx
    ON player_match_statistics (match_id, team_id, person_id);
CREATE TRIGGER player_match_statistics_touch_updated_at
    BEFORE UPDATE ON player_match_statistics FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

ALTER TABLE standings
    ADD COLUMN zone text,
    ADD COLUMN description text,
    ADD COLUMN home_played smallint,
    ADD COLUMN home_won smallint,
    ADD COLUMN home_drawn smallint,
    ADD COLUMN home_lost smallint,
    ADD COLUMN away_played smallint,
    ADD COLUMN away_won smallint,
    ADD COLUMN away_drawn smallint,
    ADD COLUMN away_lost smallint,
    ADD CONSTRAINT standings_split_counts_positive CHECK (
        (home_played IS NULL OR home_played >= 0) AND
        (home_won IS NULL OR home_won >= 0) AND
        (home_drawn IS NULL OR home_drawn >= 0) AND
        (home_lost IS NULL OR home_lost >= 0) AND
        (away_played IS NULL OR away_played >= 0) AND
        (away_won IS NULL OR away_won >= 0) AND
        (away_drawn IS NULL OR away_drawn >= 0) AND
        (away_lost IS NULL OR away_lost >= 0)
    );

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX teams_name_trgm_idx ON teams USING gin (name gin_trgm_ops);
CREATE INDEX people_display_name_trgm_idx ON people USING gin (display_name gin_trgm_ops);

DROP INDEX match_events_provider_external_idx;

ALTER TABLE leagues
    DROP CONSTRAINT leagues_provider_external_id_key,
    DROP COLUMN provider,
    DROP COLUMN external_id;
ALTER TABLE seasons
    DROP CONSTRAINT seasons_provider_external_id_key,
    DROP COLUMN provider,
    DROP COLUMN external_id;
ALTER TABLE venues
    DROP CONSTRAINT venues_provider_external_id_key,
    DROP COLUMN provider,
    DROP COLUMN external_id;
ALTER TABLE teams
    DROP CONSTRAINT teams_provider_external_id_key,
    DROP COLUMN provider,
    DROP COLUMN external_id;
ALTER TABLE people
    DROP CONSTRAINT people_provider_external_id_key,
    DROP COLUMN provider,
    DROP COLUMN external_id;
ALTER TABLE matches
    DROP CONSTRAINT matches_provider_external_id_key,
    DROP COLUMN provider,
    DROP COLUMN external_id;
ALTER TABLE match_events
    DROP COLUMN provider,
    DROP COLUMN external_id;
