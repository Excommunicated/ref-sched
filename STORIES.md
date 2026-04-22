# Engineering Decomposition: Referee Scheduling App

## Objective

Replace a manual email-based referee scheduling workflow with a responsive web application. Assignors import match schedules, manage a verified referee pool, and assign qualified referees to role slots per match. Referees self-manage profiles and mark availability. Target: reduce the end-to-end scheduling cycle from ~2 days to ≤ 4 hours, before the August season start.

---

## Scope

**In scope (v1)**
- Google OAuth2 authentication (no passwords)
- Referee profiles: date of birth, certifications with expiry
- Assignor-controlled referee verification (pending → active)
- CSV import of match schedule (Stack Team App format)
- Automatic age group parsing and role slot configuration
- Duplicate match detection on import
- Manual match editing (reschedule, cancel)
- Eligibility engine: age-based (U10 and younger) and certification-based (U12 and older)
- Referee availability marking (per eligible match)
- Assignor assignment interface: filtered eligible/available referee lists per role slot
- Referee assignment view with full location and meeting time
- Assignment acknowledgment (stretch goal — may defer to v2)
- Responsive web app (mobile-first, 320px+)
- Docker deployment to Azure

**Out of scope (v1)**
- Auto-assignment
- Email/push notifications
- Payments or expense tracking
- League/team/player management
- Stack Team App API integration
- Native iOS/Android apps
- Non-Google OAuth providers

---

## Assumptions

1. All users have or can create a Google account.
2. The Stack Team App CSV format (`team_name`, `start_date`, `start_time`, `end_time`, `location`, `description`, `event_name`, `reference_id`) is stable within a season.
3. Age group is reliably extractable from `team_name` using the pattern `"Under {N} {Gender} [- Team]"`.
4. Only home matches are uploaded; the assignor filters before exporting from Stack Team App.
5. ~22 total users (1–2 assignors, ~20 referees); no horizontal scaling required for v1.
6. "Certified" is binary (certified / not certified) with a single expiry date. There are no named certification grade levels. A separate assignor-managed referee grade (Junior / Mid / Senior) exists for advisory assignment purposes only.
7. Concurrent assignors (if 2 active) can both assign without explicit locking; last-write-wins is acceptable at this scale.
8. The assignor role is seeded manually at initial setup; there is no self-service assignor registration.

---

## Functional Requirements (summary)

| ID | Requirement |
|----|-------------|
| FR-01 | CSV import of match schedule |
| FR-02 | Age group parsed from `team_name`; role slots and eligibility rules applied automatically |
| FR-03 | Duplicate match detection on import |
| FR-04 | Manual match edit / cancel / reschedule |
| FR-05 | Referee profile: DOB, certification + expiry |
| FR-06 | Age and certification computed at match date; expired certs flagged |
| FR-07 | Referee views and marks availability on eligible matches only |
| FR-08 | Eligibility rules enforced (age-based U10−, cert-based U12+) |
| FR-09 | Role slot counts per age group (U6/U8: 1CR; U10: 1CR+0–2AR; U12+: 1CR+2AR) |
| FR-10 | Assignment interface filtered by eligible + available referees |
| FR-11 | Assignor assigns / reassigns / removes referee from role slot |
| FR-12 | Referee views assignments with full location and meeting time |
| FR-13 | Assignment acknowledgment (stretch) |
| FR-14 | Assignor schedule view with completion status per match |
| FR-15 | New Google accounts land in pending state; profile editable, matches not visible |
| FR-16 | Assignor referee management: activate / deactivate / remove |

## Non-Functional Requirements (summary)

| Category | Requirement | Priority |
|----------|-------------|----------|
| Responsiveness | Fully functional at 320px+ | High |
| Performance | < 2s page load on mobile | Medium |
| Security | No passwords; OAuth2 tokens secured; PII (name, DOB) only | High |
| Auditability | Assignment history retained for ≥ 1 season | Medium |
| Maintainability | Simple stack; single developer | High |

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| CSV format changes between seasons | Medium | Medium | Configurable column mapping in import step |
| Age group parse fails on edge-case team names | Medium | High | Validate early against real exports; manual override on import |
| Frontend framework unfamiliar to developer | High | Medium | Allocate spike time; choose well-documented framework |
| August deadline pressure causes scope creep | Medium | High | Acknowledgment story is explicit stretch; cut it first |
| Single-developer bus factor | High | High | Keep architecture simple; avoid over-engineering |

