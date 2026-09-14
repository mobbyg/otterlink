package chat

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type Message struct {
	From      User   `json:"from"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type Channel struct {
	ID                       int64  `json:"id"`
	Name                     string `json:"name"`
	Creator                  string `json:"creator"`
	Permanent                bool   `json:"permanent"`
	AllowOpsToCreateOps      bool   `json:"allow_ops_to_create_ops"`
	OriginalMod              string `json:"original_mod,omitempty"`
}

type Member struct {
	User       User   `json:"user"`
	Role       string `json:"role"`
	Active     bool   `json:"active"`
}

type Service struct {
	DB    *sql.DB
	limit int
	mu    sync.Mutex
}

// Hub is retained as the chat service name because the client/server wiring
// already uses that name. Channel state that must survive a restart is stored
// in SQLite; the mutex only serializes lifecycle operations.
type Hub = Service

func NewHub(db *sql.DB, limit int) *Hub {
	if limit < 1 {
		limit = 100
	}
	return &Service{DB: db, limit: limit}
}

func (h *Service) ListChannels() ([]Channel, error) {
	rows, err := h.DB.Query(`
		SELECT c.id, c.name, COALESCE(creator.username, ''), c.permanent,
		       c.allow_ops_to_create_ops, COALESCE(original_mod.username, '')
		FROM chat_channels c
		LEFT JOIN users creator ON creator.id = c.creator_user_id
		LEFT JOIN users original_mod ON original_mod.id = c.original_mod_user_id
		ORDER BY c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := make([]Channel, 0)
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Name, &c.Creator, &c.Permanent, &c.AllowOpsToCreateOps, &c.OriginalMod); err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}
	return channels, rows.Err()
}

func (h *Service) CreateChannel(user User, name string, permanent, allowOps bool, adminCreated bool) (Channel, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 {
		return Channel{}, errors.New("channel name must be 1-64 characters")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var originalMod any
	if !adminCreated {
		originalMod = user.ID
	}
	result, err := h.DB.Exec(`
		INSERT INTO chat_channels
		(name, creator_user_id, original_mod_user_id, permanent, allow_ops_to_create_ops)
		VALUES (?, ?, ?, ?, ?)`, name, user.ID, originalMod, permanent, allowOps)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Channel{}, errors.New("a channel with that name already exists")
		}
		return Channel{}, fmt.Errorf("create channel: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Channel{}, err
	}
	return h.channel(id)
}

