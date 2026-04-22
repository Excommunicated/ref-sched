# Architecture: Referee Scheduling App

## 1. System Overview

The Referee Scheduling App is a multi-user web application serving approximately 22 users (1–2 assignors, ~20 referees) for a single club soccer association. The system replaces a manual email-based process with an eligibility-aware, role-gated web interface.

### Component Diagram

```
                        ┌─────────────────────────────────────────────┐
                        │                  Azure Cloud                │
                        │                                             │
  ┌──────────┐  HTTPS   │  ┌──────────────┐      ┌────────────────┐  │
  │ Browser  │◄────────►│  │   SvelteKit  │      │   Go Backend   │  │
  │(Mobile / │          │  │   Frontend   │◄────►│   (REST API)   │  │
  │ Desktop) │          │  │ (Static SPA) │ JSON │                │  │
  └──────────┘          │  └──────────────┘      └───────┬────────┘  │
                        │     Docker Container           │           │
                        │                               │ pgx        │
                        │                    ┌──────────▼────────┐   │
                        │                    │ Azure Database for │   │
                        │                    │    PostgreSQL      │   │
                        │                    │   (Flexible Srv)  │   │
                        │                    └───────────────────┘   │
                        └─────────────────────────────────────────────┘
                                       │
                                       │ OAuth2
                                       ▼
                              ┌─────────────────┐
                              │   Google OAuth2  │
                              │   (accounts.    │
                              │   google.com)   │
                              └─────────────────┘
```

### Service Roles

| Component | Technology | Responsibility |
|-----------|-----------|----------------|
| Frontend | SvelteKit (static adapter, served by Caddy or nginx) | UI rendering, form handling, API calls |
| Backend | Go 1.23+, `chi` router | REST API, business logic, eligibility engine, OAuth2 exchange |
| Database | PostgreSQL 16 (Azure Database for PostgreSQL Flexible Server) | Persistent state, relational integrity |
| Auth Provider | Google OAuth2 / OIDC | Identity assertion only |
| Reverse Proxy | Caddy (sidecar container) | TLS termination, HTTPS redirect, static asset serving |

---

## 2. Technology Choices

### Backend: Go

Go is the developer's primary language. Its standard library covers the vast majority of what this application needs. The ecosystem for HTTP servers, PostgreSQL access, OAuth2, and CSV parsing is mature. Compilation produces a single statically linked binary that runs in a minimal Docker image. No runtime dependency management at execution time.

### Database: PostgreSQL 16

The data model is highly relational: referees have profiles, matches have role slots, role slots have assignments, assignments have audit logs. PostgreSQL handles this naturally. The developer is fluent in SQL. Azure Database for PostgreSQL Flexible Server provides managed backups, connection pooling (pgBouncer built-in), and point-in-time restore at low cost within the free credit allocation.

### Frontend: SvelteKit

SvelteKit is chosen over Next.js for this project. Full rationale is in `adr/adr-003-frontend-framework.md`. In summary: Svelte's component model is simpler to learn for a developer with limited modern JS/TS experience; the compiler produces smaller bundles; and the routing and form conventions are straightforward. The static adapter emits plain HTML/CSS/JS files that can be served by any web server or CDN — no Node.js runtime in production.

### Deployment: Azure (Docker containers)

Azure Container Apps or Azure App Service (Containers) can run the pre-built Docker images. Azure Database for PostgreSQL Flexible Server manages the database. This keeps infrastructure overhead low for a single developer while using existing free credits. Docker containers ensure the same image runs locally (via docker-compose) and in Azure without environment-specific changes.

### Reverse Proxy: Caddy

