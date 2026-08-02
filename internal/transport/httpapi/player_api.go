package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/platform/httpx"
)

func (api *API) registerPlayerRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /v1/players", api.listPlayers)
	router.HandleFunc("GET /v1/players/compare", api.comparePlayers)
	router.HandleFunc("GET /v1/players/{id}/memberships", api.listPlayerMemberships)
	router.HandleFunc("GET /v1/players/{id}/matches", api.listPlayerMatches)
	router.HandleFunc("GET /v1/players/{id}/statistics", api.getPlayerStatistics)
	router.HandleFunc("GET /v1/players/{id}/career", api.getPlayerCareer)
	router.HandleFunc("GET /v1/players/{id}/traits", api.getPlayerTraits)
	router.HandleFunc("GET /v1/players/{id}/heatmap", api.getPlayerHeatmap)
	router.HandleFunc("GET /v1/players/{id}/shots", api.listPlayerShots)
	router.HandleFunc("GET /v1/players/{id}/valuation", api.getPlayerValuation)

	router.HandleFunc("GET /v1/installations/{id}/player-follows", api.listPlayerFollows)
	router.HandleFunc("PUT /v1/installations/{id}/player-follows/{player_id}", api.setPlayerFollow)
	router.HandleFunc("DELETE /v1/installations/{id}/player-follows/{player_id}", api.deletePlayerFollow)
	router.HandleFunc("GET /v1/installations/{id}/notification-preferences", api.getNotificationPreferences)
	router.HandleFunc("PUT /v1/installations/{id}/notification-preferences", api.setNotificationPreferences)

	router.Handle("PUT /v1/internal/players/{id}/traits/{source}/{external_id}", api.requireIngestAuth(http.HandlerFunc(api.upsertPlayerTraits)))
	router.Handle("PUT /v1/internal/matches/{match_id}/players/{id}/spatial/{source}/{external_id}", api.requireIngestAuth(http.HandlerFunc(api.upsertPlayerSpatial)))
	router.Handle("PUT /v1/internal/players/{id}/valuations/{source}/{external_id}", api.requireIngestAuth(http.HandlerFunc(api.upsertPlayerValuation)))
}

func (api *API) listPlayers(writer http.ResponseWriter, request *http.Request) {
	limit, err := parseLimit(request.URL.Query().Get("limit"), 25, 100)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	filter := football.PlayerDiscoveryFilter{
		Query: request.URL.Query().Get("q"), LeagueID: request.URL.Query().Get("league_id"),
		SeasonID: request.URL.Query().Get("season_id"), TeamID: request.URL.Query().Get("team_id"),
		Position: request.URL.Query().Get("position"), Limit: limit + 1,
	}
	if raw := request.URL.Query().Get("cursor"); raw != "" {
		var cursor playerCursor
		if err := decodeCursor(raw, &cursor); err != nil || cursor.ID == "" {
			api.badRequest(writer, request, errors.New("cursor is invalid"))
			return
		}
		filter.AfterName, filter.AfterID = cursor.Name, cursor.ID
	}
	players, err := api.football.ListPlayers(request.Context(), filter)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	response := page[football.PlayerDiscoveryResult]{Data: players, Page: pageInfo{}}
	if len(players) > limit {
		response.Data = players[:limit]
		last := response.Data[len(response.Data)-1].Player
		response.Page = pageInfo{HasMore: true, NextCursor: encodeCursor(playerCursor{Name: last.DisplayName, ID: last.ID})}
	}
	api.writeCacheable(writer, request, response)
}

func (api *API) comparePlayers(writer http.ResponseWriter, request *http.Request) {
	ids := make([]string, 0, len(request.URL.Query()["player_id"]))
	for _, value := range request.URL.Query()["player_id"] {
		ids = append(ids, strings.Split(value, ",")...)
	}
	value, err := api.football.ComparePlayers(request.Context(), football.PlayerComparisonFilter{
		PlayerIDs: ids, SeasonID: request.URL.Query().Get("season_id"), LeagueID: request.URL.Query().Get("league_id"),
	})
	api.writeResult(writer, request, value, err)
}

