package protocol

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

type DefaultHandler struct {
	Accounts accounts.Service
}

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenPayload struct {
	Token string `json:"token"`
}

func (h DefaultHandler) Handle(_ context.Context, msg Message) Message {
	if msg.Service != ServiceSession {
		return ErrorResponse(msg.ID, ErrNotFound, "service or action not found")
	}

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
		if _, err := h.Accounts.FromToken(token); err != nil {
			return ErrorResponse(msg.ID, ErrUnauthorized, "invalid session")
		}
		if err := h.Accounts.Logout(token); err != nil {
			return ErrorResponse(msg.ID, ErrServer, "unable to end session")
		}
		return SuccessResponse(msg.ID, ServiceSession, "logout", map[string]string{"status": "ok"})
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
		return "", ErrUnauthorized
	}
	return input.Token, nil
}
