# Auth Model: Referee Scheduling App

## 1. Overview

Authentication is Google OAuth2 with PKCE. All identity assertions come from Google's OIDC ID token. The backend maintains server-side sessions stored in PostgreSQL. The browser holds only an opaque session cookie.

No passwords are stored anywhere in the system.

---

## 2. Google OAuth2 PKCE Flow

PKCE (Proof Key for Code Exchange) prevents authorization code interception attacks. Although PKCE is primarily a protection for public clients (mobile apps), applying it to server-side web apps is recommended by current OAuth2 best practices and costs nothing.

### Step-by-step

```
Step 1 — Backend generates PKCE values
───────────────────────────────────────
  code_verifier  = cryptographically random 43-128 char string (URL-safe base64)
  code_challenge = BASE64URL(SHA256(code_verifier))
  state          = cryptographically random string (CSRF token)

  These are stored in a short-lived (10 min), HTTP-only, Secure, SameSite=Lax
  cookie named `oauth_state`.

Step 2 — Backend builds the Google authorization URL
──────────────────────────────────────────────────────
  https://accounts.google.com/o/oauth2/v2/auth
    ?client_id=CLIENT_ID
    &redirect_uri=https://app.example.com/api/auth/callback
    &response_type=code
    &scope=openid email profile
    &state=STATE
    &code_challenge=CODE_CHALLENGE
    &code_challenge_method=S256

Step 3 — User is redirected to Google, grants consent

Step 4 — Google redirects to /api/auth/callback?code=CODE&state=STATE

Step 5 — Backend validates state and exchanges code for tokens
──────────────────────────────────────────────────────────────
  POST https://oauth2.googleapis.com/token
    grant_type=authorization_code
    &code=CODE
    &redirect_uri=https://app.example.com/api/auth/callback
    &client_id=CLIENT_ID
    &client_secret=CLIENT_SECRET
    &code_verifier=CODE_VERIFIER   ← proves possession of verifier

  Google responds with:
    {
      "access_token": "...",
      "id_token": "...",       ← JWT, contains identity claims
      "refresh_token": "...",  ← only on first consent; stored nowhere
      "expires_in": 3600
    }

Step 6 — Backend validates the id_token
────────────────────────────────────────
  Using google.golang.org/api or golang.org/x/oauth2/google:
  - Fetch Google's public JWKS from: https://www.googleapis.com/oauth2/v3/certs
  - Verify JWT signature against the matching public key.
  - Verify claims:
      iss  == "https://accounts.google.com"
      aud  == CLIENT_ID
      exp  > now
      iat  > now - 5 min (clock skew tolerance)

  Extract: sub (stable Google user ID), email, name

Step 7 — Upsert user and create session
────────────────────────────────────────
  BEGIN;
  INSERT INTO users (google_sub, email, display_name, role)
  VALUES ($1, $2, $3, 'pending_referee')
  ON CONFLICT (google_sub)
  DO UPDATE SET email = EXCLUDED.email,
                display_name = EXCLUDED.display_name,
                updated_at = NOW()
  RETURNING id, role;

  INSERT INTO sessions (token, user_id, expires_at)
  VALUES (generate_session_token(), $user_id, NOW() + INTERVAL '30 days')
  RETURNING token;
  COMMIT;

Step 8 — Set cookie and redirect
──────────────────────────────────
  Set-Cookie: session=TOKEN; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=2592000
  302 → /dashboard
```

### Google access_token and refresh_token

- The Google `access_token` is **not stored**. The application has no need to call Google APIs on behalf of the user after the initial identity assertion.
- The `refresh_token` from Google is **not stored**. If the user's Google session expires, they simply re-authenticate (click "Sign in with Google" again).
- The application's own session lifespan (30 days, sliding) is managed entirely server-side.

---

## 3. Session Management

### Decision: Server-side sessions over JWTs

Full rationale in `adr/adr-004-session-management.md`. Summary:

- At 22 users, there is no scalability benefit to stateless JWTs.
- Server-side sessions allow **immediate revocation**: when an assignor deactivates or removes a referee, all active sessions for that user are deleted and the user is locked out on the next request — no waiting for a JWT to expire.
- Session data (user_id, role) is tiny. Looking it up from PostgreSQL on each request is a single indexed primary key read — effectively free at this scale.

### Session table

See `data_model.md` → `sessions` table.

### Session lifecycle

- **Creation**: on successful OAuth2 callback.
- **Renewal (sliding expiry)**: on each authenticated request, if `expires_at` is within 7 days of expiry, `last_seen` and `expires_at` are updated to extend for another 30 days.
- **Expiry**: expired sessions are checked at lookup time. If a session is expired, it is deleted and the request is treated as unauthenticated (`401`).
- **Logout**: the session row is deleted and the cookie is cleared (`Max-Age=0`).
- **User removal**: all sessions for the removed user are deleted before (or atomically with) setting `referee_profiles.status = 'removed'`.

