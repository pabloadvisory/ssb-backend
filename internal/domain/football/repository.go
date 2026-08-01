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
	GetTeam(context.Context, string) (Team, error)
	GetPlayer(context.Context, string) (Player, error)
	GetCoach(context.Context, string) (Coach, error)
	ListMatches(context.Context, MatchFilter) ([]Match, error)
	GetMatch(context.Context, string) (Match, error)
	ListMatchEvents(context.Context, string, EventFilter) ([]MatchEvent, error)
	UpsertMatchSnapshot(context.Context, string, string, MatchSnapshot) (Match, error)
}
