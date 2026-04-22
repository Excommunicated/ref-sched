package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// getEasternLocation returns US Eastern timezone
// All match dates and times are stored and displayed in US Eastern Time
func getEasternLocation() (*time.Location, error) {
	return time.LoadLocation("America/New_York")
}

// Match represents a scheduled match
type Match struct {
	ID          int64      `json:"id"`
	EventName   string     `json:"event_name"`
	TeamName    string     `json:"team_name"`
	AgeGroup    *string    `json:"age_group"`
	MatchDate   time.Time  `json:"match_date"`
	StartTime   string     `json:"start_time"`
	EndTime     string     `json:"end_time"`
	Location    string     `json:"location"`
	Description *string    `json:"description"`
	ReferenceID *string    `json:"reference_id"`
	Status      string     `json:"status"`
	CreatedBy   int64      `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// MatchRole represents a role slot for a match
type MatchRole struct {
	ID                 int64      `json:"id"`
	MatchID            int64      `json:"match_id"`
	RoleType           string     `json:"role_type"` // center, assistant_1, assistant_2
	AssignedRefereeID  *int64     `json:"assigned_referee_id"`
	AssignedRefereeName *string   `json:"assigned_referee_name,omitempty"`
	Acknowledged       bool       `json:"acknowledged"`
	AcknowledgedAt     *string    `json:"acknowledged_at,omitempty"`
	AckOverdue         bool       `json:"ack_overdue"` // True if assigned >24h ago and not acknowledged
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CSVRow represents a parsed row from Stack Team App CSV
type CSVRow struct {
	EventName   string  `json:"event_name"`
	TeamName    string  `json:"team_name"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	ReferenceID string  `json:"reference_id"`
	AgeGroup    *string `json:"age_group"`
	Error       *string `json:"error"`
	RowNumber   int     `json:"row_number"`
}

// ImportPreviewResponse contains parsed rows and any duplicates found
type ImportPreviewResponse struct {
	Rows       []CSVRow              `json:"rows"`
	Duplicates []DuplicateMatchGroup `json:"duplicates"`
}

// DuplicateMatchGroup represents a set of duplicate matches
type DuplicateMatchGroup struct {
	Signal   string   `json:"signal"` // reference_id or datetime_location
	Matches  []CSVRow `json:"matches"`
	Existing *Match   `json:"existing,omitempty"` // If duplicate with existing DB record
}

// ImportConfirmRequest contains the user's resolution of duplicates
type ImportConfirmRequest struct {
	Rows        []CSVRow                `json:"rows"`
	Resolutions map[string][]CSVRow     `json:"resolutions"` // Key: duplicate group ID, Value: rows to import
}

// extractAgeGroup extracts age group from team name
// Pattern: "Under {N}" → U{N}
// Examples: "Under 12 Girls - Falcons" → "U12", "Under 8 Boys" → "U8"
func extractAgeGroup(teamName string) *string {
	re := regexp.MustCompile(`(?i)under\s+(\d+)`)
	matches := re.FindStringSubmatch(teamName)
	if len(matches) > 1 {
		ageGroup := "U" + matches[1]
		return &ageGroup
	}
	return nil
}

