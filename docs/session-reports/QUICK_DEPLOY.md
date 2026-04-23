# Quick Deployment Guide (Traefik + Cloudflare)

One-page reference for deploying with existing Traefik and Cloudflare Tunnel.

## Prerequisites Checklist

- [ ] Docker & Docker Compose installed
- [ ] Traefik running with `traefik_default` network
- [ ] Cloudflare Tunnel configured
- [ ] Domain ready in Cloudflare
- [ ] Google OAuth credentials ready

## Deployment Steps

### 1. Clone & Setup

```bash
# Clone repository
git clone <repo-url> /opt/referee-scheduler
cd /opt/referee-scheduler

# Run setup
chmod +x scripts/traefik-setup.sh
./scripts/traefik-setup.sh
```

### 2. Configure Cloudflare Tunnel

Add to your tunnel config or Cloudflare Zero Trust dashboard:

```yaml
ingress:
  - hostname: ref-sched.yourdomain.com
    service: http://traefik:80
```

### 3. Configure Google OAuth

In [Google Cloud Console](https://console.cloud.google.com/apis/credentials):

- Add redirect URI: `https://ref-sched.yourdomain.com/api/auth/google/callback`
- Copy Client ID and Secret to `.env.production`

### 4. Deploy

```bash
# Build and start
docker-compose -f docker-compose.prod.yml up -d --build

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 5. Verify

Visit: `https://ref-sched.yourdomain.com`

## Quick Commands

```bash
# Status
docker-compose -f docker-compose.prod.yml ps

# Logs
docker-compose -f docker-compose.prod.yml logs -f [service]

# Restart
docker-compose -f docker-compose.prod.yml restart [service]

# Update
git pull && docker-compose -f docker-compose.prod.yml up -d --build

# Backup
./scripts/backup-database.sh

# Restore
./scripts/restore-database.sh ./backups/file.sql.gz
```

## Environment Variables (.env.production)

```bash
DOMAIN=ref-sched.yourdomain.com
FRONTEND_URL=https://ref-sched.yourdomain.com
VITE_API_URL=https://ref-sched.yourdomain.com/api

POSTGRES_USER=referee_scheduler
POSTGRES_PASSWORD=<strong-password>
POSTGRES_DB=referee_scheduler

SESSION_SECRET=<random-secret>

GOOGLE_CLIENT_ID=<your-client-id>
GOOGLE_CLIENT_SECRET=<your-secret>
GOOGLE_REDIRECT_URL=https://ref-sched.yourdomain.com/api/auth/google/callback

BACKUP_RETENTION_DAYS=30
```

## Traefik Labels

The docker-compose file includes these labels:

**Frontend:**
- Rule: `Host(${DOMAIN})`
- Port: 3000

**Backend:**
- Rule: `Host(${DOMAIN}) && PathPrefix(/api)`
- Port: 8080

## Troubleshooting

### Not accessible
1. Check Traefik dashboard for registered routes
2. Verify Cloudflare Tunnel is running
3. Check container logs

### OAuth fails
1. Verify redirect URI in Google Console
2. Check GOOGLE_REDIRECT_URL in .env
3. Ensure FRONTEND_URL matches domain

### Database issues
1. `docker-compose -f docker-compose.prod.yml logs db`
2. `docker-compose -f docker-compose.prod.yml exec db pg_isready`
3. Check network: `docker network inspect traefik_default`

## Automated Backups

```bash
# Add to crontab
crontab -e

# Daily at 2 AM
0 2 * * * cd /opt/referee-scheduler && ./scripts/backup-database.sh >> /var/log/ref-sched-backup.log 2>&1
```

## Network Architecture

```
Internet
   ↓
Cloudflare Tunnel
   ↓
Traefik (traefik_default network)
   ↓
   ├─→ Frontend (Port 3000) - Host: ref-sched.yourdomain.com
   └─→ Backend (Port 8080)  - Host: ref-sched.yourdomain.com/api
        ↓
   Database (Port 5432) - Internal only
```

## Files & Directories

```
ref-sched/
├── docker-compose.prod.yml   # Production compose file
├── .env.production           # Your configuration (DO NOT COMMIT)
├── backups/                  # Database backups
├── scripts/
│   ├── traefik-setup.sh     # Initial setup
│   ├── backup-database.sh   # Backup script
│   └── restore-database.sh  # Restore script
├── backend/                  # Go backend
└── frontend/                 # SvelteKit frontend
```

## Security Checklist

- [ ] Strong database password (32+ chars)
- [ ] Random session secret
- [ ] .env.production permissions: `chmod 600`
- [ ] Database not exposed to host
- [ ] Regular backups configured
- [ ] Google OAuth redirect URI restricted
- [ ] Cloudflare proxy enabled (orange cloud)

## Support

- Full guide: `DEPLOYMENT_TRAEFIK.md`
- Check logs first
- Verify Traefik dashboard
- Test with curl/httpie

---

**Need help?** See detailed deployment guide in DEPLOYMENT_TRAEFIK.md
