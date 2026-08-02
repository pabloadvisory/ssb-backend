ALTER TABLE team_memberships
    ADD COLUMN is_loan boolean NOT NULL DEFAULT false,
    ADD COLUMN parent_team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
    ADD COLUMN transfer_type text NOT NULL DEFAULT 'unknown',
    ADD CONSTRAINT team_memberships_parent_team_different CHECK (
        parent_team_id IS NULL OR parent_team_id <> team_id
    ),
    ADD CONSTRAINT team_memberships_loan_parent_valid CHECK (
        is_loan OR parent_team_id IS NULL
    ),
    ADD CONSTRAINT team_memberships_transfer_type_valid CHECK (
        transfer_type IN ('permanent', 'loan', 'free', 'youth', 'unknown')
    );

ALTER TABLE player_match_statistics
    ADD COLUMN passes_completed smallint,
    ADD COLUMN key_passes smallint,
    ADD COLUMN interceptions smallint,
    ADD COLUMN clearances smallint,
    ADD COLUMN blocks smallint,
    ADD COLUMN duels smallint,
    ADD COLUMN duels_won smallint,
    ADD COLUMN expected_goals numeric(7,3),
    ADD COLUMN expected_assists numeric(7,3),
    ADD CONSTRAINT player_match_statistics_advanced_counts_valid CHECK (
        (passes_completed IS NULL OR passes_completed BETWEEN 0 AND passes) AND
        (key_passes IS NULL OR key_passes >= 0) AND
        (interceptions IS NULL OR interceptions >= 0) AND
        (clearances IS NULL OR clearances >= 0) AND
        (blocks IS NULL OR blocks >= 0) AND
        (duels IS NULL OR duels >= 0) AND
        (duels_won IS NULL OR duels_won >= 0) AND
        (duels_won IS NULL OR duels IS NULL OR duels_won <= duels) AND
        (expected_goals IS NULL OR expected_goals >= 0) AND
        (expected_assists IS NULL OR expected_assists >= 0)
    );

CREATE INDEX player_match_statistics_player_idx
    ON player_match_statistics (person_id, match_id);
CREATE INDEX players_position_idx ON players (position, person_id);
CREATE INDEX team_memberships_roster_idx
    ON team_memberships (team_id, person_id, starts_on, ends_on);
CREATE INDEX match_events_substitution_primary_idx
    ON match_events (primary_person_id, match_id)
    WHERE type = 'substitution' AND primary_person_id IS NOT NULL;
CREATE INDEX match_events_substitution_secondary_idx
    ON match_events (secondary_person_id, match_id)
    WHERE type = 'substitution' AND secondary_person_id IS NOT NULL;

CREATE TABLE player_trait_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id uuid NOT NULL REFERENCES players(person_id) ON DELETE CASCADE,
    team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
    league_id uuid NOT NULL REFERENCES leagues(id) ON DELETE CASCADE,
    season_id uuid NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    source text NOT NULL,
    external_id text NOT NULL,
    position_group text NOT NULL,
    minimum_minutes integer NOT NULL,
    cohort_size integer NOT NULL,
    player_minutes integer NOT NULL,
    observed_at timestamptz NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CONSTRAINT player_trait_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT player_trait_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT player_trait_position_valid CHECK (
        position_group IN ('goalkeeper', 'defender', 'midfielder', 'forward')
    ),
    CONSTRAINT player_trait_cohort_valid CHECK (
        minimum_minutes >= 0 AND cohort_size >= 2 AND player_minutes >= 0
    )
);
CREATE INDEX player_trait_player_latest_idx
    ON player_trait_snapshots (person_id, season_id, league_id, observed_at DESC, id DESC);
CREATE INDEX player_trait_cohort_idx
    ON player_trait_snapshots (league_id, season_id, position_group, minimum_minutes);
CREATE TRIGGER player_trait_snapshots_touch_updated_at
    BEFORE UPDATE ON player_trait_snapshots FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE player_trait_metrics (
    snapshot_id uuid NOT NULL REFERENCES player_trait_snapshots(id) ON DELETE CASCADE,
    metric_key text NOT NULL,
    label text NOT NULL,
    category text NOT NULL,
    raw_value double precision,
    per_90_value double precision,
    percentile numeric(5,2) NOT NULL,
    unit text,
    direction text NOT NULL,
    PRIMARY KEY (snapshot_id, metric_key),
    CONSTRAINT player_trait_metric_key_valid CHECK (
        metric_key ~ '^[a-z0-9]+(?:_[a-z0-9]+)*$'
    ),
    CONSTRAINT player_trait_metric_label_not_blank CHECK (btrim(label) <> ''),
    CONSTRAINT player_trait_metric_category_valid CHECK (
        category IN ('attacking', 'possession', 'progression', 'creation', 'defending', 'goalkeeping', 'discipline', 'physical')
    ),
    CONSTRAINT player_trait_metric_percentile_valid CHECK (percentile BETWEEN 0 AND 100),
    CONSTRAINT player_trait_metric_direction_valid CHECK (
        direction IN ('higher_is_better', 'lower_is_better', 'neutral')
    )
);

