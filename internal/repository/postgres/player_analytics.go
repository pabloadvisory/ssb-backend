package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

const canonicalPlayerSpatialOrientation = "attacking_left_to_right"

func ensureFootballPlayer(ctx context.Context, queryer repositoryQuerier, playerID string) error {
	var exists int
	return mapError(queryer.QueryRow(ctx, `SELECT 1 FROM players WHERE person_id = $1`, playerID).Scan(&exists))
}

func ensureFootballTeam(ctx context.Context, queryer repositoryQuerier, teamID string) error {
	var exists int
	return mapError(queryer.QueryRow(ctx, `SELECT 1 FROM teams WHERE id = $1`, teamID).Scan(&exists))
}

func loadPlayerTraits(ctx context.Context, queryer repositoryQuerier, snapshotID string) (football.PlayerTraits, error) {
	var result football.PlayerTraits
	err := queryer.QueryRow(ctx, `
		SELECT id, person_id, team_id, league_id, season_id, source, external_id,
			position_group, minimum_minutes, cohort_size, player_minutes, observed_at, metadata
		FROM player_trait_snapshots
		WHERE id = $1`, snapshotID).Scan(
		&result.ID, &result.PlayerID, &result.TeamID, &result.LeagueID, &result.SeasonID,
		&result.Source, &result.ExternalID, &result.PositionGroup, &result.MinimumMinutes,
		&result.CohortSize, &result.PlayerMinutes, &result.ObservedAt, &result.Metadata,
	)
	if err != nil {
		return football.PlayerTraits{}, mapError(err)
	}

	result.Metrics = make([]football.TraitMetric, 0)
	rows, err := queryer.Query(ctx, `
		SELECT metric_key, label, category, raw_value, per_90_value, percentile, unit, direction
		FROM player_trait_metrics
		WHERE snapshot_id = $1
		ORDER BY category, metric_key`, snapshotID)
	if err != nil {
		return football.PlayerTraits{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var metric football.TraitMetric
		if err := rows.Scan(
			&metric.Key, &metric.Label, &metric.Category, &metric.RawValue,
			&metric.Per90Value, &metric.Percentile, &metric.Unit, &metric.Direction,
		); err != nil {
			return football.PlayerTraits{}, mapError(err)
		}
		result.Metrics = append(result.Metrics, metric)
	}
	return result, mapError(rows.Err())
}

func (store *Store) GetPlayerTraits(
	ctx context.Context,
	playerID string,
	filter football.PlayerAnalyticsFilter,
) (football.PlayerTraits, error) {
	if err := ensureFootballPlayer(ctx, store.pool, playerID); err != nil {
		return football.PlayerTraits{}, err
	}

	var snapshotID string
	err := store.pool.QueryRow(ctx, `
		SELECT id
		FROM player_trait_snapshots
		WHERE person_id = $1
		  AND ($2 = '' OR season_id = NULLIF($2, '')::uuid)
		  AND ($3 = '' OR league_id = NULLIF($3, '')::uuid)
		  AND ($4 = '' OR source = $4)
		ORDER BY observed_at DESC, id DESC
		LIMIT 1`, playerID, filter.SeasonID, filter.LeagueID, filter.Source).Scan(&snapshotID)
	if err != nil {
		return football.PlayerTraits{}, mapError(err)
	}
	return loadPlayerTraits(ctx, store.pool, snapshotID)
}

func (store *Store) GetPlayerHeatmap(
	ctx context.Context,
	playerID string,
	filter football.PlayerAnalyticsFilter,
) (football.PlayerHeatmap, error) {
	if err := ensureFootballPlayer(ctx, store.pool, playerID); err != nil {
		return football.PlayerHeatmap{}, err
	}

	result := football.PlayerHeatmap{
		PlayerID: playerID,
		CoordinateSystem: football.CoordinateSystem{
			XMin: 0, XMax: 100, YMin: 0, YMax: 100,
			Origin: "bottom_left", Orientation: canonicalPlayerSpatialOrientation,
		},
		Data: make([]football.HeatmapPoint, 0),
	}
	rows, err := store.pool.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (snapshot.match_id) snapshot.id
			FROM player_spatial_snapshots snapshot
			JOIN matches match ON match.id = snapshot.match_id
			WHERE snapshot.person_id = $1
			  AND ($2 = '' OR match.season_id = NULLIF($2, '')::uuid)
			  AND ($3 = '' OR match.league_id = NULLIF($3, '')::uuid)
			  AND ($4 = '' OR snapshot.match_id = NULLIF($4, '')::uuid)
			  AND ($5 = '' OR snapshot.source = $5)
			ORDER BY snapshot.match_id, snapshot.observed_at DESC, snapshot.id DESC
		)
		SELECT point.x, point.y, sum(point.intensity), count(*)::integer
		FROM latest
		JOIN player_touch_points point ON point.snapshot_id = latest.id
		GROUP BY point.x, point.y
		ORDER BY point.y, point.x`,
		playerID, filter.SeasonID, filter.LeagueID, filter.MatchID, filter.Source)
	if err != nil {
		return football.PlayerHeatmap{}, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var point football.HeatmapPoint
		if err := rows.Scan(&point.X, &point.Y, &point.Intensity, &point.Touches); err != nil {
			return football.PlayerHeatmap{}, mapError(err)
		}
		result.Data = append(result.Data, point)
	}
	return result, mapError(rows.Err())
}

func boundedPlayerShotLimit(limit int) int {
	if limit < 1 {
		return 100
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (store *Store) ListPlayerShots(
	ctx context.Context,
	playerID string,
	filter football.PlayerAnalyticsFilter,
) ([]football.PlayerShot, error) {
	if err := ensureFootballPlayer(ctx, store.pool, playerID); err != nil {
		return nil, err
	}

	limit := boundedPlayerShotLimit(filter.Limit)
	result := make([]football.PlayerShot, 0, limit)
	rows, err := store.pool.Query(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (snapshot.match_id) snapshot.id, snapshot.match_id
			FROM player_spatial_snapshots snapshot
			JOIN matches match ON match.id = snapshot.match_id
			WHERE snapshot.person_id = $1
			  AND ($2 = '' OR match.season_id = NULLIF($2, '')::uuid)
			  AND ($3 = '' OR match.league_id = NULLIF($3, '')::uuid)
			  AND ($4 = '' OR snapshot.match_id = NULLIF($4, '')::uuid)
			  AND ($5 = '' OR snapshot.source = $5)
			ORDER BY snapshot.match_id, snapshot.observed_at DESC, snapshot.id DESC
		)
		SELECT shot.id, latest.match_id, shot.sequence, shot.minute, shot.stoppage_minute,
			shot.x, shot.y, shot.expected_goals, shot.outcome, shot.body_part, shot.shot_type
		FROM latest
		JOIN player_shots shot ON shot.snapshot_id = latest.id
		WHERE ($6::smallint IS NULL OR
			(COALESCE(shot.minute, -1), shot.sequence) < ($6::smallint, $7::integer))
		ORDER BY COALESCE(shot.minute, -1) DESC, shot.sequence DESC, latest.match_id DESC, shot.id DESC
		LIMIT $8`, playerID, filter.SeasonID, filter.LeagueID, filter.MatchID, filter.Source,
		filter.BeforeMinute, filter.BeforeSequence, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var shot football.PlayerShot
		if err := rows.Scan(
			&shot.ID, &shot.MatchID, &shot.Sequence, &shot.Minute, &shot.StoppageMinute,
			&shot.X, &shot.Y, &shot.ExpectedGoals, &shot.Outcome, &shot.BodyPart, &shot.ShotType,
		); err != nil {
			return nil, mapError(err)
		}
		result = append(result, shot)
	}
	return result, mapError(rows.Err())
}

