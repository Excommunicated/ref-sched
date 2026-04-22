package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// EligibleReferee represents a referee with computed eligibility for a match role
type EligibleReferee struct {
	ID             int64   `json:"id"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	Email          string  `json:"email"`
	Grade          *string `json:"grade"` // Junior, Mid, Senior, or null
	DateOfBirth    *string `json:"date_of_birth"`
	Certified      bool    `json:"certified"`
	CertExpiry     *string `json:"cert_expiry"`
	AgeAtMatch     *int    `json:"age_at_match"`      // computed age on match date
	IsEligible     bool    `json:"is_eligible"`       // overall eligibility for this role
	IneligibleReason *string `json:"ineligible_reason"` // why not eligible, if applicable
	IsAvailable    bool    `json:"is_available"`      // has the referee marked availability for this match
}

// getEligibleRefereesHandler returns all referees with eligibility status for a specific match and role
func getEligibleRefereesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	roleType := r.URL.Query().Get("role")
	if roleType == "" {
		roleType = "center" // default to center role
	}

	// Validate role type
	if roleType != "center" && roleType != "assistant_1" && roleType != "assistant_2" {
		http.Error(w, "Invalid role type. Must be: center, assistant_1, or assistant_2", http.StatusBadRequest)
		return
	}

	// Get match details
	var match struct {
		ID        int64
		AgeGroup  string
		MatchDate time.Time
	}

	err = db.QueryRow(`
		SELECT id, age_group, match_date
		FROM matches
		WHERE id = $1 AND status = 'active'
	`, matchID).Scan(&match.ID, &match.AgeGroup, &match.MatchDate)

	if err == sql.ErrNoRows {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all active referees (including assignors who have filled out their profile)
	// with their availability status for this match
	rows, err := db.Query(`
		SELECT
			u.id, u.first_name, u.last_name, u.email, u.grade,
			u.date_of_birth, u.certified, u.cert_expiry,
			COALESCE(a.match_id IS NOT NULL, false) as is_available
		FROM users u
		LEFT JOIN availability a ON a.referee_id = u.id AND a.match_id = $1
		WHERE (u.role = 'referee' OR u.role = 'assignor')
		  AND u.status = 'active'
		  AND u.first_name IS NOT NULL
		  AND u.last_name IS NOT NULL
		  AND u.date_of_birth IS NOT NULL
		ORDER BY
			CASE WHEN a.match_id IS NOT NULL THEN 0 ELSE 1 END,
			u.last_name, u.first_name
	`, matchID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var referees []EligibleReferee

	for rows.Next() {
		var ref EligibleReferee
		var dob, certExpiry sql.NullTime
		var grade sql.NullString

		err := rows.Scan(
			&ref.ID, &ref.FirstName, &ref.LastName, &ref.Email, &grade,
			&dob, &ref.Certified, &certExpiry, &ref.IsAvailable,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Scan error: %v", err), http.StatusInternalServerError)
			return
		}

		// Convert nullable fields
		if grade.Valid {
			ref.Grade = &grade.String
		}

		if dob.Valid {
			dobStr := dob.Time.Format("2006-01-02")
			ref.DateOfBirth = &dobStr

			// Calculate age at match date
			age := calculateAgeAtDate(dob.Time, match.MatchDate)
			ref.AgeAtMatch = &age
		}

		if certExpiry.Valid {
			certExpiryStr := certExpiry.Time.Format("2006-01-02")
			ref.CertExpiry = &certExpiryStr
		}

		// Determine eligibility
		ref.IsEligible, ref.IneligibleReason = checkEligibility(
			match.AgeGroup,
			roleType,
			match.MatchDate,
			dob.Time,
			ref.Certified,
			certExpiry,
		)

		referees = append(referees, ref)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(referees)
}

// calculateAgeAtDate calculates age in years at a specific date
func calculateAgeAtDate(birthDate, targetDate time.Time) int {
	age := targetDate.Year() - birthDate.Year()

	// Adjust if birthday hasn't occurred yet this year
	if targetDate.Month() < birthDate.Month() ||
		(targetDate.Month() == birthDate.Month() && targetDate.Day() < birthDate.Day()) {
		age--
	}

	return age
}

// checkEligibility determines if a referee is eligible for a specific role on a match
// Returns (isEligible, ineligibleReason)
func checkEligibility(
	ageGroup string,
	roleType string,
	matchDate time.Time,
	dob time.Time,
	certified bool,
	certExpiry sql.NullTime,
) (bool, *string) {
	// Parse age group to get the age number (e.g., "U12" -> 12)
	var ageGroupNum int
	_, err := fmt.Sscanf(ageGroup, "U%d", &ageGroupNum)
	if err != nil {
		reason := "Invalid age group format"
		return false, &reason
	}

	age := calculateAgeAtDate(dob, matchDate)

	// Rule 1: U10 and younger - age-based eligibility for all roles
	if ageGroupNum <= 10 {
		requiredAge := ageGroupNum + 1
		if age < requiredAge {
			reason := fmt.Sprintf("Must be at least %d years old (currently %d)", requiredAge, age)
			return false, &reason
		}
		return true, nil
	}

	// Rule 2: U12 and older - Center Referee requires certification
	if roleType == "center" {
		if !certified {
			reason := "Certification required for center referee role on U12+ matches"
			return false, &reason
		}

		// Check if certification is valid on match date
		if !certExpiry.Valid {
			reason := "Certification expiry date missing"
			return false, &reason
		}

		// Cert must be valid on match date (expires AFTER match date)
		if !certExpiry.Time.After(matchDate) {
			reason := fmt.Sprintf("Certification expires before match date (%s)", certExpiry.Time.Format("2006-01-02"))
			return false, &reason
		}

		return true, nil
	}

	// Rule 3: U12 and older - Assistant Referee has no restrictions
	if roleType == "assistant_1" || roleType == "assistant_2" {
		return true, nil
	}

	// Should not reach here
	reason := "Unknown role type"
	return false, &reason
}
