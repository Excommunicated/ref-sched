# ADR-005: CSV Import Strategy

## Status: Accepted

## Context

Match schedules are exported from Stack Team App as CSV files and uploaded to this application. The CSV format has specific characteristics that make a naive single-pass import risky:

1. **One CSV row per team entry, not per match**: A U10 match between Eagles and Owls will appear as two rows — one for each team. These rows have different `reference_id` values but share `date + start_time + location`. Importing both rows would create two database records for the same physical match.

2. **Stack Team App known bug**: A match that was missed in a previous export may be re-exported with an identical `reference_id` as a replacement. Two rows with the same `reference_id` should not both be imported.

3. **Age group parsing may fail**: The `team_name` field contains the age group (e.g., "Under 12 Girls - Falcons"), but edge cases exist. A row where age group cannot be determined cannot be imported without manual intervention.

4. **Atomicity requirement**: No match rows should be written to the database until the assignor has reviewed and resolved all issues. A partial import that writes some rows and fails on others would leave the database in an inconsistent state.

The real-world CSV observed (`events-2.csv`) confirms all of the above: multiple team-per-match row pairs exist (e.g., rows for "Under 10 Boys - Eagles" and "Under 10 Boys - Owls" share the same date, start_time, and location), and the data is otherwise clean.

## Decision

**A two-phase parse-preview-resolve-commit flow with atomic commit.**

### Phase 1: Parse and Preview (POST /api/matches/import)

The uploaded CSV is parsed entirely server-side. The backend:
1. Reads all rows and normalizes field values.
2. Attempts to extract `age_group` from `team_name` using the pattern `Under {N}`.
3. Detects duplicate groups using two signals:
   - **Signal A**: two rows share the same `reference_id`.
   - **Signal B**: two rows share the same `start_date + start_time + location` (different `reference_id`).
4. Checks existing `reference_id` values in the `matches` table (rows matching an existing record are surfaced as Signal A duplicates where the "existing record" is one side of the pair).
5. Stores all parsed rows in `import_staging` table with an opaque `import_token`.
6. Returns the full preview to the client: all rows with parse status, all duplicate groups, and the import token.

**No rows are written to `matches` at this point.**

### Phase 2: Resolve Duplicates (PATCH /api/matches/import/{import_id}/resolutions)

The assignor views the duplicate groups in the UI and makes a decision for each:
- **Signal A**: accept row 1, accept row 2, accept both (treating them as separate matches), or skip both.
- **Signal B**: accept row 1, accept row 2, or skip both. ("Accept both" is not offered — they represent one physical match.)

Resolutions are stored in the `import_staging` table. The client can call this endpoint multiple times (once per group or in batches) until all groups are resolved.

The endpoint returns `ready_to_commit: true` when all groups have a resolution.

### Phase 3: Commit (POST /api/matches/import/{import_id}/commit)

The backend:
1. Verifies all duplicate groups are resolved.
2. Begins a database transaction.
3. For each accepted staging row: inserts a `matches` record and configures `match_role_slots` based on age group.
4. For each skipped staging row: no action.
5. Deletes the staging rows.
6. Commits the transaction.
7. Returns an import summary: rows imported, rows skipped, rows with unresolved parse errors.

**All writes to `matches` happen in a single atomic transaction or not at all.**

### Staging table cleanup

Staging rows older than 24 hours are treated as abandoned and can be garbage-collected. A cleanup query runs at backend startup. This prevents the staging table from accumulating stale rows from abandoned import sessions.

## Consequences

**Positive:**
- Assignor always sees what will be imported before it happens.
- No partial imports — the database is always consistent.
- Duplicate detection handles both the known Stack Team App bug and the team-per-match row pattern.
- Staging in PostgreSQL avoids adding a second service (no Redis/in-memory store).
- The two-phase approach allows the assignor to walk away mid-resolution and return to the same import session (within 24 hours).

**Negative:**
- More complex than a single-shot import. Requires the staging table and the multi-step API.
- The import token must be tracked by the frontend across the three API calls.
- A staging row cleanup strategy must be implemented and tested.

## Alternatives Considered

**Single-pass import with auto-deduplication**: The backend automatically picks one row when a duplicate is detected (e.g., always use the first occurrence). Rejected because the Stack Team App bug scenario (same `reference_id`, different actual data) requires human judgment about which row is correct.

**Client-side CSV parsing**: Parse the CSV in the browser and send normalized JSON to the backend. Rejected because: (1) CSV parsing in the browser is JavaScript, adding frontend complexity; (2) the duplicate check against the existing database can only happen server-side; (3) keeping parsing server-side means a single, testable implementation.

**Streaming import (write each row as it's parsed)**: Write rows immediately, skip/flag duplicates inline. Rejected because of the atomicity requirement — if an error occurs mid-import, the database would have a partial import with no clean rollback.
