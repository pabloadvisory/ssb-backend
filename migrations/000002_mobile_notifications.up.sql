CREATE TABLE app_installations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    secret_hash bytea NOT NULL,
    platform text NOT NULL,
    app_id text NOT NULL,
    user_reference text,
    locale text,
    timezone text,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    disabled_at timestamptz,
    CONSTRAINT app_installations_platform_valid CHECK (platform IN ('ios', 'android'))
);
CREATE INDEX app_installations_user_idx ON app_installations (user_reference) WHERE user_reference IS NOT NULL;

CREATE TABLE push_endpoints (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id uuid NOT NULL REFERENCES app_installations(id) ON DELETE CASCADE,
    transport text NOT NULL,
    kind text NOT NULL,
    token text NOT NULL,
    token_hash bytea NOT NULL,
    environment text NOT NULL DEFAULT 'production',
    match_id uuid REFERENCES matches(id) ON DELETE CASCADE,
    activity_id text,
    frequent_updates_enabled boolean NOT NULL DEFAULT false,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'active',
    registered_at timestamptz NOT NULL DEFAULT now(),
    last_success_at timestamptz,
    last_failure_at timestamptz,
    last_failure_reason text,
    UNIQUE (token_hash),
    CONSTRAINT push_endpoints_transport_valid CHECK (transport IN ('apns', 'fcm')),
    CONSTRAINT push_endpoints_kind_valid CHECK (kind IN ('standard', 'live_activity', 'push_to_start')),
    CONSTRAINT push_endpoints_environment_valid CHECK (environment IN ('sandbox', 'production')),
    CONSTRAINT push_endpoints_status_valid CHECK (status IN ('active', 'invalid', 'disabled')),
    CONSTRAINT push_endpoints_live_activity_valid CHECK (
        kind <> 'live_activity' OR (transport = 'apns' AND match_id IS NOT NULL AND activity_id IS NOT NULL)
    ),
    CONSTRAINT push_endpoints_push_to_start_valid CHECK (
        kind <> 'push_to_start' OR (transport = 'apns' AND match_id IS NULL)
    )
);
CREATE INDEX push_endpoints_installation_idx ON push_endpoints (installation_id, kind);
CREATE INDEX push_endpoints_live_match_idx ON push_endpoints (match_id, id) WHERE kind = 'live_activity' AND status = 'active';

CREATE TABLE match_subscriptions (
    installation_id uuid NOT NULL REFERENCES app_installations(id) ON DELETE CASCADE,
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    notifications_enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (installation_id, match_id)
);
CREATE INDEX match_subscriptions_match_idx ON match_subscriptions (match_id, installation_id) WHERE notifications_enabled;

CREATE TABLE notification_deliveries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id uuid NOT NULL REFERENCES push_endpoints(id) ON DELETE CASCADE,
    match_id uuid REFERENCES matches(id) ON DELETE CASCADE,
    kind text NOT NULL,
    payload jsonb NOT NULL,
    collapse_key text,
    priority text NOT NULL DEFAULT 'normal',
    idempotency_key text NOT NULL,
    state text NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    locked_until timestamptz,
    provider_message_id text,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    sent_at timestamptz,
    UNIQUE (endpoint_id, idempotency_key),
    CONSTRAINT notification_deliveries_kind_valid CHECK (kind IN ('match_update', 'match_finished', 'live_activity_start', 'live_activity_update', 'live_activity_end')),
    CONSTRAINT notification_deliveries_priority_valid CHECK (priority IN ('normal', 'high')),
    CONSTRAINT notification_deliveries_state_valid CHECK (state IN ('pending', 'sending', 'sent', 'failed')),
    CONSTRAINT notification_deliveries_attempts_positive CHECK (attempts >= 0)
);
CREATE INDEX notification_deliveries_claim_idx ON notification_deliveries (next_attempt_at, created_at, id)
    WHERE state IN ('pending', 'sending');

CREATE TRIGGER match_subscriptions_touch_updated_at BEFORE UPDATE ON match_subscriptions FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
