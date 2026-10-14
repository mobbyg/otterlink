package web

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type nativeNewsArticle struct {
	ID        int64  `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Published bool   `json:"published"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type nativeNewsSaveRequest struct {
	ID        int64  `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Published bool   `json:"published"`
}

var nativeNewsSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func (s *Server) adminNativeNewsIndex(w http.ResponseWriter, _ *http.Request) {
	data, err := staticFiles.ReadFile("static/native-news.html")
	if err != nil {
		http.Error(w, "native news editor unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) adminNativeNewsList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	rows, err := s.Content.DB.Query(`SELECT id, slug, title, body, published, created_at, updated_at
		FROM native_news_articles ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		http.Error(w, "unable to list native articles", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	articles := make([]nativeNewsArticle, 0)
	for rows.Next() {
		var article nativeNewsArticle
		var published int
		if err := rows.Scan(&article.ID, &article.Slug, &article.Title, &article.Body,
			&published, &article.CreatedAt, &article.UpdatedAt); err != nil {
			http.Error(w, "unable to read native articles", http.StatusInternalServerError)
			return
		}
		article.Published = published != 0
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "unable to read native articles", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"articles": articles})
}

func (s *Server) adminNativeNewsSave(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var req nativeNewsSaveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	req.Title = strings.TrimSpace(req.Title)
	if req.ID == 0 && !nativeNewsSlugPattern.MatchString(req.Slug) {
		http.Error(w, "slug must be 1-64 lowercase letters, numbers, hyphens, or underscores and start with a letter or number", http.StatusBadRequest)
		return
	}
	if req.Title == "" || len([]rune(req.Title)) > 160 {
		http.Error(w, "title must be 1-160 characters", http.StatusBadRequest)
		return
	}
	if len(req.Body) > 1024*1024 {
		http.Error(w, "article body must be no larger than 1 MB", http.StatusBadRequest)
		return
	}

	var id int64
	if req.ID > 0 {
		result, err := s.Content.DB.Exec(`UPDATE native_news_articles
			SET title=?, body=?, published=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			req.Title, req.Body, req.Published, req.ID)
		if err != nil {
			http.Error(w, "unable to update native article", http.StatusInternalServerError)
			return
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			http.Error(w, "native article not found", http.StatusNotFound)
			return
		}
		id = req.ID
	} else {
		result, err := s.Content.DB.Exec(`INSERT INTO native_news_articles
			(slug, title, body, published) VALUES (?, ?, ?, ?)`,
			req.Slug, req.Title, req.Body, req.Published)
		if err != nil {
			http.Error(w, "unable to create native article (the slug may already be in use)", http.StatusConflict)
			return
		}
		id, err = result.LastInsertId()
		if err != nil {
			http.Error(w, "unable to identify saved native article", http.StatusInternalServerError)
			return
		}
	}

	article, err := getNativeNewsByID(s.Content.DB, id)
	if err != nil {
		http.Error(w, "unable to load saved native article", http.StatusInternalServerError)
		return
	}
	_ = s.Accounts.LogAudit(admin, "admin.native_news_save", article.Slug, "success",
		"Native OtterLink article saved", nil)
	writeJSON(w, http.StatusOK, article)
}

func (s *Server) nativeNewsArticle(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.user(r); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
	if !nativeNewsSlugPattern.MatchString(slug) {
		http.NotFound(w, r)
		return
	}
	var article nativeNewsArticle
	var published int
	err := s.Content.DB.QueryRow(`SELECT id, slug, title, body, published, created_at, updated_at
		FROM native_news_articles WHERE slug=? AND published=1`, slug).Scan(
		&article.ID, &article.Slug, &article.Title, &article.Body, &published,
		&article.CreatedAt, &article.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "unable to load native article", http.StatusInternalServerError)
		return
	}
	article.Published = published != 0
	writeJSON(w, http.StatusOK, article)
}

func getNativeNewsByID(db *sql.DB, id int64) (nativeNewsArticle, error) {
	var article nativeNewsArticle
	var published int
	err := db.QueryRow(`SELECT id, slug, title, body, published, created_at, updated_at
		FROM native_news_articles WHERE id=?`, id).Scan(
		&article.ID, &article.Slug, &article.Title, &article.Body, &published,
		&article.CreatedAt, &article.UpdatedAt)
	if err != nil {
		return nativeNewsArticle{}, err
	}
	article.Published = published != 0
	return article, nil
}

var _ = strconv.IntSize
var _ = time.Second
