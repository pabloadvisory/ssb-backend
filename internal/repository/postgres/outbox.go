package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/eventing"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/realtime"
)

func (store *Store) ClaimOutboxEvents(ctx context.Context, limit int, lockDuration time.Duration) ([]eventing.Event, error) {
	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT event.id FROM outbox_events event
			WHERE event.published_at IS NULL AND event.failed_at IS NULL AND event.available_at <= now()
			  AND (event.locked_until IS NULL OR event.locked_until < now())
			  AND NOT EXISTS (
				SELECT 1 FROM outbox_events earlier
				WHERE earlier.aggregate_type = event.aggregate_type
				  AND earlier.aggregate_id = event.aggregate_id
				  AND earlier.published_at IS NULL AND earlier.failed_at IS NULL
				  AND (earlier.occurred_at, earlier.id) < (event.occurred_at, event.id)
			  )
			ORDER BY event.available_at, event.occurred_at, event.id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		), claimed AS (
			UPDATE outbox_events event
			SET attempts = attempts + 1, locked_until = now() + ($2 * interval '1 second'),
				lock_token = gen_random_uuid()
			FROM candidates WHERE event.id = candidates.id
			RETURNING event.id, event.event_type, event.payload, event.lock_token, event.attempts
		)
		SELECT id, event_type, payload, lock_token, attempts FROM claimed`, limit, int64(lockDuration.Seconds()))
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	events := make([]eventing.Event, 0, limit)
	for rows.Next() {
		var event eventing.Event
		if err := rows.Scan(&event.ID, &event.Type, &event.Payload, &event.LockToken, &event.Attempts); err != nil {
			return nil, mapError(err)
		}
		events = append(events, event)
	}
	return events, mapError(rows.Err())
}

func (store *Store) PublishMatchChanged(
	ctx context.Context,
	event eventing.Event,
	changed eventing.MatchChanged,
	plans []notification.DeliveryPlan,
) error {
	matchID := changed.Notification.Current.MatchID
	if matchID == "" || matchID != changed.Realtime.MatchID {
		return fmt.Errorf("match changed event has inconsistent match IDs")
	}
	realtimePayload, err := json.Marshal(changed.Realtime)
	if err != nil {
		return fmt.Errorf("encode realtime update: %w", err)
	}
	pushPayload, err := json.Marshal(changed.Notification.Current)
	if err != nil {
		return fmt.Errorf("encode push update: %w", err)
	}

	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin outbox publication: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := tx.Exec(ctx, `
		UPDATE outbox_events
		SET published_at = now(), locked_until = NULL, lock_token = NULL, last_error = NULL
		WHERE id = $1 AND published_at IS NULL AND failed_at IS NULL AND lock_token = $2`, event.ID, event.LockToken)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() != 1 {
		return eventing.ErrLeaseLost
	}

	for _, plan := range plans {
		if err := enqueueDeliveryPlan(ctx, tx, matchID, changed.Notification.Current.Version, pushPayload, plan); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, "SELECT pg_notify($1, $2)", realtime.Channel, string(realtimePayload)); err != nil {
		return mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return mapError(err)
	}
	return nil
}

func enqueueDeliveryPlan(
	ctx context.Context,
	tx pgx.Tx,
	matchID string,
	version int64,
	payload []byte,
	plan notification.DeliveryPlan,
) error {
	var query string
	switch plan.Audience {
	case notification.AudienceLiveActivities:
		query = `
			INSERT INTO notification_deliveries (
				endpoint_id, match_id, kind, payload, collapse_key, priority, idempotency_key
			)
			SELECT endpoint.id, $1::uuid, $2, $3::jsonb, 'match:' || $1::uuid::text, $4,
				$6 || ':' || $1::uuid::text || ':' || $5::bigint::text
			FROM push_endpoints endpoint
			JOIN app_installations installation ON installation.id = endpoint.installation_id
			WHERE endpoint.kind = 'live_activity' AND endpoint.match_id = $1::uuid
			  AND endpoint.status = 'active' AND installation.disabled_at IS NULL
			ON CONFLICT (endpoint_id, idempotency_key) DO NOTHING`
	case notification.AudienceSubscribers:
		query = `
			INSERT INTO notification_deliveries (
				endpoint_id, match_id, kind, payload, collapse_key, priority, idempotency_key
			)
			SELECT endpoint.id, $1::uuid, $2, $3::jsonb, 'match:' || $1::uuid::text, $4,
				$6 || ':' || $1::uuid::text || ':' || $5::bigint::text
			FROM match_subscriptions subscription
			JOIN push_endpoints endpoint ON endpoint.installation_id = subscription.installation_id
			JOIN app_installations installation ON installation.id = endpoint.installation_id
			WHERE subscription.match_id = $1::uuid AND subscription.notifications_enabled
			  AND endpoint.kind = 'standard' AND endpoint.status = 'active'
			  AND installation.disabled_at IS NULL
			ON CONFLICT (endpoint_id, idempotency_key) DO NOTHING`
	case notification.AudiencePushToStart:
		query = `
			INSERT INTO notification_deliveries (
				endpoint_id, match_id, kind, payload, collapse_key, priority, idempotency_key
			)
			SELECT endpoint.id, $1::uuid, $2, $3::jsonb, 'match:' || $1::uuid::text, $4,
				$6 || ':' || $1::uuid::text || ':' || $5::bigint::text
			FROM match_subscriptions subscription
			JOIN push_endpoints endpoint ON endpoint.installation_id = subscription.installation_id
			JOIN app_installations installation ON installation.id = endpoint.installation_id
			WHERE subscription.match_id = $1::uuid AND subscription.notifications_enabled
			  AND endpoint.kind = 'push_to_start' AND endpoint.status = 'active'
			  AND installation.disabled_at IS NULL
			ON CONFLICT (endpoint_id, idempotency_key) DO NOTHING`
	default:
		return fmt.Errorf("unsupported notification audience %q", plan.Audience)
	}
	if _, err := tx.Exec(ctx, query, matchID, plan.Kind, payload, plan.Priority, version, plan.IdempotencyPrefix); err != nil {
		return mapError(err)
	}
	return nil
}

func (store *Store) RetryOutboxEvent(
	ctx context.Context,
	id, lockToken string,
	nextAttempt time.Time,
	reason string,
	terminal bool,
) error {
	result, err := store.pool.Exec(ctx, `
		UPDATE outbox_events
		SET available_at = $3, locked_until = NULL, lock_token = NULL, last_error = $4,
			failed_at = CASE WHEN $5 THEN now() ELSE NULL END
		WHERE id = $1 AND published_at IS NULL AND failed_at IS NULL AND lock_token = $2`,
		id, lockToken, nextAttempt, reason, terminal,
	)
	if err != nil {
		return mapError(err)
	}
	if result.RowsAffected() != 1 {
		return eventing.ErrLeaseLost
	}
	return nil
}

var _ eventing.Store = (*Store)(nil)