func (h *Service) channel(id int64) (Channel, error) {
	var c Channel
	err := h.DB.QueryRow(`
		SELECT c.id, c.name, COALESCE(creator.username, ''), c.permanent,
		       c.allow_ops_to_create_ops, COALESCE(original_mod.username, '')
		FROM chat_channels c
		LEFT JOIN users creator ON creator.id = c.creator_user_id
		LEFT JOIN users original_mod ON original_mod.id = c.original_mod_user_id
		WHERE c.id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Creator, &c.Permanent, &c.AllowOpsToCreateOps, &c.OriginalMod)
	return c, err
}

func (h *Service) Join(channelID int64, user User) (Channel, Member, []Member, []Message, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	channel, err := h.channel(channelID)
	if err != nil {
		return Channel{}, Member{}, nil, nil, err
	}
	if h.isBanned(channelID, user.ID) {
		return Channel{}, Member{}, nil, nil, errors.New("you are banned from this channel")
	}

	role := "user"
	if strings.EqualFold(channel.OriginalMod, user.Username) {
		role = "original_mod"
	}
	var existing string
	_ = h.DB.QueryRow(`SELECT role FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`, channelID, user.ID).Scan(&existing)
	if existing == "mod" || existing == "op" {
		role = existing
	}

	_, err = h.DB.Exec(`
		INSERT INTO chat_channel_members (channel_id, user_id, role, joined_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_id, user_id) DO UPDATE SET role = excluded.role, joined_at = CURRENT_TIMESTAMP`,
		channelID, user.ID, role)
	if err != nil {
		return Channel{}, Member{}, nil, nil, err
	}
	members, err := h.members(channelID)
	if err != nil {
		return Channel{}, Member{}, nil, nil, err
	}
	messages, err := h.messages(channelID)
	if err != nil {
		return Channel{}, Member{}, nil, nil, err
	}
	return channel, Member{User: user, Role: role, Active: true}, members, messages, nil
}

func (h *Service) Leave(channelID, userID int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, err := h.DB.Exec(`DELETE FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`, channelID, userID); err != nil {
		return err
	}
	var permanent bool
	if err := h.DB.QueryRow(`SELECT permanent FROM chat_channels WHERE id = ?`, channelID).Scan(&permanent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if !permanent {
		var count int
		if err := h.DB.QueryRow(`SELECT COUNT(*) FROM chat_channel_members WHERE channel_id = ?`, channelID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			_, err := h.DB.Exec(`DELETE FROM chat_channels WHERE id = ?`, channelID)
			return err
		}
	}
	return nil
}

func (h *Service) Members(channelID int64) ([]Member, error) {
	return h.members(channelID)
}

func (h *Service) members(channelID int64) ([]Member, error) {
	rows, err := h.DB.Query(`
		SELECT u.id, u.username, u.display_name, m.role
		FROM chat_channel_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.channel_id = ?
		ORDER BY CASE m.role WHEN 'original_mod' THEN 0 WHEN 'mod' THEN 1 WHEN 'op' THEN 2 ELSE 3 END,
		         u.username COLLATE NOCASE`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := make([]Member, 0)
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.User.ID, &m.User.Username, &m.User.DisplayName, &m.Role); err != nil {
			return nil, err
		}
		m.Active = true
		members = append(members, m)
	}
	return members, rows.Err()
}

func (h *Service) Publish(channelID int64, user User, text string) (Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Message{}, errors.New("message is required")
	}
	if len([]rune(text)) > 2000 {
		return Message{}, errors.New("message exceeds 2000 characters")
	}
	var count int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`, channelID, user.ID).Scan(&count); err != nil {
		return Message{}, err
	}
	if count == 0 {
		return Message{}, errors.New("you are not in this channel")
	}
	if h.isBanned(channelID, user.ID) {
		return Message{}, errors.New("you are banned from this channel")
	}
	message := Message{From: user, Message: text, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	_, err := h.DB.Exec(`INSERT INTO chat_messages (channel_id, user_id, message, created_at) VALUES (?, ?, ?, ?)`, channelID, user.ID, text, message.Timestamp)
	if err != nil {
		return Message{}, err
	}
	_, err = h.DB.Exec(`DELETE FROM chat_messages WHERE channel_id = ? AND id NOT IN (SELECT id FROM chat_messages WHERE channel_id = ? ORDER BY id DESC LIMIT ?)`, channelID, channelID, h.limit)
	return message, err
}

func (h *Service) Messages(channelID int64) ([]Message, error) {
	return h.messages(channelID)
}

func (h *Service) messages(channelID int64) ([]Message, error) {
	rows, err := h.DB.Query(`
		SELECT m.user_id, u.username, u.display_name, m.message, m.created_at
		FROM chat_messages m JOIN users u ON u.id = m.user_id
		WHERE m.channel_id = ? ORDER BY m.id`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.From.ID, &m.From.Username, &m.From.DisplayName, &m.Message, &m.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (h *Service) Role(channelID, userID int64) (string, error) {
	var role string
	err := h.DB.QueryRow(`SELECT role FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`, channelID, userID).Scan(&role)
	return role, err
}

func (h *Service) SetRole(channelID, actorID, targetID int64, role string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	actorRole, err := h.Role(channelID, actorID)
	if err != nil {
		return errors.New("actor is not a member of this channel")
	}
	if role != "mod" && role != "op" && role != "user" {
		return errors.New("invalid channel role")
	}
	if targetID == actorID {
		return errors.New("you cannot change your own role")
	}
	var originalModID sql.NullInt64
	if err := h.DB.QueryRow(`SELECT original_mod_user_id FROM chat_channels WHERE id = ?`, channelID).Scan(&originalModID); err != nil {
		return err
	}
	if originalModID.Valid && targetID == originalModID.Int64 {
		return errors.New("the original mod cannot be demoted")
	}
	if role == "mod" && actorRole != "original_mod" {
		return errors.New("only the original mod can create mods")
	}
	if role == "op" && actorRole != "original_mod" && actorRole != "mod" && actorRole != "op" {
		return errors.New("only a mod or permitted operator can create operators")
	}
	if role == "op" && actorRole == "op" {
		var allowed bool
		if err := h.DB.QueryRow(`SELECT allow_ops_to_create_ops FROM chat_channels WHERE id = ?`, channelID).Scan(&allowed); err != nil || !allowed {
			return errors.New("operators are not permitted to create operators in this channel")
		}
	}
	var targetRole string
	if err := h.DB.QueryRow(`SELECT role FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`, channelID, targetID).Scan(&targetRole); err != nil {
		return errors.New("target user is not in this channel")
	}
	if actorRole == "op" && (targetRole == "op" || targetRole == "mod" || targetRole == "original_mod") {
		return errors.New("operators can only manage ordinary users")
	}
	if actorRole == "mod" && (targetRole == "mod" || targetRole == "original_mod") && role != "op" {
		return errors.New("only the original mod can manage mods")
	}
	_, err = h.DB.Exec(`UPDATE chat_channel_members SET role = ? WHERE channel_id = ? AND user_id = ?`, role, channelID, targetID)
	return err
}

func (h *Service) Kick(channelID, actorID, targetID int64) error {
	return h.moderate(channelID, actorID, targetID, false)
}

func (h *Service) Ban(channelID, actorID, targetID int64) error {
	if err := h.moderate(channelID, actorID, targetID, true); err != nil {
		return err
	}
	return nil
}

func (h *Service) moderate(channelID, actorID, targetID int64, ban bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	actorRole, err := h.Role(channelID, actorID)
	if err != nil {
		return errors.New("actor is not a member of this channel")
	}
	var originalModID sql.NullInt64
	if err := h.DB.QueryRow(`SELECT original_mod_user_id FROM chat_channels WHERE id = ?`, channelID).Scan(&originalModID); err != nil {
		return err
	}
	if originalModID.Valid && targetID == originalModID.Int64 {
		return errors.New("the original mod cannot be moderated")
	}
	var targetRole string
	if err := h.DB.QueryRow(`SELECT role FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`, channelID, targetID).Scan(&targetRole); err != nil {
		return errors.New("target user is not in this channel")
	}
	allowed := actorRole == "original_mod" || (actorRole == "mod" && targetRole != "mod" && targetRole != "original_mod") || (actorRole == "op" && targetRole == "user")
	if !allowed {
		return errors.New("you do not have permission to moderate that user")
	}
	if ban {
		_, err = h.DB.Exec(`INSERT OR IGNORE INTO chat_channel_bans (channel_id, user_id) VALUES (?, ?)`, channelID, targetID)
	}
	if err == nil {
		_, err = h.DB.Exec(`DELETE FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`, channelID, targetID)
	}
	return err
}

func (h *Service) Unban(channelID, actorID, targetID int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	role, err := h.Role(channelID, actorID)
	if err != nil || (role != "original_mod" && role != "mod") {
		return errors.New("only a mod can unban users")
	}
	_, err = h.DB.Exec(`DELETE FROM chat_channel_bans WHERE channel_id = ? AND user_id = ?`, channelID, targetID)
	return err
}

func (h *Service) isBanned(channelID, userID int64) bool {
	var count int
	if err := h.DB.QueryRow(`SELECT COUNT(*) FROM chat_channel_bans WHERE channel_id = ? AND user_id = ?`, channelID, userID).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func (h *Service) DeleteChannel(channelID int64) error {
	_, err := h.DB.Exec(`DELETE FROM chat_channels WHERE id = ?`, channelID)
	return err
}
