package presence

import (
	"testing"

	"github.com/mobbyg/otterlink/server/internal/accounts"
)

func TestOnlineListAndOffline(t *testing.T) {
	s := NewService()
	user := accounts.User{ID: 1, Username: "testotter", DisplayName: "Test Otter", Status: "active"}

	entry := s.Online(user)
	if entry.Status != "online" {
		t.Fatalf("expected online status, got %q", entry.Status)
	}
	if len(s.List()) != 1 {
		t.Fatalf("expected one online user")
	}
	if _, ok := s.Get(user.ID); !ok {
		t.Fatalf("expected user to be online")
	}

	s.Offline(user.ID)
	if len(s.List()) != 0 {
		t.Fatalf("expected no online users after offline")
	}
}
