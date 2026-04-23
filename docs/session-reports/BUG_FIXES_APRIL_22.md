# Bug Fixes & Feature - April 22, 2026

**Date**: 2026-04-22  
**Status**: ✅ **ALL FIXES DEPLOYED**

---

## Summary

Fixed two critical UI bugs and implemented optional Assistant Referees for U10 matches.

---

## Bug #1: Clear Preference Button Invisible

### Problem
The "Clear Preference" button (—) on match availability had a white icon on white background when a match was marked as available or unavailable, making it invisible in light mode.

### Root Cause
The `.btn-clear` CSS class did not specify a default color for the non-active state, causing it to inherit white or transparent text on a white background.

### Fix Applied

**File**: `frontend/src/routes/referee/matches/+page.svelte`

**Change**:
```css
/* Added default color for non-active state */
.btn-clear {
    color: #6b7280; /* Ensure icon is visible when not active */
}

.btn-clear.active {
    background: #6b7280;
    color: white;
    border-color: #6b7280;
}
```

### Result
✅ Clear preference button (—) now visible in all states
✅ Consistent with hover behavior
✅ Works in light mode

---

## Bug #2: No Indication of Unavailability in Assignor View

### Problem
When a referee marks themselves as unavailable for a match, the assignor sees no indication of this in the referee picker. This makes it difficult for assignors to know which referees have said they cannot do the match.

**User Request**: 
> "When a referee marks a match or a day as unavailable, there is nothing on the Assignor view to indicate that they are unavailable. It would be good to show that the referee has said they cannot do that match so that the assignor can easily determine that."

### Design Decision
- Show unavailable status clearly with visual indicators
- Still allow assignment (assignor may have communicated outside the app)
- Use red styling to match "unavailable" semantics

### Fix Applied

**File**: `frontend/src/routes/assignor/matches/+page.svelte`

**Changes**:

1. **Added unavailability indicator** (line ~690-730):
```svelte
{#if referee.is_available}
    <span class="availability-badge available-star" title="Marked as available">★</span>
{:else}
    <span class="availability-badge unavailable-x" title="Marked as unavailable">✗</span>
{/if}

<!-- In referee details -->
{#if referee.is_available}
    <span class="available-indicator">Available</span>
{:else}
    <span class="unavailable-indicator">Said Unavailable</span>
{/if}
```

2. **Added CSS styling**:
```css
.availability-badge.available-star {
    color: #22c55e; /* Green */
}

.availability-badge.unavailable-x {
    color: #ef4444; /* Red */
}

.unavailable-indicator {
    background-color: #ef4444;
    color: white;
    padding: 0.2rem 0.6rem;
    border-radius: 0.25rem;
    font-size: 0.8rem;
    font-weight: 600;
}

.referee-item.marked-unavailable {
    background-color: #fef2f2;
    border-color: #fca5a5;
}

.referee-item.marked-unavailable:hover {
    border-color: #ef4444;
    background-color: #fee2e2;
}
```

### Result
✅ Red ✗ icon appears for unavailable referees
✅ "Said Unavailable" badge shows in referee details
✅ Entire referee card has red background/border
✅ Assignor can still assign them if needed
✅ Clear visual distinction between available (★ green) and unavailable (✗ red)

### Visual Indicators

**Available Referee**:
- Green star (★)
- Green badge: "Available"
- Green background and border
- Appears at top of eligible list

**Unavailable Referee**:
- Red X (✗)
- Red badge: "Said Unavailable"
- Light red background and border
- Still clickable/assignable

---

## Feature: Optional Assistant Referees for U10 Matches

### Problem
**User Request**:
> "I would like to be able to have the option to assign ARs to U10 matches. They should be 100% optional and the lack of assignment of ARs should not make a U10s match look like it is only partially assigned."

### Current Behavior
- U10 matches only get a center referee slot by default ✓
- Assignors can manually add AR slots if needed ✓
- **Problem**: If AR slots are added but not filled, match shows as "partial"

### Fix Applied

**File**: `backend/matches.go`

**Modified**: `getMatchRoles()` function (line ~453-527)

**Changes**:
1. Fetch the match's age group
2. Detect if match is U10 or younger
3. Exclude AR slots from assignment status calculation for U10 matches
4. Still return AR slots in the response (so they can be assigned)

**Code**:
```go
// First, get the match's age group to determine if ARs are optional
var ageGroup sql.NullString
err := db.QueryRow("SELECT age_group FROM matches WHERE id = $1", matchID).Scan(&ageGroup)
if err != nil {
    return []MatchRole{}, "unassigned"
}

// ... (query execution)

// Determine if this is a U10 match (ARs are optional)
isU10OrYounger := false
if ageGroup.Valid {
    ageStr := strings.TrimPrefix(ageGroup.String, "U")
    age, err := strconv.Atoi(ageStr)
    if err == nil && age <= 10 {
        isU10OrYounger = true
    }
}

for rows.Next() {
    // ... (scan role data)

    roles = append(roles, role)

    // For U10 and younger, only count center referee toward assignment status
    // ARs are optional and don't affect whether match is "full" or "partial"
    if isU10OrYounger && (role.RoleType == "assistant_1" || role.RoleType == "assistant_2") {
        // Don't count AR slots toward total for U10
        continue
    }

    totalSlots++
    if role.AssignedRefereeID != nil {
        assignedSlots++
    }
}
```

### Result

**U10 Match Status Examples**:

