package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/chat"
)

func (s *Server) adminChatChannels(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	channels, err := s.Chat.ListChannels()
	if err != nil {
		http.Error(w, "unable to list chat channels", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": channels})
}

func (s *Server) adminChatChannelCreate(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var req chatChannelRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	channel, err := s.Chat.CreateChannel(chatUser(admin), req.Name, true, req.AllowOpsToCreateOps, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Accounts.LogAudit(admin, "admin.chat_channel_create", channel.Name, "success", "Permanent chat channel created", &channel.ID); err != nil {
		http.Error(w, "unable to record audit event", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, channel)
}

func (s *Server) adminChatChannelDelete(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("channelID"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	channels, err := s.Chat.ListChannels()
	if err != nil {
		http.Error(w, "unable to list chat channels", http.StatusInternalServerError)
		return
	}
	var name string
	for _, channel := range channels {
		if channel.ID == id { name = channel.Name; break }
	}
	if name == "" {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}
	if err := s.Chat.DeleteChannel(id); err != nil {
		http.Error(w, "unable to delete channel", http.StatusInternalServerError)
		return
	}
	if err := s.Accounts.LogAudit(admin, "admin.chat_channel_delete", name, "success", "Chat channel deleted", &id); err != nil {
		http.Error(w, "unable to record audit event", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adminChatChannelRole(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("channelID"), 10, 64)
	if err != nil || id < 1 { http.Error(w, "invalid channel id", http.StatusBadRequest); return }
	username := strings.TrimSpace(r.PathValue("username"))
	target, err := s.Accounts.GetByUsername(username)
	if err != nil { http.Error(w, "user not found", http.StatusNotFound); return }
	var req chatRoleRequest
	if !decodeJSON(w, r, &req) { return }
	if req.Role != "mod" && req.Role != "op" && req.Role != "user" {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}
	if err := ensurePermanentChannel(s, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// The server admin is not subject to channel-role hierarchy. To use the
	// same role storage, temporarily ensure the target is a member first.
	if _, err := s.Chat.Role(id, target.ID); err != nil {
		if _, err := s.Chat.Join(id, chatUser(target)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if req.Role == "mod" {
		if err := s.Chat.AdminSetRole(id, target.ID, "mod"); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	} else if req.Role == "op" {
		if err := s.Chat.AdminSetRole(id, target.ID, "op"); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	} else {
		if err := s.Chat.AdminSetRole(id, target.ID, "user"); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	}
	if err := s.Accounts.LogAudit(admin, "admin.chat_channel_role", target.Username, "success", "Permanent channel role changed", &target.ID); err != nil {
		http.Error(w, "unable to record audit event", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func ensurePermanentChannel(s *Server, id int64) error {
	channels, err := s.Chat.ListChannels()
	if err != nil { return err }
	for _, channel := range channels {
		if channel.ID == id {
			if !channel.Permanent { return errors.New("administrator role management is limited to permanent channels") }
			return nil
		}
	}
	return errors.New("channel not found")
}
