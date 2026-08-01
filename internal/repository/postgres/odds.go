package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

type repositoryQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func repositoryMetadata(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}

func ensureFootballMatch(ctx context.Context, queryer repositoryQuerier, matchID string) error {
	var exists int
	return mapError(queryer.QueryRow(ctx, `SELECT 1 FROM matches WHERE id = $1`, matchID).Scan(&exists))
}

func (store *Store) ListMatchOdds(ctx context.Context, matchID, bookmakerSlug string) (football.MatchOdds, error) {
	result := football.MatchOdds{MatchID: matchID, Data: make([]football.MatchOddsSnapshot, 0)}
	if err := ensureFootballMatch(ctx, store.pool, matchID); err != nil {
		return football.MatchOdds{}, err
	}

	rows, err := store.pool.Query(ctx, `
		SELECT latest.id
		FROM (
			SELECT DISTINCT ON (snapshot.bookmaker_id)
				snapshot.id, snapshot.bookmaker_id, snapshot.observed_at, bookmaker.slug
			FROM odds_snapshots snapshot
			JOIN bookmakers bookmaker ON bookmaker.id = snapshot.bookmaker_id
			WHERE snapshot.match_id = $1
			  AND ($2 = '' OR bookmaker.slug = $2)
			ORDER BY snapshot.bookmaker_id, snapshot.observed_at DESC, snapshot.id DESC
		) latest
		ORDER BY latest.slug, latest.id`, matchID, bookmakerSlug)
	if err != nil {
		return football.MatchOdds{}, mapError(err)
	}

	snapshotIDs := make([]string, 0)
	for rows.Next() {
		var snapshotID string
		if err := rows.Scan(&snapshotID); err != nil {
			rows.Close()
			return football.MatchOdds{}, mapError(err)
		}
		snapshotIDs = append(snapshotIDs, snapshotID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return football.MatchOdds{}, mapError(err)
	}
	rows.Close()

	for _, snapshotID := range snapshotIDs {
		snapshot, err := loadMatchOddsSnapshot(ctx, store.pool, snapshotID)
		if err != nil {
			return football.MatchOdds{}, err
		}
		result.Data = append(result.Data, snapshot)
	}
	return result, nil
}

func loadMatchOddsSnapshot(ctx context.Context, queryer repositoryQuerier, snapshotID string) (football.MatchOddsSnapshot, error) {
	var snapshot football.MatchOddsSnapshot
	err := queryer.QueryRow(ctx, `
		SELECT snapshot.id, snapshot.match_id, snapshot.source, snapshot.external_id,
			bookmaker.slug, bookmaker.name, bookmaker.logo_url,
			snapshot.observed_at, snapshot.valid_until, snapshot.metadata
		FROM odds_snapshots snapshot
		JOIN bookmakers bookmaker ON bookmaker.id = snapshot.bookmaker_id
		WHERE snapshot.id = $1`, snapshotID).Scan(
		&snapshot.ID, &snapshot.MatchID, &snapshot.Source, &snapshot.ExternalID,
		&snapshot.BookmakerSlug, &snapshot.BookmakerName, &snapshot.BookmakerLogoURL,
		&snapshot.ObservedAt, &snapshot.ValidUntil, &snapshot.Metadata,
	)
	if err != nil {
		return football.MatchOddsSnapshot{}, mapError(err)
	}
	snapshot.Markets = make([]football.OddsMarket, 0)

	rows, err := queryer.Query(ctx, `
		SELECT id, market_key, name, status, metadata
		FROM odds_markets
		WHERE snapshot_id = $1
		ORDER BY market_key, id`, snapshotID)
	if err != nil {
		return football.MatchOddsSnapshot{}, mapError(err)
	}

	type marketRecord struct {
		id     string
		market football.OddsMarket
	}
	markets := make([]marketRecord, 0)
	for rows.Next() {
		record := marketRecord{market: football.OddsMarket{Selections: make([]football.OddsSelection, 0)}}
		if err := rows.Scan(
			&record.id, &record.market.Key, &record.market.Name, &record.market.Status, &record.market.Metadata,
		); err != nil {
			rows.Close()
			return football.MatchOddsSnapshot{}, mapError(err)
		}
		markets = append(markets, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return football.MatchOddsSnapshot{}, mapError(err)
	}
	rows.Close()

	for _, record := range markets {
		selectionRows, err := queryer.Query(ctx, `
			SELECT selection_key, name, line, decimal_odds, result, metadata
			FROM odds_selections
			WHERE market_id = $1
			ORDER BY selection_key, line NULLS FIRST, id`, record.id)
		if err != nil {
			return football.MatchOddsSnapshot{}, mapError(err)
		}
		market := record.market
		for selectionRows.Next() {
			var selection football.OddsSelection
			if err := selectionRows.Scan(
				&selection.Key, &selection.Name, &selection.Line, &selection.DecimalOdds,
				&selection.Result, &selection.Metadata,
			); err != nil {
				selectionRows.Close()
				return football.MatchOddsSnapshot{}, mapError(err)
			}
			market.Selections = append(market.Selections, selection)
		}
		if err := selectionRows.Err(); err != nil {
			selectionRows.Close()
			return football.MatchOddsSnapshot{}, mapError(err)
		}
		selectionRows.Close()
		snapshot.Markets = append(snapshot.Markets, market)
	}
	return snapshot, nil
}

func (store *Store) UpsertMatchOdds(
	ctx context.Context,
	matchID, source, externalID string,
	command football.UpsertOddsSnapshot,
) (football.MatchOddsSnapshot, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return football.MatchOddsSnapshot{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureFootballMatch(ctx, tx, matchID); err != nil {
		return football.MatchOddsSnapshot{}, err
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('odds:' || $1 || ':' || $2, 0))`, source, externalID); err != nil {
		return football.MatchOddsSnapshot{}, mapError(err)
	}

	var bookmakerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO bookmakers (slug, name, logo_url, website_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (slug) DO UPDATE SET
			name = EXCLUDED.name,
			logo_url = EXCLUDED.logo_url,
			website_url = EXCLUDED.website_url
		RETURNING id`, command.BookmakerSlug, command.BookmakerName,
		command.BookmakerLogoURL, command.BookmakerWebsiteURL).Scan(&bookmakerID)
	if err != nil {
		return football.MatchOddsSnapshot{}, mapError(err)
	}

	var snapshotID, existingMatchID string
	err = tx.QueryRow(ctx, `
		SELECT id, match_id FROM odds_snapshots
		WHERE source = $1 AND external_id = $2
		FOR UPDATE`, source, externalID).Scan(&snapshotID, &existingMatchID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `
			INSERT INTO odds_snapshots (
				match_id, bookmaker_id, source, external_id, observed_at, valid_until, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7)
			RETURNING id`, matchID, bookmakerID, source, externalID, command.ObservedAt,
			command.ValidUntil, repositoryMetadata(command.Metadata)).Scan(&snapshotID)
		if err != nil {
			return football.MatchOddsSnapshot{}, mapError(err)
		}
	case err != nil:
		return football.MatchOddsSnapshot{}, mapError(err)
	case existingMatchID != matchID:
		return football.MatchOddsSnapshot{}, fmt.Errorf(
			"%w: odds source and external_id belong to another match", football.ErrConflict,
		)
	default:
		_, err = tx.Exec(ctx, `
			UPDATE odds_snapshots SET
				bookmaker_id = $2, observed_at = $3, valid_until = $4, metadata = $5
			WHERE id = $1`, snapshotID, bookmakerID, command.ObservedAt,
			command.ValidUntil, repositoryMetadata(command.Metadata))
		if err != nil {
			return football.MatchOddsSnapshot{}, mapError(err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM odds_markets WHERE snapshot_id = $1`, snapshotID); err != nil {
		return football.MatchOddsSnapshot{}, mapError(err)
	}
	for _, market := range command.Markets {
		var marketID string
		err := tx.QueryRow(ctx, `
			INSERT INTO odds_markets (snapshot_id, market_key, name, status, metadata)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id`, snapshotID, market.Key, market.Name, market.Status,
			repositoryMetadata(market.Metadata)).Scan(&marketID)
		if err != nil {
			return football.MatchOddsSnapshot{}, mapError(err)
		}
		for _, selection := range market.Selections {
			if _, err := tx.Exec(ctx, `
				INSERT INTO odds_selections (
					market_id, selection_key, name, line, decimal_odds, result, metadata
				) VALUES ($1,$2,$3,$4,$5,$6,$7)`, marketID, selection.Key, selection.Name,
				selection.Line, selection.DecimalOdds, selection.Result,
				repositoryMetadata(selection.Metadata)); err != nil {
				return football.MatchOddsSnapshot{}, mapError(err)
			}
		}
	}

	snapshot, err := loadMatchOddsSnapshot(ctx, tx, snapshotID)
	if err != nil {
		return football.MatchOddsSnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return football.MatchOddsSnapshot{}, mapError(err)
	}
	return snapshot, nil
}
