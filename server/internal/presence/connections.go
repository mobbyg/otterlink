package presence

import (
	"sync"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

var connectionState = struct {
	sync.Mutex
	users map[*Service]map[int64]map[uint64]struct{}
}{users: make(map[*Service]map[int64]map[uint64]struct{})}

// OnlineConnection binds a presence record to a live protocol connection.
// It reports true only when the user transitions from offline to online.
func (s *Service) OnlineConnection(user accounts.User, connectionID uint64) (User, bool) {
	entry, alreadyOnline := s.Get(user.ID)
	if !alreadyOnline {
		entry = s.Online(user)
	}

	connectionState.Lock()
	defer connectionState.Unlock()
	byUser := connectionState.users[s]
	if byUser == nil {
		byUser = make(map[int64]map[uint64]struct{})
		connectionState.users[s] = byUser
	}
	byConnection := byUser[user.ID]
	if byConnection == nil {
		byConnection = make(map[uint64]struct{})
		byUser[user.ID] = byConnection
	}
	wasConnected := len(byConnection) > 0
	byConnection[connectionID] = struct{}{}
	return entry, !alreadyOnline && !wasConnected
}

// OfflineConnection removes one connection and reports true when it was the
// user's final connection and the user became offline.
func (s *Service) OfflineConnection(userID int64, connectionID uint64) (User, bool) {
	connectionState.Lock()
	byUser := connectionState.users[s]
	byConnection := byUser[userID]
	delete(byConnection, connectionID)
	last := len(byConnection) == 0
	if last {
		delete(byUser, userID)
	}
	connectionState.Unlock()

	entry, online := s.Get(userID)
	if !online || !last {
		return entry, false
	}
	s.Offline(userID)
	return entry, true
}
