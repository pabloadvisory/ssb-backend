package football

import (
	"context"
	"time"
)

type LeagueFilter struct {
	CountryCode string
	Limit       int
	AfterName   string
	AfterID     string
}

type MatchFilter struct {
	LeagueID     string
	SeasonID     string
	TeamID       string
	Status       MatchStatus
	From         *time.Time
	To           *time.Time
	Limit        int
	AfterKickoff *time.Time
	AfterMatchID string
}

type EventFilter struct {
	AfterSequence int
	Limit         int
}

type Store interface {
	ListLeagues(context.Context, LeagueFilter) ([]League, error)
	GetLeague(context.Context, string) (League, error)
	ListLeagueSeasons(context.Context, string) (LeagueSeasons, error)
	GetTeam(context.Context, string) (Team, error)
	GetPlayer(context.Context, string) (Player, error)
	GetCoach(context.Context, string) (Coach, error)
	GetVenue(context.Context, string) (Venue, error)
	ListMatches(context.Context, MatchFilter) ([]Match, error)
	GetMatch(context.Context, string) (Match, error)
	ListMatchEvents(context.Context, string, EventFilter) ([]MatchEvent, error)
	GetMatchLineups(context.Context, string) (MatchLineups, error)
	GetMatchStatistics(context.Context, string) (MatchStatistics, error)
	ListSeasonStandings(context.Context, string) (SeasonStandings, error)
	ListMatchOfficials(context.Context, string) (MatchOfficials, error)
	Search(context.Context, SearchFilter) ([]SearchResult, error)
	ListHeadToHeadMatches(context.Context, HeadToHeadFilter) ([]Match, error)
	ListMatchOdds(context.Context, string, string) (MatchOdds, error)
	UpsertMatchOdds(context.Context, string, string, string, UpsertOddsSnapshot) (MatchOddsSnapshot, error)
	ListMatchBroadcasts(context.Context, string, string) (MatchBroadcasts, error)
	UpsertMatchBroadcast(context.Context, string, string, string, UpsertMatchBroadcast) (MatchBroadcast, error)
	GetMatchWeather(context.Context, string) (MatchWeather, error)
	UpsertMatchWeather(context.Context, string, string, string, UpsertWeatherSnapshot) (WeatherSnapshot, error)
	GetMatchPrediction(context.Context, string) (MatchPrediction, error)
	GetInstallationPrediction(context.Context, string, []byte, string) (MatchPrediction, error)
	SetInstallationPrediction(context.Context, string, []byte, string, PredictionSelection) (MatchPrediction, error)
	ReplaceMatchCoverage(context.Context, string, string, MatchCoverageUpdate) error
	ReplaceSeasonStandings(context.Context, string, string, StandingsUpdate) error
	UpsertMatchSnapshot(context.Context, string, string, MatchSnapshot) (Match, error)
}
