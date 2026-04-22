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

// ConflictingMatch represents another assignment that conflicts with this one
type ConflictingMatch struct {
	MatchID   int64  `json:"match_id"`
	EventName string `json:"event_name"`
	TeamName  string `json:"team_name"`
	StartTime string `json:"start_time"`
	RoleType  string `json:"role_type"`
}

// MatchForReferee represents a match with eligibility and availability for a specific referee
type MatchForReferee struct {
	ID                int64              `json:"id"`
	EventName         string             `json:"event_name"`
	TeamName          string             `json:"team_name"`
	AgeGroup          string             `json:"age_group"`
	MatchDate         string             `json:"match_date"`
	StartTime         string             `json:"start_time"`
	EndTime           string             `json:"end_time"`
	Location          string             `json:"location"`
	Description       *string            `json:"description"`
	Status            string             `json:"status"`
	EligibleRoles     []string           `json:"eligible_roles"`      // Roles the referee is eligible for
	IsAvailable       bool               `json:"is_available"`        // Has the referee marked as available?
	IsUnavailable     bool               `json:"is_unavailable"`      // Has the referee marked as unavailable?
	IsAssigned        bool               `json:"is_assigned"`         // Is the referee already assigned?
	AssignedRole      *string            `json:"assigned_role"`       // What role are they assigned to?
	Acknowledged      bool               `json:"acknowledged"`        // Has the referee acknowledged this assignment?
	AcknowledgedAt    *string            `json:"acknowledged_at"`     // When did they acknowledge?
	HasConflict       bool               `json:"has_conflict"`        // Does this assignment conflict with another?
	ConflictingMatches []ConflictingMatch `json:"conflicting_matches,omitempty"` // Details of conflicting matches
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

	// Get all upcoming, non-cancelled matches
	// Exclude days marked unavailable UNLESS the referee is already assigned to that match
	rows, err := db.Query(`
		SELECT
			m.id, m.event_name, m.team_name, m.age_group,
			m.match_date, m.start_time, m.end_time,
			m.location, m.description, m.status
		FROM matches m
		WHERE m.match_date >= CURRENT_DATE
		  AND m.status = 'active'
		  AND (
			-- Either the day is not marked unavailable
			NOT EXISTS (
				SELECT 1 FROM day_unavailability du
				WHERE du.referee_id = $1 AND du.unavailable_date = m.match_date
			)
			OR
			-- OR the referee is assigned to this match (always show assignments)
			EXISTS (
				SELECT 1 FROM match_roles mr
				WHERE mr.match_id = m.id AND mr.assigned_referee_id = $1
			)
		  )
		ORDER BY m.match_date ASC, m.start_time ASC, m.id ASC
	`, user.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Initialize as empty slice, not nil, so JSON encoding returns [] instead of null
	matches := []MatchForReferee{}

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

		// Check if referee has marked availability (tri-state: available=true, unavailable=false, no record=no preference)
		var availableFlag sql.NullBool
		err = db.QueryRow(`
			SELECT available
			FROM availability
			WHERE match_id = $1 AND referee_id = $2
		`, m.ID, user.ID).Scan(&availableFlag)

		if err != nil && err != sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}

		// Set IsAvailable and IsUnavailable based on tri-state logic
		// - availableFlag.Valid && availableFlag.Bool == true  → IsAvailable=true, IsUnavailable=false
		// - availableFlag.Valid && availableFlag.Bool == false → IsAvailable=false, IsUnavailable=true
		// - !availableFlag.Valid (no record)                   → IsAvailable=false, IsUnavailable=false (no preference)
		if availableFlag.Valid {
			m.IsAvailable = availableFlag.Bool
			m.IsUnavailable = !availableFlag.Bool
		} else {
			m.IsAvailable = false
			m.IsUnavailable = false
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

			// Check for scheduling conflicts with other assignments
			// Two time ranges overlap if: start1 < end2 AND start2 < end1
			conflictRows, err := db.Query(`
				SELECT
					m2.id, m2.event_name, m2.team_name,
					m2.start_time, mr2.role_type
				FROM matches m2
				JOIN match_roles mr2 ON mr2.match_id = m2.id
				WHERE mr2.assigned_referee_id = $1
				  AND m2.id != $2
				  AND m2.status = 'active'
				  AND m2.match_date = $3
				  AND m2.start_time < $5
				  AND m2.end_time > $4
			`, user.ID, m.ID, matchDate, m.StartTime, m.EndTime)

			if err != nil {
				// Don't fail the entire request on conflict check error, just log it
				fmt.Printf("Warning: Failed to check conflicts for match %d: %v\n", m.ID, err)
			} else {
				conflicts := []ConflictingMatch{}

				for conflictRows.Next() {
					var c ConflictingMatch
					err := conflictRows.Scan(&c.MatchID, &c.EventName, &c.TeamName, &c.StartTime, &c.RoleType)
					if err != nil {
						fmt.Printf("Warning: Failed to scan conflict: %v\n", err)
						continue
					}
					conflicts = append(conflicts, c)
				}
				conflictRows.Close()

				if len(conflicts) > 0 {
					m.HasConflict = true
					m.ConflictingMatches = conflicts
				}
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
		Available *bool `json:"available"` // Pointer to support tri-state: true=available, false=unavailable, null=no preference
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

	// Tri-state logic:
	// - available = true  → insert/update with available=true (mark available)
	// - available = false → insert/update with available=false (mark unavailable)
	// - available = null  → delete record (clear preference)
	if req.Available == nil {
		// Clear preference: delete the availability record
		_, err = db.Exec(`
			DELETE FROM availability
			WHERE match_id = $1 AND referee_id = $2
		`, matchID, user.ID)
	} else {
		// Insert or update availability record with explicit available flag
		_, err = db.Exec(`
			INSERT INTO availability (match_id, referee_id, available, created_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (match_id, referee_id)
			DO UPDATE SET available = $3, created_at = NOW()
		`, matchID, user.ID, *req.Available)
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
