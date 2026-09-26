package news

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Source struct {
	ID                       int64  `json:"id"`
	Name                     string `json:"name"`
	URL                      string `json:"url"`
	Category                 string `json:"category"`
	Enabled                  bool   `json:"enabled"`
	RefreshIntervalMinutes   int    `json:"refresh_interval_minutes"`
	LastFetchedAt            string `json:"last_fetched_at,omitempty"`
	LastSuccessAt            string `json:"last_success_at,omitempty"`
	LastError                string `json:"last_error,omitempty"`
}

type Service struct {
	DB *sql.DB
	HTTPClient *http.Client
}

func (s Service) ListSources() ([]Source, error) {
	rows, err := s.DB.Query(`SELECT id,name,url,category,enabled,refresh_interval_minutes,COALESCE(last_fetched_at,''),COALESCE(last_success_at,''),COALESCE(last_error,'') FROM news_sources ORDER BY name COLLATE NOCASE, id`)
	if err != nil { return nil, err }
	defer rows.Close()

	result := make([]Source, 0)
	for rows.Next() {
		var source Source
		var enabled int
		if err := rows.Scan(&source.ID, &source.Name, &source.URL, &source.Category, &enabled, &source.RefreshIntervalMinutes, &source.LastFetchedAt, &source.LastSuccessAt, &source.LastError); err != nil {
			return nil, err
		}
		source.Enabled = enabled != 0
		result = append(result, source)
	}
	return result, rows.Err()
}

func (s Service) CreateSource(name, rawURL, category string, enabled bool, refreshMinutes int) (Source, error) {
	if err := validateSource(name, rawURL, category, refreshMinutes); err != nil { return Source{}, err }
	result, err := s.DB.Exec(`INSERT INTO news_sources (name,url,category,enabled,refresh_interval_minutes) VALUES (?,?,?,?,?)`,
		strings.TrimSpace(name), strings.TrimSpace(rawURL), strings.TrimSpace(category), enabled, refreshMinutes)
	if err != nil { return Source{}, fmt.Errorf("create news source: %w", err) }
	id, err := result.LastInsertId()
	if err != nil { return Source{}, err }
	return s.GetSource(id)
}

func (s Service) GetSource(id int64) (Source, error) {
	var source Source
	var enabled int
	err := s.DB.QueryRow(`SELECT id,name,url,category,enabled,refresh_interval_minutes,COALESCE(last_fetched_at,''),COALESCE(last_success_at,''),COALESCE(last_error,'') FROM news_sources WHERE id=?`, id).
		Scan(&source.ID, &source.Name, &source.URL, &source.Category, &enabled, &source.RefreshIntervalMinutes, &source.LastFetchedAt, &source.LastSuccessAt, &source.LastError)
	if err != nil { return Source{}, err }
	source.Enabled = enabled != 0
	return source, nil
}

func (s Service) UpdateSource(id int64, name, rawURL, category string, enabled bool, refreshMinutes int) (Source, error) {
	if err := validateSource(name, rawURL, category, refreshMinutes); err != nil { return Source{}, err }
	result, err := s.DB.Exec(`UPDATE news_sources SET name=?,url=?,category=?,enabled=?,refresh_interval_minutes=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		strings.TrimSpace(name), strings.TrimSpace(rawURL), strings.TrimSpace(category), enabled, refreshMinutes, id)
	if err != nil { return Source{}, fmt.Errorf("update news source: %w", err) }
	n, _ := result.RowsAffected()
	if n == 0 { return Source{}, sql.ErrNoRows }
	return s.GetSource(id)
}

func (s Service) DeleteSource(id int64) error {
	result, err := s.DB.Exec(`DELETE FROM news_sources WHERE id=?`, id)
	if err != nil { return err }
	n, _ := result.RowsAffected()
	if n == 0 { return sql.ErrNoRows }
	return nil
}

func (s Service) TestSource(ctx context.Context, id int64) (string, error) {
	source, err := s.GetSource(id)
	if err != nil { return "", err }

	title, err := fetchFeed(ctx, s.client(), source.URL)
	now := time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		_, _ = s.DB.Exec(`UPDATE news_sources SET last_fetched_at=?,last_error=? WHERE id=?`, now, err.Error(), id)
		return "", err
	}
	_, _ = s.DB.Exec(`UPDATE news_sources SET last_fetched_at=?,last_success_at=?,last_error='' WHERE id=?`, now, now, id)
	return title, nil
}

func (s Service) client() *http.Client {
	if s.HTTPClient != nil { return s.HTTPClient }
	return &http.Client{Timeout: 15 * time.Second}
}

func validateSource(name, rawURL, category string, refreshMinutes int) error {
	name = strings.TrimSpace(name)
	rawURL = strings.TrimSpace(rawURL)
	category = strings.TrimSpace(category)
	if name == "" || len([]rune(name)) > 120 { return errors.New("source name must be 1-120 characters") }
	if category == "" || len([]rune(category)) > 80 { return errors.New("category must be 1-80 characters") }
	if refreshMinutes < 5 || refreshMinutes > 1440 { return errors.New("refresh interval must be between 5 and 1440 minutes") }
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("feed URL must be an http or https URL")
	}
	return nil
}

type feedDocument struct {
	XMLName xml.Name
	Title   string `xml:"channel>title"`
	AtomTitle string `xml:"title"`
}

func fetchFeed(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil { return "", err }
	req.Header.Set("User-Agent", "OtterLink-News/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9, */*;q=0.1")

	resp, err := client.Do(req)
	if err != nil { return "", fmt.Errorf("feed request failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("feed returned HTTP %d", resp.StatusCode)
	}
	body := io.LimitReader(resp.Body, 2*1024*1024)
	var doc feedDocument
	if err := xml.NewDecoder(body).Decode(&doc); err != nil { return "", fmt.Errorf("invalid RSS/Atom XML: %w", err) }
	title := strings.TrimSpace(doc.Title)
	if title == "" { title = strings.TrimSpace(doc.AtomTitle) }
	if title == "" { title = "Untitled feed" }
	return title, nil
}
