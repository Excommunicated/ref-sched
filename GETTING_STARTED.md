# Getting Started with Referee Scheduler

**Epics 1-6 are complete!** 🎉

You now have a fully functional referee scheduling application with authentication, match management, availability tracking, assignments, and acknowledgments. This guide will help you get started.

---

## What You Have

✅ **Complete Referee Scheduling System**
- Google OAuth2 authentication
- Referee profile management with certification tracking
- CSV match import from Stack Team App
- Automatic age group parsing and role slot configuration
- Eligibility engine (age-based and certification-based)
- Referee availability marking (per-match and full-day)
- Assignment interface with conflict detection
- Assignment acknowledgment with overdue tracking
- Mobile-first responsive design

✅ **Backend API** (Go 1.22)
- 20 RESTful API endpoints
- Google OAuth2 authentication
- Session management
- Role-based access control (assignor/referee/pending)
- Eligibility computation engine
- Conflict detection for double-booking
- Audit trail for all assignments

✅ **Frontend** (SvelteKit + TypeScript)
- Mobile-responsive design (320px+)
- Assignor dashboard with match schedule and assignment panel
- Referee dashboard with availability marking
- Profile management
- CSV import with preview
- Real-time status updates

✅ **Database** (PostgreSQL 16)
- Users, matches, match_roles, availability tables
- Day-level unavailability tracking
- Assignment history audit trail
- Automated migrations (6 migration files)
- Timezone handling (US Eastern)

✅ **Infrastructure**
- Docker containerization
- docker-compose orchestration
- Development environment ready
- Production-ready codebase

---

## Quick Start (5 Minutes)

### 1. Get Google OAuth2 Credentials

See `SETUP.md` for detailed instructions, or quick version:

1. https://console.cloud.google.com/ → New Project
2. Enable Google+ API
3. OAuth consent screen → Add your email as test user
4. Create OAuth client ID → Web application
5. Redirect URI: `http://localhost:8080/api/auth/google/callback`
6. Copy Client ID and Secret

### 2. Configure .env

Edit `.env` and replace the placeholders:
```bash
GOOGLE_CLIENT_ID=your-id-here.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-secret-here
```

### 3. Start Everything

```bash
make up
```

### 4. Sign In & Test

1. http://localhost:3000
2. Sign in with Google
3. See "Pending Activation" page
4. Run: `make seed-assignor` (enter your email)
5. Sign out and back in
6. See Assignor Dashboard

---

## Project Structure

```
ref-sched/
├── backend/                          # Go API
│   ├── main.go                      # Routes, OAuth, middleware
│   ├── user.go                      # User auth & profiles
│   ├── profile.go                   # Profile endpoints
│   ├── referees.go                  # Referee management
│   ├── matches.go                   # Match CRUD & import
│   ├── eligibility.go               # Eligibility engine
│   ├── availability.go              # Availability marking
│   ├── assignments.go               # Assignment operations
│   ├── acknowledgment.go            # Assignment acknowledgment
│   ├── day_unavailability.go        # Day-level unavailability
│   ├── migrations/                  # SQL migrations (6 files)
│   └── Dockerfile                   # Backend container
│
├── frontend/                         # SvelteKit app
│   ├── src/routes/
│   │   ├── +page.svelte             # Login
│   │   ├── auth/callback/           # OAuth callback
│   │   ├── pending/                 # Pending activation
│   │   ├── referee/
│   │   │   ├── +page.svelte         # Dashboard
│   │   │   ├── profile/             # Profile management
│   │   │   └── matches/             # Availability & assignments
│   │   └── assignor/
│   │       ├── +page.svelte         # Dashboard
│   │       ├── referees/            # Referee management
│   │       └── matches/             # Schedule & assignments
│   └── Dockerfile                   # Frontend container
│
├── docker-compose.yml               # Orchestration
├── Makefile                         # Dev commands
├── .env                             # Environment config
│
└── Documentation/
    ├── README.md                         # Project overview
    ├── PROJECT_STATUS.md                 # Current status (91% complete)
    ├── STORIES.md                        # All epics and stories
    ├── QUICK_START.md                    # 5-minute guide
    ├── SETUP.md                          # Detailed setup
    ├── GETTING_STARTED.md                # This file
    ├── TESTING_GUIDE.md                  # Testing instructions
    ├── DEPLOYMENT.md                     # Production deployment
    └── docs/
        ├── implementation-reports/       # Epic implementation details
        ├── architecture/                 # Technical architecture
        └── session-reports/              # Development history
```

