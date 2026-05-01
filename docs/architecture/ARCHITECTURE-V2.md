# Backend Architecture: Vertical Slice Pattern

## Overview

The Referee Scheduler backend follows a **Vertical Slice Architecture** where code is organized by feature/capability rather than by technical layer. Each feature is self-contained with all layers (handler, service, repository, models) in one directory.

**Philosophy**: "Features that change together, live together."

---

## Project Structure

```
backend/
├── main.go                      # Application entry point
├── go.mod                       # Go module dependencies
├── go.sum                       # Dependency checksums
├── migrations/                  # Database migrations
│   ├── 001_initial_schema.up.sql
│   └── ...
├── shared/                      # Shared infrastructure (cross-cutting concerns)
│   ├── config/                  # Configuration management
│   │   └── config.go            # Environment variables, app config
│   ├── database/                # Database connection & utilities
│   │   ├── db.go                # Connection pool, ping, close
│   │   └── migrations.go        # Migration runner
│   ├── middleware/              # HTTP middleware
│   │   ├── auth.go              # Authentication middleware
│   │   ├── rbac.go              # Authorization/permission checks
│   │   ├── cors.go              # CORS configuration
│   │   └── logging.go           # Request logging
│   ├── errors/                  # Standard error handling
│   │   └── errors.go            # Custom error types, handlers
│   └── utils/                   # Shared utilities
│       ├── ip.go                # IP address extraction
│       └── json.go              # JSON helpers
├── features/                    # Feature slices (vertical)
│   ├── auth/                    # Authentication & OAuth
│   │   ├── handler.go           # HTTP handlers (googleAuthHandler, callbackHandler, etc.)
│   │   ├── service.go           # Business logic (token exchange, user creation)
│   │   ├── repository.go        # Data access (findOrCreateUser)
│   │   ├── models.go            # Domain models (User, Session)
│   │   └── routes.go            # Route registration
│   ├── users/                   # User management
│   │   ├── handler.go           # meHandler, getProfileHandler, updateProfileHandler
│   │   ├── service.go           # User business logic
│   │   ├── repository.go        # User data access
│   │   ├── models.go            # User models
│   │   └── routes.go            # User routes
│   ├── roles/                   # RBAC role management
│   │   ├── handler.go           # assignRoleToUser, revokeRoleFromUser, etc.
│   │   ├── service.go           # Role assignment logic, permission checks
│   │   ├── repository.go        # Role/permission data access
│   │   ├── models.go            # Role, Permission models
│   │   └── routes.go            # Role admin routes
│   ├── matches/                 # Match management
│   │   ├── handler.go           # listMatchesHandler, updateMatchHandler, importMatchesHandler
│   │   ├── service.go           # Match business logic, CSV parsing
│   │   ├── repository.go        # Match data access
│   │   ├── models.go            # Match, MatchImportRow models
│   │   └── routes.go            # Match routes
│   ├── assignments/             # Referee assignments
│   │   ├── handler.go           # assignRefereeHandler, addRoleSlotHandler, getConflictsHandler
│   │   ├── service.go           # Assignment logic, conflict detection
│   │   ├── repository.go        # Assignment data access
│   │   ├── models.go            # Assignment, Conflict models
│   │   └── routes.go            # Assignment routes
│   ├── availability/            # Referee availability
│   │   ├── handler.go           # toggleAvailabilityHandler, getDayUnavailabilityHandler
│   │   ├── service.go           # Availability business logic
│   │   ├── repository.go        # Availability data access
│   │   ├── models.go            # Availability, DayUnavailability models
│   │   └── routes.go            # Availability routes
│   ├── referees/                # Referee management
│   │   ├── handler.go           # listRefereesHandler, updateRefereeHandler
│   │   ├── service.go           # Referee business logic
│   │   ├── repository.go        # Referee data access
│   │   ├── models.go            # Referee models
│   │   └── routes.go            # Referee routes
│   ├── eligibility/             # Eligibility calculation
│   │   ├── handler.go           # getEligibleMatchesHandler, getEligibleRefereesHandler
│   │   ├── service.go           # Eligibility business logic
│   │   ├── repository.go        # Eligibility queries
│   │   ├── models.go            # EligibleMatch, EligibleReferee models
│   │   └── routes.go            # Eligibility routes
│   ├── acknowledgment/          # Assignment acknowledgment
│   │   ├── handler.go           # acknowledgeAssignmentHandler
│   │   ├── service.go           # Acknowledgment logic
│   │   ├── repository.go        # Acknowledgment data access
│   │   ├── models.go            # Acknowledgment models
│   │   └── routes.go            # Acknowledgment routes
│   └── audit/                   # Audit logging
│       ├── handler.go           # getAuditLogsHandler, exportHandler, purgeHandler
│       ├── service.go           # Audit logger, retention service
│       ├── repository.go        # Audit log queries
│       ├── models.go            # AuditLog, PurgeResult models
│       └── routes.go            # Audit routes
└── tests/                       # Test utilities
    ├── fixtures/                # Test data factories
    └── mocks/                   # Generated mocks
```

