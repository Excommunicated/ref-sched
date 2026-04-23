# Additional Fixes - April 22, 2026

**Date**: 2026-04-22  
**Status**: ✅ **ALL FIXES DEPLOYED**

---

## Summary

Fixed two critical issues:
1. **Added ability to create AR slots for U10 matches**
2. **Prevented assigning same referee to multiple roles on same match**

---

## Issue #1: No Option to Add ARs to U10 Matches

### Problem

**User Reported**:
> "There is still no option to be able to add ARs to a U10 match."

While the previous fix made ARs optional for U10 matches (don't count toward "full" status), there was no UI to actually **create** AR slots for U10 matches. U10 matches are created with only a center referee slot by default.

### Solution Implemented

#### Backend: New API Endpoint

**File**: `backend/matches.go`

**Added**: `addRoleSlotHandler()` function (line ~750)

**Route**: `POST /api/matches/{match_id}/roles/{role_type}/add`

**Functionality**:
- Allows assignors to manually add AR slots (assistant_1, assistant_2)
- Validates match exists and is active
- Prevents duplicate role slots
- Only allows adding AR slots (not center)

**Code**:
```go
func addRoleSlotHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    matchID, err := strconv.ParseInt(vars["match_id"], 10, 64)
    // ... validation ...
    
    roleType := vars["role_type"]
    if roleType != "assistant_1" && roleType != "assistant_2" {
        http.Error(w, "Can only add assistant referee slots", http.StatusBadRequest)
        return
    }
    
    // Check if role slot already exists
    // ... validation ...
    
    // Create the role slot
    _, err = db.Exec(
        "INSERT INTO match_roles (match_id, role_type) VALUES ($1, $2)",
        matchID, roleType,
    )
    // ... error handling ...
}
```

**Registered Route** in `backend/main.go`:
```go
r.HandleFunc("/api/matches/{match_id}/roles/{role_type}/add", 
    authMiddleware(assignorOnly(addRoleSlotHandler))).Methods("POST")
```

#### Frontend: Add AR Slot Buttons

**File**: `frontend/src/routes/assignor/matches/+page.svelte`

**Added Section** (after existing role cards):
```svelte
<!-- Add AR slots for U10 matches -->
{#if assignmentMatch?.age_group && assignmentMatch.age_group <= 'U10'}
    {@const hasAR1 = sortedRoles.some(r => r.role_type === 'assistant_1')}
    {@const hasAR2 = sortedRoles.some(r => r.role_type === 'assistant_2')}
    {#if !hasAR1 || !hasAR2}
        <div class="add-roles-section">
            <h4>Optional Assistant Referees</h4>
            <p class="help-text">
                U10 matches only require a center referee. 
                You can optionally add AR slots below:
            </p>
            <div class="add-roles-buttons">
                {#if !hasAR1}
                    <button class="btn-small btn-secondary" 
                            on:click={() => addRoleSlot('assistant_1')}>
                        + Add AR1 Slot
                    </button>
                {/if}
                {#if !hasAR2}
                    <button class="btn-small btn-secondary" 
                            on:click={() => addRoleSlot('assistant_2')}>
                        + Add AR2 Slot
                    </button>
                {/if}
            </div>
        </div>
    {/if}
{/if}
```

**Added Function**:
```typescript
async function addRoleSlot(roleType: string) {
    assigning = true;
    assignmentError = '';
    
    try {
        const response = await fetch(
            `${API_URL}/api/matches/${assignmentMatch.id}/roles/${roleType}/add`,
            {
                method: 'POST',
                credentials: 'include'
            }
        );
        
        if (response.ok) {
            await loadMatches();
            // Update assignmentMatch with the refreshed data
            const refreshedMatch = matches.find(m => m.id === assignmentMatch.id);
            if (refreshedMatch) {
                assignmentMatch = refreshedMatch;
            }
        } else {
            const text = await response.text();
            assignmentError = text || 'Failed to add role slot';
        }
    } catch (err) {
        assignmentError = 'Failed to add role slot';
    } finally {
        assigning = false;
    }
}
```

**Added CSS** (dashed border box style):
```css
.add-roles-section {
    margin-top: 1.5rem;
    padding: 1.5rem;
    border: 2px dashed var(--border-color);
    border-radius: 0.5rem;
    background-color: #fafafa;
}
```

### Result

✅ **UI appears for U10 matches only**  
✅ **Shows which AR slots are missing**  
✅ **Click "+ Add AR1 Slot" or "+ Add AR2 Slot"**  
✅ **Slot created instantly, appears in roles list**  
✅ **Can then assign referee to the new slot**  
✅ **Match still shows as "full" with only center assigned**  

---

## Issue #2: Can Assign Same Referee to Multiple Roles

### Problem

**User Reported**:
> "Also an assignor can assign the same referee to multiple roles on the same match. That should not be possible."

The system allowed assigning the same referee as both center referee and AR1 (or any combination of roles) on the same match, which is invalid.

### Solution Implemented

**File**: `backend/assignments.go`

**Modified**: `assignRefereeHandler()` function (line ~72-92)

**Added Validation**:
```go
// Check if referee is already assigned to another role on this match
var existingRoleType sql.NullString
err = db.QueryRow(`
    SELECT role_type
    FROM match_roles
    WHERE match_id = $1
      AND assigned_referee_id = $2
      AND role_type != $3
`, matchID, *req.RefereeID, roleType).Scan(&existingRoleType)

if err != nil && err != sql.ErrNoRows {
    http.Error(w, fmt.Sprintf("Database error: %v", err), 
        http.StatusInternalServerError)
    return
}

if existingRoleType.Valid {
    roleName := map[string]string{
        "center":      "Center Referee",
        "assistant_1": "Assistant Referee 1",
        "assistant_2": "Assistant Referee 2",
    }
    http.Error(w, 
        fmt.Sprintf("Referee is already assigned as %s for this match", 
            roleName[existingRoleType.String]), 
        http.StatusBadRequest)
    return
}
```

### How It Works

**Query Logic**:
1. Check `match_roles` table for the match
2. Look for assigned referee ID
3. Exclude the current role being assigned (`role_type != $3`)
4. If found, return error with clear message

**Error Messages**:
- "Referee is already assigned as Center Referee for this match"
- "Referee is already assigned as Assistant Referee 1 for this match"
- "Referee is already assigned as Assistant Referee 2 for this match"

### Result

✅ **Cannot assign same referee to center and AR1**  
✅ **Cannot assign same referee to AR1 and AR2**  
✅ **Cannot assign same referee to center and AR2**  
✅ **Clear error message displays in assignment panel**  
✅ **Can still assign same referee to different matches**  

---

## Files Changed

### Backend

1. **`backend/assignments.go`** (Line ~72-108)
   - Added validation to prevent double-assignment
   - Returns descriptive error message

2. **`backend/matches.go`** (Line ~750-804)
   - Added `addRoleSlotHandler()` function
   - Allows creating AR slots for matches

3. **`backend/main.go`** (Line ~134)
   - Registered new route for adding role slots

### Frontend

1. **`frontend/src/routes/assignor/matches/+page.svelte`**
   - Added UI section for U10 matches (line ~669-696)
   - Added `addRoleSlot()` function (line ~344-371)
   - Added CSS for add-roles-section (line ~1829-1854)

---

## API Documentation

### New Endpoint: Add Role Slot

**Endpoint**: `POST /api/matches/{match_id}/roles/{role_type}/add`

**Auth**: Assignor only

**Path Parameters**:
- `match_id` (int64): Match ID
- `role_type` (string): Must be `assistant_1` or `assistant_2`

**Response Success** (201 Created):
```json
{
  "success": true,
  "role_type": "assistant_1"
}
```

**Response Errors**:
- **400 Bad Request**: Invalid role type (not AR) or role already exists
- **404 Not Found**: Match not found or not active

**Example**:
```bash
curl -X POST http://localhost:8080/api/matches/123/roles/assistant_1/add \
  --cookie "session=..." \
  -H "Content-Type: application/json"
```

---

## Testing Instructions

### Test #1: Add AR Slots to U10 Match

**Prerequisites**: Have a U10 match in the system

1. Sign in as assignor
2. Go to "Match Schedule"
3. Find a U10 match (should show age badge "U10")
4. Click "Assign Referees" on the match
5. **Expected**: See role card for "Center Referee" only

6. Scroll down below the role cards
7. **Expected**: See section "Optional Assistant Referees"
8. **Expected**: See message: "U10 matches only require a center referee. You can optionally add AR slots below:"
9. **Expected**: See buttons "+ Add AR1 Slot" and "+ Add AR2 Slot"

10. Click "+ Add AR1 Slot"
11. **Expected**: Button disappears, new role card appears for "Assistant Referee 1"
12. **Expected**: AR1 role card shows "Open" status

13. Click "+ Add AR2 Slot"
14. **Expected**: Button disappears, new role card appears for "Assistant Referee 2"
15. **Expected**: Entire "Optional Assistant Referees" section disappears (both slots added)

16. Verify match status still shows "Full" or "Partial" based on center referee assignment

### Test #2: Prevent Double Assignment

**Setup**: Have a match with center and at least one AR slot

1. Sign in as assignor
2. Go to "Match Schedule"
3. Find any match with multiple role slots
4. Click "Assign Referees"

5. Click "Select Referee" for Center Referee
6. Choose a referee (e.g., "John Smith")
7. Verify assignment succeeds

8. Go back to role selection (← Back button)
9. Click "Select Referee" for AR1
10. Try to select the SAME referee ("John Smith")
11. **Expected**: Error message appears at top of panel:
    - "Referee is already assigned as Center Referee for this match"
12. **Expected**: Assignment does NOT go through

13. Select a DIFFERENT referee
14. **Expected**: Assignment succeeds

15. Go back and try to assign AR2
16. Try to assign "John Smith" again
17. **Expected**: Same error: "Referee is already assigned as Center Referee for this match"

18. Try to assign the referee you just assigned to AR1
19. **Expected**: Error: "Referee is already assigned as Assistant Referee 1 for this match"

20. Select a different referee
21. **Expected**: Assignment succeeds

### Test #3: Can Still Assign Same Referee to Different Matches

1. Assign a referee to Match A as center referee
2. Go to Match B
3. Try to assign same referee as center referee
4. **Expected**: Assignment succeeds (different matches allowed)

---

## Edge Cases Handled

### Add Role Slot Endpoint

✅ **U10 match already has AR slots**: Section doesn't appear  
✅ **Trying to add center role**: Returns error "Can only add assistant referee slots"  
✅ **Role already exists**: Returns error "Role slot already exists for this match"  
✅ **Match doesn't exist**: Returns error "Match not found or not active"  
✅ **Cancelled match**: Returns error "Match not found or not active"  

### Double Assignment Prevention

✅ **Assigning to same role**: Allowed (reassignment)  
✅ **Assigning to different role on same match**: Blocked with error  
✅ **Assigning to different match**: Allowed  
✅ **Removing assignment**: Doesn't trigger validation  

---

## Database Queries

### Check for Existing Assignment
```sql
SELECT role_type
FROM match_roles
WHERE match_id = $1
  AND assigned_referee_id = $2
  AND role_type != $3  -- Exclude current role being assigned
```

### Create Role Slot
```sql
INSERT INTO match_roles (match_id, role_type) 
VALUES ($1, $2)
```

### Check if Role Exists
```sql
SELECT EXISTS(
  SELECT 1 
  FROM match_roles 
  WHERE match_id = $1 AND role_type = $2
)
```

---

## Deployment

### Applied
1. ✅ Backend code updated (assignments.go, matches.go, main.go)
2. ✅ Backend restarted
3. ✅ Frontend code updated (assignor matches page)
4. ✅ Frontend rebuilt and restarted

### Verification
```bash
docker-compose ps

# All services should be "Up"
```

---

## Impact

### Before Fixes
- ❌ No way to add AR slots to U10 matches
- ❌ Could assign same referee to center + AR1 on same match
- ❌ Confusing assignment state
- ❌ Invalid referee assignments

### After Fixes
- ✅ Can optionally add AR slots to U10 matches
- ✅ Clear UI indicating slots are optional
- ✅ Cannot assign same referee to multiple roles
- ✅ Clear error messages for invalid assignments
- ✅ Maintains integrity of match assignments

---

## User Experience Flow

### Adding ARs to U10 Match

```
1. Open U10 match assignment panel
   ↓
2. See "Optional Assistant Referees" section
   ↓
3. Click "+ Add AR1 Slot"
   ↓
4. AR1 role card appears
   ↓
5. Click "Select Referee" on AR1
   ↓
6. Choose referee and assign
   ↓
7. AR1 filled, match still shows "Full" (ARs optional)
```

### Attempting Double Assignment

```
1. Assign John to Center Referee
   ↓
2. Try to assign John to AR1
   ↓
3. Error: "Referee is already assigned as Center Referee for this match"
   ↓
4. Assignment blocked, can select different referee
```

---

## Related Features

This completes the U10 optional AR feature:
- **Previous**: U10 assignment status doesn't require ARs ✓
- **This Fix**: UI to add AR slots to U10 matches ✓
- **This Fix**: Prevent same referee on multiple roles ✓

---

## Browser Cache

If you don't see the changes:
1. **Hard refresh**: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac)
2. **Clear cache**: Browser settings → Clear browsing data
3. **Incognito window**: Test in private browsing mode

---

## Summary

**Two Critical Issues Fixed**:

1. ✅ **Add AR slots to U10 matches**
   - New backend endpoint
   - UI section with "+ Add AR Slot" buttons
   - Only appears for U10 matches
   - Slots created instantly

2. ✅ **Prevent double assignments**
   - Backend validation
   - Clear error messages
   - Maintains assignment integrity

**All changes deployed and ready for testing!** 🎉

The assignment system now:
- ✓ Supports optional ARs for U10 matches
- ✓ Prevents invalid referee assignments
- ✓ Provides clear UI for adding optional roles
- ✓ Shows helpful error messages
