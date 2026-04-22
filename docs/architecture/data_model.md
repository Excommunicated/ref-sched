# Data Model: Referee Scheduling App

All tables use PostgreSQL 16. Conventions:
- Primary keys are `BIGINT GENERATED ALWAYS AS IDENTITY` unless noted.
- Timestamps are `TIMESTAMPTZ` stored in UTC.
- Soft deletes are modeled as status columns, not physical deletes.
- All foreign keys have explicit `ON DELETE` behavior noted.
- Table and column names use `snake_case`.

---

## Schema

### `users`

Stores every person who has authenticated via Google OAuth2. Role determines routing and access level.

```sql
CREATE TABLE users (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    google_sub      TEXT        NOT NULL,
    email           TEXT        NOT NULL,
    display_name    TEXT        NOT NULL,
    role            TEXT        NOT NULL DEFAULT 'pending_referee'
                    CHECK (role IN ('pending_referee', 'referee', 'assignor')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_users_google_sub ON users (google_sub);
CREATE UNIQUE INDEX uq_users_email      ON users (email);
```

**Notes:**
- `google_sub` is the stable Google user ID (`sub` claim in the OIDC ID token). Used for login matching across email changes.
- `role` transitions: `pending_referee` → `referee` (on assignor activation). `assignor` is seeded manually at setup; no UI path to grant this role.
- `email` is stored for display purposes and assignor search; not used as the authoritative identity key (that is `google_sub`).

---

### `sessions`

Server-side session store. The session cookie contains only the `token` (opaque random string). No session data lives in the browser.

```sql
CREATE TABLE sessions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    token       TEXT        NOT NULL,
    user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_sessions_token  ON sessions (token);
CREATE        INDEX ix_sessions_user   ON sessions (user_id);
CREATE        INDEX ix_sessions_expiry ON sessions (expires_at);
```

**Notes:**
- Session lifetime: 30 days, sliding. `last_seen` is updated on each authenticated request; `expires_at` is extended when the session is used.
- Expired sessions are deleted lazily (checked on lookup) or by a periodic cleanup query run at startup.
- On logout: the session row is deleted.
- On user deactivation/removal: `ON DELETE CASCADE` ensures sessions are purged immediately when the user row is deleted. For the soft-delete (removed) case, the application explicitly deletes all sessions for that user_id before setting status to removed.

---

### `referee_profiles`

One-to-one with `users`. Created when the user first submits their profile form. A user record can exist without a profile row (pending referee who has not yet completed their profile).

```sql
CREATE TABLE referee_profiles (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    first_name      TEXT        NOT NULL,
    last_name       TEXT        NOT NULL,
    dob             DATE        NOT NULL,
    certified       BOOLEAN     NOT NULL DEFAULT FALSE,
    cert_expiry     DATE,           -- NULL when certified = FALSE
    grade           TEXT
                    CHECK (grade IN ('junior', 'mid', 'senior') OR grade IS NULL),
    status          TEXT        NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'active', 'inactive', 'removed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_referee_profiles_user ON referee_profiles (user_id);
CREATE        INDEX ix_referee_profiles_status ON referee_profiles (status);

-- Enforce: if certified = TRUE, cert_expiry must be present
ALTER TABLE referee_profiles
    ADD CONSTRAINT ck_cert_expiry_required
    CHECK (
        (certified = FALSE) OR
        (certified = TRUE AND cert_expiry IS NOT NULL)
    );
```

**Notes:**
- `grade` is set exclusively by an assignor. The application enforces this at the API layer (the `PATCH /api/referees/{id}` endpoint that updates grade requires the `assignor` role).
- `status` drives the referee lifecycle. `pending` maps to the `pending_referee` user role. When an assignor activates a pending referee, both `referee_profiles.status` → `active` and `users.role` → `referee` are updated in the same transaction.
- `removed` is a soft delete. The user cannot log in (checked at session creation), profile data is retained for historical record.
- `dob` is a `DATE` (no time component). Age at match date is computed as: `DATE_PART('year', AGE(match.start_date, referee_profiles.dob))`.

---

### `matches`

One row per unique physical match. Duplicate resolution during CSV import ensures this table never contains two rows representing the same physical game.

