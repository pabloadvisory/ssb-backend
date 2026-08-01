package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
)

type predictionRowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (store *Store) GetMatchPrediction(ctx context.Context, matchID string) (football.MatchPrediction, error) {
	return getMatchPrediction(ctx, store.pool, matchID, nil)
}

func (store *Store) GetInstallationPrediction(
	ctx context.Context,
	installationID string,
	secretHash []byte,
	matchID string,
) (football.MatchPrediction, error) {
	if err := authenticatePredictionInstallation(ctx, store.pool, installationID, secretHash); err != nil {
		return football.MatchPrediction{}, err
	}
	return getMatchPrediction(ctx, store.pool, matchID, &installationID)
}

func (store *Store) SetInstallationPrediction(
	ctx context.Context,
	installationID string,
	secretHash []byte,
	matchID string,
	selection football.PredictionSelection,
) (football.MatchPrediction, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return football.MatchPrediction{}, fmt.Errorf("begin prediction vote: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := authenticatePredictionInstallation(ctx, tx, installationID, secretHash); err != nil {
		return football.MatchPrediction{}, err
	}

	var open bool
	err = tx.QueryRow(ctx, `
		SELECT status IN ('scheduled', 'postponed') AND kickoff_at > now()
		FROM matches
		WHERE id = $1
		FOR UPDATE`, matchID).Scan(&open)
	if err != nil {
		return football.MatchPrediction{}, mapError(err)
	}
	if !open {
		return football.MatchPrediction{}, fmt.Errorf("%w: prediction voting is closed", football.ErrConflict)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO match_predictions (match_id, installation_id, selection)
		VALUES ($1, $2, $3)
		ON CONFLICT (match_id, installation_id) DO UPDATE
		SET selection = EXCLUDED.selection`, matchID, installationID, selection); err != nil {
		return football.MatchPrediction{}, mapError(err)
	}

	prediction, err := getMatchPrediction(ctx, tx, matchID, &installationID)
	if err != nil {
		return football.MatchPrediction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return football.MatchPrediction{}, mapError(err)
	}
	return prediction, nil
}

func authenticatePredictionInstallation(
	ctx context.Context,
	querier predictionRowQuerier,
	installationID string,
	secretHash []byte,
) error {
	var authenticated bool
	err := querier.QueryRow(ctx, `
		SELECT true
		FROM app_installations
		WHERE id = $1 AND secret_hash = $2 AND disabled_at IS NULL`, installationID, secretHash).Scan(&authenticated)
	if errors.Is(err, pgx.ErrNoRows) {
		return notification.ErrUnauthorized
	}
	if err != nil {
		return mapError(err)
	}
	if !authenticated {
		return notification.ErrUnauthorized
	}
	return nil
}

func getMatchPrediction(
	ctx context.Context,
	querier predictionRowQuerier,
	matchID string,
	installationID *string,
) (football.MatchPrediction, error) {
	var prediction football.MatchPrediction
	var homeTeamID, awayTeamID string
	var homeVotes, drawVotes, awayVotes int64
	var mySelection *string
	err := querier.QueryRow(ctx, `
		SELECT match.id, match.kickoff_at,
			match.status IN ('scheduled', 'postponed') AND match.kickoff_at > now(),
			match.home_team_id, match.away_team_id,
			count(vote.installation_id) FILTER (WHERE vote.selection = 'home'),
			count(vote.installation_id) FILTER (WHERE vote.selection = 'draw'),
			count(vote.installation_id) FILTER (WHERE vote.selection = 'away'),
			mine.selection
		FROM matches match
		LEFT JOIN match_predictions vote ON vote.match_id = match.id
		LEFT JOIN match_predictions mine
			ON mine.match_id = match.id AND mine.installation_id = $2::uuid
		WHERE match.id = $1
		GROUP BY match.id, mine.selection`, matchID, installationID).Scan(
		&prediction.MatchID, &prediction.ClosesAt, &prediction.IsOpen,
		&homeTeamID, &awayTeamID, &homeVotes, &drawVotes, &awayVotes, &mySelection,
	)
	if err != nil {
		return football.MatchPrediction{}, mapError(err)
	}

	votes := [3]int64{homeVotes, drawVotes, awayVotes}
	percentages := predictionPercentages(votes)
	prediction.TotalVotes = homeVotes + drawVotes + awayVotes
	prediction.Options = []football.PredictionOption{
		{Selection: football.PredictionHome, TeamID: &homeTeamID, Votes: homeVotes, Percent: percentages[0]},
		{Selection: football.PredictionDraw, Votes: drawVotes, Percent: percentages[1]},
		{Selection: football.PredictionAway, TeamID: &awayTeamID, Votes: awayVotes, Percent: percentages[2]},
	}
	if mySelection != nil {
		selection := football.PredictionSelection(*mySelection)
		prediction.MySelection = &selection
	}
	return prediction, nil
}

// predictionPercentages uses the largest-remainder method at two decimal places.
// Ties resolve in home/draw/away order, making the three values deterministic and
// ensuring that a non-empty poll totals exactly 100.00 percent.
func predictionPercentages(votes [3]int64) [3]float64 {
	total := votes[0] + votes[1] + votes[2]
	if total == 0 {
		return [3]float64{}
	}

	var basisPoints [3]int64
	var remainders [3]int64
	var allocated int64
	for index, count := range votes {
		numerator := count * 10_000
		basisPoints[index] = numerator / total
		remainders[index] = numerator % total
		allocated += basisPoints[index]
	}
	for missing := int64(10_000) - allocated; missing > 0; missing-- {
		best := 0
		for index := 1; index < len(remainders); index++ {
			if remainders[index] > remainders[best] {
				best = index
			}
		}
		basisPoints[best]++
		remainders[best] = -1
	}

	return [3]float64{
		float64(basisPoints[0]) / 100,
		float64(basisPoints[1]) / 100,
		float64(basisPoints[2]) / 100,
	}
}
