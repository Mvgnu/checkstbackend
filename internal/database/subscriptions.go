package database

import (
	"time"
)

// Subscription represents a user's subscription record
type Subscription struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	ProductID     string    `json:"product_id"`
	TransactionID string    `json:"transaction_id"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	IsActive      bool      `json:"is_active"`
}

// SaveSubscription creates or updates a subscription record
func SaveSubscription(userID int, productID, transactionID string, expiresAt time.Time) error {
	// Check if subscription already exists for this transaction
	var existingID int
	err := DB.QueryRow(`SELECT id FROM subscriptions WHERE transaction_id = ?`, transactionID).Scan(&existingID)
	
	if err == nil {
		// Update existing
		_, err = DB.Exec(`
			UPDATE subscriptions 
			SET expires_at = ?, is_active = 1 
			WHERE id = ?
		`, expiresAt, existingID)
		return err
	}
	
	// Insert new
	_, err = DB.Exec(`
		INSERT INTO subscriptions (user_id, product_id, transaction_id, expires_at, is_active, created_at)
		VALUES (?, ?, ?, ?, 1, ?)
	`, userID, productID, transactionID, expiresAt, time.Now())
	
	return err
}

// GetActiveSubscription returns the user's active subscription if any
func GetActiveSubscription(userID int) (*Subscription, error) {
	var s Subscription
	err := DB.QueryRow(`
		SELECT id, user_id, product_id, transaction_id, expires_at, created_at, is_active
		FROM subscriptions
		WHERE user_id = ? AND is_active = 1 AND expires_at > ?
		ORDER BY expires_at DESC
		LIMIT 1
	`, userID, time.Now()).Scan(&s.ID, &s.UserID, &s.ProductID, &s.TransactionID, &s.ExpiresAt, &s.CreatedAt, &s.IsActive)
	
	if err != nil {
		return nil, err
	}
	
	return &s, nil
}

// ExpireSubscription deactivates a subscription by transaction ID
func ExpireSubscription(transactionID string) error {
	_, err := DB.Exec(`
		UPDATE subscriptions 
		SET is_active = 0 
		WHERE transaction_id = ?
	`, transactionID)
	return err
}

// UpdateUserSubscription updates the user's subscription_status field
func UpdateUserSubscription(userID int, status string) error {
	_, err := DB.Exec(`
		UPDATE users 
		SET subscription_status = ? 
		WHERE id = ?
	`, status, userID)
	return err
}

// CheckAndExpireSubscriptions is a cleanup job to expire old subscriptions
func CheckAndExpireSubscriptions() error {
	// Mark expired subscriptions as inactive
	_, err := DB.Exec(`
		UPDATE subscriptions 
		SET is_active = 0 
		WHERE expires_at < ? AND is_active = 1
	`, time.Now())
	
	if err != nil {
		return err
	}
	
	// Update users who no longer have active subscriptions to free
	_, err = DB.Exec(`
		UPDATE users 
		SET subscription_status = 'free' 
		WHERE id NOT IN (
			SELECT DISTINCT user_id FROM subscriptions WHERE is_active = 1
		) AND subscription_status = 'pro'
	`)
	
	return err
}
