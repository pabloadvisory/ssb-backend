package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (store *Store) ReplaceMatchCoverage(
	ctx context.Context,
	matchID string,
	source string,
	update football.MatchCoverageUpdate,
) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("%w: coverage source is required", football.ErrInvalid)
	}

	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin match coverage replacement: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1 || ':match-coverage:' || $2, 0))`, source, matchID); err != nil {
		return mapError(err)
	}

	var homeTeamID, awayTeamID string
	if err := tx.QueryRow(ctx, `
		SELECT home_team_id, away_team_id FROM matches WHERE id = $1 FOR UPDATE`, matchID).Scan(
		&homeTeamID, &awayTeamID,
	); err != nil {
		return mapError(err)
	}
	participant := func(teamID string) bool {
		return teamID == homeTeamID || teamID == awayTeamID
	}
	if err := validateCoverageParticipants(update, participant); err != nil {
		return err
	}

	if update.TeamInfo != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM match_team_info WHERE match_id = $1`, matchID); err != nil {
			return mapError(err)
		}
		for _, item := range *update.TeamInfo {
			metadata, err := coverageMetadata(item.Metadata, source)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO match_team_info (match_id, team_id, formation, coach_id, metadata)
				VALUES ($1, $2, $3, $4, $5)`, matchID, item.TeamID, item.Formation, item.CoachID, metadata); err != nil {
				return mapError(err)
			}
		}
	}

	if update.Lineups != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM match_lineups WHERE match_id = $1`, matchID); err != nil {
			return mapError(err)
		}
		for _, item := range *update.Lineups {
			metadata, err := coverageMetadata(item.Metadata, source)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO match_lineups (
					match_id, team_id, person_id, position, grid_position,
					shirt_number, is_starter, is_captain, metadata
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				matchID, item.TeamID, item.PersonID, item.Position, item.GridPosition,
				item.ShirtNumber, item.IsStarter, item.IsCaptain, metadata,
			); err != nil {
				return mapError(err)
			}
		}
	}

	if update.TeamStatistics != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM match_team_statistics WHERE match_id = $1`, matchID); err != nil {
			return mapError(err)
		}
		for _, item := range *update.TeamStatistics {
			metadata, err := coverageMetadata(item.Metadata, source)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO match_team_statistics (
					match_id, team_id, possession, shots, shots_on_target, shots_off_target,
					blocked_shots, shots_inside_box, shots_outside_box, corners, passes,
					passes_completed, pass_accuracy, fouls, offsides, yellow_cards, red_cards,
					saves, tackles, interceptions, clearances, expected_goals, metadata
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`,
				matchID, item.TeamID, item.Possession, item.Shots, item.ShotsOnTarget, item.ShotsOffTarget,
				item.BlockedShots, item.ShotsInsideBox, item.ShotsOutsideBox, item.Corners, item.Passes,
				item.PassesCompleted, item.PassAccuracy, item.Fouls, item.Offsides, item.YellowCards,
				item.RedCards, item.Saves, item.Tackles, item.Interceptions, item.Clearances,
				item.ExpectedGoals, metadata,
			); err != nil {
				return mapError(err)
			}
		}
	}

	if update.PlayerStatistics != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM player_match_statistics WHERE match_id = $1`, matchID); err != nil {
			return mapError(err)
		}
		for _, item := range *update.PlayerStatistics {
			metadata, err := coverageMetadata(item.Metadata, source)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO player_match_statistics (
					match_id, team_id, person_id, started, minutes_played, goals, assists,
					shots, shots_on_target, passes, passes_completed, key_passes, tackles,
					interceptions, clearances, blocks, duels, duels_won, saves,
					yellow_cards, red_cards, rating, expected_goals, expected_assists, metadata
				) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
				matchID, item.TeamID, item.PersonID, item.Started, item.MinutesPlayed,
				item.Goals, item.Assists, item.Shots, item.ShotsOnTarget, item.Passes, item.PassesCompleted,
				item.KeyPasses, item.Tackles, item.Interceptions, item.Clearances, item.Blocks,
				item.Duels, item.DuelsWon, item.Saves, item.YellowCards, item.RedCards,
				item.Rating, item.ExpectedGoals, item.ExpectedAssists, metadata,
			); err != nil {
				return mapError(err)
			}
		}
	}

	if update.Officials != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM match_officials WHERE match_id = $1`, matchID); err != nil {
			return mapError(err)
		}
		for _, item := range *update.Officials {
			metadata, err := coverageMetadata(item.Metadata, source)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO match_officials (match_id, person_id, role, metadata)
				VALUES ($1, $2, $3, $4)`, matchID, item.PersonID, item.Role, metadata); err != nil {
				return mapError(err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return mapError(err)
	}
	return nil
}

func (store *Store) ReplaceSeasonStandings(
	ctx context.Context,
	seasonID string,
	source string,
	update football.StandingsUpdate,
) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("%w: standings source is required", football.ErrInvalid)
	}

	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin standings replacement: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		SELECT pg_advisory_xact_lock(hashtextextended($1 || ':season-standings:' || $2, 0))`, source, seasonID); err != nil {
		return mapError(err)
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT true FROM seasons WHERE id = $1 FOR UPDATE`, seasonID).Scan(&exists); err != nil {
		return mapError(err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM standings WHERE season_id = $1`, seasonID); err != nil {
		return mapError(err)
	}
	for _, item := range update.Data {
		if _, err := tx.Exec(ctx, `
			INSERT INTO season_teams (season_id, team_id, metadata)
			VALUES ($1, $2, jsonb_build_object('ingestion_source', $3::text))
			ON CONFLICT (season_id, team_id) DO UPDATE
			SET metadata = season_teams.metadata || EXCLUDED.metadata`, seasonID, item.TeamID, source); err != nil {
			return mapError(err)
		}

		var homePlayed, homeWon, homeDrawn, homeLost *int16
		if item.HomeRecord != nil {
			homePlayed, homeWon = &item.HomeRecord.Played, &item.HomeRecord.Won
			homeDrawn, homeLost = &item.HomeRecord.Drawn, &item.HomeRecord.Lost
		}
		var awayPlayed, awayWon, awayDrawn, awayLost *int16
		if item.AwayRecord != nil {
			awayPlayed, awayWon = &item.AwayRecord.Played, &item.AwayRecord.Won
			awayDrawn, awayLost = &item.AwayRecord.Drawn, &item.AwayRecord.Lost
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO standings (
				season_id, team_id, group_name, position, played, won, drawn, lost,
				goals_for, goals_against, points, form, zone, description,
				home_played, home_won, home_drawn, home_lost,
				away_played, away_won, away_drawn, away_lost
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`,
			seasonID, item.TeamID, item.GroupName, item.Position, item.Played, item.Won,
			item.Drawn, item.Lost, item.GoalsFor, item.GoalsAgainst, item.Points,
			item.Form, item.Zone, item.Description,
			homePlayed, homeWon, homeDrawn, homeLost,
			awayPlayed, awayWon, awayDrawn, awayLost,
		); err != nil {
			return mapError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return mapError(err)
	}
	return nil
}

func validateCoverageParticipants(update football.MatchCoverageUpdate, participant func(string) bool) error {
	if update.TeamInfo != nil {
		for _, item := range *update.TeamInfo {
			if !participant(item.TeamID) {
				return fmt.Errorf("%w: team_info contains a team that is not in the match", football.ErrInvalid)
			}
		}
	}
	if update.Lineups != nil {
		for _, item := range *update.Lineups {
			if !participant(item.TeamID) {
				return fmt.Errorf("%w: lineups contains a team that is not in the match", football.ErrInvalid)
			}
		}
	}
	if update.TeamStatistics != nil {
		for _, item := range *update.TeamStatistics {
			if !participant(item.TeamID) {
				return fmt.Errorf("%w: team_statistics contains a team that is not in the match", football.ErrInvalid)
			}
		}
	}
	if update.PlayerStatistics != nil {
		for _, item := range *update.PlayerStatistics {
			if !participant(item.TeamID) {
				return fmt.Errorf("%w: player_statistics contains a team that is not in the match", football.ErrInvalid)
			}
		}
	}
	return nil
}

func coverageMetadata(raw json.RawMessage, source string) (json.RawMessage, error) {
	metadata := make(map[string]any)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return nil, fmt.Errorf("%w: coverage metadata must be a JSON object", football.ErrInvalid)
		}
		if metadata == nil {
			return nil, fmt.Errorf("%w: coverage metadata must be a JSON object", football.ErrInvalid)
		}
	}
	metadata["ingestion_source"] = source
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("encode coverage metadata: %w", err)
	}
	return encoded, nil
}
