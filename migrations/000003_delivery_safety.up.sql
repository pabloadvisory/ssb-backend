ALTER TABLE notification_deliveries
    ADD COLUMN lock_token uuid,
    ADD COLUMN completed_at timestamptz;

UPDATE notification_deliveries
SET completed_at = COALESCE(sent_at, created_at)
WHERE state IN ('sent', 'failed') AND completed_at IS NULL;

CREATE INDEX notification_deliveries_terminal_cleanup_idx
    ON notification_deliveries (completed_at, id)
    WHERE state IN ('sent', 'failed');

ALTER TABLE outbox_events
    ADD COLUMN available_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN locked_until timestamptz,
    ADD COLUMN lock_token uuid,
    ADD COLUMN failed_at timestamptz;

DROP INDEX outbox_events_unpublished_idx;
CREATE INDEX outbox_events_claim_idx
    ON outbox_events (available_at, occurred_at, id)
    WHERE published_at IS NULL AND failed_at IS NULL;
CREATE INDEX outbox_events_aggregate_order_idx
    ON outbox_events (aggregate_type, aggregate_id, occurred_at, id)
    WHERE published_at IS NULL AND failed_at IS NULL;
CREATE INDEX outbox_events_published_cleanup_idx
    ON outbox_events (published_at, id)
    WHERE published_at IS NOT NULL;
CREATE INDEX outbox_events_failed_cleanup_idx
    ON outbox_events (failed_at, id)
    WHERE failed_at IS NOT NULL;

CREATE TABLE push_endpoint_registration_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id uuid REFERENCES push_endpoints(id) ON DELETE SET NULL,
    previous_installation_id uuid REFERENCES app_installations(id) ON DELETE SET NULL,
    new_installation_id uuid REFERENCES app_installations(id) ON DELETE SET NULL,
    transport text NOT NULL,
    kind text NOT NULL,
    reason text NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT push_endpoint_registration_audit_reason_valid CHECK (reason IN ('token_transferred'))
);
CREATE INDEX push_endpoint_registration_audit_endpoint_idx
    ON push_endpoint_registration_audit (endpoint_id, occurred_at DESC);
CREATE INDEX push_endpoint_registration_audit_cleanup_idx
    ON push_endpoint_registration_audit (occurred_at, id);
