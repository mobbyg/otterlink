package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/dm"
)

type messageRequest struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func (s *Server) messageConversation(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	username := strings.TrimSpace(r.URL.Query().Get("with"))
	if username == "" { http.Error(w, "with is required", http.StatusBadRequest); return }
	conversation, err := s.DM.Conversation(user.ID, username)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "user not found" { status = http.StatusNotFound }
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusOK, conversation)
}

func (s *Server) messageSend(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var req messageRequest
	if !decodeJSON(w, r, &req) { return }
	message, err := s.DM.Send(user.ID, req.Username, req.Message)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "user not found" { status = http.StatusNotFound }
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusCreated, message)
}

func (s *Server) messageRead(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var req messageRequest
	if !decodeJSON(w, r, &req) { return }
	if err := s.DM.MarkRead(user.ID, req.Username); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "user not found" { status = http.StatusNotFound }
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) messageUnread(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	unread, err := s.DM.Unread(user.ID)
	if err != nil { http.Error(w, "unable to load unread messages", http.StatusInternalServerError); return }
	writeJSON(w, http.StatusOK, map[string]any{"messages": unread})
}

var _ dm.Service
var _ = json.Valid