func scanPlayerValuation(row scanner) (football.PlayerValuation, error) {
	var valuation football.PlayerValuation
	err := row.Scan(
		&valuation.ID, &valuation.PlayerID, &valuation.TeamID, &valuation.AmountMinor,
		&valuation.Currency, &valuation.ValuationDate, &valuation.Source,
		&valuation.ExternalID, &valuation.ObservedAt, &valuation.Metadata,
	)
	return valuation, mapError(err)
}

func (store *Store) GetPlayerValuation(ctx context.Context, playerID string) (football.PlayerValuation, error) {
	if err := ensureFootballPlayer(ctx, store.pool, playerID); err != nil {
		return football.PlayerValuation{}, err
	}
	return scanPlayerValuation(store.pool.QueryRow(ctx, `
		SELECT id, person_id, team_id, amount_minor, currency, valued_on,
			source, external_id, observed_at, metadata
		FROM player_valuations
		WHERE person_id = $1
		ORDER BY valued_on DESC, observed_at DESC, id DESC
		LIMIT 1`, playerID))
}

func (store *Store) UpsertPlayerTraits(
	ctx context.Context,
	playerID, source, externalID string,
	command football.UpsertPlayerTraits,
) (football.PlayerTraits, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return football.PlayerTraits{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureFootballPlayer(ctx, tx, playerID); err != nil {
		return football.PlayerTraits{}, err
	}
	var seasonExists int
	if err := tx.QueryRow(ctx, `
		SELECT 1 FROM seasons WHERE id = $1 AND league_id = $2`,
		command.SeasonID, command.LeagueID).Scan(&seasonExists); err != nil {
		return football.PlayerTraits{}, mapError(err)
	}
	if command.TeamID != nil {
		var teamExists int
		if err := tx.QueryRow(ctx, `
			SELECT 1 FROM season_teams WHERE season_id = $1 AND team_id = $2`,
			command.SeasonID, *command.TeamID).Scan(&teamExists); err != nil {
			return football.PlayerTraits{}, mapError(err)
		}
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('player-traits:' || $1 || ':' || $2, 0))`,
		source, externalID); err != nil {
		return football.PlayerTraits{}, mapError(err)
	}

	var snapshotID, existingPlayerID string
	err = tx.QueryRow(ctx, `
		SELECT id, person_id FROM player_trait_snapshots
		WHERE source = $1 AND external_id = $2
		FOR UPDATE`, source, externalID).Scan(&snapshotID, &existingPlayerID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `
			INSERT INTO player_trait_snapshots (
				person_id, team_id, league_id, season_id, source, external_id,
				position_group, minimum_minutes, cohort_size, player_minutes,
				observed_at, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING id`, playerID, command.TeamID, command.LeagueID, command.SeasonID,
			source, externalID, command.PositionGroup, command.MinimumMinutes,
			command.CohortSize, command.PlayerMinutes, command.ObservedAt,
			repositoryMetadata(command.Metadata)).Scan(&snapshotID)
		if err != nil {
			return football.PlayerTraits{}, mapError(err)
		}
	case err != nil:
		return football.PlayerTraits{}, mapError(err)
	case existingPlayerID != playerID:
		return football.PlayerTraits{}, fmt.Errorf(
			"%w: trait source and external_id belong to another player", football.ErrConflict,
		)
	default:
		_, err = tx.Exec(ctx, `
			UPDATE player_trait_snapshots SET
				team_id = $2, league_id = $3, season_id = $4, position_group = $5,
				minimum_minutes = $6, cohort_size = $7, player_minutes = $8,
				observed_at = $9, metadata = $10
			WHERE id = $1`, snapshotID, command.TeamID, command.LeagueID, command.SeasonID,
			command.PositionGroup, command.MinimumMinutes, command.CohortSize,
			command.PlayerMinutes, command.ObservedAt, repositoryMetadata(command.Metadata))
		if err != nil {
			return football.PlayerTraits{}, mapError(err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM player_trait_metrics WHERE snapshot_id = $1`, snapshotID); err != nil {
		return football.PlayerTraits{}, mapError(err)
	}
	for _, metric := range command.Metrics {
		if _, err := tx.Exec(ctx, `
			INSERT INTO player_trait_metrics (
				snapshot_id, metric_key, label, category, raw_value,
				per_90_value, percentile, unit, direction
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, snapshotID, metric.Key,
			metric.Label, metric.Category, metric.RawValue, metric.Per90Value,
			metric.Percentile, metric.Unit, metric.Direction); err != nil {
			return football.PlayerTraits{}, mapError(err)
		}
	}

	result, err := loadPlayerTraits(ctx, tx, snapshotID)
	if err != nil {
		return football.PlayerTraits{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return football.PlayerTraits{}, mapError(err)
	}
	return result, nil
}

func canonicalizePlayerSpatial(command football.UpsertPlayerSpatial) (football.UpsertPlayerSpatial, error) {
	switch command.Orientation {
	case canonicalPlayerSpatialOrientation:
		return command, nil
	case "attacking_right_to_left":
		for index := range command.Touches {
			command.Touches[index].X = 100 - command.Touches[index].X
			command.Touches[index].Y = 100 - command.Touches[index].Y
		}
		for index := range command.Shots {
			command.Shots[index].X = 100 - command.Shots[index].X
			command.Shots[index].Y = 100 - command.Shots[index].Y
		}
		command.Orientation = canonicalPlayerSpatialOrientation
		return command, nil
	default:
		return football.UpsertPlayerSpatial{}, fmt.Errorf(
			"%w: spatial orientation is invalid", football.ErrInvalid,
		)
	}
}

func (store *Store) UpsertPlayerSpatial(
	ctx context.Context,
	matchID, playerID, source, externalID string,
	command football.UpsertPlayerSpatial,
) error {
	command, err := canonicalizePlayerSpatial(command)
	if err != nil {
		return err
	}
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureFootballPlayer(ctx, tx, playerID); err != nil {
		return err
	}
	var homeTeamID, awayTeamID string
	if err := tx.QueryRow(ctx, `
		SELECT home_team_id, away_team_id FROM matches WHERE id = $1`, matchID).
		Scan(&homeTeamID, &awayTeamID); err != nil {
		return mapError(err)
	}
	if command.TeamID != homeTeamID && command.TeamID != awayTeamID {
		return fmt.Errorf("%w: spatial team does not participate in the match", football.ErrInvalid)
	}
	for _, shot := range command.Shots {
		if shot.MatchEventID == nil {
			continue
		}
		var eventExists int
		if err := tx.QueryRow(ctx, `
			SELECT 1 FROM match_events
			WHERE id = $1 AND match_id = $2 AND primary_person_id = $3`,
			*shot.MatchEventID, matchID, playerID).Scan(&eventExists); err != nil {
			return mapError(err)
		}
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('player-spatial:' || $1 || ':' || $2, 0))`,
		source, externalID); err != nil {
		return mapError(err)
	}

	var snapshotID, existingMatchID, existingPlayerID string
	err = tx.QueryRow(ctx, `
		SELECT id, match_id, person_id FROM player_spatial_snapshots
		WHERE source = $1 AND external_id = $2
		FOR UPDATE`, source, externalID).Scan(&snapshotID, &existingMatchID, &existingPlayerID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `
			INSERT INTO player_spatial_snapshots (
				match_id, person_id, team_id, source, external_id, orientation,
				observed_at, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id`, matchID, playerID, command.TeamID, source, externalID,
			command.Orientation, command.ObservedAt, repositoryMetadata(command.Metadata)).Scan(&snapshotID)
		if err != nil {
			return mapError(err)
		}
	case err != nil:
		return mapError(err)
	case existingMatchID != matchID || existingPlayerID != playerID:
		return fmt.Errorf(
			"%w: spatial source and external_id belong to another match or player", football.ErrConflict,
		)
	default:
		_, err = tx.Exec(ctx, `
			UPDATE player_spatial_snapshots SET
				team_id = $2, orientation = $3, observed_at = $4, metadata = $5
			WHERE id = $1`, snapshotID, command.TeamID, command.Orientation,
			command.ObservedAt, repositoryMetadata(command.Metadata))
		if err != nil {
			return mapError(err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM player_touch_points WHERE snapshot_id = $1`, snapshotID); err != nil {
		return mapError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM player_shots WHERE snapshot_id = $1`, snapshotID); err != nil {
		return mapError(err)
	}
	for _, point := range command.Touches {
		if _, err := tx.Exec(ctx, `
			INSERT INTO player_touch_points (
				snapshot_id, sequence, minute, stoppage_minute, x, y, intensity, touch_type
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, snapshotID, point.Sequence,
			point.Minute, point.StoppageMinute, point.X, point.Y, point.Intensity,
			point.TouchType); err != nil {
			return mapError(err)
		}
	}
	for _, shot := range command.Shots {
		if _, err := tx.Exec(ctx, `
			INSERT INTO player_shots (
				snapshot_id, sequence, match_event_id, minute, stoppage_minute,
				x, y, expected_goals, outcome, body_part, shot_type
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, snapshotID,
			shot.Sequence, shot.MatchEventID, shot.Minute, shot.StoppageMinute,
			shot.X, shot.Y, shot.ExpectedGoals, shot.Outcome, shot.BodyPart,
			shot.ShotType); err != nil {
			return mapError(err)
		}
	}
	return mapError(tx.Commit(ctx))
}

func (store *Store) UpsertPlayerValuation(
	ctx context.Context,
	playerID, source, externalID string,
	command football.UpsertPlayerValuation,
) (football.PlayerValuation, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return football.PlayerValuation{}, mapError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureFootballPlayer(ctx, tx, playerID); err != nil {
		return football.PlayerValuation{}, err
	}
	if command.TeamID != nil {
		if err := ensureFootballTeam(ctx, tx, *command.TeamID); err != nil {
			return football.PlayerValuation{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended('player-valuation:' || $1 || ':' || $2, 0))`,
		source, externalID); err != nil {
		return football.PlayerValuation{}, mapError(err)
	}

	var valuationID, existingPlayerID string
	err = tx.QueryRow(ctx, `
		SELECT id, person_id FROM player_valuations
		WHERE source = $1 AND external_id = $2
		FOR UPDATE`, source, externalID).Scan(&valuationID, &existingPlayerID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `
			INSERT INTO player_valuations (
				person_id, team_id, source, external_id, amount_minor, currency,
				valued_on, observed_at, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			RETURNING id`, playerID, command.TeamID, source, externalID, command.AmountMinor,
			command.Currency, command.ValuationDate, command.ObservedAt,
			repositoryMetadata(command.Metadata)).Scan(&valuationID)
		if err != nil {
			return football.PlayerValuation{}, mapError(err)
		}
	case err != nil:
		return football.PlayerValuation{}, mapError(err)
	case existingPlayerID != playerID:
		return football.PlayerValuation{}, fmt.Errorf(
			"%w: valuation source and external_id belong to another player", football.ErrConflict,
		)
	default:
		_, err = tx.Exec(ctx, `
			UPDATE player_valuations SET
				team_id = $2, amount_minor = $3, currency = $4, valued_on = $5,
				observed_at = $6, metadata = $7
			WHERE id = $1`, valuationID, command.TeamID, command.AmountMinor,
			command.Currency, command.ValuationDate, command.ObservedAt,
			repositoryMetadata(command.Metadata))
		if err != nil {
			return football.PlayerValuation{}, mapError(err)
		}
	}

	result, err := scanPlayerValuation(tx.QueryRow(ctx, `
		SELECT id, person_id, team_id, amount_minor, currency, valued_on,
			source, external_id, observed_at, metadata
		FROM player_valuations WHERE id = $1`, valuationID))
	if err != nil {
		return football.PlayerValuation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return football.PlayerValuation{}, mapError(err)
	}
	return result, nil
}
