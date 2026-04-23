# Scheduling Conflict Warnings & Acknowledgment Reset Fix

**Date**: 2026-04-22  
**Status**: ✅ **DEPLOYED**

---

## Summary

Implemented two important enhancements:
1. **Scheduling conflict warnings** for referees before they acknowledge assignments
2. **Automatic acknowledgment reset** when referees are removed/reassigned

---

## Feature #1: Scheduling Conflict Warnings for Referees

### Problem

**User Request**:
> "When showing a referee their assignments can we also show a warning if there are scheduling overlaps so they are aware before they acknowledge."

Currently, assignors get warnings about time conflicts, but referees don't see any indication that they've been assigned to overlapping matches until they acknowledge.

### Solution Implemented

Added automatic conflict detection and prominent warnings for referees when viewing their assignments.

#### Backend Changes

**File**: `backend/availability.go`

**Added Struct** (line ~14-20):
```go
type ConflictingMatch struct {
    MatchID   int64  `json:"match_id"`
    EventName string `json:"event_name"`
    TeamName  string `json:"team_name"`
    StartTime string `json:"start_time"`
    RoleType  string `json:"role_type"`
}
```

**Modified Struct** (line ~22-36):
```go
type MatchForReferee struct {
    // ... existing fields ...
    HasConflict       bool               `json:"has_conflict"`
    ConflictingMatches []ConflictingMatch `json:"conflicting_matches,omitempty"`
}
```

**Added Conflict Detection** (line ~228-258):
```go
// After determining a match is assigned, check for conflicts
if assignedRole.Valid {
    m.IsAssigned = true
    m.AssignedRole = &assignedRole.String
    // ... acknowledgment code ...
    
    // Check for scheduling conflicts with other assignments
    conflictRows, err := db.Query(`
        SELECT
            m2.id, m2.event_name, m2.team_name,
            m2.start_time, mr2.role_type
        FROM matches m2
        JOIN match_roles mr2 ON mr2.match_id = m2.id
        WHERE mr2.assigned_referee_id = $1
          AND m2.id != $2
          AND m2.status = 'active'
          AND m2.match_date = $3
          AND (
            (m2.start_time, m2.end_time) OVERLAPS ($4::time, $5::time)
          )
    `, user.ID, m.ID, matchDate, m.StartTime, m.EndTime)
    
    // Process conflicts...
    if len(conflicts) > 0 {
        m.HasConflict = true
        m.ConflictingMatches = conflicts
    }
}
```

#### Frontend Changes

**File**: `frontend/src/routes/referee/matches/+page.svelte`

**Added Interface** (line ~9-15):
```typescript
interface ConflictingMatch {
    match_id: number;
    event_name: string;
    team_name: string;
    start_time: string;
    role_type: string;
}
```

**Updated Match Interface** (line ~28-30):
```typescript
interface Match {
    // ... existing fields ...
    has_conflict?: boolean;
    conflicting_matches?: ConflictingMatch[];
}
```

**Added Warning Banner** (line ~348-369):
```svelte
{#if match.has_conflict && match.conflicting_matches && match.conflicting_matches.length > 0}
    <div class="scheduling-conflict-warning">
        <span class="warning-icon">⚠️</span>
        <div class="warning-text">
            <strong>Scheduling Conflict Detected</strong>
            <p>This assignment overlaps with {match.conflicting_matches.length} 
               other assignment{match.conflicting_matches.length > 1 ? 's' : ''}:</p>
            <ul class="conflict-list">
                {#each match.conflicting_matches as conflict}
                    <li>
                        <strong>{formatTime(conflict.start_time)}</strong> - 
                        {conflict.event_name} ({conflict.team_name})
                        as {conflict.role_type}
                    </li>
                {/each}
            </ul>
            <p class="conflict-advice">
                Please contact the assignor immediately to resolve this conflict 
                before acknowledging.
            </p>
        </div>
    </div>
{/if}
```

**Added CSS** (line ~829-875):
```css
.scheduling-conflict-warning {
    background: #fee2e2;
    border-left: 4px solid #dc2626;
    padding: 1rem;
    margin-bottom: 1rem;
    /* Red theme for high urgency */
}

.match-card:has(.scheduling-conflict-warning) {
    border: 3px solid #dc2626;
}
```

### How It Works

1. **Backend Detection**:
   - When loading matches for a referee
   - For each assigned match, query for time overlaps
   - Check same day, overlapping time windows
   - Return conflict details with match data

2. **Frontend Display**:
   - Red warning banner appears above match details
   - Lists all conflicting assignments with times
   - Shows which role for each conflict
   - Recommends contacting assignor

3. **Visual Hierarchy**:
   - **Red border** on entire match card
   - **Red background** on warning banner
   - **Warning icon** (⚠️)
   - **Bold conflict times** for easy scanning

### Example Warning

