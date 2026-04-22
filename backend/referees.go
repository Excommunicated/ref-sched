package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// RefereeListItem represents a referee in the management list
type RefereeListItem struct {
	ID                int64      `json:"id"`
	Email             string     `json:"email"`
	Name              string     `json:"name"`
	FirstName         *string    `json:"first_name"`
	LastName          *string    `json:"last_name"`
	DateOfBirth       *time.Time `json:"date_of_birth"`
	Certified         bool       `json:"certified"`
	CertExpiry        *time.Time `json:"cert_expiry"`
	CertStatus        string     `json:"cert_status"` // valid, expiring_soon, expired, none
	Role              string     `json:"role"`
	Status            string     `json:"status"`
	Grade             *string    `json:"grade"`
	CreatedAt         time.Time  `json:"created_at"`
}

// RefereeUpdateRequest represents the update payload from assignor
type RefereeUpdateRequest struct {
	Status *string `json:"status"` // active, inactive, removed
	Grade  *string `json:"grade"`  // Junior, Mid, Senior, or null
	Role   *string `json:"role"`   // referee, assignor
}

// listRefereesHandler returns all referees for assignor management
func listRefereesHandler(w http.ResponseWriter, r *http.Request) {
	// Check that user is an assignor (already enforced by roleMiddleware)

	query := `
		SELECT id, email, name, first_name, last_name, date_of_birth,
		       certified, cert_expiry, role, status, grade, created_at
		FROM users
		WHERE role IN ('pending_referee', 'referee', 'assignor') AND status != 'removed'
		ORDER BY
		  CASE
		    WHEN role = 'assignor' THEN 0
		    WHEN status = 'pending' THEN 1
		    WHEN status = 'active' THEN 2
		    WHEN status = 'inactive' THEN 3
		  END,
		  created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Failed to fetch referees", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	referees := []RefereeListItem{}
	now := time.Now()

	for rows.Next() {
		var ref RefereeListItem
		err := rows.Scan(
			&ref.ID,
			&ref.Email,
			&ref.Name,
			&ref.FirstName,
			&ref.LastName,
			&ref.DateOfBirth,
			&ref.Certified,
			&ref.CertExpiry,
			&ref.Role,
			&ref.Status,
			&ref.Grade,
			&ref.CreatedAt,
		)
		if err != nil {
			continue
		}

		// Determine certification status
		ref.CertStatus = determineCertStatus(ref.Certified, ref.CertExpiry, now)

		referees = append(referees, ref)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(referees)
}

// updateRefereeHandler allows assignor to update referee status and grade
func updateRefereeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	refereeID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid referee ID", http.StatusBadRequest)
		return
	}

	var req RefereeUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current referee
	referee, err := getUserByID(refereeID)
	if err != nil {
		http.Error(w, "Referee not found", http.StatusNotFound)
		return
	}

	// Get current user
	currentUser := r.Context().Value(userContextKey).(*User)

	// Don't allow assignors to modify other assignors (except to demote them)
	if referee.Role == "assignor" && currentUser.ID != referee.ID {
		// Allow changing role from assignor to referee, but nothing else
		if req.Role == nil || *req.Role == "assignor" {
			http.Error(w, "Cannot modify other assignor accounts", http.StatusForbidden)
			return
		}
	}

	// Prevent self-deactivation
	if currentUser.ID == refereeID && req.Status != nil && (*req.Status == "inactive" || *req.Status == "removed") {
		http.Error(w, "Cannot deactivate your own account", http.StatusForbidden)
		return
	}

	// Check for upcoming assignments before allowing deactivation
	if req.Status != nil && (*req.Status == "inactive" || *req.Status == "removed") {
		var hasUpcomingAssignments bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM match_roles mr
				JOIN matches m ON mr.match_id = m.id
				WHERE mr.assigned_referee_id = $1
				  AND m.match_date >= CURRENT_DATE
				  AND m.status = 'active'
			)
		`, refereeID).Scan(&hasUpcomingAssignments)

		if err != nil {
			http.Error(w, "Failed to check for upcoming assignments", http.StatusInternalServerError)
			return
		}

		if hasUpcomingAssignments {
			http.Error(w, "Cannot deactivate user with upcoming match assignments", http.StatusBadRequest)
			return
		}
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Status != nil {
		// Validate status
		validStatuses := map[string]bool{"pending": true, "active": true, "inactive": true, "removed": true}
		if !validStatuses[*req.Status] {
			http.Error(w, "Invalid status. Must be: pending, active, inactive, or removed", http.StatusBadRequest)
			return
		}

		updates = append(updates, "status = $"+strconv.Itoa(argCount))
		args = append(args, *req.Status)
		argCount++

		// When activating a pending_referee, promote to referee role (if no explicit role change)
		if *req.Status == "active" && referee.Role == "pending_referee" && req.Role == nil {
			updates = append(updates, "role = $"+strconv.Itoa(argCount))
			args = append(args, "referee")
			argCount++
		}
	}

	if req.Role != nil {
		// Validate role
		validRoles := map[string]bool{"referee": true, "assignor": true}
		if !validRoles[*req.Role] {
			http.Error(w, "Invalid role. Must be: referee or assignor", http.StatusBadRequest)
			return
		}

		// Only allow promoting to assignor or demoting from assignor
		if referee.Role != *req.Role {
			updates = append(updates, "role = $"+strconv.Itoa(argCount))
			args = append(args, *req.Role)
			argCount++

			// When promoting to assignor, ensure status is active
			if *req.Role == "assignor" && referee.Status != "active" && req.Status == nil {
				updates = append(updates, "status = $"+strconv.Itoa(argCount))
				args = append(args, "active")
				argCount++
			}
		}
	}

	if req.Grade != nil {
		// Validate grade if not null
		if *req.Grade != "" {
			validGrades := map[string]bool{"Junior": true, "Mid": true, "Senior": true}
			if !validGrades[*req.Grade] {
				http.Error(w, "Invalid grade. Must be: Junior, Mid, or Senior", http.StatusBadRequest)
				return
			}
			updates = append(updates, "grade = $"+strconv.Itoa(argCount))
			args = append(args, *req.Grade)
		} else {
			// Allow setting grade to NULL
			updates = append(updates, "grade = NULL")
		}
		argCount++
	}

	if len(updates) == 0 {
		http.Error(w, "No updates provided", http.StatusBadRequest)
		return
	}

	// Always update updated_at
	updates = append(updates, "updated_at = NOW()")

	// Add WHERE clause
	args = append(args, refereeID)

	query := "UPDATE users SET " + joinStrings(updates, ", ") + " WHERE id = $" + strconv.Itoa(argCount) +
		" RETURNING id, email, name, first_name, last_name, date_of_birth, certified, cert_expiry, role, status, grade, created_at, updated_at"

	updatedRef := &User{}
	err = db.QueryRow(query, args...).Scan(
		&updatedRef.ID,
		&updatedRef.Email,
		&updatedRef.Name,
		&updatedRef.FirstName,
		&updatedRef.LastName,
		&updatedRef.DateOfBirth,
		&updatedRef.Certified,
		&updatedRef.CertExpiry,
		&updatedRef.Role,
		&updatedRef.Status,
		&updatedRef.Grade,
		&updatedRef.CreatedAt,
		&updatedRef.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "Failed to update referee", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedRef)
}

// Helper function to determine certification status
func determineCertStatus(certified bool, certExpiry *time.Time, now time.Time) string {
	if !certified {
		return "none"
	}
	if certExpiry == nil {
		return "none"
	}
	if certExpiry.Before(now) {
		return "expired"
	}
	// Check if expiring within 30 days
	thirtyDaysFromNow := now.AddDate(0, 0, 30)
	if certExpiry.Before(thirtyDaysFromNow) {
		return "expiring_soon"
	}
	return "valid"
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