---

## Available Commands

```bash
# Start/Stop
make up                 # Start all services
make down               # Stop all services
make build              # Rebuild containers

# Logs
make logs               # All logs
make backend-logs       # Backend only
make frontend-logs      # Frontend only

# Database
make db-shell           # PostgreSQL shell
make seed-assignor      # Create assignor user

# Cleanup
make clean              # Stop and remove all data (!)
```

---

## URLs

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Health Check**: http://localhost:8080/health
- **Database**: localhost:5432

---

## API Endpoints

See `README.md` for complete API documentation. Key endpoints:

### Authentication
- `GET /health` - Health check
- `GET /api/auth/google` - Start OAuth flow
- `GET /api/auth/google/callback` - OAuth callback
- `GET /api/auth/me` - Current user info
- `POST /api/auth/logout` - Sign out

### Profile & Referee Management
- `GET /api/profile` - Get/update profile
- `GET /api/referees` - List referees (assignor only)
- `PUT /api/referees/{id}` - Update referee (assignor only)

### Match Management (Assignor)
- `POST /api/matches/import/parse` - Parse CSV
- `POST /api/matches/import/confirm` - Import matches
- `GET /api/matches` - List matches
- `PUT /api/matches/{id}` - Update match

### Availability & Assignments
- `GET /api/referee/matches` - Get eligible matches
- `POST /api/referee/matches/{id}/availability` - Mark available
- `POST /api/matches/{match_id}/roles/{role_type}/assign` - Assign referee
- `POST /api/referee/matches/{match_id}/acknowledge` - Acknowledge assignment

### Day Unavailability
- `GET /api/referee/day-unavailability` - List unavailable days
- `POST /api/referee/day-unavailability/{date}` - Toggle day unavailability

---

## User Roles

| Role | Status | Can Access |
|------|--------|------------|
| `pending_referee` | `pending` | Profile edit, pending page |
| `referee` | `active` | Referee dashboard (future: matches, availability) |
| `assignor` | `active` | Assignor dashboard (future: schedule, assignments) |

---

## Database Schema

See `PROJECT_STATUS.md` for complete schema. Key tables:

### Users
- Profile info: id, google_id, email, name, first_name, last_name, date_of_birth
- Role & status: role (assignor/referee/pending_referee), status (active/inactive/pending/removed)
- Certification: certified, cert_expiry, grade

### Matches
- Match info: event_name, team_name, age_group, match_date, start_time, end_time
- Location: location, description
- Status: status (active/cancelled), stack_reference_id

### Match Roles
- Role assignment: match_id, role_type (center/assistant_1/assistant_2), assigned_referee_id
- Acknowledgment: acknowledged, acknowledged_at

### Availability
- Per-match availability: match_id, referee_id, created_at

### Day Unavailability
- Full-day unavailability: referee_id, unavailable_date, reason, created_at

### Assignment History
- Audit trail: match_id, role_type, old_referee_id, new_referee_id, action, actor_id, created_at

---

## What's Implemented (Epics 1-6)

### Epic 1 — Foundation & Authentication ✅
- Google OAuth2 login
- Session management
- Role-based routing (assignor/referee/pending)
- PostgreSQL database with migrations

### Epic 2 — Profiles & Verification ✅
- Referee profile management (DOB, certification, expiry)
- Assignor referee management view
- Referee activation, deactivation, removal
- Grade assignment (Junior/Mid/Senior)
- Certification expiry flagging

