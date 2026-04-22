package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// ProfileUpdateRequest represents the profile update payload
type ProfileUpdateRequest struct {
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	DateOfBirth *string `json:"date_of_birth"` // YYYY-MM-DD format
	Certified   bool    `json:"certified"`
	CertExpiry  *string `json:"cert_expiry"` // YYYY-MM-DD format
}

// updateProfileHandler allows a user to update their own profile
func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate date of birth
	var dob *time.Time
	if req.DateOfBirth != nil && *req.DateOfBirth != "" {
		parsedDOB, err := time.Parse("2006-01-02", *req.DateOfBirth)
		if err != nil {
			http.Error(w, "Invalid date of birth format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		if parsedDOB.After(time.Now()) {
			http.Error(w, "Date of birth cannot be in the future", http.StatusBadRequest)
			return
		}
		dob = &parsedDOB
	}

	// Validate certification expiry
	var certExpiry *time.Time
	if req.Certified {
		if req.CertExpiry == nil || *req.CertExpiry == "" {
			http.Error(w, "Certification expiry date is required when certified", http.StatusBadRequest)
			return
		}
		parsedExpiry, err := time.Parse("2006-01-02", *req.CertExpiry)
		if err != nil {
			http.Error(w, "Invalid certification expiry format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		if parsedExpiry.Before(time.Now()) {
			http.Error(w, "Certification expiry must be in the future", http.StatusBadRequest)
			return
		}
		certExpiry = &parsedExpiry
	}

	// Update user profile
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, date_of_birth = $3,
		    certified = $4, cert_expiry = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING id, google_id, email, name, role, status, first_name, last_name,
		          date_of_birth, certified, cert_expiry, grade, created_at, updated_at
	`

	updatedUser := &User{}
	err := db.QueryRow(
		query,
		req.FirstName,
		req.LastName,
		dob,
		req.Certified,
		certExpiry,
		user.ID,
	).Scan(
		&updatedUser.ID,
		&updatedUser.GoogleID,
		&updatedUser.Email,
		&updatedUser.Name,
		&updatedUser.Role,
		&updatedUser.Status,
		&updatedUser.FirstName,
		&updatedUser.LastName,
		&updatedUser.DateOfBirth,
		&updatedUser.Certified,
		&updatedUser.CertExpiry,
		&updatedUser.Grade,
		&updatedUser.CreatedAt,
		&updatedUser.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
}

// getProfileHandler returns the current user's full profile
func getProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	// Get fresh data from database
	freshUser, err := getUserByID(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(freshUser)
}
