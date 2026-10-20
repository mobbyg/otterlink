package games

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var (
	ErrForbidden          = errors.New("game session belongs to another user")
	ErrRuntimeUnavailable = errors.New("Z-machine runtime is unavailable")
	ErrSessionEnded       = errors.New("game session has ended")
	ErrStoryFileExists    = errors.New("story file already exists")
	ErrUploadTooLarge     = errors.New("story file upload is too large")
)

const maxOutputBytes = 256 * 1024
const maxActiveSessionsPerUser = 3
const MaxStoryUploadBytes int64 = 20 << 20

type Game struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Engine      string `json:"engine"`
	StoryFile   string `json:"story_file"`
	Published   bool   `json:"published"`
	Available   bool   `json:"available"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type SaveRequest struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Engine      string `json:"engine"`
	StoryFile   string `json:"story_file"`
	Published   bool   `json:"published"`
}

type SessionInfo struct {
	ID           string `json:"id"`
	GameID       int64  `json:"game_id"`
	GameSlug     string `json:"game_slug"`
	GameTitle    string `json:"game_title"`
	Status       string `json:"status"`
	StartedAt    string `json:"started_at"`
	LastActivity string `json:"last_activity"`
}

type OutputChunk struct {
	Sequence int64  `json:"sequence"`
	Text     string `json:"text"`
}

type OutputResult struct {
	Session      SessionInfo   `json:"session"`
	Chunks       []OutputChunk `json:"chunks"`
	NextSequence int64         `json:"next_sequence"`
	Truncated    bool          `json:"truncated"`
}

type Runtime interface {
	Name() string
	Available() bool
	Start(context.Context, string, string) (Process, error)
}

type Process interface {
	WriteLine(string) error
	Output() <-chan string
	Done() <-chan error
	Stop() error
}

type Service struct {
	DB       *sql.DB
	Root     string
	Runtimes map[string]Runtime

	mu          sync.Mutex
	sessions    map[string]*runningSession
	cleanupOnce     sync.Once
	cleanupStop     chan struct{}
	cleanupStopOnce sync.Once
	idleTimeout     time.Duration
}

type runningSession struct {
	info         SessionInfo
	userID       int64
	process      Process
	mu           sync.Mutex
	chunks       []OutputChunk
	bufferBytes  int
	nextSequence int64
	truncatedTo  int64
	lastActivity time.Time
}

func NewService(database *sql.DB, root string) *Service {
	if strings.TrimSpace(root) == "" {
		root = filepath.Join("data", "games")
	}
	return &Service{
		DB: database, Root: root,
		Runtimes: map[string]Runtime{"zmachine": NewZMachineRuntime()},
		sessions: make(map[string]*runningSession),
		idleTimeout: 30 * time.Minute,
	}
}

func (s *Service) Initialize() error {
	if s.DB == nil {
		return errors.New("games database is required")
	}
	if strings.TrimSpace(s.Root) == "" {
		s.Root = filepath.Join("data", "games")
	}
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return err
	}
	s.Root = root
	if err := os.MkdirAll(s.Root, 0750); err != nil {
		return fmt.Errorf("create games root: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(s.Root, ".saves"), 0700); err != nil {
		return fmt.Errorf("create game save root: %w", err)
	}
	if s.Runtimes == nil {
		s.Runtimes = map[string]Runtime{"zmachine": NewZMachineRuntime()}
	}
	if s.sessions == nil {
		s.sessions = make(map[string]*runningSession)
	}
	if s.idleTimeout <= 0 {
		s.idleTimeout = 30 * time.Minute
	}
	statements := []string{
		"CREATE TABLE IF NOT EXISTS games (id INTEGER PRIMARY KEY AUTOINCREMENT, slug TEXT NOT NULL UNIQUE COLLATE NOCASE, title TEXT NOT NULL, author TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT '', category TEXT NOT NULL DEFAULT '', engine TEXT NOT NULL CHECK(engine IN ('zmachine')), story_file TEXT NOT NULL, published INTEGER NOT NULL DEFAULT 0 CHECK(published IN (0,1)), created_by INTEGER REFERENCES users(id) ON DELETE SET NULL, created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)",
		"CREATE INDEX IF NOT EXISTS idx_games_published_title ON games(published,title COLLATE NOCASE)",
		"CREATE TABLE IF NOT EXISTS game_sessions (id TEXT PRIMARY KEY, game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, status TEXT NOT NULL CHECK(status IN ('running','stopping','finished','failed')), started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, last_activity TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, ended_at TEXT)",
		"CREATE INDEX IF NOT EXISTS idx_game_sessions_user_status ON game_sessions(user_id,status)",
	}
	for _, statement := range statements {
		if _, err := s.DB.Exec(statement); err != nil {
			return fmt.Errorf("initialize games schema: %w", err)
		}
	}
	if _, err := s.DB.Exec("UPDATE game_sessions SET status='failed',ended_at=CURRENT_TIMESTAMP WHERE status IN ('running','stopping')"); err != nil {
		return fmt.Errorf("recover stale game sessions: %w", err)
	}
	if s.cleanupStop == nil {
		s.cleanupStop = make(chan struct{})
	}
	s.cleanupOnce.Do(func() { go s.cleanupLoop() })
	return nil
}

func (s *Service) RuntimeAvailable(engine string) bool {
	runtime, ok := s.Runtimes[strings.ToLower(strings.TrimSpace(engine))]
	return ok && runtime.Available()
}

func (s *Service) ListPublished() ([]Game, error) { return s.list(false) }
func (s *Service) ListAll() ([]Game, error)       { return s.list(true) }

func (s *Service) list(includeUnpublished bool) ([]Game, error) {
	query := "SELECT id,slug,title,author,description,category,engine,story_file,published,created_at,updated_at FROM games"
	if !includeUnpublished {
		query += " WHERE published=1"
	}
	query += " ORDER BY category COLLATE NOCASE,title COLLATE NOCASE,id"
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Game, 0)
	for rows.Next() {
		game, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		game.Available = s.gameAvailable(game)
		result = append(result, game)
	}
	return result, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanGame(row scanner) (Game, error) {
	var game Game
	var published int
	err := row.Scan(&game.ID, &game.Slug, &game.Title, &game.Author, &game.Description,
		&game.Category, &game.Engine, &game.StoryFile, &published, &game.CreatedAt, &game.UpdatedAt)
	game.Published = published != 0
	return game, err
}

func (s *Service) Get(id int64) (Game, error) {
	game, err := s.get(id, false)
	if err == nil {
		game.Available = s.gameAvailable(game)
	}
	return game, err
}

func (s *Service) GetPublished(id int64) (Game, error) {
	game, err := s.get(id, true)
	if err == nil {
		game.Available = s.gameAvailable(game)
	}
	return game, err
}

func (s *Service) get(id int64, publishedOnly bool) (Game, error) {
	query := "SELECT id,slug,title,author,description,category,engine,story_file,published,created_at,updated_at FROM games WHERE id=?"
	if publishedOnly {
		query += " AND published=1"
	}
	return scanGame(s.DB.QueryRow(query, id))
}

func (s *Service) UploadStoryFile(filename string, source io.Reader) (int64, error) {
	originalFilename := filename
	filename = strings.TrimSpace(filename)
	if filename != originalFilename {
		return 0, errors.New("story filename cannot start or end with whitespace")
	}
	if filename == "" || len(filename) > 255 || !utf8.ValidString(filename) ||
		strings.ContainsAny(filename, string([]byte{0, 10, 13})) ||
		filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\\`) ||
		strings.HasPrefix(filename, ".") {
		return 0, errors.New("story upload must use a plain filename")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := map[string]bool{
		".z1": true, ".z2": true, ".z3": true, ".z4": true,
		".z5": true, ".z6": true, ".z7": true, ".z8": true,
		".blb": true, ".blorb": true, ".zblorb": true,
	}
	if !allowed[ext] {
		return 0, errors.New("upload must be a supported Z-machine or Blorb file")
	}
	destination := filepath.Join(s.Root, filename)
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if errors.Is(err, os.ErrExist) {
		return 0, ErrStoryFileExists
	}
	if err != nil {
		return 0, err
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(destination)
		}
	}()
	size, err := io.Copy(file, io.LimitReader(source, MaxStoryUploadBytes+1))
	if err != nil {
		return 0, err
	}
	if size == 0 {
		return 0, errors.New("story file is empty")
	}
	if size > MaxStoryUploadBytes {
		return 0, ErrUploadTooLarge
	}
	if err := file.Sync(); err != nil {
		return 0, err
	}
	if err := file.Close(); err != nil {
		return 0, err
	}
	keep = true
	return size, nil
}

