package web

import (
	"crypto/sha256"
	"encoding/binary"
	"net/http"
)

func tokenConnectionID(token string) uint64 {
	sum := sha256.Sum256([]byte(token))
	id := binary.BigEndian.Uint64(sum[:8])
	if id == 0 {
		return 1
	}
	return id
}

func (s *Server) presenceAway(w http.ResponseWriter, r *http.Request) {
	user, ok := s.user(r)
	if !ok { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
	var req struct { Away bool `json:"away"` }
	if !decodeJSON(w, r, &req) { return }
	if s.Presence == nil || !s.Presence.SetAway(user.ID, req.Away) {
		http.Error(w, "presence unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"away": req.Away})
}
