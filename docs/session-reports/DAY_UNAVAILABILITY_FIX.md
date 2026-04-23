# Day Unavailability Toggle Fix

**Date**: 2026-04-22  
**Issue**: Unable to change day-level availability once marked unavailable  
**Status**: ✅ **FIXED**

---

## Problem

When a referee marked a day as unavailable:
1. ✅ All matches for that day disappeared (correct behavior)
2. ❌ The date header also disappeared
3. ❌ The button to toggle day availability disappeared
4. ❌ **No way to change the day back to available**

This created a frustrating UX where users couldn't undo their choice.

---

## Root Cause

The UI was only showing date headers for dates that had matches in `groupedMatches`. When a day was marked unavailable:
- Backend filtered out matches for that day (they didn't appear in the API response)
- Frontend had no matches for that date in `groupedMatches`
- Date was excluded from `sortedDates` array
- Date header and button were not rendered

**The Logic:**
```typescript
// OLD (problematic):
$: sortedDates = Object.keys(groupedMatches).sort();
// Only includes dates with matches showing
```

---

## Solution

### 1. Track All Dates
Include dates from BOTH match data AND unavailable days:

```typescript
// NEW (fixed):
$: allDates = new Set([
    ...Object.keys(groupedMatches),  // Dates with matches
    ...Array.from(unavailableDays)   // Dates marked unavailable
]);
$: sortedDates = Array.from(allDates).sort();
```

### 2. Show Message for Unavailable Days
When a date is marked unavailable and has no matches:

```svelte
{#if unavailableDays.has(date) && !groupedMatches[date]}
    <!-- Show message instead of matches -->
    <div class="day-unavailable-message">
        <p>You have marked this day as unavailable.</p>
        <p class="small-text">
            Individual matches for this day are hidden. 
            Click the button above to make yourself available again.
        </p>
    </div>
{:else if groupedMatches[date] && groupedMatches[date].length > 0}
    <!-- Show matches normally -->
    <div class="matches-grid">
        ...
    </div>
{/if}
```

### 3. Button Always Visible
The toggle button is now ALWAYS visible because the date header is always rendered:

```svelte
<div class="date-header-row">
    <h3 class="date-header">{formatDate(date)}</h3>
    {#if unavailableDays.has(date)}
        <button class="btn-day-toggle btn-day-unavailable" ...>
            Day Marked Unavailable - Click to Clear
        </button>
    {:else}
        <button class="btn-day-toggle" ...>
            Mark Entire Day Unavailable
        </button>
    {/if}
</div>
```

---

## User Experience (After Fix)

### Marking a Day Unavailable

**Before clicking:**
```
┌─────────────────────────────────────────────────┐
│ Saturday, April 26, 2026  [Mark Day Unavailable]│
├─────────────────────────────────────────────────┤
│ [Match 1 with ✓✗— buttons]                      │
│ [Match 2 with ✓✗— buttons]                      │
│ [Match 3 with ✓✗— buttons]                      │
└─────────────────────────────────────────────────┘
```

**After clicking and confirming:**
```
┌─────────────────────────────────────────────────┐
│ Saturday, April 26   [Day Unavailable - Clear]  │ ← Button stays!
├─────────────────────────────────────────────────┤
│ You have marked this day as unavailable.        │
│ Individual matches are hidden. Click above to   │
│ make yourself available again.                  │
└─────────────────────────────────────────────────┘
   (Red border, red button)
```

**After clicking to clear:**
```
┌─────────────────────────────────────────────────┐
│ Saturday, April 26, 2026  [Mark Day Unavailable]│
├─────────────────────────────────────────────────┤
│ [Match 1 with ✓✗— buttons] (fresh, no selection)│
│ [Match 2 with ✓✗— buttons] (fresh, no selection)│
│ [Match 3 with ✓✗— buttons] (fresh, no selection)│
└─────────────────────────────────────────────────┘
   (Back to normal, individual availability cleared)
```

---

## Visual Design

### Unavailable Day Message Box

**Styling:**
- Light red background (`#fef2f2`)
- Red border (`#ef4444`)
- Centered text
- Clear, friendly messaging
- Prominent enough to notice
- Not alarming (softer red tones)

**CSS:**
```css
.day-unavailable-message {
    background: #fef2f2;
    border: 2px solid #ef4444;
    border-radius: 8px;
    padding: 1.5rem;
    text-align: center;
    margin-top: 1rem;
}
```

---

## Testing

### Manual Test Steps

1. **Mark a day unavailable:**
   - Go to "My Matches"
   - Find a date with matches
   - Click "Mark Entire Day Unavailable"
   - Confirm the dialog
   - ✅ Verify date header STILL SHOWS
   - ✅ Verify button changes to red "Day Marked Unavailable - Click to Clear"
   - ✅ Verify message appears explaining the day is unavailable
   - ✅ Verify matches are hidden

2. **Change day back to available:**
   - Click the red "Day Marked Unavailable - Click to Clear" button
   - Confirm the dialog
   - ✅ Verify matches reappear
   - ✅ Verify button changes back to gray "Mark Entire Day Unavailable"
   - ✅ Verify all individual match selections are cleared (fresh slate)

3. **Multiple days:**
   - Mark multiple days as unavailable
   - ✅ Verify each date header shows with toggle button
   - ✅ Verify you can toggle each day independently

4. **Page refresh:**
   - Mark a day unavailable
   - Refresh the page
   - ✅ Verify the date header and button still show
   - ✅ Verify the day is still marked unavailable

---

## Edge Cases Handled

1. **Day with no matches in database:**
   - If backend returns no matches for a date, date won't show (correct)
   - Only shows dates that either have matches OR are marked unavailable

2. **Multiple unavailable days:**
   - Each shows its own header and toggle button independently
   - Can toggle each day separately

3. **Filter by date:**
   - Date filter still works
   - Unavailable days included in filter options

4. **No matches but day unavailable:**
   - Shows the unavailable message
   - Button still works to clear

---

## Files Changed

**Frontend:**
- `frontend/src/routes/referee/matches/+page.svelte` - MODIFIED
  - Updated `sortedDates` logic to include unavailable days
  - Added conditional rendering for unavailable day message
  - Added CSS for `day-unavailable-message`

**Backend:**
- No changes needed (working as designed)

---

## Benefits

✅ **Users can change their mind:** No longer stuck with unavailable days  
✅ **Clear feedback:** Message explains what happened and how to fix it  
✅ **Consistent UI:** Date headers always show when there's something to display  
✅ **Better UX:** Obvious path to undo the action  
✅ **Mobile-friendly:** Large touch target always available

---

## Future Enhancements

Possible improvements (not implemented):
- [ ] Show count of hidden matches in the message
- [ ] Add a "View Hidden Matches" toggle to peek at what's hidden
- [ ] Bulk operations: "Clear all unavailable days" button
- [ ] Calendar view showing unavailable days at a glance

---

**Status: ✅ FIXED**

The issue has been resolved. Referees can now always toggle their day-level availability, even after marking a day as unavailable. The date header and toggle button remain visible with a clear message explaining the state.

**To test:**
1. Frontend has been restarted with the fix
2. Sign in as a referee
3. Mark a day unavailable
4. Verify you can click the button to clear it
