package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	db           *sql.DB
	sessionStore *sessions.CookieStore
	oauth2Config *oauth2.Config
)

func main() {
	// Initialize database
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Append timezone parameter to ensure all time operations use Eastern Time
	if !strings.Contains(dbURL, "timezone=") {
		if strings.Contains(dbURL, "?") {
			dbURL += "&timezone=America/New_York"
		} else {
			dbURL += "?timezone=America/New_York"
		}
	}

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// Run migrations
	if err := runMigrations(dbURL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize session store
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable is required")
	}
	sessionStore = sessions.NewCookieStore([]byte(sessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
		SameSite: http.SameSiteLaxMode,
	}

	// Initialize OAuth2 config
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		log.Fatal("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, and GOOGLE_REDIRECT_URL are required")
	}

	oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// Setup router
	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/auth/google", googleAuthHandler).Methods("GET")
	r.HandleFunc("/api/auth/google/callback", googleCallbackHandler).Methods("GET")
	r.HandleFunc("/api/auth/logout", logoutHandler).Methods("POST")
	r.HandleFunc("/api/auth/me", authMiddleware(meHandler)).Methods("GET")

	// Profile routes (authenticated users)
	r.HandleFunc("/api/profile", authMiddleware(getProfileHandler)).Methods("GET")
	r.HandleFunc("/api/profile", authMiddleware(updateProfileHandler)).Methods("PUT")

	// Referee management routes (assignors only)
	r.HandleFunc("/api/referees", authMiddleware(assignorOnly(listRefereesHandler))).Methods("GET")
	r.HandleFunc("/api/referees/{id}", authMiddleware(assignorOnly(updateRefereeHandler))).Methods("PUT")

	// Match management routes (assignors only)
	r.HandleFunc("/api/matches/import/parse", authMiddleware(assignorOnly(parseCSVHandler))).Methods("POST")
	r.HandleFunc("/api/matches/import/confirm", authMiddleware(assignorOnly(importMatchesHandler))).Methods("POST")
	r.HandleFunc("/api/matches", authMiddleware(assignorOnly(listMatchesHandler))).Methods("GET")
	r.HandleFunc("/api/matches/{id}", authMiddleware(assignorOnly(updateMatchHandler))).Methods("PUT")
	r.HandleFunc("/api/matches/{id}/eligible-referees", authMiddleware(assignorOnly(getEligibleRefereesHandler))).Methods("GET")

	// Referee availability routes
	r.HandleFunc("/api/referee/matches", authMiddleware(getEligibleMatchesForRefereeHandler)).Methods("GET")
	r.HandleFunc("/api/referee/matches/{id}/availability", authMiddleware(toggleAvailabilityHandler)).Methods("POST")
	r.HandleFunc("/api/referee/matches/{match_id}/acknowledge", authMiddleware(acknowledgeAssignmentHandler)).Methods("POST")

	// Day unavailability routes
	r.HandleFunc("/api/referee/day-unavailability", authMiddleware(getDayUnavailabilityHandler)).Methods("GET")
	r.HandleFunc("/api/referee/day-unavailability/{date}", authMiddleware(toggleDayUnavailabilityHandler)).Methods("POST")

	// Assignment routes (assignors only)
	r.HandleFunc("/api/matches/{match_id}/roles/{role_type}/assign", authMiddleware(assignorOnly(assignRefereeHandler))).Methods("POST")
	r.HandleFunc("/api/matches/{match_id}/conflicts", authMiddleware(assignorOnly(getConflictingAssignmentsHandler))).Methods("GET")

	// Setup CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{os.Getenv("FRONTEND_URL")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, corsHandler.Handler(r)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func runMigrations(dbURL string) error {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Migrations completed successfully")
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func googleAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Generate random state for CSRF protection
	state := fmt.Sprintf("%d", time.Now().UnixNano())

	// Store state in session
	session, _ := sessionStore.Get(r, "auth-session")
	session.Values["oauth_state"] = state
	session.Save(r, w)

	url := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Verify state
	session, err := sessionStore.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	state := r.URL.Query().Get("state")
	storedState, ok := session.Values["oauth_state"].(string)
	if !ok || state != storedState {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	token, err := oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Get user info from Google
	client := oauth2Config.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	// Find or create user
	user, err := findOrCreateUser(userInfo.ID, userInfo.Email, userInfo.Name)
	if err != nil {
		log.Printf("Failed to find or create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Store user ID in session
	session.Values["user_id"] = user.ID
	session.Values["user_role"] = user.Role
	delete(session.Values, "oauth_state")
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirect to frontend
	frontendURL := os.Getenv("FRONTEND_URL")
	http.Redirect(w, r, frontendURL+"/auth/callback", http.StatusTemporaryRedirect)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	// Clear session
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(r, w)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}

// Middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionStore.Get(r, "auth-session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userID, ok := session.Values["user_id"].(int64)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user from database
		user, err := getUserByID(userID)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := r.Context()
		ctx = contextWithUser(ctx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// assignorOnly middleware ensures only assignors can access the route
func assignorOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*User)

		if user.Role != "assignor" {
			http.Error(w, "Forbidden: Assignor access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}
