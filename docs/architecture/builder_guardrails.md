# Builder Guardrails: Referee Scheduling App

These rules are non-negotiable. The Builder must follow them exactly. They exist to prevent security holes, maintainability problems, and bugs that are expensive to fix after the fact. When in doubt, ask before deviating.

---

## 1. Approved Libraries

### Backend (Go)

| Purpose | Library | Version |
|---------|---------|---------|
| HTTP router | `github.com/go-chi/chi/v5` | v5.x |
| PostgreSQL driver | `github.com/jackc/pgx/v5` | v5.x |
| Database migrations | `github.com/golang-migrate/migrate/v4` | v4.x |
| OAuth2 / OIDC | `golang.org/x/oauth2` | latest |
| Google OIDC ID token verification | `google.golang.org/api/idtoken` | latest |
| Structured logging | `log/slog` (Go standard library, 1.21+) | stdlib |
| Environment variable loading | `github.com/joho/godotenv` | v1.x (local dev only; not used in production) |
| CSV parsing | `encoding/csv` (Go standard library) | stdlib |
| Cryptographic random | `crypto/rand` (Go standard library) | stdlib |
| UUID generation | `github.com/google/uuid` | v1.x |

**Do not add libraries not on this list without explicit approval.** If a legitimate need arises, document the reason and confirm before adding a dependency.

**Explicitly forbidden:**
- `github.com/dgrijalva/jwt-go` or any JWT library (session management is server-side, not JWT-based)
- Any ORM (GORM, Ent, etc.) — write SQL directly using `pgx`
- Any mock authentication library or middleware that bypasses real auth
- Redis or any external cache — not needed at this scale

### Frontend (SvelteKit)

| Purpose | Library | Notes |
|---------|---------|-------|
| Framework | `@sveltejs/kit` | Latest stable |
| Build adapter | `@sveltejs/adapter-static` | Required; no Node.js runtime in production |
| CSS | Tailwind CSS v3 | Utility-first; sufficient for this app's UI needs |
| HTTP client | Browser `fetch` (native) | No Axios, no SWR, no TanStack Query needed |
| Form validation | Svelte native (`bind:`, form actions) | No heavy form libraries |
| Icons | `lucide-svelte` | Lightweight SVG icon set |

**Do not add a large component library** (shadcn-svelte, DaisyUI with JS, etc.) without explicit approval. Plain HTML + Tailwind is sufficient and keeps the bundle small.

---

## 2. Repository Structure

```
ref-sched/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # Entry point; starts DB, runs migrations, starts HTTP
│   ├── internal/
│   │   ├── api/                     # HTTP handlers, grouped by resource
│   │   │   ├── auth.go
│   │   │   ├── matches.go
│   │   │   ├── assignments.go
│   │   │   ├── referees.go
│   │   │   ├── availability.go
│   │   │   └── profile.go
│   │   ├── middleware/
│   │   │   ├── auth.go              # Session lookup middleware
│   │   │   └── role.go              # RequireRole middleware
│   │   ├── domain/
│   │   │   ├── eligibility.go       # Eligibility rules (pure functions, testable)
│   │   │   ├── csvimport.go         # CSV parsing and duplicate detection
│   │   │   └── models.go            # Shared domain types
│   │   ├── db/
│   │   │   ├── queries/             # SQL query functions (one file per resource)
│   │   │   │   ├── users.go
│   │   │   │   ├── matches.go
│   │   │   │   ├── assignments.go
│   │   │   │   └── availability.go
│   │   │   └── db.go                # Connection pool setup
│   │   └── config/
│   │       └── config.go            # Environment variable loading and validation
│   ├── db/
│   │   └── migrations/
│   │       ├── 0001_initial_schema.up.sql
│   │       ├── 0001_initial_schema.down.sql
│   │       └── ...
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +layout.svelte       # Root layout; loads /api/auth/me
│   │   │   ├── +page.svelte         # Login / home page
│   │   │   ├── dashboard/
│   │   │   │   └── +page.svelte     # Role-aware dashboard redirect
│   │   │   ├── referee/             # Referee-only routes
│   │   │   │   ├── matches/
│   │   │   │   └── profile/
│   │   │   └── assignor/            # Assignor-only routes
│   │   │       ├── matches/
│   │   │       ├── referees/
│   │   │       └── import/
│   │   ├── lib/
│   │   │   ├── api.ts               # Typed fetch wrappers for all API endpoints
│   │   │   └── stores.ts            # Svelte stores for user state
│   │   └── app.html
│   ├── static/
│   ├── svelte.config.js
│   ├── tailwind.config.js
│   └── Dockerfile
├── docs/
│   └── architecture/                # This directory
├── docker-compose.yml               # Local development: backend + frontend + postgres
├── docker-compose.prod.yml          # Production-like compose (optional)
└── .env.example                     # Template for required environment variables
```

