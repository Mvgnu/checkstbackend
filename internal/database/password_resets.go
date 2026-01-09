package database

import (
	"time"
)

// SaveResetCode saves a password reset code for an email, expiring in 10 minutes
func SaveResetCode(email, code string) error {
	// Delete any existing codes for this email to prevent clutter
	_, err := DB.Exec("DELETE FROM password_resets WHERE email = ?", email)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(10 * time.Minute)
	_, err = DB.Exec("INSERT INTO password_resets (email, code, expires_at) VALUES (?, ?, ?)", email, code, expiresAt)
	return err
}

// VerifyAndConsumeResetCode checks if the code is valid and not expired. 
// If valid, it deletes the code (consumes it) and returns true.
func VerifyAndConsumeResetCode(email, code string) bool {
	var expiresAt time.Time
	var storedCode string

	err := DB.QueryRow("SELECT code, expires_at FROM password_resets WHERE email = ?", email).Scan(&storedCode, &expiresAt)
	if err != nil {
		return false // Not found
	}

	if time.Now().After(expiresAt) {
		// Clean up expired
		DB.Exec("DELETE FROM password_resets WHERE email = ?", email)
		return false
	}

	if storedCode != code {
		return false // Wrong code
	}

	// Consume the code so it cannot be reused
	DB.Exec("DELETE FROM password_resets WHERE email = ?", email)
	return true
}

// UpdateUserPassword updates the password hash for a user by email
func UpdateUserPassword(email, hash string) error {
	_, err := DB.Exec("UPDATE users SET password_hash = ? WHERE email = ?", hash, email)
	return err
}
