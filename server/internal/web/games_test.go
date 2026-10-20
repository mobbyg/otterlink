package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGameRoutesRequireAuthentication(t *testing.T) {
	server := (&Server{}).Handler()
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/games"},
		{http.MethodGet, "/api/games/1"},
		{http.MethodPost, "/api/games/1/sessions"},
		{http.MethodGet, "/api/game-sessions/0123456789abcdef/output"},
		{http.MethodPost, "/api/game-sessions/0123456789abcdef/input"},
		{http.MethodDelete, "/api/game-sessions/0123456789abcdef"},
		{http.MethodGet, "/api/admin/games"},
		{http.MethodPost, "/api/admin/games"},
		{http.MethodPost, "/api/admin/games/files"},
		{http.MethodPut, "/api/admin/games/1"},
		{http.MethodDelete, "/api/admin/games/1"},
		{http.MethodGet, "/api/content/screens/slug/games"},
	}

	for _, test := range cases {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("expected unauthenticated request to return %d, got %d: %s",
					http.StatusUnauthorized, response.Code, response.Body.String())
			}
		})
	}
}
