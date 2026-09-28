package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/keywords"
)

type keywordRequest struct {
	DisplayName string             `json:"display_name"`
	Description string             `json:"description"`
	Targets     []keywords.Target `json:"targets"`
}

func (s *Server) keywordResolve(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	key := strings.TrimSpace(r.PathValue("keyword"))
	result, err := s.Keywords.Resolve(key)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "keyword not found", http.StatusNotFound)
			return
		}
		http.Error(w, "unable to resolve keyword", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) adminKeywords(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok { return }
	result, err := s.Keywords.List()
	if err != nil {
		http.Error(w, "unable to list keywords", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"keywords": result})
}

func (s *Server) validateKeywordTargets(targets []keywords.Target) error {
	for _, target := range targets {
		switch target.Type {
		case "chat":
			if _, err := s.chatChannelByID(target.ID); err != nil {
				return errors.New("keyword chat target not found")
			}
		case "event":
			if _, err := s.Events.Get(target.ID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return errors.New("keyword event target not found")
				}
				return errors.New("unable to load keyword event target")
			}
		case "bulletin":
			return errors.New("bulletin keyword targets are reserved for future integration")
		default:
			return errors.New("invalid keyword target type")
		}
	}
	return nil
}

func (s *Server) adminKeywordUpsert(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok { return }
	key := strings.TrimSpace(r.PathValue("keyword"))
	var req keywordRequest
	if !decodeJSON(w, r, &req) { return }
	if err := s.validateKeywordTargets(req.Targets); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Keywords.Upsert(keywords.Keyword{
		Keyword: key, DisplayName: req.DisplayName, Description: req.Description, Targets: req.Targets,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.Accounts.LogAudit(admin, "admin.service_keyword_upsert", key, "success", "Service keyword configured", nil); err != nil {
		http.Error(w, "unable to record audit event", http.StatusInternalServerError)
		return
	}
	result, err := s.Keywords.Resolve(key)
	if err != nil {
		http.Error(w, "unable to load saved keyword", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) adminKeywordDelete(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok { return }
	key := strings.TrimSpace(r.PathValue("keyword"))
	if err := s.Keywords.Delete(key); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "keyword not found", http.StatusNotFound)
			return
		}
		http.Error(w, "unable to delete keyword", http.StatusInternalServerError)
		return
	}
	if err := s.Accounts.LogAudit(admin, "admin.service_keyword_delete", key, "success", "Service keyword deleted", nil); err != nil {
		http.Error(w, "unable to record audit event", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