### Epic 3 — Match Management ✅
- CSV import from Stack Team App
- Automatic age group extraction
- Role slot configuration (U6/U8: 1 CR, U10: 1 CR, U12+: 1 CR + 2 AR)
- Manual match editing
- Match cancellation
- Assignment status badges

### Epic 4 — Eligibility & Availability ✅
- Age-based eligibility (U10 and younger: age ≥ age_group + 1)
- Certification-based eligibility (U12+ center: requires valid cert)
- Referee availability marking
- Day-level unavailability
- Eligible matches filtered by profile completion

### Epic 5 — Assignment Interface ✅
- Assignment panel with role overview
- Eligible referee picker per role
- Assign, reassign, remove operations
- Double-booking conflict detection
- Assignment audit trail

### Epic 6 — Referee Assignment View & Acknowledgment ✅
- View assigned matches with full details
- Assignment acknowledgment by referees
- Assignor visibility of acknowledgment status
- Overdue acknowledgment tracking (>24 hours)
- Day-level unavailability marking

---

## What's Next (Epic 7)

- **Story 7.1**: Production-ready Docker images
- **Story 7.2**: Azure deployment with HTTPS, managed PostgreSQL, and daily backups

**The MVP is feature-complete!** Only deployment remains.

See `PROJECT_STATUS.md` for detailed status and `STORIES.md` for the full epic breakdown.

---

## Troubleshooting

### OAuth Errors

**"invalid_client"**
```bash
# Check .env has correct credentials
# Restart backend
docker-compose restart backend
```

**"redirect_uri_mismatch"**
- URI must be exactly: `http://localhost:8080/api/auth/google/callback`
- Check Google Cloud Console → Credentials → Your OAuth client

**"This app is blocked"**
- Add your email as test user in OAuth consent screen

### Connection Errors

**Frontend can't reach backend**
```bash
# Check all services are running
docker-compose ps

# Check backend logs
make backend-logs
```

**Database connection failed**
```bash
# Check database is healthy
docker-compose ps

# View database logs
docker-compose logs db
```

### General Issues

**Services won't start**
```bash
# Clean everything and restart
make clean
make up
```

**Changes not appearing**
```bash
# Rebuild containers
make down
make build
make up
```

---

## Development Workflow

1. **Make code changes** in `backend/` or `frontend/`
2. **Backend**: Container auto-restarts on Go file changes
3. **Frontend**: Hot module replacement (instant updates)
4. **View logs**: `make logs`
5. **Test changes**: Refresh browser

### Adding Database Changes

1. Create migration files in `backend/migrations/`:
   ```
   002_add_matches_table.up.sql
   002_add_matches_table.down.sql
   ```
2. Restart backend: `docker-compose restart backend`
3. Migrations run automatically

---

## Testing the Implementation

### Manual Test Checklist

- [ ] `make up` starts all services
- [ ] http://localhost:8080/health returns `{"status":"ok"}`
- [ ] http://localhost:3000 shows login page
- [ ] "Sign in with Google" works
- [ ] User lands on pending page
- [ ] `make seed-assignor` promotes user
- [ ] Sign out works
- [ ] Sign in again shows assignor dashboard
- [ ] Session persists on page refresh

### Database Verification

```bash
make db-shell
```

```sql
-- View all users
SELECT id, email, name, role, status FROM users;

-- Check indexes
\d users

-- Exit
\q
```

---

## Resources

- **PRD**: Full product requirements
- **STORIES.md**: All epics and stories
- **SETUP.md**: Detailed Google OAuth setup
- **QUICK_START.md**: 5-minute quickstart
- **docs/implementation-reports/**: Complete implementation details for all epics
- **README.md**: Project overview

---

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review `SETUP.md` for detailed setup steps
3. Check logs: `make logs`
4. Review `docs/implementation-reports/` for implementation details

---

## Next Steps

1. ✅ Complete Google OAuth setup (see SETUP.md)
2. ✅ Start the application (`make up`)
3. ✅ Create your assignor account
4. ✅ Verify everything works
5. 📋 Ready to build Epic 2!

**Welcome to Referee Scheduler!** 🎉
