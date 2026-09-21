package dm

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID int64 `json:"id"`
	Username string `json:"username"`
	DisplayName string `json:"display_name"`
}

type Message struct {
	ID int64 `json:"id"`
	From User `json:"from"`
	To User `json:"to"`
	Message string `json:"message"`
	Timestamp string `json:"timestamp"`
	Read bool `json:"read"`
}

type Conversation struct {
	With User `json:"with"`
	Messages []Message `json:"messages"`
	UnreadCount int `json:"unread_count"`
}

type Unread struct {
	Username string `json:"username"`
	Count int `json:"count"`
}

type Service struct { DB *sql.DB; Limit int }

func (s Service) Send(senderID int64, recipientUsername, text string) (Message, error) {
	text = strings.TrimSpace(text)
	if text == "" { return Message{}, errors.New("message is required") }
	if len([]rune(text)) > 2000 { return Message{}, errors.New("message exceeds 2000 characters") }

	var recipient User
	if err := s.DB.QueryRow(`SELECT id, username, display_name FROM users WHERE username = ? COLLATE NOCASE`, recipientUsername).
		Scan(&recipient.ID, &recipient.Username, &recipient.DisplayName); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return Message{}, errors.New("user not found") }
		return Message{}, err
	}
	if recipient.ID == senderID { return Message{}, errors.New("cannot send a private message to yourself") }

	var sender User
	if err := s.DB.QueryRow(`SELECT id, username, display_name FROM users WHERE id = ?`, senderID).
		Scan(&sender.ID, &sender.Username, &sender.DisplayName); err != nil { return Message{}, err }

	ts := time.Now().UTC().Format(time.RFC3339)
	result, err := s.DB.Exec(`INSERT INTO direct_messages (sender_id, recipient_id, message, created_at) VALUES (?, ?, ?, ?)`, senderID, recipient.ID, text, ts)
	if err != nil { return Message{}, fmt.Errorf("send private message: %w", err) }
	id, err := result.LastInsertId()
	if err != nil { return Message{}, err }
	return Message{ID:id, From:sender, To:recipient, Message:text, Timestamp:ts}, nil
}

func (s Service) Conversation(userID int64, username string) (Conversation, error) {
	var other User
	if err := s.DB.QueryRow(`SELECT id, username, display_name FROM users WHERE username = ? COLLATE NOCASE`, strings.TrimSpace(username)).
		Scan(&other.ID, &other.Username, &other.DisplayName); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return Conversation{}, errors.New("user not found") }
		return Conversation{}, err
	}
	rows, err := s.DB.Query(`
		SELECT m.id, s.id, s.username, s.display_name, r.id, r.username, r.display_name,
		       m.message, m.created_at, m.read_at
		FROM direct_messages m
		JOIN users s ON s.id = m.sender_id
		JOIN users r ON r.id = m.recipient_id
		WHERE (m.sender_id = ? AND m.recipient_id = ?) OR (m.sender_id = ? AND m.recipient_id = ?)
		ORDER BY m.id DESC LIMIT ?`, userID, other.ID, other.ID, userID, s.limit())
	if err != nil { return Conversation{}, err }
	defer rows.Close()

	messages := make([]Message, 0)
	for rows.Next() {
		var m Message
		var readAt sql.NullString
		if err := rows.Scan(&m.ID, &m.From.ID, &m.From.Username, &m.From.DisplayName,
			&m.To.ID, &m.To.Username, &m.To.DisplayName, &m.Message, &m.Timestamp, &readAt); err != nil { return Conversation{}, err }
		m.Read = readAt.Valid
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil { return Conversation{}, err }
	for i,j := 0,len(messages)-1; i<j; i,j=i+1,j-1 { messages[i],messages[j]=messages[j],messages[i] }

	var unread int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM direct_messages WHERE sender_id = ? AND recipient_id = ? AND read_at IS NULL`, other.ID, userID).Scan(&unread); err != nil { return Conversation{}, err }
	return Conversation{With:other, Messages:messages, UnreadCount:unread}, nil
}

func (s Service) MarkRead(userID int64, username string) error {
	var otherID int64
	if err := s.DB.QueryRow(`SELECT id FROM users WHERE username = ? COLLATE NOCASE`, strings.TrimSpace(username)).Scan(&otherID); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return errors.New("user not found") }
		return err
	}
	_, err := s.DB.Exec(`UPDATE direct_messages SET read_at = ? WHERE sender_id = ? AND recipient_id = ? AND read_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), otherID, userID)
	return err
}

func (s Service) Unread(userID int64) ([]Unread, error) {
	rows, err := s.DB.Query(`
		SELECT u.username, COUNT(*) FROM direct_messages m JOIN users u ON u.id = m.sender_id
		WHERE m.recipient_id = ? AND m.read_at IS NULL
		GROUP BY m.sender_id, u.username ORDER BY u.username COLLATE NOCASE`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	result := make([]Unread, 0)
	for rows.Next() {
		var item Unread
		if err := rows.Scan(&item.Username, &item.Count); err != nil { return nil, err }
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s Service) limit() int { if s.Limit < 1 { return 100 }; return s.Limit }
