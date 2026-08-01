ALTER TABLE teams
    ADD COLUMN primary_color text,
    ADD COLUMN secondary_color text,
    ADD CONSTRAINT teams_primary_color_valid CHECK (
        primary_color IS NULL OR primary_color ~ '^#[0-9A-Fa-f]{6}$'
    ),
    ADD CONSTRAINT teams_secondary_color_valid CHECK (
        secondary_color IS NULL OR secondary_color ~ '^#[0-9A-Fa-f]{6}$'
    );

CREATE TABLE match_team_statistics (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    possession numeric(5,2),
    shots smallint,
    shots_on_target smallint,
    shots_off_target smallint,
    blocked_shots smallint,
    shots_inside_box smallint,
    shots_outside_box smallint,
    corners smallint,
    passes smallint,
    passes_completed smallint,
    pass_accuracy numeric(5,2),
    fouls smallint,
    offsides smallint,
    yellow_cards smallint,
    red_cards smallint,
    saves smallint,
    tackles smallint,
    interceptions smallint,
    clearances smallint,
    expected_goals numeric(6,3),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, team_id),
    CONSTRAINT match_team_statistics_percentages_valid CHECK (
        (possession IS NULL OR possession BETWEEN 0 AND 100) AND
        (pass_accuracy IS NULL OR pass_accuracy BETWEEN 0 AND 100)
    ),
    CONSTRAINT match_team_statistics_counts_valid CHECK (
        (shots IS NULL OR shots >= 0) AND
        (shots_on_target IS NULL OR shots_on_target >= 0) AND
        (shots_off_target IS NULL OR shots_off_target >= 0) AND
        (blocked_shots IS NULL OR blocked_shots >= 0) AND
        (shots_inside_box IS NULL OR shots_inside_box >= 0) AND
        (shots_outside_box IS NULL OR shots_outside_box >= 0) AND
        (corners IS NULL OR corners >= 0) AND
        (passes IS NULL OR passes >= 0) AND
        (passes_completed IS NULL OR passes_completed >= 0) AND
        (fouls IS NULL OR fouls >= 0) AND
        (offsides IS NULL OR offsides >= 0) AND
        (yellow_cards IS NULL OR yellow_cards >= 0) AND
        (red_cards IS NULL OR red_cards >= 0) AND
        (saves IS NULL OR saves >= 0) AND
        (tackles IS NULL OR tackles >= 0) AND
        (interceptions IS NULL OR interceptions >= 0) AND
        (clearances IS NULL OR clearances >= 0) AND
        (expected_goals IS NULL OR expected_goals >= 0)
    )
);
CREATE TRIGGER match_team_statistics_touch_updated_at
    BEFORE UPDATE ON match_team_statistics FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE INDEX matches_head_to_head_idx
    ON matches (
        LEAST(home_team_id, away_team_id),
        GREATEST(home_team_id, away_team_id),
        kickoff_at DESC,
        id DESC
    )
    WHERE status IN ('finished', 'awarded')
      AND home_score IS NOT NULL
      AND away_score IS NOT NULL;

CREATE INDEX teams_search_trgm_idx
    ON teams USING gin (
        (name || ' ' || COALESCE(short_name, '') || ' ' || COALESCE(code, '')) gin_trgm_ops
    );

CREATE TABLE bookmakers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    logo_url text,
    website_url text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT bookmakers_slug_valid CHECK (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
    CONSTRAINT bookmakers_name_not_blank CHECK (btrim(name) <> '')
);
CREATE TRIGGER bookmakers_touch_updated_at
    BEFORE UPDATE ON bookmakers FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE odds_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    bookmaker_id uuid NOT NULL REFERENCES bookmakers(id) ON DELETE RESTRICT,
    source text NOT NULL,
    external_id text NOT NULL,
    observed_at timestamptz NOT NULL,
    valid_until timestamptz,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CONSTRAINT odds_snapshots_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT odds_snapshots_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT odds_snapshots_validity_order CHECK (valid_until IS NULL OR valid_until >= observed_at)
);
CREATE INDEX odds_snapshots_match_latest_idx
    ON odds_snapshots (match_id, bookmaker_id, observed_at DESC, id DESC);
CREATE TRIGGER odds_snapshots_touch_updated_at
    BEFORE UPDATE ON odds_snapshots FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE odds_markets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id uuid NOT NULL REFERENCES odds_snapshots(id) ON DELETE CASCADE,
    market_key text NOT NULL,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'open',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (snapshot_id, market_key),
    CONSTRAINT odds_markets_key_not_blank CHECK (btrim(market_key) <> ''),
    CONSTRAINT odds_markets_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT odds_markets_status_valid CHECK (status IN ('open', 'suspended', 'settled'))
);