| Center | AR1 | AR2 | Status | Explanation |
|--------|-----|-----|--------|-------------|
| ✓ Assigned | — | — | **Full** | Center filled, ARs don't count |
| ✓ Assigned | ✓ Assigned | — | **Full** | Center filled, ARs optional |
| ✓ Assigned | ✓ Assigned | ✓ Assigned | **Full** | All filled |
| — | — | — | **Unassigned** | No center assigned |
| — | ✓ Assigned | ✓ Assigned | **Unassigned** | Center not filled |

**U12+ Match Status** (unchanged):

| Center | AR1 | AR2 | Status |
|--------|-----|-----|--------|
| ✓ | ✓ | ✓ | **Full** |
| ✓ | ✓ | — | **Partial** |
| ✓ | — | — | **Partial** |
| — | — | — | **Unassigned** |

✅ U10 matches show as "full" when center is assigned
✅ ARs can still be added and assigned
✅ Unfilled AR slots don't make match look partial
✅ U12+ matches unchanged (ARs still required)

---

## Files Changed

### Frontend
1. **`frontend/src/routes/referee/matches/+page.svelte`**
   - Added default color to `.btn-clear` CSS class
   - Line ~788-792

2. **`frontend/src/routes/assignor/matches/+page.svelte`**
   - Added unavailability indicators to referee picker
   - Modified referee item template (lines ~690-730)
   - Added CSS for unavailable styling (lines ~1670-1710)

### Backend
1. **`backend/matches.go`**
   - Modified `getMatchRoles()` function
   - Added age group detection for optional AR logic
   - Lines ~453-527

---

## Testing Instructions

### Test #1: Clear Preference Button Visibility

1. Sign in as referee
2. Go to "My Matches"
3. Find a match
4. Click **✓** button (mark available)
5. **Verify**: All three buttons (✓ ✗ —) are visible
6. Click **✗** button (mark unavailable)
7. **Verify**: All three buttons (✓ ✗ —) are visible
8. **Expected**: The — button should always be visible with gray color

### Test #2: Unavailability Indication in Assignor View

1. Sign in as referee
2. Go to "My Matches"
3. Click **✗** button on a match (mark unavailable)
4. Sign out, sign in as assignor
5. Go to "Match Schedule"
6. Click "Assign Referees" on that match
7. Click "Select Referee" for any role
8. **Expected**:
   - Referee appears in eligible list (if eligible)
   - Has red **✗** icon instead of green **★**
   - Shows red badge: "Said Unavailable"
   - Card has light red background/border
   - Can still click to assign

9. Sign back in as referee
10. Click **✓** button on same match (mark available)
11. Sign back in as assignor
12. Reopen referee picker
13. **Expected**:
    - Referee now has green **★** icon
    - Shows green badge: "Available"
    - Card has green background/border
    - Appears at top of list

### Test #3: Optional ARs for U10 Matches

**Setup**: Create or find a U10 match

1. Sign in as assignor
2. Go to "Match Schedule"
3. Find a U10 match
4. **Verify**: Match shows as "Unassigned" (red badge)
5. Click "Assign Referees"
6. Assign a center referee
7. Go back to Match Schedule
8. **Expected**: Match shows as "Full" (green badge)

9. Click "Assign Referees" again on the same match
10. Note: You may need to manually add AR slots (not covered in this fix)
11. If AR slots exist, verify they show but are empty
12. Go back to Match Schedule
13. **Expected**: Match still shows as "Full" (not partial)

14. Assign an AR1
15. **Expected**: Still shows as "Full"

16. Assign AR2
17. **Expected**: Still shows as "Full"

**Compare with U12 match**:
1. Find a U12 match
2. Assign center referee only
3. **Expected**: Shows as "Partial" (yellow badge)
4. Assign both ARs
5. **Expected**: Shows as "Full" (green badge)

---

## Deployment

### Applied
1. ✅ Frontend rebuilt (with CSS and template fixes)
2. ✅ Frontend restarted
3. ✅ Backend code updated (U10 AR logic)
4. ✅ Backend restarted

### Verification
```bash
docker-compose ps

# Expected output:
# backend:  Up X seconds
# frontend: Up X seconds  
# db:       Up X minutes (healthy)
```

---

## Impact

### Before Fixes
- ❌ Clear preference button invisible in light mode
- ❌ Assignors can't see which referees marked unavailable
- ❌ U10 matches with only center referee show as "partial"
- ❌ Confusion about match assignment status

### After Fixes
- ✅ Clear preference button always visible
- ✅ Assignors see clear unavailability indicators
- ✅ Can still assign unavailable referees if needed
- ✅ U10 matches correctly show as "full" with only center
- ✅ ARs can be optionally added to U10 matches
- ✅ U12+ matches unchanged (ARs still required)

---

## Browser Cache

If you don't see the changes:
1. **Hard refresh**: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac)
2. **Clear cache**: Browser settings → Clear browsing data → Cached images and files
3. **Incognito/Private window**: Test in a new private browsing window

---

## Summary

**Three Issues Resolved**:

1. ✅ **Clear preference button visible** - Fixed CSS color for non-active state
2. ✅ **Unavailability shown to assignors** - Red indicators (✗, badge, background) for unavailable referees
3. ✅ **Optional ARs for U10** - Matches show as "full" when center is assigned, regardless of AR status

**All changes deployed and ready for testing!** 🎉

The application now has:
- Clear visual distinction between available/unavailable referees
- Proper assignment status for U10 matches
- Improved UX for both referees and assignors
- Visible tri-state availability buttons in all states