**Rules:**
- Keep `internal/domain/` free of HTTP handler concerns. Domain functions take plain Go types and return plain Go types. They do not reference `http.Request`, `http.ResponseWriter`, or `pgx`.
- Keep `internal/api/` handlers thin: parse the request, call domain/db functions, write the response. No business logic in handlers.
- All SQL lives in `internal/db/queries/`. No raw SQL strings in handlers or domain functions.

---

## 3. Testing Requirements

### Unit tests (required)

These functions must have unit tests. They are testable without a database or HTTP server.

| Package | What to test |
|---------|-------------|
| `internal/domain/eligibility.go` | Every eligibility rule branch: U6/U8/U10 age check (at boundary, one above, one below), U12+ center cert check (valid cert, expired cert, no cert), U12+ assistant (always eligible as active referee), cert expiry exactly on match date |
| `internal/domain/csvimport.go` | Age group extraction from `team_name`: standard cases (U6, U8, U10, U12, U14), emoji in event_name, missing age group (expect error), gender extraction, venue_detail parsing from description field |
| `internal/domain/csvimport.go` | Duplicate detection: Signal A (same reference_id, two rows), Signal B (same date+time+location, different reference_id), no duplicates (clean batch), both signals in same file |
| `internal/config/config.go` | Required env vars present, required env vars missing (expect error) |

Test file naming: `foo_test.go` in the same package as `foo.go`.

### Integration tests (required)

These tests require a real PostgreSQL instance. Use a test database (`DATABASE_URL_TEST` env var pointing to a local test DB or a dockerized test DB started by the test harness).

| What to test |
|-------------|
| Session creation, lookup, and expiry (sessions table) |
| User upsert on OAuth2 callback (existing google_sub reuses record; new google_sub creates) |
| Referee profile create and update |
| Match import commit: staging rows → matches + role_slots (all in transaction) |
| Eligibility query returns correct results for each role/age-group combination |
| Assignment write: creates assignment row and audit_log row in same transaction |
| Reassignment: updates assignment row and creates `reassign` audit_log entry |
| Assignment removal: deletes assignment row and creates `remove` audit_log entry |
| Availability mark: upsert works; ineligible mark rejected |

Run integration tests with: `go test ./... -tags integration` (use build tag to skip in environments without a test DB).

### End-to-end tests (optional for v1)

Not required before August given timeline. If time permits, use Playwright to test the OAuth2 login flow (with a test Google account) and the CSV import flow.

### What not to test

- Database connection pooling internals.
- Third-party library behavior (pgx, chi, golang-migrate).
- The Google OAuth2 flow itself (test the callback handler with a mock token, not a live Google call).

---

## 4. API Conventions

### Request / Response format

- All bodies: `Content-Type: application/json`.
- All timestamps: RFC 3339 UTC (`2026-04-21T14:30:00Z`).
- All dates: `YYYY-MM-DD` (`2026-04-25`).
- All times (without date): `HH:MM` in 24-hour format (`08:30`, `14:15`).
- All IDs: integers (`"id": 101`), not UUIDs. The `import_token` (for staging) is the only UUID in the system and is a string field named `import_id`.
- Boolean fields: `true` / `false` (JSON booleans, not strings).
- Nullable fields: use JSON `null`, not empty string `""`.

### Error response format

Every non-2xx response must use this exact structure:

