package chat

import "testing"

func TestHubKeepsMostRecentMessages(t *testing.T) {
	h := NewHub(2)
	u := User{ID: 1, Username: "alice", DisplayName: "Alice"}
	h.Publish(u, "one")
	h.Publish(u, "two")
	h.Publish(u, "three")
	messages := h.List()
	if len(messages) != 2 { t.Fatalf("got %d messages, want 2", len(messages)) }
	if messages[0].Message != "two" || messages[1].Message != "three" { t.Fatalf("unexpected messages: %#v", messages) }
}
