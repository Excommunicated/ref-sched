# ADR-001: Backend Language

## Status: Accepted

## Context

The application needs a backend HTTP server that handles REST API requests, implements OAuth2, runs business logic (eligibility engine, CSV parsing), and communicates with PostgreSQL. The backend must be maintainable by a single developer over multiple months and be deployable as a Docker container to Azure.

The developer is fluent in Go and has used it as their preferred backend language. Python was explicitly listed as a language to avoid. Node.js is an option but the developer's TypeScript/JavaScript experience is limited.

## Decision

Go (version 1.23 or later) is the backend language.

## Consequences

**Positive:**
- Developer fluency means faster implementation and fewer bugs in unfamiliar idioms.
- Go compiles to a single static binary; the Docker image can be based on `scratch` or `alpine`, keeping it minimal.
- The standard library provides `net/http`, `encoding/csv`, `crypto/rand`, and `crypto/subtle` — covering the majority of what this app needs without external dependencies.
- Strong concurrency primitives are available if needed (not required for this scale).
- `golang.org/x/oauth2` provides first-class OAuth2 support.
- `github.com/jackc/pgx/v5` is a high-quality, well-maintained PostgreSQL driver.
- `github.com/go-chi/chi/v5` provides lightweight, idiomatic HTTP routing.
- `github.com/golang-migrate/migrate/v4` handles database migrations.
- Excellent tooling: `go vet`, `staticcheck`, `golangci-lint`, `go test`.
- Fast compile times support a tight development loop.

**Negative:**
- No generics-heavy framework magic to reduce boilerplate — handler code is explicit, which is a feature at this scale but slightly more verbose than, say, a Spring Boot or Django app.
- Error handling is explicit (`if err != nil`); this is idiomatic but unfamiliar to developers from exception-based languages. Not a concern here given the developer's fluency.

## Alternatives Considered

**Python (FastAPI / Flask)**: Explicitly avoided per developer preference.

**Node.js / TypeScript (Express / Fastify)**: The developer has passing JS/TS experience. Using Node.js for the backend would mean the developer is less comfortable on both ends (frontend and backend). Go backend + SvelteKit frontend is a cleaner division where the developer's strongest skill (Go) handles the complex business logic.

**Ruby on Rails**: No stated familiarity. Would be a significant learning investment.