---

## Vertical Slice Pattern

### What is a Vertical Slice?

A vertical slice contains **all layers** of a feature in one directory:
- **Handler**: HTTP request/response handling, validation, serialization
- **Service**: Business logic, orchestration, authorization checks
- **Repository**: Database queries, data access layer
- **Models**: Domain objects, DTOs, request/response types
- **Routes**: Route registration for the feature

### Benefits

1. **High Cohesion**: Related code lives together
2. **Low Coupling**: Features are independent, reducing side effects
3. **Easy Navigation**: Everything for "matches" is in `features/matches/`
4. **Parallel Development**: Teams can work on different features without conflicts
5. **Clear Ownership**: Each feature slice has a clear boundary
6. **Easier Testing**: Mock dependencies at slice boundaries
7. **Simpler Deletion**: Remove entire feature by deleting one directory

### Comparison to Layered Architecture

**Layered (Horizontal)**:
```
handlers/
  match_handler.go
  user_handler.go
  assignment_handler.go
services/
  match_service.go
  user_service.go
  assignment_service.go
repositories/
  match_repository.go
  user_repository.go
  assignment_repository.go
```

**Vertical Slice**:
```
features/matches/
  handler.go
  service.go
  repository.go
features/users/
  handler.go
  service.go
  repository.go
features/assignments/
  handler.go
  service.go
  repository.go
```

---

## Layer Responsibilities

### 1. Handler Layer (`handler.go`)

**Responsibility**: HTTP request/response handling

**Concerns**:
- Parse request parameters (path, query, body)
- Validate input formats (not business rules)
- Call service layer
- Serialize response
- Set HTTP status codes
- Handle HTTP-specific errors

**Does NOT**:
- Contain business logic
- Query database directly
- Make authorization decisions (except extracting user from context)

**Example**:
```go
func (h *MatchHandler) ListMatches(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    queryParams := r.URL.Query()
    status := queryParams.Get("status")
    
    // Call service
    matches, err := h.service.ListMatches(r.Context(), status)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Serialize response
    json.NewEncoder(w).Encode(matches)
}
```

### 2. Service Layer (`service.go`)

**Responsibility**: Business logic and orchestration

**Concerns**:
- Business rules and validation
- Authorization checks (can this user do this action?)
- Orchestrate multiple repository calls
- Transaction management
- Cross-feature coordination
- Audit logging
- Error handling with business context

**Does NOT**:
- Parse HTTP requests
- Write SQL queries
- Set HTTP status codes

**Example**:
```go
func (s *MatchService) CreateMatch(ctx context.Context, data MatchCreateData) (*Match, error) {
    // Business validation
    if data.Date.Before(time.Now()) {
        return nil, errors.New("cannot create match in the past")
    }
    
    // Authorization check
    user := ctx.Value("user").(*User)
    if !user.HasPermission("can_create_matches") {
        return nil, errors.New("unauthorized")
    }
    
    // Call repository
    match, err := s.repo.Create(ctx, data)
    if err != nil {
        return nil, fmt.Errorf("failed to create match: %w", err)
    }
    
    // Audit log
    s.auditLogger.Log(ctx, "create", "match", match.ID, nil, match)
    
    return match, nil
}
```