func (s *Service) Save(id int64, req SaveRequest, actorID int64) (Game, error) {
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.Title = strings.TrimSpace(req.Title)
	req.Author = strings.TrimSpace(req.Author)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)
	req.Engine = strings.ToLower(strings.TrimSpace(req.Engine))
	req.StoryFile = strings.TrimSpace(req.StoryFile)
	if err := validateSave(req); err != nil {
		return Game{}, err
	}
	if _, err := s.resolveStoryPath(req.StoryFile); err != nil {
		return Game{}, fmt.Errorf("story file is not available under the games directory: %w", err)
	}
	published := 0
	if req.Published {
		published = 1
	}
	if id > 0 {
		result, err := s.DB.Exec("UPDATE games SET slug=?,title=?,author=?,description=?,category=?,engine=?,story_file=?,published=?,updated_at=CURRENT_TIMESTAMP WHERE id=?",
			req.Slug, req.Title, req.Author, req.Description, req.Category, req.Engine, req.StoryFile, published, id)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return Game{}, errors.New("a game with that slug already exists")
			}
			return Game{}, err
		}
		changed, _ := result.RowsAffected()
		if changed == 0 {
			return Game{}, sql.ErrNoRows
		}
		return s.Get(id)
	}
	result, err := s.DB.Exec("INSERT INTO games(slug,title,author,description,category,engine,story_file,published,created_by) VALUES(?,?,?,?,?,?,?,?,?)",
		req.Slug, req.Title, req.Author, req.Description, req.Category, req.Engine, req.StoryFile, published, actorID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Game{}, errors.New("a game with that slug already exists")
		}
		return Game{}, err
	}
	newID, err := result.LastInsertId()
	if err != nil {
		return Game{}, err
	}
	return s.Get(newID)
}

