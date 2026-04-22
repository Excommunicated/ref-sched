# API Contracts: Referee Scheduling App

## Conventions

- **Base path**: `/api`
- **Format**: All request and response bodies are `application/json` unless noted (file upload uses `multipart/form-data`).
- **Auth levels**:
  - `public` — no session required
  - `pending` — any authenticated user (including pending_referee)
  - `referee` — authenticated user with role `referee` or `assignor`
  - `assignor` — authenticated user with role `assignor` only
- **Error format** (all error responses):
  ```json
  {
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "Human-readable description",
      "fields": { "field_name": "field-specific error" }  // optional, for 422
    }
  }
  ```
- **Common status codes**:
  - `200` — success (with body)
  - `201` — created
  - `204` — success (no body)
  - `400` — bad request (malformed JSON, invalid file type)
  - `401` — not authenticated
  - `403` — authenticated but insufficient role or access denied
  - `404` — resource not found
  - `409` — conflict (duplicate, constraint violation)
  - `422` — validation error (semantically invalid input)
  - `500` — internal server error

---

## Auth

### `GET /api/auth/login`

Initiates the Google OAuth2 PKCE flow.

**Auth**: public

**Response**: `302` redirect to Google authorization URL.

Sets a short-lived (10 min) cookie containing the PKCE `code_verifier` and `state` value for CSRF validation.

---

### `GET /api/auth/callback`

OAuth2 callback. Google redirects here after the user grants consent.

**Auth**: public

**Query params**:
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | string | yes | Authorization code from Google |
| `state` | string | yes | Must match the state stored in the PKCE cookie |
| `error` | string | no | Set by Google on denial; backend redirects to `/` with error param |

**Success flow**:
1. Validate `state` against PKCE cookie value.
2. Exchange `code` for tokens via Google token endpoint.
3. Validate `id_token` (signature, issuer, audience, expiry).
4. Extract `sub`, `email`, `name` from `id_token` claims.
5. Upsert user: if `google_sub` exists, reuse record; otherwise create with `role = 'pending_referee'`.
6. Create session, set `Set-Cookie: session=<token>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=2592000`.
7. `302` redirect to `/dashboard` (role-appropriate frontend route).

**Error cases**:
- State mismatch → `302 /login?error=csrf`
- Google returns `error` → `302 /login?error=oauth_denied`
- Token validation failure → `302 /login?error=token_invalid`
- DB error → `500`

---

### `POST /api/auth/logout`

Destroys the current session.

**Auth**: pending (any authenticated user)

**Request body**: none

**Response**: `204`

Deletes the session row from the database and sets `Set-Cookie: session=; Max-Age=0`.

---

### `GET /api/auth/me`

Returns the current user's identity and role. Used by the frontend on load to determine routing.

**Auth**: pending

**Response** `200`:
```json
{
  "id": 1,
  "email": "user@example.com",
  "display_name": "Jane Smith",
  "role": "referee",
  "profile_complete": true
}
```

`profile_complete` is `true` if a `referee_profiles` row exists for this user. Used to redirect pending referees to profile setup.

**Error**: `401` if no valid session.

---

## Users / Profile

### `GET /api/profile`

Returns the authenticated user's referee profile.

**Auth**: pending

**Response** `200`:
```json
{
  "user_id": 1,
  "first_name": "Jane",
  "last_name": "Smith",
  "dob": "2005-03-15",
  "certified": true,
  "cert_expiry": "2026-12-31",
  "grade": "mid",
  "status": "active"
}
```

`grade` is `null` if not yet set by an assignor. Referees receive `grade: null` in their own profile response — the field is present but never editable by the referee.

**Error**: `404` if no profile exists yet.

---

### `PUT /api/profile`

Creates or updates the authenticated user's referee profile. Referees can update their own profile fields only (not `grade` or `status`).

**Auth**: pending

**Request body**:
```json
{
  "first_name": "Jane",
  "last_name": "Smith",
  "dob": "2005-03-15",
  "certified": true,
  "cert_expiry": "2026-12-31"
}
```

