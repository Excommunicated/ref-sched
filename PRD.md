# PRD: Referee Scheduling App

## Objective

Club soccer associations rely on a manual, email-based process to schedule referees for matches — a workflow that consumes multiple days of an assignor's time per weekend. This application replaces that process with a web-based platform where referees self-manage their availability and qualifications, and assignors assign referees to matches through a simple, eligibility-aware interface. Success is measured by reducing the end-to-end scheduling cycle from multiple days to a few hours.

---

## Background & Context

Today, the assignor emails a spreadsheet screenshot to all referees, waits up to two days for availability replies, manually compiles the data, assigns referees, re-emails the schedule, and then collects acknowledgments. This process is fragile, asynchronous, and does not scale — every cancellation or reschedule restarts the cycle.

No existing tool fits this club's specific needs: the match schedule lives in Stack Team App (a third-party club management system), eligibility rules are age-group-specific, and the assignor needs a focused matching interface rather than a general-purpose scheduling tool.

---

## Users

| Persona | Description | Primary Need |
|---------|-------------|--------------|
| **Assignor** | Club representative (1–2 people). Understands match requirements and referee qualifications. Moderate technical comfort. Uses desktop primarily, but may reference on mobile. | Quickly assign qualified, available referees to all matches for the upcoming weekend. |
| **Referee** | Club referees (~20 people). Varying ages (some as young as 11). Must work on mobile. Low-to-moderate technical comfort. | Easily communicate availability and keep their profile current without emailing back and forth. |

---

## Scope

### In scope (v1)

- **Match schedule import**: Upload a CSV exported from Stack Team App to populate the match schedule.
- **Match schedule management**: Assignor can manually update, cancel, or reschedule matches after import.
- **Referee profiles**: Referees manage their own age, certifications, and other qualifications used for eligibility matching.
- **Availability / interest marking**: Referees view upcoming matches they are eligible for and mark which ones they are interested in being considered for.
- **Eligibility-aware assignment interface**: Assignor views a match, sees pre-filtered lists of eligible and available referees per role (center / assistant), and assigns with minimal clicks.
- **Role-based match requirements**: The system enforces how many referees of each role a match needs, based on age group.
- **Assignment acknowledgment**: Referees can view their assigned matches and acknowledge them in-app (nice-to-have; may be deferred if time-constrained).
- **Social login**: Authentication via Google OAuth2 only. No passwords stored.
- **Responsive web app**: Full functionality on mobile browsers; no native iOS/Android app.

### Out of scope (v1)

- Auto-assignment / AI-driven scheduling
- League or team management
- Payments or expense tracking
- Push notifications or email notifications
- Stack Team App API integration (CSV import only)
- Native mobile applications
- Facebook or other non-Google social login providers

---

## Functional Requirements

1. **FR-01**: The assignor can upload a CSV file in the Stack Team App export format; the system parses and imports match records including event name, date, time, location (venue + specific field from description), and age group.
2. **FR-02**: The system infers the age group from the `team_name` field using the pattern `"Under {N} {Gender} [- Team]"` (e.g., `"Under 12 Girls - Falcons"` → U12). Role requirements and eligibility rules are applied automatically based on the extracted age group.
3. **FR-03**: During import, the system detects and prompts the assignor to resolve potential duplicates before any rows are written. Two duplicate conditions are checked:
   - **Same `reference_id`**: flags the Stack Team App known bug where a missing match is replaced by a duplicate entry with an identical `reference_id`. The assignor is shown both rows and must choose which (if either) to import.
   - **Same date + start time + location (different `reference_id`)**: flags the case where the same physical match appears twice under different team entries. The assignor is shown both rows and must choose which to import.
4. **FR-04**: The assignor can manually edit, cancel, or reschedule any match after import.
5. **FR-05**: Referees can log in via Google OAuth2 and access a personal profile where they record their date of birth, certification details (including expiry date), and any other qualifications.
6. **FR-06**: The system computes a referee's age at the time of any given match for eligibility evaluation, and flags certifications that are expired or will expire before a match date.
7. **FR-07**: A referee can view only the matches for which they are eligible and mark the ones they are willing to be considered for.
8. **FR-08**: Eligibility rules are enforced as follows:
   - **U12 and older — Center referee**: Must hold a current (non-expired) certification. No minimum age beyond the age-based rule below.
   - **U12 and older — Assistant referees**: No certification required. No minimum age requirement.
   - **U10 and younger — any role**: Referee must be at least 1 year older than the age group at the time of the match (e.g., must be ≥ 11 to referee a U10 match). No certification required.
9. **FR-09**: Role slot requirements per age group:
   - **U6 / U8**: 1 Center Referee, 0 Assistants.
   - **U10**: 1 Center Referee by default; assignor may assign up to 2 Assistant Referees if the pool allows.
   - **U12 and older**: 1 Center Referee, 2 Assistant Referees.
