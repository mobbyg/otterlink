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
	mu    sync.RWMutex
	online map[int64]User
}

func NewService() *Service {
	return &Service{online: make(map[int64]User)}
}

func (s *Service) Online(user accounts.User) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := User{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Status: "online", Since: time.Now().UTC().Format(time.RFC3339)}
	s.online[user.ID] = entry
	return entry
}

func (s *Service) Offline(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.online, userID)
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
