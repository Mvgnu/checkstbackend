package database

import "database/sql"

// LeaderboardEntry represents a user's position on the leaderboard
type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	XP       int    `json:"xp"`
	Level    int    `json:"level"`
	Streak   int    `json:"streak"`
}

// GetLeaderboard returns top users by XP for a given time period
func GetLeaderboard(period string, limit int) ([]LeaderboardEntry, error) {
	// For now, return all-time leaderboard from users table
	// In production, would query from a separate user_xp_history table
	
	query := `
		SELECT id, username, xp, level, streak
		FROM users
		ORDER BY xp DESC
		LIMIT ?
	`
	
	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var entries []LeaderboardEntry
	rank := 0
	for rows.Next() {
		rank++
		var e LeaderboardEntry
		var xp, level, streak sql.NullInt64
		
		if err := rows.Scan(&e.UserID, &e.Username, &xp, &level, &streak); err != nil {
			continue
		}
		
		e.Rank = rank
		e.XP = int(xp.Int64)
		e.Level = int(level.Int64)
		e.Streak = int(streak.Int64)
		
		entries = append(entries, e)
	}
	
	return entries, nil
}

// GetGroupLeaderboard returns top users within a specific group
func GetGroupLeaderboard(groupId string, limit int) ([]LeaderboardEntry, error) {
	query := `
		SELECT u.id, u.username, u.xp, u.level, u.streak
		FROM users u
		INNER JOIN group_members gm ON u.id = gm.user_id
		WHERE gm.group_id = ?
		ORDER BY u.xp DESC
		LIMIT ?
	`
	
	rows, err := DB.Query(query, groupId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var entries []LeaderboardEntry
	rank := 0
	for rows.Next() {
		rank++
		var e LeaderboardEntry
		var xp, level, streak sql.NullInt64
		
		if err := rows.Scan(&e.UserID, &e.Username, &xp, &level, &streak); err != nil {
			continue
		}
		
		e.Rank = rank
		e.XP = int(xp.Int64)
		e.Level = int(level.Int64)
		e.Streak = int(streak.Int64)
		
		entries = append(entries, e)
	}
	
	return entries, nil
}
