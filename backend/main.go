package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"github.com/msheeley/referee-scheduler/features/acknowledgment"
	"github.com/msheeley/referee-scheduler/features/assignments"
	auditFeature "github.com/msheeley/referee-scheduler/features/audit"
	"github.com/msheeley/referee-scheduler/features/availability"
	"github.com/msheeley/referee-scheduler/features/eligibility"
	"github.com/msheeley/referee-scheduler/features/match_reports"
	"github.com/msheeley/referee-scheduler/features/matches"
	"github.com/msheeley/referee-scheduler/features/rbac"
	"github.com/msheeley/referee-scheduler/features/referees"
	"github.com/msheeley/referee-scheduler/features/users"
	auditShared "github.com/msheeley/referee-scheduler/shared/audit"
	"github.com/msheeley/referee-scheduler/shared/config"
	"github.com/msheeley/referee-scheduler/shared/database"
	"github.com/msheeley/referee-scheduler/shared/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	cfg          *config.Config
	sessionStore *sessions.CookieStore
	oauth2Config *oauth2.Config

	usersService *users.Service
)

func main() {
	// Load configuration
	cfg = config.Load()

	// Connect to database
	dbConn, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	db := dbConn.DB

	// Run migrations
	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize shared audit logger
	auditLogger := auditShared.NewLogger(db)
	defer auditLogger.Close()
	log.Println("Audit logger initialized")

	// Initialize audit retention service
	auditRetentionService := auditFeature.NewRetentionService(db, cfg.AuditRetentionDays)
	auditRetentionService.Start()
	defer auditRetentionService.Stop()
	log.Println("Audit retention service started")

	// Initialize match retention service
	matchRetentionService := matches.NewRetentionService(db, cfg.MatchRetentionDays)
	matchRetentionService.Start()
	defer matchRetentionService.Stop()
	log.Println("Match retention service started")

	// Initialize session store
	sessionStore = sessions.NewCookieStore([]byte(cfg.SessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	}

	// Initialize OAuth2 config
	oauth2Config = &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// Initialize middleware
	authMW := middleware.NewAuthMiddleware(sessionStore, db)
	rbacMW := middleware.NewRBACMiddleware(sessionStore, db)

	// Initialize feature slices
	usersRepo := users.NewRepository(db)
	usersService = users.NewService(usersRepo)
	usersHandler := users.NewHandler(usersService)
	log.Println("Users feature initialized")

	matchesRepo := matches.NewRepository(db)
	matchesService := matches.NewService(matchesRepo)
	matchesHandler := matches.NewHandler(matchesService)
	log.Println("Matches feature initialized")

	assignmentsRepo := assignments.NewRepository(db)
	assignmentsService := assignments.NewService(assignmentsRepo)
	assignmentsHandler := assignments.NewHandler(assignmentsService)
	log.Println("Assignments feature initialized")

	acknowledgmentRepo := acknowledgment.NewRepository(db)
	acknowledgmentService := acknowledgment.NewService(acknowledgmentRepo)
	acknowledgmentHandler := acknowledgment.NewHandler(acknowledgmentService)
	log.Println("Acknowledgment feature initialized")

	refereesRepo := referees.NewRepository(db)
	refereesService := referees.NewService(refereesRepo)
	refereesHandler := referees.NewHandler(refereesService)
	log.Println("Referees feature initialized")

	availabilityRepo := availability.NewRepository(db)
	availabilityService := availability.NewService(availabilityRepo)
	availabilityHandler := availability.NewHandler(availabilityService, db)
	log.Println("Availability feature initialized")

	eligibilityRepo := eligibility.NewRepository(db)
	eligibilityService := eligibility.NewService(eligibilityRepo)
	eligibilityHandler := eligibility.NewHandler(eligibilityService)
	log.Println("Eligibility feature initialized")

	matchReportsRepo := match_reports.NewRepository(db)
	matchReportsService := match_reports.NewService(matchReportsRepo, db)
	matchReportsHandler := match_reports.NewHandler(matchReportsService, db)
	log.Println("Match reports feature initialized")

	auditHandler := auditFeature.NewHandler(db, auditRetentionService)
	log.Println("Audit feature initialized")

	rbacHandler := rbac.NewHandler(db, auditLogger, authMW)
	log.Println("RBAC feature initialized")

	// Setup router
	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/auth/google", googleAuthHandler).Methods("GET")
	r.HandleFunc("/api/auth/google/callback", googleCallbackHandler).Methods("GET")
	r.HandleFunc("/api/auth/logout", logoutHandler).Methods("POST")

	// Feature routes
	usersHandler.RegisterRoutes(r, authMW.RequireAuth)
	matchesHandler.RegisterRoutes(r, authMW.RequireAuth, rbacMW.RequirePermission)
	assignmentsHandler.RegisterRoutes(r, authMW.RequireAuth, rbacMW.RequirePermission)
	acknowledgmentHandler.RegisterRoutes(r, authMW.RequireAuth)
	refereesHandler.RegisterRoutes(r, authMW.RequireAuth, rbacMW.RequirePermission)
	availabilityHandler.RegisterRoutes(r, authMW.RequireAuth)
	eligibilityHandler.RegisterRoutes(r, authMW.RequireAuth, rbacMW.RequirePermission)
	matchReportsHandler.RegisterRoutes(r, authMW.RequireAuth)
	auditHandler.RegisterRoutes(r, rbacMW.RequirePermission)
	rbacHandler.RegisterRoutes(r, authMW.RequireAuth, rbacMW.RequirePermission)

	// Match retention purge route
	r.HandleFunc("/api/admin/matches/purge", rbacMW.RequirePermission("can_view_audit_logs", purgeArchivedMatchesHandler(matchRetentionService))).Methods("POST")

	// Setup CORS
	corsHandler := middleware.NewCORSHandler(cfg.FrontendURL)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, corsHandler.Handler(r)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func googleAuthHandler(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "Failed to generate random state", http.StatusInternalServerError)
		return
	}
	state := base64.URLEncoding.EncodeToString(b)

	session, _ := sessionStore.Get(r, "auth-session")
	session.Values["oauth_state"] = state
	session.Save(r, w)

	url := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
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

	code := r.URL.Query().Get("code")
	token, err := oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

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

	user, err := usersService.FindOrCreate(r.Context(), userInfo.ID, userInfo.Email, userInfo.Name)
	if err != nil {
		log.Printf("Failed to find or create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = user.ID
	session.Values["user_role"] = user.Role
	delete(session.Values, "oauth_state")
	if err := session.Save(r, w); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, cfg.FrontendURL+"/auth/callback", http.StatusTemporaryRedirect)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(r, w)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

// purgeArchivedMatchesHandler returns a handler that triggers archived match purge
func purgeArchivedMatchesHandler(retentionService *matches.RetentionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if retentionService == nil {
			http.Error(w, "Match retention service not initialized", http.StatusInternalServerError)
			return
		}

		log.Println("Manual archived match purge triggered by admin")

		result, err := retentionService.PurgeOldMatches()
		if err != nil {
			log.Printf("Error during manual match purge: %v", err)
			http.Error(w, "Failed to purge archived matches", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}
