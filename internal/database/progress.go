package database

import "strings"

// UserProgress represents a user's gamification progress
type UserProgress struct {
	XP                   int      `json:"xp"`
	Level                int      `json:"level"`
	Streak               int      `json:"streak"`
	TotalReviews         int      `json:"total_reviews"`
	CardsLearned         int      `json:"cards_learned"`
	UnlockedAchievements []string `json:"unlocked_achievements"`
	LastReviewAt         int64    `json:"last_review_at"`
}

// GetUserProgress retrieves progress for a user
func GetUserProgress(userId int) (*UserProgress, error) {
	query := `
		SELECT xp, level, streak, total_reviews, cards_learned, 
		       unlocked_achievements, last_review_at
		FROM users WHERE id = ?
	`
	
	var progress UserProgress
	var achievements string
	var lastReview *int64
	
	err := DB.QueryRow(query, userId).Scan(
		&progress.XP,
		&progress.Level,
		&progress.Streak,
		&progress.TotalReviews,
		&progress.CardsLearned,
		&achievements,
		&lastReview,
	)
	
	if err != nil {
		return nil, err
	}
	
	if achievements != "" {
		progress.UnlockedAchievements = strings.Split(achievements, ",")
	}
	if lastReview != nil {
		progress.LastReviewAt = *lastReview
	}
	
	return &progress, nil
}

// UpdateUserProgress updates a user's progress
func UpdateUserProgress(userId int, xp, level, streak int) error {
	query := `
		UPDATE users 
		SET xp = ?, level = ?, streak = ?
		WHERE id = ?
	`
	_, err := DB.Exec(query, xp, level, streak, userId)
	return err
}

// SaveUnlockedAchievements saves achievement IDs for a user
func SaveUnlockedAchievements(userId int, achievements []string) error {
	achievementStr := strings.Join(achievements, ",")
	query := `UPDATE users SET unlocked_achievements = ? WHERE id = ?`
	_, err := DB.Exec(query, achievementStr, userId)
	return err
}

// AddUserXP increments a user's XP by the given amount
func AddUserXP(userId int, xpToAdd int) error {
	query := `UPDATE users SET xp = COALESCE(xp, 0) + ? WHERE id = ?`
	_, err := DB.Exec(query, xpToAdd, userId)
	return err
}