```json
{
  "error": {
    "code": "SCREAMING_SNAKE_CASE",
    "message": "Human-readable sentence.",
    "fields": {
      "field_name": "Field-specific error message."
    }
  }
}
```

`fields` is omitted (not `null`, not `{}`) when there are no field-level errors.

**Standard error codes** (use these exactly; do not invent new ones without justification):

| HTTP Status | Code | When to use |
|-------------|------|-------------|
| 400 | `BAD_REQUEST` | Malformed JSON, wrong Content-Type, invalid file type |
| 401 | `NOT_AUTHENTICATED` | No session or expired session |
| 403 | `FORBIDDEN` | Authenticated but wrong role, or eligibility check failed |
| 403 | `PENDING_ACTIVATION` | User is pending_referee attempting a referee-only action |
| 404 | `NOT_FOUND` | Resource does not exist |
| 409 | `CONFLICT` | Duplicate resource, constraint violation, unresolved duplicates on import |
| 409 | `HAS_FUTURE_ASSIGNMENTS` | Cannot remove referee who has future assignments |
| 409 | `SCHEDULING_CONFLICT` | Assignment would double-book a referee (requires override_conflict) |
| 422 | `VALIDATION_ERROR` | Semantically invalid input; use `fields` to indicate which fields |
| 500 | `INTERNAL_ERROR` | Unexpected server error; never expose stack traces or DB errors |

### HTTP status code rules

- `GET` returning an empty list: `200` with `{"items": []}` — never `404` for an empty list.
- Successful creation: `201`.
- Successful update: `200` with the updated resource.
- Successful delete: `204` (no body).
- Idempotent operations (PUT availability, POST assignment with same referee): `200` if already in the desired state.

---

## 5. Logging Standards

Use `log/slog` with the JSON handler in production. In development, the text handler is acceptable.

```go
// Production setup in main.go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
slog.SetDefault(logger)
```

### What to log

| Event | Level | Required fields |
|-------|-------|----------------|
| Server startup | INFO | port, environment |
| Database connected | INFO | — |
| Migration completed | INFO | version |
| HTTP request | INFO | method, path, status, duration_ms, request_id |
| OAuth2 callback success | INFO | user_id, is_new_user |
| OAuth2 callback failure | WARN | reason (no user data) |
| Assignment created | INFO | match_id, role_slot_id, referee_id, assigned_by |
| Assignment removed | INFO | match_id, role_slot_id, referee_id, removed_by |
| Import committed | INFO | import_id, imported, skipped, errors, created_by |
| Referee status changed | INFO | user_id, old_status, new_status, changed_by |
| Database error | ERROR | operation, error (sanitized — no query parameters) |
| Unexpected error | ERROR | request_id, error |

### What never to log

These fields must never appear in any log line. This is a security and privacy requirement.

- Date of birth (DOB)
- Certification expiry date
- Full names (use user_id instead)
- Email addresses
- Session tokens
- Google `sub` values
- Any OAuth2 tokens (access_token, id_token, code)
- Database connection strings or passwords
- Any value from `r.Header.Get("Cookie")`
- CSV row contents (may contain PII)

### Request ID

Attach a unique `request_id` (UUID) to each incoming HTTP request via middleware. Include it in all log lines for that request and in `500` error responses (for correlation).

```go
// Add to context in middleware
ctx = context.WithValue(r.Context(), ctxKeyRequestID, uuid.New().String())
```

---

## 6. Security Requirements

These rules enforce the security model. Violating any of them is a bug, not a style preference.

### Authentication and authorization

- **Every non-public route must be registered inside a `chi.Router` group that applies `AuthMiddleware`.** There is no per-handler opt-in. If a handler is missing from the protected group, it is unauthenticated.
- **Every assignor-only route must be registered inside a group that also applies `RequireRole("assignor")`.** The role check is in addition to the session check — both must pass.
- **Never trust a role claim from the client.** The role is loaded from the database session lookup. Any field the client sends claiming to represent a role is ignored.
- **Pending referees are blocked from match routes by the `RequireRole("referee", "assignor")` check.** Do not add a special `pending_referee` exception to match routes.

### Eligibility

