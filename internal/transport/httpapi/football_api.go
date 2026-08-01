package httpapi

import (
	"net/http"
	"strings"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/platform/httpx"
)

func (api *API) registerFootballCoverageRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /v1/venues/{id}", api.getVenue)
	router.HandleFunc("GET /v1/search", api.searchFootball)
	router.HandleFunc("GET /v1/matches/head-to-head", api.headToHead)
	router.HandleFunc("GET /v1/matches/{id}/lineups", api.getMatchLineups)
	router.HandleFunc("GET /v1/matches/{id}/statistics", api.getMatchStatistics)
	router.HandleFunc("GET /v1/matches/{id}/officials", api.getMatchOfficials)
	router.HandleFunc("GET /v1/matches/{id}/odds", api.getMatchOdds)
	router.HandleFunc("GET /v1/matches/{id}/broadcasts", api.getMatchBroadcasts)
	router.HandleFunc("GET /v1/matches/{id}/weather", api.getMatchWeather)
	router.HandleFunc("GET /v1/matches/{id}/prediction", api.getMatchPrediction)
	router.HandleFunc("GET /v1/seasons/{id}/standings", api.getSeasonStandings)
	router.HandleFunc("GET /v1/installations/{id}/matches/{match_id}/prediction", api.getInstallationPrediction)
	router.HandleFunc("PUT /v1/installations/{id}/matches/{match_id}/prediction", api.setInstallationPrediction)

	router.Handle("PUT /v1/internal/matches/{id}/coverage/{source}", api.requireIngestAuth(http.HandlerFunc(api.replaceMatchCoverage)))
	router.Handle("PUT /v1/internal/seasons/{id}/standings/{source}", api.requireIngestAuth(http.HandlerFunc(api.replaceSeasonStandings)))
	router.Handle("PUT /v1/internal/matches/{id}/odds/{source}/{external_id}", api.requireIngestAuth(http.HandlerFunc(api.upsertMatchOdds)))
	router.Handle("PUT /v1/internal/matches/{id}/broadcasts/{source}/{external_id}", api.requireIngestAuth(http.HandlerFunc(api.upsertMatchBroadcast)))
	router.Handle("PUT /v1/internal/matches/{id}/weather/{source}/{external_id}", api.requireIngestAuth(http.HandlerFunc(api.upsertMatchWeather)))
}

func (api *API) getVenue(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetVenue(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getMatchLineups(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetMatchLineups(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getMatchStatistics(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetMatchStatistics(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getSeasonStandings(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.ListSeasonStandings(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getMatchOfficials(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.ListMatchOfficials(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) searchFootball(writer http.ResponseWriter, request *http.Request) {
	limit, err := parseLimit(request.URL.Query().Get("limit"), 20, 50)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	values := request.URL.Query()["type"]
	types := make([]football.SearchEntityType, 0, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			if item = strings.TrimSpace(item); item != "" {
				types = append(types, football.SearchEntityType(item))
			}
		}
	}
	results, err := api.football.Search(request.Context(), football.SearchFilter{
		Query: request.URL.Query().Get("q"), Types: types, Limit: limit,
	})
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	api.writeCacheable(writer, request, map[string]any{"data": results})
}

func (api *API) headToHead(writer http.ResponseWriter, request *http.Request) {
	limit, err := parseLimit(request.URL.Query().Get("limit"), 10, 50)
	if err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.football.HeadToHead(request.Context(), football.HeadToHeadFilter{
		TeamAID: request.URL.Query().Get("team_a"), TeamBID: request.URL.Query().Get("team_b"), Limit: limit,
	})
	api.writeResult(writer, request, value, err)
}

func (api *API) getMatchOdds(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.ListMatchOdds(request.Context(), request.PathValue("id"), request.URL.Query().Get("bookmaker"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getMatchBroadcasts(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.ListMatchBroadcasts(request.Context(), request.PathValue("id"), request.URL.Query().Get("country_code"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getMatchWeather(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetMatchWeather(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getMatchPrediction(writer http.ResponseWriter, request *http.Request) {
	value, err := api.football.GetMatchPrediction(request.Context(), request.PathValue("id"))
	api.writeResult(writer, request, value, err)
}

func (api *API) getInstallationPrediction(writer http.ResponseWriter, request *http.Request) {
	credential, ok := bearerCredential(request)
	if !ok {
		writer.Header().Set("WWW-Authenticate", "Bearer")
		httpx.WriteError(writer, request, http.StatusUnauthorized, "unauthorized", "installation credential is required")
		return
	}
	value, err := api.football.GetInstallationPrediction(
		request.Context(), request.PathValue("id"), credential, request.PathValue("match_id"),
	)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func (api *API) setInstallationPrediction(writer http.ResponseWriter, request *http.Request) {
	credential, ok := bearerCredential(request)
	if !ok {
		writer.Header().Set("WWW-Authenticate", "Bearer")
		httpx.WriteError(writer, request, http.StatusUnauthorized, "unauthorized", "installation credential is required")
		return
	}
	var body struct {
		Selection football.PredictionSelection `json:"selection"`
	}
	if err := httpx.DecodeJSON(writer, request, &body, 4<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.football.SetInstallationPrediction(
		request.Context(), request.PathValue("id"), credential, request.PathValue("match_id"), body.Selection,
	)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func (api *API) replaceMatchCoverage(writer http.ResponseWriter, request *http.Request) {
	var command football.MatchCoverageUpdate
	if err := httpx.DecodeJSON(writer, request, &command, 1<<20); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	if err := api.football.ReplaceMatchCoverage(request.Context(), request.PathValue("id"), request.PathValue("source"), command); err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (api *API) replaceSeasonStandings(writer http.ResponseWriter, request *http.Request) {
	var command football.StandingsUpdate
	if err := httpx.DecodeJSON(writer, request, &command, 512<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	if err := api.football.ReplaceSeasonStandings(request.Context(), request.PathValue("id"), request.PathValue("source"), command); err != nil {
		api.handleError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (api *API) upsertMatchOdds(writer http.ResponseWriter, request *http.Request) {
	var command football.UpsertOddsSnapshot
	if err := httpx.DecodeJSON(writer, request, &command, 256<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.football.UpsertMatchOdds(
		request.Context(), request.PathValue("id"), request.PathValue("source"), request.PathValue("external_id"), command,
	)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func (api *API) upsertMatchBroadcast(writer http.ResponseWriter, request *http.Request) {
	var command football.UpsertMatchBroadcast
	if err := httpx.DecodeJSON(writer, request, &command, 128<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.football.UpsertMatchBroadcast(
		request.Context(), request.PathValue("id"), request.PathValue("source"), request.PathValue("external_id"), command,
	)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, value)
}

func (api *API) upsertMatchWeather(writer http.ResponseWriter, request *http.Request) {
	var command football.UpsertWeatherSnapshot
	if err := httpx.DecodeJSON(writer, request, &command, 64<<10); err != nil {
		api.badRequest(writer, request, err)
		return
	}
	value, err := api.football.UpsertMatchWeather(
		request.Context(), request.PathValue("id"), request.PathValue("source"), request.PathValue("external_id"), command,
	)
	if err != nil {
		api.handleError(writer, request, err)
		return
	}
	httpx.WriteJSON(writer, http.StatusOK, value)
}
