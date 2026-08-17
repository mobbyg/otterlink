package buddies

import "database/sql"

// Watchers returns the users who currently have buddyID on their buddy lists.
func (s Service) Watchers(buddyID int64) ([]int64, error) {
	rows, err := s.DB.Query(`SELECT user_id FROM buddies WHERE buddy_id = ?`, buddyID)
	if err != nil { return nil, err }
	defer rows.Close()
	result := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil { return nil, err }
		result = append(result, userID)
	}
	if err := rows.Err(); err != nil && err != sql.ErrNoRows { return nil, err }
	return result, nil
}
