package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

type adminUser struct {
	accounts.User
	Online       bool   `json:"online"`
	SessionCount int    `json:"session_count"`
	LastActivity string `json:"last_activity,omitempty"`
}

func (s *Server) adminUser(r *http.Request) (accounts.User, bool, bool) {
	user, authenticated := s.user(r)
	if !authenticated {
		return accounts.User{}, false, false
	}
	return user, user.Role == "admin", true
}

func (s *Server) adminIndex(w http.ResponseWriter, _ *http.Request) {
	data, err := staticFiles.ReadFile("static/admin.html")
	if err != nil {
		http.Error(w, "admin client unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (accounts.User, bool) {
	user, admin, authenticated := s.adminUser(r)
	if !authenticated {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return accounts.User{}, false
	}
	if !admin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return accounts.User{}, false
	}
	return user, true
}

func (s *Server) adminUserInfo(user accounts.User) (adminUser, error) {
	info := adminUser{User: user}
	if s.Presence != nil {
		for _, present := range s.Presence.List() {
			if strings.EqualFold(present.Username, user.Username) {
				info.Online = true
				break
			}
		}
	}
	summary, err := s.Accounts.SessionSummary(user.ID)
	if err != nil {
		return adminUser{}, err
	}
	info.SessionCount = summary.Count
	info.LastActivity = summary.LastActivity
	return info, nil
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}

	users, err := s.Accounts.List()
	if err != nil {
		http.Error(w, "unable to list users", http.StatusInternalServerError)
		return
	}

	result := make([]adminUser, 0, len(users))
	for _, user := range users {
		info, err := s.adminUserInfo(user)
		if err != nil {
			http.Error(w, "unable to load user sessions", http.StatusInternalServerError)
			return
		}
		result = append(result, info)
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": result})
}

type adminUserUpdateRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Status      string `json:"status"`
	Role        string `json:"role"`
}

type adminPasswordResetRequest struct {
	Password string `json:"password"`
}

type adminDeleteRequest struct {
	Password string `json:"password"`
}

func (s *Server) adminUserDetail(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	user, err := s.Accounts.GetByUsername(username)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unable to load user", http.StatusInternalServerError)
		return
	}
	info, err := s.adminUserInfo(user)
	if err != nil {
		http.Error(w, "unable to load user sessions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) adminUserUpdate(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	username := strings.TrimSpace(r.PathValue("username"))
	user, err := s.Accounts.GetByUsername(username)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unable to load user", http.StatusInternalServerError)
		return
	}

	var req adminUserUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if user.ID == admin.ID && (req.Role != "admin" || req.Status != "active") {
		http.Error(w, "you cannot disable or demote your own admin account", http.StatusBadRequest)
		return
	}

	updated, err := s.Accounts.UpdateAdminUser(user.ID, req.DisplayName, req.Email, req.Status, req.Role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) adminUserPasswordReset(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	username := strings.TrimSpace(r.PathValue("username"))
	user, err := s.Accounts.GetByUsername(username)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unable to load user", http.StatusInternalServerError)
		return
	}

	var req adminPasswordResetRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if user.ID == admin.ID {
		http.Error(w, "use the account password flow to change your own password", http.StatusBadRequest)
		return
	}
	if err := s.Accounts.ResetPassword(user.ID, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) adminUserSessionsRevoke(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	username := strings.TrimSpace(r.PathValue("username"))
	user, err := s.Accounts.GetByUsername(username)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unable to load user", http.StatusInternalServerError)
		return
	}
	if user.ID == admin.ID {
		http.Error(w, "you cannot revoke your own admin session", http.StatusBadRequest)
		return
	}
	if err := s.Accounts.RevokeSessions(user.ID); err != nil {
		http.Error(w, "unable to revoke sessions", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adminUserDelete(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	username := strings.TrimSpace(r.PathValue("username"))
	user, err := s.Accounts.GetByUsername(username)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unable to load user", http.StatusInternalServerError)
		return
	}
	if user.ID == admin.ID {
		http.Error(w, "you cannot delete your own admin account", http.StatusBadRequest)
		return
	}

	var req adminDeleteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !s.Accounts.VerifyPassword(admin.ID, req.Password) {
		http.Error(w, "admin password is incorrect", http.StatusUnauthorized)
		return
	}
	if user.Role == "admin" {
		users, err := s.Accounts.List()
		if err != nil {
			http.Error(w, "unable to verify administrator count", http.StatusInternalServerError)
			return
		}
		adminCount := 0
		for _, candidate := range users {
			if candidate.Role == "admin" && candidate.Status == "active" {
				adminCount++
			}
		}
		if adminCount <= 1 {
			http.Error(w, "cannot delete the last active administrator", http.StatusBadRequest)
			return
		}
	}

	if err := s.Accounts.Delete(user.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "unable to delete user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
