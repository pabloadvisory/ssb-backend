package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/domain/news"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
	"github.com/pabloadvisory/ssb-backend/internal/platform/httpx"
	"github.com/pabloadvisory/ssb-backend/internal/realtime"
	"github.com/pabloadvisory/ssb-backend/internal/service"
)

type fakeStore struct {
	leagues       []football.League
	seasons       football.LeagueSeasons
	match         football.Match
	venue         football.Venue
	lineups       football.MatchLineups
	statistics    football.MatchStatistics
	standings     football.SeasonStandings
	officials     football.MatchOfficials
	searchResults []football.SearchResult
	h2hMatches    []football.Match
	odds          football.MatchOdds
	broadcasts    football.MatchBroadcasts
	weather       football.MatchWeather
	prediction    football.MatchPrediction
	onGetMatch    func()
}

func (store *fakeStore) ListLeagues(context.Context, football.LeagueFilter) ([]football.League, error) {
	return store.leagues, nil
}
func (store *fakeStore) GetLeague(context.Context, string) (football.League, error) {
	if len(store.leagues) == 0 {
		return football.League{}, football.ErrNotFound
	}
	return store.leagues[0], nil
}
func (store *fakeStore) ListLeagueSeasons(context.Context, string) (football.LeagueSeasons, error) {
	if store.seasons.LeagueID == "" {
		return football.LeagueSeasons{}, football.ErrNotFound
	}
	return store.seasons, nil
}
func (*fakeStore) GetTeam(context.Context, string) (football.Team, error) {
	return football.Team{}, football.ErrNotFound
}
func (*fakeStore) GetPlayer(context.Context, string) (football.Player, error) {
	return football.Player{}, football.ErrNotFound
}
func (*fakeStore) GetCoach(context.Context, string) (football.Coach, error) {
	return football.Coach{}, football.ErrNotFound
}
func (*fakeStore) ListMatches(context.Context, football.MatchFilter) ([]football.Match, error) {
	return nil, nil
}
func (store *fakeStore) GetMatch(context.Context, string) (football.Match, error) {
	if store.onGetMatch != nil {
		store.onGetMatch()
	}
	if store.match.ID == "" {
		return football.Match{}, football.ErrNotFound
	}
	return store.match, nil
}
func (*fakeStore) ListMatchEvents(context.Context, string, football.EventFilter) ([]football.MatchEvent, error) {
	return nil, nil
}
func (store *fakeStore) UpsertMatchSnapshot(context.Context, string, string, football.MatchSnapshot) (football.Match, error) {
	return store.match, nil
}

type healthyDatabase struct{}

func (healthyDatabase) Ping(context.Context) error { return nil }

type fakeNewsStore struct {
	articles []news.ArticleSummary
	article  news.Article
}

func (store *fakeNewsStore) ListPublishedArticles(context.Context, news.Filter) ([]news.ArticleSummary, error) {
	return store.articles, nil
}

func (store *fakeNewsStore) GetPublishedArticleBySlug(context.Context, string) (news.Article, error) {
	if store.article.ID == "" {
		return news.Article{}, news.ErrNotFound
	}
	return store.article, nil
}

func (store *fakeNewsStore) UpsertArticle(context.Context, string, string, news.UpsertArticle) (news.Article, error) {
	return store.article, nil
}

func testHandler(store *fakeStore, ingestKey string) http.Handler {
	clientIPs, err := httpx.NewClientIPResolver(nil)
	if err != nil {
		panic(err)
	}
	return testHandlerWithAbuse(store, ingestKey, AbuseControls{
		ClientIPs:           clientIPs,
		Installations:       httpx.NewRateLimiter(1_000, time.Minute, 1_000, 10_000),
		RealtimeConnections: httpx.NewConnectionLimiter(1_000, 10_000),
	})
}

func testHandlerWithAbuse(store *fakeStore, ingestKey string, abuse AbuseControls) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(
		service.NewFootball(store), service.NewNews(&fakeNewsStore{}), nil,
		healthyDatabase{}, realtime.NewHub(), logger, observability.NopMetrics{}, ingestKey, "editorial-secret", abuse,
	).Handler()
}

func TestLivenessIncludesRequestID(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()
	testHandler(&fakeStore{}, "secret").ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
}

func TestPublicGETSupportsETagRevalidation(t *testing.T) {
	t.Parallel()
	handler := testHandler(&fakeStore{leagues: []football.League{{ID: "league-1", Name: "League"}}}, "secret")
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/v1/leagues", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" || !strings.Contains(first.Header().Get("Cache-Control"), "public") {
		t.Fatalf("expected public cache headers, got ETag=%q Cache-Control=%q", etag, first.Header().Get("Cache-Control"))
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/leagues", nil)
	request.Header.Set("If-None-Match", etag)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, request)
	if second.Code != http.StatusNotModified || second.Body.Len() != 0 {
		t.Fatalf("expected empty 304, got status=%d body=%q", second.Code, second.Body.String())
	}
}