func validateSave(req SaveRequest) error {
	if req.Slug == "" || len([]rune(req.Slug)) > 64 {
		return errors.New("game slug must be 1-64 characters")
	}
	for _, r := range req.Slug {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return errors.New("game slug may contain only lowercase letters, numbers, '-' and '_'")
		}
	}
	if req.Title == "" || len([]rune(req.Title)) > 160 {
		return errors.New("game title must be 1-160 characters")
	}
	if len([]rune(req.Author)) > 120 || len([]rune(req.Description)) > 4000 || len([]rune(req.Category)) > 64 {
		return errors.New("game metadata exceeds the allowed length")
	}
	if req.Engine != "zmachine" {
		return errors.New("only the Z-machine runtime is supported in this release")
	}
	if req.StoryFile == "" || filepath.Base(req.StoryFile) != req.StoryFile ||
		strings.ContainsAny(req.StoryFile, `/\\`) || strings.HasPrefix(req.StoryFile, ".") {
		return errors.New("story_file must be a filename inside the configured games directory")
	}
	ext := strings.ToLower(filepath.Ext(req.StoryFile))
	allowed := map[string]bool{".z1": true, ".z2": true, ".z3": true, ".z4": true, ".z5": true, ".z6": true, ".z7": true, ".z8": true, ".blorb": true, ".zblorb": true}
	if !allowed[ext] {
		return errors.New("story_file must be a supported Z-machine or Blorb file")
	}
	return nil
}

func (s *Service) gameAvailable(game Game) bool {
	runtime, ok := s.Runtimes[game.Engine]
	if !ok || !runtime.Available() {
		return false
	}
	_, err := s.resolveStoryPath(game.StoryFile)
	return err == nil
}