**Validation**:
- `first_name`, `last_name`: non-empty, max 100 chars.
- `dob`: valid date, must be in the past.
- `certified`: boolean, required.
- `cert_expiry`: required if `certified = true`; must be a future date (relative to today).

**Response** `200`: same shape as `GET /api/profile`.

**Error cases**:
- `422` with field errors for validation failures.

---

## Referee Management (Assignor)

### `GET /api/referees`

Lists all referees with their profile and status. Assignor-only.

**Auth**: assignor

**Query params**:
| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter by `pending`, `active`, `inactive`, `removed`. Multiple allowed: `?status=pending&status=active` |
| `search` | string | Case-insensitive substring match on first_name + last_name |

**Response** `200`:
```json
{
  "referees": [
    {
      "user_id": 2,
      "email": "ref@example.com",
      "display_name": "Bob Jones",
      "first_name": "Bob",
      "last_name": "Jones",
      "dob": "2010-06-20",
      "certified": true,
      "cert_expiry": "2025-12-31",
      "cert_status": "expired",
      "grade": "junior",
      "status": "active"
    }
  ]
}
```

`cert_status` computed by server: `"none"` | `"active"` | `"expiring_soon"` (within 30 days) | `"expired"`.

---

### `GET /api/referees/{user_id}`

Returns a single referee's full profile. Assignor-only.

**Auth**: assignor

**Response** `200`: same shape as a single item from `GET /api/referees`.

**Error**: `404` if user does not exist or has no profile.

---

### `PATCH /api/referees/{user_id}`

Updates assignor-controlled fields: `status` and/or `grade`.

**Auth**: assignor

**Request body** (all fields optional; send only what changes):
```json
{
  "status": "active",
  "grade": "mid"
}
```

**Validation**:
- `status`: one of `active`, `inactive`, `removed`.
- `grade`: one of `junior`, `mid`, `senior`, or `null` to clear.
- Cannot set `status = removed` if the referee has future assignments (returns `409` with `"code": "HAS_FUTURE_ASSIGNMENTS"`).

**Side effects**:
- `status = active` on a `pending` referee: also sets `users.role = 'referee'` in the same transaction.
- `status = removed`: also deletes all active sessions for that user.

**Response** `200`: updated referee profile (same shape as `GET /api/referees/{user_id}`).

**Error cases**:
- `404` — referee not found.
- `409` — cannot remove referee with future assignments.
- `422` — invalid status or grade value.

---

## Matches

### `GET /api/matches`

Returns the match schedule. Response shape differs by role.

**Auth**: referee (active referees and assignors)

**Query params**:
| Param | Type | Role | Description |
|-------|------|------|-------------|
| `from` | date (YYYY-MM-DD) | any | Start of date range filter (inclusive). Defaults to today. |
| `to` | date (YYYY-MM-DD) | any | End of date range filter (inclusive). |
| `age_group` | int | assignor | Filter by age group (e.g. `12`). |
| `assignment_status` | string | assignor | `unassigned`, `partial`, `full`. |
| `show_cancelled` | bool | assignor | Include cancelled matches. Default `false`. |
| `eligible` | bool | referee | If `true`, return only matches the authenticated referee is eligible for. |

**Response** `200` (assignor view):
```json
{
  "matches": [
    {
      "id": 101,
      "reference_id": "30663414",
      "event_name": "U12G Super Squirrels v Falcons",
      "age_group": 12,
      "gender": "girls",
      "start_date": "2026-04-25",
      "start_time": "08:30",
      "end_time": "10:15",
      "location": "10945 Rogers Cir",
      "venue_detail": "Shakerag Park Turf Field",
      "status": "scheduled",
      "total_slots": 3,
      "filled_slots": 1,
      "assignment_status": "partial"
    }
  ]
}
```

**Response** `200` (referee view): same shape minus `assignment_status`/slot counts. Includes `is_available: bool` and `is_assigned: bool` per match.

---

### `GET /api/matches/{id}`

Returns a single match with full details including role slots and current assignments.

