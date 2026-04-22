# Testing Guide - Referee Scheduler

## Quick Testing Steps

### 1. Testing Referee Availability Marking

**Prerequisites:**
- Be signed in as a referee (matthew.sheeley@gmail.com) or assignor who has filled out their referee profile
- Have matches in the database (you already have 10+ matches)
- Have a complete profile (first name, last name, date of birth)

**Steps:**
1. Go to http://localhost:3000
2. Sign in with Google (if not already signed in)
3. From the dashboard, click **"View Matches & Mark Availability"** button
4. You should see a list of matches grouped by date
5. Click the **"Mark Available"** button on any match
6. The button should change to **"✓ Available"** with a green checkmark
7. The card border should turn green
8. Refresh the page - your availability should persist

**Troubleshooting Availability:**
- If you see "Complete Your Profile" message:
  - Go to `/referee/profile`
  - Fill in: First Name, Last Name, Date of Birth
  - Check "Certified" if applicable (needed for U12+ center referee)
  - Set Certification Expiry (future date)
  - Click "Save Profile"
  - Return to `/referee/matches`

- If you see "No upcoming matches":
  - Check that matches exist: View as assignor at `/assignor/matches`
  - Matches must be in the future (match_date >= today)
  - Your age/certification must meet eligibility requirements

---

### 2. Testing Assignment Interface

**Prerequisites:**
- Be signed in as assignor (msheeley@jackhenry.com)
- Have matches imported
- Have active referees with complete profiles

**Steps:**
1. Go to http://localhost:3000
2. Sign in as assignor
3. Go to **Manage Match Schedule** or navigate to `/assignor/matches`
4. Find any match in the list
5. Click the **"Assign Referees"** button (should be blue, not grayed out)
6. Assignment panel opens showing:
   - Match details at top
   - Role cards (CR, AR1, AR2 depending on age group)
7. Click **"Select Referee"** on Center Referee role
8. Referee picker opens showing eligible referees
9. Click a referee name to assign them
10. Panel closes automatically
11. Match status badge updates to "Partial" or "Full"

**Testing Change/Remove:**
1. Re-open assignment panel on same match
2. See referee name in the role card
3. Click **"Change"** button to pick different referee
4. OR click **"Remove"** button to clear assignment
5. Confirm the dialog
6. Assignment updates

**Testing Conflict Detection:**
1. Assign a referee to Match 1 (e.g., 9:00 AM - 10:00 AM on 4/25)
2. Try to assign same referee to Match 2 (e.g., 9:30 AM - 11:00 AM on 4/25)
3. Dialog appears: "Referee is already assigned to another match at this time. Assign anyway?"
4. Click "Cancel" - assignment doesn't proceed
5. Try again, click "OK" - assignment proceeds with conflict

---

## Current Database State

**Users:**
- User 1 (assignor): msheeley@jackhenry.com - Mike Sierra - DOB: 2013-08-17
- User 2 (referee): matthew.sheeley@gmail.com - Matthew Sheeley - DOB: 1978-12-08

**Matches:**
- 10+ matches on 2026-04-25 (all active)
- Age groups: U8, U10, U12
- All have role slots created

---

## Common Issues & Solutions

### Issue: "Assign Referees" button is grayed out
**Solution:** This happens for cancelled matches. Make sure match status is "active".

### Issue: No matches appear for referee
**Possible causes:**
1. Profile incomplete (missing first name, last name, or DOB)
2. Not eligible for any matches due to age/certification
3. All matches are in the past
4. No matches in database

**Check:**
```sql
-- From database:
docker exec referee-scheduler-db psql -U referee_scheduler -c "SELECT first_name, last_name, date_of_birth, certified FROM users WHERE id = 2;"
```

### Issue: No referees appear in assignment picker
**Possible causes:**
1. No active referees in database
2. No referees with complete profiles
3. Eligibility rules filtering everyone out

**Check:**
```sql
-- From database:
docker exec referee-scheduler-db psql -U referee_scheduler -c "SELECT id, email, first_name, last_name, date_of_birth, certified, status FROM users WHERE role IN ('referee', 'assignor');"
```

### Issue: Assignment doesn't save
**Check browser console for errors:**
1. Press F12 in browser
2. Go to Console tab
3. Try assignment again
4. Look for red error messages

**Check backend logs:**
```bash
docker-compose logs backend --tail=50
```

---

## API Testing (for debugging)

### Test Referee Matches Endpoint
```bash
# Get matches for current user (requires session cookie)
curl http://localhost:8080/api/referee/matches \
  --cookie "your-session-cookie" \
  -H "Content-Type: application/json"
```

