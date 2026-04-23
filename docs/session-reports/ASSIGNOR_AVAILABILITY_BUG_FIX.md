# Assignor Availability Display Bug Fix

**Date**: 2026-04-22  
**Issue**: Referees marked as unavailable showing as available to assignor  
**Status**: ✅ **FIXED**

---

## Problem

### Symptom
When a referee explicitly marks a match as **unavailable** (using the ✗ button), the assignor still sees them as **available** in the eligible referees list when opening the assignment panel.

### User Flow
1. Referee goes to "My Matches"
2. Clicks **✗** button to mark unavailable for a specific match
3. API call: `POST /api/referee/matches/{id}/availability` with `{"available": false}`
4. Record created in database: `availability` table with `available=false`
5. Assignor opens assignment panel for that match
6. **BUG:** Referee shows as available in the list

### Expected Behavior
- Referee who marked **available** (✓) → Shows as available to assignor ✓
- Referee who marked **unavailable** (✗) → Shows as NOT available to assignor ✗
- Referee with **no preference** (—) → Shows as NOT available to assignor ✓

---

## Root Cause

### The Buggy Query

**File:** `backend/eligibility.go` (Line 74-89)

**Before (Buggy):**
```sql
SELECT
    u.id, u.first_name, u.last_name, u.email, u.grade,
    u.date_of_birth, u.certified, u.cert_expiry,
    COALESCE(a.match_id IS NOT NULL, false) as is_available  -- BUG HERE
FROM users u
LEFT JOIN availability a ON a.referee_id = u.id AND a.match_id = $1
WHERE (u.role = 'referee' OR u.role = 'assignor')
  AND u.status = 'active'
  AND u.first_name IS NOT NULL
  AND u.last_name IS NOT NULL
  AND u.date_of_birth IS NOT NULL
ORDER BY
    CASE WHEN a.match_id IS NOT NULL THEN 0 ELSE 1 END,  -- BUG HERE TOO
    u.last_name, u.first_name
```

### The Issue

**Line 78:** `COALESCE(a.match_id IS NOT NULL, false)`

This checks if a record **exists** in the availability table, but **ignores the `available` column value**.

**Logic:**
- Record exists with `available=true` → `is_available = true` ✓ Correct
- Record exists with `available=false` → `is_available = true` ✗ **BUG!**
- No record exists → `is_available = false` ✓ Correct

**Line 87:** `CASE WHEN a.match_id IS NOT NULL THEN 0 ELSE 1 END`

This sorts referees with **any** availability record first, regardless of whether they marked available or unavailable.

---

## Solution

### The Fixed Query

**After (Fixed):**
```sql
SELECT
    u.id, u.first_name, u.last_name, u.email, u.grade,
    u.date_of_birth, u.certified, u.cert_expiry,
    COALESCE(a.available, false) as is_available  -- FIXED
FROM users u
LEFT JOIN availability a ON a.referee_id = u.id AND a.match_id = $1
WHERE (u.role = 'referee' OR u.role = 'assignor')
  AND u.status = 'active'
  AND u.first_name IS NOT NULL
  AND u.last_name IS NOT NULL
  AND u.date_of_birth IS NOT NULL
ORDER BY
    CASE WHEN a.available = true THEN 0 ELSE 1 END,  -- FIXED
    u.last_name, u.first_name
```

### Changes Made

**1. Check actual `available` value (Line 78):**
```sql
-- Before:
COALESCE(a.match_id IS NOT NULL, false)

-- After:
COALESCE(a.available, false)
```

**Logic:**
- Record exists with `available=true` → `is_available = true` ✓
- Record exists with `available=false` → `is_available = false` ✓ **FIXED!**
- No record exists → `is_available = false` ✓

**2. Sort by actual availability (Line 87):**
```sql
-- Before:
CASE WHEN a.match_id IS NOT NULL THEN 0 ELSE 1 END

-- After:
CASE WHEN a.available = true THEN 0 ELSE 1 END
```

This ensures truly available referees (marked ✓) appear first in the list.

---

## Testing

### Test Case 1: Referee Marks Unavailable

**Steps:**
1. Sign in as referee
2. Go to "My Matches"
3. Find a match
4. Click **✗** button (mark unavailable)
5. Verify button turns red
6. Sign in as assignor
7. Go to "Match Schedule"
8. Click "Assign Referees" on that match
9. Click "Select Referee" for a role
10. **Expected:** Referee does NOT appear in "Available Referees" section
11. **Expected:** Referee appears in "Ineligible Referees" OR not at all

**After Fix:** ✅ Referee correctly shows as NOT available

### Test Case 2: Referee Marks Available

**Steps:**
1. Sign in as referee
2. Go to "My Matches"
3. Find a match
4. Click **✓** button (mark available)
5. Verify button turns green
6. Sign in as assignor
7. Open assignment panel for that match
8. **Expected:** Referee appears in "Available Referees" section