---

## Epics & Stories

---

### Epic 1 — Foundation & Authentication

Establishes the project skeleton, database, and Google OAuth2 login with role-aware routing.

---

#### Story 1.1 — Project skeleton

> As a developer, I want a working backend, database, and frontend scaffold so that all subsequent features have a consistent foundation to build on.

**Acceptance Criteria**
- [ ] Go backend serves `GET /health` returning `200 OK`.
- [ ] PostgreSQL database is connected; connection failure prevents startup with a clear error.
- [ ] Frontend renders a home/login page at `/`.
- [ ] `docker-compose.yml` runs backend + frontend + database locally.
- [ ] Database migration tooling is in place (e.g., `golang-migrate`); initial migration creates a `schema_migrations` table.

**Dependencies:** None  
**Size:** M

---

#### Story 1.2 — Google OAuth2 login

> As a user, I want to sign in with my Google account so that I don't need a separate password.

**Acceptance Criteria**
- [ ] "Sign in with Google" button on the home page initiates the OAuth2 flow.
- [ ] After successful Google consent, the user is redirected back to the app with a valid session.
- [ ] User's Google `sub`, email, and display name are stored in the `users` table on first login.
- [ ] Subsequent logins with the same Google account reuse the existing user record.
- [ ] Session is persisted via a secure, HTTP-only cookie.
- [ ] Signing out clears the session and redirects to the home page.
- [ ] All non-public routes return `401` / redirect to login if no valid session exists.

**Dependencies:** 1.1  
**Size:** M

---

#### Story 1.3 — Role-based routing

> As the system, I want to route users to the correct interface based on their role so that assignors and referees see only what is relevant to them.

**Acceptance Criteria**
- [ ] Users have a `role` field: `pending_referee`, `referee`, `assignor`.
- [ ] New users are created with role `pending_referee`.
- [ ] Assignor role is set via a database seed or CLI command (no UI in v1).
- [ ] Authenticated referees (role `referee`) are routed to the referee interface.
- [ ] Authenticated assignors are routed to the assignor interface.
- [ ] Pending referees are routed to a "pending activation" screen that links to profile completion.
- [ ] Role is checked server-side on every protected route; client-side role hints are not trusted.

**Dependencies:** 1.2  
**Size:** S

---

### Epic 2 — Referee Profiles & Verification

Covers referee profile creation and the assignor's ability to manage the verified pool.

---

#### Story 2.1 — Referee profile management

> As a referee, I want to enter and update my personal details and certification so that the system can determine my eligibility for matches.

**Acceptance Criteria**
- [ ] Profile form fields: first name, last name, date of birth, certified (yes/no toggle), certification expiry date (required if certified).
- [ ] DOB must be in the past; form rejects future dates.
- [ ] Certification expiry must be after today if certified is selected.
- [ ] Saving updates the profile in place (no duplicate records).
- [ ] Profile is accessible to pending referees (they can complete it while awaiting activation).
- [ ] Profile page is mobile-responsive.

**Dependencies:** 1.3  
**Size:** S

---

#### Story 2.2 — Assignor referee management view

> As an assignor, I want to see all registered referees with their status so that I can manage who is in the active pool.

**Acceptance Criteria**
- [ ] Page lists all referees with columns: name, email, date of birth, certification status (certified / expired / none), verification status (pending / active / inactive).
- [ ] Pending referees are shown at the top or highlighted distinctly.
- [ ] Assignor can filter the list by verification status.
- [ ] Assignor can search by name.
- [ ] Page is accessible only to assignors; referees receive `403`.

**Dependencies:** 2.1  
**Size:** S

---

#### Story 2.3 — Referee activation, deactivation, and grading