func TestLeagueListReturnsOpaqueNextCursor(t *testing.T) {
	t.Parallel()

	store := &fakeStore{leagues: []football.League{
		{ID: "00000000-0000-0000-0000-000000000001", Name: "League A"},
		{ID: "00000000-0000-0000-0000-000000000002", Name: "League B"},
	}}
	request := httptest.NewRequest(http.MethodGet, "/v1/leagues?limit=1", nil)
	response := httptest.NewRecorder()
	testHandler(store, "secret").ServeHTTP(response, request)

	var body struct {
		Data []football.League `json:"data"`
		Page pageInfo          `json:"page"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || !body.Page.HasMore || body.Page.NextCursor == "" {
		t.Fatalf("unexpected page response: %+v", body)
	}
}

func TestNewsListReturnsSummaryPageWithOpaqueCursor(t *testing.T) {
	t.Parallel()

	publishedAt := time.Now().UTC()
	newsStore := &fakeNewsStore{articles: []news.ArticleSummary{
		{ID: "00000000-0000-0000-0000-000000000002", Slug: "latest", Title: "Latest", PublishedAt: &publishedAt},
		{ID: "00000000-0000-0000-0000-000000000001", Slug: "earlier", Title: "Earlier", PublishedAt: &publishedAt},
	}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(
		service.NewFootball(&fakeStore{}), service.NewNews(newsStore), nil,
		healthyDatabase{}, realtime.NewHub(), logger, observability.NopMetrics{}, "secret", "editorial-secret", AbuseControls{},
	).Handler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/news?limit=1", nil))

	var body struct {
		Data []news.ArticleSummary `json:"data"`
		Page pageInfo              `json:"page"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(body.Data) != 1 || !body.Page.HasMore || body.Page.NextCursor == "" {
		t.Fatalf("unexpected news page: status=%d body=%+v", response.Code, body)
	}
}

func TestIngestRequiresBearerToken(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPut, "/v1/internal/matches/provider/42", nil)
	response := httptest.NewRecorder()
	testHandler(&fakeStore{}, "expected-secret").ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestEditorialUpsertRequiresSeparateBearerToken(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPut, "/v1/internal/news/cms/42", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	testHandler(&fakeStore{}, "secret").ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestInvalidMatchStatusIsRejected(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/v1/matches?status=unknown", nil)
	response := httptest.NewRecorder()
	testHandler(&fakeStore{}, "secret").ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", response.Code)
	}
}

func TestInstallationCreationIsRateLimitedByClientIP(t *testing.T) {
	t.Parallel()
	clientIPs, err := httpx.NewClientIPResolver(nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := testHandlerWithAbuse(&fakeStore{}, "secret", AbuseControls{
		ClientIPs:           clientIPs,
		Installations:       httpx.NewRateLimiter(1, time.Minute, 1, 100),
		RealtimeConnections: httpx.NewConnectionLimiter(10, 100),
	})

	first := httptest.NewRequest(http.MethodPost, "/v1/installations", strings.NewReader("{"))
	first.RemoteAddr = "203.0.113.7:1234"
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected first request to reach validation, got %d", firstResponse.Code)
	}

	second := httptest.NewRequest(http.MethodPost, "/v1/installations", strings.NewReader("{"))
	second.RemoteAddr = "203.0.113.7:5678"
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", secondResponse.Code)
	}
	if secondResponse.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestMatchStreamStartsWithReadyEvent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/v1/matches/match-1/stream", nil).WithContext(ctx)
	response := newStreamingRecorder(cancel)
	store := &fakeStore{match: football.Match{ID: "match-1", Version: 3, KickoffAt: time.Now()}}
	testHandler(store, "secret").ServeHTTP(response, request)

	if response.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected event stream content type, got %q", response.Header().Get("Content-Type"))
	}
	if response.body == "" {
		t.Fatal("expected initial SSE event")
	}
}

func TestMatchWebSocketStartsWithReadyEvent(t *testing.T) {
	t.Parallel()

	store := &fakeStore{match: football.Match{ID: "match-1", Version: 7, KickoffAt: time.Now()}}
	server := httptest.NewServer(testHandler(store, "secret"))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/v1/matches/match-1/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(websocket.StatusNormalClosure, "test complete")

	var update realtime.Update
	if err := wsjson.Read(ctx, connection, &update); err != nil {
		t.Fatal(err)
	}
	if update.Type != "ready" || update.MatchID != "match-1" || update.Version != 7 {
		t.Fatalf("unexpected update: %+v", update)
	}
}

func TestMatchWebSocketDoesNotLoseUpdateDuringSnapshotRead(t *testing.T) {
	t.Parallel()

	hub := realtime.NewHub()
	store := &fakeStore{match: football.Match{ID: "match-1", Version: 7, KickoffAt: time.Now()}}
	store.onGetMatch = func() {
		hub.Publish(realtime.Update{MatchID: "match-1", Type: "match.updated", Version: 8})
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(
		service.NewFootball(store), service.NewNews(&fakeNewsStore{}), nil,
		healthyDatabase{}, hub, logger, observability.NopMetrics{}, "secret", "editorial-secret", AbuseControls{},
	).Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/v1/matches/match-1/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(websocket.StatusNormalClosure, "test complete")

	var ready, changed realtime.Update
	if err := wsjson.Read(ctx, connection, &ready); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Read(ctx, connection, &changed); err != nil {
		t.Fatal(err)
	}
	if ready.Type != "ready" || ready.Version != 7 || changed.Type != "match.updated" || changed.Version != 8 {
		t.Fatalf("unexpected convergence sequence: ready=%+v changed=%+v", ready, changed)
	}
}

type streamingRecorder struct {
	header http.Header
	body   string
	cancel context.CancelFunc
}

func newStreamingRecorder(cancel context.CancelFunc) *streamingRecorder {
	return &streamingRecorder{header: make(http.Header), cancel: cancel}
}

func (recorder *streamingRecorder) Header() http.Header { return recorder.header }
func (recorder *streamingRecorder) WriteHeader(int)     {}
func (recorder *streamingRecorder) Write(body []byte) (int, error) {
	recorder.body += string(body)
	return len(body), nil
}
func (recorder *streamingRecorder) Flush() { recorder.cancel() }
