package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
)

func authenticateInstallationTx(ctx context.Context, tx pgx.Tx, installationID string, secretHash []byte) error {
	var canonicalID string
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM app_installations
		WHERE id = $1 AND secret_hash = $2 AND disabled_at IS NULL
		FOR SHARE`, installationID, secretHash).Scan(&canonicalID)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.ErrUnauthorized
	}
	return mapNotificationError(err)
}

func (store *Store) SetPlayerFollow(
	ctx context.Context,
	installationID string,
	secretHash []byte,
	playerID string,
	command notification.SetPlayerFollow,
) (notification.PlayerFollow, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return notification.PlayerFollow{}, fmt.Errorf("begin player follow: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authenticateInstallationTx(ctx, tx, installationID, secretHash); err != nil {
		return notification.PlayerFollow{}, err
	}
	var canonicalPlayerID string
	err = tx.QueryRow(ctx, `SELECT person_id FROM players WHERE person_id = $1`, playerID).Scan(&canonicalPlayerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.PlayerFollow{}, football.ErrNotFound
	}
	if err != nil {
		return notification.PlayerFollow{}, mapNotificationError(err)
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(
			hashtextextended('player-follow:' || $1::uuid::text || ':' || $2::uuid::text, 0)
		)`, installationID, canonicalPlayerID); err != nil {
		return notification.PlayerFollow{}, mapNotificationError(err)
	}

	var follow notification.PlayerFollow
	err = tx.QueryRow(ctx, `
		WITH changed AS (
			INSERT INTO player_follows (installation_id, player_id, notifications_enabled)
			VALUES ($1, $2, $3)
			ON CONFLICT (installation_id, player_id) DO UPDATE
			SET notifications_enabled = EXCLUDED.notifications_enabled
			WHERE player_follows.notifications_enabled IS DISTINCT FROM EXCLUDED.notifications_enabled
			RETURNING installation_id, player_id, notifications_enabled, created_at, updated_at
		), followed AS (
			SELECT installation_id, player_id, notifications_enabled, created_at, updated_at FROM changed
			UNION ALL
			SELECT installation_id, player_id, notifications_enabled, created_at, updated_at
			FROM player_follows
			WHERE installation_id = $1 AND player_id = $2
				AND NOT EXISTS (SELECT 1 FROM changed)
		)
		SELECT followed.installation_id,
			person.id, person.display_name, person.first_name, person.last_name,
			person.birth_date, person.country_code, person.photo_url,
			player.position, player.detailed_position, player.preferred_foot, player.height_cm,
			followed.notifications_enabled, followed.created_at, followed.updated_at
		FROM followed
		JOIN people person ON person.id = followed.player_id
		JOIN players player ON player.person_id = followed.player_id`,
		installationID, canonicalPlayerID, command.NotificationsEnabled,
	).Scan(
		&follow.InstallationID,
		&follow.Player.ID, &follow.Player.DisplayName, &follow.Player.FirstName, &follow.Player.LastName,
		&follow.Player.BirthDate, &follow.Player.CountryCode, &follow.Player.PhotoURL,
		&follow.Player.Position, &follow.Player.DetailedPosition, &follow.Player.PreferredFoot, &follow.Player.HeightCM,
		&follow.NotificationsEnabled, &follow.FollowedAt, &follow.UpdatedAt,
	)
	if err != nil {
		return notification.PlayerFollow{}, mapNotificationError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return notification.PlayerFollow{}, mapNotificationError(err)
	}
	return follow, nil
}