- **Eligibility is always validated server-side at assignment time**, regardless of what the client claims. The `POST /api/assignments` handler re-runs the eligibility check even if the client previously called `GET /api/matches/{match_id}/eligible-referees`. A referee who becomes ineligible between the list call and the assignment call is rejected.
- **Never skip the eligibility check** because the assignor "already saw the eligible list." The eligible list is a hint; the assignment write is the enforcement point.

### Secrets and credentials

- **No secrets in code.** Google OAuth2 client secret, database password, and session signing key are read exclusively from environment variables.
- **No secrets in Docker images.** Build args or baked-in credentials are forbidden. Use runtime environment variable injection (Azure Container Apps secrets, or `.env` in local compose).
- **No secrets in log output.** See logging standards above.
- **Session tokens are generated with `crypto/rand`.** Do not use `math/rand`.

### Input validation

- **Validate all user input server-side.** Client-side validation in SvelteKit is for UX only. The backend validates independently and returns `422` on invalid input.
- **CSV parsing is defensive.** Unknown columns are ignored; missing required columns return `400`. Malformed values (unparseable date, negative age group) result in a row-level parse error in the preview, not a server panic. Use `recover()` if needed to prevent a malformed CSV from crashing the server.
- **Date and time inputs are validated** for semantic correctness (DOB must be in the past; `start_time` must be before `end_time`; `cert_expiry` must be in the future if certified).

### Transport security

- **TLS is enforced at the reverse proxy (Caddy or Azure Container Apps ingress).** The Go server does not need to handle TLS. In production, the backend listens only on a private port within the container network.
- **The Go server's HTTP listener should be bound to `0.0.0.0:8080` (or a configured port), not `0.0.0.0:443`.** TLS is terminated upstream.
- **The `SESSION_SECRET` env var is required at startup.** If absent, the server must refuse to start with a clear error message.

### Database

- **The database connection string must use `sslmode=require` (or `sslmode=verify-full` if the Azure CA certificate is available).** Plaintext database connections are forbidden in production.
- **Use parameterized queries exclusively.** Never use string concatenation or `fmt.Sprintf` to build SQL queries with user-supplied values. `pgx` parameterized queries handle this correctly by default.

---

## 7. Database Conventions

### Migration tool

`github.com/golang-migrate/migrate/v4` with the PostgreSQL driver and the `iofs` source (embed migration files in the binary using `//go:embed`).

```go
//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(db *sql.DB) error {
    src, _ := iofs.New(migrationsFS, "migrations")
    m, err := migrate.NewWithSourceInstance("iofs", src, "postgres://...")
    if err != nil { return err }
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    return nil
}
```

Migrations run at backend startup, before the HTTP server begins accepting connections.

### Migration file naming

`NNNN_short_description.up.sql` and `NNNN_short_description.down.sql` where `NNNN` is a zero-padded integer starting at `0001`.

Examples:
- `0001_initial_schema.up.sql`
- `0002_add_acknowledged_to_assignments.up.sql`

### Table and column naming

- All names: `snake_case`.
- Table names: plural nouns (`users`, `matches`, `assignments`).
- Column names: descriptive and unambiguous (`start_date` not `date`, `role_type` not `type`).
- Foreign key columns: named `{referenced_table_singular}_id` (e.g., `match_id`, `referee_id`, `role_slot_id`).
- Timestamp columns: `_at` suffix (`created_at`, `assigned_at`, `marked_at`).
- Boolean columns: `is_` prefix or descriptive adjective (`certified`, `available`, `acknowledged`).

### Index naming

- Unique indexes: `uq_{table}_{columns}` (e.g., `uq_users_google_sub`).
- Non-unique indexes: `ix_{table}_{columns}` (e.g., `ix_matches_start_date`).
- Partial indexes: `ix_{table}_{columns}_{condition_summary}` (e.g., `ix_matches_reference_id_notnull`).

### No soft-delete magic

Soft deletes are modeled as explicit `status` columns, not a generic `deleted_at IS NULL` pattern. Queries that should exclude removed referees explicitly filter on `status != 'removed'` or `status = 'active'`. This is intentional and readable.

