package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func (service *Football) GetVenue(ctx context.Context, id string) (football.Venue, error) {
	return service.store.GetVenue(ctx, id)
}

func (service *Football) GetMatchLineups(ctx context.Context, matchID string) (football.MatchLineups, error) {
	return service.store.GetMatchLineups(ctx, matchID)
}

func (service *Football) GetMatchStatistics(ctx context.Context, matchID string) (football.MatchStatistics, error) {
	return service.store.GetMatchStatistics(ctx, matchID)
}

func (service *Football) ListSeasonStandings(ctx context.Context, seasonID string) (football.SeasonStandings, error) {
	return service.store.ListSeasonStandings(ctx, seasonID)
}

func (service *Football) ListMatchOfficials(ctx context.Context, matchID string) (football.MatchOfficials, error) {
	return service.store.ListMatchOfficials(ctx, matchID)
}

func (service *Football) Search(ctx context.Context, filter football.SearchFilter) ([]football.SearchResult, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	if len([]rune(filter.Query)) < 2 || len([]rune(filter.Query)) > 100 {
		return nil, fmt.Errorf("%w: q must contain between 2 and 100 characters", football.ErrInvalid)
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	if filter.Limit < 1 || filter.Limit > 50 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 50", football.ErrInvalid)
	}
	if len(filter.Types) == 0 {
		filter.Types = []football.SearchEntityType{
			football.SearchLeague, football.SearchTeam, football.SearchPlayer,
			football.SearchCoach, football.SearchFixture,
		}
	}
	seen := make(map[football.SearchEntityType]struct{}, len(filter.Types))
	for _, entityType := range filter.Types {
		switch entityType {
		case football.SearchLeague, football.SearchTeam, football.SearchPlayer, football.SearchCoach, football.SearchFixture:
		default:
			return nil, fmt.Errorf("%w: search type %q is invalid", football.ErrInvalid, entityType)
		}
		if _, duplicate := seen[entityType]; duplicate {
			continue
		}
		seen[entityType] = struct{}{}
	}
	filter.Types = filter.Types[:0]
	for _, entityType := range []football.SearchEntityType{
		football.SearchLeague, football.SearchTeam, football.SearchPlayer,
		football.SearchCoach, football.SearchFixture,
	} {
		if _, ok := seen[entityType]; ok {
			filter.Types = append(filter.Types, entityType)
		}
	}
	return service.store.Search(ctx, filter)
}

func (service *Football) HeadToHead(ctx context.Context, filter football.HeadToHeadFilter) (football.HeadToHead, error) {
	filter.TeamAID = strings.TrimSpace(filter.TeamAID)
	filter.TeamBID = strings.TrimSpace(filter.TeamBID)
	if filter.TeamAID == "" || filter.TeamBID == "" {
		return football.HeadToHead{}, fmt.Errorf("%w: team_a and team_b are required", football.ErrInvalid)
	}
	if filter.TeamAID == filter.TeamBID {
		return football.HeadToHead{}, fmt.Errorf("%w: team_a and team_b must differ", football.ErrInvalid)
	}
	if filter.Limit == 0 {
		filter.Limit = 10
	}
	if filter.Limit < 1 || filter.Limit > 50 {
		return football.HeadToHead{}, fmt.Errorf("%w: limit must be between 1 and 50", football.ErrInvalid)
	}
	meetings, err := service.store.ListHeadToHeadMatches(ctx, filter)
	if err != nil {
		return football.HeadToHead{}, err
	}
	result := football.HeadToHead{
		TeamAID:  filter.TeamAID,
		TeamBID:  filter.TeamBID,
		Meetings: meetings,
		Summary:  football.HeadToHeadSummary{MatchesConsidered: len(meetings)},
	}
	for _, match := range meetings {
		winnerID := match.WinnerTeamID
		if winnerID == nil && match.HomeScore != nil && match.AwayScore != nil {
			switch {
			case *match.HomeScore > *match.AwayScore:
				winnerID = &match.HomeTeamID
			case *match.AwayScore > *match.HomeScore:
				winnerID = &match.AwayTeamID
			}
		}
		switch {
		case winnerID == nil:
			result.Summary.Draws++
		case *winnerID == filter.TeamAID:
			result.Summary.TeamAWins++
		case *winnerID == filter.TeamBID:
			result.Summary.TeamBWins++
		}
	}
	return result, nil
}
