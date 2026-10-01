package assets

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxUploadSize = 10 * 1024 * 1024
)

var allowedMIMETypes = map[string]bool{
	"image/gif":  true,
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/svg+xml": true,
}

type Asset struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	MIME      string `json:"mime"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Service struct {
	DB   *sql.DB
	Root string
}

func (s Service) List() ([]Asset, error) {
	rows, err := s.DB.Query(`SELECT id FROM content_assets ORDER BY id DESC`)
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
	result := make([]Asset, 0, len(ids))
	for _, id := range ids {
		asset, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		result = append(result, asset)
	}
	return result, nil
}

func (s Service) Get(id int64) (Asset, error) {
	var asset Asset
	err := s.DB.QueryRow(`SELECT id, name, mime, size, sha256, created_at, updated_at
		FROM content_assets WHERE id=?`, id).Scan(
		&asset.ID, &asset.Name, &asset.MIME, &asset.Size, &asset.SHA256,
		&asset.CreatedAt, &asset.UpdatedAt,
	)
	if err != nil {
		return Asset{}, err
	}
	return asset, nil
}

func (s Service) pathFor(asset Asset) string {
	return filepath.Join(s.Root, fmt.Sprintf("%d-%s", asset.ID, asset.SHA256))
}

func (s Service) Save(name, contentType string, size int64, src io.Reader) (Asset, error) {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return Asset{}, errors.New("asset name is required")
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if !allowedMIMETypes[contentType] {
		return Asset{}, errors.New("unsupported asset type")
	}
	if size < 1 || size > MaxUploadSize {
		return Asset{}, fmt.Errorf("asset must be between 1 byte and %d MB", MaxUploadSize/(1024*1024))
	}

	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return Asset{}, fmt.Errorf("create asset directory: %w", err)
	}

	temp, err := os.CreateTemp(s.Root, ".upload-*")
	if err != nil {
		return Asset{}, fmt.Errorf("create asset upload: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	hash := sha256.New()
	written, err := io.CopyN(io.MultiWriter(temp, hash), src, size)
	if err != nil {
		temp.Close()
		return Asset{}, fmt.Errorf("store asset upload: %w", err)
	}
	if written != size {
		temp.Close()
		return Asset{}, errors.New("asset upload size changed during upload")
	}
	if err := temp.Close(); err != nil {
		return Asset{}, fmt.Errorf("close asset upload: %w", err)
	}

	sum := hex.EncodeToString(hash.Sum(nil))
	result, err := s.DB.Exec(`INSERT INTO content_assets (name, mime, size, sha256) VALUES (?, ?, ?, ?)`,
		name, contentType, size, sum)
	if err != nil {
		return Asset{}, fmt.Errorf("save asset metadata: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Asset{}, fmt.Errorf("get asset id: %w", err)
	}

	asset, err := s.Get(id)
	if err != nil {
		return Asset{}, err
	}
	if err := os.Rename(tempName, s.pathFor(asset)); err != nil {
		_, _ = s.DB.Exec(`DELETE FROM content_assets WHERE id=?`, id)
		return Asset{}, fmt.Errorf("store asset file: %w", err)
	}
	return asset, nil
}

func (s Service) Open(id int64) (Asset, *os.File, error) {
	asset, err := s.Get(id)
	if err != nil {
		return Asset{}, nil, err
	}
	file, err := os.Open(s.pathFor(asset))
	if err != nil {
		return Asset{}, nil, err
	}
	return asset, file, nil
}

func (s Service) Delete(id int64) error {
	asset, err := s.Get(id)
	if err != nil {
		return err
	}
	if err := os.Remove(s.pathFor(asset)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove asset file: %w", err)
	}
	result, err := s.DB.Exec(`DELETE FROM content_assets WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func IsAllowedMIME(value string) bool {
	return allowedMIMETypes[strings.ToLower(strings.TrimSpace(value))]
}

func GuessMIME(name string) string {
	value := strings.ToLower(filepath.Ext(name))
	if value == ".jpg" {
		value = ".jpeg"
	}
	return mime.TypeByExtension(value)
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
