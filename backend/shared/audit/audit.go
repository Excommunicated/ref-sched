package audit

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/msheeley/referee-scheduler/shared/middleware"
)

// ActionType represents the type of action performed
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
)

// Entry represents a single audit log entry
type Entry struct {
	UserID     *int64      `json:"user_id,omitempty"`
	ActionType ActionType  `json:"action_type"`
	EntityType string      `json:"entity_type"`
	EntityID   int64       `json:"entity_id"`
	OldValues  interface{} `json:"old_values,omitempty"`
	NewValues  interface{} `json:"new_values,omitempty"`
	IPAddress  string      `json:"ip_address,omitempty"`
}

// Logger provides async audit logging functionality
type Logger struct {
	db      *sql.DB
	logChan chan Entry
}

// NewLogger creates a new audit logger with async processing
func NewLogger(db *sql.DB) *Logger {
	logger := &Logger{
		db:      db,
		logChan: make(chan Entry, 100),
	}

	go logger.processLogs()

	return logger
}

// Log queues an audit entry for async processing
func (a *Logger) Log(entry Entry) {
	select {
	case a.logChan <- entry:
	default:
		log.Printf("Warning: Audit log channel full, dropping entry for %s %s:%d",
			entry.ActionType, entry.EntityType, entry.EntityID)
	}
}

// LogWithContext is a convenience method to log with user and IP from request context
func (a *Logger) LogWithContext(r *http.Request, actionType ActionType, entityType string, entityID int64, oldValues, newValues interface{}) {
	entry := Entry{
		ActionType: actionType,
		EntityType: entityType,
		EntityID:   entityID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  GetClientIP(r),
	}

	if user, ok := middleware.GetUserFromContext(r.Context()); ok {
		entry.UserID = &user.ID
	}

	a.Log(entry)
}

func (a *Logger) processLogs() {
	for entry := range a.logChan {
		if err := a.writeLog(entry); err != nil {
			log.Printf("Error writing audit log: %v", err)
		}
	}
}

func ptr(s string) *string {
	return &s
}

func (a *Logger) writeLog(entry Entry) error {
	var oldValuesJSON, newValuesJSON *string
	var err error

	if entry.OldValues != nil {
		ovj, err := json.Marshal(entry.OldValues)
		if err != nil {
			return err
		}
		oldValuesJSON = ptr(string(ovj))
	}

	if entry.NewValues != nil {
		nvj, err := json.Marshal(entry.NewValues)
		if err != nil {
			return err
		}
		newValuesJSON = ptr(string(nvj))
	}

	query := `
		INSERT INTO audit_logs (user_id, action_type, entity_type, entity_id, old_values, new_values, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = a.db.Exec(query,
		entry.UserID,
		entry.ActionType,
		entry.EntityType,
		entry.EntityID,
		oldValuesJSON,
		newValuesJSON,
		entry.IPAddress,
		time.Now(),
	)

	return err
}

// Close stops the audit logger and waits for pending entries to be written
func (a *Logger) Close() {
	close(a.logChan)
}

// GetClientIP extracts the client IP address from the request
func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	return r.RemoteAddr
}