> As an assignor, I want to activate, deactivate, or remove a referee from the pool, and assign them a grade, so that only vetted referees appear in the assignment interface and I can make informed assignment decisions.

**Acceptance Criteria**
- [ ] Assignor can set a referee's status to `active`, `inactive`, or `removed` from the management view.
- [ ] Activating a `pending_referee` changes their role to `referee` and status to `active`.
- [ ] Inactive referees: can still log in and view their own profile/assignments, but do not appear in the assignment eligible list and cannot mark availability on new matches.
- [ ] Removed referees are soft-deleted: login is blocked, data is retained, status shows `removed` in management view.
- [ ] Status changes take effect immediately (no page reload required).
- [ ] Assignor can set a referee's grade to **Junior**, **Mid**, or **Senior** from the management view or inline on the referee row.
- [ ] Grade defaults to unset (blank) for newly activated referees; the assignor is not forced to set it.
- [ ] Referees cannot view or edit their own grade.
- [ ] Grade is shown alongside the referee's name in the assignment picker (Story 5.2) to assist the assignor.

**Dependencies:** 2.2  
**Size:** S

---

#### Story 2.4 — Certification expiry flagging

> As an assignor, I want expired or soon-expiring certifications flagged so that I don't assign an ineligible center referee to a U12+ match.

**Acceptance Criteria**
- [ ] In the referee management view, referees with certifications expiring within 30 days are shown with a warning indicator.
- [ ] Referees with expired certifications are shown with an error indicator.
- [ ] In the assignment interface (Epic 5), referees with expired certifications are excluded from center role eligibility on U12+ matches.
- [ ] The referee's own profile shows a warning if their certification is expired or expiring within 30 days.

**Dependencies:** 2.2  
**Size:** S

---

### Epic 3 — Match Schedule Management

Covers importing the season schedule from CSV and keeping it up to date manually.

---

#### Story 3.1 — CSV import

> As an assignor, I want to upload a Stack Team App CSV export so that the match schedule is populated without manual data entry.

**Acceptance Criteria**
- [ ] File picker accepts `.csv` files only; other file types are rejected with a clear message.
- [ ] Parser reads: `event_name`, `team_name`, `start_date`, `end_date`, `start_time`, `end_time`, `description`, `location`, `reference_id`.
- [ ] Age group is extracted from `team_name` using the pattern `"Under {N}"` (e.g., `"Under 12 Girls - Falcons"` → `U12`). Rows where extraction fails are listed with an "unrecognised age group" error and excluded from import unless the assignor manually sets the age group.
- [ ] An import preview table shows all parsed rows (success + errors) before the assignor confirms.
- [ ] Confirming import writes valid rows to the `matches` table; rows with unresolved errors are skipped.
- [ ] Import summary shows: X matches imported, Y skipped, Z errors.
- [ ] If a row's `reference_id` already exists in the database, it is flagged as a potential duplicate (not silently updated) and routed through the duplicate resolution step in Story 3.2 before any write occurs.

**Dependencies:** 1.3  
**Size:** L

---

#### Story 3.2 — Duplicate match detection and resolution

> As an assignor, I want to be shown and prompted to resolve any potential duplicate matches during import so that I don't create the same match twice and can catch the Stack Team App export bug where a missing match is replaced by a duplicate entry with the same reference ID.

**Acceptance Criteria**
- [ ] Two duplicate signals are checked during import preview, before any rows are written:
  - **Signal A — same `reference_id`**: two rows share an identical `reference_id`. This is the Stack Team App known bug where a missed match is replaced by a duplicate export entry. The assignor is shown both rows side-by-side with their full details and must choose: import row 1, import row 2, import both (treating them as separate matches), or skip both.
  - **Signal B — same date + start time + location (different `reference_id`)**: two rows represent the same physical match under different team entries. The assignor is shown both rows and must choose: import row 1, import row 2, or skip both. "Import both" is not offered for Signal B as they represent one physical match.
- [ ] All flagged pairs must be resolved by the assignor before the import confirmation button is enabled.
- [ ] Rows that are not part of any flagged pair proceed to import normally.
- [ ] If a row's `reference_id` matches an already-imported match in the database, it is also surfaced as a Signal A duplicate (the existing record is shown as "row 1") so the assignor can decide whether to overwrite or skip.