**Auth**: referee

**Response** `200`:
```json
{
  "id": 101,
  "reference_id": "30663414",
  "event_name": "U12G Super Squirrels v Falcons",
  "age_group": 12,
  "gender": "girls",
  "start_date": "2026-04-25",
  "start_time": "08:30",
  "end_time": "10:15",
  "location": "10945 Rogers Cir",
  "venue_detail": "Shakerag Park Turf Field",
  "description": "Meet 8.30\nKO 9am\nShakerag Park Turf Field\n(Not Georgia Express Soccer Complex)",
  "status": "scheduled",
  "role_slots": [
    {
      "id": 501,
      "role_type": "center",
      "slot_number": 1,
      "assignment": {
        "id": 301,
        "referee_id": 2,
        "first_name": "Bob",
        "last_name": "Jones",
        "grade": "junior",
        "assigned_at": "2026-04-20T14:30:00Z",
        "acknowledged": false
      }
    },
    {
      "id": 502,
      "role_type": "assistant",
      "slot_number": 1,
      "assignment": null
    },
    {
      "id": 503,
      "role_type": "assistant",
      "slot_number": 2,
      "assignment": null
    }
  ],
  "availability_count": 5
}
```

`availability_count` is the number of active referees who have marked availability for this match (assignor view only).

---

### `POST /api/matches`

Manually creates a single match.

**Auth**: assignor

**Request body**:
```json
{
  "event_name": "U12B Manual Match",
  "age_group": 12,
  "gender": "boys",
  "start_date": "2026-05-10",
  "start_time": "09:00",
  "end_time": "10:45",
  "location": "Jones Bridge Park",
  "venue_detail": "Field 3",
  "description": "Meet 9am KO 9:30am"
}
```

**Validation**: All required fields except `venue_detail` and `description`. `start_time` must be before `end_time`.

**Response** `201`: newly created match (same shape as `GET /api/matches/{id}`).

---

### `PUT /api/matches/{id}`

Updates a match's details. Triggers re-evaluation of role slots and assignment eligibility if `age_group` changes.

**Auth**: assignor

**Request body**: same shape as `POST /api/matches` (full replacement).

**Response** `200`:
```json
{
  "match": { /* updated match */ },
  "eligibility_warnings": [
    {
      "role_slot_id": 501,
      "referee_id": 2,
      "referee_name": "Bob Jones",
      "reason": "Certification expired before new match date"
    }
  ]
}
```

`eligibility_warnings` is an empty array if no assigned referees are affected. The assignor must decide what to do with flagged assignments — the API does not automatically remove them.

**Error cases**:
- `404` — match not found.
- `422` — validation error.

---

### `PATCH /api/matches/{id}/status`

Cancels or un-cancels a match.

**Auth**: assignor

**Request body**:
```json
{ "status": "cancelled" }
```
or
```json
{ "status": "scheduled" }
```

**Response** `200`: updated match.

---

### `POST /api/matches/{id}/role-slots`

Adds an assistant slot to a U10 match. Returns `409` if the match already has 2 assistant slots.

**Auth**: assignor

**Request body**:
```json
{ "role_type": "assistant" }
```

**Response** `201`: the new slot:
```json
{ "id": 504, "role_type": "assistant", "slot_number": 2, "assignment": null }
```

**Error cases**:
- `409` — slot limit reached for this age group.
- `422` — match age group does not allow assistant slot management (U6/U8 returns 422).

---

### `DELETE /api/matches/{id}/role-slots/{slot_id}`

Removes an assistant slot from a U10 match. The slot must have no current assignment.

**Auth**: assignor

**Response** `204`

**Error cases**:
- `409` — slot is currently assigned; cannot delete.
- `422` — slot is not an assistant type or not on a U10 match.

---

## CSV Import

### `POST /api/matches/import`

Stage 1: Upload and parse a CSV file. Returns a preview with all rows, parse errors, and duplicate groups. No data is written to `matches`.

**Auth**: assignor

**Request**: `multipart/form-data`, field name `file`.

