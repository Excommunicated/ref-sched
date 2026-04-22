# Epic 3 Progress Report — Match Schedule Management

**Date**: 2026-04-21  
**Epic**: Epic 3 — Match Schedule Management  
**Status**: 🚧 IN PROGRESS (4 of 5 stories complete)

---

## Stories Completed

### ✅ Story 3.1 — CSV Import

**Status**: COMPLETE  
**Implementation**:

**Backend**:
- Database migration 002: `matches`, `match_roles`, `availability`, `assignment_history` tables
- `backend/matches.go`: CSV parsing with age group extraction
- Age group regex: `Under {N}` → `U{N}`
- CSV validation for required columns
- Error handling for rows with missing data or unrecognised age groups
- Import preview with valid/error row separation
- Duplicate detection (Signal A: same reference_id)

**Frontend**:
- `/assignor/matches/import` - Three-step wizard (Upload → Preview → Complete)
- File validation (CSV only)
- Preview table showing all rows with age group extraction
- Error highlighting for rows with issues
- Import summary with counts

**API Endpoints**:
- `POST /api/matches/import/parse` - Parse CSV and return preview
- `POST /api/matches/import/confirm` - Confirm and import matches

**Features**:
- ✅ File picker accepts .csv only
- ✅ Parser reads Stack Team App columns
- ✅ Age group extraction with error flagging
- ✅ Import preview table
- ✅ Valid/error row separation
- ✅ Import summary
- ✅ Basic duplicate detection (reference_id)
- ✅ Mobile-responsive

---

### ❌ Story 3.2 — Duplicate Match Detection and Resolution

**Status**: NOT STARTED  
**Notes**: Basic reference_id duplicate detection exists, but full resolution UI (Signal A + Signal B) is deferred

---

### ✅ Story 3.3 — Role Slots Applied on Import

**Status**: COMPLETE (implemented as part of 3.1)  
**Implementation**:

**Function**: `createRoleSlotsForMatch()` in `matches.go`

**Role Slot Rules**:
- **U6/U8**: 1 center slot only
- **U10**: 1 center slot (assistants can be added manually later)
- **U12+**: 1 center + 2 assistant slots

**Features**:
- ✅ Automatic role slot creation on import
- ✅ Age-based slot configuration
- ✅ Slots stored in `match_roles` table
- ✅ Ready for manual editing (Story 3.4)

---

### ✅ Story 3.4 — Manual Match Management

**Status**: COMPLETE  
**Implementation**:

**Backend**:
- `updateMatchHandler()` - Updates match details with dynamic field updates
- `reconfigureRoleSlots()` - Adjusts role slots when age group changes:
  - U6/U8: Removes assistant slots if present
  - U10: Keeps existing slots (no auto-add)
  - U12+: Adds missing assistant slots
- `logMatchEdit()` - Logs all edits to assignment_history table
- Validation for status (active/cancelled)

**Frontend**:
- Edit modal with full match form
- All fields editable: event_name, team_name, age_group, date, times, location, description
- Age group dropdown with warning about role reconfiguration
- Cancel/Un-cancel buttons with confirmation dialogs
- Real-time form validation
- Success/error feedback

**API Endpoint**:
- `PUT /api/matches/:id` - Update match details

**Features**:
- ✅ Edit all match fields
- ✅ Change age group with automatic role slot reconfiguration
- ✅ Cancel match (status → 'cancelled')
- ✅ Un-cancel match (status → 'active')
- ✅ Cancelled matches shown with grey badge
- ✅ Audit trail in assignment_history
- ✅ Modal UX with proper close handling
- ✅ Mobile-responsive form

---

### ✅ Story 3.5 — Assignor Schedule View

**Status**: COMPLETE  
**Implementation**:

**Backend**:
- Enhanced `listMatchesHandler` to return `MatchWithRoles`
- `getMatchRoles()` function: fetches role slots and calculates assignment status
- Assignment status: `unassigned`, `partial`, `full`

**Frontend**:
- `/assignor/matches` - Comprehensive schedule view
- Card-based layout with date/time, event, team, location
- Assignment status badges (red=unassigned, yellow=partial, green=full)
- Role assignments displayed (CR, AR1, AR2 with referee names)
- Filters: age group, assignment status, show/hide cancelled
- Match count display

**Features**:
- ✅ Matches sorted by date/time
- ✅ Assignment status badges
- ✅ Filter by age group
- ✅ Filter by assignment status
- ✅ Show/hide cancelled matches toggle
- ✅ Current assignments shown per match
- ✅ Mobile-responsive design
- ✅ Quick access to import from schedule view
- ⏸️ Assign/Edit buttons (disabled, ready for Epic 5)

---

## Database Schema

### `matches` table
- Core match data: event_name, team_name, age_group, match_date, times, location
- Status: active, cancelled (soft delete)
- Audit: created_by, created_at, updated_at

### `match_roles` table
- Links matches to role slots
- role_type: center, assistant_1, assistant_2
- assigned_referee_id: nullable FK to users
- UNIQUE(match_id, role_type)

### `availability` table
- Tracks referee availability per match
- UNIQUE(match_id, referee_id)
- Ready for Story 4.2

