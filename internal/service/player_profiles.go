package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (service *Football) ListPlayerMemberships(ctx context.Context, playerID string) (football.PlayerMemberships, error) {
	return service.store.ListPlayerMemberships(ctx, strings.TrimSpace(playerID))
}

func (service *Football) ListPlayerMatches(ctx context.Context, playerID string, filter football.PlayerMatchFilter) ([]football.PlayerMatchHistoryItem, error) {
	if filter.Limit == 0 {
		filter.Limit = 25
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 100", football.ErrInvalid)
	}
	return service.store.ListPlayerMatches(ctx, strings.TrimSpace(playerID), filter)
}

func (service *Football) GetPlayerSeasonStatistics(ctx context.Context, playerID, seasonID, leagueID string) (football.PlayerSeasonStatistics, error) {
	seasonID, leagueID = strings.TrimSpace(seasonID), strings.TrimSpace(leagueID)
	if seasonID == "" && leagueID == "" {
		return football.PlayerSeasonStatistics{}, fmt.Errorf("%w: season_id or league_id is required", football.ErrInvalid)
	}
	return service.store.GetPlayerSeasonStatistics(ctx, strings.TrimSpace(playerID), seasonID, leagueID)
}

func (service *Football) GetPlayerCareer(ctx context.Context, playerID string) (football.PlayerCareer, error) {
	return service.store.GetPlayerCareer(ctx, strings.TrimSpace(playerID))
}

func (service *Football) ListPlayers(ctx context.Context, filter football.PlayerDiscoveryFilter) ([]football.PlayerDiscoveryResult, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Position = strings.ToLower(strings.TrimSpace(filter.Position))
	if len([]rune(filter.Query)) > 100 {
		return nil, fmt.Errorf("%w: q must not exceed 100 characters", football.ErrInvalid)
	}
	if filter.Position != "" && !playerPosition(filter.Position) {
		return nil, fmt.Errorf("%w: position is invalid", football.ErrInvalid)
	}
	if filter.Limit == 0 {
		filter.Limit = 25
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 100", football.ErrInvalid)
	}
	return service.store.ListPlayers(ctx, filter)
}

func (service *Football) ComparePlayers(ctx context.Context, filter football.PlayerComparisonFilter) (football.PlayerComparison, error) {
	if filter.SeasonID == "" && filter.LeagueID == "" {
		return football.PlayerComparison{}, fmt.Errorf("%w: season_id or league_id is required", football.ErrInvalid)
	}
	seen := make(map[string]struct{}, len(filter.PlayerIDs))
	ids := make([]string, 0, len(filter.PlayerIDs))
	for _, id := range filter.PlayerIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, duplicate := seen[id]; !duplicate {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 || len(ids) > 5 {
		return football.PlayerComparison{}, fmt.Errorf("%w: compare requires 2 to 5 distinct player_id values", football.ErrInvalid)
	}
	filter.PlayerIDs = ids
	return service.store.ComparePlayers(ctx, filter)
}

func (service *Football) GetPlayerTraits(ctx context.Context, playerID string, filter football.PlayerAnalyticsFilter) (football.PlayerTraits, error) {
	if filter.SeasonID == "" && filter.LeagueID == "" {
		return football.PlayerTraits{}, fmt.Errorf("%w: season_id or league_id is required", football.ErrInvalid)
	}
	filter.Source = strings.ToLower(strings.TrimSpace(filter.Source))
	return service.store.GetPlayerTraits(ctx, strings.TrimSpace(playerID), filter)
}

func (service *Football) GetPlayerHeatmap(ctx context.Context, playerID string, filter football.PlayerAnalyticsFilter) (football.PlayerHeatmap, error) {
	if filter.MatchID == "" && filter.SeasonID == "" && filter.LeagueID == "" {
		return football.PlayerHeatmap{}, fmt.Errorf("%w: match_id, season_id, or league_id is required", football.ErrInvalid)
	}
	filter.Source = strings.ToLower(strings.TrimSpace(filter.Source))
	return service.store.GetPlayerHeatmap(ctx, strings.TrimSpace(playerID), filter)
}

func (service *Football) ListPlayerShots(ctx context.Context, playerID string, filter football.PlayerAnalyticsFilter) ([]football.PlayerShot, error) {
	if filter.MatchID == "" && filter.SeasonID == "" && filter.LeagueID == "" {
		return nil, fmt.Errorf("%w: match_id, season_id, or league_id is required", football.ErrInvalid)
	}
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 200", football.ErrInvalid)
	}
	filter.Source = strings.ToLower(strings.TrimSpace(filter.Source))
	return service.store.ListPlayerShots(ctx, strings.TrimSpace(playerID), filter)
}

