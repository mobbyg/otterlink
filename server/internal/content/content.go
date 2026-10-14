package content

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Screen struct {
	ID          int64           `json:"id"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Published   bool            `json:"published"`
	StartAt     *string         `json:"start_at,omitempty"`
	EndAt       *string         `json:"end_at,omitempty"`
	Priority    int             `json:"priority"`
	Version     int64           `json:"version"`
	Content     json.RawMessage `json:"content"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

type Service struct {
	DB *sql.DB
}

type SaveRequest struct {
	Slug      string          `json:"slug"`
	Title     string          `json:"title"`
	Published bool            `json:"published"`
	StartAt   *string         `json:"start_at"`
	EndAt     *string         `json:"end_at"`
	Priority  int             `json:"priority"`
	Content   json.RawMessage `json:"content"`
}

func (s Service) List() ([]Screen, error) {
	rows, err := s.DB.Query(`SELECT id FROM content_screens ORDER BY priority DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	result := make([]Screen, 0, len(ids))
	for _, id := range ids {
		screen, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		result = append(result, screen)
	}
	return result, nil
}

func (s Service) Get(id int64) (Screen, error) {
	var screen Screen
	var startAt, endAt sql.NullString
	var published int
	var content string
	err := s.DB.QueryRow(`SELECT id, slug, title, published, start_at, end_at, priority, version, content_json, created_at, updated_at
		FROM content_screens WHERE id=?`, id).Scan(
		&screen.ID, &screen.Slug, &screen.Title, &published, &startAt, &endAt,
		&screen.Priority, &screen.Version, &content, &screen.CreatedAt, &screen.UpdatedAt,
	)
	if err != nil {
		return Screen{}, err
	}
	screen.Published = published != 0
	if startAt.Valid {
		screen.StartAt = &startAt.String
	}
	if endAt.Valid {
		screen.EndAt = &endAt.String
	}
	screen.Content = json.RawMessage(content)
	return screen, nil
}

func (s Service) Active(slug string, now time.Time) (Screen, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Screen{}, errors.New("screen slug is required")
	}

	rows, err := s.DB.Query(`SELECT id FROM content_screens
		WHERE slug=? AND published=1
		ORDER BY priority DESC, id DESC`, slug)
	if err != nil {
		return Screen{}, err
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return Screen{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Screen{}, err
	}
	if err := rows.Close(); err != nil {
		return Screen{}, err
	}

	for _, id := range ids {
		screen, err := s.Get(id)
		if err != nil {
			return Screen{}, err
		}
		if activeAt(screen, now) {
			return screen, nil
		}
	}
	return Screen{}, sql.ErrNoRows
}

func (s Service) Save(req SaveRequest) (Screen, error) {
	if err := validate(req); err != nil {
		return Screen{}, err
	}

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(string(req.Content))

	if req.StartAt != nil {
		value := strings.TrimSpace(*req.StartAt)
		req.StartAt = &value
	}
	if req.EndAt != nil {
		value := strings.TrimSpace(*req.EndAt)
		req.EndAt = &value
	}

	_, err := s.DB.Exec(`INSERT INTO content_screens
		(slug, title, published, start_at, end_at, priority, version, content_json)
		VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, 1, ?)
		ON CONFLICT(slug) DO UPDATE SET
		title=excluded.title,
		published=excluded.published,
		start_at=excluded.start_at,
		end_at=excluded.end_at,
		priority=excluded.priority,
		version=content_screens.version+1,
		content_json=excluded.content_json,
		updated_at=CURRENT_TIMESTAMP`,
		slug, title, req.Published, nullableString(req.StartAt), nullableString(req.EndAt),
		req.Priority, content)
	if err != nil {
		return Screen{}, fmt.Errorf("save screen: %w", err)
	}

	// LastInsertId is not reliable for an upsert that took the UPDATE path.
	// Resolve the screen by its unique slug so both insert and update return
	// the actual row that was saved.
	var id int64
	if err := s.DB.QueryRow(`SELECT id FROM content_screens WHERE slug=?`, slug).Scan(&id); err != nil {
		return Screen{}, fmt.Errorf("find saved screen: %w", err)
	}
	return s.Get(id)
}

func (s Service) Delete(id int64) error {
	result, err := s.DB.Exec(`DELETE FROM content_screens WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func validate(req SaveRequest) error {
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" || len([]rune(slug)) > 64 {
		return errors.New("screen slug must be 1-64 characters")
	}
	for _, r := range slug {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return errors.New("screen slug may contain only lowercase letters, numbers, '-' and '_'")
		}
	}
	if strings.TrimSpace(req.Title) == "" || len([]rune(req.Title)) > 160 {
		return errors.New("screen title must be 1-160 characters")
	}
	if len(req.Content) == 0 || len(req.Content) > 256*1024 || !json.Valid(req.Content) {
		return errors.New("screen content must be valid JSON no larger than 256 KB")
	}
	if req.Priority < -100000 || req.Priority > 100000 {
		return errors.New("screen priority is out of range")
	}
	if err := validateTime(req.StartAt); err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}
	if err := validateTime(req.EndAt); err != nil {
		return fmt.Errorf("invalid end time: %w", err)
	}
	if req.StartAt != nil && req.EndAt != nil && strings.TrimSpace(*req.StartAt) != "" && strings.TrimSpace(*req.EndAt) != "" {
		start, _ := time.Parse(time.RFC3339, strings.TrimSpace(*req.StartAt))
		end, _ := time.Parse(time.RFC3339, strings.TrimSpace(*req.EndAt))
		if end.Before(start) {
			return errors.New("screen end time must not be before its start")
		}
	}
	return nil
}

func validateTime(value *string) error {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(*value)); err != nil {
		return errors.New("must use RFC3339")
	}
	return nil
}

func activeAt(screen Screen, now time.Time) bool {
	if !screen.Published {
		return false
	}
	if screen.StartAt != nil && strings.TrimSpace(*screen.StartAt) != "" {
		start, err := time.Parse(time.RFC3339, strings.TrimSpace(*screen.StartAt))
		if err != nil || now.Before(start) {
			return false
		}
	}
	if screen.EndAt != nil && strings.TrimSpace(*screen.EndAt) != "" {
		end, err := time.Parse(time.RFC3339, strings.TrimSpace(*screen.EndAt))
		if err != nil || !now.Before(end) {
			return false
		}
	}
	return true
}

func nullableString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
