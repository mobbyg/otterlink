package chat

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT UNIQUE, display_name TEXT);
		CREATE TABLE chat_channels (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE, creator_user_id INTEGER, original_mod_user_id INTEGER, permanent INTEGER, allow_ops_to_create_ops INTEGER);
		CREATE TABLE chat_channel_members (channel_id INTEGER, user_id INTEGER, role TEXT, joined_at TEXT, PRIMARY KEY(channel_id,user_id));
		CREATE TABLE chat_channel_bans (channel_id INTEGER, user_id INTEGER, PRIMARY KEY(channel_id,user_id));
		CREATE TABLE chat_messages (id INTEGER PRIMARY KEY AUTOINCREMENT, channel_id INTEGER, user_id INTEGER, message TEXT, created_at TEXT);
	`)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func TestCreateJoinAndDeleteTemporaryChannel(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO users(id, username, display_name) VALUES (1, 'alice', 'Alice'), (2, 'bob', 'Bob')`)
	h := NewHub(db, 2)
	alice := User{ID: 1, Username: "alice", DisplayName: "Alice"}
	bob := User{ID: 2, Username: "bob", DisplayName: "Bob"}

	channel, _, _, _, err := h.JoinChannelForTest(alice, "Lobby")
	if err != nil {
		t.Fatal(err)
	}
	if channel.OriginalMod != "alice" {
		t.Fatalf("original mod = %q, want alice", channel.OriginalMod)
	}
	if _, _, _, _, err := h.Join(channel.ID, bob); err != nil {
		t.Fatal(err)
	}
	if err := h.Leave(channel.ID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := h.channel(channel.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.Leave(channel.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := h.channel(channel.ID); err == nil {
		t.Fatal("temporary channel still exists after last user left")
	}
}

func (h *Service) JoinChannelForTest(user User, name string) (Channel, Member, []Member, []Message, error) {
	channel, err := h.CreateChannel(user, name, false, false, false)
	if err != nil {
		return Channel{}, Member{}, nil, nil, err
	}
	return h.Join(channel.ID, user)
}

func TestOriginalModCanCreateModsButDelegatedModCannot(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO users(id, username, display_name) VALUES (1, 'alice', 'Alice'), (2, 'bob', 'Bob'), (3, 'carol', 'Carol')`)
	h := NewHub(db, 2)
	alice := User{ID: 1, Username: "alice", DisplayName: "Alice"}
	bob := User{ID: 2, Username: "bob", DisplayName: "Bob"}
	carol := User{ID: 3, Username: "carol", DisplayName: "Carol"}
	channel, _, _, _, err := h.JoinChannelForTest(alice, "Mods")
	if err != nil { t.Fatal(err) }
	if _, _, _, _, err = h.Join(channel.ID, bob); err != nil { t.Fatal(err) }
	if _, _, _, _, err = h.Join(channel.ID, carol); err != nil { t.Fatal(err) }
	if err := h.SetRole(channel.ID, alice.ID, bob.ID, "mod"); err != nil { t.Fatal(err) }
	if err := h.SetRole(channel.ID, bob.ID, carol.ID, "mod"); err == nil { t.Fatal("delegated mod was allowed to create a mod") }
}

func TestOperatorCreationRequiresChannelFlag(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO users(id, username, display_name) VALUES (1, 'alice', 'Alice'), (2, 'bob', 'Bob')`)
	h := NewHub(db, 2)
	alice := User{ID: 1, Username: "alice", DisplayName: "Alice"}
	bob := User{ID: 2, Username: "bob", DisplayName: "Bob"}
	channel, _, _, _, err := h.JoinChannelForTest(alice, "Ops")
	if err != nil { t.Fatal(err) }
	if _, _, _, _, err = h.Join(channel.ID, bob); err != nil { t.Fatal(err) }
	if err := h.SetRole(channel.ID, alice.ID, bob.ID, "op"); err != nil { t.Fatal(err) }
	if err := h.SetRole(channel.ID, bob.ID, alice.ID, "op"); err == nil { t.Fatal("operator affected original mod") }
}
