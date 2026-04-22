package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// acknowledgeAssignmentHandler allows a referee to acknowledge their assignment
func acknowledgeAssignmentHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	// Only referees can acknowledge assignments
	if user.Role != "referee" && user.Role != "assignor" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	matchID, err := strconv.ParseInt(vars["match_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	// Verify the referee is actually assigned to this match
	var roleType string
	err = db.QueryRow(`
		SELECT role_type
		FROM match_roles
		WHERE match_id = $1 AND assigned_referee_id = $2
	`, matchID, user.ID).Scan(&roleType)

	if err == sql.ErrNoRows {
		http.Error(w, "You are not assigned to this match", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Update acknowledgment status
	now := time.Now()
	_, err = db.Exec(`
		UPDATE match_roles
		SET acknowledged = true, acknowledged_at = $1
		WHERE match_id = $2 AND assigned_referee_id = $3
	`, now, matchID, user.ID)

	if err != nil {
		http.Error(w, "Failed to acknowledge assignment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"acknowledged_at": now,
	})
}
