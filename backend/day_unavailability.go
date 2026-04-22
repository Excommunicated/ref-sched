package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// DayUnavailability represents a day when a referee is unavailable
type DayUnavailability struct {
	ID               int64   `json:"id"`
	RefereeID        int64   `json:"referee_id"`
	UnavailableDate  string  `json:"unavailable_date"`
	Reason           *string `json:"reason,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

// getDayUnavailabilityHandler returns all days marked as unavailable for the current referee
func getDayUnavailabilityHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	rows, err := db.Query(`
		SELECT id, referee_id, unavailable_date, reason, created_at
		FROM day_unavailability
		WHERE referee_id = $1
		ORDER BY unavailable_date
	`, user.ID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var unavailableDays []DayUnavailability

	for rows.Next() {
		var day DayUnavailability
		var unavailableDate time.Time
		var createdAt time.Time
		var reason sql.NullString

		err := rows.Scan(&day.ID, &day.RefereeID, &unavailableDate, &reason, &createdAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("Scan error: %v", err), http.StatusInternalServerError)
			return
		}

		day.UnavailableDate = unavailableDate.Format("2006-01-02")
		day.CreatedAt = createdAt.Format(time.RFC3339)
		if reason.Valid {
			day.Reason = &reason.String
		}

		unavailableDays = append(unavailableDays, day)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(unavailableDays)
}

// toggleDayUnavailabilityHandler marks or unmarks a day as unavailable for the current referee
func toggleDayUnavailabilityHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	vars := mux.Vars(r)
	dateStr := vars["date"]

	// Validate date format
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	var req struct {
		Unavailable bool    `json:"unavailable"`
		Reason      *string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Unavailable {
		// Mark day as unavailable
		_, err = db.Exec(`
			INSERT INTO day_unavailability (referee_id, unavailable_date, reason, created_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (referee_id, unavailable_date)
			DO UPDATE SET reason = $3
		`, user.ID, dateStr, req.Reason)

		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}

		// Remove individual match availability for this day
		_, err = db.Exec(`
			DELETE FROM availability
			WHERE referee_id = $1
			  AND match_id IN (
				SELECT id FROM matches WHERE match_date = $2
			  )
		`, user.ID, dateStr)

		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// Remove day unavailability
		_, err = db.Exec(`
			DELETE FROM day_unavailability
			WHERE referee_id = $1 AND unavailable_date = $2
		`, user.ID, dateStr)

		if err != nil {
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"unavailable": req.Unavailable,
		"date":        dateStr,
	})
}
