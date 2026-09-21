package web

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/buddies"
	"github.com/mobbyg/otterlink/server/internal/chat"
	"github.com/mobbyg/otterlink/server/internal/dm"
	"github.com/mobbyg/otterlink/server/internal/presence"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	Accounts accounts.Service
	Buddies  buddies.Service
	Presence *presence.Service
	Chat     *chat.Hub
	DM       dm.Service
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /admin", s.adminIndex)
	mux.HandleFunc("GET /static/", s.static)
	mux.HandleFunc("GET /api/presence", s.presenceList)
	mux.HandleFunc("POST /api/presence/away", s.presenceAway)
	mux.HandleFunc("GET /api/buddies", s.buddyList)
	mux.HandleFunc("POST /api/buddies", s.buddyAdd)
	mux.HandleFunc("DELETE /api/buddies", s.buddyRemove)
	mux.HandleFunc("GET /api/messages", s.messageConversation)
	mux.HandleFunc("POST /api/messages", s.messageSend)
	mux.HandleFunc("POST /api/messages/read", s.messageRead)
	mux.HandleFunc("GET /api/messages/unread", s.messageUnread)

	mux.HandleFunc("GET /api/chat/channels", s.chatChannels)
	mux.HandleFunc("POST /api/chat/channels", s.chatChannelCreate)
	mux.HandleFunc("GET /api/chat/channels/{channelID}", s.chatChannel)
	mux.HandleFunc("POST /api/chat/channels/{channelID}/join", s.chatChannelJoin)
	mux.HandleFunc("POST /api/chat/channels/{channelID}/leave", s.chatChannelLeave)
	mux.HandleFunc("POST /api/chat/channels/{channelID}/messages", s.chatChannelMessages)
	mux.HandleFunc("POST /api/chat/channels/{channelID}/users/{username}/role", s.chatChannelRole)
	mux.HandleFunc("POST /api/chat/channels/{channelID}/users/{username}/moderate", s.chatChannelModerate)

	// Legacy routes remain so older clients fail cleanly while the channel-aware
	// client is being rolled out.
	mux.HandleFunc("GET /api/chat", s.chatLegacyList)
	mux.HandleFunc("POST /api/chat", s.chatLegacySend)

	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	mux.HandleFunc("GET /api/admin/audit", s.adminAudit)
	mux.HandleFunc("GET /api/admin/users/{username}", s.adminUserDetail)
	mux.HandleFunc("PATCH /api/admin/users/{username}", s.adminUserUpdate)
	mux.HandleFunc("POST /api/admin/users/{username}/password", s.adminUserPasswordReset)
	mux.HandleFunc("POST /api/admin/users/{username}/sessions/revoke", s.adminUserSessionsRevoke)
	mux.HandleFunc("DELETE /api/admin/users/{username}", s.adminUserDelete)
	mux.HandleFunc("GET /api/admin/chat/channels", s.adminChatChannels)
	mux.HandleFunc("POST /api/admin/chat/channels", s.adminChatChannelCreate)
	mux.HandleFunc("DELETE /api/admin/chat/channels/{channelID}", s.adminChatChannelDelete)
	mux.HandleFunc("POST /api/admin/chat/channels/{channelID}/users/{username}/role", s.adminChatChannelRole)
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

type buddyRequest struct { Username string `json:"username"` }

func (s *Server) buddyAdd(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var req buddyRequest
	if !decodeJSON(w, r, &req) { return }
	buddy, err := s.Buddies.Add(user.ID, req.Username)
	if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	writeJSON(w, http.StatusCreated, buddy)
}

func (s *Server) buddyRemove(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" { http.Error(w, "username is required", http.StatusBadRequest); return }
	if err := s.Buddies.Remove(user.ID, username); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) chatLegacyList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	channels, err := s.Chat.ListChannels()
	if err != nil { http.Error(w, "unable to list chat channels", http.StatusInternalServerError); return }
	writeJSON(w, http.StatusOK, map[string]any{"channels": channels})
}

func (s *Server) chatLegacySend(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	http.Error(w, "chat now requires a channel", http.StatusGone)
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
