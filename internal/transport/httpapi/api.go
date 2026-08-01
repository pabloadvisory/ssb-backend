package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
	"github.com/pabloadvisory/ssb-backend/internal/platform/httpx"
	"github.com/pabloadvisory/ssb-backend/internal/realtime"
	"github.com/pabloadvisory/ssb-backend/internal/service"
)

type databasePinger interface {
	Ping(context.Context) error
}

type API struct {
	football      *service.Football
	notifications *service.Notifications
	database      databasePinger
	hub           *realtime.Hub
	logger        *slog.Logger
	ingestKey     string
	abuse         AbuseControls
	metrics       observability.Metrics
}

type AbuseControls struct {
	ClientIPs           *httpx.ClientIPResolver
	Installations       *httpx.RateLimiter
	RealtimeConnections *httpx.ConnectionLimiter
}

func New(footballService *service.Football, notificationService *service.Notifications, database databasePinger, hub *realtime.Hub, logger *slog.Logger, metrics observability.Metrics, ingestKey string, abuse AbuseControls) *API {
	if metrics == nil {
		metrics = observability.NopMetrics{}
	}
	if abuse.ClientIPs == nil {
		abuse.ClientIPs, _ = httpx.NewClientIPResolver(nil)
	}
	if abuse.Installations == nil {
		abuse.Installations = httpx.NewRateLimiter(20, time.Minute, 5, 100_000)
	}
	if abuse.RealtimeConnections == nil {
		abuse.RealtimeConnections = httpx.NewConnectionLimiter(20, 100_000)
	}
	return &API{
		football: footballService, notifications: notificationService, database: database,
		hub: hub, logger: logger, ingestKey: ingestKey, abuse: abuse, metrics: metrics,
	}
}

func (api *API) Handler() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("GET /health/live", api.live)
	router.HandleFunc("GET /health/ready", api.ready)
	router.HandleFunc("GET /v1/leagues", api.listLeagues)
	router.HandleFunc("GET /v1/leagues/{id}", api.getLeague)
	router.HandleFunc("GET /v1/teams/{id}", api.getTeam)
	router.HandleFunc("GET /v1/players/{id}", api.getPlayer)
	router.HandleFunc("GET /v1/coaches/{id}", api.getCoach)
	router.HandleFunc("GET /v1/matches", api.listMatches)
	router.HandleFunc("GET /v1/matches/{id}", api.getMatch)
	router.HandleFunc("GET /v1/matches/{id}/events", api.listMatchEvents)
	router.HandleFunc("GET /v1/matches/{id}/stream", api.streamMatch)
	router.HandleFunc("GET /v1/matches/{id}/ws", api.websocketMatch)
	router.HandleFunc("POST /v1/installations", api.createInstallation)
	router.HandleFunc("PUT /v1/installations/{id}/push-endpoints/{kind}", api.registerPushEndpoint)
	router.HandleFunc("PUT /v1/installations/{id}/matches/{match_id}", api.setMatchSubscription)
	router.Handle("PUT /v1/internal/matches/{provider}/{external_id}", api.requireIngestAuth(http.HandlerFunc(api.upsertMatch)))

	return httpx.Chain(router,
		httpx.RequestID,
		httpx.ResolveClientIP(api.abuse.ClientIPs),
		httpx.Recover(api.logger),
		httpx.SecurityHeaders,
		httpx.AccessLog(api.logger, api.metrics),
	)
}

type page[T any] struct {
	Data []T      `json:"data"`
	Page pageInfo `json:"page"`
}

type pageInfo struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