**Dependencies:** 3.1  
**Size:** M

---

#### Story 3.3 — Role slots applied on import

> As an assignor, I want role slots configured automatically based on age group so that each match is ready for assignment without further setup.

**Acceptance Criteria**
- [ ] U6 / U8 matches receive 1 center slot, 0 assistant slots.
- [ ] U10 matches receive 1 center slot, 0 assistant slots by default (assignor may add up to 2 assistant slots manually post-import).
- [ ] U12 / U14+ matches receive 1 center slot, 2 assistant slots.
- [ ] Role slot configuration is visible and editable per match by the assignor after import.
- [ ] Adding or removing assistant slots on a U10 match does not affect other matches.

**Dependencies:** 3.1  
**Size:** S

---

#### Story 3.4 — Manual match management

> As an assignor, I want to edit, cancel, or reschedule a match after import so that the schedule stays accurate when things change.

**Acceptance Criteria**
- [ ] Assignor can edit: event name, date, start time, end time, location, description, age group.
- [ ] Changing age group reconfigures role slots and re-evaluates eligibility for any existing assignments (assignor is warned if existing assignments become ineligible).
- [ ] Assignor can mark a match as `cancelled`; cancelled matches remain visible with a "Cancelled" badge and are excluded from referee availability and assignment flows.
- [ ] Assignor can un-cancel a match.
- [ ] Cancelled matches retain their existing assignment data (for record-keeping).
- [ ] All edits are logged with a timestamp and the acting assignor's identity.

**Dependencies:** 3.3  
**Size:** M

---

#### Story 3.5 — Assignor schedule view

> As an assignor, I want to see all upcoming matches with their assignment completion status so that I can identify what still needs referees.

**Acceptance Criteria**
- [ ] Matches are listed sorted by date then start time.
- [ ] Each row shows: date, start time, age group, event name, location, assignment status badge (Unassigned / Partial / Full).
- [ ] Assignor can filter by: date range, age group, assignment status.
- [ ] Cancelled matches are visually distinct and can be shown/hidden via a toggle.
- [ ] Clicking a match opens the assignment panel (Epic 5).
- [ ] View is responsive and usable on mobile.

**Dependencies:** 3.3  
**Size:** M

---

### Epic 4 — Eligibility Engine & Referee Availability

Core business logic: who is eligible for what, and who has said they're available.

---

#### Story 4.1 — Eligibility engine

> As the system, I want to evaluate each referee's eligibility for each match role so that invalid assignments are impossible and the assignment interface is pre-filtered.

**Acceptance Criteria**
- [ ] For **center referee on U12+ matches**: referee must have a non-expired certification on the match date. No minimum age requirement.
- [ ] For **assistant referee on U12+ matches**: no certification required, no minimum age requirement.
- [ ] For **any role on U10 and younger**: referee must be at least `age_group + 1` years old on the match date (e.g., U10 → must be ≥ 11; U8 → ≥ 9; U6 → ≥ 7). No certification required.
- [ ] Eligibility is computed on-the-fly at query time using the referee's stored DOB and certification expiry vs. the match date; it is not cached.
- [ ] A referee's eligibility is re-evaluated automatically when their profile changes.
- [ ] Eligibility results are queryable: "which referees are eligible for match X in role Y?"

**Dependencies:** 2.1, 3.3  
**Size:** M

---

#### Story 4.2 — Referee availability marking

> As a referee, I want to view upcoming matches I'm eligible for and mark which ones I want to be considered for so that the assignor knows my preferences.

**Acceptance Criteria**
- [ ] Referee sees only: upcoming (not past), non-cancelled matches for which they are eligible in at least one role.
- [ ] Each match card shows: event name, age group, date, start time, venue, specific field (from description).
- [ ] Referee can toggle "Available" / "Not available" per match. Default state is "Not available" (opt-in model).
- [ ] Toggle saves immediately (no submit button); a subtle confirmation (e.g., colour change) confirms the save.
- [ ] Matches the referee is already assigned to are shown separately under "My Assignments" and are not toggleable.
- [ ] View is optimised for mobile.