### Test Eligible Referees Endpoint
```bash
# Get eligible referees for match 35, center role
curl "http://localhost:8080/api/matches/35/eligible-referees?role=center" \
  --cookie "your-session-cookie"
```

### Test Assignment Endpoint
```bash
# Assign referee 2 to match 35, center role
curl -X POST http://localhost:8080/api/matches/35/roles/center/assign \
  --cookie "your-session-cookie" \
  -H "Content-Type: application/json" \
  -d '{"referee_id": 2}'
```

---

## Verifying Data in Database

### Check if referee has complete profile
```bash
docker exec referee-scheduler-db psql -U referee_scheduler -c "
SELECT id, email, first_name, last_name, date_of_birth, certified, cert_expiry 
FROM users 
WHERE email = 'matthew.sheeley@gmail.com';"
```

### Check matches
```bash
docker exec referee-scheduler-db psql -U referee_scheduler -c "
SELECT id, event_name, age_group, match_date, status 
FROM matches 
WHERE match_date >= CURRENT_DATE 
ORDER BY match_date 
LIMIT 5;"
```

### Check role slots
```bash
docker exec referee-scheduler-db psql -U referee_scheduler -c "
SELECT m.id, m.event_name, mr.role_type, mr.assigned_referee_id, u.name
FROM matches m
JOIN match_roles mr ON mr.match_id = m.id
LEFT JOIN users u ON u.id = mr.assigned_referee_id
WHERE m.id = 35;"
```

### Check availability
```bash
docker exec referee-scheduler-db psql -U referee_scheduler -c "
SELECT a.match_id, m.event_name, a.referee_id, u.name, a.created_at
FROM availability a
JOIN matches m ON m.id = a.match_id
JOIN users u ON u.id = a.referee_id
ORDER BY a.created_at DESC
LIMIT 10;"
```

### Check assignments
```bash
docker exec referee-scheduler-db psql -U referee_scheduler -c "
SELECT 
  m.event_name,
  mr.role_type,
  u.name as referee_name
FROM matches m
JOIN match_roles mr ON mr.match_id = m.id
LEFT JOIN users u ON u.id = mr.assigned_referee_id
WHERE mr.assigned_referee_id IS NOT NULL
ORDER BY m.match_date;"
```

---

## Expected Behavior

### Referee Availability Page (`/referee/matches`)
**Should show:**
- "My Assignments" section (if assigned to any matches)
- "Available Matches" section grouped by date
- Each match card shows:
  - Event name and age group badge
  - Date, time, location, field
  - "Mark Available" button (or "✓ Available" if already marked)
  - Eligible roles listed at bottom

**Should NOT show:**
- Cancelled matches
- Past matches
- Matches the referee is not eligible for

### Assignment Panel (`/assignor/matches`)
**Should show:**
- Modal with match details at top
- 1-3 role cards depending on age group:
  - U6/U8: 1 CR only
  - U10: 1 CR only (can manually add ARs later)
  - U12+: 1 CR + 2 ARs
- Each role card shows:
  - Role name
  - "Assigned" (green) or "Open" (red) badge
  - Referee name if assigned
  - "Select Referee", "Change", or "Remove" buttons

### Referee Picker
**Should show:**
- "Eligible Referees" section with clickable referee items
- "Ineligible Referees" section with grayed-out items showing reasons
- For each referee:
  - Full name
  - Grade badge (if set)
  - Age at match date
  - Cert badge for center roles
  - Ineligible reason in red italic (if ineligible)

---

## Success Criteria

✅ **Referee can:**
- See eligible upcoming matches
- Mark availability with one click
- See confirmation (green border, checkmark)
- Availability persists after refresh

✅ **Assignor can:**
- Open assignment panel for any match
- See all role slots with current status
- Click role to see eligible referees
- Assign referee with one click
- Change or remove assignments
- See conflict warning for double-booking
- See updated match status after assignment

---

## Next Steps if Issues Persist

1. **Clear browser cache and cookies**
   - Hard refresh: Ctrl+Shift+R (Windows) or Cmd+Shift+R (Mac)

2. **Check browser console for JavaScript errors**
   - Press F12, go to Console tab

3. **Verify you're on the latest code**
   ```bash
   docker-compose down
   docker-compose up --build -d
   ```

4. **Test with a different browser**
   - Try Chrome, Firefox, or Safari

5. **Check if ports are accessible**
   ```bash
   curl http://localhost:3000  # Frontend
   curl http://localhost:8080/health  # Backend
   ```

6. **Review backend logs for API errors**
   ```bash
   docker-compose logs backend --tail=100
   ```
