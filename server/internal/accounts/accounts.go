package accounts

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mobbyg/otterlink/server/internal/auth"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email,omitempty"`
	Status      string `json:"status"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
}

type SessionSummary struct {
	Count        int    `json:"count"`
	LastActivity string `json:"last_activity,omitempty"`
}

type AuditEvent struct {
	ID            int64  `json:"id"`
	ActorUsername string `json:"actor_username"`
	Action        string `json:"action"`
	TargetUsername string `json:"target_username,omitempty"`
	Result        string `json:"result"`
	Details       string `json:"details,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type Service struct {
	DB *sql.DB
}

func (s Service) Register(username, displayName, email, password string) (User, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	email = strings.TrimSpace(email)
	if !usernamePattern.MatchString(username) {
		return User{}, errors.New("username must be 3-32 characters using letters, numbers, or underscore")
	}
	if displayName == "" {
		displayName = username
	}
	if len(password) < 12 {
		return User{}, errors.New("password must be at least 12 characters")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return User{}, err
	}

	result, err := s.DB.Exec(`INSERT INTO users (username, display_name, email, password_hash) VALUES (?, ?, NULLIF(?, ''), ?)`, username, displayName, email, hash)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return s.Get(id)
}

func (s Service) Authenticate(username, password string) (User, string, error) {
	var id int64
	var hash string
	var user User
	err := s.DB.QueryRow(`SELECT id, username, display_name, COALESCE(email, ''), status, role, created_at, password_hash FROM users WHERE username = ? COLLATE NOCASE`, username).
		Scan(&id, &user.Username, &user.DisplayName, &user.Email, &user.Status, &user.Role, &user.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) || !auth.VerifyPassword(password, hash) {
		return User{}, "", errors.New("invalid username or password")
	}
	user.ID = id
	if user.Status != "active" {
		return User{}, "", errors.New("account is not active")
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return User{}, "", fmt.Errorf("generate session token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	tokenHash := sha256.Sum256([]byte(token))
	expires := time.Now().UTC().Add(30 * 24 * time.Hour)
	_, err = s.DB.Exec(`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)`, user.ID, hex.EncodeToString(tokenHash[:]), expires.Format(time.RFC3339))
	if err != nil {
		return User{}, "", fmt.Errorf("create session: %w", err)
	}
	return user, token, nil
}

func (s Service) Get(id int64) (User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, username, display_name, COALESCE(email, ''), status, role, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status, &u.Role, &u.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s Service) GetByUsername(username string) (User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, username, display_name, COALESCE(email, ''), status, role, created_at FROM users WHERE username = ? COLLATE NOCASE`, strings.TrimSpace(username)).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status, &u.Role, &u.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s Service) List() ([]User, error) {
	rows, err := s.DB.Query(`SELECT id, username, display_name, COALESCE(email, ''), status, role, created_at FROM users ORDER BY username COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s Service) SessionSummary(userID int64) (SessionSummary, error) {
	var summary SessionSummary
	var lastActivity sql.NullString
	if err := s.DB.QueryRow(`SELECT COUNT(*), MAX(last_seen) FROM sessions WHERE user_id = ? AND expires_at > CURRENT_TIMESTAMP`, userID).Scan(&summary.Count, &lastActivity); err != nil {
		return SessionSummary{}, err
	}
	if lastActivity.Valid {
		summary.LastActivity = lastActivity.String
	}
	return summary, nil
}

func (s Service) RevokeSessions(userID int64) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return nil
}

func (s Service) LogAudit(actor User, action, targetUsername, result, details string, targetID *int64) error {
	_, err := s.DB.Exec(`INSERT INTO audit_log (actor_user_id, actor_username, action, target_user_id, target_username, result, details) VALUES (?, ?, ?, ?, ?, ?, ?)`, actor.ID, actor.Username, action, targetID, strings.TrimSpace(targetUsername), result, strings.TrimSpace(details))
	if err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}
	return nil
}

func (s Service) ListAudit(limit int) ([]AuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.DB.Query(`SELECT id, actor_username, action, COALESCE(target_username, ''), result, COALESCE(details, ''), created_at FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]AuditEvent, 0)
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(&event.ID, &event.ActorUsername, &event.Action, &event.TargetUsername, &event.Result, &event.Details, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (s Service) EnsureAdmin(username string) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, nil
	}
	result, err := s.DB.Exec(`UPDATE users SET role = 'admin', updated_at = CURRENT_TIMESTAMP WHERE username = ? COLLATE NOCASE`, username)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s Service) VerifyPassword(userID int64, password string) bool {
	var hash string
	if err := s.DB.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&hash); err != nil {
		return false
	}
	return auth.VerifyPassword(password, hash)
}

func (s Service) UpdateAdminUser(id int64, displayName, email, status, role string) (User, error) {
	displayName = strings.TrimSpace(displayName)
	email = strings.TrimSpace(email)
	status = strings.TrimSpace(status)
	role = strings.TrimSpace(role)
	if displayName == "" {
		return User{}, errors.New("display name is required")
	}
	if status != "active" && status != "disabled" {
		return User{}, errors.New("status must be active or disabled")
	}
	if role != "user" && role != "admin" {
		return User{}, errors.New("role must be user or admin")
	}
	if _, err := s.DB.Exec(`UPDATE users SET display_name = ?, email = NULLIF(?, ''), status = ?, role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, displayName, email, status, role, id); err != nil {
		return User{}, fmt.Errorf("update user: %w", err)
	}
	return s.Get(id)
}

func (s Service) ResetPassword(id int64, password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	result, err := s.DB.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, hash, id)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	_, err = s.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, id)
	return err
}

func (s Service) Delete(id int64) error {
	result, err := s.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Service) FromToken(token string) (User, error) {
	if token == "" {
		return User{}, errors.New("missing session token")
	}
	h := sha256.Sum256([]byte(token))
	var u User
	var expires string
	err := s.DB.QueryRow(`SELECT u.id, u.username, u.display_name, COALESCE(u.email, ''), u.status, u.role, u.created_at, s.expires_at FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = ?`, hex.EncodeToString(h[:])).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status, &u.Role, &u.CreatedAt, &expires)
	if err != nil {
		return User{}, errors.New("invalid session")
	}
	when, err := time.Parse(time.RFC3339, expires)
	if err != nil || !when.After(time.Now().UTC()) || u.Status != "active" {
		return User{}, errors.New("session expired or account inactive")
	}
	_, _ = s.DB.Exec(`UPDATE sessions SET last_seen = CURRENT_TIMESTAMP WHERE token_hash = ?`, hex.EncodeToString(h[:]))
	return u, nil
}

func (s Service) Logout(token string) error {
	h := sha256.Sum256([]byte(token))
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hex.EncodeToString(h[:]))
	return err
}