func (service *Football) GetPlayerValuation(ctx context.Context, playerID string) (football.PlayerValuation, error) {
	return service.store.GetPlayerValuation(ctx, strings.TrimSpace(playerID))
}

func (service *Football) UpsertPlayerTraits(ctx context.Context, playerID, source, externalID string, command football.UpsertPlayerTraits) (football.PlayerTraits, error) {
	source, externalID, err := normalizedSourceIdentity(source, externalID)
	if err != nil {
		return football.PlayerTraits{}, err
	}
	command.PositionGroup = strings.ToLower(strings.TrimSpace(command.PositionGroup))
	if command.LeagueID == "" || command.SeasonID == "" || !playerPosition(command.PositionGroup) ||
		command.MinimumMinutes < 0 || command.CohortSize < 2 || command.PlayerMinutes < 0 || command.ObservedAt.IsZero() || len(command.Metrics) == 0 {
		return football.PlayerTraits{}, fmt.Errorf("%w: trait cohort is invalid", football.ErrInvalid)
	}
	seen := make(map[string]struct{}, len(command.Metrics))
	for index := range command.Metrics {
		metric := &command.Metrics[index]
		metric.Key = strings.ToLower(strings.TrimSpace(metric.Key))
		metric.Label = strings.TrimSpace(metric.Label)
		if !metricKey(metric.Key) || metric.Label == "" || metric.Percentile < 0 || metric.Percentile > 100 ||
			!traitCategory(metric.Category) || !traitDirection(metric.Direction) {
			return football.PlayerTraits{}, fmt.Errorf("%w: metrics[%d] is invalid", football.ErrInvalid, index)
		}
		if _, duplicate := seen[metric.Key]; duplicate {
			return football.PlayerTraits{}, fmt.Errorf("%w: metric keys must be unique", football.ErrInvalid)
		}
		seen[metric.Key] = struct{}{}
	}
	if err := normalizeMetadata(&command.Metadata); err != nil {
		return football.PlayerTraits{}, err
	}
	return service.store.UpsertPlayerTraits(ctx, strings.TrimSpace(playerID), source, externalID, command)
}

