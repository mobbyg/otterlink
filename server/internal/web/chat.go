package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/chat"
)

type chatChannelRequest struct {
	Name                  string `json:"name"`
	AllowOpsToCreateOps   bool   `json:"allow_ops_to_create_ops"`
}

type chatRoleRequest struct {
	Role string `json:"role"`
}

func parseChannelID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("channelID"), 10, 64)
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

func chatUser(user accounts.User) chat.User {
	return chat.User{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName}
}

func (s *Server) chatChannels(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channels, err := s.Chat.ListChannels()
	if err != nil {
		http.Error(w, "unable to list chat channels", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": channels})
}

func (s *Server) chatChannelCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req chatChannelRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	channel, err := s.Chat.CreateChannel(chatUser(user), req.Name, false, req.AllowOpsToCreateOps, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _, _, _, err = s.Chat.Join(channel.ID, chatUser(user))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, channel)
}

func (s *Server) chatChannel(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channelID, ok := parseChannelID(r)
	if !ok {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	channel, members, messages, err := s.loadChannel(channelID)
	if err != nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}
	role, roleErr := s.Chat.Role(channelID, user.ID)
	if roleErr != nil {
		role = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channel": channel,
		"role": role,
		"members": members,
		"messages": messages,
	})
}

func (s *Server) loadChannel(channelID int64) (chat.Channel, []chat.Member, []chat.Message, error) {
	channels, err := s.Chat.ListChannels()
	if err != nil {
		return chat.Channel{}, nil, nil, err
	}
	var channel chat.Channel
	for _, candidate := range channels {
		if candidate.ID == channelID {
			channel = candidate
			break
		}
	}
	if channel.ID == 0 {
		return chat.Channel{}, nil, nil, errors.New("channel not found")
	}
	members, err := s.Chat.Members(channelID)
	if err != nil {
		return chat.Channel{}, nil, nil, err
	}
	messages, err := s.Chat.Messages(channelID)
	if err != nil {
		return chat.Channel{}, nil, nil, err
	}
	return channel, members, messages, nil
}

func (s *Server) chatChannelJoin(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channelID, ok := parseChannelID(r)
	if !ok {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	channel, member, members, messages, err := s.Chat.Join(channelID, chatUser(user))
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channel": channel, "member": member, "members": members, "messages": messages})
}

func (s *Server) chatChannelLeave(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channelID, ok := parseChannelID(r)
	if !ok {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	if err := s.Chat.Leave(channelID, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) chatChannelMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channelID, ok := parseChannelID(r)
	if !ok {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	var req struct { Message string `json:"message"` }
	if !decodeJSON(w, r, &req) {
		return
	}
	message, err := s.Chat.Publish(channelID, chatUser(user), req.Message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, message)
}

func (s *Server) chatChannelRole(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channelID, ok := parseChannelID(r)
	if !ok {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	targetUsername := strings.TrimSpace(r.PathValue("username"))
	if targetUsername == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}
	target, err := s.Accounts.GetByUsername(targetUsername)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	var req chatRoleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.Chat.SetRole(channelID, user.ID, target.ID, strings.TrimSpace(req.Role)); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) chatChannelModerate(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	channelID, ok := parseChannelID(r)
	if !ok {
		http.Error(w, "invalid channel id", http.StatusBadRequest)
		return
	}
	targetUsername := strings.TrimSpace(r.PathValue("username"))
	target, err := s.Accounts.GetByUsername(targetUsername)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	var action struct { Action string `json:"action"` }
	if !decodeJSON(w, r, &action) {
		return
	}
	var actionErr error
	switch strings.ToLower(strings.TrimSpace(action.Action)) {
	case "kick":
		actionErr = s.Chat.Kick(channelID, user.ID, target.ID)
	case "ban":
		actionErr = s.Chat.Ban(channelID, user.ID, target.ID)
	case "unban":
		actionErr = s.Chat.Unban(channelID, user.ID, target.ID)
	default:
		actionErr = errors.New("invalid moderation action")
	}
	if actionErr != nil {
		http.Error(w, actionErr.Error(), http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
