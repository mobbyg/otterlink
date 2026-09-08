package chat

import (
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

type Hub struct {
	mu      sync.RWMutex
	messages []Message
	limit   int
}

func NewHub(limit int) *Hub {
	if limit < 1 { limit = 100 }
	return &Hub{limit: limit}
}

func (h *Hub) Publish(user User, text string) Message {
	message := Message{From: user, Message: text, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	h.mu.Lock()
	h.messages = append(h.messages, message)
	if len(h.messages) > h.limit { h.messages = append([]Message(nil), h.messages[len(h.messages)-h.limit:]...) }
	h.mu.Unlock()
	return message
}

func (h *Hub) List() []Message {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.messages) == 0 {
		return []Message{}
	}
	return append([]Message(nil), h.messages...)
}
