package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

type AuthAPI struct {
	Accounts accounts.Service
}

type registerRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	User  accounts.User `json:"user"`
	Token string        `json:"token"`
}

func (a AuthAPI) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := a.Accounts.Register(req.Username, req.DisplayName, req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (a AuthAPI) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, token, err := a.Accounts.Authenticate(req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{User: user, Token: token})
}

func (a AuthAPI) Logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token == "" {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	if err := a.Accounts.Logout(token); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a AuthAPI) Me(w http.ResponseWriter, r *http.Request) {
	user, err := a.Accounts.FromToken(bearerToken(r))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func bearerToken(r *http.Request) string {
	value := r.Header.Get("Authorization")
	if !strings.HasPrefix(value, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
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
