package web

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
)

type adminNewsSourceRequest struct {
	Name                     string `json:"name"`
	URL                      string `json:"url"`
	Category                 string `json:"category"`
	Enabled                  bool   `json:"enabled"`
	RefreshIntervalMinutes   int    `json:"refresh_interval_minutes"`
}

func (s *Server) adminNewsSources(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok { return }
	sources, err := s.News.ListSources()
	if err != nil { http.Error(w, "unable to list news sources", http.StatusInternalServerError); return }
	writeJSON(w, http.StatusOK, map[string]any{"sources": sources})
}

func (s *Server) adminNewsSourceCreate(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok { return }
	var req adminNewsSourceRequest
	if !decodeJSON(w, r, &req) { return }
	source, err := s.News.CreateSource(req.Name, req.URL, req.Category, req.Enabled, req.RefreshIntervalMinutes)
	if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	_ = s.Accounts.LogAudit(admin, "admin.news_source_create", source.Name, "success", "News feed source created", nil)
	writeJSON(w, http.StatusCreated, source)
}

func (s *Server) adminNewsSourceUpdate(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok { return }
	id, err := strconv.ParseInt(r.PathValue("sourceID"), 10, 64)
	if err != nil || id < 1 { http.Error(w, "invalid news source id", http.StatusBadRequest); return }
	var req adminNewsSourceRequest
	if !decodeJSON(w, r, &req) { return }
	source, err := s.News.UpdateSource(id, req.Name, req.URL, req.Category, req.Enabled, req.RefreshIntervalMinutes)
	if errors.Is(err, sql.ErrNoRows) { http.Error(w, "news source not found", http.StatusNotFound); return }
	if err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	_ = s.Accounts.LogAudit(admin, "admin.news_source_update", source.Name, "success", "News feed source updated", nil)
	writeJSON(w, http.StatusOK, source)
}

func (s *Server) adminNewsSourceDelete(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok { return }
	id, err := strconv.ParseInt(r.PathValue("sourceID"), 10, 64)
	if err != nil || id < 1 { http.Error(w, "invalid news source id", http.StatusBadRequest); return }
	source, err := s.News.GetSource(id)
	if errors.Is(err, sql.ErrNoRows) { http.Error(w, "news source not found", http.StatusNotFound); return }
	if err != nil { http.Error(w, "unable to load news source", http.StatusInternalServerError); return }
	if err := s.News.DeleteSource(id); errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "news source not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "unable to delete news source", http.StatusInternalServerError)
		return
	}
	_ = s.Accounts.LogAudit(admin, "admin.news_source_delete", source.Name, "success", "News feed source deleted", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adminNewsSourceTest(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok { return }
	id, err := strconv.ParseInt(r.PathValue("sourceID"), 10, 64)
	if err != nil || id < 1 { http.Error(w, "invalid news source id", http.StatusBadRequest); return }
	title, err := s.News.TestSource(context.Background(), id)
	if err != nil { http.Error(w, err.Error(), http.StatusBadGateway); return }
	_ = s.Accounts.LogAudit(admin, "admin.news_source_test", strconv.FormatInt(id, 10), "success", "News feed source tested successfully", nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "feed_title": title})
}
