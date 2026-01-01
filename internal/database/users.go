package database

import (
	"database/sql"
)

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Username     string `json:"username"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	University   string `json:"university,omitempty"`
	Degree       string `json:"degree,omitempty"`
	SubscriptionStatus string `json:"subscription_status"`
	SubscriptionExpiry *string `json:"subscription_expiry,omitempty"`
}

func CreateUser(email, passwordHash, username string) (*User, error) {
	query := `INSERT INTO users (email, password_hash, username) VALUES (?, ?, ?)`
	result, err := DB.Exec(query, email, passwordHash, username)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           int(id),
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
	}, nil
}

func GetUserByEmail(email string) (*User, error) {
	query := `SELECT id, email, password_hash, username, avatar_url, university, degree, subscription_status, subscription_expiry FROM users WHERE email = ?`
	row := DB.QueryRow(query, email)

	var u User
	var avatarURL, university, degree, subscriptionExpiry sql.NullString
	var subscriptionStatus string

	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Username, &avatarURL, &university, &degree, &subscriptionStatus, &subscriptionExpiry)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}

	u.AvatarURL = avatarURL.String
	u.University = university.String
	u.Degree = degree.String
	u.SubscriptionStatus = subscriptionStatus
	if subscriptionExpiry.Valid {
		expiry := subscriptionExpiry.String
		u.SubscriptionExpiry = &expiry
	}

	return &u, nil
}

func UpdateUser(id int, avatarURL, university, degree string) error {
    query := `UPDATE users SET avatar_url = ?, university = ?, degree = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
    _, err := DB.Exec(query, avatarURL, university, degree, id)
    return err
}

func DeleteUser(id int) error {
    query := `DELETE FROM users WHERE id = ?`
    _, err := DB.Exec(query, id)
    return err
}

func GetUserByID(id int) (*User, error) {
    query := `SELECT id, email, password_hash, username, avatar_url, university, degree, subscription_status, subscription_expiry FROM users WHERE id = ?`
    row := DB.QueryRow(query, id)

    var u User
    var avatarURL, university, degree, subscriptionExpiry sql.NullString
	var subscriptionStatus string

    err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Username, &avatarURL, &university, &degree, &subscriptionStatus, &subscriptionExpiry)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil // Not found
        }
        return nil, err
    }

    if avatarURL.Valid {
        u.AvatarURL = avatarURL.String
    }
    if university.Valid {
        u.University = university.String
    }
    if degree.Valid {
        u.Degree = degree.String
    }
	u.SubscriptionStatus = subscriptionStatus
	if subscriptionExpiry.Valid {
		expiry := subscriptionExpiry.String
		u.SubscriptionExpiry = &expiry
	}

    return &u, nil
}