CREATE TABLE odds_selections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    market_id uuid NOT NULL REFERENCES odds_markets(id) ON DELETE CASCADE,
    selection_key text NOT NULL,
    name text NOT NULL,
    line numeric(10,3),
    decimal_odds numeric(10,4) NOT NULL,
    result text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (market_id, selection_key, line),
    CONSTRAINT odds_selections_key_not_blank CHECK (btrim(selection_key) <> ''),
    CONSTRAINT odds_selections_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT odds_selections_price_valid CHECK (decimal_odds > 1),
    CONSTRAINT odds_selections_result_valid CHECK (result IS NULL OR result IN ('won', 'lost', 'push', 'void'))
);

CREATE TABLE match_broadcasts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    source text NOT NULL,
    external_id text NOT NULL,
    network_name text NOT NULL,
    service_name text,
    kind text NOT NULL,
    availability_scope text NOT NULL DEFAULT 'unknown',
    language_tags text[] NOT NULL DEFAULT '{}',
    starts_at timestamptz,
    ends_at timestamptz,
    is_free boolean NOT NULL DEFAULT false,
    requires_subscription boolean NOT NULL DEFAULT false,
    web_url text,
    deep_link_url text,
    status text NOT NULL DEFAULT 'scheduled',
    observed_at timestamptz NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CONSTRAINT match_broadcasts_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT match_broadcasts_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT match_broadcasts_network_not_blank CHECK (btrim(network_name) <> ''),
    CONSTRAINT match_broadcasts_kind_valid CHECK (kind IN ('television', 'radio', 'stream')),
    CONSTRAINT match_broadcasts_scope_valid CHECK (availability_scope IN ('global', 'territorial', 'unknown')),
    CONSTRAINT match_broadcasts_status_valid CHECK (status IN ('scheduled', 'live', 'ended', 'cancelled', 'unavailable')),
    CONSTRAINT match_broadcasts_time_order CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at >= starts_at)
);
CREATE INDEX match_broadcasts_match_idx ON match_broadcasts (match_id, status, starts_at, id);
CREATE TRIGGER match_broadcasts_touch_updated_at
    BEFORE UPDATE ON match_broadcasts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE match_broadcast_regions (
    broadcast_id uuid NOT NULL REFERENCES match_broadcasts(id) ON DELETE CASCADE,
    country_code char(2) NOT NULL,
    PRIMARY KEY (broadcast_id, country_code),
    CONSTRAINT match_broadcast_regions_country_valid CHECK (country_code = upper(country_code))
);
CREATE INDEX match_broadcast_regions_country_idx ON match_broadcast_regions (country_code, broadcast_id);

CREATE TABLE match_weather_snapshots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    source text NOT NULL,
    external_id text NOT NULL,
    kind text NOT NULL,
    valid_at timestamptz NOT NULL,
    issued_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    temperature_c numeric(5,2),
    feels_like_c numeric(5,2),
    humidity_percent numeric(5,2),
    precipitation_probability_percent numeric(5,2),
    precipitation_mm numeric(8,2),
    wind_speed_kph numeric(7,2),
    wind_gust_kph numeric(7,2),
    wind_direction_degrees smallint,
    pressure_hpa numeric(7,2),
    visibility_km numeric(7,2),
    condition_code text,
    condition_text text,
    icon_url text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id),
    CONSTRAINT match_weather_source_not_blank CHECK (btrim(source) <> ''),
    CONSTRAINT match_weather_external_id_not_blank CHECK (btrim(external_id) <> ''),
    CONSTRAINT match_weather_kind_valid CHECK (kind IN ('forecast', 'observed')),
    CONSTRAINT match_weather_percentages_valid CHECK (
        (humidity_percent IS NULL OR humidity_percent BETWEEN 0 AND 100) AND
        (precipitation_probability_percent IS NULL OR precipitation_probability_percent BETWEEN 0 AND 100)
    ),
    CONSTRAINT match_weather_nonnegative_valid CHECK (
        (precipitation_mm IS NULL OR precipitation_mm >= 0) AND
        (wind_speed_kph IS NULL OR wind_speed_kph >= 0) AND
        (wind_gust_kph IS NULL OR wind_gust_kph >= 0) AND
        (pressure_hpa IS NULL OR pressure_hpa > 0) AND
        (visibility_km IS NULL OR visibility_km >= 0)
    ),
    CONSTRAINT match_weather_wind_direction_valid CHECK (
        wind_direction_degrees IS NULL OR wind_direction_degrees BETWEEN 0 AND 359
    )
);
CREATE INDEX match_weather_match_kind_idx
    ON match_weather_snapshots (match_id, kind, valid_at, issued_at DESC, id DESC);
CREATE TRIGGER match_weather_snapshots_touch_updated_at
    BEFORE UPDATE ON match_weather_snapshots FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

CREATE TABLE match_predictions (
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    installation_id uuid NOT NULL REFERENCES app_installations(id) ON DELETE CASCADE,
    selection text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (match_id, installation_id),
    CONSTRAINT match_predictions_selection_valid CHECK (selection IN ('home', 'draw', 'away'))
);
CREATE INDEX match_predictions_totals_idx ON match_predictions (match_id, selection);
CREATE TRIGGER match_predictions_touch_updated_at
    BEFORE UPDATE ON match_predictions FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