---

## 8. Forbidden Shortcuts

The following are explicitly forbidden. If a deadline is approaching and one of these seems tempting, cut a feature instead.

| Shortcut | Why it's forbidden |
|----------|--------------------|
| Client-side-only role checks | A user can modify their browser's local storage or cookies. Role must be checked server-side on every request. |
| Client-side-only eligibility checks | Eligibility can change between when the eligible list is fetched and when the assignment is submitted. The server must re-validate at write time. |
| Hardcoded credentials | Any credential in source code is visible to anyone who has read access to the repository. Use environment variables. |
| Skipping eligibility validation because "the UI already filters it" | The UI is a convenience, not a security control. The backend is the enforcement point. |
| Using `math/rand` for session tokens | `math/rand` is predictable. Use `crypto/rand`. |
| Storing OAuth2 tokens in localStorage | LocalStorage is accessible to JavaScript. Session tokens must be in `HttpOnly` cookies. |
| Checking the `role` field from the request body or query param | The role comes from the server-side session, never from client input. |
| Using an ORM's auto-migration | Use `golang-migrate` with explicit SQL files. Auto-migration hides what changes are being made to the schema. |
| Writing audit log entries outside a transaction | The audit log must be written in the same transaction as the assignment change. If they're separate, a crash between them leaves the log inconsistent. |
| Returning stack traces or internal error details in API responses | Return `INTERNAL_ERROR` with a `request_id`. Log the details server-side. |
| Skipping `state` validation in the OAuth2 callback | The `state` parameter is the CSRF token for the OAuth2 flow. |
| Building SQL queries with string concatenation | Use `pgx` parameterized queries (`$1`, `$2`, ...). |
| Committing import rows before all duplicates are resolved | The commit endpoint must verify `unresolved_groups == 0` before writing any rows. |
| Seeding the assignor role via an API endpoint | The assignor role is set via a database command or seed script only. There must be no HTTP endpoint that grants the `assignor` role. |
| Deploying with `APP_ENV=development` settings in production | Production must use `Secure` cookies, TLS connections, and structured JSON logging. |

---

## 9. Environment Variables

All required environment variables must be present at startup. Missing variables must cause the server to log a clear error and exit with a non-zero status code. Do not start the server with defaults for required security parameters.

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string. Must include `sslmode=require` in production. |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth2 client ID. |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth2 client secret. |
| `SESSION_SECRET` | Yes | 32-byte hex-encoded random key for signing the PKCE state cookie. |
| `APP_BASE_URL` | Yes | Public base URL (e.g., `https://refsched.example.com`). Used to build the OAuth2 redirect URI. |
| `PORT` | No | HTTP listen port. Default: `8080`. |
| `LOG_LEVEL` | No | `debug`, `info`, `warn`, `error`. Default: `info`. |
| `APP_ENV` | No | `development` or `production`. Controls cookie `Secure` flag. Default: `production`. |

The `.env.example` file in the repository root lists all variables with placeholder values and a comment for each. Never commit a `.env` file with real values.

---

## 10. Docker Requirements

### Backend Dockerfile (multi-stage)

```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Stage 2: Run
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /server .
EXPOSE 8080
ENTRYPOINT ["/app/server"]
```

- Final image must not include the Go toolchain, source code, or build artifacts.
- `ca-certificates` is required for TLS connections to Azure PostgreSQL and the Google OIDC endpoint.
- `CGO_ENABLED=0` produces a fully static binary compatible with `alpine`.

### Frontend Dockerfile

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM caddy:2-alpine
COPY --from=builder /app/build /srv
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 80 443
```

Caddyfile serves static files and proxies `/api/*` to the backend.

### docker-compose.yml (local development)

Must start three services with a single `docker compose up`:
1. `postgres` — PostgreSQL 16 with persistent volume.
2. `backend` — Go server with `DATABASE_URL` pointed at the compose postgres service.
3. `frontend` — Caddy serving SvelteKit static build with `/api` proxy to backend.

The backend service must depend on the `postgres` service being healthy (use `healthcheck` and `depends_on: condition: service_healthy`).
