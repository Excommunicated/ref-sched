# Critical Fix: Assignments on Unavailable Days Now Visible

**Date**: 2026-04-22  
**Priority**: 🚨 **CRITICAL**  
**Status**: ✅ **FIXED AND DEPLOYED**

---

## Problem

### Symptom

If a referee marked a day as unavailable, but an assignor then assigned them to a match on that day anyway, **the referee would never see the assignment**.

### User Flow (Before Fix)

1. Referee marks Tuesday as unavailable
2. All Tuesday matches disappear from "My Matches"
3. Assignor calls referee, confirms they can actually do one match on Tuesday
4. Assignor assigns referee to that match
5. **BUG**: Referee still can't see the match (entire day hidden)
6. Referee doesn't acknowledge assignment
7. Referee doesn't show up to the match 🚨

### Why This Is Critical

- **Missed matches**: Referees don't know they've been assigned
- **No acknowledgment**: System shows assignment as "overdue"
- **Communication breakdown**: Assignor thinks referee knows, referee has no idea
- **Match coverage**: Could result in matches without officials

---

## Root Cause

### The Buggy Query

**File**: `backend/availability.go` (Line 82-95)

**Before (Buggy)**:
```sql
SELECT m.id, m.event_name, m.team_name, ...
FROM matches m
WHERE m.match_date >= CURRENT_DATE
  AND m.status = 'active'
  AND NOT EXISTS (
    SELECT 1 FROM day_unavailability du
    WHERE du.referee_id = $1 
      AND du.unavailable_date = m.match_date
  )
```

### The Issue

This query **excludes ALL matches on unavailable days**, even if the referee has been assigned to one of them.

The `NOT EXISTS` clause says: "Don't show any match on a day marked unavailable."

**Problem**: This hides assignments too!

---

## Solution

### Modified Query Logic

**After (Fixed)**:
```sql
SELECT m.id, m.event_name, m.team_name, ...
FROM matches m
WHERE m.match_date >= CURRENT_DATE
  AND m.status = 'active'
  AND (
    -- Either the day is not marked unavailable
    NOT EXISTS (
      SELECT 1 FROM day_unavailability du
      WHERE du.referee_id = $1 
        AND du.unavailable_date = m.match_date
    )
    OR
    -- OR the referee is assigned to this match (always show assignments)
    EXISTS (
      SELECT 1 FROM match_roles mr
      WHERE mr.match_id = m.id 
        AND mr.assigned_referee_id = $1
    )
  )
```

### New Logic

**Show a match if**:
- The day is NOT marked unavailable, OR
- The referee is assigned to that match (regardless of day unavailability)

**Result**: Assigned matches are ALWAYS visible, even on unavailable days.

---

## Visual Warning Added

When a referee is assigned to a match on a day they marked unavailable, a prominent warning banner appears:

### Warning Banner

```
┌─────────────────────────────────────────────────────┐
│ ⚠️  Assigned on Unavailable Day                     │
│                                                     │
│ You marked this day as unavailable, but you've      │
│ been assigned to this match. Please contact the     │
│ assignor if this is an error.                       │
└─────────────────────────────────────────────────────┘
```

