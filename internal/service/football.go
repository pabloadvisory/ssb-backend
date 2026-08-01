package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

type Football struct {
	store football.Store
}

func NewFootball(store football.Store) *Football {
	return &Football{store: store}
}

func (service *Football) ListLeagues(ctx context.Context, filter football.LeagueFilter) ([]football.League, error) {
	return service.store.ListLeagues(ctx, filter)
}

func (service *Football) GetLeague(ctx context.Context, id string) (football.League, error) {
	return service.store.GetLeague(ctx, id)
}

func (service *Football) GetTeam(ctx context.Context, id string) (football.Team, error) {
	return service.store.GetTeam(ctx, id)
}

func (service *Football) GetPlayer(ctx context.Context, id string) (football.Player, error) {
	return service.store.GetPlayer(ctx, id)
}

func (service *Football) GetCoach(ctx context.Context, id string) (football.Coach, error) {
	return service.store.GetCoach(ctx, id)
}

func (service *Football) ListMatches(ctx context.Context, filter football.MatchFilter) ([]football.Match, error) {
	if filter.Status != "" && !filter.Status.Valid() {
		return nil, fmt.Errorf("%w: unknown match status", football.ErrInvalid)
	}
	if filter.From != nil && filter.To != nil && filter.To.Before(*filter.From) {
		return nil, fmt.Errorf("%w: date_to must not be before date_from", football.ErrInvalid)
	}
	return service.store.ListMatches(ctx, filter)
}

func (service *Football) GetMatch(ctx context.Context, id string) (football.Match, error) {
	return service.store.GetMatch(ctx, id)
}

func (service *Football) ListMatchEvents(ctx context.Context, id string, filter football.EventFilter) ([]football.MatchEvent, error) {
	return service.store.ListMatchEvents(ctx, id, filter)
}

func (service *Football) UpsertMatchSnapshot(ctx context.Context, provider, externalID string, snapshot football.MatchSnapshot) (football.Match, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	externalID = strings.TrimSpace(externalID)
	if provider == "" || len(provider) > 64 || externalID == "" || len(externalID) > 256 {
		return football.Match{}, fmt.Errorf("%w: provider and external_id are required", football.ErrInvalid)
	}
	for _, character := range provider {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' && character != '_' {
			return football.Match{}, fmt.Errorf("%w: provider contains unsupported characters", football.ErrInvalid)
		}
	}
	if len(snapshot.Metadata) == 0 {
		snapshot.Metadata = json.RawMessage(`{}`)
	}
	if !json.Valid(snapshot.Metadata) {
		return football.Match{}, fmt.Errorf("%w: metadata must be valid JSON", football.ErrInvalid)
	}
	for index := range snapshot.Events {
		if len(snapshot.Events[index].Metadata) == 0 {
			snapshot.Events[index].Metadata = json.RawMessage(`{}`)
		}
		if !json.Valid(snapshot.Events[index].Metadata) {
			return football.Match{}, fmt.Errorf("%w: events[%d].metadata must be valid JSON", football.ErrInvalid, index)
		}
	}
	sort.Slice(snapshot.Events, func(i, j int) bool { return snapshot.Events[i].Sequence < snapshot.Events[j].Sequence })
	if err := snapshot.Validate(); err != nil {
		return football.Match{}, err
	}
	return service.store.UpsertMatchSnapshot(ctx, provider, externalID, snapshot)
}