CREATE TABLE player_spatial_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    person_id uuid NOT NULL REFERENCES players(person_id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    source text NOT NULL,
    external_id text NOT NULL,
    orientation text NOT NULL DEFAULT 'attacking_left_to_right',
    observed_at timestamptz NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CONSTRAINT player_spatial_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT player_spatial_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT player_spatial_orientation_canonical CHECK (orientation = 'attacking_left_to_right')
);
CREATE INDEX player_spatial_player_match_idx
    ON player_spatial_snapshots (person_id, match_id, observed_at DESC, id DESC);
CREATE TRIGGER player_spatial_snapshots_touch_updated_at
    BEFORE UPDATE ON player_spatial_snapshots FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE player_touch_points (
    snapshot_id uuid NOT NULL REFERENCES player_spatial_snapshots(id) ON DELETE CASCADE,
    sequence integer NOT NULL,
    minute smallint,
    stoppage_minute smallint,
    x double precision NOT NULL,
    y double precision NOT NULL,
    intensity double precision NOT NULL DEFAULT 1,
    touch_type text,
    PRIMARY KEY (snapshot_id, sequence),
    CONSTRAINT player_touch_sequence_positive CHECK (sequence > 0),
    CONSTRAINT player_touch_minute_valid CHECK (minute IS NULL OR minute BETWEEN 0 AND 200),
    CONSTRAINT player_touch_stoppage_valid CHECK (stoppage_minute IS NULL OR stoppage_minute BETWEEN 0 AND 99),
    CONSTRAINT player_touch_coordinates_valid CHECK (x BETWEEN 0 AND 100 AND y BETWEEN 0 AND 100),
    CONSTRAINT player_touch_intensity_positive CHECK (intensity > 0)
);

CREATE TABLE player_shots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id uuid NOT NULL REFERENCES player_spatial_snapshots(id) ON DELETE CASCADE,
    sequence integer NOT NULL,
    match_event_id uuid REFERENCES match_events(id) ON DELETE SET NULL,
    minute smallint,
    stoppage_minute smallint,
    x double precision NOT NULL,
    y double precision NOT NULL,
    expected_goals numeric(5,4) NOT NULL,
    outcome text NOT NULL,
    body_part text NOT NULL,
    shot_type text,
    UNIQUE (snapshot_id, sequence),
    CONSTRAINT player_shot_sequence_positive CHECK (sequence > 0),
    CONSTRAINT player_shot_minute_valid CHECK (minute IS NULL OR minute BETWEEN 0 AND 200),
    CONSTRAINT player_shot_stoppage_valid CHECK (stoppage_minute IS NULL OR stoppage_minute BETWEEN 0 AND 99),
    CONSTRAINT player_shot_coordinates_valid CHECK (x BETWEEN 0 AND 100 AND y BETWEEN 0 AND 100),
    CONSTRAINT player_shot_xg_valid CHECK (expected_goals BETWEEN 0 AND 1),
    CONSTRAINT player_shot_outcome_valid CHECK (
        outcome IN ('goal', 'saved', 'blocked', 'off_target', 'woodwork', 'own_goal', 'unknown')
    ),
    CONSTRAINT player_shot_body_part_valid CHECK (
        body_part IN ('left_foot', 'right_foot', 'head', 'other', 'unknown')
    )
);
CREATE INDEX player_shots_snapshot_time_idx
    ON player_shots (snapshot_id, minute DESC, sequence DESC);

CREATE TABLE player_valuations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id uuid NOT NULL REFERENCES players(person_id) ON DELETE CASCADE,
    team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
    source text NOT NULL,
    external_id text NOT NULL,
    amount_minor bigint NOT NULL,
    currency char(3) NOT NULL,
    valued_on date NOT NULL,
    observed_at timestamptz NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CONSTRAINT player_valuation_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT player_valuation_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT player_valuation_amount_valid CHECK (amount_minor >= 0),
    CONSTRAINT player_valuation_currency_valid CHECK (currency ~ '^[A-Z]{3}$')
);
CREATE INDEX player_valuations_latest_idx
    ON player_valuations (person_id, valued_on DESC, observed_at DESC, id DESC);
CREATE TRIGGER player_valuations_touch_updated_at
    BEFORE UPDATE ON player_valuations FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE player_follows (
    installation_id uuid NOT NULL REFERENCES app_installations(id) ON DELETE CASCADE,
    player_id uuid NOT NULL REFERENCES players(person_id) ON DELETE CASCADE,
    notifications_enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (installation_id, player_id)
);
CREATE INDEX player_follows_notification_audience_idx
    ON player_follows (player_id, installation_id)
    WHERE notifications_enabled;
CREATE TRIGGER player_follows_touch_updated_at
    BEFORE UPDATE ON player_follows FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE installation_notification_preferences (
    installation_id uuid PRIMARY KEY REFERENCES app_installations(id) ON DELETE CASCADE,
    match_updates_enabled boolean NOT NULL DEFAULT true,
    match_finished_enabled boolean NOT NULL DEFAULT true,
    followed_player_events_enabled boolean NOT NULL DEFAULT true,
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER installation_notification_preferences_touch_updated_at
    BEFORE UPDATE ON installation_notification_preferences FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
