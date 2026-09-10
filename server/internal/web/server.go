package web

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/buddies"
	"github.com/mobbyg/otterlink/server/internal/chat"
	"github.com/mobbyg/otterlink/server/internal/presence"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	Accounts accounts.Service
	Buddies  buddies.Service
	Presence *presence.Service
	Chat     *chat.Hub
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /admin", s.adminIndex)
	mux.HandleFunc("GET /static/", s.static)
	mux.HandleFunc("GET /api/presence", s.presenceList)
	mux.HandleFunc("GET /api/buddies", s.buddyList)
	mux.HandleFunc("POST /api/buddies", s.buddyAdd)
	mux.HandleFunc("DELETE /api/buddies", s.buddyRemove)
	mux.HandleFunc("GET /api/chat", s.chatList)
	mux.HandleFunc("POST /api/chat", s.chatSend)
	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "web client unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	http.FileServer(http.FS(staticFiles)).ServeHTTP(w, r)
}

func (s *Server) user(r *http.Request) (accounts.User, bool) {
	value := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if value == "" {
		return accounts.User{}, false
	}
	user, err := s.Accounts.FromToken(value)
	if err != nil {
		return accounts.User{}, false
	}

	// HTTP clients are request/response based rather than persistent protocol
	// connections. Refresh the expiring HTTP presence session on each request.
	if s.Presence != nil {
		s.Presence.OnlineHTTP(user, tokenConnectionID(value))
	}
	return user, true
}

func (s *Server) presenceList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.Presence == nil {
		http.Error(w, "presence unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": s.Presence.List()})
}

func (s *Server) buddyList(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	list, err := s.Buddies.List(user.ID)
	if err != nil {
		http.Error(w, "unable to list buddies", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"buddies": list})
}

type buddyRequest struct {
	Username string `json:"username"`
}

func (s *Server) buddyAdd(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req buddyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	buddy, err := s.Buddies.Add(user.ID, req.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, buddy)
}

func (s *Server) buddyRemove(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}
	if err := s.Buddies.Remove(user.ID, username); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) chatList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.Chat == nil {
		http.Error(w, "chat unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": s.Chat.List()})
}

type chatRequest struct {
	Message string `json:"message"`
}

func (s *Server) chatSend(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.Chat == nil {
		http.Error(w, "chat unavailable", http.StatusInternalServerError)
		return
	}
	var req chatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if len([]rune(message)) > 2000 {
		http.Error(w, "message exceeds 2000 characters", http.StatusBadRequest)
		return
	}
	created := s.Chat.Publish(chat.User{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName}, message)
	writeJSON(w, http.StatusCreated, created)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
