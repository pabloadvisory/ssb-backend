package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (store *Store) ListMatchBroadcasts(ctx context.Context, matchID, countryCode string) (football.MatchBroadcasts, error) {
	result := football.MatchBroadcasts{MatchID: matchID, Data: make([]football.MatchBroadcast, 0)}
	if err := ensureFootballMatch(ctx, store.pool, matchID); err != nil {
		return football.MatchBroadcasts{}, err
	}

	rows, err := store.pool.Query(ctx, `
		SELECT broadcast.id
		FROM match_broadcasts broadcast
		WHERE broadcast.match_id = $1
		  AND (
			$2 = ''
			OR broadcast.availability_scope = 'global'
			OR (
				broadcast.availability_scope = 'territorial'
				AND EXISTS (
					SELECT 1 FROM match_broadcast_regions region
					WHERE region.broadcast_id = broadcast.id
					  AND region.country_code = upper($2)::char(2)
				)
			)
		  )
		ORDER BY broadcast.starts_at NULLS LAST, broadcast.network_name,
			broadcast.service_name NULLS FIRST, broadcast.id`, matchID, countryCode)
	if err != nil {
		return football.MatchBroadcasts{}, mapError(err)
	}

	broadcastIDs := make([]string, 0)
	for rows.Next() {
		var broadcastID string
		if err := rows.Scan(&broadcastID); err != nil {
			rows.Close()
			return football.MatchBroadcasts{}, mapError(err)
		}
		broadcastIDs = append(broadcastIDs, broadcastID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return football.MatchBroadcasts{}, mapError(err)
	}
	rows.Close()

	for _, broadcastID := range broadcastIDs {
		broadcast, err := loadMatchBroadcast(ctx, store.pool, broadcastID)
		if err != nil {
			return football.MatchBroadcasts{}, err
		}
		result.Data = append(result.Data, broadcast)
	}
	return result, nil
}

func loadMatchBroadcast(ctx context.Context, queryer repositoryQuerier, broadcastID string) (football.MatchBroadcast, error) {
	var broadcast football.MatchBroadcast
	err := queryer.QueryRow(ctx, `
		SELECT id, match_id, source, external_id, network_name, service_name, kind,
			availability_scope, language_tags, starts_at, ends_at, is_free,
			requires_subscription, web_url, deep_link_url, status, observed_at, metadata
		FROM match_broadcasts
		WHERE id = $1`, broadcastID).Scan(
		&broadcast.ID, &broadcast.MatchID, &broadcast.Source, &broadcast.ExternalID,
		&broadcast.NetworkName, &broadcast.ServiceName, &broadcast.Kind,
		&broadcast.AvailabilityScope, &broadcast.LanguageTags, &broadcast.StartsAt,
		&broadcast.EndsAt, &broadcast.IsFree, &broadcast.RequiresSubscription,
		&broadcast.WebURL, &broadcast.DeepLinkURL, &broadcast.Status,
		&broadcast.ObservedAt, &broadcast.Metadata,
	)
	if err != nil {
		return football.MatchBroadcast{}, mapError(err)
	}
	if broadcast.LanguageTags == nil {
		broadcast.LanguageTags = make([]string, 0)
	}
	broadcast.Regions = make([]string, 0)

	rows, err := queryer.Query(ctx, `
		SELECT country_code::text
		FROM match_broadcast_regions
		WHERE broadcast_id = $1
		ORDER BY country_code`, broadcastID)
	if err != nil {
		return football.MatchBroadcast{}, mapError(err)
	}
	for rows.Next() {
		var countryCode string
		if err := rows.Scan(&countryCode); err != nil {
			rows.Close()
			return football.MatchBroadcast{}, mapError(err)
		}
		broadcast.Regions = append(broadcast.Regions, countryCode)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return football.MatchBroadcast{}, mapError(err)
	}
	rows.Close()
	return broadcast, nil
}

func (store *Store) UpsertMatchBroadcast(
	ctx context.Context,
	matchID, source, externalID string,
	command football.UpsertMatchBroadcast,
) (football.MatchBroadcast, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return football.MatchBroadcast{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureFootballMatch(ctx, tx, matchID); err != nil {
		return football.MatchBroadcast{}, err
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('broadcast:' || $1 || ':' || $2, 0))`, source, externalID); err != nil {
		return football.MatchBroadcast{}, mapError(err)
	}

	languageTags := command.LanguageTags
	if languageTags == nil {
		languageTags = make([]string, 0)
	}
	var broadcastID, existingMatchID string
	err = tx.QueryRow(ctx, `
		SELECT id, match_id FROM match_broadcasts
		WHERE source = $1 AND external_id = $2
		FOR UPDATE`, source, externalID).Scan(&broadcastID, &existingMatchID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `
			INSERT INTO match_broadcasts (
				match_id, source, external_id, network_name, service_name, kind,
				availability_scope, language_tags, starts_at, ends_at, is_free,
				requires_subscription, web_url, deep_link_url, status, observed_at, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			RETURNING id`, matchID, source, externalID, command.NetworkName,
			command.ServiceName, command.Kind, command.AvailabilityScope, languageTags,
			command.StartsAt, command.EndsAt, command.IsFree, command.RequiresSubscription,
			command.WebURL, command.DeepLinkURL, command.Status, command.ObservedAt,
			repositoryMetadata(command.Metadata)).Scan(&broadcastID)
		if err != nil {
			return football.MatchBroadcast{}, mapError(err)
		}
	case err != nil:
		return football.MatchBroadcast{}, mapError(err)
	case existingMatchID != matchID:
		return football.MatchBroadcast{}, fmt.Errorf(
			"%w: broadcast source and external_id belong to another match", football.ErrConflict,
		)
	default:
		_, err = tx.Exec(ctx, `
			UPDATE match_broadcasts SET
				network_name = $2, service_name = $3, kind = $4,
				availability_scope = $5, language_tags = $6, starts_at = $7, ends_at = $8,
				is_free = $9, requires_subscription = $10, web_url = $11,
				deep_link_url = $12, status = $13, observed_at = $14, metadata = $15
			WHERE id = $1`, broadcastID, command.NetworkName, command.ServiceName,
			command.Kind, command.AvailabilityScope, languageTags, command.StartsAt,
			command.EndsAt, command.IsFree, command.RequiresSubscription, command.WebURL,
			command.DeepLinkURL, command.Status, command.ObservedAt,
			repositoryMetadata(command.Metadata))
		if err != nil {
			return football.MatchBroadcast{}, mapError(err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM match_broadcast_regions WHERE broadcast_id = $1`, broadcastID); err != nil {
		return football.MatchBroadcast{}, mapError(err)
	}
	for _, countryCode := range command.Regions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO match_broadcast_regions (broadcast_id, country_code)
			VALUES ($1, upper($2))
			ON CONFLICT (broadcast_id, country_code) DO NOTHING`, broadcastID, countryCode); err != nil {
			return football.MatchBroadcast{}, mapError(err)
		}
	}

	broadcast, err := loadMatchBroadcast(ctx, tx, broadcastID)
	if err != nil {
		return football.MatchBroadcast{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return football.MatchBroadcast{}, mapError(err)
	}
	return broadcast, nil
}
