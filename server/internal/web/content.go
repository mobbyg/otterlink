package web

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mobbyg/otterlink/server/internal/content"
)

func (s *Server) contentHome(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	screen, err := s.Content.Active("home", time.Now().UTC())
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "home screen is not published", http.StatusNotFound)
			return
		}
		http.Error(w, "unable to load home screen", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, screen)
}

func (s *Server) contentScreen(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("screenID"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid screen id", http.StatusBadRequest)
		return
	}
	screen, err := s.Content.Get(id)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "unable to load screen", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, screen)
}

func (s *Server) adminContentIndex(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	data, err := staticFiles.ReadFile("static/content.html")
	if err != nil {
		http.Error(w, "content editor unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) adminContentScreens(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	screens, err := s.Content.List()
	if err != nil {
		http.Error(w, "unable to list content screens", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"screens": screens})
}

func (s *Server) adminContentSave(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req content.SaveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	screen, err := s.Content.Save(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, screen)
}

func (s *Server) adminContentDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("screenID"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid screen id", http.StatusBadRequest)
		return
	}
	if err := s.Content.Delete(id); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "unable to delete screen", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) contentStaticHome(w http.ResponseWriter, _ *http.Request) {
	http.Redirect(w, r.URL.Path, http.StatusTemporaryRedirect)
}

var _ = strings.TrimSpace
