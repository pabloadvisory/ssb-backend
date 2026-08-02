package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlayerRoutesValidateRequests(t *testing.T) {
	t.Parallel()
	handler := testHandler(&fakeStore{}, "ingest-secret")
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "discovery", method: http.MethodGet, path: "/v1/players?q=alex", status: http.StatusOK},
		{name: "comparison needs cohort", method: http.MethodGet, path: "/v1/players/compare?player_id=one&player_id=two", status: http.StatusUnprocessableEntity},
		{name: "statistics needs season", method: http.MethodGet, path: "/v1/players/player-1/statistics", status: http.StatusUnprocessableEntity},
		{name: "traits needs season", method: http.MethodGet, path: "/v1/players/player-1/traits", status: http.StatusUnprocessableEntity},
		{name: "heatmap needs scope", method: http.MethodGet, path: "/v1/players/player-1/heatmap", status: http.StatusUnprocessableEntity},
		{name: "history cursor", method: http.MethodGet, path: "/v1/players/player-1/matches?cursor=invalid", status: http.StatusBadRequest},
		{name: "private follows", method: http.MethodGet, path: "/v1/installations/installation-1/player-follows", status: http.StatusUnauthorized},
		{name: "internal traits", method: http.MethodPut, path: "/v1/internal/players/player-1/traits/demo/trait-1", status: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("expected %d, got %d: %s", test.status, response.Code, response.Body.String())
			}
		})
	}
}
