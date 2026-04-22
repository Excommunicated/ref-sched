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

// AssignmentRequest represents a request to assign or remove a referee
type AssignmentRequest struct {
	RefereeID *int64 `json:"referee_id"` // null to remove assignment
}

// assignRefereeHandler assigns or removes a referee from a role slot
func assignRefereeHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := r.Context().Value(userContextKey).(*User)

	vars := mux.Vars(r)
	matchID, err := strconv.ParseInt(vars["match_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	roleType := vars["role_type"]
	if roleType != "center" && roleType != "assistant_1" && roleType != "assistant_2" {
		http.Error(w, "Invalid role type", http.StatusBadRequest)
		return
	}

	var req AssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify match exists
	var matchExists bool
	err = db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM matches WHERE id = $1 AND status = 'active')
	`, matchID).Scan(&matchExists)

	if err != nil || !matchExists {
		http.Error(w, "Match not found or not active", http.StatusNotFound)
		return
	}

	// Check if role slot exists
	var roleID int64
	var currentRefereeID sql.NullInt64

	err = db.QueryRow(`
		SELECT id, assigned_referee_id
		FROM match_roles
		WHERE match_id = $1 AND role_type = $2
	`, matchID, roleType).Scan(&roleID, &currentRefereeID)

	if err == sql.ErrNoRows {
		http.Error(w, "Role slot not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// If assigning (not removing)
	if req.RefereeID != nil {
		// Verify referee exists and is active
		var refereeExists bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM users
				WHERE id = $1
				  AND (role = 'referee' OR role = 'assignor')
				  AND status = 'active'
			)
		`, *req.RefereeID).Scan(&refereeExists)

		if err != nil || !refereeExists {
			http.Error(w, "Referee not found or not active", http.StatusBadRequest)
			return
		}

		// TODO: Check eligibility (optional for v1, can assign anyone)
		// TODO: Check for double-booking conflicts (Story 5.4)
	}

	// Update assignment
	var newRefereeID sql.NullInt64
	if req.RefereeID != nil {
		newRefereeID = sql.NullInt64{Int64: *req.RefereeID, Valid: true}
	} else {
		newRefereeID = sql.NullInt64{Valid: false}
	}

	_, err = db.Exec(`
		UPDATE match_roles
		SET assigned_referee_id = $1
		WHERE id = $2
	`, newRefereeID, roleID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update assignment: %v", err), http.StatusInternalServerError)
		return
	}

	// Log assignment history
	action := "unassigned"
	if req.RefereeID != nil {
		if currentRefereeID.Valid {
			action = "reassigned"
		} else {
			action = "assigned"
		}
	}

	var oldRefereeID sql.NullInt64
	if currentRefereeID.Valid {
		oldRefereeID = currentRefereeID
	}

	var newRefereeIDForLog sql.NullInt64
	if req.RefereeID != nil {
		newRefereeIDForLog = sql.NullInt64{Int64: *req.RefereeID, Valid: true}
	}

	_, err = db.Exec(`
		INSERT INTO assignment_history (
			match_id, role_type, old_referee_id, new_referee_id,
			action, actor_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, matchID, roleType, oldRefereeID, newRefereeIDForLog, action, currentUser.ID)

	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to log assignment history: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"action":  action,
	})
}

// getConflictingAssignmentsHandler checks if a referee has conflicting assignments
func getConflictingAssignmentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, err := strconv.ParseInt(vars["match_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	refereeIDStr := r.URL.Query().Get("referee_id")
	if refereeIDStr == "" {
		http.Error(w, "referee_id query parameter required", http.StatusBadRequest)
		return
	}

	refereeID, err := strconv.ParseInt(refereeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid referee_id", http.StatusBadRequest)
		return
	}

	// Get match time window
	var matchStart, matchEnd time.Time
	err = db.QueryRow(`
		SELECT
			match_date + start_time::interval,
			match_date + end_time::interval
		FROM matches
		WHERE id = $1
	`, matchID).Scan(&matchStart, &matchEnd)

	if err == sql.ErrNoRows {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Find overlapping assignments
	rows, err := db.Query(`
		SELECT
			m.id, m.event_name, m.team_name,
			m.match_date, m.start_time, mr.role_type
		FROM matches m
		JOIN match_roles mr ON mr.match_id = m.id
		WHERE mr.assigned_referee_id = $1
		  AND m.id != $2
		  AND m.status = 'active'
		  AND (
			(m.match_date + m.start_time::interval, m.match_date + m.end_time::interval)
			OVERLAPS
			($3::timestamp, $4::timestamp)
		  )
	`, refereeID, matchID, matchStart, matchEnd)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Conflict struct {
		MatchID    int64  `json:"match_id"`
		EventName  string `json:"event_name"`
		TeamName   string `json:"team_name"`
		MatchDate  string `json:"match_date"`
		StartTime  string `json:"start_time"`
		RoleType   string `json:"role_type"`
	}

	conflicts := []Conflict{}

	for rows.Next() {
		var c Conflict
		var matchDate time.Time

		err := rows.Scan(
			&c.MatchID, &c.EventName, &c.TeamName,
			&matchDate, &c.StartTime, &c.RoleType,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Scan error: %v", err), http.StatusInternalServerError)
			return
		}

		c.MatchDate = matchDate.Format("2006-01-02")
		conflicts = append(conflicts, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"has_conflict": len(conflicts) > 0,
		"conflicts":    conflicts,
	})
}
