# Referee Scheduler

A web application for managing referee scheduling for club soccer associations.

## Features

- Google OAuth2 authentication
- Role-based access (Assignor, Referee, Pending Referee)
- Match schedule management (CSV import from Stack Team App)
- Referee profile management with certification tracking
- Referee availability marking (per-match and full-day)
- Day-level unavailability tracking
- Assignment workflow with conflict detection
- Assignment acknowledgment by referees
- Overdue acknowledgment tracking (>24 hours)
- Mobile-responsive design

## Important Notes

### Timezone Handling

**All dates and times are stored and displayed in US Eastern Time (America/New_York).**

- Stack Team App CSV exports are in Eastern Time
- All match dates and times are treated as Eastern Time
- No timezone conversion is applied - this is appropriate for a local sports club where all matches occur in the Eastern timezone

## Tech Stack

- **Backend**: Go 1.22
- **Frontend**: SvelteKit
- **Database**: PostgreSQL 16
- **Auth**: Google OAuth2
- **Deployment**: Docker

## Prerequisites

- Docker and Docker Compose
- Google Cloud Project with OAuth2 credentials

## Google OAuth2 Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google+ API:
   - Navigate to "APIs & Services" > "Library"
   - Search for "Google+ API"
   - Click "Enable"
4. Create OAuth2 credentials:
   - Navigate to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth client ID"
   - Choose "Web application"
   - Add authorized redirect URIs:
     - `http://localhost:8080/api/auth/google/callback` (for local development)
   - Click "Create"
5. Copy the Client ID and Client Secret

## Local Development Setup

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd referee-scheduler
   ```

2. Create a `.env` file from the example:
   ```bash
   cp .env.example .env
   ```

3. Edit `.env` and add your Google OAuth2 credentials:
   ```
   GOOGLE_CLIENT_ID=your-client-id-here.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your-client-secret-here
   ```

4. Start the application with Docker Compose:
   ```bash
   docker-compose up --build
   ```

5. Access the application:
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080
   - Health check: http://localhost:8080/health

## Database Migrations

Migrations run automatically when the backend starts. Migration files are located in `backend/migrations/`.

To create a new migration:
1. Create two files in `backend/migrations/`:
   - `XXX_description.up.sql` (for applying the migration)
   - `XXX_description.down.sql` (for rolling back)
2. Restart the backend container

## Creating an Assignor Account

By default, new users are created with the `pending_referee` role. To create an assignor account:

1. Sign in with Google to create your user account
2. Connect to the database:
   ```bash
   docker exec -it referee-scheduler-db psql -U referee_scheduler
   ```
3. Update your user role:
   ```sql
   UPDATE users SET role = 'assignor', status = 'active' WHERE email = 'your-email@example.com';
   ```
4. Exit the database:
   ```
   \q
   ```
5. Sign out and sign back in to see the assignor dashboard

## Project Structure

```
referee-scheduler/
├── backend/               # Go backend
│   ├── main.go           # Application entry point
│   ├── user.go           # User model and database operations
│   ├── migrations/       # Database migrations
│   ├── Dockerfile        # Backend container config
│   └── go.mod            # Go dependencies
├── frontend/             # SvelteKit frontend
│   ├── src/
│   │   ├── routes/       # SvelteKit routes
│   │   ├── app.html      # HTML template
│   │   └── app.css       # Global styles
│   ├── Dockerfile        # Frontend container config
│   └── package.json      # Node dependencies
├── docker-compose.yml    # Docker orchestration
├── .env.example          # Environment variables template
└── README.md             # This file
```

## Development Workflow

### Backend Development

The backend automatically recompiles when you make changes (if using volume mounts). To see logs:
```bash
docker-compose logs -f backend
```

### Frontend Development

The frontend uses hot module replacement. Changes are reflected immediately. To see logs:
```bash
docker-compose logs -f frontend
```

### Database Access

To access the PostgreSQL database:
```bash
docker exec -it referee-scheduler-db psql -U referee_scheduler
```

Common commands:
- `\dt` - List all tables
- `\d users` - Describe users table
- `SELECT * FROM users;` - View all users

## API Endpoints

### Authentication
- `GET /health` - Health check
- `GET /api/auth/google` - Initiate Google OAuth2 flow
- `GET /api/auth/google/callback` - OAuth2 callback
- `POST /api/auth/logout` - Sign out
- `GET /api/auth/me` - Get current user (requires authentication)

### Profile Management
- `GET /api/profile` - Get current user's profile
- `PUT /api/profile` - Update current user's profile

### Referee Management (Assignor Only)
- `GET /api/referees` - List all referees with filtering
- `PUT /api/referees/{id}` - Update referee status/grade

### Match Management (Assignor Only)
- `POST /api/matches/import/parse` - Parse CSV file for preview
- `POST /api/matches/import/confirm` - Confirm and import matches
- `GET /api/matches` - List all matches with filters
- `PUT /api/matches/{id}` - Update match details
- `GET /api/matches/{id}/eligible-referees` - Get eligible referees for a role
- `GET /api/matches/{match_id}/conflicts` - Check for assignment conflicts

### Referee Availability
- `GET /api/referee/matches` - Get eligible matches for current referee
- `POST /api/referee/matches/{id}/availability` - Toggle availability for a match
- `GET /api/referee/day-unavailability` - Get days marked unavailable
- `POST /api/referee/day-unavailability/{date}` - Toggle full-day unavailability

### Assignment Operations (Assignor Only)
- `POST /api/matches/{match_id}/roles/{role_type}/assign` - Assign/reassign/remove referee

### Assignment Acknowledgment (Referee)
- `POST /api/referee/matches/{match_id}/acknowledge` - Acknowledge assignment

## User Roles

- **pending_referee**: New user awaiting assignor approval
- **referee**: Active referee who can view matches and mark availability
- **assignor**: Admin who can manage referees and assign matches

## Troubleshooting

### "Failed to connect to database"
- Ensure PostgreSQL container is running: `docker-compose ps`
- Check database logs: `docker-compose logs db`

### "OAuth2 error" or "Invalid credentials"
- Verify your Google OAuth2 credentials in `.env`
- Ensure the redirect URI matches exactly in Google Cloud Console

### Frontend can't connect to backend
- Ensure both containers are running: `docker-compose ps`
- Check CORS configuration in `backend/main.go`
- Verify `VITE_API_URL` in docker-compose.yml

## Documentation

See **[DOCS_INDEX.md](DOCS_INDEX.md)** for a complete guide to all documentation.

Key documents:
- **[GETTING_STARTED.md](GETTING_STARTED.md)** - Setup and usage guide
- **[DEPLOYMENT.md](DEPLOYMENT.md)** - Production deployment guide
- **[PROJECT_STATUS.md](PROJECT_STATUS.md)** - Current status (91% complete)
- **[STORIES.md](STORIES.md)** - All epics and user stories

## Project Status

✅ **Epics 1-6 Complete** - All core features implemented  
🚧 **Epic 7 In Progress** - Self-hosted deployment infrastructure ready

See [PROJECT_STATUS.md](PROJECT_STATUS.md) for detailed status.

## License

Private - Club use only