func (service *Football) UpsertPlayerSpatial(ctx context.Context, matchID, playerID, source, externalID string, command football.UpsertPlayerSpatial) error {
	source, externalID, err := normalizedSourceIdentity(source, externalID)
	if err != nil {
		return err
	}
	if command.TeamID == "" || command.ObservedAt.IsZero() ||
		(command.Orientation != "attacking_left_to_right" && command.Orientation != "attacking_right_to_left") {
		return fmt.Errorf("%w: spatial snapshot is invalid", football.ErrInvalid)
	}
	seenTouches := make(map[int]struct{}, len(command.Touches))
	for index := range command.Touches {
		point := &command.Touches[index]
		if point.Sequence < 1 || !coordinate(point.X) || !coordinate(point.Y) || point.Intensity <= 0 || math.IsInf(point.Intensity, 0) || math.IsNaN(point.Intensity) {
			return fmt.Errorf("%w: touches[%d] is invalid", football.ErrInvalid, index)
		}
		if _, duplicate := seenTouches[point.Sequence]; duplicate {
			return fmt.Errorf("%w: touch sequences must be unique", football.ErrInvalid)
		}
		seenTouches[point.Sequence] = struct{}{}
	}
	seenShots := make(map[int]struct{}, len(command.Shots))
	for index := range command.Shots {
		shot := &command.Shots[index]
		if shot.Sequence < 1 || !coordinate(shot.X) || !coordinate(shot.Y) || shot.ExpectedGoals < 0 || shot.ExpectedGoals > 1 ||
			!shotOutcome(shot.Outcome) || !shotBodyPart(shot.BodyPart) {
			return fmt.Errorf("%w: shots[%d] is invalid", football.ErrInvalid, index)
		}
		if _, duplicate := seenShots[shot.Sequence]; duplicate {
			return fmt.Errorf("%w: shot sequences must be unique", football.ErrInvalid)
		}
		seenShots[shot.Sequence] = struct{}{}
	}
	if command.Orientation == "attacking_right_to_left" {
		for index := range command.Touches {
			command.Touches[index].X = 100 - command.Touches[index].X
			command.Touches[index].Y = 100 - command.Touches[index].Y
		}
		for index := range command.Shots {
			command.Shots[index].X = 100 - command.Shots[index].X
			command.Shots[index].Y = 100 - command.Shots[index].Y
		}
	}
	command.Orientation = "attacking_left_to_right"
	if err := normalizeMetadata(&command.Metadata); err != nil {
		return err
	}
	return service.store.UpsertPlayerSpatial(ctx, strings.TrimSpace(matchID), strings.TrimSpace(playerID), source, externalID, command)
}

func (service *Football) UpsertPlayerValuation(ctx context.Context, playerID, source, externalID string, command football.UpsertPlayerValuation) (football.PlayerValuation, error) {
	source, externalID, err := normalizedSourceIdentity(source, externalID)
	if err != nil {
		return football.PlayerValuation{}, err
	}
	command.Currency = strings.ToUpper(strings.TrimSpace(command.Currency))
	if command.AmountMinor < 0 || len(command.Currency) != 3 || command.ValuationDate.IsZero() || command.ObservedAt.IsZero() {
		return football.PlayerValuation{}, fmt.Errorf("%w: valuation is invalid", football.ErrInvalid)
	}
	for _, character := range command.Currency {
		if character < 'A' || character > 'Z' {
			return football.PlayerValuation{}, fmt.Errorf("%w: currency must be a three-letter uppercase code", football.ErrInvalid)
		}
	}
	if err := normalizeMetadata(&command.Metadata); err != nil {
		return football.PlayerValuation{}, err
	}
	return service.store.UpsertPlayerValuation(ctx, strings.TrimSpace(playerID), source, externalID, command)
}

func playerPosition(value string) bool {
	switch value {
	case "goalkeeper", "defender", "midfielder", "forward":
		return true
	default:
		return false
	}
}

func metricKey(value string) bool {
	if value == "" || len(value) > 64 || value[0] == '_' || value[len(value)-1] == '_' || strings.Contains(value, "__") {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

func traitCategory(value string) bool {
	switch value {
	case "attacking", "possession", "progression", "creation", "defending", "goalkeeping", "discipline", "physical":
		return true
	default:
		return false
	}
}

func traitDirection(value string) bool {
	return value == "higher_is_better" || value == "lower_is_better" || value == "neutral"
}

func coordinate(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 100
}

func shotOutcome(value string) bool {
	switch value {
	case "goal", "saved", "blocked", "off_target", "woodwork", "own_goal", "unknown":
		return true
	default:
		return false
	}
}

func shotBodyPart(value string) bool {
	switch value {
	case "left_foot", "right_foot", "head", "other", "unknown":
		return true
	default:
		return false
	}
}
