package main

import (
	"context"
	"database/sql"
	"time"
)

// User represents a user in the system
type User struct {
	ID          int64     `json:"id"`
	GoogleID    string    `json:"google_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	FirstName   *string   `json:"first_name,omitempty"`
	LastName    *string   `json:"last_name,omitempty"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	Certified   bool      `json:"certified"`
	CertExpiry  *time.Time `json:"cert_expiry,omitempty"`
	Grade       *string   `json:"grade,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type contextKey string

const userContextKey contextKey = "user"

func contextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// findOrCreateUser finds an existing user by Google ID or creates a new one
func findOrCreateUser(googleID, email, name string) (*User, error) {
	// Try to find existing user
	user, err := getUserByGoogleID(googleID)
	if err == nil {
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user with pending_referee role
	query := `
		INSERT INTO users (google_id, email, name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, google_id, email, name, role, status, created_at, updated_at
	`

	user = &User{}
	err = db.QueryRow(query, googleID, email, name, "pending_referee", "pending").Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return user, err
}

// getUserByGoogleID retrieves a user by their Google ID
func getUserByGoogleID(googleID string) (*User, error) {
	query := `
		SELECT id, google_id, email, name, role, status, first_name, last_name,
		       date_of_birth, certified, cert_expiry, grade, created_at, updated_at
		FROM users
		WHERE google_id = $1 AND status != 'removed'
	`

	user := &User{}
	err := db.QueryRow(query, googleID).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.Status,
		&user.FirstName,
		&user.LastName,
		&user.DateOfBirth,
		&user.Certified,
		&user.CertExpiry,
		&user.Grade,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return user, err
}

// getUserByID retrieves a user by their ID
func getUserByID(id int64) (*User, error) {
	query := `
		SELECT id, google_id, email, name, role, status, first_name, last_name,
		       date_of_birth, certified, cert_expiry, grade, created_at, updated_at
		FROM users
		WHERE id = $1 AND status IN ('active', 'pending')
	`

	user := &User{}
	err := db.QueryRow(query, id).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.Status,
		&user.FirstName,
		&user.LastName,
		&user.DateOfBirth,
		&user.Certified,
		&user.CertExpiry,
		&user.Grade,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return user, err
}
