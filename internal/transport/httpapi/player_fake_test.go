package httpapi

import (
	"context"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (*fakeStore) ListPlayerMemberships(context.Context, string) (football.PlayerMemberships, error) {
	return football.PlayerMemberships{Data: []football.PlayerMembership{}}, nil
}

func (*fakeStore) ListPlayerMatches(context.Context, string, football.PlayerMatchFilter) ([]football.PlayerMatchHistoryItem, error) {
	return []football.PlayerMatchHistoryItem{}, nil
}

func (*fakeStore) GetPlayerSeasonStatistics(context.Context, string, string, string) (football.PlayerSeasonStatistics, error) {
	return football.PlayerSeasonStatistics{}, football.ErrNotFound
}

func (*fakeStore) GetPlayerCareer(context.Context, string) (football.PlayerCareer, error) {
	return football.PlayerCareer{Spells: []football.PlayerCareerSpell{}}, nil
}

func (*fakeStore) ListPlayers(context.Context, football.PlayerDiscoveryFilter) ([]football.PlayerDiscoveryResult, error) {
	return []football.PlayerDiscoveryResult{}, nil
}

func (*fakeStore) ComparePlayers(context.Context, football.PlayerComparisonFilter) (football.PlayerComparison, error) {
	return football.PlayerComparison{}, football.ErrNotFound
}

func (*fakeStore) GetPlayerTraits(context.Context, string, football.PlayerAnalyticsFilter) (football.PlayerTraits, error) {
	return football.PlayerTraits{}, football.ErrNotFound
}

func (*fakeStore) GetPlayerHeatmap(context.Context, string, football.PlayerAnalyticsFilter) (football.PlayerHeatmap, error) {
	return football.PlayerHeatmap{Data: []football.HeatmapPoint{}}, nil
}

func (*fakeStore) ListPlayerShots(context.Context, string, football.PlayerAnalyticsFilter) ([]football.PlayerShot, error) {
	return []football.PlayerShot{}, nil
}

func (*fakeStore) GetPlayerValuation(context.Context, string) (football.PlayerValuation, error) {
	return football.PlayerValuation{}, football.ErrNotFound
}

func (*fakeStore) UpsertPlayerTraits(context.Context, string, string, string, football.UpsertPlayerTraits) (football.PlayerTraits, error) {
	return football.PlayerTraits{}, nil
}

func (*fakeStore) UpsertPlayerSpatial(context.Context, string, string, string, string, football.UpsertPlayerSpatial) error {
	return nil
}

func (*fakeStore) UpsertPlayerValuation(context.Context, string, string, string, football.UpsertPlayerValuation) (football.PlayerValuation, error) {
	return football.PlayerValuation{}, nil
}
