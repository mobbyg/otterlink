package boards

import (
	"database/sql"
	"testing"

	"github.com/mobbyg/otterlink/server/internal/db"
	_ "modernc.org/sqlite"
)

func newModerationTestService(t *testing.T) (*sql.DB, Service, User, User) {
	t.Helper()

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })

	if err := db.Initialize(database); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		"INSERT INTO users(username,display_name,email,password_hash) VALUES" +
			"('owner','Board Owner','owner@example.test','test-hash')," +
			"('moderator','Board Moderator','moderator@example.test','test-hash')," +
			"('member','Board Member','member@example.test','test-hash')",
	); err != nil {
		t.Fatal(err)
	}

	service := Service{DB: database}
	if err := service.Initialize(); err != nil {
		t.Fatal(err)
	}

	var ownerID, moderatorID int64
	for username, target := range map[string]*int64{
		"owner": &ownerID, "moderator": &moderatorID,
	} {
		if err := database.QueryRow("SELECT id FROM users WHERE username=?", username).Scan(target); err != nil {
			t.Fatal(err)
		}
	}

	owner := User{ID: ownerID, Username: "owner", DisplayName: "Board Owner"}
	moderator := User{ID: moderatorID, Username: "moderator", DisplayName: "Board Moderator"}
	return database, service, owner, moderator
}

func TestBoardModeratorAndBanScope(t *testing.T) {
	database, service, owner, moderator := newModerationTestService(t)
	var memberID int64
	if err := database.QueryRow("SELECT id FROM users WHERE username='member'").Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	member := User{ID: memberID, Username: "member", DisplayName: "Board Member"}

	first, err := service.CreateBoard("Retro Computing", "Classic systems", "Be kind.", owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateBoard("General", "Everything else", "", owner.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.SetModerator(first.ID, moderator.ID, true); err != nil {
		t.Fatal(err)
	}
	if !service.IsModerator(first.ID, moderator.ID) {
		t.Fatal("appointed moderator should moderate their assigned board")
	}
	if service.IsModerator(second.ID, moderator.ID) {
		t.Fatal("board moderator must not receive permissions on another board")
	}

	thread, _, err := service.CreateThread(first.ID, owner, "Welcome", "A first post.")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Ban(first.ID, member.ID, moderator.ID, "Repeated spam"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.CreateThread(first.ID, member, "Spam", "This should be blocked."); err == nil {
		t.Fatal("banned member must not create a thread on the banned board")
	}
	if _, err := service.Reply(thread.ID, member, "This should also be blocked."); err == nil {
		t.Fatal("banned member must not reply on the banned board")
	}

	if _, _, err := service.CreateThread(second.ID, member, "General topic", "Posting elsewhere."); err != nil {
		t.Fatalf("a board ban must not affect another board: %v", err)
	}
	if err := service.Unban(first.ID, member.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Reply(thread.ID, member, "Allowed after unban."); err != nil {
		t.Fatalf("member should be able to reply after unban: %v", err)
	}
	if err := service.SetModerator(first.ID, moderator.ID, false); err != nil {
		t.Fatal(err)
	}
	if service.IsModerator(first.ID, moderator.ID) {
		t.Fatal("removed moderator should lose board permissions")
	}
}

func TestBoardDeletePostAndBoard(t *testing.T) {
	_, service, owner, _ := newModerationTestService(t)
	board, err := service.CreateBoard("Test Board", "", "", owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	thread, _, err := service.CreateThread(board.ID, owner, "Test thread", "Opening post.")
	if err != nil {
		t.Fatal(err)
	}
	reply, err := service.Reply(thread.ID, owner, "Second post.")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DeletePost(reply.ID); err != nil {
		t.Fatal(err)
	}
	loaded, posts, err := service.GetThread(thread.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ReplyCount != 0 || len(posts) != 1 {
		t.Fatalf("deleting a reply should preserve the opening post: thread=%#v posts=%#v", loaded, posts)
	}

	if err := service.DeleteBoard(board.ID); err != nil {
		t.Fatal(err)
	}
	boards, err := service.ListBoards(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(boards) != 0 {
		t.Fatalf("expected deleted board to be absent, got %#v", boards)
	}
	if _, _, err := service.GetThread(thread.ID, owner.ID); err == nil {
		t.Fatal("deleting a board should remove its threads")
	}
}

func TestBoardNamesAreUniqueWithoutCaseSensitivity(t *testing.T) {
	_, service, owner, _ := newModerationTestService(t)
	if _, err := service.CreateBoard("Retro Computing", "", "", owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateBoard("retro computing", "", "", owner.ID); err == nil {
		t.Fatal("board names should be unique regardless of letter case")
	}
	if _, err := service.CreateBoard("   ", "", "", owner.ID); err == nil {
		t.Fatal("board name must not be empty")
	}
}

func TestAdministratorCanStillUseBoardAfterBoardBan(t *testing.T) {
	_, service, owner, moderator := newModerationTestService(t)
	board, err := service.CreateBoard("Admin Test", "", "", owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Ban(board.ID, owner.ID, moderator.ID, "test ban"); err != nil {
		t.Fatal(err)
	}

	admin := owner
	admin.Role = "admin"
	if _, _, err := service.CreateThread(board.ID, admin, "Admin access", "Administrators retain access."); err != nil {
		t.Fatalf("a board-scoped ban must not lock out a global administrator: %v", err)
	}
}

func TestDeletingOpeningPostDeletesWholeThread(t *testing.T) {
	_, service, owner, _ := newModerationTestService(t)
	board, err := service.CreateBoard("Opening Post Test", "", "", owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	thread, posts, err := service.CreateThread(board.ID, owner, "Opening post", "This post anchors the thread.")
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected one opening post, got %d", len(posts))
	}
	if err := service.Reply(thread.ID, owner, "A reply."); err != nil {
		t.Fatal(err)
	}
	if err := service.DeletePost(posts[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.GetThread(thread.ID, owner.ID); err == nil {
		t.Fatal("deleting the opening post should remove the thread and replies")
	}
}
