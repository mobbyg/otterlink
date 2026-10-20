package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommunityBoardRoutesRequireAuthentication(t *testing.T) {
	server := (&Server{}).Handler()
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/boards"},
		{http.MethodGet, "/api/boards/1/threads"},
		{http.MethodPost, "/api/boards/1/threads"},
		{http.MethodGet, "/api/board-threads/1"},
		{http.MethodPost, "/api/board-threads/1/replies"},
		{http.MethodPost, "/api/board-threads/1/read"},
		{http.MethodPatch, "/api/board-threads/1/moderation"},
		{http.MethodDelete, "/api/board-threads/1"},
		{http.MethodDelete, "/api/board-posts/1"},
		{http.MethodPost, "/api/boards/1/users/member/ban"},
		{http.MethodGet, "/api/admin/boards"},
		{http.MethodPost, "/api/admin/boards"},
		{http.MethodPut, "/api/admin/boards/1"},
		{http.MethodDelete, "/api/admin/boards/1"},
		{http.MethodPut, "/api/admin/boards/1/moderators/member"},
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
