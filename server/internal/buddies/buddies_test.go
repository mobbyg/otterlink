package buddies

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
	_, err = database.Exec(`INSERT INTO users (username, display_name, password_hash) VALUES ('testotter', 'Test Otter', 'x'), ('otter2', 'Second Otter', 'x')`)
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { database.Close() })
	return database
}

func TestAddListRemove(t *testing.T) {
	database := testDB(t)
	s := Service{DB: database}

	buddy, err := s.Add(1, "otter2")
	if err != nil { t.Fatal(err) }
	if buddy.Username != "otter2" { t.Fatalf("unexpected buddy: %+v", buddy) }

	list, err := s.List(1)
	if err != nil { t.Fatal(err) }
	if len(list) != 1 || list[0].Username != "otter2" { t.Fatalf("unexpected list: %+v", list) }

	if err := s.Remove(1, "otter2"); err != nil { t.Fatal(err) }
	list, err = s.List(1)
	if err != nil { t.Fatal(err) }
	if len(list) != 0 { t.Fatalf("expected empty list, got %+v", list) }
}

func TestCannotAddSelf(t *testing.T) {
	database := testDB(t)
	_, err := (Service{DB: database}).Add(1, "testotter")
	if err == nil || err.Error() != "cannot add yourself as a buddy" { t.Fatalf("unexpected error: %v", err) }
}