func (store *Store) DeletePlayerFollow(ctx context.Context, installationID string, secretHash []byte, playerID string) error {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin player unfollow: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authenticateInstallationTx(ctx, tx, installationID, secretHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM player_follows
		WHERE installation_id = $1 AND player_id = $2`, installationID, playerID); err != nil {
		return mapNotificationError(err)
	}
	return mapNotificationError(tx.Commit(ctx))
}

func (store *Store) ListPlayerFollows(
	ctx context.Context,
	installationID string,
	secretHash []byte,
	filter notification.PlayerFollowFilter,
) ([]notification.PlayerFollow, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin player follows list: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authenticateInstallationTx(ctx, tx, installationID, secretHash); err != nil {
		return nil, err
	}
	var beforePlayerID any
	if filter.BeforePlayerID != "" {
		beforePlayerID = filter.BeforePlayerID
	}
	rows, err := tx.Query(ctx, `
		SELECT follow.installation_id,
			person.id, person.display_name, person.first_name, person.last_name,
			person.birth_date, person.country_code, person.photo_url,
			player.position, player.detailed_position, player.preferred_foot, player.height_cm,
			follow.notifications_enabled, follow.created_at, follow.updated_at
		FROM player_follows follow
		JOIN people person ON person.id = follow.player_id
		JOIN players player ON player.person_id = follow.player_id
		WHERE follow.installation_id = $1
			AND ($2::timestamptz IS NULL OR (follow.created_at, follow.player_id) < ($2, $3::uuid))
		ORDER BY follow.created_at DESC, follow.player_id DESC
		LIMIT $4`, installationID, filter.BeforeFollowedAt, beforePlayerID, filter.Limit)
	if err != nil {
		return nil, mapNotificationError(err)
	}
	defer rows.Close()

	follows := make([]notification.PlayerFollow, 0, filter.Limit)
	for rows.Next() {
		var follow notification.PlayerFollow
		if err := rows.Scan(
			&follow.InstallationID,
			&follow.Player.ID, &follow.Player.DisplayName, &follow.Player.FirstName, &follow.Player.LastName,
			&follow.Player.BirthDate, &follow.Player.CountryCode, &follow.Player.PhotoURL,
			&follow.Player.Position, &follow.Player.DetailedPosition, &follow.Player.PreferredFoot, &follow.Player.HeightCM,
			&follow.NotificationsEnabled, &follow.FollowedAt, &follow.UpdatedAt,
		); err != nil {
			return nil, mapNotificationError(err)
		}
		follows = append(follows, follow)
	}
	if err := rows.Err(); err != nil {
		return nil, mapNotificationError(err)
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, mapNotificationError(err)
	}
	return follows, nil
}

func (store *Store) GetNotificationPreferences(ctx context.Context, installationID string, secretHash []byte) (notification.NotificationPreferences, error) {
	var preferences notification.NotificationPreferences
	err := store.pool.QueryRow(ctx, `
		SELECT installation.id,
			COALESCE(preferences.match_updates_enabled, true),
			COALESCE(preferences.match_finished_enabled, true),
			COALESCE(preferences.followed_player_events_enabled, true),
			COALESCE(preferences.updated_at, installation.created_at)
		FROM app_installations installation
		LEFT JOIN installation_notification_preferences preferences
			ON preferences.installation_id = installation.id
		WHERE installation.id = $1 AND installation.secret_hash = $2
			AND installation.disabled_at IS NULL`, installationID, secretHash).Scan(
		&preferences.InstallationID,
		&preferences.MatchUpdatesEnabled,
		&preferences.MatchFinishedEnabled,
		&preferences.FollowedPlayerEventsEnabled,
		&preferences.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.NotificationPreferences{}, notification.ErrUnauthorized
	}
	return preferences, mapNotificationError(err)
}

func (store *Store) SetNotificationPreferences(
	ctx context.Context,
	installationID string,
	secretHash []byte,
	command notification.SetNotificationPreferences,
) (notification.NotificationPreferences, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return notification.NotificationPreferences{}, fmt.Errorf("begin notification preferences update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authenticateInstallationTx(ctx, tx, installationID, secretHash); err != nil {
		return notification.NotificationPreferences{}, err
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(
			hashtextextended('notification-preferences:' || $1::uuid::text, 0)
		)`, installationID); err != nil {
		return notification.NotificationPreferences{}, mapNotificationError(err)
	}
	var preferences notification.NotificationPreferences
	err = tx.QueryRow(ctx, `
		WITH changed AS (
			INSERT INTO installation_notification_preferences (
				installation_id, match_updates_enabled, match_finished_enabled, followed_player_events_enabled
			) VALUES ($1, $2, $3, $4)
			ON CONFLICT (installation_id) DO UPDATE SET
				match_updates_enabled = EXCLUDED.match_updates_enabled,
				match_finished_enabled = EXCLUDED.match_finished_enabled,
				followed_player_events_enabled = EXCLUDED.followed_player_events_enabled
			WHERE installation_notification_preferences.match_updates_enabled IS DISTINCT FROM EXCLUDED.match_updates_enabled
				OR installation_notification_preferences.match_finished_enabled IS DISTINCT FROM EXCLUDED.match_finished_enabled
				OR installation_notification_preferences.followed_player_events_enabled IS DISTINCT FROM EXCLUDED.followed_player_events_enabled
			RETURNING installation_id, match_updates_enabled, match_finished_enabled,
				followed_player_events_enabled, updated_at
		), preferences AS (
			SELECT installation_id, match_updates_enabled, match_finished_enabled,
				followed_player_events_enabled, updated_at FROM changed
			UNION ALL
			SELECT installation_id, match_updates_enabled, match_finished_enabled,
				followed_player_events_enabled, updated_at
			FROM installation_notification_preferences
			WHERE installation_id = $1 AND NOT EXISTS (SELECT 1 FROM changed)
		)
		SELECT installation_id, match_updates_enabled, match_finished_enabled,
			followed_player_events_enabled, updated_at
		FROM preferences`,
		installationID, command.MatchUpdatesEnabled, command.MatchFinishedEnabled, command.FollowedPlayerEventsEnabled,
	).Scan(
		&preferences.InstallationID,
		&preferences.MatchUpdatesEnabled,
		&preferences.MatchFinishedEnabled,
		&preferences.FollowedPlayerEventsEnabled,
		&preferences.UpdatedAt,
	)
	if err != nil {
		return notification.NotificationPreferences{}, mapNotificationError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return notification.NotificationPreferences{}, mapNotificationError(err)
	}
	return preferences, nil
}
