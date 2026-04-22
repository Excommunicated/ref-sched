# ADR-004: Session Management

## Status: Accepted

## Context

After a user authenticates via Google OAuth2, the application needs a mechanism to identify the user on subsequent requests. Two mainstream approaches exist:

1. **Server-side sessions**: The backend generates an opaque random token, stores session metadata (user_id, expiry) in the database, and gives the token to the browser in a cookie. On each request, the backend looks up the token to retrieve the user.

2. **JSON Web Tokens (JWTs)**: The backend signs a JWT containing the user's claims (user_id, role, expiry) and gives it to the browser. On each request, the backend verifies the JWT signature — no database lookup required.

## Decision

**Server-side sessions stored in PostgreSQL, delivered via an HTTP-only cookie.**

## Rationale

### Immediate revocation

When an assignor deactivates or removes a referee, that change must take effect immediately. With server-side sessions, the backend deletes the session row and the user's next request returns `401`. With JWTs, a deactivated user continues to have a valid token until it expires (typically 15 min–1 hour). Workarounds (token blocklist, short expiry + refresh) add complexity that negates the statelessness benefit.

For this application — where an assignor removing a referee expects that referee to be locked out now, not in 15 minutes — immediate revocation is a hard requirement.

### Simplicity

At 22 users, the "statelessness benefit" of JWTs (avoiding a database lookup per request) is irrelevant. The database is already queried on every authenticated API call for business data. One additional indexed lookup on `sessions(token)` is not a bottleneck.

JWTs require: token signing key management, key rotation strategy, clock skew handling, and either a blocklist (stateful anyway) or a refresh token flow. None of these complexities are present with server-side sessions.

### No sensitive data in the cookie

The session cookie contains only an opaque random token. No user data, no role, no claims. Even if a session cookie were intercepted (mitigated by `HttpOnly; Secure; SameSite=Lax`), the attacker gets only a token that can be invalidated server-side.

### Session store

Sessions are stored in the `sessions` table in PostgreSQL. At 22 users with a 30-day session lifetime, this table will have at most a few dozen rows. A periodic cleanup (run at startup or on logout) removes expired rows. There is no operational overhead.

## Consequences

**Positive:**
- Immediate revocation of deactivated/removed users.
- No JWT signing key management.
- No token refresh flow.
- Cookie carries no sensitive information.
- Session lookup is a single indexed read — fast enough for any scale this app will reach.
- No client-side session data to tamper with.

**Negative:**
- Requires a database lookup on every authenticated request. Acceptable at this scale.
- Sessions are not portable across multiple backend instances without a shared database. Not relevant — horizontal scaling is explicitly out of scope for v1.
- If the database is unavailable, authenticated users cannot access the app. This is true of any architecture where the database is required for business operations (which it always is for this app).

## Alternatives Considered

**JWTs (short-lived + refresh token)**: Common pattern for SPAs. Adds: a refresh token store (stateful), a token rotation strategy, clock skew handling, and a blocklist or short expiry window for revocation. All of this complexity exists to avoid a session database lookup — a tradeoff that makes no sense for 22 users. The revocation problem (a removed referee stays logged in until the JWT expires) is a concrete correctness issue, not a theoretical one.

**JWTs (long-lived, no refresh)**: Simpler, but revocation is impossible without a blocklist. Unacceptable given the user lifecycle requirements.

**Cookie-based session with Redis as the session store**: A reasonable choice for applications that want fast session lookups without hitting the primary database. Adds operational complexity (another service to run). Not warranted at this scale.
