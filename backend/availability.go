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

// MatchForReferee represents a match with eligibility and availability for a specific referee
type MatchForReferee struct {
	ID              int64   `json:"id"`
	EventName       string  `json:"event_name"`
	TeamName        string  `json:"team_name"`
	AgeGroup        string  `json:"age_group"`
	MatchDate       string  `json:"match_date"`
	StartTime       string  `json:"start_time"`
	EndTime         string  `json:"end_time"`
	Location        string  `json:"location"`
	Description     *string `json:"description"`
	Status          string  `json:"status"`
	EligibleRoles   []string `json:"eligible_roles"`   // Roles the referee is eligible for
	IsAvailable     bool    `json:"is_available"`      // Has the referee marked availability?
	IsAssigned      bool    `json:"is_assigned"`       // Is the referee already assigned?
	AssignedRole    *string `json:"assigned_role"`     // What role are they assigned to?
	Acknowledged    bool    `json:"acknowledged"`      // Has the referee acknowledged this assignment?
	AcknowledgedAt  *string `json:"acknowledged_at"`   // When did they acknowledge?
}

// getEligibleMatchesForRefereeHandler returns all upcoming matches that the current referee is eligible for
func getEligibleMatchesForRefereeHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	// Check if user has completed their profile
	var hasProfile bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM users
			WHERE id = $1
			  AND first_name IS NOT NULL
			  AND last_name IS NOT NULL
			  AND date_of_birth IS NOT NULL
		)
	`, user.ID).Scan(&hasProfile)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	if !hasProfile {
		// Return empty list if profile incomplete
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]MatchForReferee{})
		return
	}

	// Get referee profile details for eligibility checking
	var referee struct {
		ID         int64
		DOB        time.Time
		Certified  bool
		CertExpiry sql.NullTime
	}

	err = db.QueryRow(`
		SELECT id, date_of_birth, certified, cert_expiry
		FROM users
		WHERE id = $1
	`, user.ID).Scan(&referee.ID, &referee.DOB, &referee.Certified, &referee.CertExpiry)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Get all upcoming, non-cancelled matches excluding days marked unavailable
	rows, err := db.Query(`
		SELECT
			m.id, m.event_name, m.team_name, m.age_group,
			m.match_date, m.start_time, m.end_time,
			m.location, m.description, m.status
		FROM matches m
		WHERE m.match_date >= CURRENT_DATE
		  AND m.status = 'active'
		  AND NOT EXISTS (
			SELECT 1 FROM day_unavailability du
			WHERE du.referee_id = $1 AND du.unavailable_date = m.match_date
		  )
		ORDER BY m.match_date, m.start_time
	`, user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var matches []MatchForReferee

	for rows.Next() {
		var m MatchForReferee
		var matchDate time.Time
		var description sql.NullString

		err := rows.Scan(
			&m.ID, &m.EventName, &m.TeamName, &m.AgeGroup,
			&matchDate, &m.StartTime, &m.EndTime,
			&m.Location, &description, &m.Status,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Scan error: %v", err), http.StatusInternalServerError)
			return
		}

		m.MatchDate = matchDate.Format("2006-01-02")

		if description.Valid {
			m.Description = &description.String
		}

		// Check eligibility for each role type
		eligibleRoles := []string{}

		// Check center role
		isEligible, _ := checkEligibility(
			m.AgeGroup, "center", matchDate,
			referee.DOB, referee.Certified, referee.CertExpiry,
		)
		if isEligible {
			eligibleRoles = append(eligibleRoles, "center")
		}

		// Check assistant roles (both use same logic)
		isEligible, _ = checkEligibility(
			m.AgeGroup, "assistant_1", matchDate,
			referee.DOB, referee.Certified, referee.CertExpiry,
		)
		if isEligible {
			eligibleRoles = append(eligibleRoles, "assistant")
		}

		// Skip this match if not eligible for any role
		if len(eligibleRoles) == 0 {
			continue
		}

		m.EligibleRoles = eligibleRoles

		// Check if referee has marked availability
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM availability
				WHERE match_id = $1 AND referee_id = $2
			)
		`, m.ID, user.ID).Scan(&m.IsAvailable)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}

		// Check if referee is already assigned to this match
		var assignedRole sql.NullString
		var acknowledged bool
		var acknowledgedAt sql.NullTime
		err = db.QueryRow(`
			SELECT role_type, acknowledged, acknowledged_at
			FROM match_roles
			WHERE match_id = $1 AND assigned_referee_id = $2
			LIMIT 1
		`, m.ID, user.ID).Scan(&assignedRole, &acknowledged, &acknowledgedAt)

		if err != nil && err != sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}

		if assignedRole.Valid {
			m.IsAssigned = true
			m.AssignedRole = &assignedRole.String
			m.Acknowledged = acknowledged
			if acknowledgedAt.Valid {
				ackTime := acknowledgedAt.Time.Format(time.RFC3339)
				m.AcknowledgedAt = &ackTime
			}
		}

		matches = append(matches, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}

// toggleAvailabilityHandler marks or unmarks a referee's availability for a match
func toggleAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	vars := mux.Vars(r)
	matchID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Available bool `json:"available"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify match exists and is active
	var matchExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM matches
			WHERE id = $1 AND status = 'active' AND match_date >= CURRENT_DATE
		)
	`, matchID).Scan(&matchExists)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	if !matchExists {
		http.Error(w, "Match not found or not available for marking", http.StatusNotFound)
		return
	}

	if req.Available {
		// Insert or update availability record
		_, err = db.Exec(`
			INSERT INTO availability (match_id, referee_id, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (match_id, referee_id) DO NOTHING
		`, matchID, user.ID)
	} else {
		// Remove availability record
		_, err = db.Exec(`
			DELETE FROM availability
			WHERE match_id = $1 AND referee_id = $2
		`, matchID, user.ID)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"available": req.Available,
	})
}
