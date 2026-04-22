# Getting Started with Referee Scheduler

**Epic 1 is complete!** 🎉

You now have a fully functional authentication and role-based routing system. This guide will help you get started.

---

## What You Have

✅ **Backend API** (Go 1.22)
- RESTful API with health check
- Google OAuth2 authentication
- Session management
- User database with migrations
- Role-based access control

✅ **Frontend** (SvelteKit + TypeScript)
- Mobile-responsive design
- Google sign-in
- Role-based routing
- Three distinct dashboards (assignor, referee, pending)

✅ **Database** (PostgreSQL 16)
- Users table with roles and status
- Automated migrations
- Health checks

✅ **Infrastructure**
- Docker containerization
- docker-compose orchestration
- Development environment ready

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
├── backend/                 # Go API
│   ├── main.go             # Routes, OAuth, middleware
│   ├── user.go             # User model & database
│   ├── migrations/         # SQL migrations
│   └── Dockerfile          # Backend container
│
├── frontend/               # SvelteKit app
│   ├── src/
│   │   ├── routes/         # Pages
│   │   │   ├── +page.svelte              # Login
│   │   │   ├── auth/callback/            # OAuth callback
│   │   │   ├── pending/                  # Pending activation
│   │   │   ├── referee/                  # Referee dashboard
│   │   │   └── assignor/                 # Assignor dashboard
│   │   ├── app.css         # Global styles
│   │   └── app.html        # HTML template
│   └── Dockerfile          # Frontend container
│
├── docker-compose.yml      # Orchestration
├── Makefile               # Dev commands
├── .env                   # Environment config
│
└── docs/
    ├── QUICK_START.md                    # 5-minute guide
    ├── SETUP.md                          # Detailed setup
    ├── README.md                         # Overview
    └── EPIC1_IMPLEMENTATION_REPORT.md    # Full implementation details
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

### Public
- `GET /health` - Health check
- `GET /api/auth/google` - Start OAuth flow
- `GET /api/auth/google/callback` - OAuth callback

### Protected (requires auth)
- `GET /api/auth/me` - Current user info
- `POST /api/auth/logout` - Sign out

---

## User Roles

| Role | Status | Can Access |
|------|--------|------------|
| `pending_referee` | `pending` | Profile edit, pending page |
| `referee` | `active` | Referee dashboard (future: matches, availability) |
| `assignor` | `active` | Assignor dashboard (future: schedule, assignments) |

---

## Database Schema

### Users Table
```sql
id              BIGSERIAL PRIMARY KEY
google_id       VARCHAR(255) UNIQUE NOT NULL
email           VARCHAR(255) NOT NULL
name            VARCHAR(255) NOT NULL
role            VARCHAR(50) DEFAULT 'pending_referee'
status          VARCHAR(50) DEFAULT 'pending'
first_name      VARCHAR(100)
last_name       VARCHAR(100)
date_of_birth   DATE
certified       BOOLEAN DEFAULT FALSE
cert_expiry     DATE
grade           VARCHAR(20)
created_at      TIMESTAMP DEFAULT NOW()
updated_at      TIMESTAMP DEFAULT NOW()
```

---

## What's Implemented (Epic 1)

### Story 1.1 — Project Skeleton ✅
- Go backend with health endpoint
- PostgreSQL database with connection management
- SvelteKit frontend
- Docker Compose setup
- Database migrations (golang-migrate)

### Story 1.2 — Google OAuth2 Login ✅
- Google sign-in button
- OAuth2 flow with CSRF protection
- User creation and retrieval
- Secure session management
- Sign out functionality
- Protected routes

### Story 1.3 — Role-based Routing ✅
- Three user roles (pending_referee, referee, assignor)
- Automatic role-based redirects
- Role enforcement (server-side)
- Distinct dashboards per role
- Assignor promotion via CLI

---

## What's Next (Epic 2)

- **Story 2.1**: Referee profile management (DOB, certification)
- **Story 2.2**: Assignor referee management view
- **Story 2.3**: Referee activation, deactivation, and grading
- **Story 2.4**: Certification expiry flagging

See `STORIES.md` for the full epic breakdown.

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
- **EPIC1_IMPLEMENTATION_REPORT.md**: Complete implementation details
- **README.md**: Project overview

---

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review `SETUP.md` for detailed setup steps
3. Check logs: `make logs`
4. Review `EPIC1_IMPLEMENTATION_REPORT.md` for implementation details

---

## Next Steps

1. ✅ Complete Google OAuth setup (see SETUP.md)
2. ✅ Start the application (`make up`)
3. ✅ Create your assignor account
4. ✅ Verify everything works
5. 📋 Ready to build Epic 2!

**Welcome to Referee Scheduler!** 🎉
