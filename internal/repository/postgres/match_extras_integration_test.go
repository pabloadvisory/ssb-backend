package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/demo"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	postgresrepo "github.com/pabloadvisory/ssb-backend/internal/repository/postgres"
	"github.com/pabloadvisory/ssb-backend/migrations"
)

func TestMatchExtrasRepositoryIntegration(t *testing.T) {
	pool := newTestDatabase(t)
	ctx := context.Background()
	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := demo.Seed(ctx, pool); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}

	repository := postgresrepo.New(pool)
	const matchID = "40000000-0000-0000-0000-000000000001"
	const missingMatchID = "40000000-0000-0000-0000-999999999999"

	t.Run("odds replace children and return latest bookmaker snapshot", func(t *testing.T) {
		if _, err := repository.ListMatchOdds(ctx, missingMatchID, ""); !errors.Is(err, football.ErrNotFound) {
			t.Fatalf("unknown match: %v", err)
		}

		observedAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
		created, err := repository.UpsertMatchOdds(ctx, matchID, "integration", "odds-1", football.UpsertOddsSnapshot{
			BookmakerSlug: "island-book", BookmakerName: "Island Book", ObservedAt: observedAt,
			Markets: []football.OddsMarket{{
				Key: "winner", Name: "Winner", Status: "open",
				Selections: []football.OddsSelection{{Key: "old", Name: "Old", DecimalOdds: 2}},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		updated, err := repository.UpsertMatchOdds(ctx, matchID, "integration", "odds-1", football.UpsertOddsSnapshot{
			BookmakerSlug: "island-book", BookmakerName: "Island Book", ObservedAt: observedAt,
			Markets: []football.OddsMarket{{
				Key: "winner", Name: "Winner", Status: "open",
				Selections: []football.OddsSelection{
					{Key: "home", Name: "Home", DecimalOdds: 2.1},
					{Key: "away", Name: "Away", DecimalOdds: 3.2},
				},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated.ID != created.ID || len(updated.Markets) != 1 || len(updated.Markets[0].Selections) != 2 {
			t.Fatalf("snapshot replacement was not idempotent: created=%+v updated=%+v", created, updated)
		}
		if updated.Markets[0].Selections[0].Key != "away" || updated.Markets[0].Selections[1].Key != "home" {
			t.Fatalf("selections are not deterministic: %+v", updated.Markets[0].Selections)
		}

		latest, err := repository.UpsertMatchOdds(ctx, matchID, "integration", "odds-2", football.UpsertOddsSnapshot{
			BookmakerSlug: "island-book", BookmakerName: "Island Book",
			ObservedAt: observedAt.Add(time.Minute), Markets: []football.OddsMarket{},
		})
		if err != nil {
			t.Fatal(err)
		}
		listed, err := repository.ListMatchOdds(ctx, matchID, "island-book")
		if err != nil {
			t.Fatal(err)
		}
		if listed.Data == nil || len(listed.Data) != 1 || listed.Data[0].ID != latest.ID || listed.Data[0].Markets == nil {
			t.Fatalf("expected stable latest snapshot response, got %+v", listed)
		}
	})

	t.Run("broadcast country filtering excludes unknown availability", func(t *testing.T) {
		if _, err := repository.ListMatchBroadcasts(ctx, missingMatchID, "SC"); !errors.Is(err, football.ErrNotFound) {
			t.Fatalf("unknown match: %v", err)
		}
		observedAt := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
		commands := []struct {
			externalID string
			name       string
			scope      football.BroadcastScope
			regions    []string
		}{
			{externalID: "global", name: "Global Network", scope: football.BroadcastGlobal},
			{externalID: "seychelles", name: "Seychelles Network", scope: football.BroadcastTerritorial, regions: []string{"SC"}},
			{externalID: "unknown", name: "Unknown Network", scope: football.BroadcastUnknown},
		}
		for _, item := range commands {
			_, err := repository.UpsertMatchBroadcast(ctx, matchID, "integration", item.externalID, football.UpsertMatchBroadcast{
				NetworkName: item.name, Kind: football.BroadcastStream, AvailabilityScope: item.scope,
				Regions: item.regions, LanguageTags: []string{"en"}, Status: football.BroadcastScheduled,
				ObservedAt: observedAt,
			})
			if err != nil {
				t.Fatal(err)
			}
		}

		seychelles, err := repository.ListMatchBroadcasts(ctx, matchID, "sc")
		if err != nil {
			t.Fatal(err)
		}
		if len(seychelles.Data) != 2 {
			t.Fatalf("expected global and Seychelles broadcasts, got %+v", seychelles.Data)
		}
		elsewhere, err := repository.ListMatchBroadcasts(ctx, matchID, "GB")
		if err != nil {
			t.Fatal(err)
		}
		if len(elsewhere.Data) != 1 || elsewhere.Data[0].AvailabilityScope != football.BroadcastGlobal {
			t.Fatalf("unknown availability must not be treated as worldwide: %+v", elsewhere.Data)
		}
	})

	t.Run("weather returns nearest forecast and latest observation", func(t *testing.T) {
		if _, err := repository.GetMatchWeather(ctx, missingMatchID); !errors.Is(err, football.ErrNotFound) {
			t.Fatalf("unknown match: %v", err)
		}
		var kickoffAt time.Time
		if err := pool.QueryRow(ctx, `SELECT kickoff_at FROM matches WHERE id = $1`, matchID).Scan(&kickoffAt); err != nil {
			t.Fatal(err)
		}
		insert := func(externalID string, command football.UpsertWeatherSnapshot) football.WeatherSnapshot {
			t.Helper()
			value, err := repository.UpsertMatchWeather(ctx, matchID, "integration", externalID, command)
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		insert("forecast-far", football.UpsertWeatherSnapshot{
			Kind: football.WeatherForecast, ValidAt: kickoffAt.Add(-3 * time.Hour), IssuedAt: kickoffAt.Add(-6 * time.Hour),
		})
		nearest := insert("forecast-near", football.UpsertWeatherSnapshot{
			Kind: football.WeatherForecast, ValidAt: kickoffAt.Add(15 * time.Minute), IssuedAt: kickoffAt.Add(-time.Hour),
		})
		insert("observation-old", football.UpsertWeatherSnapshot{
			Kind: football.WeatherObserved, ValidAt: kickoffAt, IssuedAt: kickoffAt,
		})
		latestObservation := insert("observation-new", football.UpsertWeatherSnapshot{
			Kind: football.WeatherObserved, ValidAt: kickoffAt.Add(time.Hour), IssuedAt: kickoffAt.Add(time.Hour),
		})

		weather, err := repository.GetMatchWeather(ctx, matchID)
		if err != nil {
			t.Fatal(err)
		}
		if weather.Forecast == nil || weather.Forecast.ID != nearest.ID {
			t.Fatalf("unexpected forecast: %+v", weather.Forecast)
		}
		if weather.Observation == nil || weather.Observation.ID != latestObservation.ID {
			t.Fatalf("unexpected observation: %+v", weather.Observation)
		}
	})
}
