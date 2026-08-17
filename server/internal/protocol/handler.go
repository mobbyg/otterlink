package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/presence"
)

type DefaultHandler struct {
	Accounts accounts.Service
	Presence *presence.Service
}

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenPayload struct {
	Token string `json:"token"`
}

func (h DefaultHandler) Handle(_ context.Context, msg Message) Message {
	switch msg.Service {
	case ServiceSession:
		return h.handleSession(msg)
	case ServicePresence:
		return h.handlePresence(msg)
	default:
		return ErrorResponse(msg.ID, ErrNotFound, "service or action not found")
	}
}

func (h DefaultHandler) handleSession(msg Message) Message {
	switch msg.Action {
	case "ping":
		return SuccessResponse(msg.ID, ServiceSession, "pong", map[string]string{"status": "ok"})
	case "login":
		var input loginPayload
		if err := decodePayload(msg.Payload, &input); err != nil || strings.TrimSpace(input.Username) == "" || input.Password == "" {
			return ErrorResponse(msg.ID, ErrBadRequest, "username and password are required")
		}
		user, token, err := h.Accounts.Authenticate(input.Username, input.Password)
		if err != nil {
			return ErrorResponse(msg.ID, ErrInvalidCredentials, "invalid username or password")
		}
		if h.Presence != nil {
			h.Presence.Online(user)
		}
		return SuccessResponse(msg.ID, ServiceSession, "login", map[string]interface{}{"user": user, "token": token})
	case "whoami":
		user, err := h.authenticate(msg)
		if err != nil {
			return ErrorResponse(msg.ID, ErrUnauthorized, err.Error())
		}
		return SuccessResponse(msg.ID, ServiceSession, "whoami", user)
	case "logout":
		token, err := tokenFromPayload(msg.Payload)
		if err != nil {
			return ErrorResponse(msg.ID, ErrUnauthorized, "valid session token required")
		}
		user, err := h.Accounts.FromToken(token)
		if err != nil {
			return ErrorResponse(msg.ID, ErrUnauthorized, "invalid session")
		}
		if err := h.Accounts.Logout(token); err != nil {
			return ErrorResponse(msg.ID, ErrServer, "unable to end session")
		}
		if h.Presence != nil {
			h.Presence.Offline(user.ID)
		}
		return SuccessResponse(msg.ID, ServiceSession, "logout", map[string]string{"status": "ok"})
	default:
		return ErrorResponse(msg.ID, ErrNotFound, "service or action not found")
	}
}

func (h DefaultHandler) handlePresence(msg Message) Message {
	user, err := h.authenticate(msg)
	if err != nil {
		return ErrorResponse(msg.ID, ErrUnauthorized, "valid session token required")
	}
	if h.Presence == nil {
		return ErrorResponse(msg.ID, ErrServer, "presence service unavailable")
	}

	switch msg.Action {
	case "list":
		return SuccessResponse(msg.ID, ServicePresence, "list", map[string]interface{}{"users": h.Presence.List()})
	case "get":
		entry, ok := h.Presence.Get(user.ID)
		if !ok {
			return SuccessResponse(msg.ID, ServicePresence, "get", map[string]interface{}{"online": false})
		}
		return SuccessResponse(msg.ID, ServicePresence, "get", entry)
	default:
		return ErrorResponse(msg.ID, ErrNotFound, "service or action not found")
	}
}

func (h DefaultHandler) authenticate(msg Message) (accounts.User, error) {
	token, err := tokenFromPayload(msg.Payload)
	if err != nil {
		return accounts.User{}, err
	}
	return h.Accounts.FromToken(token)
}

func decodePayload(payload interface{}, target interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func tokenFromPayload(payload interface{}) (string, error) {
	var input tokenPayload
	if err := decodePayload(payload, &input); err != nil {
		return "", err
	}
	input.Token = strings.TrimSpace(input.Token)
	if input.Token == "" {
		return "", errors.New("missing session token")
	}
	return input.Token, nil
}
