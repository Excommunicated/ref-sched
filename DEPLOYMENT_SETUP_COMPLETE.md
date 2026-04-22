# Production Deployment Setup - Complete

**Date**: 2026-04-22  
**Status**: ✅ **READY FOR DEPLOYMENT**

---

## Summary

Production deployment configuration has been set up for **self-hosted deployment** using:
- **Traefik** reverse proxy (your existing setup)
- **Cloudflare Tunnel** for secure ingress
- **Docker Compose** for orchestration
- **PostgreSQL** database with automated backups

---

## Files Created

### Docker Compose Configuration

**`docker-compose.prod.yml`**
- Production-ready compose file
- Traefik labels configured
- Connects to `traefik_default` network
- Health checks enabled
- Auto-restart policies

### Environment Configuration

**`.env.production.example`**
- Template for production environment variables
- Includes all required configuration
- Comments explaining each variable

### Scripts

**`scripts/traefik-setup.sh`** ✓ Executable
- Interactive setup wizard
- Generates secure passwords
- Creates .env.production
- Verifies prerequisites

**`scripts/backup-database.sh`** ✓ Executable
- Automated database backup
- Compression (gzip)
- Retention policy (30 days default)
- Timestamped backups

**`scripts/restore-database.sh`** ✓ Executable
- Database restore from backup
- Safety prompts
- Error handling

### Documentation

**`DEPLOYMENT_README.md`**
- Overview of deployment architecture
- File structure explanation
- Quick reference

**`DEPLOYMENT_TRAEFIK.md`**
- Complete deployment guide
- Step-by-step instructions
- Troubleshooting section
- Maintenance procedures

**`QUICK_DEPLOY.md`**
- One-page quick reference
- Essential commands
- Troubleshooting shortcuts

**`DEPLOYMENT.md`** (Alternative)
- Generic deployment guide
- Nginx-based (for reference)
- Not used with Traefik setup

### Frontend Production Dockerfile

**`frontend/Dockerfile.prod`**
- Production build configuration
- Multi-stage build
- Optimized image size
- Build-time API URL injection

### Nginx Configuration (Reference Only)

**`nginx/nginx.conf`**
- Main nginx configuration
- Included for reference
- Not used with Traefik

**`nginx/conf.d/ref-sched.conf.template`**
- Site-specific configuration template
- Included for reference
- Not used with Traefik

### Updated Files

**`.gitignore`**
- Added `.env.production` (secrets)
- Added `backups/` directory
- Added nginx/certbot artifacts
- Prevents accidental commit of sensitive data

---

## Architecture

```
Internet
    ↓
Cloudflare Tunnel (SSL/TLS, DDoS, CDN)
    ↓
Traefik (Reverse Proxy)
    ↓
    ├─→ Frontend (ref-sched.yourdomain.com)
    └─→ Backend (ref-sched.yourdomain.com/api)
         ↓
    PostgreSQL Database (Internal Network Only)
```

---

## Quick Start Commands

### 1. Initial Setup
```bash
cd /opt/referee-scheduler
./scripts/traefik-setup.sh
```

### 2. Configure Cloudflare Tunnel
Add to your tunnel configuration:
```yaml
ingress:
  - hostname: ref-sched.yourdomain.com
    service: http://traefik:80
```

### 3. Deploy
```bash
docker-compose -f docker-compose.prod.yml up -d --build
```

### 4. Verify
```bash
docker-compose -f docker-compose.prod.yml ps
docker-compose -f docker-compose.prod.yml logs -f
```

---

## Configuration Requirements

Before deploying, you need:

### 1. Domain Configuration
- [ ] Domain added to Cloudflare
- [ ] Cloudflare Tunnel configured
- [ ] DNS pointing to tunnel

### 2. Google OAuth
- [ ] OAuth Client ID created
- [ ] OAuth Client Secret obtained
- [ ] Redirect URI added: `https://your-domain.com/api/auth/google/callback`

### 3. Traefik
- [ ] Traefik running
- [ ] `traefik_default` network exists
- [ ] HTTPS/websecure entrypoint configured

### 4. Environment Variables
- [ ] `.env.production` created (use setup script)
- [ ] All values filled in
- [ ] Secrets are strong/random