```
┌────────────────────────────────────────────────┐
│ ⚠️ Scheduling Conflict Detected                │
│                                                │
│ This assignment overlaps with 2 other          │
│ assignments:                                   │
│                                                │
│ • 3:00 PM - Spring Tournament (U12 Girls)     │
│   as Center Referee                           │
│ • 3:30 PM - Fall League (U14 Boys) as AR1     │
│                                                │
│ Please contact the assignor immediately to    │
│ resolve this conflict before acknowledging.   │
└────────────────────────────────────────────────┘
```

---

## Feature #2: Automatic Acknowledgment Reset

### Problem

**User Request**:
> "Also when a referee is removed from an assignment, we should also remove their acknowledgment from that database so if they are re assigned again in the future, it is not pre acknowledged."

When an assignor:
1. Assigns Referee A to a match
2. Referee A acknowledges
3. Assignor removes Referee A
4. Assignor assigns Referee A back to same match later

**Bug**: The assignment would show as already acknowledged (from step 2).

**Why This Is a Problem**:
- Stale acknowledgments
- No fresh confirmation
- Referee might not actually be available anymore
- Misleading status information

### Solution Implemented

**File**: `backend/assignments.go`

**Modified Query** (line ~127-134):
```go
// When removing or reassigning, also clear acknowledgment
// This ensures a new/different referee must acknowledge the assignment
_, err = db.Exec(`
    UPDATE match_roles
    SET assigned_referee_id = $1,
        acknowledged = false,
        acknowledged_at = NULL
    WHERE id = $2
`, newRefereeID, roleID)
```

### How It Works

**Before** (buggy):
```
1. Assign Referee A → acknowledged=false
2. Referee A acknowledges → acknowledged=true, acknowledged_at=2026-04-22
3. Remove Referee A → assigned_referee_id=NULL, acknowledged=true (STALE!)
4. Re-assign Referee A → assigned_referee_id=A, acknowledged=true (BUG!)
```

**After** (fixed):
```
1. Assign Referee A → acknowledged=false
2. Referee A acknowledges → acknowledged=true, acknowledged_at=2026-04-22
3. Remove Referee A → assigned_referee_id=NULL, acknowledged=false, acknowledged_at=NULL
4. Re-assign Referee A → assigned_referee_id=A, acknowledged=false ✓
```

### Applies To

This fix applies to:
- ✅ **Removing assignment** (setting referee_id to NULL)
- ✅ **Reassigning** (changing from Referee A to Referee B)
- ✅ **Same referee re-assigned** (removed then assigned back)

---

## Files Changed

### Backend

1. **`backend/availability.go`**
   - Added `ConflictingMatch` struct
   - Added conflict fields to `MatchForReferee`
   - Added conflict detection query
   - Lines: ~14-20, ~22-36, ~228-258

2. **`backend/assignments.go`**
   - Modified assignment UPDATE query
   - Reset acknowledged fields on assignment change
   - Lines: ~127-134

### Frontend

1. **`frontend/src/routes/referee/matches/+page.svelte`**
   - Added conflict interface and fields
   - Added conflict warning UI
   - Added conflict warning CSS
   - Lines: ~9-15, ~28-30, ~348-369, ~829-875

---

## Testing Instructions

### Test #1: Conflict Warning Display

**Setup**: Need 2 matches on same day with overlapping times

1. Sign in as assignor
2. Go to "Match Schedule"
3. Find/create two matches on same day:
   - Match A: 3:00 PM - 4:30 PM
   - Match B: 3:30 PM - 5:00 PM (overlaps by 1 hour)

4. Assign same referee to BOTH matches:
   - Match A → Assign Referee John as Center
   - Match B → Assign Referee John as AR1

5. Sign in as referee (John)
6. Go to "My Assignments"
7. **Expected**: 
   - BOTH matches show red border
   - BOTH matches have red conflict warning banner
   - Each banner lists the OTHER conflicting match
   - Warning shows time, event, team, and role

8. **Expected Warning Text**:
   - "Scheduling Conflict Detected"
   - "This assignment overlaps with 1 other assignment:"
   - Shows conflicting match details
   - "Please contact the assignor immediately..."

