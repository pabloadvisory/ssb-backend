package httpapi

import (
	"context"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (store *fakeStore) GetVenue(context.Context, string) (football.Venue, error) {
	if store.venue.ID == "" {
		return football.Venue{}, football.ErrNotFound
	}
	return store.venue, nil
}

func (store *fakeStore) GetMatchLineups(context.Context, string) (football.MatchLineups, error) {
	if store.lineups.MatchID == "" {
		return football.MatchLineups{}, football.ErrNotFound
	}
	return store.lineups, nil
}

func (store *fakeStore) GetMatchStatistics(context.Context, string) (football.MatchStatistics, error) {
	if store.statistics.MatchID == "" {
		return football.MatchStatistics{}, football.ErrNotFound
	}
	return store.statistics, nil
}

func (store *fakeStore) ListSeasonStandings(context.Context, string) (football.SeasonStandings, error) {
	if store.standings.SeasonID == "" {
		return football.SeasonStandings{}, football.ErrNotFound
	}
	return store.standings, nil
}

func (store *fakeStore) ListMatchOfficials(context.Context, string) (football.MatchOfficials, error) {
	if store.officials.MatchID == "" {
		return football.MatchOfficials{}, football.ErrNotFound
	}
	return store.officials, nil
}

func (store *fakeStore) Search(context.Context, football.SearchFilter) ([]football.SearchResult, error) {
	return store.searchResults, nil
}

func (store *fakeStore) ListHeadToHeadMatches(context.Context, football.HeadToHeadFilter) ([]football.Match, error) {
	return store.h2hMatches, nil
}

func (store *fakeStore) ListMatchOdds(context.Context, string, string) (football.MatchOdds, error) {
	if store.odds.MatchID == "" {
		return football.MatchOdds{}, football.ErrNotFound
	}
	return store.odds, nil
}

func (store *fakeStore) UpsertMatchOdds(context.Context, string, string, string, football.UpsertOddsSnapshot) (football.MatchOddsSnapshot, error) {
	if len(store.odds.Data) == 0 {
		return football.MatchOddsSnapshot{}, nil
	}
	return store.odds.Data[0], nil
}

func (store *fakeStore) ListMatchBroadcasts(context.Context, string, string) (football.MatchBroadcasts, error) {
	if store.broadcasts.MatchID == "" {
		return football.MatchBroadcasts{}, football.ErrNotFound
	}
	return store.broadcasts, nil
}

func (store *fakeStore) UpsertMatchBroadcast(context.Context, string, string, string, football.UpsertMatchBroadcast) (football.MatchBroadcast, error) {
	if len(store.broadcasts.Data) == 0 {
		return football.MatchBroadcast{}, nil
	}
	return store.broadcasts.Data[0], nil
}

func (store *fakeStore) GetMatchWeather(context.Context, string) (football.MatchWeather, error) {
	if store.weather.MatchID == "" {
		return football.MatchWeather{}, football.ErrNotFound
	}
	return store.weather, nil
}

func (store *fakeStore) UpsertMatchWeather(context.Context, string, string, string, football.UpsertWeatherSnapshot) (football.WeatherSnapshot, error) {
	if store.weather.Forecast != nil {
		return *store.weather.Forecast, nil
	}
	if store.weather.Observation != nil {
		return *store.weather.Observation, nil
	}
	return football.WeatherSnapshot{}, nil
}

func (store *fakeStore) GetMatchPrediction(context.Context, string) (football.MatchPrediction, error) {
	if store.prediction.MatchID == "" {
		return football.MatchPrediction{}, football.ErrNotFound
	}
	return store.prediction, nil
}

func (store *fakeStore) GetInstallationPrediction(context.Context, string, []byte, string) (football.MatchPrediction, error) {
	return store.GetMatchPrediction(context.Background(), "")
}

func (store *fakeStore) SetInstallationPrediction(context.Context, string, []byte, string, football.PredictionSelection) (football.MatchPrediction, error) {
	return store.GetMatchPrediction(context.Background(), "")
}

func (*fakeStore) ReplaceMatchCoverage(context.Context, string, string, football.MatchCoverageUpdate) error {
	return nil
}

func (*fakeStore) ReplaceSeasonStandings(context.Context, string, string, football.StandingsUpdate) error {
	return nil
}
