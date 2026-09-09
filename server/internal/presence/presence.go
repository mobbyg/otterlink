package presence

import (
	"sync"
	"time"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

const httpPresenceTimeout = 15 * time.Second

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	Since       string `json:"since"`
}

type Service struct {
	mu              sync.RWMutex
	online           map[int64]User
	connections     map[int64]map[uint64]struct{}
	httpConnections map[uint64]int64
	httpLastSeen    map[uint64]time.Time
}

func NewService() *Service {
	return &Service{
		online:           make(map[int64]User),
		connections:     make(map[int64]map[uint64]struct{}),
		httpConnections: make(map[uint64]int64),
		httpLastSeen:    make(map[uint64]time.Time),
	}
}

func (s *Service) Online(user accounts.User) User {
	entry, _ := s.OnlineConnection(user, 0)
	return entry
}

// OnlineConnection associates a persistent protocol connection with a user.
// HTTP clients should use OnlineHTTP instead so their request/response
// activity can expire naturally when the client disappears.
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

// OnlineHTTP records activity for a request/response client. HTTP presence is
// deliberately separate from persistent protocol connections so it can expire
// when a browser or native client disappears without logging out.
func (s *Service) OnlineHTTP(user accounts.User, connectionID uint64) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.pruneHTTP(now)

	entry, exists := s.online[user.ID]
	if !exists {
		entry = User{ID: user.ID, Username: user.Username, DisplayName: user.DisplayName, Status: "online", Since: now.UTC().Format(time.RFC3339)}
		s.online[user.ID] = entry
	}
	if connectionID != 0 {
		s.httpConnections[connectionID] = user.ID
		s.httpLastSeen[connectionID] = now
	}
	return entry, !exists
}

func (s *Service) Offline(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.online, userID)
	delete(s.connections, userID)
	for connectionID, id := range s.httpConnections {
		if id == userID {
			delete(s.httpConnections, connectionID)
			delete(s.httpLastSeen, connectionID)
		}
	}
}

// OfflineConnection removes one persistent protocol connection. HTTP clients
// should use OfflineHTTP instead.
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

func (s *Service) OfflineHTTP(userID int64, connectionID uint64) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.online[userID]
	if !exists {
		return User{}, false
	}

	if connectionID != 0 {
		delete(s.httpConnections, connectionID)
		delete(s.httpLastSeen, connectionID)
	}

	if s.hasPersistentConnection(userID) || s.hasHTTPConnection(userID) {
		return entry, false
	}

	delete(s.online, userID)
	return entry, true
}

func (s *Service) List() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneHTTP(time.Now())

	users := make([]User, 0, len(s.online))
	for _, user := range s.online {
		users = append(users, user)
	}
	return users
}

func (s *Service) Get(userID int64) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneHTTP(time.Now())
	user, ok := s.online[userID]
	return user, ok
}

func (s *Service) hasPersistentConnection(userID int64) bool {
	return len(s.connections[userID]) > 0
}

func (s *Service) hasHTTPConnection(userID int64) bool {
	for _, id := range s.httpConnections {
		if id == userID {
			return true
		}
	}
	return false
}

func (s *Service) pruneHTTP(now time.Time) {
	cutoff := now.Add(-httpPresenceTimeout)
	for connectionID, lastSeen := range s.httpLastSeen {
		if lastSeen.After(cutoff) {
			continue
		}

		userID, ok := s.httpConnections[connectionID]
		delete(s.httpLastSeen, connectionID)
		delete(s.httpConnections, connectionID)
		if !ok || s.hasPersistentConnection(userID) || s.hasHTTPConnection(userID) {
			continue
		}
		delete(s.online, userID)
	}
}