**Response** `200`:
```json
{
  "import_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "summary": {
    "total_rows": 30,
    "parseable_rows": 28,
    "parse_errors": 2,
    "duplicate_groups": 3
  },
  "rows": [
    {
      "row_index": 0,
      "status": "ok",
      "parsed": {
        "event_name": "U12G Super Squirrels v Falcons",
        "team_name": "Under 12 Girls - Falcons",
        "age_group": 12,
        "gender": "girls",
        "start_date": "2026-04-25",
        "start_time": "08:30",
        "end_time": "10:15",
        "location": "10945 Rogers Cir",
        "venue_detail": "Shakerag Park Turf Field",
        "reference_id": "30663414"
      },
      "duplicate_group": null,
      "duplicate_signal": null
    },
    {
      "row_index": 5,
      "status": "parse_error",
      "parsed": null,
      "parse_error": "Cannot extract age group from team_name: 'Referees Only'",
      "duplicate_group": null,
      "duplicate_signal": null
    },
    {
      "row_index": 10,
      "status": "duplicate",
      "parsed": { "...": "..." },
      "duplicate_group": "dt:2026-04-25T11:30:West Gwinnett Park Aquatic Center",
      "duplicate_signal": "date_time_location"
    }
  ],
  "duplicate_groups": [
    {
      "group_key": "dt:2026-04-25T11:30:West Gwinnett Park Aquatic Center",
      "signal": "date_time_location",
      "row_indexes": [10, 11],
      "existing_match_id": null
    }
  ]
}
```

**Error cases**:
- `400` — file is not a CSV.
- `400` — CSV has no rows or is missing required columns (`event_name`, `team_name`, `start_date`, `start_time`, `end_time`, `location`, `reference_id`).
- `409` — an in-progress import already exists for this user (returns existing `import_id`).

---

### `PATCH /api/matches/import/{import_id}/resolutions`

Updates the resolution for one or more duplicate groups.

**Auth**: assignor

**Request body**:
```json
{
  "resolutions": [
    {
      "group_key": "dt:2026-04-25T11:30:West Gwinnett Park Aquatic Center",
      "action": "accept_row",
      "row_index": 10
    },
    {
      "group_key": "ref:30663414",
      "action": "accept_both"
    },
    {
      "group_key": "ref:30727899",
      "action": "skip"
    }
  ]
}
```

**Actions**:
- `accept_row` + `row_index`: import this specific row; skip others in the group. Valid for both Signal A (with `accept_both` as alternative) and Signal B.
- `accept_both`: import both rows as separate matches. Valid only for Signal A (same reference_id).
- `skip`: skip all rows in this group.

**Response** `200`:
```json
{
  "import_id": "f47ac10b-...",
  "unresolved_groups": 0,
  "ready_to_commit": true
}
```

**Error cases**:
- `404` — `import_id` not found or expired.
- `422` — `accept_both` attempted on a Signal B group.

---

### `POST /api/matches/import/{import_id}/commit`

Stage 2: Commits all accepted rows. All duplicate groups must be resolved before this endpoint allows a commit.

**Auth**: assignor

**Request body**: none

**Response** `200`:
```json
{
  "imported": 24,
  "skipped": 4,
  "errors": 2
}
```

**Error cases**:
- `404` — `import_id` not found or expired.
- `409` — unresolved duplicate groups remain.

---

### `DELETE /api/matches/import/{import_id}`

Discards a staged import.

**Auth**: assignor

**Response** `204`

---

## Availability

### `GET /api/availability`

Returns the authenticated referee's availability decisions for upcoming eligible matches.

**Auth**: referee

**Response** `200`:
```json
{
  "availability": [
    {
      "match_id": 101,
      "available": true,
      "marked_at": "2026-04-20T10:00:00Z"
    }
  ]
}
```

---

### `PUT /api/availability/{match_id}`

Sets the authenticated referee's availability for a match. Idempotent — safe to call with the current value.

**Auth**: referee

**Request body**:
```json
{ "available": true }
```

