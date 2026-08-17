package accounts

import (
	"database/sql"
	"testing"

	"github.com/mobbyg/otterlink/server/internal/db"
	_ "modernc.org/sqlite"
)

func testService(t *testing.T) Service {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Initialize(database); err != nil {
		t.Fatal(err)
	}
	return Service{DB: database}
}

func TestRegisterAuthenticateAndSession(t *testing.T) {
	svc := testService(t)
	user, err := svc.Register("freedomotter", "Freedom Otter", "", "a very long test password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Username != "freedomotter" || user.DisplayName != "Freedom Otter" {
		t.Fatalf("unexpected user: %+v", user)
	}

	got, token, err := svc.Authenticate("FREEDOMOTTER", "a very long test password")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.ID != user.ID || token == "" {
		t.Fatalf("unexpected authentication result: user=%+v token=%q", got, token)
	}

	fromToken, err := svc.FromToken(token)
	if err != nil {
		t.Fatalf("FromToken() error = %v", err)
	}
	if fromToken.ID != user.ID {
		t.Fatalf("FromToken() returned user %d, want %d", fromToken.ID, user.ID)
	}

	if err := svc.Logout(token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := svc.FromToken(token); err == nil {
		t.Fatal("session remained valid after logout")
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	svc := testService(t)
	if _, err := svc.Register("otter", "Otter", "", "too short"); err == nil {
		t.Fatal("expected short password to be rejected")
	}
}