10. **FR-10**: The assignor's assignment interface shows, per match role, only referees who are eligible for that role and have marked availability for that match.
11. **FR-11**: The assignor can assign a referee to a specific role slot on a match and can reassign or remove assignments at any time before the match date.
12. **FR-12**: Referees can view their upcoming assignments including the full location (venue name/address and specific field) and meeting time.
13. **FR-13**: Assigned referees can view their upcoming assignments in-app and (if implemented in v1) mark each assignment as acknowledged. This requirement may be deferred to v2 if time-constrained.
14. **FR-14**: The assignor can view the overall schedule with assignment completion status (fully assigned, partially assigned, unassigned) at a glance.
15. **FR-15**: Any person with a Google account can register and complete their referee profile (date of birth, certifications). However, they have a **pending** status and cannot view the match schedule or mark availability until an assignor explicitly verifies and activates their account.
16. **FR-16**: The assignor has a referee management view listing all registered referees with their verification status (pending / active / inactive). The assignor can activate, deactivate, or remove referees from the pool.
17. **FR-17**: The assignor can assign a grade of **Junior**, **Mid**, or **Senior** to each active referee. Grade is set and updated exclusively by the assignor (referees cannot self-grade). Grade is displayed in the assignment interface to help the assignor make informed assignment decisions based on domain knowledge. Grade is advisory only and does not act as a hard eligibility gate.

---

## Non-Functional Requirements

| Category | Requirement | Priority |
|----------|-------------|----------|
| **Responsiveness** | All screens must be fully functional on mobile browsers (320px and up). | High |
| **Performance** | Pages must load in under 2 seconds on a standard mobile connection given the small user base (~22 users). | Medium |
| **Security** | No passwords stored. OAuth2 tokens handled securely. Referee PII (name, DOB) stored but no payment or government ID data collected. | High |
| **Availability** | Best-effort uptime; brief downtime during deployments is acceptable given the small team. | Low |
| **Auditability** | Assignment history should be retained (who assigned whom, when) for at least one season. | Medium |
| **Maintainability** | Single developer; prefer simple, well-understood technology choices over complex ones. | High |

---

## Integrations & Dependencies

| System | Integration Method | Notes |
|--------|--------------------|-------|
| **Stack Team App** | CSV file export / manual upload | No public API confirmed. CSV fields: `event_name`, `team_name`, `start_date`, `end_date`, `start_time`, `end_time`, `description`, `location`, `access_groups`, `reference_id`. Age group must be inferred from team name strings in `access_groups`. |
| **Google OAuth2** | OAuth2 / OpenID Connect | Social login only; no password storage. |

---

## Recommended Technical Approach

Given the developer's background and constraints, the following stack is recommended (not prescriptive):

- **Backend**: Go (REST API or lightweight RPC). Aligns with developer fluency and performs well for this scale.
- **Database**: PostgreSQL. Relational model fits the structured scheduling data; developer is fluent in SQL.
- **Frontend**: SvelteKit or Next.js (React). Modern, mobile-friendly, component-based. Lower learning curve than returning to Angular after 10 years.
- **Auth**: Google OAuth2 via a library (e.g., `golang.org/x/oauth2` on the backend, or a managed auth service like Auth.js).
- **Hosting**: Microsoft Azure (free monthly credits). Containerised deployment (Docker) for portability between Azure and self-hosted Proxmox.

---

## Success Metrics

| Metric | Target |
|--------|--------|
| End-to-end scheduling cycle time | Reduced from ~2 days to ≤ 4 hours |
| Assignor manual steps eliminated | No email compilation; single-screen assignment workflow |
| Referee response method | 100% in-app (no email responses required) |
| Assignment coverage | Assignor can fully staff a weekend's matches in one session |

---

## Risks & Assumptions

| Risk / Assumption | Likelihood | Impact | Mitigation |
|-------------------|------------|--------|------------|
| Stack Team App CSV format changes between seasons | Medium | Medium | Build a configurable column-mapping step in the import flow so field names can be remapped without code changes. |
| Age group cannot be reliably parsed from team name strings | Medium | High | Validate parsing logic against real exports early; allow assignor to manually tag age group on import if parsing fails. |
| Stack Team App has a usable API (not yet investigated) | Unknown | Low | Investigate early; if available, API import is a v2 enhancement. CSV remains the v1 path regardless. |
| Developer unfamiliar with chosen frontend framework | High | Medium | Allocate learning time early; choose a framework with strong documentation and a small surface area for this use case. |
| Referee acknowledgment feature adds scope | Medium | Low | Treat as a stretch goal; stub the UI but cut if it threatens the August deadline. |
| Single-developer project; no redundancy | High | High | Keep the architecture simple; prioritise a working MVP over feature completeness. |

---

## Open Questions

1. ~~**Referee access control**: Resolved — any Google account can register, but the assignor must explicitly verify and add the referee to the active pool before they can view or request matches. Unverified accounts can log in and complete their profile but cannot interact with the schedule.~~
2. ~~**Certification levels**: Resolved — certification is binary (certified / not certified) with an expiry date. No named grade levels. A separate assignor-managed referee grade (Junior / Mid / Senior) is used for assignment guidance (FR-17).~~
3. ~~**Assistant referee age floor on U12+**: Resolved — no minimum age requirement for assistant referees.~~
4. ~~**Multiple assignors**: Resolved — last-write-wins is acceptable at this scale.~~
5. ~~**Stack Team App API**: Deferred — moved to v2+ backlog. Direct API integration with Stack Team App is a future enhancement; CSV import is the only supported method in v1.~~
6. ~~**CSV duplicate handling**: Resolved — assignor is always prompted to resolve duplicates before import. Two duplicate signals are checked: same `reference_id` (Stack Team App known bug) and same date + start time + location (two team rows for the same physical match).~~

---

## Out of Scope (v1)

To prevent scope creep, the following are explicitly excluded from this version:

- Automated or AI-assisted referee assignment
- Any form of payment, invoicing, or mileage tracking
- Email or push notification delivery
- League, team, or player management
- Facebook or other OAuth providers beyond Google
- Native iOS or Android applications
- Integration with any external system beyond CSV file import (Stack Team App API integration is a tracked v2+ backlog item)
