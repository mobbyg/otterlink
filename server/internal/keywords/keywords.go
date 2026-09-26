package keywords

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Target struct {
	Type  string `json:"type"`
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

type Keyword struct {
	Keyword     string   `json:"keyword"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Targets     []Target `json:"targets"`
}

type Service struct{ DB *sql.DB }

func NewService(db *sql.DB) Service { return Service{DB: db} }

func normalize(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func validTarget(t Target) bool {
	return (t.Type == "chat" || t.Type == "bulletin" || t.Type == "event") && t.ID > 0 && strings.TrimSpace(t.Label) != ""
}

func (s Service) Resolve(value string) (Keyword, error) {
	key := normalize(value)
	if key == "" || len([]rune(key)) > 32 {
		return Keyword{}, sql.ErrNoRows
	}
	var k Keyword
	err := s.DB.QueryRow(`SELECT keyword, display_name, description FROM service_keywords WHERE keyword = ?`, key).
		Scan(&k.Keyword, &k.DisplayName, &k.Description)
	if err != nil {
		return Keyword{}, err
	}
	k.Targets, err = s.targets(key)
	return k, err
}

func (s Service) targets(key string) ([]Target, error) {
	rows, err := s.DB.Query(`SELECT target_type, target_id, label FROM service_keyword_targets WHERE keyword = ? ORDER BY id`, key)
	if err != nil { return nil, err }
	defer rows.Close()
	targets := make([]Target, 0)
	for rows.Next() {
		var t Target
		if err := rows.Scan(&t.Type, &t.ID, &t.Label); err != nil { return nil, err }
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

func (s Service) List() ([]Keyword, error) {
	rows, err := s.DB.Query(`SELECT keyword FROM service_keywords ORDER BY keyword`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []Keyword
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil { return nil, err }
		k, err := s.Resolve(key)
		if err != nil { return nil, err }
		result = append(result, k)
	}
	return result, rows.Err()
}

func (s Service) Upsert(k Keyword) error {
	k.Keyword = normalize(k.Keyword)
	k.DisplayName = strings.TrimSpace(k.DisplayName)
	k.Description = strings.TrimSpace(k.Description)
	if k.Keyword == "" || len([]rune(k.Keyword)) > 32 {
		return errors.New("keyword must be 1-32 characters")
	}
	if k.DisplayName == "" || len([]rune(k.DisplayName)) > 80 {
		return errors.New("display name must be 1-80 characters")
	}
	if len([]rune(k.Description)) > 500 {
		return errors.New("description exceeds 500 characters")
	}
	seen := map[string]bool{}
	for _, t := range k.Targets {
		if !validTarget(t) || seen[fmt.Sprintf("%s:%d", t.Type, t.ID)] {
			return errors.New("invalid or duplicate keyword target")
		}
		seen[fmt.Sprintf("%s:%d", t.Type, t.ID)] = true
	}

	tx, err := s.DB.Begin()
	if err != nil { return err }
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO service_keywords(keyword,display_name,description) VALUES(?,?,?)
		ON CONFLICT(keyword) DO UPDATE SET display_name=excluded.display_name, description=excluded.description`,
		k.Keyword, k.DisplayName, k.Description); err != nil { return fmt.Errorf("save keyword: %w", err) }
	if _, err = tx.Exec(`DELETE FROM service_keyword_targets WHERE keyword=?`, k.Keyword); err != nil { return err }
	for _, t := range k.Targets {
		if _, err = tx.Exec(`INSERT INTO service_keyword_targets(keyword,target_type,target_id,label) VALUES(?,?,?,?)`,
			k.Keyword, t.Type, t.ID, strings.TrimSpace(t.Label)); err != nil { return fmt.Errorf("save keyword target: %w", err) }
	}
	return tx.Commit()
}

func (s Service) Delete(value string) error {
	key := normalize(value)
	result, err := s.DB.Exec(`DELETE FROM service_keywords WHERE keyword=?`, key)
	if err != nil { return err }
	n, err := result.RowsAffected()
	if err != nil { return err }
	if n == 0 { return sql.ErrNoRows }
	return nil
}