**Styling**:
- Yellow/amber background (#fef3c7)
- Orange left border (#f59e0b)
- Orange card border
- Clear warning icon and message
- Appears at top of assigned match card

---

## Files Changed

### Backend

**File**: `backend/availability.go`

**Lines**: 82-103 (query modification)

**Change**: Modified `WHERE` clause to always include assigned matches

### Frontend

**File**: `frontend/src/routes/referee/matches/+page.svelte`

**Changes**:
1. **Line ~326**: Added conditional class for warning state
2. **Line ~329-338**: Added warning banner template
3. **Line ~755-789**: Added CSS styling for warning

---

## Example Scenarios

### Scenario 1: Assigned on Unavailable Day

**Setup**:
- Referee marks Saturday as unavailable
- Assignor assigns referee to Saturday 3:00 PM match (after phone call)

**Before Fix**:
- ❌ Referee sees no matches on Saturday
- ❌ Assignment invisible
- ❌ No acknowledgment possible

**After Fix**:
- ✅ Referee sees Saturday 3:00 PM match
- ✅ Big yellow warning banner at top
- ✅ Can acknowledge assignment
- ✅ "Mark Entire Day Unavailable" button still visible

### Scenario 2: Not Assigned, Day Unavailable

**Setup**:
- Referee marks Sunday as unavailable
- Has no assignments on Sunday

**Before Fix**:
- ✓ All Sunday matches hidden

**After Fix**:
- ✓ All Sunday matches still hidden (correct behavior)

### Scenario 3: Multiple Matches, One Assigned

**Setup**:
- Referee marks Monday as unavailable
- Monday has 5 matches
- Referee is assigned to 1 of the 5 matches

**Before Fix**:
- ❌ All 5 matches hidden (including the assignment)

**After Fix**:
- ✅ The 1 assigned match is visible (with warning)
- ✓ The other 4 matches remain hidden

---

## Testing Instructions

### Test Case 1: Assignment on Unavailable Day

**Prerequisites**: Need a match in the future

1. Sign in as referee
2. Go to "My Matches"
3. Choose a date with matches
4. Click "Mark Entire Day Unavailable"
5. **Verify**: All matches for that day disappear
6. **Verify**: Date header stays visible with red toggle button

7. Sign out, sign in as assignor
8. Go to "Match Schedule"
9. Find a match on that unavailable day
10. Click "Assign Referees"
11. Assign the referee to Center Referee role
12. Click the referee's name in the picker

13. Sign back in as referee
14. Go to "My Matches"
15. **Expected**: 
    - Date is still there (with unavailable day button)
    - The assigned match is NOW VISIBLE
    - Big yellow warning banner at top of card:
      - "⚠️ Assigned on Unavailable Day"
      - Warning message about contacting assignor
    - Match card has orange border
16. Scroll to "My Assignments" section
17. **Expected**: Match appears there too with warning

### Test Case 2: Clear Day Unavailability

1. With the assigned match visible (from test case 1)
2. Click "Day Marked Unavailable - Click to Clear"
3. **Expected**: 
    - Warning banner disappears
    - Orange border disappears
    - Match displays normally
    - Other matches on that day reappear

### Test Case 3: Not Assigned, Day Unavailable

1. Sign in as referee
2. Mark a day unavailable that has NO assignments
3. **Expected**: All matches on that day hidden (correct)
4. **Expected**: Date header visible with unavailable button

---

## Database Queries

### Check for Assignment
```sql
EXISTS (
  SELECT 1 
  FROM match_roles mr
  WHERE mr.match_id = m.id 
    AND mr.assigned_referee_id = $1
)
```

### Check for Day Unavailability
```sql
NOT EXISTS (
  SELECT 1 
  FROM day_unavailability du
  WHERE du.referee_id = $1 
    AND du.unavailable_date = m.match_date
)
```

### Combined Logic (OR)
```sql
(NOT EXISTS day_unavailability)
OR
(EXISTS assignment)
```

---

## Impact

### Before Fix

- 🚨 **Critical Bug**: Assignments could be completely invisible
- ❌ Referees miss matches
- ❌ No acknowledgment possible
- ❌ Assignors frustrated
- ❌ Potential no-shows at matches

### After Fix

- ✅ Assignments ALWAYS visible
- ✅ Clear warning when assigned on unavailable day
- ✅ Referees can acknowledge
- ✅ Communication maintained
- ✅ System integrity preserved

---

## Edge Cases Handled

✅ **Assigned to multiple matches on unavailable day**: All assignments visible with warnings  
✅ **Clear day unavailability**: Warnings disappear, shows normally  
✅ **Unassigned matches on unavailable day**: Still hidden (correct)  
✅ **Assignment added after day marked unavailable**: Immediately visible  
✅ **Day marked unavailable after assignment**: Assignment stays visible with warning  

---

## User Experience

### Referee View

**Before**:
```
Tuesday (Unavailable)
┌────────────────────────────────────┐
│ Day is marked unavailable          │
│ (Hidden: 1 assigned match!)        │
└────────────────────────────────────┘
```

**After**:
```
Tuesday (Unavailable)

┌────────────────────────────────────┐
│ ⚠️ Assigned on Unavailable Day     │
│ Contact assignor if this is error  │
├────────────────────────────────────┤
│ Spring Tournament                  │
│ U14 Girls - Center Referee         │
│ 3:00 PM - Field 2                  │
│                                    │
│ [Acknowledge Assignment]           │
└────────────────────────────────────┘
```

---

## Related Features

This fix integrates with:
- ✅ Day-level unavailability (Epic 4.1)
- ✅ Match-level tri-state availability (Epic 4.2)
- ✅ Assignment acknowledgment system (Epic 5)
- ✅ Assignor assignment panel (Epic 3)

---

## Prevention

### Code Review Checklist

When implementing filters that hide data:
- [ ] Consider if filtered items might be needed for other reasons
- [ ] Use OR logic to preserve critical items (assignments, etc.)
- [ ] Add visual indicators when showing exceptions
- [ ] Test with data that should be hidden BUT also needed
- [ ] Document the exception logic clearly

### Testing Checklist

- [ ] Test filter with no matching items
- [ ] Test filter with matching items
- [ ] Test filter with items that SHOULD show despite filter
- [ ] Test changing filter state dynamically
- [ ] Test acknowledgment/interaction with filtered items

---

## Deployment

### Applied
1. ✅ Backend query updated
2. ✅ Backend restarted
3. ✅ Frontend warning banner added
4. ✅ Frontend rebuilt and restarted

### Verification
```bash
docker-compose ps

# All services should be "Up"
```

---

## Browser Cache

If you don't see the changes:
1. **Hard refresh**: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac)
2. **Clear cache**: Browser settings → Clear browsing data
3. **Incognito window**: Test in private browsing mode

---

## Summary

**The Critical Bug**:
Assignments on unavailable days were completely invisible to referees.

**The Fix**:
Modified query to ALWAYS show assignments, even on unavailable days.

**The Enhancement**:
Added clear warning banner when assigned on unavailable day.

**The Result**:
- Assignments never hidden
- Clear communication
- System integrity maintained
- No missed matches

---

**Status**: ✅ **FIXED AND DEPLOYED**

This critical bug has been resolved. Referees will now always see their assignments, with clear warnings when assigned to days they marked unavailable.

🎉 **Ready for testing!**

Follow Test Case 1 above to verify the fix works correctly.
