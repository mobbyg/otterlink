package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/games"
)

func (s *Server) gameList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	items, err := s.Games.ListPublished()
	if err != nil {
		http.Error(w, "unable to list games", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": items})
}

func (s *Server) gameGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := gameID(r)
	if !ok {
		http.Error(w, "invalid game id", http.StatusBadRequest)
		return
	}
	game, err := s.Games.GetPublished(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unable to load game", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, game)
}

func (s *Server) gameSessionStart(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, ok := gameID(r)
	if !ok {
		http.Error(w, "invalid game id", http.StatusBadRequest)
		return
	}
	session, err := s.Games.StartSession(user.ID, id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			http.Error(w, "game not found", http.StatusNotFound)
		case errors.Is(err, games.ErrRuntimeUnavailable):
			http.Error(w, "the Z-machine runtime is unavailable; install the frotz package on the server", http.StatusServiceUnavailable)
		case strings.Contains(err.Error(), "story file is unavailable"):
			http.Error(w, "the game story file is unavailable on the server", http.StatusServiceUnavailable)
		case strings.Contains(err.Error(), "active session") || strings.Contains(err.Error(), "active game sessions"):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) gameSessionOutput(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	after := int64(0)
	if value := strings.TrimSpace(r.URL.Query().Get("after")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 0 {
			http.Error(w, "invalid output sequence", http.StatusBadRequest)
			return
		}
		after = parsed
	}
	result, err := s.Games.PollOutput(user.ID, r.PathValue("sessionID"), after, user.Role == "admin")
	if err != nil {
		writeGameSessionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) gameSessionInput(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		Text string
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.Games.SendInput(user.ID, r.PathValue("sessionID"), req.Text, user.Role == "admin"); err != nil {
		writeGameSessionError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) gameSessionStop(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := s.Games.StopSession(user.ID, r.PathValue("sessionID"), user.Role == "admin"); err != nil {
		writeGameSessionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeGameSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		http.Error(w, "game session not found", http.StatusNotFound)
	case errors.Is(err, games.ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, games.ErrSessionEnded):
		http.Error(w, "game session has ended", http.StatusConflict)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (s *Server) adminGames(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.Games.ListAll()
		if err != nil {
			http.Error(w, "unable to list games", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"games": items, "runtime_available": s.Games.RuntimeAvailable("zmachine"),
		})
	case http.MethodPost:
		var req games.SaveRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		game, err := s.Games.Save(0, req, user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = s.Accounts.LogAudit(user, "admin.game_create", game.Title, "success", "Game registered", &game.ID)
		writeJSON(w, http.StatusCreated, game)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) adminGameFileUpload(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, games.MaxStoryUploadBytes+(1<<20))
	parseErr := r.ParseMultipartForm(1 << 20)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if parseErr != nil {
		http.Error(w, "story upload is too large or invalid", http.StatusRequestEntityTooLarge)
		return
	}
	file, header, err := r.FormFile("story_file")
	if err != nil {
		http.Error(w, "story_file upload is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	size, err := s.Games.UploadStoryFile(header.Filename, file)
	if errors.Is(err, games.ErrStoryFileExists) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if errors.Is(err, games.ErrUploadTooLarge) {
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.Accounts.LogAudit(user, "admin.game_file_upload", header.Filename, "success", strconv.FormatInt(size, 10)+" bytes", nil)
	writeJSON(w, http.StatusCreated, map[string]any{"filename": header.Filename, "bytes": size})
}

func (s *Server) adminGame(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if user.Role != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id, ok := gameID(r)
	if !ok {
		http.Error(w, "invalid game id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var req games.SaveRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		game, err := s.Games.Save(id, req, user.ID)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = s.Accounts.LogAudit(user, "admin.game_update", game.Title, "success", "Game metadata updated", &game.ID)
		writeJSON(w, http.StatusOK, game)
	case http.MethodDelete:
		game, err := s.Games.Get(id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "unable to load game", http.StatusInternalServerError)
			return
		}
		if err := s.Games.Delete(id); err != nil {
			http.Error(w, "unable to delete game", http.StatusInternalServerError)
			return
		}
		_ = s.Accounts.LogAudit(user, "admin.game_delete", game.Title, "success", "Game and its sessions deleted", &game.ID)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func gameID(r *http.Request) (int64, bool) {
	value := strings.TrimSpace(r.PathValue("gameID"))
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}
