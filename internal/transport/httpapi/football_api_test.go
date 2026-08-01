package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func TestFootballCoverageReadRoutes(t *testing.T) {
	t.Parallel()
	matchID := "40000000-0000-0000-0000-000000000001"
	seasonID := "10000000-0000-0000-0000-000000000002"
	venueID := "10000000-0000-0000-0000-000000000003"
	store := &fakeStore{
		venue: football.Venue{ID: venueID, Name: "National Stadium"},
		lineups: football.MatchLineups{
			MatchID: matchID,
			Home:    football.TeamLineup{Team: football.TeamSummary{ID: "home", Name: "Home"}, Starters: []football.LineupPlayer{}, Substitutes: []football.LineupPlayer{}},
			Away:    football.TeamLineup{Team: football.TeamSummary{ID: "away", Name: "Away"}, Starters: []football.LineupPlayer{}, Substitutes: []football.LineupPlayer{}},
		},
		statistics: football.MatchStatistics{
			MatchID: matchID,
			Home:    football.TeamMatchStatistics{Team: football.TeamSummary{ID: "home", Name: "Home"}, Players: []football.PlayerStatistics{}},
			Away:    football.TeamMatchStatistics{Team: football.TeamSummary{ID: "away", Name: "Away"}, Players: []football.PlayerStatistics{}},
		},
		standings: football.SeasonStandings{SeasonID: seasonID, Data: []football.StandingEntry{}},
		officials: football.MatchOfficials{MatchID: matchID, Data: []football.MatchOfficialDetail{}},
	}
	handler := testHandler(store, "secret")

	for _, path := range []string{
		"/v1/venues/" + venueID,
		"/v1/matches/" + matchID + "/lineups",
		"/v1/matches/" + matchID + "/statistics",
		"/v1/matches/" + matchID + "/officials",
		"/v1/seasons/" + seasonID + "/standings",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s returned %d: %s", path, response.Code, response.Body.String())
		}
		if response.Header().Get("ETag") == "" {
			t.Fatalf("GET %s did not return an ETag", path)
		}
	}
}

func TestLeagueSeasonDiscovery(t *testing.T) {
	t.Parallel()
	leagueID := "10000000-0000-0000-0000-000000000001"
	seasonID := "10000000-0000-0000-0000-000000000002"
	store := &fakeStore{
		leagues: []football.League{{ID: leagueID, Name: "Seychelles Premier League", CurrentSeasonID: &seasonID}},
		seasons: football.LeagueSeasons{
			LeagueID: leagueID,
			Data:     []football.Season{{ID: seasonID, LeagueID: leagueID, Name: "2026", IsCurrent: true}},
		},
	}
	handler := testHandler(store, "secret")

	leagueResponse := httptest.NewRecorder()
	handler.ServeHTTP(leagueResponse, httptest.NewRequest(http.MethodGet, "/v1/leagues/"+leagueID, nil))
	if leagueResponse.Code != http.StatusOK || !strings.Contains(leagueResponse.Body.String(), `"current_season_id":"`+seasonID+`"`) {
		t.Fatalf("league current season missing: status=%d body=%s", leagueResponse.Code, leagueResponse.Body.String())
	}

	seasonsResponse := httptest.NewRecorder()
	handler.ServeHTTP(seasonsResponse, httptest.NewRequest(http.MethodGet, "/v1/leagues/"+leagueID+"/seasons", nil))
	if seasonsResponse.Code != http.StatusOK || !strings.Contains(seasonsResponse.Body.String(), `"is_current":true`) {
		t.Fatalf("league seasons missing: status=%d body=%s", seasonsResponse.Code, seasonsResponse.Body.String())
	}
}