```sql
CREATE TABLE matches (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reference_id    TEXT,           -- Stack Team App reference ID; NULL for manually created matches
    event_name      TEXT        NOT NULL,
    age_group       SMALLINT    NOT NULL,   -- numeric: 6, 8, 10, 12, 14, 16, 19
    gender          TEXT        NOT NULL
                    CHECK (gender IN ('boys', 'girls', 'coed', 'unknown')),
    start_date      DATE        NOT NULL,
    start_time      TIME        NOT NULL,
    end_time        TIME        NOT NULL,
    location        TEXT        NOT NULL,   -- venue address / name
    venue_detail    TEXT,                   -- specific field parsed from description (e.g. "Field 3")
    description     TEXT,                  -- raw description from CSV (meeting time, field, etc.)
    status          TEXT        NOT NULL DEFAULT 'scheduled'
                    CHECK (status IN ('scheduled', 'cancelled')),
    created_by      BIGINT      NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      BIGINT          REFERENCES users (id) ON DELETE RESTRICT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_matches_reference_id
    ON matches (reference_id)
    WHERE reference_id IS NOT NULL;

-- Supports duplicate detection signal B (same date+time+location)
CREATE INDEX ix_matches_date_time_location
    ON matches (start_date, start_time, location);

-- Common query: upcoming matches by date
CREATE INDEX ix_matches_start_date ON matches (start_date);

-- Common query: filter by age group
CREATE INDEX ix_matches_age_group ON matches (age_group);
```

**Notes:**
- `age_group` is stored as a SMALLINT (6, 8, 10, 12, 14, 16, 19) so eligibility arithmetic is straightforward in SQL: `referee_age >= age_group + 1`.
- `reference_id` has a partial unique index (WHERE NOT NULL) to allow multiple manually-created matches with no reference_id.
- `venue_detail` is parsed from the `description` field during CSV import (e.g., "Field 3" or "Valor Academy\nField D"). If parsing fails, it is left NULL and the full `description` is retained.
- `updated_by` / `updated_at` satisfy the Story 3.4 audit requirement for match edits without a full event log.

---

### `match_role_slots`

Defines the assignable slots on a match. Each row is one fillable position.

```sql
CREATE TABLE match_role_slots (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    match_id        BIGINT      NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    role_type       TEXT        NOT NULL
                    CHECK (role_type IN ('center', 'assistant')),
    slot_number     SMALLINT    NOT NULL,   -- 1 for center; 1 or 2 for assistant
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_match_role_slots_match_role_slot
    ON match_role_slots (match_id, role_type, slot_number);

CREATE INDEX ix_match_role_slots_match ON match_role_slots (match_id);
```

**Slot configuration by age group (applied on import and match creation):**

| Age Group | Center Slots | Assistant Slots |
|-----------|-------------|-----------------|
| U6, U8    | 1           | 0               |
| U10       | 1           | 0 (0–2 max, assignor-configurable) |
| U12+      | 1           | 2               |

**Notes:**
- U10 assistant slots are created on demand by the assignor (up to 2). The application enforces the maximum at the API layer.
- When the assignor adds/removes assistant slots on a U10 match, only the slots with no current assignment can be removed. The API enforces this.
- If an assignor changes a match's `age_group`, the backend recalculates required slots, removes any now-excess slots (if unassigned), and warns about ineligible existing assignments.

---

### `referee_availability`

Records which referees have opted in for which matches.

```sql
CREATE TABLE referee_availability (
    referee_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    match_id        BIGINT      NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    available       BOOLEAN     NOT NULL DEFAULT TRUE,
    marked_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (referee_id, match_id)
);

CREATE INDEX ix_availability_match ON referee_availability (match_id);
CREATE INDEX ix_availability_referee ON referee_availability (referee_id);
```

**Notes:**
- Availability is opt-in. A missing row means not available (same as `available = FALSE`). The `available` column exists so a toggle-off can be recorded without deleting the row; this simplifies the UPSERT logic.
- The backend checks that the referee is eligible for the match before allowing an availability mark. Ineligible marks are rejected with `403`.
- Availability records for past matches are retained (not deleted) for historical reference.

---

### `assignments`

The authoritative record of which referee is assigned to which role slot.

```sql
CREATE TABLE assignments (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    match_id        BIGINT      NOT NULL REFERENCES matches (id) ON DELETE RESTRICT,
    role_slot_id    BIGINT      NOT NULL REFERENCES match_role_slots (id) ON DELETE RESTRICT,
    referee_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    assigned_by     BIGINT      NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged    BOOLEAN     NOT NULL DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,

    CONSTRAINT ck_acknowledged_at
        CHECK (
            (acknowledged = FALSE AND acknowledged_at IS NULL) OR
            (acknowledged = TRUE  AND acknowledged_at IS NOT NULL)
        )
);

-- One referee per slot at any time
CREATE UNIQUE INDEX uq_assignments_slot ON assignments (role_slot_id);

-- Reassignment query
CREATE INDEX ix_assignments_match   ON assignments (match_id);
CREATE INDEX ix_assignments_referee ON assignments (referee_id);
```

