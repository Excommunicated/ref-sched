# Testing Instructions - Tri-State Availability & Dashboard

**Date**: 2026-04-22  
**Status**: Ready for Testing

---

## ✅ What Was Fixed

1. **Day-level availability toggle** - Can now change availability even after marking unavailable
2. **Frontend rebuilt** - Latest code is now running in the container

---

## 🧪 Test #1: Day Availability Toggle

### Prerequisites
- Sign in as a referee
- Have complete profile (name, DOB, certification)
- Have at least one upcoming match

### Steps

**1. Navigate to Matches Page**
```
http://localhost:3000 → Sign In → My Matches
```

**2. Mark a Day Unavailable**
- Find a date with matches
- Click gray button: **"Mark Entire Day Unavailable"**
- Click **OK** in confirmation dialog

**Expected Result:**
- ✅ Date header **stays visible** (e.g., "Saturday, April 26, 2026")
- ✅ Button turns **red** with text: "Day Marked Unavailable - Click to Clear"
- ✅ Matches **disappear** (correctly hidden)
- ✅ **Red message box** appears with text:
  > "You have marked this day as unavailable.  
  > Individual matches for this day are hidden.  
  > Click the button above to make yourself available again."

**3. Clear Day Unavailability**
- Click the red button: **"Day Marked Unavailable - Click to Clear"**
- Click **OK** in confirmation dialog

**Expected Result:**
- ✅ Button turns **gray** again: "Mark Entire Day Unavailable"
- ✅ Matches **reappear**
- ✅ All individual match availabilities **cleared** (fresh state - no ✓✗— selected)
- ✅ Message box **disappears**

**4. Refresh Page Test**
- Mark a day unavailable again
- Refresh the browser (F5)

**Expected Result:**
- ✅ Date header **still visible** after refresh
- ✅ Red button **still shows**
- ✅ State **persists** across page reloads

---

## 🧪 Test #2: Individual Match Availability (Tri-State)

### Steps

**1. Find a Match (not assigned)**
- Should have three buttons: **[✓] [✗] [—]**

**2. Mark Available**
- Click **✓** button

**Expected:**
- ✅ Button turns **green**
- ✅ Card border turns **green**
- ✅ Other buttons become inactive

**3. Change to Unavailable**
- Click **✗** button

**Expected:**
- ✅ Button turns **red**
- ✅ Card border turns **red**
- ✅ ✓ button becomes inactive

**4. Clear Preference**
- Click **—** button

**Expected:**
- ✅ Button turns **gray**
- ✅ Card border returns to **default gray**
- ✅ Other buttons inactive

**5. Rapid Changes**
- Click ✓ → ✗ → — → ✓ quickly

**Expected:**
- ✅ Responds to each click
- ✅ Final state reflects last click
- ✅ No errors in console

---

## 🧪 Test #3: Dashboard (Check for Freeze)

### Scenario A: Dashboard with Matches

**Steps:**
1. Go to http://localhost:3000/dashboard
2. Wait for page to load

**Expected:**
- ✅ Page loads successfully
- ✅ Shows "Welcome back, [Name]"
- ✅ Shows quick action cards
- ✅ Shows upcoming matches sections
- ✅ All links are clickable
- ✅ "Sign Out" button works

### Scenario B: Dashboard with NO Matches

**Steps:**
1. Make sure you have no upcoming matches (mark all days unavailable, or wait until there are none)
2. Go to http://localhost:3000/dashboard
3. Wait for page to load

**Expected:**
- ✅ Page loads successfully
- ✅ Shows "Welcome back, [Name]"
- ✅ Shows quick action cards
- ✅ Shows message: "No upcoming matches at this time"
- ✅ Navigation cards are **clickable**
- ✅ "Sign Out" button **works**
- ✅ No freeze or hang

**If Page Freezes:**
1. Open browser console (F12)
2. Check Console tab for errors
3. Check Network tab for hanging requests
4. Take screenshots of both
5. Report what you see

---

## 🧪 Test #4: Day Precedence Rules

### Test: Day Unavailability Overrides Match Availability

**Steps:**
1. Mark individual match as **available** (✓)
2. Mark the **entire day** as unavailable

**Expected:**
- ✅ Day marked unavailable
- ✅ All matches hidden (including the one you marked available)
- ✅ Individual availability was **deleted** (backend behavior)

**Then:**
3. Unmark the day

**Expected:**
- ✅ Matches reappear
- ✅ Previous match availability **NOT restored** (clean slate)
- ✅ All three buttons inactive (no preference)

---

## 🐛 Debugging: If Something Doesn't Work

### Browser Console (F12)

**Open Developer Tools:**
- Chrome/Firefox: Press **F12** or Right-click → Inspect
- Safari: Enable Developer Menu first, then Cmd+Option+I

**Check for Errors:**
1. **Console tab** - Look for red error messages
2. **Network tab** - Look for failed requests (red status codes)
3. **Elements tab** - Verify the button is in the DOM

### Common Issues

**Issue**: Button doesn't appear
- **Check**: Is date header visible?
- **Check**: Browser console for JavaScript errors
- **Fix**: Hard refresh (Ctrl+Shift+R)

**Issue**: Clicking button does nothing
- **Check**: Console for errors
- **Check**: Network tab - is POST request being made?
- **Fix**: Check backend logs: `docker-compose logs backend --tail 50`

**Issue**: Changes don't persist
- **Check**: Network tab - are requests succeeding (200 OK)?
- **Check**: Backend logs for database errors
- **Try**: Sign out and back in

**Issue**: Dashboard freezes
- **Check**: Network tab - is a request hanging?
- **Check**: Console tab - JavaScript errors?
- **Note**: Does hard refresh fix it?

### API Endpoint Testing

**Test Day Unavailability Endpoint:**
```bash
# Get current unavailable days
curl http://localhost:8080/api/referee/day-unavailability \
  --cookie "session=your_cookie" \
  -v

# Mark day unavailable
curl http://localhost:8080/api/referee/day-unavailability/2026-04-26 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"unavailable": true}' \
  --cookie "session=your_cookie" \
  -v

# Clear day unavailability  
curl http://localhost:8080/api/referee/day-unavailability/2026-04-26 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"unavailable": false}' \
  --cookie "session=your_cookie" \
  -v
```

---

## ✅ Success Criteria

All of these should work:

- [ ] Can mark day unavailable
- [ ] Date header stays visible when day unavailable
- [ ] Button changes to red "Clear" button
- [ ] Can click button to clear unavailability
- [ ] Matches reappear after clearing
- [ ] Can mark individual matches (✓ ✗ —)
- [ ] Can change match availability multiple times
- [ ] Dashboard loads without freezing
- [ ] All navigation links work
- [ ] Page refreshes preserve state

---

## 📸 If Reporting Issues

Please include:
1. **Which test** you were running
2. **What you expected** to happen
3. **What actually happened**
4. **Screenshot** of browser console (if applicable)
5. **Screenshot** of the page showing the issue
6. **Any error messages**

---

## 🚀 Quick Start

```bash
# 1. Make sure containers are running
docker-compose ps

# 2. Check backend is healthy
curl http://localhost:8080/health

# 3. Open browser
http://localhost:3000

# 4. Sign in and test!
```

---

**The frontend has been rebuilt and should now work correctly!**

Please test the day availability toggle first, as that was the confirmed fix. If you still see the dashboard freeze, follow the debugging steps and let me know what you find.