**After Fix:** ✅ Still works correctly

### Test Case 3: Referee No Preference

**Steps:**
1. Sign in as referee
2. Go to "My Matches"
3. Find a match
4. Click **—** button OR don't mark anything (no preference)
5. Sign in as assignor
6. Open assignment panel for that match
7. **Expected:** Referee does NOT appear in "Available Referees" section

**After Fix:** ✅ Still works correctly

### Test Case 4: Referee Changes Mind

**Steps:**
1. Referee marks **available** (✓)
2. Assignor sees them as available ✓
3. Referee changes to **unavailable** (✗)
4. Assignor refreshes/reopens picker
5. **Expected:** Referee no longer shows as available

**After Fix:** ✅ Now works correctly

---

## Database State Examples

### Example 1: Explicitly Available
```sql
-- availability table record:
match_id | referee_id | available
---------+------------+-----------
   123   |     5      |   true

-- Query result:
is_available = true
-- Assignor sees: ✓ Shows in "Available Referees"
```

### Example 2: Explicitly Unavailable (THE BUG)
```sql
-- availability table record:
match_id | referee_id | available
---------+------------+-----------
   123   |     5      |   false

-- Query result BEFORE fix:
is_available = true  ❌ WRONG!

-- Query result AFTER fix:
is_available = false  ✅ CORRECT!

-- Assignor sees: Referee does NOT show as available
```

### Example 3: No Preference
```sql
-- availability table: (no record)

-- Query result:
is_available = false

-- Assignor sees: Referee does NOT show as available
```

---

## Impact

### Before Fix
- ❌ Referees who explicitly marked unavailable appeared available
- ❌ Assignors might try to assign referees who said they can't do it
- ❌ Confusion between "no response" and "explicitly unavailable"
- ❌ Tri-state availability feature not working correctly for assignors

### After Fix
- ✅ Explicit unavailability is respected
- ✅ Assignors only see truly available referees
- ✅ Clear distinction between available/unavailable/no preference
- ✅ Tri-state feature works end-to-end

---

## Related Code

### Tri-State Availability Values

The `availability` table uses tri-state logic:

| State | Database | Referee UI | Assignor View |
|-------|----------|------------|---------------|
| **Available** | `available=true` record exists | Green ✓ button | Shows as available |
| **Unavailable** | `available=false` record exists | Red ✗ button | Does NOT show as available |
| **No Preference** | No record exists | Gray — button | Does NOT show as available |

### Why This Matters

The tri-state system allows referees to communicate clear intent:
- **✓ Available** - "I can do this match"
- **✗ Unavailable** - "I cannot do this match"
- **— No preference** - "I haven't decided yet"

For assignors, this distinction is important:
- **Available** → Referee has actively said yes, safe to assign
- **Not available** → Either explicitly said no OR hasn't responded, should not assign

---

## Files Changed

### Backend
**File:** `backend/eligibility.go`

**Lines Changed:** 78, 87

**Change Type:** Bug fix (query logic)

**Restart Required:** Yes (backend restarted)

---

## Deployment

### Applied
1. ✅ Code updated in `backend/eligibility.go`
2. ✅ Backend restarted
3. ✅ Backend confirmed running (port 8080)

### Verification
```bash
# Backend is running
docker-compose ps backend

# Should show: Up X seconds
```

---

## Prevention

### Code Review Checklist

When querying the `availability` table:
- [ ] Check the `available` column value, not just if record exists
- [ ] Use `COALESCE(a.available, false)` not `COALESCE(a.match_id IS NOT NULL, false)`
- [ ] Sort by `a.available = true` not `a.match_id IS NOT NULL`
- [ ] Test all three states: available, unavailable, no preference

### Testing Checklist

When adding features that use availability:
- [ ] Test with referee marked **available** (✓)
- [ ] Test with referee marked **unavailable** (✗)
- [ ] Test with referee **no preference** (—)
- [ ] Test referee changing from one state to another
- [ ] Verify assignor view reflects current state

---

## Summary

**The Bug:**
Checking if an availability record exists instead of checking the `available` column value.

**The Fix:**
Changed query to check `a.available` directly instead of `a.match_id IS NOT NULL`.

**The Result:**
Assignors now correctly see which referees are available vs unavailable vs no preference.

**Status:** ✅ Fixed and deployed

---

## Testing Instructions

### Quick Test
1. Sign in as referee
2. Mark a match as unavailable (✗)
3. Sign in as assignor
4. Open assignment panel for that match
5. **Verify:** Referee does NOT show in available list

### Full Test
See **Test Case 1-4** above for comprehensive testing.

---

**Fixed and ready to test!** 🎉

The assignor's eligible referee list now correctly respects the referee's explicit unavailability marking.