### 3. Repository Layer (`repository.go`)

**Responsibility**: Data access

**Concerns**:
- SQL queries
- Database transactions
- CRUD operations
- Query building
- Row scanning
- NULL handling
- Database errors

**Does NOT**:
- Business validation
- Authorization
- HTTP handling
- Audit logging (service does this)

**Example**:
```go
func (r *MatchRepository) FindByID(ctx context.Context, id int64) (*Match, error) {
    query := `
        SELECT id, date, home_team, away_team, status, created_at
        FROM matches
        WHERE id = $1
    `
    
    var match Match
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &match.ID,
        &match.Date,
        &match.HomeTeam,
        &match.AwayTeam,
        &match.Status,
        &match.CreatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, nil // Not found
    }
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    
    return &match, nil
}
```

### 4. Models Layer (`models.go`)

**Responsibility**: Domain types and DTOs

**Contains**:
- Domain models (Match, User, Assignment)
- Request DTOs (CreateMatchRequest)
- Response DTOs (MatchListResponse)
- Enums and constants

**Example**:
```go
// Domain model
type Match struct {
    ID       int64     `json:"id"`
    Date     time.Time `json:"date"`
    HomeTeam string    `json:"home_team"`
    AwayTeam string    `json:"away_team"`
    Status   string    `json:"status"`
}

// Request DTO
type CreateMatchRequest struct {
    Date     string `json:"date"`
    HomeTeam string `json:"home_team"`
    AwayTeam string `json:"away_team"`
}

// Response DTO
type MatchListResponse struct {
    Matches    []Match `json:"matches"`
    TotalCount int     `json:"total_count"`
}
```

### 5. Routes Layer (`routes.go`)

**Responsibility**: Route registration

**Concerns**:
- Register HTTP routes
- Apply middleware (auth, RBAC, logging)
- Connect routes to handlers

**Example**:
```go
func (f *MatchFeature) RegisterRoutes(r *mux.Router, authMW, rbacMW func(http.HandlerFunc) http.HandlerFunc) {
    // Public routes (none for matches)
    
    // Authenticated routes
    r.HandleFunc("/api/matches", authMW(f.handler.ListMatches)).Methods("GET")
    r.HandleFunc("/api/matches/{id}", authMW(f.handler.GetMatch)).Methods("GET")
    
    // Admin routes (require permissions)
    r.HandleFunc("/api/matches", rbacMW("can_create_matches", f.handler.CreateMatch)).Methods("POST")
    r.HandleFunc("/api/matches/{id}", rbacMW("can_update_matches", f.handler.UpdateMatch)).Methods("PUT")
}
```

---

## Shared Infrastructure

### When to use `shared/` vs feature slice?

**Use `shared/`** for:
- Database connection management
- Middleware (auth, logging, CORS)
- Configuration loading
- Error types used across features
- Utilities with no feature-specific logic

**Keep in feature slice** for:
- Feature-specific business logic
- Feature-specific models
- Feature-specific queries

### Shared Packages

#### `shared/config/`
- Load environment variables
- Application configuration
- Database URLs, ports, secrets

#### `shared/database/`
- Database connection pool
- Migration runner
- Transaction helpers

#### `shared/middleware/`
- Authentication middleware
- RBAC/permission checks
- CORS configuration
- Request logging
- Error recovery

#### `shared/errors/`
- Standard error types (ValidationError, NotFoundError, UnauthorizedError)
- Error response formatting
- HTTP status code mapping

#### `shared/utils/`
- IP address extraction
- JSON helpers
- Date/time utilities
- String utilities

---

## Naming Conventions

### Files
- `handler.go` - HTTP handlers
- `service.go` - Business logic
- `repository.go` - Data access
- `models.go` - Domain types
- `routes.go` - Route registration

### Types
- Handlers: `MatchHandler`, `UserHandler`
- Services: `MatchService`, `UserService`
- Repositories: `MatchRepository`, `UserRepository`
- Models: `Match`, `User`, `Assignment`
- Requests: `CreateMatchRequest`, `UpdateUserRequest`
- Responses: `MatchListResponse`, `UserResponse`

