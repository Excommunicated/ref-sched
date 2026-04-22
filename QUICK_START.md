# Quick Start Guide

Get the Referee Scheduler running in 5 minutes!

## Prerequisites

- Docker and Docker Compose installed
- Google account

## Step 1: Get Google OAuth2 Credentials (5 minutes)

1. Go to https://console.cloud.google.com/
2. Create a new project: "Referee Scheduler"
3. Enable Google+ API (or People API)
4. Configure OAuth consent screen:
   - External, add your email as test user
5. Create credentials → OAuth client ID → Web application
6. Add redirect URI: `http://localhost:8080/api/auth/google/callback`
7. Copy Client ID and Client Secret

## Step 2: Configure Environment

Edit the `.env` file:
```bash
GOOGLE_CLIENT_ID=paste-your-client-id-here.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=paste-your-client-secret-here
```

## Step 3: Start the Application

```bash
make up
```

Wait ~30 seconds for services to start.

## Step 4: Sign In

1. Open http://localhost:3000
2. Click "Sign in with Google"
3. Complete the OAuth flow
4. You'll see the "Pending Activation" page

## Step 5: Make Yourself an Assignor

```bash
make seed-assignor
# Enter your email when prompted
```

## Step 6: Access the Assignor Dashboard

1. Sign out (button on pending page)
2. Sign back in
3. You'll now see the Assignor Dashboard!

---

## Common Commands

```bash
make up          # Start all services
make down        # Stop all services
make logs        # View all logs
make db-shell    # Open database shell
make clean       # Remove all data (DANGER!)
```

## Troubleshooting

**"OAuth error: invalid_client"**
- Check your Client ID and Secret in `.env`
- Restart backend: `docker-compose restart backend`

**"redirect_uri_mismatch"**
- Ensure redirect URI is exactly: `http://localhost:8080/api/auth/google/callback`
- No trailing slash, must be http (not https)

**Frontend won't load**
- Check all services are running: `docker-compose ps`
- Wait ~30 seconds after `make up`

---

## What's Next?

Epic 1 is complete! You now have:
- ✅ Working authentication
- ✅ Role-based access
- ✅ Database with migrations
- ✅ Mobile-responsive UI

Next epics will add:
- Epic 2: Referee profiles and management
- Epic 3: Match schedule import and management
- Epic 4: Eligibility and availability
- Epic 5: Assignment interface
- Epic 6: Referee assignment view
- Epic 7: Azure deployment

See `EPIC1_IMPLEMENTATION_REPORT.md` for full details.
