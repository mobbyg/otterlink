package presence

import (
	"testing"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

func TestConnectionBoundPresence(t *testing.T) {
	s := NewService()
	user := accounts.User{ID: 42, Username: "otter", DisplayName: "Otter"}

	if _, online := s.OnlineConnection(user, 1); !online { t.Fatal("expected first connection to make user online") }
	if _, online := s.OnlineConnection(user, 2); online { t.Fatal("expected second connection not to create another online transition") }
	if _, offline := s.OfflineConnection(user.ID, 1); offline { t.Fatal("expected user to remain online with second connection") }
	if _, ok := s.Get(user.ID); !ok { t.Fatal("expected user to remain online") }
	if _, offline := s.OfflineConnection(user.ID, 2); !offline { t.Fatal("expected final connection to make user offline") }
	if _, ok := s.Get(user.ID); ok { t.Fatal("expected user to be offline") }
}