### Functions
- Handlers: `ListMatches`, `GetMatch`, `CreateMatch`
- Services: `ListMatches`, `GetMatchByID`, `CreateMatch`
- Repositories: `FindAll`, `FindByID`, `Create`, `Update`, `Delete`

### Variables
- Handler: `h` (e.g., `func (h *MatchHandler)`)
- Service: `s` (e.g., `func (s *MatchService)`)
- Repository: `r` (e.g., `func (r *MatchRepository)`)

---

## Dependency Injection

Each feature slice is initialized with its dependencies:

```go
// Feature initialization in main.go
type MatchFeature struct {
    handler    *MatchHandler
    service    *MatchService
    repository *MatchRepository
}

func NewMatchFeature(db *sql.DB, auditLogger *AuditLogger) *MatchFeature {
    repo := NewMatchRepository(db)
    service := NewMatchService(repo, auditLogger)
    handler := NewMatchHandler(service)
    
    return &MatchFeature{
        handler:    handler,
        service:    service,
        repository: repo,
    }
}
```

**Benefits**:
- Easy to test (inject mocks)
- Clear dependencies
- No global state (except db connection)
- Compile-time dependency checking

---

## Testing Strategy

### Unit Tests
- Test service layer with mocked repository
- Test repository with test database
- Test handlers with mocked service

**Example**:
```go
// Service test with mocked repository
func TestMatchService_CreateMatch(t *testing.T) {
    mockRepo := mocks.NewMatchRepository(t)
    mockAudit := mocks.NewAuditLogger(t)
    service := NewMatchService(mockRepo, mockAudit)
    
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(&Match{ID: 1}, nil)
    
    match, err := service.CreateMatch(ctx, CreateMatchData{...})
    
    assert.NoError(t, err)
    assert.Equal(t, int64(1), match.ID)
    mockRepo.AssertExpectations(t)
}
```

### Integration Tests
- Test entire vertical slice (handler → service → repository)
- Use test database
- No mocks

---

## Migration Strategy

### Phase 1: Setup Shared Infrastructure (Story 8.2)
- Create `shared/` packages
- Extract shared code from `main.go`
- No feature changes yet

### Phase 2: Migrate Feature Slices (Stories 8.3-8.6)
- One feature at a time
- Move handlers, create service/repository layers
- Keep old files until new structure works
- Test each feature after migration

### Phase 3: Update Main (Story 8.7)
- Simplify `main.go`
- Register routes from features
- Clean initialization

### Phase 4: Cleanup (Story 8.9)
- Delete old flat structure files
- Remove duplicate code
- Final testing

---

## Adding a New Feature

**Step-by-step guide**:

1. **Create directory**: `features/newfeature/`

2. **Define models** (`models.go`):
   ```go
   type NewFeature struct {
       ID   int64  `json:"id"`
       Name string `json:"name"`
   }
   ```

3. **Create repository** (`repository.go`):
   ```go
   type NewFeatureRepository struct {
       db *sql.DB
   }
   
   func (r *NewFeatureRepository) FindAll(ctx context.Context) ([]NewFeature, error) {
       // SQL query
   }
   ```

4. **Create service** (`service.go`):
   ```go
   type NewFeatureService struct {
       repo *NewFeatureRepository
   }
   
   func (s *NewFeatureService) List(ctx context.Context) ([]NewFeature, error) {
       // Business logic
       return s.repo.FindAll(ctx)
   }
   ```

5. **Create handler** (`handler.go`):
   ```go
   type NewFeatureHandler struct {
       service *NewFeatureService
   }
   
   func (h *NewFeatureHandler) List(w http.ResponseWriter, r *http.Request) {
       // HTTP handling
       features, err := h.service.List(r.Context())
       json.NewEncoder(w).Encode(features)
   }
   ```

6. **Register routes** (`routes.go`):
   ```go
   func (f *NewFeature) RegisterRoutes(r *mux.Router, authMW func(...) ...) {
       r.HandleFunc("/api/newfeature", authMW(f.handler.List)).Methods("GET")
   }
   ```

7. **Initialize in main.go**:
   ```go
   newFeature := NewNewFeature(db, auditLogger)
   newFeature.RegisterRoutes(router, authMiddleware, rbacMiddleware)
   ```