func (api *API) live(writer http.ResponseWriter, request *http.Request) {
	httpx.WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (api *API) ready(writer http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	if err := api.database.Ping(ctx); err != nil {
		api.logger.Error("readiness check failed", "error", err)
		httpx.WriteError(writer, request, http.StatusServiceUnavailable, "not_ready", "service dependencies are unavailable")
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func (api *API) listLeagues(writer http.ResponseWriter, request *http.Request) {
	limit, err := parseLimit(request.URL.Query().Get("limit"), 20, 100)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	filter := football.LeagueFilter{CountryCode: request.URL.Query().Get("country_code"), Limit: limit + 1}
	if raw := request.URL.Query().Get("cursor"); raw != "" {
		var cursor leagueCursor
		if err := decodeCursor(raw, &cursor); err != nil || cursor.ID == "" {
			api.badRequest(writer, request, errors.New("cursor is invalid"))
			return
		}
		filter.AfterName, filter.AfterID = cursor.Name, cursor.ID
	}
	leagues, err := api.football.ListLeagues(request.Context(), filter)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	response := page[football.League]{Data: leagues, Page: pageInfo{}}
	if len(leagues) > limit {
		response.Data = leagues[:limit]
		last := response.Data[len(response.Data)-1]
		response.Page = pageInfo{HasMore: true, NextCursor: encodeCursor(leagueCursor{Name: last.Name, ID: last.ID})}
	}
	api.writeCacheable(writer, request, response)
}

func (api *API) getLeague(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetLeague(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getTeam(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetTeam(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getPlayer(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetPlayer(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getCoach(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetCoach(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) listMatches(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	limit, err := parseLimit(query.Get("limit"), 25, 100)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	from, err := parseTime(query.Get("date_from"), false)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	to, err := parseTime(query.Get("date_to"), true)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	filter := football.MatchFilter{
		LeagueID: query.Get("league_id"), SeasonID: query.Get("season_id"), TeamID: query.Get("team_id"),
		Status: football.MatchStatus(query.Get("status")), From: from, To: to, Limit: limit + 1,
	}
	if raw := query.Get("cursor"); raw != "" {
		var cursor matchCursor
		if err := decodeCursor(raw, &cursor); err != nil || cursor.ID == "" || cursor.KickoffAt.IsZero() {
			api.badRequest(writer, request, errors.New("cursor is invalid"))
			return
		}
		filter.AfterKickoff, filter.AfterMatchID = &cursor.KickoffAt, cursor.ID
	}
	matches, err := api.football.ListMatches(request.Context(), filter)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	response := page[football.Match]{Data: matches, Page: pageInfo{}}
	if len(matches) > limit {
		response.Data = matches[:limit]
		last := response.Data[len(response.Data)-1]
		response.Page = pageInfo{HasMore: true, NextCursor: encodeCursor(matchCursor{KickoffAt: last.KickoffAt, ID: last.ID})}
	}
	api.writeCacheable(writer, request, response)
}

func (api *API) getMatch(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetMatch(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) listMatchEvents(writer http.ResponseWriter, request *http.Request) {
	limit, err := parseLimit(request.URL.Query().Get("limit"), 100, 500)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	after := 0
	if raw := request.URL.Query().Get("after_sequence"); raw != "" {
		after, err = strconv.Atoi(raw)
		if err != nil || after < 0 {
			api.badRequest(writer, request, errors.New("after_sequence must be a non-negative integer"))
			return
		}
	}
	events, err := api.football.ListMatchEvents(request.Context(), request.PathValue("id"), football.EventFilter{AfterSequence: after, Limit: limit})
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	api.writeCacheable(writer, request, map[string]any{"data": events})
}

func (api *API) streamMatch(writer http.ResponseWriter, request *http.Request) {
	release, ok := api.acquireRealtime(writer, request)
	if !ok {
		return
	}
	defer release()

	matchID := request.PathValue("id")
	updates, cancel := api.hub.Subscribe(matchID)
	defer cancel()
	match, err := api.football.GetMatch(request.Context(), matchID)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	controller := http.NewResponseController(writer)
	if err := controller.SetWriteDeadline(time.Time{}); err != nil {
		api.logger.Debug("could not disable stream write deadline", "error", err)
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache, no-transform")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	if err := writeSSE(writer, controller, "ready", realtime.Update{MatchID: match.ID, Type: "ready", Version: match.Version}); err != nil {
		return
	}
	latestVersion := match.Version

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if update.Version <= latestVersion {
				continue
			}
			latestVersion = update.Version
			if err := writeSSE(writer, controller, "match.updated", update); err != nil {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(writer, ": heartbeat\n\n"); err != nil {
				return
			}
			if err := controller.Flush(); err != nil {
				return
			}
		}
	}
}

func (api *API) websocketMatch(writer http.ResponseWriter, request *http.Request) {
	release, ok := api.acquireRealtime(writer, request)
	if !ok {
		return
	}
	defer release()

	matchID := request.PathValue("id")
	updates, cancel := api.hub.Subscribe(matchID)
	defer cancel()
	match, err := api.football.GetMatch(request.Context(), matchID)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}

	connection, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		api.logger.Warn("WebSocket upgrade rejected", "error", err, "request_id", httpx.RequestIDFromContext(request.Context()))
		return
	}
	defer connection.Close(websocket.StatusNormalClosure, "stream closed")
	connection.SetReadLimit(1024)
	connectionContext := connection.CloseRead(request.Context())

	if err := writeWebSocket(connectionContext, connection, realtime.Update{MatchID: match.ID, Type: "ready", Version: match.Version}); err != nil {
		return
	}
	latestVersion := match.Version

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-connectionContext.Done():
			return
		case update, ok := <-updates:
			if !ok {
				return
			}
			if update.Version <= latestVersion {
				continue
			}
			latestVersion = update.Version
			if err := writeWebSocket(connectionContext, connection, update); err != nil {
				return
			}
		case <-heartbeat.C:
			pingContext, cancel := context.WithTimeout(connectionContext, 5*time.Second)
			err := connection.Ping(pingContext)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func writeWebSocket(ctx context.Context, connection *websocket.Conn, value any) error {
	writeContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return wsjson.Write(writeContext, connection, value)
}

func writeSSE(writer http.ResponseWriter, controller *http.ResponseController, event string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	return controller.Flush()
}

func (api *API) upsertMatch(writer http.ResponseWriter, request *http.Request) {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		httpx.WriteError(writer, request, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}
	var snapshot football.MatchSnapshot
	if err := httpx.DecodeJSON(writer, request, &snapshot, 4<<20); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	match, err := api.football.UpsertMatchSnapshot(request.Context(), request.PathValue("provider"), request.PathValue("external_id"), snapshot)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("ETag", fmt.Sprintf(`"%d"`, match.Version))
	httpx.WriteJSON(writer, http.StatusOK, match)
}

func (api *API) createInstallation(writer http.ResponseWriter, request *http.Request) {
	clientIP := httpx.ClientIPFromContext(request.Context())
	if allowed, retryAfter := api.abuse.Installations.Allow(clientIP); !allowed {
		writer.Header().Set("Retry-After", strconv.Itoa(max(1, int(retryAfter.Round(time.Second)/time.Second))))
		httpx.WriteError(writer, request, http.StatusTooManyRequests, "rate_limited", "too many installation requests")
		return
	}
	var command notification.CreateInstallation
	if err := httpx.DecodeJSON(writer, request, &command, 16<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	installation, err := api.notifications.CreateInstallation(request.Context(), command)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("Location", "/v1/installations/"+installation.ID)
	httpx.WriteJSON(writer, http.StatusCreated, installation)
}

func (api *API) acquireRealtime(writer http.ResponseWriter, request *http.Request) (func(), bool) {
	clientIP := httpx.ClientIPFromContext(request.Context())
	release, ok := api.abuse.RealtimeConnections.Acquire(clientIP)
	if !ok {
		writer.Header().Set("Retry-After", "30")
		httpx.WriteError(writer, request, http.StatusTooManyRequests, "connection_limit_exceeded", "too many realtime connections")
		return nil, false
	}
	return release, true
}

func (api *API) registerPushEndpoint(writer http.ResponseWriter, request *http.Request) {
	credential, ok := bearerCredential(request)
	if !ok {
		writer.Header().Set("WWW-Authenticate", "Bearer")
		httpx.WriteError(writer, request, http.StatusUnauthorized, "unauthorized", "installation credential is required")
		return
	}
	var command notification.RegisterEndpoint
	if err := httpx.DecodeJSON(writer, request, &command, 32<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	endpoint, err := api.notifications.RegisterEndpoint(
		request.Context(), request.PathValue("id"), credential,
		notification.EndpointKind(request.PathValue("kind")), command,
	)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, endpoint)
}

func (api *API) setMatchSubscription(writer http.ResponseWriter, request *http.Request) {
	credential, ok := bearerCredential(request)
	if !ok {
		writer.Header().Set("WWW-Authenticate", "Bearer")
		httpx.WriteError(writer, request, http.StatusUnauthorized, "unauthorized", "installation credential is required")
		return
	}
	var body struct {
		NotificationsEnabled *bool `json:"notifications_enabled"`
	}
	if err := httpx.DecodeJSON(writer, request, &body, 4<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	if body.NotificationsEnabled == nil {
		api.badRequest(writer, request, errors.New("notifications_enabled is required"))
		return
	}
	subscription, err := api.notifications.SetMatchSubscription(
		request.Context(), request.PathValue("id"), credential, request.PathValue("match_id"), *body.NotificationsEnabled,
	)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, subscription)
}

func bearerCredential(request *http.Request) (string, bool) {
	header := request.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	credential := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return credential, credential != ""
}

func (api *API) requireIngestAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if api.ingestKey == "" {
			httpx.WriteError(writer, request, http.StatusServiceUnavailable, "ingestion_unavailable", "ingestion is not configured")
			return
		}
		provided := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
		if len(provided) != len(api.ingestKey) || subtle.ConstantTimeCompare([]byte(provided), []byte(api.ingestKey)) != 1 {
			writer.Header().Set("WWW-Authenticate", "Bearer")
			httpx.WriteError(writer, request, http.StatusUnauthorized, "unauthorized", "valid ingestion credentials are required")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (api *API) writeResult(writer http.ResponseWriter, request *http.Request, value any, err error) {
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	api.writeCacheable(writer, request, value)
}

func (api *API) writeCacheable(writer http.ResponseWriter, request *http.Request, value any) {
	if err := httpx.WriteCacheableJSON(writer, request, http.StatusOK, value, 15*time.Second); err != nil {
		api.logger.Error("encode cached response", "error", err, "request_id", httpx.RequestIDFromContext(request.Context()))
	}
}

func (api *API) badRequest(writer http.ResponseWriter, request *http.Request, err error) {
	httpx.WriteError(writer, request, http.StatusBadRequest, "invalid_request", err.Error())
}

func (api *API) handleError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, football.ErrNotFound):
		httpx.WriteError(writer, request, http.StatusNotFound, "not_found", "resource was not found")
	case errors.Is(err, football.ErrInvalid):
		httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	case errors.Is(err, football.ErrConflict):
		httpx.WriteError(writer, request, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, notification.ErrUnauthorized):
		writer.Header().Set("WWW-Authenticate", "Bearer")
		httpx.WriteError(writer, request, http.StatusUnauthorized, "unauthorized", "installation credentials are invalid")
	case errors.Is(err, notification.ErrInvalid):
		httpx.WriteError(writer, request, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	case errors.Is(err, context.Canceled):
		return
	default:
		api.logger.Error("request failed", "error", err, "request_id", httpx.RequestIDFromContext(request.Context()))
		httpx.WriteError(writer, request, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
