package dm

import (
	"database/sql"
	"testing"
	_ "modernc.org/sqlite"
	"github.com/mobbyg/otterlink/server/internal/db"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil { t.Fatal(err) }
	if err := db.Initialize(database); err != nil { t.Fatal(err) }
	_, err = database.Exec(`INSERT INTO users (username, display_name, password_hash) VALUES ('one', 'One', 'x'), ('two', 'Two', 'x')`)
	if err != nil { t.Fatal(err) }
	t.Cleanup(func(){ database.Close() })
	return database
}

func TestSendConversationAndRead(t *testing.T) {
	database := testDB(t)
	s := Service{DB:database, Limit:100}
	if _, err := s.Send(1, "two", "hello"); err != nil { t.Fatal(err) }
	conversation, err := s.Conversation(2, "one")
	if err != nil { t.Fatal(err) }
	if conversation.UnreadCount != 1 || len(conversation.Messages) != 1 { t.Fatalf("unexpected conversation: %+v", conversation) }
	if err := s.MarkRead(2, "one"); err != nil { t.Fatal(err) }
	conversation, err = s.Conversation(2, "one")
	if err != nil { t.Fatal(err) }
	if conversation.UnreadCount != 0 || !conversation.Messages[0].Read { t.Fatalf("message was not marked read: %+v", conversation.Messages) }
}
