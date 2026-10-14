package web

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mobbyg/otterlink/server/internal/assets"
)

func (s *Server) contentAsset(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("assetID"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid asset id", http.StatusBadRequest)
		return
	}
	asset, file, err := s.Assets.Open(id)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "unable to load asset", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	etag := `"` + asset.SHA256 + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("Content-Type", asset.MIME)
	w.Header().Set("Content-Length", strconv.FormatInt(asset.Size, 10))
	if strings.TrimSpace(r.Header.Get("If-None-Match")) == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = io.Copy(w, file)
}

func (s *Server) adminContentAssets(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	assetsList, err := s.Assets.List()
	if err != nil {
		http.Error(w, "unable to list content assets", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assetsList})
}

func (s *Server) adminContentAssetUpload(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, assets.MaxUploadSize+1024*1024)
	if err := r.ParseMultipartForm(assets.MaxUploadSize + 1024*1024); err != nil {
		http.Error(w, "asset upload is too large or invalid", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("asset")
	if err != nil {
		http.Error(w, "asset file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	if !assets.IsAllowedMIME(contentType) {
		contentType = assets.GuessMIME(header.Filename)
	}
	if !assets.IsAllowedMIME(contentType) {
		http.Error(w, "unsupported asset type; use PNG, JPEG, GIF, or WebP", http.StatusBadRequest)
		return
	}
	if header.Size < 1 || header.Size > assets.MaxUploadSize {
		http.Error(w, "asset must be between 1 byte and 10 MB", http.StatusBadRequest)
		return
	}

	asset, err := s.Assets.Save(header.Filename, contentType, header.Size, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, asset)
}

func (s *Server) adminContentAssetDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("assetID"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid asset id", http.StatusBadRequest)
		return
	}
	if err := s.Assets.Delete(id); err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "unable to delete asset", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