func (api *API) listPlayerMemberships(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.ListPlayerMemberships(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) listPlayerMatches(writer http.ResponseWriter, request *http.Request) {
	limit, err := parseLimit(request.URL.Query().Get("limit"), 25, 100)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	filter := football.PlayerMatchFilter{
		SeasonID: request.URL.Query().Get("season_id"), LeagueID: request.URL.Query().Get("league_id"), Limit: limit + 1,
	}
	if raw := request.URL.Query().Get("cursor"); raw != "" {
		var cursor playerMatchCursor
		if err := decodeCursor(raw, &cursor); err != nil || cursor.ID == "" || cursor.KickoffAt.IsZero() {
			api.badRequest(writer, request, errors.New("cursor is invalid"))
			return
		}
		filter.BeforeKickoff, filter.BeforeMatchID = &cursor.KickoffAt, cursor.ID
	}
	matches, err := api.football.ListPlayerMatches(request.Context(), request.PathValue("id"), filter)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	response := page[football.PlayerMatchHistoryItem]{Data: matches, Page: pageInfo{}}
	if len(matches) > limit {
		response.Data = matches[:limit]
		last := response.Data[len(response.Data)-1].Fixture
		response.Page = pageInfo{HasMore: true, NextCursor: encodeCursor(playerMatchCursor{KickoffAt: last.KickoffAt, ID: last.ID})}
	}
	api.writeCacheable(writer, request, response)
}

func (api *API) getPlayerStatistics(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetPlayerSeasonStatistics(request.Context(), request.PathValue("id"), request.URL.Query().Get("season_id"), request.URL.Query().Get("league_id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getPlayerCareer(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetPlayerCareer(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func playerAnalyticsFilter(request *http.Request) football.PlayerAnalyticsFilter {
	return football.PlayerAnalyticsFilter{
		SeasonID: request.URL.Query().Get("season_id"), LeagueID: request.URL.Query().Get("league_id"),
		MatchID: request.URL.Query().Get("match_id"), Source: request.URL.Query().Get("source"),
	}
}

func (api *API) getPlayerTraits(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetPlayerTraits(request.Context(), request.PathValue("id"), playerAnalyticsFilter(request))
	api.writeResult(writer, request, value, err)
}

func (api *API) getPlayerHeatmap(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetPlayerHeatmap(request.Context(), request.PathValue("id"), playerAnalyticsFilter(request))
	api.writeResult(writer, request, value, err)
}

func (api *API) listPlayerShots(writer http.ResponseWriter, request *http.Request) {
	limit, err := parseLimit(request.URL.Query().Get("limit"), 100, 200)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	filter := playerAnalyticsFilter(request)
	filter.Limit = limit
	value, err := api.football.ListPlayerShots(request.Context(), request.PathValue("id"), filter)
	api.writeResult(writer, request, map[string]any{"data": value}, err)
}

func (api *API) getPlayerValuation(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetPlayerValuation(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) upsertPlayerTraits(writer http.ResponseWriter, request *http.Request) {
	var command football.UpsertPlayerTraits
	if err := httpx.DecodeJSON(writer, request, &command, 256<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.football.UpsertPlayerTraits(request.Context(), request.PathValue("id"), request.PathValue("source"), request.PathValue("external_id"), command)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func (api *API) upsertPlayerSpatial(writer http.ResponseWriter, request *http.Request) {
	var command football.UpsertPlayerSpatial
	if err := httpx.DecodeJSON(writer, request, &command, 2<<20); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	if err := api.football.UpsertPlayerSpatial(request.Context(), request.PathValue("match_id"), request.PathValue("id"), request.PathValue("source"), request.PathValue("external_id"), command); err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (api *API) upsertPlayerValuation(writer http.ResponseWriter, request *http.Request) {
	var command football.UpsertPlayerValuation
	if err := httpx.DecodeJSON(writer, request, &command, 32<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.football.UpsertPlayerValuation(request.Context(), request.PathValue("id"), request.PathValue("source"), request.PathValue("external_id"), command)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func installationCredential(writer http.ResponseWriter, request *http.Request) (string, bool) {
	credential, ok := bearerCredential(request)
	if !ok {
		writer.Header().Set("WWW-Authenticate", "Bearer")
		httpx.WriteError(writer, request, http.StatusUnauthorized, "unauthorized", "installation credential is required")
	}
	return credential, ok
}

func (api *API) listPlayerFollows(writer http.ResponseWriter, request *http.Request) {
	credential, ok := installationCredential(writer, request)
	if !ok {
		return
	}
	limit, err := parseLimit(request.URL.Query().Get("limit"), 20, 100)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	filter := notification.PlayerFollowFilter{Limit: limit + 1}
	if raw := request.URL.Query().Get("cursor"); raw != "" {
		var cursor playerFollowCursor
		if err := decodeCursor(raw, &cursor); err != nil || cursor.PlayerID == "" || cursor.FollowedAt.IsZero() {
			api.badRequest(writer, request, errors.New("cursor is invalid"))
			return
		}
		filter.BeforeFollowedAt, filter.BeforePlayerID = &cursor.FollowedAt, cursor.PlayerID
	}
	follows, err := api.notifications.ListPlayerFollows(request.Context(), request.PathValue("id"), credential, filter)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	response := page[notification.PlayerFollow]{Data: follows, Page: pageInfo{}}
	if len(follows) > limit {
		response.Data = follows[:limit]
		last := response.Data[len(response.Data)-1]
		response.Page = pageInfo{HasMore: true, NextCursor: encodeCursor(playerFollowCursor{FollowedAt: last.FollowedAt, PlayerID: last.Player.ID})}
	}
	writer.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(writer, http.StatusOK, response)
}

func (api *API) setPlayerFollow(writer http.ResponseWriter, request *http.Request) {
	credential, ok := installationCredential(writer, request)
	if !ok {
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
	value, err := api.notifications.SetPlayerFollow(request.Context(), request.PathValue("id"), credential, request.PathValue("player_id"), notification.SetPlayerFollow{NotificationsEnabled: *body.NotificationsEnabled})
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func (api *API) deletePlayerFollow(writer http.ResponseWriter, request *http.Request) {
	credential, ok := installationCredential(writer, request)
	if !ok {
		return
	}
	if err := api.notifications.DeletePlayerFollow(request.Context(), request.PathValue("id"), credential, request.PathValue("player_id")); err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("Cache-Control", "private, no-store")
	writer.WriteHeader(http.StatusNoContent)
}

func (api *API) getNotificationPreferences(writer http.ResponseWriter, request *http.Request) {
	credential, ok := installationCredential(writer, request)
	if !ok {
		return
	}
	value, err := api.notifications.GetNotificationPreferences(request.Context(), request.PathValue("id"), credential)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func (api *API) setNotificationPreferences(writer http.ResponseWriter, request *http.Request) {
	credential, ok := installationCredential(writer, request)
	if !ok {
		return
	}
	var command notification.SetNotificationPreferences
	if err := httpx.DecodeJSON(writer, request, &command, 4<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.notifications.SetNotificationPreferences(request.Context(), request.PathValue("id"), credential, command)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(writer, http.StatusOK, value)
}