**Notes:**
- One slot can have at most one assigned referee (`UNIQUE` on `role_slot_id`).
- Reassignment: the existing row is updated (new `referee_id`, `assigned_by`, `assigned_at`). The previous assignment is preserved in `assignment_audit_log`.
- `ON DELETE RESTRICT` on `match_id` and `referee_id` prevents accidental deletion of matches or users when assignments exist. Removal must go through the application (clear assignments first, or cascade explicitly).
- `acknowledged` and `acknowledged_at` support the stretch goal (Story 6.2). These columns are included in v1 schema but the feature may not be implemented.

---

### `assignment_audit_log`

Immutable append-only log of every assignment action.

```sql
CREATE TABLE assignment_audit_log (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    match_id            BIGINT      NOT NULL REFERENCES matches (id) ON DELETE RESTRICT,
    role_slot_id        BIGINT      NOT NULL REFERENCES match_role_slots (id) ON DELETE RESTRICT,
    action              TEXT        NOT NULL
                        CHECK (action IN ('assign', 'reassign', 'remove')),
    referee_added_id    BIGINT          REFERENCES users (id) ON DELETE RESTRICT,   -- NULL on 'remove'
    referee_removed_id  BIGINT          REFERENCES users (id) ON DELETE RESTRICT,   -- NULL on 'assign'
    acting_assignor_id  BIGINT      NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_audit_log_match      ON assignment_audit_log (match_id);
CREATE INDEX ix_audit_log_referee    ON assignment_audit_log (referee_added_id)   WHERE referee_added_id IS NOT NULL;
CREATE INDEX ix_audit_log_removed    ON assignment_audit_log (referee_removed_id) WHERE referee_removed_id IS NOT NULL;
CREATE INDEX ix_audit_log_timestamp  ON assignment_audit_log (timestamp);
```

**Action semantics:**

| Action | `referee_added_id` | `referee_removed_id` |
|--------|--------------------|----------------------|
| `assign` | new referee | NULL |
| `reassign` | new referee | previous referee |
| `remove` | NULL | removed referee |

**Notes:**
- This table is never updated, only inserted into.
- Both the assignment write and the audit log insert happen in a single database transaction.
- Foreign key references use `ON DELETE RESTRICT`. Historical audit records must be retained even if a user is "removed" from the system. The soft-delete model (users are never physically deleted) ensures these FK references are always valid.

---

### `import_staging`

Temporary staging table for the two-phase CSV import. Rows are written during parse (step 1) and cleaned up after commit (step 2).

```sql
CREATE TABLE import_staging (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    import_token    TEXT        NOT NULL,   -- random UUID, returned to client as import_id
    row_index       INT         NOT NULL,
    raw_data        JSONB       NOT NULL,   -- original CSV row key-value pairs
    parsed_data     JSONB,                  -- parsed/normalized fields (NULL if parse failed)
    age_group       SMALLINT,
    parse_error     TEXT,                   -- description of parse failure, if any
    duplicate_group TEXT,                   -- NULL or group key (e.g. "ref:ABC123" or "dt:2026-04-25T09:00:Jones Bridge Park")
    duplicate_signal TEXT                   -- 'reference_id' | 'date_time_location' | NULL
                    CHECK (duplicate_signal IN ('reference_id', 'date_time_location') OR duplicate_signal IS NULL),
    resolution      TEXT                    -- 'accept' | 'skip' | NULL (unresolved)
                    CHECK (resolution IN ('accept', 'skip') OR resolution IS NULL),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX ix_import_staging_token     ON import_staging (import_token);
CREATE INDEX ix_import_staging_created   ON import_staging (created_at);
```

**Notes:**
- `import_token` is generated server-side (UUID) and returned to the client as `import_id`. Used to correlate the resolve and commit calls to the correct staged import.
- Staging rows older than 24 hours are garbage-collected. A cleanup runs at backend startup and can also be triggered manually.
- The same assignor cannot have two in-flight imports simultaneously (enforced by checking for an existing unresolved `import_token` for the same `created_by` user).

---

## Key Query Patterns

### Eligibility check — who is eligible for a given role slot?