**Dependencies:** 4.1  
**Size:** M

---

#### Story 4.3 — Referee upcoming matches view

> As a referee, I want to see all my eligible upcoming matches grouped by date so that I can plan ahead.

**Acceptance Criteria**
- [ ] Matches are grouped by date in ascending order.
- [ ] Past matches are not shown.
- [ ] Each match shows: event name, age group, date, start time, meeting time (parsed from description if present), venue name, specific field, current availability status.
- [ ] The referee can change their availability status directly from this view.

**Dependencies:** 4.2  
**Size:** S

---

### Epic 5 — Assignment Interface

The assignor's core workflow: assigning qualified, available referees to role slots.

---

#### Story 5.1 — Match assignment panel

> As an assignor, I want to open any match and see its role slots, current assignments, and the number of available referees so that I have a clear picture of what needs attention.

**Acceptance Criteria**
- [ ] Accessible from the schedule view (Story 3.5) by clicking/tapping a match.
- [ ] Panel displays: event name, age group, date, start time, venue, specific field, and all role slots.
- [ ] Each role slot shows: role name (Center Referee / Assistant Referee 1 / Assistant Referee 2), assigned referee name or "Unassigned".
- [ ] A summary line shows how many referees have marked availability for this match.
- [ ] Panel is usable on mobile (side panel on desktop, full screen on mobile).

**Dependencies:** 3.5, 4.2  
**Size:** M

---

#### Story 5.2 — Eligible and available referee list per role

> As an assignor, I want to see a pre-filtered list of referees who are eligible and available for a specific role so that I can assign quickly.

**Acceptance Criteria**
- [ ] Tapping an unassigned role slot reveals a referee picker.
- [ ] Picker shows two sections:
  - **Available & eligible**: referees who marked availability and pass eligibility rules for this role.
  - **Eligible but not available**: referees who are eligible but did not mark availability (assignor can still assign with one extra tap).
- [ ] Each referee row shows: name, grade (Junior / Mid / Senior, or blank if unset), computed age on match date, certification status (if relevant to the role).
- [ ] Referees already assigned to another match with overlapping time are shown with a conflict indicator in both sections.
- [ ] Inactive or removed referees do not appear.

**Dependencies:** 5.1, 4.1  
**Size:** M

---

#### Story 5.3 — Assign, reassign, and remove

> As an assignor, I want to assign a referee to a role, replace them, or remove the assignment so that I have full control over the schedule.

**Acceptance Criteria**
- [ ] Tapping a referee in the picker assigns them to the selected role slot. The slot updates immediately.
- [ ] Tapping an already-assigned slot re-opens the picker (reassign flow) or shows a "Remove" option.
- [ ] Removing an assignment clears the slot to "Unassigned" immediately.
- [ ] All assignment changes are recorded: match, role, referee assigned, referee removed (if replacing), acting assignor, timestamp.
- [ ] The match's assignment status badge on the schedule view updates immediately after a change.

**Dependencies:** 5.2  
**Size:** M

---

#### Story 5.4 — Double-booking conflict warning

> As an assignor, I want a warning when I'm about to double-book a referee so that I don't accidentally schedule them for two matches at the same time.

**Acceptance Criteria**
- [ ] Before completing an assignment, the system checks whether the referee is already assigned to another match whose time window overlaps (start_time to end_time).
- [ ] If a conflict exists, a confirmation dialog is shown: "This referee is already assigned to [Match Name] at [Time]. Assign anyway?"
- [ ] The assignor can confirm or cancel. Confirming proceeds with the assignment.
- [ ] The conflict indicator in the picker (Story 5.2) appears before the assignor even taps the referee.

**Dependencies:** 5.3  
**Size:** S

---

### Epic 6 — Referee Assignment View & Acknowledgment *(Stretch)*

Referees see their confirmed assignments; optionally acknowledge them.

---

#### Story 6.1 — My assignments view

> As a referee, I want to see all matches I've been assigned to with full details so that I know exactly where and when to show up.

