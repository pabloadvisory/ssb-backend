package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
)

func (store *Store) CreateInstallation(ctx context.Context, command notification.CreateInstallation, secretHash []byte) (notification.Installation, error) {
	var installation notification.Installation
	err := store.pool.QueryRow(ctx, `
		INSERT INTO app_installations (secret_hash, platform, app_id, locale, timezone)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, platform, app_id, locale, timezone, created_at`,
		secretHash, command.Platform, command.AppID, command.Locale, command.Timezone,
	).Scan(&installation.ID, &installation.Platform, &installation.AppID, &installation.Locale, &installation.Timezone, &installation.CreatedAt)
	return installation, mapNotificationError(err)
}

func (store *Store) RegisterEndpoint(
	ctx context.Context,
	installationID string,
	secretHash []byte,
	kind notification.EndpointKind,
	command notification.RegisterEndpoint,
	tokenHash []byte,
) (notification.Endpoint, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return notification.Endpoint{}, fmt.Errorf("begin endpoint registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var platform notification.Platform
	err = tx.QueryRow(ctx, `
		SELECT platform FROM app_installations
		WHERE id = $1 AND secret_hash = $2 AND disabled_at IS NULL`, installationID, secretHash).Scan(&platform)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.Endpoint{}, notification.ErrUnauthorized
	}
	if err != nil {
		return notification.Endpoint{}, mapNotificationError(err)
	}
	if (platform == notification.PlatformIOS && command.Transport != notification.TransportAPNs) ||
		(platform == notification.PlatformAndroid && command.Transport != notification.TransportFCM) {
		return notification.Endpoint{}, fmt.Errorf("%w: transport does not match installation platform", notification.ErrInvalid)
	}

	// Serialize registration of a token so an ownership transfer cannot race the audit record.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(encode($1::bytea, 'hex'), 0))`, tokenHash); err != nil {
		return notification.Endpoint{}, mapNotificationError(err)
	}
	var endpointID, previousInstallationID string
	err = tx.QueryRow(ctx, `
		SELECT endpoint.id, endpoint.installation_id
		FROM push_endpoint_tokens token
		JOIN push_endpoints endpoint ON endpoint.id = token.endpoint_id
		WHERE token.token_hash = $1
		FOR UPDATE OF token, endpoint`, tokenHash).Scan(&endpointID, &previousInstallationID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return notification.Endpoint{}, mapNotificationError(err)
	}

	var endpoint notification.Endpoint
	metadata := command.Metadata
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	if endpointID == "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO push_endpoints (
				installation_id, transport, kind, environment, match_id, activity_id,
				frequent_updates_enabled, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id, installation_id, transport, kind, environment, match_id, activity_id,
				frequent_updates_enabled, metadata, registered_at`,
			installationID, command.Transport, kind, command.Environment, command.MatchID,
			command.ActivityID, command.FrequentUpdatesEnabled, metadata,
		).Scan(
			&endpoint.ID, &endpoint.InstallationID, &endpoint.Transport, &endpoint.Kind, &endpoint.Environment,
			&endpoint.MatchID, &endpoint.ActivityID, &endpoint.FrequentUpdatesEnabled, &endpoint.Metadata, &endpoint.RegisteredAt,
		)
	} else {
		err = tx.QueryRow(ctx, `
			UPDATE push_endpoints SET
				installation_id=$2, transport=$3, kind=$4, environment=$5, match_id=$6, activity_id=$7,
				frequent_updates_enabled=$8, metadata=$9, status='active', registered_at=now(),
				last_failure_at=NULL, last_failure_reason=NULL
			WHERE id=$1
			RETURNING id, installation_id, transport, kind, environment, match_id, activity_id,
				frequent_updates_enabled, metadata, registered_at`,
			endpointID, installationID, command.Transport, kind, command.Environment, command.MatchID,
			command.ActivityID, command.FrequentUpdatesEnabled, metadata,
		).Scan(
			&endpoint.ID, &endpoint.InstallationID, &endpoint.Transport, &endpoint.Kind, &endpoint.Environment,
			&endpoint.MatchID, &endpoint.ActivityID, &endpoint.FrequentUpdatesEnabled, &endpoint.Metadata, &endpoint.RegisteredAt,
		)
	}
	if err != nil {
		return notification.Endpoint{}, mapNotificationError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO push_endpoint_tokens (endpoint_id, token, token_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (endpoint_id) DO UPDATE SET
			token = EXCLUDED.token, token_hash = EXCLUDED.token_hash, updated_at = now()`,
		endpoint.ID, command.Token, tokenHash,
	); err != nil {
		return notification.Endpoint{}, mapNotificationError(err)
	}
	if previousInstallationID != "" && previousInstallationID != installationID {
		if _, err := tx.Exec(ctx, `
			INSERT INTO push_endpoint_registration_audit (
				endpoint_id, previous_installation_id, new_installation_id, transport, kind, reason
			) VALUES ($1, $2, $3, $4, $5, 'token_transferred')`,
			endpoint.ID, previousInstallationID, installationID, endpoint.Transport, endpoint.Kind,
		); err != nil {
			return notification.Endpoint{}, mapNotificationError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return notification.Endpoint{}, mapNotificationError(err)
	}
	return endpoint, nil
}

