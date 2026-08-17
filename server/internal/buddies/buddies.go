package buddies

import (
	"database/sql"
	"errors"
	"strings"
)

type Buddy struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type Service struct {
	DB *sql.DB
}

func (s Service) Add(userID int64, username string) (Buddy, error) {
	username = strings.TrimSpace(username)
	var buddy Buddy
	var buddyStatus string
	err := s.DB.QueryRow(`SELECT id, username, display_name, status FROM users WHERE username = ? COLLATE NOCASE`, username).
		Scan(&buddy.ID, &buddy.Username, &buddy.DisplayName, &buddyStatus)
	if err == sql.ErrNoRows {
		return Buddy{}, errors.New("user not found")
	}
	if err != nil {
		return Buddy{}, err
	}
	if buddy.ID == userID {
		return Buddy{}, errors.New("cannot add yourself as a buddy")
	}
	buddy.Status = buddyStatus
	if _, err := s.DB.Exec(`INSERT INTO buddies (user_id, buddy_id) VALUES (?, ?)`, userID, buddy.ID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Buddy{}, errors.New("buddy already exists")
		}
		return Buddy{}, err
	}
	return buddy, nil
}

func (s Service) Remove(userID int64, username string) error {
	result, err := s.DB.Exec(`DELETE FROM buddies WHERE user_id = ? AND buddy_id = (SELECT id FROM users WHERE username = ? COLLATE NOCASE)`, userID, strings.TrimSpace(username))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("buddy not found")
	}
	return nil
}

func (s Service) List(userID int64) ([]Buddy, error) {
	rows, err := s.DB.Query(`SELECT u.id, u.username, u.display_name, u.status FROM buddies b JOIN users u ON u.id = b.buddy_id WHERE b.user_id = ? ORDER BY u.username COLLATE NOCASE`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Buddy, 0)
	for rows.Next() {
		var buddy Buddy
		if err := rows.Scan(&buddy.ID, &buddy.Username, &buddy.DisplayName, &buddy.Status); err != nil {
			return nil, err
		}
		result = append(result, buddy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