**Acceptance Criteria**
- [ ] Referee sees a list of their confirmed assignments sorted by date.
- [ ] Each assignment shows: event name, age group, date, start time, meeting time (from description), full venue address, specific field, role (Center / Assistant Referee).
- [ ] Past assignments are shown in a collapsed or secondary section.
- [ ] View is mobile-first; all details are readable without horizontal scrolling.

**Dependencies:** 5.3  
**Size:** S

---

#### Story 6.2 — Assignment acknowledgment *(Stretch — may defer to v2)*

> As a referee, I want to acknowledge my assignment in-app so that the assignor knows I've seen and accepted it.

**Acceptance Criteria**
- [ ] Each unacknowledged assignment shows an "Acknowledge" button.
- [ ] Tapping "Acknowledge" records `acknowledged = true` and a timestamp.
- [ ] The button disappears after acknowledgment; a "Confirmed" indicator replaces it.
- [ ] In the assignment panel (Story 5.1), each assigned slot shows the referee's acknowledgment status.
- [ ] Assignments unacknowledged after 24 hours are highlighted in the assignor's schedule view.

**Dependencies:** 6.1  
**Size:** S

---

### Epic 7 — Infrastructure & Deployment

---

#### Story 7.1 — Docker containerisation

> As a developer, I want the app packaged as Docker containers so that it deploys consistently to any environment.

**Acceptance Criteria**
- [ ] Backend has a multi-stage `Dockerfile`; final image contains only the compiled binary.
- [ ] Frontend has a `Dockerfile` that builds and serves static assets.
- [ ] `docker-compose.yml` runs backend + frontend + PostgreSQL locally with a single command.
- [ ] Database migrations run automatically on backend startup.
- [ ] Environment variables (DB connection string, OAuth2 credentials) are injected via `.env` file locally and environment config in production.

**Dependencies:** 1.1  
**Size:** M

---

#### Story 7.2 — Azure deployment

> As a developer, I want the app running on Azure so that it's accessible to all club members over HTTPS without VPN.

**Acceptance Criteria**
- [ ] Application is accessible via a stable public URL (custom domain or Azure-assigned).
- [ ] HTTPS is enforced; HTTP redirects to HTTPS.
- [ ] PostgreSQL data is persisted (Azure Database for PostgreSQL or equivalent managed service).
- [ ] Database is backed up at least daily (managed service backup is acceptable).
- [ ] Deployment can be updated by pushing a new container image (manual or simple CI step).
- [ ] Azure free-tier credits are used where possible.

**Dependencies:** 7.1  
**Size:** M

---

## Recommended Build Order

| Phase | Epics | Goal |
|-------|-------|------|
| **Phase 1** | 1 + 7.1 | Running skeleton, auth, Docker |
| **Phase 2** | 2 + 3 | Referee profiles, schedule import, match management |
| **Phase 3** | 4 | Eligibility engine, availability marking |
| **Phase 4** | 5 | Assignment interface (core value delivery) |
| **Phase 5** | 6 | Referee assignment view + acknowledgment (stretch) |
| **Phase 6** | 7.2 | Azure deployment |

Phase 6 (deployment) can be started in parallel with Phase 4 or 5 — having a staging environment early de-risks the final launch.

---

## Open Questions (Carry-forward)

1. ~~**Certification levels**: Resolved — binary certified/not with expiry date. A separate assignor-managed grade (Junior / Mid / Senior) is used for assignment guidance only; it is not a hard eligibility gate.~~
2. ~~**Assistant referee age floor on U12+**: Resolved — no minimum age for assistant referees.~~
3. ~~**Multiple assignors**: Resolved — last-write-wins is acceptable at this scale.~~
4. ~~**Stack Team App API**: Deferred to v2+ backlog. Direct schedule sync via API is a future enhancement; CSV import is the only method in v1.~~
5. ~~**CSV duplicate handling**: Resolved — assignor is always prompted to resolve duplicates. Two signals checked: same `reference_id` (Stack Team App known export bug) and same date + start time + location (two team rows for one physical match).~~