---

## Common Patterns

### Pattern: Pagination
```go
// models.go
type PaginationParams struct {
    Page     int
    PageSize int
}

// repository.go
func (r *Repo) FindWithPagination(ctx context.Context, params PaginationParams) ([]Item, int, error) {
    // Count total
    var total int
    db.QueryRow("SELECT COUNT(*) FROM items").Scan(&total)
    
    // Get page
    offset := (params.Page - 1) * params.PageSize
    rows := db.Query("SELECT * FROM items LIMIT $1 OFFSET $2", params.PageSize, offset)
    
    return items, total, nil
}
```

### Pattern: Filtering
```go
// models.go
type MatchFilters struct {
    Status    string
    StartDate time.Time
    EndDate   time.Time
}

// repository.go
func (r *MatchRepository) FindWithFilters(ctx context.Context, filters MatchFilters) ([]Match, error) {
    query := "SELECT * FROM matches WHERE 1=1"
    args := []interface{}{}
    argCount := 1
    
    if filters.Status != "" {
        query += fmt.Sprintf(" AND status = $%d", argCount)
        args = append(args, filters.Status)
        argCount++
    }
    
    // ... more filters
    
    rows, err := r.db.Query(query, args...)
    // ... scan rows
}
```

### Pattern: Transactions
```go
func (s *Service) ComplexOperation(ctx context.Context) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback() // Rollback if not committed
    
    // Multiple operations
    _, err = tx.Exec("INSERT INTO table1 ...")
    if err != nil {
        return err
    }
    
    _, err = tx.Exec("UPDATE table2 ...")
    if err != nil {
        return err
    }
    
    return tx.Commit()
}
```

---

## FAQs

**Q: Where should I put code that's used by multiple features?**  
A: If it's infrastructure (auth, logging), put it in `shared/`. If it's business logic, consider creating a new feature slice or a shared service that both features depend on.

**Q: Can a feature call another feature's service?**  
A: Yes, but prefer keeping features independent. If there's significant cross-feature logic, it might indicate a missing abstraction or a new feature slice.

**Q: Should handlers have business logic?**  
A: No. Handlers should only handle HTTP concerns. Business logic belongs in the service layer.

**Q: Where do I put validation?**  
A: Format validation (required fields, valid JSON) in handler. Business validation (date must be in future, user must have permission) in service.

**Q: When should I create a new feature slice vs adding to existing?**  
A: Create a new slice when the code has a distinct bounded context and could reasonably be developed/deployed independently. Add to existing when it's an extension of current functionality.

---

## Architecture Decision Record (ADR)

### ADR-001: Adopt Vertical Slice Architecture

**Status**: Accepted  
**Date**: 2026-04-27  
**Context**: Current flat file structure makes it hard to navigate codebase and understand feature boundaries. As the application grows, files are becoming large and mixed with multiple concerns.

**Decision**: Adopt vertical slice architecture organized by feature/capability.

**Consequences**:
- ✅ **Positive**: Easier to find all code related to a feature
- ✅ **Positive**: Reduced merge conflicts (features are isolated)
- ✅ **Positive**: Clearer ownership and boundaries
- ✅ **Positive**: Easier onboarding for new developers
- ⚠️ **Negative**: More files and directories (mitigated by clear naming)
- ⚠️ **Negative**: Migration effort required (planned in Epic 8)

**Alternatives Considered**:
1. **Layered architecture** (handlers/, services/, repositories/) - Rejected: Scatters feature code across directories
2. **Domain-driven design** (DDD) with aggregates - Rejected: Too complex for current application size
3. **Keep flat structure** - Rejected: Doesn't scale as codebase grows

---

## References

- [Vertical Slice Architecture by Jimmy Bogard](https://www.jimmybogard.com/vertical-slice-architecture/)
- [Feature Slices for ASP.NET Core MVC by Jimmy Bogard](https://www.youtube.com/watch?v=5kOzZz2vj2o)
- [Screaming Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2011/09/30/Screaming-Architecture.html)

---

**Version**: 1.0  
**Last Updated**: 2026-04-27  
**Author**: Development Team