func (store *Store) SetMatchSubscription(ctx context.Context, installationID string, secretHash []byte, matchID string, enabled bool) (notification.Subscription, error) {
	var authenticated bool
	err := store.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM app_installations WHERE id = $1 AND secret_hash = $2 AND disabled_at IS NULL
		)`, installationID, secretHash).Scan(&authenticated)
	if err != nil {
		return notification.Subscription{}, mapNotificationError(err)
	}
	if !authenticated {
		return notification.Subscription{}, notification.ErrUnauthorized
	}

	var subscription notification.Subscription
	err = store.pool.QueryRow(ctx, `
		INSERT INTO match_subscriptions (installation_id, match_id, notifications_enabled)
		VALUES ($1, $2, $3)
		ON CONFLICT (installation_id, match_id) DO UPDATE SET notifications_enabled = EXCLUDED.notifications_enabled
		RETURNING installation_id, match_id, notifications_enabled, updated_at`, installationID, matchID, enabled).Scan(
		&subscription.InstallationID, &subscription.MatchID, &subscription.NotificationsEnabled, &subscription.UpdatedAt,
	)
	return subscription, mapNotificationError(err)
}

func (store *Store) ClaimDeliveries(ctx context.Context, limit int, lockDuration time.Duration) ([]notification.Delivery, error) {
	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT delivery.id FROM notification_deliveries delivery
			JOIN push_endpoints endpoint ON endpoint.id = delivery.endpoint_id
			JOIN push_endpoint_tokens token ON token.endpoint_id = endpoint.id
			JOIN app_installations installation ON installation.id = endpoint.installation_id
			WHERE ((delivery.state = 'pending' AND delivery.next_attempt_at <= now())
			   OR (delivery.state = 'sending' AND delivery.locked_until < now()))
			  AND endpoint.status = 'active' AND installation.disabled_at IS NULL
			ORDER BY delivery.next_attempt_at, delivery.created_at, delivery.id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		), claimed AS (
			UPDATE notification_deliveries delivery
			SET state = 'sending', attempts = attempts + 1,
				locked_until = now() + ($2 * interval '1 second'), lock_token = gen_random_uuid()
			FROM candidates WHERE delivery.id = candidates.id
			RETURNING delivery.*
		)
		SELECT claimed.id, claimed.lock_token, endpoint.id, endpoint.transport, claimed.kind, token.token,
			endpoint.environment, installation.app_id, claimed.match_id, endpoint.frequent_updates_enabled,
			claimed.payload, claimed.collapse_key, claimed.priority, claimed.attempts
		FROM claimed
		JOIN push_endpoints endpoint ON endpoint.id = claimed.endpoint_id
		JOIN push_endpoint_tokens token ON token.endpoint_id = endpoint.id
		JOIN app_installations installation ON installation.id = endpoint.installation_id`, limit, int64(lockDuration.Seconds()))
	if err != nil {
		return nil, mapNotificationError(err)
	}
	defer rows.Close()

	deliveries := make([]notification.Delivery, 0, limit)
	for rows.Next() {
		var delivery notification.Delivery
		var kind, priority string
		if err := rows.Scan(
			&delivery.ID, &delivery.LockToken, &delivery.EndpointID, &delivery.Transport, &kind, &delivery.Token,
			&delivery.Environment, &delivery.AppID, &delivery.MatchID, &delivery.FrequentUpdatesEnabled,
			&delivery.Payload, &delivery.CollapseKey, &priority, &delivery.Attempts,
		); err != nil {
			return nil, mapNotificationError(err)
		}
		delivery.Kind = notification.DeliveryKind(kind)
		delivery.Priority = notification.Priority(priority)
		deliveries = append(deliveries, delivery)
	}
	return deliveries, mapNotificationError(rows.Err())
}

func (store *Store) CompleteDelivery(ctx context.Context, id, lockToken, providerMessageID string) error {
	result, err := store.pool.Exec(ctx, `
		UPDATE notification_deliveries SET state = 'sent', sent_at = now(), locked_until = NULL,
			lock_token = NULL, completed_at = now(), provider_message_id = $3, last_error = NULL
		WHERE id = $1 AND state = 'sending' AND lock_token = $2`, id, lockToken, providerMessageID)
	if err != nil {
		return mapNotificationError(err)
	}
	if result.RowsAffected() != 1 {
		return notification.ErrLeaseLost
	}
	return nil
}

func (store *Store) RetryDelivery(ctx context.Context, id, lockToken string, nextAttempt time.Time, reason string, terminal bool) error {
	result, err := store.pool.Exec(ctx, `
		UPDATE notification_deliveries SET state = CASE WHEN $5 THEN 'failed' ELSE 'pending' END,
			next_attempt_at = $3, locked_until = NULL, lock_token = NULL, last_error = $4,
			completed_at = CASE WHEN $5 THEN now() ELSE NULL END
		WHERE id = $1 AND state = 'sending' AND lock_token = $2`, id, lockToken, nextAttempt, reason, terminal)
	if err != nil {
		return mapNotificationError(err)
	}
	if result.RowsAffected() != 1 {
		return notification.ErrLeaseLost
	}
	return nil
}

func (store *Store) InvalidateEndpoint(ctx context.Context, id, reason string) error {
	_, err := store.pool.Exec(ctx, `
		WITH invalidated AS (
			UPDATE push_endpoints SET status = 'invalid', last_failure_at = now(), last_failure_reason = $2
			WHERE id = $1 RETURNING id
		)
		UPDATE notification_deliveries delivery
		SET state = 'failed', completed_at = now(), locked_until = NULL, lock_token = NULL,
			last_error = $2
		FROM invalidated
		WHERE delivery.endpoint_id = invalidated.id AND delivery.state IN ('pending', 'sending')`, id, reason)
	return mapNotificationError(err)
}

func mapNotificationError(err error) error {
	if err == nil {
		return nil
	}
	mapped := mapError(err)
	if errors.Is(mapped, football.ErrInvalid) {
		return fmt.Errorf("%w: %v", notification.ErrInvalid, mapped)
	}
	return mapped
}

var _ notification.Store = (*Store)(nil)
