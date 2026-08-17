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
	CreatedAt   string `json:"created_at"`
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
	err := s.DB.QueryRow(`SELECT id, username, display_name, COALESCE(email, ''), status, created_at, password_hash FROM users WHERE username = ? COLLATE NOCASE`, username).
		Scan(&id, &user.Username, &user.DisplayName, &user.Email, &user.Status, &user.CreatedAt, &hash)
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
	err := s.DB.QueryRow(`SELECT id, username, display_name, COALESCE(email, ''), status, created_at FROM users WHERE id = ?`, id).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status, &u.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s Service) FromToken(token string) (User, error) {
	if token == "" {
		return User{}, errors.New("missing session token")
	}
	h := sha256.Sum256([]byte(token))
	var u User
	var expires string
	err := s.DB.QueryRow(`SELECT u.id, u.username, u.display_name, COALESCE(u.email, ''), u.status, u.created_at, s.expires_at FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = ?`, hex.EncodeToString(h[:])).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status, &u.CreatedAt, &expires)
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
