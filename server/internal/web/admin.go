package web

import (
	"net/http"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

type adminUser struct {
	accounts.User
	Online bool `json:"online"`
}

func (s *Server) adminUser(r *http.Request) (accounts.User, bool, bool) {
	user, authenticated := s.user(r)
	if !authenticated {
		return accounts.User{}, false, false
	}
	return user, user.Role == "admin", true
}

// The admin page shell itself is public; all administration data and actions
// remain protected by adminUser. This lets a browser navigate to /admin before
// its JavaScript attaches the bearer token from local storage.
func (s *Server) adminIndex(w http.ResponseWriter, _ *http.Request) {
	data, err := staticFiles.ReadFile("static/admin.html")
	if err != nil {
		http.Error(w, "admin client unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	if _, admin, authenticated := s.adminUser(r); !authenticated {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	} else if !admin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	users, err := s.Accounts.List()
	if err != nil {
		http.Error(w, "unable to list users", http.StatusInternalServerError)
		return
	}

	online := make(map[string]bool)
	if s.Presence != nil {
		for _, user := range s.Presence.List() {
			online[user.Username] = true
		}
	}

	result := make([]adminUser, 0, len(users))
	for _, user := range users {
		result = append(result, adminUser{User: user, Online: online[user.Username]})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": result})
}
