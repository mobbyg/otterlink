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
	ID                   int64  `json:"id"`
	Name                 string `json:"name"`
	Creator              string `json:"creator"`
	Permanent            bool   `json:"permanent"`
	AllowOpsToCreateOps  bool   `json:"allow_ops_to_create_ops"`
	OriginalMod          string `json:"original_mod,omitempty"`
}

type Member struct {
	User   User   `json:"user"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

type Service struct {
	DB    *sql.DB
	limit int
	mu    sync.Mutex
}

type Hub = Service

func NewHub(db *sql.DB, limit int) *Hub {
	if limit < 1 {
		limit = 100
	}
	return &Service{DB: db, limit: limit}
}

func (h *Service) ListChannels() ([]Channel, error) {
	rows, err := h.DB.Query(`
		SELECT c.id, c.name,
			COALESCE(NULLIF(c.creator_username, ''), creator.username, ''),
			c.permanent,
			c.allow_ops_to_create_ops,
			COALESCE(NULLIF(c.original_mod_username, ''), original_mod.username, '')
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
		var channel Channel
		if err := rows.Scan(
			&channel.ID,
			&channel.Name,
			&channel.Creator,
			&channel.Permanent,
			&channel.AllowOpsToCreateOps,
			&channel.OriginalMod,
		); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func (h *Service) CreateChannel(user User, name string, permanent, allowOps, adminCreated bool) (Channel, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > 64 {
		return Channel{}, errors.New("channel name must be 1-64 characters")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var originalModID any
	var originalModUsername any
	if !adminCreated {
		originalModID = user.ID
		originalModUsername = user.Username
	}

	result, err := h.DB.Exec(`
		INSERT INTO chat_channels (
			name, creator_user_id, creator_username,
			original_mod_user_id, original_mod_username,
			permanent, allow_ops_to_create_ops
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name,
		user.ID,
		user.Username,
		originalModID,
		originalModUsername,
		permanent,
		allowOps,
	)
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
	var channel Channel
	err := h.DB.QueryRow(`
		SELECT c.id, c.name,
			COALESCE(NULLIF(c.creator_username, ''), creator.username, ''),
			c.permanent,
			c.allow_ops_to_create_ops,
			COALESCE(NULLIF(c.original_mod_username, ''), original_mod.username, '')
		FROM chat_channels c
		LEFT JOIN users creator ON creator.id = c.creator_user_id
		LEFT JOIN users original_mod ON original_mod.id = c.original_mod_user_id
		WHERE c.id = ?`, id).Scan(
		&channel.ID,
		&channel.Name,
		&channel.Creator,
		&channel.Permanent,
		&channel.AllowOpsToCreateOps,
		&channel.OriginalMod,
	)
	return channel, err
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
	var originalModID sql.NullInt64
	if err := h.DB.QueryRow(
		`SELECT original_mod_user_id FROM chat_channels WHERE id = ?`,
		channelID,
	).Scan(&originalModID); err != nil {
		return Channel{}, Member{}, nil, nil, err
	}
	if originalModID.Valid && originalModID.Int64 == user.ID {
		role = "original_mod"
	}

	var existing string
	_ = h.DB.QueryRow(
		`SELECT role FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`,
		channelID,
		user.ID,
	).Scan(&existing)
	if role != "original_mod" && (existing == "mod" || existing == "op") {
		role = existing
	}

	_, err = h.DB.Exec(`
		INSERT INTO chat_channel_members (channel_id, user_id, role, joined_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_id, user_id)
		DO UPDATE SET role = excluded.role, joined_at = CURRENT_TIMESTAMP`,
		channelID,
		user.ID,
		role,
	)
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
	return h.leaveNoLock(channelID, userID)
}

func (h *Service) LeaveAll(userID int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	rows, err := h.DB.Query(`SELECT channel_id FROM chat_channel_members WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, id := range ids {
		if err := h.leaveNoLock(id, userID); err != nil {
			return err
		}
	}
	return nil
}

func (h *Service) leaveNoLock(channelID, userID int64) error {
	if _, err := h.DB.Exec(
		`DELETE FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`,
		channelID,
		userID,
	); err != nil {
		return err
	}

	var permanent bool
	if err := h.DB.QueryRow(
		`SELECT permanent FROM chat_channels WHERE id = ?`,
		channelID,
	).Scan(&permanent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if !permanent {
		var count int
		if err := h.DB.QueryRow(
			`SELECT COUNT(*) FROM chat_channel_members WHERE channel_id = ?`,
			channelID,
		).Scan(&count); err != nil {
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
		ORDER BY CASE m.role
			WHEN 'original_mod' THEN 0
			WHEN 'mod' THEN 1
			WHEN 'op' THEN 2
			ELSE 3
		END, u.username COLLATE NOCASE`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]Member, 0)
	for rows.Next() {
		var member Member
		if err := rows.Scan(
			&member.User.ID,
			&member.User.Username,
			&member.User.DisplayName,
			&member.Role,
		); err != nil {
			return nil, err
		}
		member.Active = true
		members = append(members, member)
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
	if err := h.DB.QueryRow(
		`SELECT COUNT(*) FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`,
		channelID,
		user.ID,
	).Scan(&count); err != nil {
		return Message{}, err
	}
	if count == 0 {
		return Message{}, errors.New("you are not in this channel")
	}
	if h.isBanned(channelID, user.ID) {
		return Message{}, errors.New("you are banned from this channel")
	}

	message := Message{
		From:      user,
		Message:   text,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if _, err := h.DB.Exec(
		`INSERT INTO chat_messages (channel_id, user_id, message, created_at) VALUES (?, ?, ?, ?)`,
		channelID,
		user.ID,
		text,
		message.Timestamp,
	); err != nil {
		return Message{}, err
	}

	_, err := h.DB.Exec(`
		DELETE FROM chat_messages
		WHERE channel_id = ?
		AND id NOT IN (
			SELECT id FROM chat_messages
			WHERE channel_id = ?
			ORDER BY id DESC LIMIT ?
		)`, channelID, channelID, h.limit)
	return message, err
}

func (h *Service) Messages(channelID int64) ([]Message, error) {
	return h.messages(channelID)
}

func (h *Service) messages(channelID int64) ([]Message, error) {
	rows, err := h.DB.Query(`
		SELECT m.user_id, u.username, u.display_name, m.message, m.created_at
		FROM chat_messages m
		JOIN users u ON u.id = m.user_id
		WHERE m.channel_id = ?
		ORDER BY m.id`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]Message, 0)
	for rows.Next() {
		var message Message
		if err := rows.Scan(
			&message.From.ID,
			&message.From.Username,
			&message.From.DisplayName,
			&message.Message,
			&message.Timestamp,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (h *Service) Role(channelID, userID int64) (string, error) {
	return h.roleNoLock(channelID, userID)
}

func (h *Service) AdminSetRole(channelID, targetID int64, role string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if role != "mod" && role != "op" && role != "user" {
		return errors.New("invalid channel role")
	}

	var permanent bool
	var originalModID sql.NullInt64
	if err := h.DB.QueryRow(
		`SELECT permanent, original_mod_user_id FROM chat_channels WHERE id = ?`,
		channelID,
	).Scan(&permanent, &originalModID); err != nil {
		return err
	}
	if !permanent {
		return errors.New("only permanent channels can be managed by an administrator")
	}
	if originalModID.Valid && targetID == originalModID.Int64 {
		return errors.New("the original mod cannot be changed")
	}

	_, err := h.DB.Exec(`
		UPDATE chat_channel_members
		SET role = ?
		WHERE channel_id = ? AND user_id = ?`, role, channelID, targetID)
	return err
}

func (h *Service) SetRole(channelID, actorID, targetID int64, role string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	actorRole, err := h.roleNoLock(channelID, actorID)
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
	if err := h.DB.QueryRow(
		`SELECT original_mod_user_id FROM chat_channels WHERE id = ?`,
		channelID,
	).Scan(&originalModID); err != nil {
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
		if err := h.DB.QueryRow(
			`SELECT allow_ops_to_create_ops FROM chat_channels WHERE id = ?`,
			channelID,
		).Scan(&allowed); err != nil || !allowed {
			return errors.New("operators are not permitted to create operators in this channel")
		}
	}

	targetRole, err := h.roleNoLock(channelID, targetID)
	if err != nil {
		return errors.New("target user is not in this channel")
	}
	if actorRole == "op" && (targetRole == "op" || targetRole == "mod" || targetRole == "original_mod") {
		return errors.New("operators can only manage ordinary users")
	}
	if actorRole == "mod" && (targetRole == "mod" || targetRole == "original_mod") {
		return errors.New("only the original mod can manage mods")
	}

	_, err = h.DB.Exec(`
		UPDATE chat_channel_members
		SET role = ?
		WHERE channel_id = ? AND user_id = ?`, role, channelID, targetID)
	return err
}

func (h *Service) roleNoLock(channelID, userID int64) (string, error) {
	var role string
	err := h.DB.QueryRow(
		`SELECT role FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`,
		channelID,
		userID,
	).Scan(&role)
	return role, err
}

func (h *Service) Kick(channelID, actorID, targetID int64) error {
	return h.moderate(channelID, actorID, targetID, false)
}

func (h *Service) Ban(channelID, actorID, targetID int64) error {
	return h.moderate(channelID, actorID, targetID, true)
}

func (h *Service) moderate(channelID, actorID, targetID int64, ban bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	actorRole, err := h.roleNoLock(channelID, actorID)
	if err != nil {
		return errors.New("actor is not a member of this channel")
	}

	var originalModID sql.NullInt64
	if err := h.DB.QueryRow(
		`SELECT original_mod_user_id FROM chat_channels WHERE id = ?`,
		channelID,
	).Scan(&originalModID); err != nil {
		return err
	}
	if originalModID.Valid && targetID == originalModID.Int64 {
		return errors.New("the original mod cannot be moderated")
	}

	targetRole, err := h.roleNoLock(channelID, targetID)
	if err != nil {
		return errors.New("target user is not in this channel")
	}

	allowed := actorRole == "original_mod" ||
		(actorRole == "mod" && targetRole != "mod" && targetRole != "original_mod") ||
		(actorRole == "op" && targetRole == "user")
	if !allowed {
		return errors.New("you do not have permission to moderate that user")
	}

	if ban {
		if _, err = h.DB.Exec(
			`INSERT OR IGNORE INTO chat_channel_bans (channel_id, user_id) VALUES (?, ?)`,
			channelID,
			targetID,
		); err != nil {
			return err
		}
	}

	_, err = h.DB.Exec(
		`DELETE FROM chat_channel_members WHERE channel_id = ? AND user_id = ?`,
		channelID,
		targetID,
	)
	return err
}

func (h *Service) Unban(channelID, actorID, targetID int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	role, err := h.roleNoLock(channelID, actorID)
	if err != nil || (role != "original_mod" && role != "mod") {
		return errors.New("only a mod can unban users")
	}

	_, err = h.DB.Exec(
		`DELETE FROM chat_channel_bans WHERE channel_id = ? AND user_id = ?`,
		channelID,
		targetID,
	)
	return err
}

func (h *Service) isBanned(channelID, userID int64) bool {
	var count int
	if err := h.DB.QueryRow(
		`SELECT COUNT(*) FROM chat_channel_bans WHERE channel_id = ? AND user_id = ?`,
		channelID,
		userID,
	).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func (h *Service) DeleteChannel(channelID int64) error {
	_, err := h.DB.Exec(`DELETE FROM chat_channels WHERE id = ?`, channelID)
	return err
}
