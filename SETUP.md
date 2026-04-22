# Setup Instructions

## Quick Start

1. **Set up Google OAuth2 credentials** (see below)
2. **Configure environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env and add your Google OAuth2 credentials
   ```
3. **Start the application**:
   ```bash
   make up
   # or: docker-compose up -d
   ```
4. **Access the app**: http://localhost:3000
5. **Create an assignor account** (see below)

---

## Detailed Google OAuth2 Setup

### Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" dropdown at the top
3. Click "New Project"
4. Enter a project name (e.g., "Referee Scheduler")
5. Click "Create"

### Step 2: Enable Required APIs

1. In your new project, go to "APIs & Services" > "Library"
2. Search for "Google+ API" (or "People API")
3. Click on it and click "Enable"

### Step 3: Configure OAuth Consent Screen

1. Go to "APIs & Services" > "OAuth consent screen"
2. Select "External" (unless you have a Google Workspace)
3. Click "Create"
4. Fill in the required fields:
   - **App name**: Referee Scheduler
   - **User support email**: Your email
   - **Developer contact email**: Your email
5. Click "Save and Continue"
6. On "Scopes" page, click "Save and Continue" (we'll use default scopes)
7. On "Test users" page, add your email address as a test user
8. Click "Save and Continue"
9. Review and click "Back to Dashboard"

### Step 4: Create OAuth2 Credentials

1. Go to "APIs & Services" > "Credentials"
2. Click "+ Create Credentials" > "OAuth client ID"
3. Choose "Web application" as the application type
4. Enter a name (e.g., "Referee Scheduler Web")
5. Under "Authorized JavaScript origins", add:
   - `http://localhost:3000`
6. Under "Authorized redirect URIs", add:
   - `http://localhost:8080/api/auth/google/callback`
7. Click "Create"
8. A dialog will appear with your credentials:
   - **Client ID**: Copy this (looks like `xxxxx.apps.googleusercontent.com`)
   - **Client Secret**: Copy this
9. Click "OK"

### Step 5: Add Credentials to .env File

Edit your `.env` file:
```bash
GOOGLE_CLIENT_ID=your-client-id-here.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret-here
```

---

## Starting the Application

### Option 1: Using Make (recommended)

```bash
make up
```

This will:
- Build all containers
- Start PostgreSQL, backend, and frontend
- Run database migrations automatically

### Option 2: Using Docker Compose Directly

```bash
docker-compose up --build -d
```

### Verify Everything is Running

```bash
docker-compose ps
```

You should see three services running:
- `referee-scheduler-db` (PostgreSQL)
- `referee-scheduler-backend` (Go API)
- `referee-scheduler-frontend` (SvelteKit)

Check the health endpoint:
```bash
curl http://localhost:8080/health
```

Should return: `{"status":"ok","time":"..."}`

---

## Creating Your First Assignor Account

1. **Sign in with Google**:
   - Open http://localhost:3000
   - Click "Sign in with Google"
   - Complete the OAuth flow
   - You'll land on the "Pending Activation" page

2. **Promote your account to assignor**:

   Option A - Using Make:
   ```bash
   make seed-assignor
   # Enter your email when prompted
   ```

   Option B - Using SQL directly:
   ```bash
   docker exec -it referee-scheduler-db psql -U referee_scheduler
   ```

   Then run:
   ```sql
   UPDATE users SET role = 'assignor', status = 'active' WHERE email = 'your-email@example.com';
   \q
   ```

3. **Sign out and sign back in**:
   - You'll now see the Assignor Dashboard

---

## Common Commands

```bash
# Start services
make up

# Stop services
make down

# View logs
make logs

# View backend logs only
make backend-logs

# View frontend logs only
make frontend-logs

# Access database shell
make db-shell

# Clean everything (removes data!)
make clean

# Create assignor account
make seed-assignor
```

---

## Troubleshooting

### "OAuth2 error: invalid_client"
- Double-check your Client ID and Client Secret in `.env`
- Make sure there are no extra spaces or quotes
- Restart the backend: `docker-compose restart backend`

### "redirect_uri_mismatch"
- Ensure `http://localhost:8080/api/auth/google/callback` is exactly listed in your Google Cloud Console redirect URIs
- No trailing slash
- Must be http (not https) for localhost

### Frontend shows "Failed to fetch user"
- Check backend is running: `docker-compose ps`
- Check backend logs: `make backend-logs`
- Verify CORS settings in `backend/main.go`

### Database connection errors
- Ensure PostgreSQL container is healthy: `docker-compose ps`
- Check database logs: `docker-compose logs db`
- Verify DATABASE_URL in docker-compose.yml

### "This app is blocked" when signing in with Google
- Your app is in testing mode and you haven't added your email as a test user
- Go to OAuth consent screen > Test users > Add your email

---

## Next Steps

Once Epic 1 is working:
1. ✅ You can sign in with Google
2. ✅ Role-based routing works (assignor/referee/pending)
3. ✅ Sessions persist across page refreshes
4. ✅ Sign out works

Epic 2 will add:
- Referee profile management (DOB, certification)
- Assignor referee management view
- Referee activation and grading

---

## For Production Deployment

When deploying to Azure or production:

1. **Update redirect URIs** in Google Cloud Console:
   - Add your production domain
   - Example: `https://your-domain.com/api/auth/google/callback`

2. **Update environment variables**:
   ```bash
   GOOGLE_REDIRECT_URL=https://your-domain.com/api/auth/google/callback
   FRONTEND_URL=https://your-domain.com
   ENV=production
   SESSION_SECRET=<generate-a-long-random-string>
   ```

3. **Enable HTTPS**:
   - Set `Secure: true` for cookies (already done when ENV=production)
   - Use a reverse proxy (nginx) or Azure App Service for SSL termination

4. **Change OAuth consent screen** from "Testing" to "In Production" (in Google Cloud Console)
