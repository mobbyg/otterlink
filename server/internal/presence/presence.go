package presence

import (
	"sync"
	"time"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	Since       string `json:"since"`
}

type Service struct {
	mu          sync.RWMutex
	online      map[int64]User
	connections map[int64]map[uint64]struct{}
}

func NewService() *Service {
	return &Service{online: make(map[int64]User), connections: make(map[int64]map[uint64]struct{})}
}

func (s *Service) Online(user accounts.User) User {
	entry, _ := s.OnlineConnection(user, 0)
	return entry
}

// OnlineConnection associates a connection with a user. The boolean is true
// only when this connection makes the user transition from offline to online.
func (s *Service) OnlineConnection(user accounts.User, connectionID uint64) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.online[user.ID]
	if !exists {
		entry = User{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Status: "online", Since: time.Now().UTC().Format(time.RFC3339)}
		s.online[user.ID] = entry
	}
	if connectionID != 0 {
		if s.connections[user.ID] == nil {
			s.connections[user.ID] = make(map[uint64]struct{})
		}
		s.connections[user.ID][connectionID] = struct{}{}
	}
	return entry, !exists
}

func (s *Service) Offline(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.online, userID)
	delete(s.connections, userID)
}

// OfflineConnection removes one connection. The boolean is true only when
// that was the user's final active connection.
func (s *Service) OfflineConnection(userID int64, connectionID uint64) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.online[userID]
	if !exists {
		return User{}, false
	}
	if connectionID != 0 {
		if conns := s.connections[userID]; conns != nil {
			delete(conns, connectionID)
			if len(conns) > 0 {
				return entry, false
			}
		}
	}
	delete(s.online, userID)
	delete(s.connections, userID)
	return entry, true
}

func (s *Service) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]User, 0, len(s.online))
	for _, user := range s.online {
		users = append(users, user)
	}
	return users
}

func (s *Service) Get(userID int64) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.online[userID]
	return user, ok
}