func (s *Service) resolveStoryPath(filename string) (string, error) {
	if filename == "" || filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\\`) {
		return "", errors.New("invalid story filename")
	}
	root, err := filepath.EvalSymlinks(s.Root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, filename))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("story file resolves outside the games directory")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("story file is not a regular file")
	}
	return resolved, nil
}

func (s *Service) Delete(id int64) error {
	s.mu.Lock()
	var active []*runningSession
	for _, session := range s.sessions {
		if session.info.GameID != id {
			continue
		}
		session.mu.Lock()
		running := session.info.Status == "running" || session.info.Status == "stopping"
		session.mu.Unlock()
		if running {
			active = append(active, session)
		}
	}
	s.mu.Unlock()
	for _, session := range active {
		_ = session.process.Stop()
	}
	result, err := s.DB.Exec("DELETE FROM games WHERE id=?", id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Service) StartSession(userID, gameID int64) (SessionInfo, error) {
	game, err := s.GetPublished(gameID)
	if err != nil {
		return SessionInfo{}, err
	}
	runtime, ok := s.Runtimes[game.Engine]
	if !ok || !runtime.Available() {
		return SessionInfo{}, ErrRuntimeUnavailable
	}
	storyPath, err := s.resolveStoryPath(game.StoryFile)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("story file is unavailable: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	activeForUser := 0
	for _, existing := range s.sessions {
		existing.mu.Lock()
		active := existing.info.Status == "running" || existing.info.Status == "stopping"
		existingGameID := existing.info.GameID
		existingUserID := existing.userID
		existing.mu.Unlock()
		if !active || existingUserID != userID {
			continue
		}
		activeForUser++
		if existingGameID == gameID {
			return SessionInfo{}, errors.New("you already have an active session for this game")
		}
	}
	if activeForUser >= maxActiveSessionsPerUser {
		return SessionInfo{}, errors.New("you have reached the maximum of three active game sessions")
	}

	sessionID, err := newSessionID()
	if err != nil {
		return SessionInfo{}, err
	}
	saveDir := filepath.Join(s.Root, ".saves", strconv.FormatInt(userID, 10), strconv.FormatInt(gameID, 10))
	if err := os.MkdirAll(saveDir, 0700); err != nil {
		return SessionInfo{}, fmt.Errorf("create game save directory: %w", err)
	}
	if err := copyStoryFiles(storyPath, saveDir); err != nil {
		return SessionInfo{}, fmt.Errorf("prepare game story: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	process, err := runtime.Start(ctx, filepath.Join(saveDir, filepath.Base(storyPath)), saveDir)
	if err != nil {
		cancel()
		return SessionInfo{}, err
	}
	now := time.Now().UTC()
	info := SessionInfo{ID: sessionID, GameID: game.ID, GameSlug: game.Slug, GameTitle: game.Title,
		Status: "running", StartedAt: now.Format(time.RFC3339), LastActivity: now.Format(time.RFC3339)}
	if _, err := s.DB.Exec("INSERT INTO game_sessions(id,game_id,user_id,status,started_at,last_activity) VALUES(?,?,?,'running',?,?)",
		info.ID, info.GameID, userID, info.StartedAt, info.LastActivity); err != nil {
		_ = process.Stop()
		cancel()
		return SessionInfo{}, err
	}
	session := &runningSession{info: info, userID: userID, process: process, lastActivity: now}
	s.sessions[info.ID] = session
	go s.watchSession(session, cancel)
	return info, nil
}

func (s *Service) watchSession(session *runningSession, cancel context.CancelFunc) {
	for text := range session.process.Output() {
		if text == "" {
			continue
		}
		session.mu.Lock()
		session.appendOutput(text)
		session.lastActivity = time.Now().UTC()
		session.info.LastActivity = session.lastActivity.Format(time.RFC3339)
		session.mu.Unlock()
	}
	var runErr error
	if done := session.process.Done(); done != nil {
		runErr = <-done
	}
	cancel()
	status := "finished"
	session.mu.Lock()
	wasStopping := session.info.Status == "stopping"
	session.mu.Unlock()
	if runErr != nil && !errors.Is(runErr, context.Canceled) && !wasStopping {
		status = "failed"
	}
	now := time.Now().UTC()
	session.mu.Lock()
	session.info.Status = status
	session.info.LastActivity = now.Format(time.RFC3339)
	session.lastActivity = now
	session.mu.Unlock()
	_, _ = s.DB.Exec("UPDATE game_sessions SET status=?,last_activity=?,ended_at=? WHERE id=?",
		status, now.Format(time.RFC3339), now.Format(time.RFC3339), session.info.ID)
}

func (session *runningSession) appendOutput(text string) {
	session.nextSequence++
	session.chunks = append(session.chunks, OutputChunk{Sequence: session.nextSequence, Text: text})
	session.bufferBytes += len(text)
	for session.bufferBytes > maxOutputBytes && len(session.chunks) > 1 {
		session.bufferBytes -= len(session.chunks[0].Text)
		session.truncatedTo = session.chunks[0].Sequence
		session.chunks = session.chunks[1:]
	}
}

func (s *Service) PollOutput(userID int64, sessionID string, after int64, admin bool) (OutputResult, error) {
	session, err := s.findSession(userID, sessionID, admin)
	if err != nil {
		return OutputResult{}, err
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	result := OutputResult{Session: session.info, Chunks: make([]OutputChunk, 0), NextSequence: session.nextSequence, Truncated: after < session.truncatedTo}
	for _, chunk := range session.chunks {
		if chunk.Sequence > after {
			result.Chunks = append(result.Chunks, chunk)
		}
	}
	return result, nil
}

func (s *Service) SendInput(userID int64, sessionID, text string, admin bool) error {
	text = strings.TrimSpace(text)
	if text == "" || len(text) > 2048 || strings.ContainsAny(text, string([]byte{10, 13, 0})) || !utf8.ValidString(text) {
		return errors.New("input must be a single line of 1-2048 valid UTF-8 bytes")
	}
	session, err := s.findSession(userID, sessionID, admin)
	if err != nil {
		return err
	}
	session.mu.Lock()
	active := session.info.Status == "running"
	session.mu.Unlock()
	if !active {
		return ErrSessionEnded
	}
	if err := session.process.WriteLine(text); err != nil {
		return err
	}
	now := time.Now().UTC()
	session.mu.Lock()
	session.lastActivity = now
	session.info.LastActivity = now.Format(time.RFC3339)
	session.mu.Unlock()
	_, _ = s.DB.Exec("UPDATE game_sessions SET last_activity=? WHERE id=?", now.Format(time.RFC3339), sessionID)
	return nil
}

func (s *Service) StopSession(userID int64, sessionID string, admin bool) error {
	session, err := s.findSession(userID, sessionID, admin)
	if err != nil {
		return err
	}
	session.mu.Lock()
	active := session.info.Status == "running" || session.info.Status == "stopping"
	if active {
		session.info.Status = "stopping"
	}
	session.mu.Unlock()
	if !active {
		return nil
	}
	return session.process.Stop()
}

func (s *Service) findSession(userID int64, sessionID string, admin bool) (*runningSession, error) {
	s.mu.Lock()
	session := s.sessions[sessionID]
	s.mu.Unlock()
	if session == nil {
		return nil, sql.ErrNoRows
	}
	if !admin && session.userID != userID {
		return nil, ErrForbidden
	}
	return session, nil
}

func (s *Service) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.cleanupStop:
			return
		case <-ticker.C:
		}
		now := time.Now().UTC()
		s.mu.Lock()
		var expired []*runningSession
		var remove []string
		for id, session := range s.sessions {
			session.mu.Lock()
			if (session.info.Status == "running" || session.info.Status == "stopping") && now.Sub(session.lastActivity) > s.idleTimeout {
				session.info.Status = "stopping"
				expired = append(expired, session)
			} else if (session.info.Status == "finished" || session.info.Status == "failed") && now.Sub(session.lastActivity) > 2*time.Hour {
				remove = append(remove, id)
			}
			session.mu.Unlock()
		}
		for _, id := range remove {
			delete(s.sessions, id)
		}
		s.mu.Unlock()
		for _, session := range expired {
			_ = session.process.Stop()
		}
	}
}

func (s *Service) Close() {
	if s.cleanupStop != nil {
		s.cleanupStopOnce.Do(func() { close(s.cleanupStop) })
	}
	s.mu.Lock()
	sessions := make([]*runningSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	s.mu.Unlock()
	for _, session := range sessions {
		_ = session.process.Stop()
	}
}

func newSessionID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func copyStoryFiles(storyPath, destination string) error {
	base := filepath.Base(storyPath)
	files := []string{storyPath}
	ext := filepath.Ext(base)
	if strings.HasPrefix(strings.ToLower(ext), ".z") && len(ext) == 3 {
		stem := strings.TrimSuffix(storyPath, ext)
		for _, companionExt := range []string{".blb", ".blorb", ".zblorb"} {
			companion := stem + companionExt
			if info, err := os.Stat(companion); err == nil && info.Mode().IsRegular() {
				files = append(files, companion)
			}
		}
	}
	for _, source := range files {
		if err := copyFile(source, filepath.Join(destination, filepath.Base(source))); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	temp, err := os.CreateTemp(filepath.Dir(destination), ".story-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := io.Copy(temp, in); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, destination)
}