func TestFootballLineupsUseEmptyArrays(t *testing.T) {
	t.Parallel()
	store := &fakeStore{lineups: football.MatchLineups{
		MatchID: "match-1",
		Home:    football.TeamLineup{Starters: []football.LineupPlayer{}, Substitutes: []football.LineupPlayer{}},
		Away:    football.TeamLineup{Starters: []football.LineupPlayer{}, Substitutes: []football.LineupPlayer{}},
	}}
	response := httptest.NewRecorder()
	testHandler(store, "secret").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/matches/match-1/lineups", nil))
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), `"starters":null`) || !strings.Contains(response.Body.String(), `"starters":[]`) {
		t.Fatalf("expected stable empty arrays, status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestFootballSearchAndHeadToHeadContracts(t *testing.T) {
	t.Parallel()
	teamA, teamB := "20000000-0000-0000-0000-000000000001", "20000000-0000-0000-0000-000000000002"
	homeScore, awayScore := int16(2), int16(1)
	drawScore := int16(1)
	store := &fakeStore{
		searchResults: []football.SearchResult{
			{EntityType: football.SearchLeague, ID: "league-1", Name: "Seychelles Premier League"},
			{EntityType: football.SearchFixture, ID: "match-1", Name: "Victoria United v La Passe"},
		},
		h2hMatches: []football.Match{
			{ID: "match-1", HomeTeamID: teamA, AwayTeamID: teamB, HomeScore: &homeScore, AwayScore: &awayScore},
			{ID: "match-2", HomeTeamID: teamB, AwayTeamID: teamA, HomeScore: &drawScore, AwayScore: &drawScore},
		},
	}
	handler := testHandler(store, "secret")

	searchResponse := httptest.NewRecorder()
	handler.ServeHTTP(searchResponse, httptest.NewRequest(http.MethodGet, "/v1/search?q=premier&type=league&type=fixture", nil))
	if searchResponse.Code != http.StatusOK ||
		!strings.Contains(searchResponse.Body.String(), `"entity_type":"league"`) ||
		!strings.Contains(searchResponse.Body.String(), `"entity_type":"fixture"`) {
		t.Fatalf("unexpected search response: %d %s", searchResponse.Code, searchResponse.Body.String())
	}

	h2hResponse := httptest.NewRecorder()
	handler.ServeHTTP(h2hResponse, httptest.NewRequest(http.MethodGet, "/v1/matches/head-to-head?team_a="+teamA+"&team_b="+teamB, nil))
	var result football.HeadToHead
	if err := json.NewDecoder(h2hResponse.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if h2hResponse.Code != http.StatusOK || result.Summary.TeamAWins != 1 || result.Summary.Draws != 1 || result.Summary.MatchesConsidered != 2 {
		t.Fatalf("unexpected H2H response: status=%d result=%+v", h2hResponse.Code, result)
	}
}

func TestFootballSearchValidationRejectsBadQueries(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"/v1/search?q=x", "/v1/search?q=victoria&type=unknown"} {
		response := httptest.NewRecorder()
		testHandler(&fakeStore{}, "secret").ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422 for %s, got %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestFootballExtrasAndPredictionRoutes(t *testing.T) {
	t.Parallel()
	matchID := "40000000-0000-0000-0000-000000000002"
	now := time.Now().UTC()
	selection := football.PredictionHome
	store := &fakeStore{
		odds:       football.MatchOdds{MatchID: matchID, Data: []football.MatchOddsSnapshot{}},
		broadcasts: football.MatchBroadcasts{MatchID: matchID, Data: []football.MatchBroadcast{}},
		weather:    football.MatchWeather{MatchID: matchID, Forecast: &football.WeatherSnapshot{ID: "weather-1", MatchID: matchID, ValidAt: now, IssuedAt: now, ReceivedAt: now}},
		prediction: football.MatchPrediction{
			MatchID: matchID, ClosesAt: now.Add(time.Hour), IsOpen: true, MySelection: &selection,
			Options: []football.PredictionOption{{Selection: football.PredictionHome}},
		},
	}
	handler := testHandler(store, "secret")
	for _, path := range []string{
		"/v1/matches/" + matchID + "/odds",
		"/v1/matches/" + matchID + "/broadcasts?country_code=SC",
		"/v1/matches/" + matchID + "/weather",
		"/v1/matches/" + matchID + "/prediction",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s returned %d: %s", path, response.Code, response.Body.String())
		}
	}

	privateRequest := httptest.NewRequest(http.MethodGet, "/v1/installations/installation-1/matches/"+matchID+"/prediction", nil)
	privateRequest.Header.Set("Authorization", "Bearer installation-secret")
	privateResponse := httptest.NewRecorder()
	handler.ServeHTTP(privateResponse, privateRequest)
	if privateResponse.Code != http.StatusOK || privateResponse.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("unexpected private prediction response: %d headers=%v body=%s", privateResponse.Code, privateResponse.Header(), privateResponse.Body.String())
	}
}

func TestFootballInternalCoverageRequiresIngestAuth(t *testing.T) {
	t.Parallel()
	request := httptest.NewRequest(http.MethodPut, "/v1/internal/matches/match-1/coverage/provider", strings.NewReader(`{"lineups":[]}`))
	response := httptest.NewRecorder()
	testHandler(&fakeStore{}, "expected-secret").ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestFootballBroadcastRejectsUnsafeURLs(t *testing.T) {
	t.Parallel()
	body := `{
      "network_name":"Unsafe Network",
      "kind":"stream",
      "availability_scope":"global",
      "status":"scheduled",
      "observed_at":"2026-08-01T12:00:00Z",
      "web_url":"javascript:alert(1)"
    }`
	request := httptest.NewRequest(http.MethodPut, "/v1/internal/matches/match-1/broadcasts/provider/external-1", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	testHandler(&fakeStore{}, "secret").ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", response.Code, response.Body.String())
	}
}