```sql
-- Parameters: :match_id, :role_slot_id
-- Returns referee user IDs that are eligible

WITH match_info AS (
    SELECT m.start_date, m.age_group, mrs.role_type
    FROM   match_role_slots mrs
    JOIN   matches m ON m.id = mrs.match_id
    WHERE  mrs.id = :role_slot_id
),
eligible AS (
    SELECT rp.user_id
    FROM   referee_profiles rp
    CROSS JOIN match_info mi
    WHERE  rp.status = 'active'
    AND (
        -- U10 and younger: any role — must be age_group + 1 years old on match date
        (mi.age_group <= 10
            AND DATE_PART('year', AGE(mi.start_date, rp.dob)) >= mi.age_group + 1
        )
        OR
        -- U12+ center: must have non-expired certification
        (mi.age_group >= 12 AND mi.role_type = 'center'
            AND rp.certified = TRUE
            AND rp.cert_expiry >= mi.start_date
        )
        OR
        -- U12+ assistant: no restrictions (only needs active status)
        (mi.age_group >= 12 AND mi.role_type = 'assistant')
    )
)
SELECT e.user_id,
       rp.first_name,
       rp.last_name,
       rp.grade,
       rp.certified,
       rp.cert_expiry,
       DATE_PART('year', AGE(mi.start_date, rp.dob)) AS age_at_match,
       EXISTS (
           SELECT 1 FROM referee_availability ra
           WHERE ra.referee_id = e.user_id
             AND ra.match_id = :match_id
             AND ra.available = TRUE
       ) AS has_marked_available
FROM   eligible e
JOIN   referee_profiles rp ON rp.user_id = e.user_id
CROSS JOIN match_info mi
ORDER  BY has_marked_available DESC, rp.last_name, rp.first_name;
```

### Schedule view — matches with assignment completion status

```sql
SELECT
    m.id,
    m.event_name,
    m.age_group,
    m.gender,
    m.start_date,
    m.start_time,
    m.end_time,
    m.location,
    m.venue_detail,
    m.status,
    COUNT(mrs.id)                                          AS total_slots,
    COUNT(a.id)                                            AS filled_slots,
    CASE
        WHEN COUNT(a.id) = 0                              THEN 'unassigned'
        WHEN COUNT(a.id) = COUNT(mrs.id)                  THEN 'full'
        ELSE                                                   'partial'
    END                                                    AS assignment_status
FROM   matches m
LEFT JOIN match_role_slots mrs ON mrs.match_id = m.id
LEFT JOIN assignments a        ON a.role_slot_id = mrs.id
WHERE  m.start_date >= CURRENT_DATE   -- upcoming only; remove for full history
GROUP BY m.id
ORDER BY m.start_date, m.start_time;
```

### Referee availability view — matches a referee is eligible for

```sql
-- Parameters: :referee_id
SELECT
    m.id,
    m.event_name,
    m.age_group,
    m.gender,
    m.start_date,
    m.start_time,
    m.location,
    m.venue_detail,
    m.description,
    COALESCE(ra.available, FALSE) AS is_available,
    a.id IS NOT NULL              AS is_assigned
FROM   matches m
JOIN   referee_profiles rp ON rp.user_id = :referee_id
LEFT JOIN referee_availability ra
    ON ra.match_id = m.id AND ra.referee_id = :referee_id
LEFT JOIN match_role_slots mrs ON mrs.match_id = m.id
LEFT JOIN assignments a
    ON a.role_slot_id = mrs.id AND a.referee_id = :referee_id
WHERE  m.status = 'scheduled'
AND    m.start_date >= CURRENT_DATE
AND    rp.status = 'active'
-- Eligibility filter (same logic as above, applied per match)
AND (
    (m.age_group <= 10
        AND DATE_PART('year', AGE(m.start_date, rp.dob)) >= m.age_group + 1
    )
    OR
    (m.age_group >= 12
        AND (
            -- eligible as center
            (rp.certified = TRUE AND rp.cert_expiry >= m.start_date)
            OR
            -- eligible as assistant (no constraints)
            EXISTS (
                SELECT 1 FROM match_role_slots s
                WHERE s.match_id = m.id AND s.role_type = 'assistant'
            )
        )
    )
)
GROUP BY m.id, ra.available, a.id, rp.dob, rp.status
ORDER BY m.start_date, m.start_time;
```

---

## Migration Strategy

- Tool: `golang-migrate` (`github.com/golang-migrate/migrate/v4`).
- Migration files: `db/migrations/NNNN_description.up.sql` and `db/migrations/NNNN_description.down.sql`.
- Migrations run automatically at backend startup before the HTTP server starts accepting traffic.
- Initial migration file: `0001_initial_schema.up.sql` — creates all tables above.
- Subsequent migrations (one file per schema change) are numbered sequentially.
- Migration state is tracked in the `schema_migrations` table (managed by golang-migrate).