---

## Traefik Labels Explained

### Frontend Service
```yaml
traefik.http.routers.ref-sched-frontend.rule=Host(`${DOMAIN}`)
traefik.http.routers.ref-sched-frontend.entrypoints=websecure
traefik.http.services.ref-sched-frontend.loadbalancer.server.port=3000
```
- Routes all traffic for your domain to frontend
- Uses HTTPS entrypoint
- Forwards to container port 3000

### Backend Service
```yaml
traefik.http.routers.ref-sched-backend.rule=Host(`${DOMAIN}`) && PathPrefix(`/api`)
traefik.http.routers.ref-sched-backend.entrypoints=websecure
traefik.http.services.ref-sched-backend.loadbalancer.server.port=8080
```
- Routes `/api` paths to backend
- Uses HTTPS entrypoint
- Forwards to container port 8080

---

## Security Features

### Database
- ✅ Not exposed to host
- ✅ Only accessible via Docker network
- ✅ Strong password (auto-generated)
- ✅ Regular backups

### Application
- ✅ Session secret (auto-generated)
- ✅ Google OAuth only
- ✅ Environment-based configuration
- ✅ No hardcoded secrets

### Network
- ✅ All traffic through Cloudflare
- ✅ DDoS protection
- ✅ SSL/TLS encryption
- ✅ Firewall-ready

---

## Automated Backups

### Setup Cron Job
```bash
crontab -e

# Add this line for daily 2 AM backups:
0 2 * * * cd /opt/referee-scheduler && ./scripts/backup-database.sh >> /var/log/ref-sched-backup.log 2>&1
```

### Manual Backup
```bash
./scripts/backup-database.sh
```

### Restore
```bash
./scripts/restore-database.sh ./backups/referee_scheduler_YYYYMMDD_HHMMSS.sql.gz
```

Backups are:
- Compressed (gzip)
- Timestamped
- Stored in `./backups/`
- Auto-cleaned after 30 days (configurable)

---

## Maintenance Commands

### View Logs
```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f backend
```

### Restart Services
```bash
# All
docker-compose -f docker-compose.prod.yml restart

# Specific
docker-compose -f docker-compose.prod.yml restart backend
```

### Update Application
```bash
git pull
docker-compose -f docker-compose.prod.yml up -d --build
```

### Check Status
```bash
docker-compose -f docker-compose.prod.yml ps
```

### Database Access
```bash
docker-compose -f docker-compose.prod.yml exec db \
  psql -U referee_scheduler -d referee_scheduler
```

---

## Directory Structure

```
ref-sched/
├── docker-compose.prod.yml          # Production Docker Compose
├── .env.production.example          # Environment template
├── .env.production                  # Your config (DO NOT COMMIT)
│
├── scripts/
│   ├── traefik-setup.sh            # Setup wizard
│   ├── backup-database.sh          # Backup database
│   └── restore-database.sh         # Restore database
│
├── nginx/                           # Reference only (not used)
│   ├── nginx.conf
│   └── conf.d/
│       └── ref-sched.conf.template
│
├── backups/                         # Database backups (auto-created)
│
├── backend/
│   ├── Dockerfile                   # Production backend image
│   └── ...
│
├── frontend/
│   ├── Dockerfile.prod              # Production frontend image
│   └── ...
│
└── Documentation
    ├── DEPLOYMENT_README.md         # Overview
    ├── DEPLOYMENT_TRAEFIK.md        # Full guide
    ├── QUICK_DEPLOY.md              # Quick reference
    └── DEPLOYMENT.md                # Generic (nginx)
```

---

## Cost Estimation

Self-hosting with Traefik + Cloudflare:

| Component | Cost/Month |
|-----------|------------|
| VPS (2GB RAM, 2 CPU, 40GB SSD) | $5-15 |
| Cloudflare (Free tier) | $0 |
| Domain | ~$1 (annual/12) |
| Backup Storage (optional) | $0-5 |
| **Total** | **$6-21** |

Compare to managed hosting:
- Azure: $50-100+
- AWS: $30-80+
- Heroku: $50+

