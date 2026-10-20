package games

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"strings"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mobbyg/otterlink/server/internal/db"
	_ "modernc.org/sqlite"
)

type fakeRuntime struct {
	mu      sync.Mutex
	process *fakeProcess
}

func (r *fakeRuntime) Name() string    { return "test runtime" }
func (r *fakeRuntime) Available() bool { return true }
func (r *fakeRuntime) Start(context.Context, string, string) (Process, error) {
	process := &fakeProcess{
		output: make(chan string, 8),
		done:   make(chan error, 1),
		input:  make(chan string, 8),
	}
	process.output <- "Welcome to the test game.\n"
	r.mu.Lock()
	r.process = process
	r.mu.Unlock()
	return process, nil
}

type fakeProcess struct {
	output chan string
	done   chan error
	input  chan string
	once   sync.Once
}

func (p *fakeProcess) WriteLine(value string) error {
	select {
	case <-p.done:
		return ErrSessionEnded
	default:
	}
	p.input <- value
	return nil
}
func (p *fakeProcess) Output() <-chan string { return p.output }
func (p *fakeProcess) Done() <-chan error    { return p.done }
func (p *fakeProcess) Stop() error {
	p.once.Do(func() {
		close(p.output)
		p.done <- nil
		close(p.done)
	})
	return nil
}

func setupGamesTest(t *testing.T) (*Service, int64, *fakeRuntime) {
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
	if _, err := database.Exec("INSERT INTO users(username,display_name,email,password_hash) VALUES('gamer','Game Tester','gamer@example.test','test-hash')"); err != nil {
		t.Fatal(err)
	}
	var userID int64
	if err := database.QueryRow("SELECT id FROM users WHERE username='gamer'").Scan(&userID); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "demo.z5"), []byte("fake story bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	runtime := &fakeRuntime{}
	service := NewService(database, root)
	service.Runtimes = map[string]Runtime{"zmachine": runtime}
	if err := service.Initialize(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	return service, userID, runtime
}

func TestCatalogAndSessionOwnership(t *testing.T) {
	service, userID, runtime := setupGamesTest(t)
	game, err := service.Save(0, SaveRequest{
		Slug: "demo-game", Title: "Demo Game", Author: "OtterLink",
		Description: "A tiny test story.", Category: "Interactive Fiction",
		Engine: "zmachine", StoryFile: "demo.z5", Published: true,
	}, userID)
	if err != nil {
		t.Fatal(err)
	}
	if !game.Published || !game.Available {
		t.Fatalf("expected published, available game, got %#v", game)
	}
	if _, err := service.Save(0, SaveRequest{
		Slug: "DEMO-GAME", Title: "Duplicate", Engine: "zmachine", StoryFile: "demo.z5",
	}, userID); err == nil {
		t.Fatal("duplicate game slug should be rejected case-insensitively")
	}

	session, err := service.StartSession(userID, game.ID)
	if err != nil {
		t.Fatal(err)
	}
	if session.Status != "running" {
		t.Fatalf("expected running session, got %q", session.Status)
	}
	waitForOutput(t, service, userID, session.ID)

	if _, err := service.PollOutput(userID+1, session.ID, 0, false); err != ErrForbidden {
		t.Fatalf("another user must not read this session, got %v", err)
	}
	if err := service.SendInput(userID, session.ID, "look", false); err != nil {
		t.Fatal(err)
	}
	runtime.mu.Lock()
	process := runtime.process
	runtime.mu.Unlock()
	select {
	case input := <-process.input:
		if input != "look" {
			t.Fatalf("expected command to reach runtime, got %q", input)
		}
	case <-time.After(time.Second):
		t.Fatal("runtime did not receive input")
	}

	if err := service.StopSession(userID, session.ID, false); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		result, err := service.PollOutput(userID, session.ID, 0, false)
		if err == nil && result.Session.Status == "finished" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("stopped game session did not finish")
}

func waitForOutput(t *testing.T, service *Service, userID int64, sessionID string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		result, err := service.PollOutput(userID, sessionID, 0, false)
		if err == nil && len(result.Chunks) > 0 {
			if result.Chunks[0].Text == "" {
				t.Fatal("runtime output chunk should not be empty")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("game session did not produce output")
}

func TestRejectUnsafeStoryFileNames(t *testing.T) {
	for _, name := range []string{"../outside.z5", "/tmp/story.z5", "folder\\story.z5", ".hidden.z5", "notes.txt"} {
		err := validateSave(SaveRequest{Slug: "story", Title: "Story", Engine: "zmachine", StoryFile: name})
		if err == nil {
			t.Errorf("expected unsafe story filename %q to be rejected", name)
		}
	}
}

func TestLimitsActiveSessionsPerUser(t *testing.T) {
	service, userID, _ := setupGamesTest(t)
	for i := 1; i <= maxActiveSessionsPerUser; i++ {
		game, err := service.Save(0, SaveRequest{
			Slug: "game-" + string(rune('0'+i)), Title: "Game " + string(rune('0'+i)),
			Engine: "zmachine", StoryFile: "demo.z5", Published: true,
		}, userID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.StartSession(userID, game.ID); err != nil {
			t.Fatalf("session %d should start: %v", i, err)
		}
	}
	extra, err := service.Save(0, SaveRequest{
		Slug: "game-extra", Title: "Extra Game", Engine: "zmachine",
		StoryFile: "demo.z5", Published: true,
	}, userID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartSession(userID, extra.ID); err == nil {
		t.Fatal("fourth active session should be rejected")
	}
}

func TestStoryUploadIsBoundedAndNeverOverwrites(t *testing.T) {
	service, _, _ := setupGamesTest(t)
	size, err := service.UploadStoryFile("upload.z5", strings.NewReader("story bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len("story bytes")) {
		t.Fatalf("expected upload size %d, got %d", len("story bytes"), size)
	}
	if _, err := service.UploadStoryFile("upload.z5", strings.NewReader("replacement")); err != ErrStoryFileExists {
		t.Fatalf("expected existing story file to be protected, got %v", err)
	}
	for _, name := range []string{"../escape.z5", "folder/story.z5", "notes.txt"} {
		if _, err := service.UploadStoryFile(name, strings.NewReader("data")); err == nil {
			t.Errorf("expected invalid upload filename %q to be rejected", name)
		}
	}
}

func TestStoryUploadRejectsOversizedFiles(t *testing.T) {
	service, _, _ := setupGamesTest(t)
	data := bytes.Repeat([]byte("x"), int(MaxStoryUploadBytes+1))
	if _, err := service.UploadStoryFile("too-large.z5", bytes.NewReader(data)); err != ErrUploadTooLarge {
		t.Fatalf("expected oversized upload to be rejected, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(service.Root, "too-large.z5")); !os.IsNotExist(err) {
		t.Fatalf("partial oversized upload should be removed, stat error: %v", err)
	}
}