9. Verify cannot (shouldn't) acknowledge without resolving

### Test #2: Acknowledgment Reset on Remove

1. Sign in as assignor
2. Assign Referee A to a match
3. Sign in as Referee A
4. Go to "My Assignments"
5. Click "Acknowledge Assignment" ✓
6. Verify shows "Confirmed"

7. Sign back in as assignor
8. Open that match's assignment panel
9. Click "Remove" for Referee A
10. **Expected**: Assignment removed

11. Re-assign Referee A to same match
12. Sign back in as Referee A
13. Go to "My Assignments"
14. **Expected**: 
    - Match shows "Acknowledge Assignment" button (NOT "Confirmed")
    - acknowledged=false in database
    - Must acknowledge again

### Test #3: Acknowledgment Reset on Reassign

1. Assign Referee A to Match as Center
2. Referee A acknowledges
3. Assignor changes assignment to Referee B (same role)
4. Sign in as Referee B
5. **Expected**: Must acknowledge (not pre-acknowledged)

---

## Database Schema

No schema changes required. Using existing fields:

### match_roles Table
```sql
CREATE TABLE match_roles (
    id SERIAL PRIMARY KEY,
    match_id INTEGER NOT NULL,
    role_type VARCHAR(50) NOT NULL,
    assigned_referee_id INTEGER,  -- NULL when unassigned
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## SQL Queries

### Conflict Detection
```sql
SELECT
    m2.id, m2.event_name, m2.team_name,
    m2.start_time, mr2.role_type
FROM matches m2
JOIN match_roles mr2 ON mr2.match_id = m2.id
WHERE mr2.assigned_referee_id = $1  -- This referee
  AND m2.id != $2                   -- Different match
  AND m2.status = 'active'
  AND m2.match_date = $3            -- Same day
  AND (
    (m2.start_time, m2.end_time) OVERLAPS ($4::time, $5::time)
  )
```

**PostgreSQL OVERLAPS**:
```
(start1, end1) OVERLAPS (start2, end2)
```
Returns TRUE if time ranges overlap.

### Assignment Update with Ack Reset
```sql
UPDATE match_roles
SET assigned_referee_id = $1,
    acknowledged = false,
    acknowledged_at = NULL
WHERE id = $2
```

---

## Impact

### Before Fixes

**Conflict Detection**:
- ❌ Referees unaware of scheduling conflicts
- ❌ Acknowledge conflicting assignments unknowingly
- ❌ Show up late or not at all due to double-booking
- ❌ Assignors don't get early warning from referee

**Acknowledgment**:
- ❌ Stale acknowledgments persist
- ❌ Re-assigned referees show as pre-acknowledged
- ❌ No fresh confirmation required
- ❌ Misleading status information

### After Fixes

**Conflict Detection**:
- ✅ Referees immediately see scheduling conflicts
- ✅ Clear visual warnings before acknowledgment
- ✅ Lists all conflicting assignments
- ✅ Can contact assignor to resolve
- ✅ Informed decision-making

**Acknowledgment**:
- ✅ Acknowledgments reset on assignment change
- ✅ Fresh confirmation always required
- ✅ Accurate status tracking
- ✅ No stale data

---

## Edge Cases Handled

### Conflict Detection

✅ **Multiple conflicts**: Shows all overlapping assignments  
✅ **Different roles**: Shows which role for each conflict  
✅ **Same day only**: Only checks conflicts on same calendar day  
✅ **Active matches only**: Ignores cancelled matches  
✅ **Query failure**: Doesn't break entire request, just logs warning  

### Acknowledgment Reset

✅ **Remove then reassign**: Requires new acknowledgment  
✅ **Change referee**: New referee must acknowledge  
✅ **Same referee reassigned**: Must acknowledge again  
✅ **Null assignments**: Properly clears all fields  

---

## User Experience

### Referee View - With Conflict

```
┌─────────────────────────────────────────────┐
│ MY ASSIGNMENTS (2)                          │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐ ← Red border
│ ⚠️ Scheduling Conflict Detected             │
│                                             │
│ This assignment overlaps with 1 other       │
│ assignment:                                 │
│                                             │
│ • 3:30 PM - Fall League (U14 Boys) as AR1  │
│                                             │
│ Please contact assignor immediately...      │
├─────────────────────────────────────────────┤
│ Spring Tournament                           │
│ U12 Girls - Center Referee                  │
│ 📅 Saturday, April 23                       │
│ 🕐 3:00 PM (Meet: 2:45 PM)                  │
│ 📍 Lincoln Park - Field 2                   │
│                                             │
│ [Acknowledge Assignment]                    │
└─────────────────────────────────────────────┘
```

### Assignor View - After Reassignment

```
Referee A was assigned and acknowledged ✓
↓
Assignor removes Referee A
↓
Assignor assigns Referee A back
↓
Status: Pending (not pre-acknowledged) ✓
```

---

## Browser Cache

If you don't see the changes:
1. **Hard refresh**: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac)
2. **Clear cache**: Browser settings → Clear browsing data
3. **Incognito window**: Test in private browsing mode

---

## Deployment

### Applied
1. ✅ Backend conflict detection added
2. ✅ Backend acknowledgment reset implemented
3. ✅ Frontend conflict warnings added
4. ✅ Backend restarted
5. ✅ Frontend rebuilt and restarted

### Verification
```bash
docker-compose ps

# All services should be "Up"
```

---

## Summary

**Two Enhancements Implemented**:

1. ✅ **Scheduling Conflict Warnings**
   - Automatic detection of time overlaps
   - Prominent red warning banners
   - Lists all conflicting assignments
   - Encourages contacting assignor

2. ✅ **Acknowledgment Reset**
   - Clears acknowledged status on removal
   - Clears acknowledged status on reassignment
   - Requires fresh confirmation
   - Prevents stale acknowledgments

**All changes deployed and ready for testing!** 🎉

The system now provides better communication and data integrity for referee assignments.