// parseCSVHandler parses uploaded CSV and returns preview with errors
func parseCSVHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (10MB limit)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		http.Error(w, "Only .csv files are accepted", http.StatusBadRequest)
		return
	}

	// Parse CSV
	reader := csv.NewReader(file)

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		http.Error(w, "Failed to read CSV headers", http.StatusBadRequest)
		return
	}

	// Map column names to indices (case-insensitive)
	colMap := make(map[string]int)
	for i, header := range headers {
		colMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	// Validate required columns
	requiredCols := []string{"event_name", "team_name", "start_date", "start_time", "end_time", "location"}
	for _, col := range requiredCols {
		if _, exists := colMap[col]; !exists {
			http.Error(w, fmt.Sprintf("Missing required column: %s", col), http.StatusBadRequest)
			return
		}
	}

	// Parse all rows
	rows := []CSVRow{}
	rowNumber := 1 // Start at 1 (header is row 0)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Skip malformed rows but continue parsing
			rowNumber++
			continue
		}

		rowNumber++

		// Extract fields
		row := CSVRow{
			RowNumber: rowNumber,
		}

		// Safe column access
		getCol := func(name string) string {
			if idx, ok := colMap[name]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}

		row.EventName = getCol("event_name")
		row.TeamName = getCol("team_name")
		row.StartDate = getCol("start_date")
		row.EndDate = getCol("end_date")
		row.StartTime = getCol("start_time")
		row.EndTime = getCol("end_time")
		row.Description = getCol("description")
		row.Location = getCol("location")
		row.ReferenceID = getCol("reference_id")

		// Validate required fields
		if row.EventName == "" || row.TeamName == "" || row.StartDate == "" ||
		   row.StartTime == "" || row.EndTime == "" || row.Location == "" {
			errMsg := "Missing required field(s)"
			row.Error = &errMsg
			rows = append(rows, row)
			continue
		}

		// Extract age group
		row.AgeGroup = extractAgeGroup(row.TeamName)
		if row.AgeGroup == nil {
			errMsg := "Unrecognised age group - could not extract 'Under N' from team name"
			row.Error = &errMsg
		}

		rows = append(rows, row)
	}

	// Check for duplicates (Story 3.2 will handle resolution)
	duplicates := detectDuplicates(rows)

	response := ImportPreviewResponse{
		Rows:       rows,
		Duplicates: duplicates,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// detectDuplicates finds duplicate matches in the upload
func detectDuplicates(rows []CSVRow) []DuplicateMatchGroup {
	duplicates := []DuplicateMatchGroup{}

	// Signal A: Same reference_id
	refIDMap := make(map[string][]CSVRow)
	for _, row := range rows {
		if row.ReferenceID != "" && row.Error == nil {
			refIDMap[row.ReferenceID] = append(refIDMap[row.ReferenceID], row)
		}
	}

	for _, matches := range refIDMap {
		if len(matches) > 1 {
			duplicates = append(duplicates, DuplicateMatchGroup{
				Signal:  "reference_id",
				Matches: matches,
			})
		}
	}

	// Signal B: Same date + start time + location (different reference_id)
	// TODO: Implement in Story 3.2

	return duplicates
}

// importMatchesHandler confirms and imports matches to database
func importMatchesHandler(w http.ResponseWriter, r *http.Request) {
	var req ImportConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	currentUser := r.Context().Value(userContextKey).(*User)

	imported := 0
	skipped := 0
	errors := []string{}

	// Load US Eastern timezone
	loc, err := getEasternLocation()
	if err != nil {
		http.Error(w, "Failed to load timezone", http.StatusInternalServerError)
		return
	}

	for _, row := range req.Rows {
		// Skip rows with unresolved errors
		if row.Error != nil {
			skipped++
			continue
		}

		// Parse date in Eastern Time
		var matchDate time.Time
		// Try parsing as YYYY-MM-DD
		parsedDate, err := time.ParseInLocation("2006-01-02", row.StartDate, loc)
		if err != nil {
			// Try parsing as DD/MM/YYYY
			parsedDate, err = time.ParseInLocation("02/01/2006", row.StartDate, loc)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Row %d: Invalid date format: %s", row.RowNumber, row.StartDate))
				skipped++
				continue
			}
		}
		matchDate = parsedDate

		// Insert match
		query := `
			INSERT INTO matches (event_name, team_name, age_group, match_date, start_time, end_time,
			                     location, description, reference_id, status, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', $10)
			RETURNING id
		`

		var matchID int64
		err = db.QueryRow(
			query,
			row.EventName,
			row.TeamName,
			row.AgeGroup,
			matchDate,
			row.StartTime,
			row.EndTime,
			row.Location,
			row.Description,
			row.ReferenceID,
			currentUser.ID,
		).Scan(&matchID)

		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: Database error: %s", row.RowNumber, err.Error()))
			skipped++
			continue
		}

		// Create role slots based on age group (Story 3.3)
		if row.AgeGroup != nil {
			err = createRoleSlotsForMatch(matchID, *row.AgeGroup)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Row %d: Failed to create role slots: %s", row.RowNumber, err.Error()))
			}
		}

		imported++
	}

	response := map[string]interface{}{
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createRoleSlotsForMatch creates appropriate role slots based on age group
// U6/U8: 1 center
// U10: 1 center (assistant slots can be added manually later)
// U12+: 1 center + 2 assistants
func createRoleSlotsForMatch(matchID int64, ageGroup string) error {
	// Extract numeric age from U6, U8, etc.
	ageStr := strings.TrimPrefix(ageGroup, "U")
	age, err := strconv.Atoi(ageStr)
	if err != nil {
		return fmt.Errorf("invalid age group format: %s", ageGroup)
	}

	// All matches get a center referee slot
	_, err = db.Exec(
		"INSERT INTO match_roles (match_id, role_type) VALUES ($1, 'center')",
		matchID,
	)
	if err != nil {
		return err
	}

	// U12+ matches get 2 assistant referee slots
	if age >= 12 {
		_, err = db.Exec(
			"INSERT INTO match_roles (match_id, role_type) VALUES ($1, 'assistant_1'), ($1, 'assistant_2')",
			matchID,
		)
		if err != nil {
			return err
		}
	}

	// U10 gets no assistant slots by default (assignor can add manually)
	// U6/U8 get no assistant slots

	return nil
}

// MatchWithRoles includes match data and role assignments
type MatchWithRoles struct {
	Match
	Roles            []MatchRole `json:"roles"`
	AssignmentStatus string      `json:"assignment_status"` // unassigned, partial, full
	HasOverdueAck    bool        `json:"has_overdue_ack"`    // true if any assignment is overdue
}

// listMatchesHandler returns all matches for assignor schedule view
func listMatchesHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, event_name, team_name, age_group, match_date, start_time, end_time,
		       location, description, reference_id, status, created_by, created_at, updated_at
		FROM matches
		WHERE status != 'deleted'
		ORDER BY match_date ASC, start_time ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Failed to fetch matches", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	matches := []MatchWithRoles{}
	for rows.Next() {
		var m Match
		err := rows.Scan(
			&m.ID,
			&m.EventName,
			&m.TeamName,
			&m.AgeGroup,
			&m.MatchDate,
			&m.StartTime,
			&m.EndTime,
			&m.Location,
			&m.Description,
			&m.ReferenceID,
			&m.Status,
			&m.CreatedBy,
			&m.CreatedAt,
			&m.UpdatedAt,
		)
		if err != nil {
			continue
		}

		// Get role slots and assignments for this match
		roles, status := getMatchRoles(m.ID)

		// Check if any role has an overdue acknowledgment
		hasOverdueAck := false
		for _, role := range roles {
			if role.AckOverdue {
				hasOverdueAck = true
				break
			}
		}

		matches = append(matches, MatchWithRoles{
			Match:            m,
			Roles:            roles,
			AssignmentStatus: status,
			HasOverdueAck:    hasOverdueAck,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}

// getMatchRoles fetches all role slots for a match and determines assignment status
func getMatchRoles(matchID int64) ([]MatchRole, string) {
	query := `
		SELECT mr.id, mr.match_id, mr.role_type, mr.assigned_referee_id,
		       COALESCE(u.first_name || ' ' || u.last_name, u.name) as referee_name,
		       mr.acknowledged, mr.acknowledged_at,
		       mr.created_at, mr.updated_at
		FROM match_roles mr
		LEFT JOIN users u ON mr.assigned_referee_id = u.id
		WHERE mr.match_id = $1
		ORDER BY mr.role_type
	`

	rows, err := db.Query(query, matchID)
	if err != nil {
		return []MatchRole{}, "unassigned"
	}
	defer rows.Close()

	roles := []MatchRole{}
	totalSlots := 0
	assignedSlots := 0

	for rows.Next() {
		var role MatchRole
		var refereeName *string
		var acknowledgedAt sql.NullTime

		err := rows.Scan(
			&role.ID,
			&role.MatchID,
			&role.RoleType,
			&role.AssignedRefereeID,
			&refereeName,
			&role.Acknowledged,
			&acknowledgedAt,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			continue
		}

		role.AssignedRefereeName = refereeName
		if acknowledgedAt.Valid {
			ackTime := acknowledgedAt.Time.Format(time.RFC3339)
			role.AcknowledgedAt = &ackTime
		}

		// Check if acknowledgment is overdue (assigned >24h ago and not acknowledged)
		if role.AssignedRefereeID != nil && !role.Acknowledged {
			hoursSinceAssignment := time.Since(role.UpdatedAt).Hours()
			if hoursSinceAssignment > 24 {
				role.AckOverdue = true
			}
		}

		roles = append(roles, role)

		totalSlots++
		if role.AssignedRefereeID != nil {
			assignedSlots++
		}
	}

	// Determine assignment status
	status := "unassigned"
	if assignedSlots == totalSlots && totalSlots > 0 {
		status = "full"
	} else if assignedSlots > 0 {
		status = "partial"
	}

	return roles, status
}

// MatchUpdateRequest represents the update payload for a match
type MatchUpdateRequest struct {
	EventName   *string `json:"event_name"`
	TeamName    *string `json:"team_name"`
	AgeGroup    *string `json:"age_group"`
	MatchDate   *string `json:"match_date"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
	Location    *string `json:"location"`
	Description *string `json:"description"`
	Status      *string `json:"status"` // active, cancelled
}

// updateMatchHandler updates a match
func updateMatchHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	var req MatchUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Load US Eastern timezone
	loc, err := getEasternLocation()
	if err != nil {
		http.Error(w, "Failed to load timezone", http.StatusInternalServerError)
		return
	}

	currentUser := r.Context().Value(userContextKey).(*User)

	// Get current match
	var currentMatch Match
	err = db.QueryRow(
		"SELECT id, age_group FROM matches WHERE id = $1 AND status != 'deleted'",
		matchID,
	).Scan(&currentMatch.ID, &currentMatch.AgeGroup)
	if err != nil {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.EventName != nil {
		updates = append(updates, "event_name = $"+strconv.Itoa(argCount))
		args = append(args, *req.EventName)
		argCount++
	}

	if req.TeamName != nil {
		updates = append(updates, "team_name = $"+strconv.Itoa(argCount))
		args = append(args, *req.TeamName)
		argCount++
	}

	ageGroupChanged := false
	if req.AgeGroup != nil && *req.AgeGroup != "" {
		// Check if age group is changing
		currentAgeGroupStr := ""
		if currentMatch.AgeGroup != nil {
			currentAgeGroupStr = *currentMatch.AgeGroup
		}
		if *req.AgeGroup != currentAgeGroupStr {
			ageGroupChanged = true
		}

		updates = append(updates, "age_group = $"+strconv.Itoa(argCount))
		args = append(args, *req.AgeGroup)
		argCount++
	}

	if req.MatchDate != nil {
		// Parse the date in Eastern Time
		parsedDate, err := time.ParseInLocation("2006-01-02", *req.MatchDate, loc)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}
		updates = append(updates, "match_date = $"+strconv.Itoa(argCount))
		args = append(args, parsedDate)
		argCount++
	}

	if req.StartTime != nil {
		updates = append(updates, "start_time = $"+strconv.Itoa(argCount))
		args = append(args, *req.StartTime)
		argCount++
	}

	if req.EndTime != nil {
		updates = append(updates, "end_time = $"+strconv.Itoa(argCount))
		args = append(args, *req.EndTime)
		argCount++
	}

	if req.Location != nil {
		updates = append(updates, "location = $"+strconv.Itoa(argCount))
		args = append(args, *req.Location)
		argCount++
	}

	if req.Description != nil {
		updates = append(updates, "description = $"+strconv.Itoa(argCount))
		args = append(args, *req.Description)
		argCount++
	}

	if req.Status != nil {
		// Validate status
		validStatuses := map[string]bool{"active": true, "cancelled": true}
		if !validStatuses[*req.Status] {
			http.Error(w, "Invalid status. Must be: active or cancelled", http.StatusBadRequest)
			return
		}

		updates = append(updates, "status = $"+strconv.Itoa(argCount))
		args = append(args, *req.Status)
		argCount++
	}

	if len(updates) == 0 {
		http.Error(w, "No updates provided", http.StatusBadRequest)
		return
	}

	// Always update updated_at
	updates = append(updates, "updated_at = NOW()")

	// Add WHERE clause
	args = append(args, matchID)

	query := "UPDATE matches SET " + joinStrings(updates, ", ") + " WHERE id = $" + strconv.Itoa(argCount)

	_, err = db.Exec(query, args...)
	if err != nil {
		http.Error(w, "Failed to update match", http.StatusInternalServerError)
		return
	}

	// If age group changed, reconfigure role slots
	if ageGroupChanged && req.AgeGroup != nil {
		err = reconfigureRoleSlots(matchID, *req.AgeGroup)
		if err != nil {
			// Log error but don't fail the update
			fmt.Printf("Warning: Failed to reconfigure role slots for match %d: %v\n", matchID, err)
		}
	}

	// Log the edit
	logMatchEdit(matchID, currentUser.ID, &req)

	// Return updated match
	var updated MatchWithRoles
	err = db.QueryRow(
		`SELECT id, event_name, team_name, age_group, match_date, start_time, end_time,
		        location, description, reference_id, status, created_by, created_at, updated_at
		 FROM matches WHERE id = $1`,
		matchID,
	).Scan(
		&updated.ID,
		&updated.EventName,
		&updated.TeamName,
		&updated.AgeGroup,
		&updated.MatchDate,
		&updated.StartTime,
		&updated.EndTime,
		&updated.Location,
		&updated.Description,
		&updated.ReferenceID,
		&updated.Status,
		&updated.CreatedBy,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		http.Error(w, "Failed to fetch updated match", http.StatusInternalServerError)
		return
	}

	roles, status := getMatchRoles(matchID)
	updated.Roles = roles
	updated.AssignmentStatus = status

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// reconfigureRoleSlots adjusts role slots when age group changes
func reconfigureRoleSlots(matchID int64, newAgeGroup string) error {
	// Extract numeric age
	ageStr := strings.TrimPrefix(newAgeGroup, "U")
	age, err := strconv.Atoi(ageStr)
	if err != nil {
		return fmt.Errorf("invalid age group format: %s", newAgeGroup)
	}

	// Get current role slots
	rows, err := db.Query(
		"SELECT role_type FROM match_roles WHERE match_id = $1 ORDER BY role_type",
		matchID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	currentRoles := []string{}
	for rows.Next() {
		var roleType string
		if err := rows.Scan(&roleType); err != nil {
			continue
		}
		currentRoles = append(currentRoles, roleType)
	}

	// Determine required roles for new age group
	requiredRoles := []string{"center"}
	if age >= 12 {
		requiredRoles = append(requiredRoles, "assistant_1", "assistant_2")
	}
	// U10 and below: center only (but keep existing assistants if any)

	// Add missing roles
	for _, required := range requiredRoles {
		found := false
		for _, current := range currentRoles {
			if current == required {
				found = true
				break
			}
		}
		if !found {
			_, err := db.Exec(
				"INSERT INTO match_roles (match_id, role_type) VALUES ($1, $2)",
				matchID, required,
			)
			if err != nil {
				return err
			}
		}
	}

	// For U6/U8: Remove assistant slots if they exist
	if age < 10 {
		_, err := db.Exec(
			"DELETE FROM match_roles WHERE match_id = $1 AND role_type IN ('assistant_1', 'assistant_2')",
			matchID,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// logMatchEdit logs a match edit to assignment_history
func logMatchEdit(matchID int64, actorID int64, changes *MatchUpdateRequest) {
	// Build change description
	changeDesc := "Match edited: "
	parts := []string{}

	if changes.EventName != nil {
		parts = append(parts, "event_name")
	}
	if changes.TeamName != nil {
		parts = append(parts, "team_name")
	}
	if changes.AgeGroup != nil {
		parts = append(parts, "age_group")
	}
	if changes.MatchDate != nil {
		parts = append(parts, "date")
	}
	if changes.StartTime != nil || changes.EndTime != nil {
		parts = append(parts, "time")
	}
	if changes.Location != nil {
		parts = append(parts, "location")
	}
	if changes.Description != nil {
		parts = append(parts, "description")
	}
	if changes.Status != nil {
		parts = append(parts, "status")
	}

	changeDesc += joinStrings(parts, ", ")

	// Insert into assignment_history (repurposing for match edits)
	_, err := db.Exec(
		`INSERT INTO assignment_history (match_id, role_type, action, actor_id)
		 VALUES ($1, 'match_edit', $2, $3)`,
		matchID, changeDesc, actorID,
	)
	if err != nil {
		fmt.Printf("Warning: Failed to log match edit: %v\n", err)
	}
}
