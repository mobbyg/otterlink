package web

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) newsList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = value
	}

	var sourceID int64
	if raw := strings.TrimSpace(r.URL.Query().Get("source_id")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1 {
			http.Error(w, "invalid source_id", http.StatusBadRequest)
			return
		}
		sourceID = value
	}

	items, err := s.News.ListItems(limit, r.URL.Query().Get("category"), sourceID)
	if err != nil {
		http.Error(w, "unable to list news", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) newsSources(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sources, err := s.News.ListSources()
	if err != nil {
		http.Error(w, "unable to list news sources", http.StatusInternalServerError)
		return
	}

	type sourceOption struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Category string `json:"category"`
	}
	result := make([]sourceOption, 0, len(sources))
	for _, source := range sources {
		if !source.Enabled {
			continue
		}
		result = append(result, sourceOption{ID: source.ID, Name: source.Name, Category: source.Category})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": result})
}