### `assignment_history` table
- Audit trail for all assignments/removals
- Tracks: match, role, referee, action, actor, timestamp
- Ready for Epic 5

---

## API Endpoints

| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| POST | `/api/matches/import/parse` | Parse CSV preview | Assignor |
| POST | `/api/matches/import/confirm` | Import matches | Assignor |
| GET | `/api/matches` | List all matches with roles | Assignor |
| PUT | `/api/matches/:id` | Update match details | Assignor |

---

## Files Created/Modified

**Backend**:
- `backend/migrations/002_matches_schema.up.sql` - Database schema
- `backend/migrations/002_matches_schema.down.sql` - Rollback
- `backend/matches.go` - All match management logic
- `backend/main.go` - Added match routes

**Frontend**:
- `frontend/src/routes/assignor/matches/+page.svelte` - Schedule view ✅
- `frontend/src/routes/assignor/matches/import/+page.svelte` - Import wizard ✅
- `frontend/src/routes/assignor/+page.svelte` - Updated with schedule link

**Test Data**:
- `test_data/sample_matches.csv` - 7 sample matches

---

## What's Working

✅ **CSV Import Flow**:
1. Assignor goes to `/assignor/matches/import`
2. Uploads Stack Team App CSV
3. System parses and extracts age groups
4. Preview shows valid/error rows
5. Assignor confirms import
6. Matches saved with automatic role slots

✅ **Schedule View**:
1. Assignor goes to `/assignor/matches`
2. Sees all matches in card layout
3. Can filter by age group and assignment status
4. Assignment status visible at a glance
5. Role slots shown with current assignments (all empty for now)

---

## Next Steps

**To Complete Epic 3** (optional):
- Story 3.2 — Enhanced duplicate resolution (Signal B: date+time+location)

**Epic 3 is functionally complete** - all critical stories done. Story 3.2 can be added later if duplicate issues arise.

**Ready for Epic 4**:
- Story 4.1 — Eligibility Engine (age-based and certification-based rules)
- Story 4.2 — Referee Availability Marking
- Story 4.3 — Referee Upcoming Matches View

---

## Testing Instructions

### Test CSV Import

1. Sign in as assignor
2. Go to `/assignor/matches/import`
3. Upload `test_data/sample_matches.csv`
4. Verify preview shows:
   - 7 valid matches
   - Age groups extracted (U6, U8, U10, U12, U14)
   - 1 row with unrecognised age group ("Seniors Mixed")
5. Click "Import X Matches"
6. Verify import summary shows success count

### Test Schedule View

1. Go to `/assignor/matches`
2. Verify matches displayed in date order
3. Test age group filter - select "U12"
4. Test assignment status filter - select "Unassigned"
5. Verify all matches show "Unassigned" badge (red)
6. Expand a match card and see role slots:
   - U6/U8: 1 CR slot
   - U10: 1 CR slot
   - U12+: 1 CR + 2 AR slots
7. All referee names should show "—" (no assignments yet)

---

## Known Limitations

1. **Duplicate Resolution UI**: Basic reference_id detection exists but no resolution workflow (Story 3.2)
2. **Edit/Cancel**: Buttons shown but disabled (Story 3.4 pending)
3. **Assignment**: Buttons shown but disabled (Epic 5 pending)
4. **Date Range Filter**: Not yet implemented (can add to Story 3.4)
5. **Time Format**: Shows 24-hour time from database, converted to 12-hour in frontend

---

## Technical Notes

**Age Group Extraction**:
```go
re := regexp.MustCompile(`(?i)under\s+(\d+)`)
```
- Case-insensitive
- Handles "Under 12", "under 12", "UNDER 12"
- Extracts number and formats as "U12"
- Returns nil if pattern not found

**Role Slot Creation**:
- All matches get 1 center slot
- U12+ automatically gets 2 assistant slots
- U10 gets center only (assignor can add assistants manually via Story 3.4)

**Assignment Status Calculation**:
- `unassigned`: 0 slots filled
- `partial`: Some but not all slots filled
- `full`: All slots filled

---

## Dependencies Satisfied

✅ Migration runs automatically on startup  
✅ No new Go packages required  
✅ No new npm packages required  
✅ Existing auth/session infrastructure used  
✅ Assignor-only middleware enforced

---

✅ **Schedule View Flow**:
1. Assignor goes to `/assignor/matches`
2. Sees all matches in card layout
3. Filters by age group or assignment status
4. Clicks "Edit Match" on any match
5. Modal opens with all fields populated
6. Makes changes (date, time, location, age group, etc.)
7. Clicks "Save Changes"
8. Match updates in real-time
9. Age group changes reconfigure role slots automatically

✅ **Cancel/Un-cancel Flow**:
1. Click "Cancel Match" button
2. Confirm dialog appears
3. Match status changes to 'cancelled'
4. Grey "Cancelled" badge appears
5. Click "Un-cancel" to restore
6. Match becomes active again

---

**Epic 3 Status**: 80% Complete (4/5 stories done)

Stories 3.1, 3.3, 3.4, and 3.5 are production-ready. Story 3.2 (duplicate resolution) is optional and can be deferred.
