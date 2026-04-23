# Migration 006 Fix Summary

**Date**: 2026-04-22  
**Status**: ✅ **RESOLVED**  
**Migration**: 006_tristate_availability

---

## What Happened

### The Error
The migration 006 partially ran and left the database in a "dirty" state:
```
Failed to run migrations: Dirty database version 6. Fix and force version.
```

### Root Cause
When the migration first ran:
1. ✅ Successfully added the `available` column to the `availability` table
2. ❌ Failed before creating the `idx_availability_status` index
3. ❌ Migration was marked as "dirty" (incomplete)

The golang-migrate library refuses to run any migrations when one is marked as dirty, requiring manual intervention.

---

## How It Was Fixed

### Step 1: Diagnose the Issue
```sql
-- Checked the migration state
SELECT * FROM schema_migrations;
-- Result: version 6, dirty = true

-- Checked what was applied
\d availability
-- Result: 'available' column exists but with wrong default (false instead of true)
-- Result: 'idx_availability_status' index does NOT exist
```

### Step 2: Complete the Migration Manually
```sql
-- Fix the default value
ALTER TABLE availability ALTER COLUMN available SET DEFAULT true;

-- Update existing records (none existed, so 0 rows affected)
UPDATE availability SET available = true;

-- Create the missing index
CREATE INDEX IF NOT EXISTS idx_availability_status ON availability(referee_id, available);
```

### Step 3: Mark Migration as Clean
```sql
-- Clear the dirty flag
UPDATE schema_migrations SET dirty = false WHERE version = 6;
```

### Step 4: Verify and Restart
```bash
# Restart the backend
docker-compose restart backend

# Confirmed successful startup:
# "Migrations completed successfully"
# "Server starting on port 8080"
```

---

## Current State

### ✅ Database Schema (Correct)
```
Table "public.availability"
   Column   |  Type   | Default  
------------+---------+----------
 id         | bigint  | ...
 match_id   | bigint  | 
 referee_id | bigint  | 
 available  | boolean | true     ← Correct!
 created_at | timestamp | now()
 updated_at | timestamp | now()

Indexes:
 - availability_pkey (PRIMARY KEY)
 - availability_match_id_referee_id_key (UNIQUE)
 - idx_availability_match_id
 - idx_availability_referee_id
 - idx_availability_status              ← Fixed!
```

### ✅ Migration State (Clean)
```
 version | dirty 
---------+-------
       6 | f       ← Clean!
```

### ✅ Backend Status
```
✅ Database connection established
✅ Migrations completed successfully
✅ Server starting on port 8080
```

---

## Why This Happened

**Likely causes:**
1. The migration ran during a restart while the database was busy
2. A timeout or connection issue during the migration
3. The migration file had a syntax issue initially (now fixed)

**Why golang-migrate marks it dirty:**
- If a migration fails partway through, the database could be in an inconsistent state
- The "dirty" flag prevents further migrations until you manually verify and fix the issue
- This is a safety feature to prevent cascading failures

---

## Prevention for Future

### For Development
When writing migrations:
1. Test them on a development database first
2. Keep migrations small and atomic
3. Use transactions where possible (though golang-migrate doesn't support this for all DDL)

### If This Happens Again
If you see "Dirty database version X":

```bash
# 1. Check what was applied
docker exec referee-scheduler-db psql -U referee_scheduler -c "\d table_name"

# 2. Check migration state
docker exec referee-scheduler-db psql -U referee_scheduler -c "SELECT * FROM schema_migrations;"

# 3. Manually complete the migration
# (review the .up.sql file and run missing statements)

# 4. Mark as clean
docker exec referee-scheduler-db psql -U referee_scheduler -c "UPDATE schema_migrations SET dirty = false WHERE version = X;"

# 5. Restart backend
docker-compose restart backend
```

---

## Impact Assessment

### ✅ No Data Loss
- Zero existing availability records in the database
- No data was corrupted or lost

### ✅ No User Impact
- Issue occurred during development
- No users were affected
- Feature not yet in production

### ✅ Migration Now Complete
- All schema changes applied correctly
- Indexes created
- Default values set correctly
- Backend running normally

---

## Next Steps

### Immediate
1. ✅ Backend is running - **No action needed**
2. ✅ Migration is complete - **No action needed**
3. ⏩ Proceed with testing the tri-state availability feature

### Testing
```bash
# Restart frontend to see UI changes
docker-compose restart frontend

# Test the feature as documented in IMPLEMENTATION_COMPLETE.md
```

---

## Files Involved

**Migration Files:**
- `backend/migrations/006_tristate_availability.up.sql` (applied successfully)
- `backend/migrations/006_tristate_availability.down.sql` (not needed)

**Database:**
- Table: `availability` (modified)
- Schema version: 6 (clean)

---

**Status: ✅ FIXED AND VERIFIED**

The backend is now running successfully with migration 006 applied. You can proceed with testing the tri-state availability feature!