**Savings**: ~$300-1000/year

---

## Deployment Checklist

### Pre-Deployment
- [ ] Docker & Docker Compose installed
- [ ] Traefik running
- [ ] Cloudflare Tunnel configured
- [ ] Domain DNS configured
- [ ] Google OAuth credentials ready
- [ ] Server resources adequate (2GB+ RAM)

### Setup
- [ ] Repository cloned
- [ ] `./scripts/traefik-setup.sh` run
- [ ] `.env.production` configured
- [ ] Google OAuth redirect URI added
- [ ] Cloudflare Tunnel pointing to domain

### Deploy
- [ ] `docker-compose -f docker-compose.prod.yml up -d --build`
- [ ] All containers running
- [ ] No errors in logs
- [ ] Application accessible via HTTPS
- [ ] Google OAuth login works

### Post-Deployment
- [ ] First admin user created
- [ ] Test match import
- [ ] Automated backups configured (cron)
- [ ] Test backup/restore
- [ ] Monitoring set up (optional)
- [ ] Documentation updated with specifics

---

## Testing the Deployment

### 1. Container Health
```bash
docker-compose -f docker-compose.prod.yml ps

# All should show "Up"
```

### 2. Application Access
```bash
curl -I https://ref-sched.yourdomain.com

# Should return 200 OK
```

### 3. API Health
```bash
curl https://ref-sched.yourdomain.com/api/health

# Should return JSON
```

### 4. Database Connection
```bash
docker-compose -f docker-compose.prod.yml exec db pg_isready

# Should return: ready
```

### 5. Traefik Routes
- Check Traefik dashboard
- Verify routes registered for your domain
- Check service health indicators

---

## Troubleshooting Quick Reference

### Application Not Accessible
1. Check containers: `docker-compose -f docker-compose.prod.yml ps`
2. Check Traefik dashboard for routes
3. Verify Cloudflare Tunnel is running
4. Check logs: `docker-compose -f docker-compose.prod.yml logs -f`

### Google OAuth Fails
1. Verify redirect URI in Google Console
2. Check `GOOGLE_REDIRECT_URL` in `.env.production`
3. Ensure `FRONTEND_URL` matches actual domain

### Database Connection Error
1. Check network: `docker network inspect traefik_default`
2. Both `db` and `backend` should be listed
3. Check credentials in `.env.production`
4. View db logs: `docker-compose -f docker-compose.prod.yml logs db`

### Container Keeps Restarting
1. View logs: `docker-compose -f docker-compose.prod.yml logs [service]`
2. Check environment variables are set
3. Verify dependencies are running
4. Check resource limits (RAM/CPU)

---

## Next Steps

1. **Read the documentation:**
   - Start with [QUICK_DEPLOY.md](QUICK_DEPLOY.md)
   - Reference [DEPLOYMENT_TRAEFIK.md](DEPLOYMENT_TRAEFIK.md) for details

2. **Run the setup:**
   ```bash
   ./scripts/traefik-setup.sh
   ```

3. **Configure Cloudflare Tunnel:**
   - Add your domain
   - Point to Traefik

4. **Deploy:**
   ```bash
   docker-compose -f docker-compose.prod.yml up -d --build
   ```

5. **Test and verify:**
   - Visit your domain
   - Test login
   - Create first user
   - Import test data

6. **Set up backups:**
   - Configure cron job
   - Test backup/restore

7. **Monitor and maintain:**
   - Check logs regularly
   - Update periodically
   - Keep backups offsite

---

## Support Resources

- **Quick Start:** [QUICK_DEPLOY.md](QUICK_DEPLOY.md)
- **Full Guide:** [DEPLOYMENT_TRAEFIK.md](DEPLOYMENT_TRAEFIK.md)
- **Main README:** [README.md](README.md)
- **Getting Started:** [GETTING_STARTED.md](GETTING_STARTED.md)
- **Testing:** [TESTING_GUIDE.md](TESTING_GUIDE.md)

---

**Production Deployment Configuration Complete! 🎉**

You now have everything needed to deploy the Referee Scheduler application to production using Traefik and Cloudflare Tunnel.

Start with: `./scripts/traefik-setup.sh`