Caddy handles automatic TLS certificate management (Let's Encrypt), HTTP-to-HTTPS redirect, and serves the built SvelteKit static files. It runs as a sidecar container or can be replaced by Azure-native TLS if using Azure Container Apps with built-in ingress.

---

## 3. Major Data Flows

### 3.1 Login Flow

```
Browser                 SvelteKit              Go Backend          Google OAuth2
   │                       │                       │                    │
   │  GET /                │                       │                    │
   │──────────────────────►│                       │                    │
   │  (render login page)  │                       │                    │
   │◄──────────────────────│                       │                    │
   │                       │                       │                    │
   │  Click "Sign in"      │                       │                    │
   │──────────────────────►│                       │                    │
   │                       │  GET /api/auth/login  │                    │
   │                       │──────────────────────►│                    │
   │                       │  (generate PKCE       │                    │
   │                       │   code_verifier,      │                    │
   │                       │   state, store        │                    │
   │                       │   in session)         │                    │
   │                       │  302 → Google URL     │                    │
   │◄──────────────────────│◄──────────────────────│                    │
   │                                                                    │
   │  Redirect to Google consent screen                                 │
   │───────────────────────────────────────────────────────────────────►│
   │                                                                    │
   │  User grants consent → redirect to /api/auth/callback?code=...    │
   │◄───────────────────────────────────────────────────────────────────│
   │                                                                    │
   │  GET /api/auth/callback?code=...&state=...                         │
   │──────────────────────────────────────────►│                        │
   │                                           │ POST token exchange    │
   │                                           │───────────────────────►│
   │                                           │  (access_token,        │
   │                                           │   id_token)            │
   │                                           │◄───────────────────────│
   │                                           │                        │
   │                                           │ Validate id_token      │
   │                                           │ Upsert user record     │
   │                                           │ Create server session  │
   │                                           │ Set session cookie     │
   │  302 → /dashboard (role-appropriate)      │                        │
   │◄──────────────────────────────────────────│                        │
```

### 3.2 CSV Import Flow

```
Assignor Browser             Go Backend                  PostgreSQL
      │                          │                            │
      │  POST /api/matches/import│                            │
      │  (multipart: csv file)   │                            │
      │─────────────────────────►│                            │
      │                          │  Parse CSV rows            │
      │                          │  Extract age groups        │
      │                          │  Detect duplicate signals: │
      │                          │    - same reference_id     │
      │                          │    - same date+time+loc    │
      │                          │  Check existing reference_ │
      │                          │  ids in DB                 │
      │                          │──────────────────────────►│
      │                          │◄──────────────────────────│
      │                          │                            │
      │  200 {                   │                            │
      │    import_id,            │                            │
      │    preview_rows[],       │                            │
      │    duplicate_groups[],   │                            │
      │    parse_errors[]        │                            │
      │  }                       │                            │
      │◄─────────────────────────│                            │
      │                          │                            │
      │  (Assignor resolves      │                            │
      │   duplicates in UI)      │                            │
      │                          │                            │
      │  POST /api/matches/import│                            │
      │       /{import_id}/commit│                            │
      │  { resolutions: [...] }  │                            │
      │─────────────────────────►│                            │
      │                          │  Validate all duplicates   │
      │                          │  resolved                  │
      │                          │  BEGIN TRANSACTION         │
      │                          │  Insert accepted rows      │
      │                          │  Insert role_slots         │
      │                          │  COMMIT                    │
      │                          │──────────────────────────►│
      │                          │◄──────────────────────────│
      │                          │                            │
      │  200 {                   │                            │
      │    imported: N,          │                            │
      │    skipped: M,           │                            │
      │    errors: K             │                            │
      │  }                       │                            │
      │◄─────────────────────────│                            │
```

Note: The import is a two-step operation. Step 1 (parse + preview) stores intermediate state in a short-lived table or in-memory (Redis is not used; at this scale a PostgreSQL staging table is used with a TTL-based cleanup job or simple cleanup on next import by same user). Step 2 (commit) writes all accepted rows atomically.

### 3.3 Availability Marking Flow

```
Referee Browser              Go Backend                 PostgreSQL
      │                          │                            │
      │  GET /api/matches?eligible=true                       │
      │─────────────────────────►│                            │
      │                          │  Query: matches where      │
      │                          │  referee passes eligibility│
      │                          │  rules (inline SQL check   │
      │                          │  against DOB, cert_expiry, │
      │                          │  age_group, role_type)     │
      │                          │──────────────────────────►│
      │                          │◄──────────────────────────│
      │  200 { matches: [...] }  │                            │
      │◄─────────────────────────│                            │
      │                          │                            │
      │  PUT /api/availability/  │                            │
      │    {match_id}            │                            │
      │  { available: true }     │                            │
      │─────────────────────────►│                            │
      │                          │  Check: referee is active  │
      │                          │  Check: match is not past  │
      │                          │  Check: match not cancelled│
      │                          │  Check: referee eligible   │
      │                          │  UPSERT referee_availability│
      │                          │──────────────────────────►│
      │                          │◄──────────────────────────│
      │  200 { available: true } │                            │
      │◄─────────────────────────│                            │
```

### 3.4 Assignment Flow

```
Assignor Browser             Go Backend                  PostgreSQL
      │                          │                            │
      │  GET /api/matches/{id}/  │                            │
      │    eligible-referees?    │                            │
      │    role_slot_id={sid}    │                            │
      │─────────────────────────►│                            │
      │                          │  Fetch slot → role_type,   │
      │                          │  match age_group, date     │
      │                          │  Run eligibility query:    │
      │                          │   - filter by role rules   │
      │                          │   - mark availability      │
      │                          │   - compute age at match   │
      │                          │   - flag conflicts         │
      │                          │──────────────────────────►│
      │                          │◄──────────────────────────│
      │  200 {                   │                            │
      │    available_eligible[], │                            │
      │    eligible_only[]       │                            │
      │  }                       │                            │
      │◄─────────────────────────│                            │
      │                          │                            │
      │  POST /api/assignments   │                            │
      │  { match_id, role_slot_id│                            │
      │    referee_id }          │                            │
      │─────────────────────────►│                            │
      │                          │  Re-validate eligibility   │
      │                          │  Check conflict            │
      │                          │  BEGIN TRANSACTION         │
      │                          │  Upsert assignment row     │
      │                          │  Insert audit_log row      │
      │                          │  COMMIT                    │
      │                          │──────────────────────────►│
      │                          │◄──────────────────────────│
      │  200 { assignment }      │                            │
      │◄─────────────────────────│                            │
```

---

## 4. Non-Functional Considerations

### 4.1 Security

- **Authentication**: Google OAuth2 with PKCE. No passwords stored anywhere in the system.
- **Session**: HTTP-only, Secure, SameSite=Lax cookies containing an opaque session ID. Session data (user_id, role) lives server-side in PostgreSQL (`sessions` table). No sensitive data in the cookie itself.
- **Authorization**: Middleware on every protected route checks the session and resolves role. No route trusts a client-supplied role claim. Pending referees are explicitly blocked from match-related routes at the middleware layer.
- **Eligibility**: Eligibility is always computed server-side at assignment time. The API endpoint for the eligible-referee list computes it; the assignment endpoint re-validates it before writing. There is no client-only eligibility check.
- **Input validation**: All user input validated and sanitized server-side. CSV parsing is defensive: unknown columns are ignored, malformed rows are reported as errors, not panicked on.
- **PII handling**: DOB and certification data are stored. No government ID, no payment data. Logs must never include DOB, certification details, or full names (see `builder_guardrails.md`).
- **TLS**: Enforced at the reverse proxy (Caddy or Azure-native ingress). The Go backend does not need to handle TLS directly.
- **Secrets**: All credentials (Google OAuth2 client ID/secret, DB connection string, session signing key) are injected via environment variables. No secrets in code or Docker images.
- **CSRF**: SameSite=Lax cookie provides baseline CSRF protection for same-site flows. For the OAuth2 callback, the `state` parameter serves as a CSRF token.

### 4.2 Performance

At 22 users there are no meaningful performance concerns. The following are still considered good practice:

- **Eligibility queries**: Eligibility is computed via SQL (not application-side loops). Indexes on `referee_profiles(dob, cert_expiry)`, `matches(start_date)`, and `referee_availability(referee_id, match_id)` keep queries fast even as data grows.
- **Assignment list query**: A single JOIN query fetches match + slots + current assignments in one round trip.
- **No caching layer**: Redis/memcached is not warranted. PostgreSQL query performance is sufficient for this scale.
- **Static assets**: SvelteKit static build is served by Caddy with gzip compression and proper cache headers. JS/CSS bundles are small.

### 4.3 Auditability

- Every assignment, reassignment, and removal writes a row to `assignment_audit_log` within the same database transaction as the assignment change. This means the audit log is always consistent with the `assignments` table.
- Match edits are not individually audit-logged in v1 (story 3.4 notes "timestamp and acting assignor" on edits — `matches` table has `updated_by` and `updated_at` columns that satisfy this at the record level).
- The audit log is retained indefinitely in v1. Archiving is a future concern.

### 4.4 Availability and Deployment

- Downtime during deployments is explicitly acceptable (PRD NFR: availability is Low priority).
- Azure Container Apps supports rolling updates but a simple stop/start is acceptable for this scale.
- Database migrations run automatically at backend startup using `golang-migrate`. Migrations are forward-only in v1 (no rollback scripts required but not forbidden).
- Daily automated backups via Azure Database for PostgreSQL Flexible Server are sufficient.

---

## 5. Assumptions and Tradeoffs

| Assumption / Tradeoff | Rationale |
|----------------------|-----------|
| No horizontal scaling | 22 users; a single container is sufficient for the lifetime of v1. |
| Server-side sessions over JWTs | Simpler revocation (deactivating a user immediately blocks access); no token refresh complexity; session data is tiny. Full rationale in `adr/adr-004-session-management.md`. |
| SvelteKit static adapter | No Node.js runtime in production; simpler deployment model. Full rationale in `adr/adr-003-frontend-framework.md`. |
| Import staging in PostgreSQL | Avoids adding Redis; simpler operational model at this scale. |
| No email notifications | Out of scope per PRD. The system does not send any outbound email in v1. |
| Eligibility computed at query time | Not cached or pre-computed. Profile changes and match date changes automatically reflect without a separate sync job. Acceptable at this scale. |
| Last-write-wins on concurrent assignor edits | Two assignors can both assign simultaneously without a lock. For 1–2 assignors, this is an acceptable tradeoff over adding pessimistic locking complexity. |