### Cookie settings

```
Set-Cookie: session=<opaque-token>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=2592000
```

| Attribute | Value | Reason |
|-----------|-------|--------|
| `HttpOnly` | set | Prevents JavaScript from reading the cookie; mitigates XSS token theft |
| `Secure` | set | Cookie only sent over HTTPS |
| `SameSite=Lax` | set | Blocks cross-site POST cookie inclusion (CSRF protection); Lax allows GET navigations (needed for OAuth2 redirect back to app) |
| `Path=/` | set | Cookie sent on all requests to the domain |
| `Max-Age=2592000` | 30 days | Browser persists cookie across browser restarts |

The cookie value is an opaque, cryptographically random token (32 bytes from `crypto/rand`, base64url-encoded = 43 chars). It carries no embedded data.

---

## 4. Authentication Middleware

Every non-public route must pass through the auth middleware. The middleware is registered on the router at the group level — there is no per-handler opt-in.

### Middleware logic (pseudocode)

```go
func AuthMiddleware(db *pgxpool.Pool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("session")
            if err != nil {
                writeJSON(w, 401, errorBody("NOT_AUTHENTICATED", "Login required"))
                return
            }

            session, user, err := db.LookupSession(r.Context(), cookie.Value)
            if err != nil || session.ExpiresAt.Before(time.Now()) {
                clearSessionCookie(w)
                writeJSON(w, 401, errorBody("SESSION_EXPIRED", "Session expired"))
                return
            }

            // Extend sliding session if needed
            if time.Until(session.ExpiresAt) < 7*24*time.Hour {
                db.ExtendSession(r.Context(), session.ID)
            }

            // Attach user to context
            ctx := context.WithValue(r.Context(), ctxKeyUser, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

The `user` object attached to the context contains: `ID`, `Role`, `ProfileStatus`.

### Role middleware

Applied after `AuthMiddleware` for role-restricted routes:

```go
func RequireRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := UserFromContext(r.Context())
            for _, role := range roles {
                if user.Role == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            writeJSON(w, 403, errorBody("FORBIDDEN", "Insufficient role"))
        })
    }
}
```

Usage:
```go
r.Group(func(r chi.Router) {
    r.Use(AuthMiddleware(db))
    r.Use(RequireRole("assignor"))
    r.Get("/api/referees", handleListReferees)
    r.Patch("/api/referees/{id}", handleUpdateReferee)
    // ...
})
```

---

## 5. Pending Referee Access Gates

New users are created with `role = 'pending_referee'`. They can:
- Log in.
- View and edit their own profile (`GET /api/profile`, `PUT /api/profile`).
- View the pending activation screen.

They cannot:
- View any match data.
- Mark availability.
- Be assigned to matches.

This is enforced at the router group level. Match-related routes require `RequireRole("referee", "assignor")`. A `pending_referee` user fails this check with `403`.

```go
r.Group(func(r chi.Router) {
    r.Use(AuthMiddleware(db))
    r.Use(RequireRole("referee", "assignor"))
    r.Get("/api/matches", handleListMatches)
    r.Put("/api/availability/{match_id}", handleSetAvailability)
    // ...
})
```

There is no special `pending` middleware — the existing role check handles it. The frontend uses `GET /api/auth/me` to determine the user's role and renders the pending activation page for `pending_referee` users.

---

## 6. Token Refresh Strategy

The application does not use Google's `refresh_token`. There is nothing to refresh.

The application's own session uses a sliding 30-day expiry. As long as a referee opens the app at least once every 30 days, their session stays valid. If the session expires (e.g., during the off-season), the user clicks "Sign in with Google" again and a new session is created. Google's "Sign in" button flow typically completes without showing a consent screen again (Google recognizes the previously-consented app).

This is the simplest possible token strategy for a web app at this scale, and it is completely correct.

---

## 7. Security Checklist

| Check | Implementation |
|-------|---------------|
| Passwords stored | Never — Google OAuth2 only |
| State parameter validation | Yes — random state stored in PKCE cookie, validated in callback |
| PKCE code_verifier | Generated and stored server-side, sent only in token exchange |
| id_token signature validation | Yes — validated against Google's JWKS endpoint |
| id_token claims validated | Yes — iss, aud, exp, iat |
| Google access_token stored | No — not needed |
| Google refresh_token stored | No — not needed |
| Session cookie HttpOnly | Yes |
| Session cookie Secure | Yes |
| Session cookie SameSite | Lax |
| Session data in cookie | No — cookie contains only opaque token |
| Immediate revocation on removal | Yes — sessions deleted on user removal |
| Role checked server-side | Yes — middleware on every protected route group |
| Client-side role bypass possible | No — client receives no role claim |
| Pending referee blocked from matches | Yes — RequireRole middleware |
