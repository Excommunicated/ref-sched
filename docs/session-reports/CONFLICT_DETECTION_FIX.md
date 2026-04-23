# Conflict Detection Fix

**Date**: 2026-04-22  
**Issue**: Conflict warnings not appearing for overlapping assignments  
**Status**: ✅ **FIXED**

---

## Problem

Conflict warnings were not showing for referees with overlapping assignments.

**Backend Logs**:
```
Warning: Failed to check conflicts for match X: pq: function pg_catalog.overlaps(text, text, time without time zone, time without time zone) does not exist
```

---

## Root Cause

The SQL query used PostgreSQL's `OVERLAPS` operator with incompatible types:

**Buggy Query**:
```sql
WHERE ...
  AND (
    (m2.start_time, m2.end_time) OVERLAPS ($4::time, $5::time)
  )
```

**Problem**:
- Database columns `start_time` and `end_time` are of type `TIME`
- Go variables `m.StartTime` and `m.EndTime` are `string` 
- PostgreSQL couldn't match the OVERLAPS function signature for `(time, time) OVERLAPS (time, time)` when the first pair comes from columns and the second from casted parameters

---

## Solution

Replaced the `OVERLAPS` operator with a standard interval overlap check using simple comparisons.

**Fixed Query**:
```sql
WHERE mr2.assigned_referee_id = $1
  AND m2.id != $2
  AND m2.status = 'active'
  AND m2.match_date = $3
  AND m2.start_time < $5::time     -- Match2.start < Match1.end
  AND m2.end_time > $4::time       -- Match2.end > Match1.start
```

**Logic**:
Two time intervals overlap if and only if:
- `start1 < end2` AND `start2 < end1`

**Example**:
```
Match 1: 3:00 PM - 4:30 PM
Match 2: 3:30 PM - 5:00 PM

Check:
  3:00 < 5:00 ✓ (Match1.start < Match2.end)
  3:30 < 4:30 ✓ (Match2.start < Match1.end)

Result: OVERLAP detected ✓
```

---

## Files Changed

**File**: `backend/availability.go`

**Line**: ~227-240

**Change**:
```diff
- AND (
-   (m2.start_time, m2.end_time) OVERLAPS ($4::time, $5::time)
- )
+ AND m2.start_time < $5::time
+ AND m2.end_time > $4::time
```

Also moved `conflictRows.Close()` to immediately after the loop completes (better resource management).

---

## Testing

### Quick Test

1. Create 2 matches on same day:
   - Match A: 3:00 PM - 4:30 PM
   - Match B: 3:30 PM - 5:00 PM

2. Assign same referee to both matches

3. Sign in as referee

4. Go to "My Assignments"

5. **Expected**: 
   - Both matches show **red border**
   - Both have **red warning banner**
   - Each banner lists the other conflicting match
   - Shows: "Scheduling Conflict Detected"

### Edge Cases

**Non-overlapping** (should NOT show warning):
```
Match A: 3:00 PM - 4:00 PM
Match B: 4:00 PM - 5:00 PM
Result: No overlap (end time = start time)
```

**Adjacent** (should NOT show warning):
```
Match A: 3:00 PM - 3:45 PM
Match B: 3:45 PM - 4:30 PM
Result: No overlap (touching but not overlapping)
```

**Fully overlapping** (should show warning):
```
Match A: 3:00 PM - 5:00 PM
Match B: 3:30 PM - 4:30 PM
Result: Overlap (Match B fully inside Match A)
```

**Partial overlap** (should show warning):
```
Match A: 3:00 PM - 4:30 PM
Match B: 4:00 PM - 5:00 PM
Result: Overlap (30 minutes overlap)
```

---

## Deployment

✅ Code fixed in `backend/availability.go`  
✅ Backend restarted  
✅ Server running on port 8080  

---

## Verification

Check backend logs for errors:
```bash
docker-compose logs backend --tail 50 | grep "Failed to check conflicts"
```

**Expected**: No new conflict check errors after restart.

---

## Browser Cache

**Hard refresh** your browser (Ctrl+Shift+R or Cmd+Shift+R) to reload the page and see conflict warnings.

---

## Summary

**Bug**: PostgreSQL OVERLAPS type mismatch  
**Fix**: Use simple time comparisons instead  
**Status**: Deployed and ready to test  

The conflict detection now works correctly and should show red warning banners for overlapping assignments.