**Validation**:
- Match must exist, be scheduled (not cancelled), and be in the future.
- Referee must be active.
- Referee must be eligible for the match in at least one role (checked server-side).

**Response** `200`:
```json
{
  "match_id": 101,
  "available": true,
  "marked_at": "2026-04-20T10:05:00Z"
}
```

**Error cases**:
- `403` — match is in the past, cancelled, or referee is not eligible.
- `404` — match not found.

---

## Assignments

### `GET /api/matches/{match_id}/eligible-referees`

Returns referees eligible for a specific role slot, split into available/not-available sections.

**Auth**: assignor

**Query params**:
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `role_slot_id` | int | yes | The slot to evaluate eligibility for |

**Response** `200`:
```json
{
  "role_slot_id": 502,
  "role_type": "assistant",
  "available_and_eligible": [
    {
      "user_id": 3,
      "first_name": "Alice",
      "last_name": "Chen",
      "grade": "senior",
      "age_at_match": 16,
      "certified": true,
      "cert_expiry": "2027-06-30",
      "has_conflict": false
    }
  ],
  "eligible_not_available": [
    {
      "user_id": 4,
      "first_name": "Tom",
      "last_name": "Walsh",
      "grade": null,
      "age_at_match": 14,
      "certified": false,
      "cert_expiry": null,
      "has_conflict": true,
      "conflict_match": {
        "id": 99,
        "event_name": "U8B Eagles v Owls",
        "start_time": "08:30",
        "end_time": "10:00"
      }
    }
  ]
}
```

`has_conflict` is `true` if the referee is assigned to another match whose time window overlaps this match's `start_time`–`end_time`.

---

### `POST /api/assignments`

Assigns a referee to a role slot. If the slot is already assigned, this is a reassignment (the existing assignment is replaced and both audit entries are written).

**Auth**: assignor

**Request body**:
```json
{
  "match_id": 101,
  "role_slot_id": 502,
  "referee_id": 3,
  "override_conflict": false
}
```

`override_conflict`: if `false` (default) and the referee has a time conflict, the endpoint returns `409`. If `true`, the conflict is acknowledged and the assignment proceeds.

**Validation** (all server-side):
- Match is scheduled and in the future.
- Slot belongs to the match.
- Referee is active.
- Referee passes eligibility rules for this slot (age-group + role type).

**Response** `201`:
```json
{
  "id": 302,
  "match_id": 101,
  "role_slot_id": 502,
  "referee_id": 3,
  "assigned_by": 1,
  "assigned_at": "2026-04-21T09:00:00Z",
  "acknowledged": false
}
```

**Error cases**:
- `403` — referee is not eligible for this slot (returns reason).
- `409` — referee has a time conflict and `override_conflict` was not set to `true`.
- `409` — referee is already assigned to this slot (idempotent re-POST with same referee is a no-op — returns `200` not `201`).
- `404` — match, slot, or referee not found.

---

### `DELETE /api/assignments/{assignment_id}`

Removes an assignment. Audit log entry is written.

**Auth**: assignor

**Response** `204`

**Error cases**:
- `404` — assignment not found.
- `422` — match is in the past (cannot modify past assignments).

---

## Audit Log (Assignor)

### `GET /api/matches/{match_id}/audit-log`

Returns the assignment history for a match.

**Auth**: assignor

**Response** `200`:
```json
{
  "entries": [
    {
      "id": 1001,
      "role_slot_id": 502,
      "role_type": "assistant",
      "slot_number": 1,
      "action": "assign",
      "referee_added": { "user_id": 3, "display_name": "Alice Chen" },
      "referee_removed": null,
      "acting_assignor": { "user_id": 1, "display_name": "Admin User" },
      "timestamp": "2026-04-21T09:00:00Z"
    }
  ]
}
```

---

## Health

### `GET /health`

**Auth**: public

**Response** `200`:
```json
{ "status": "ok", "db": "ok" }
```

`db` is `"ok"` if a simple `SELECT 1` query succeeds; `"error"` otherwise.
