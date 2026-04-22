# Self-Hosted Deployment with Traefik & Cloudflare Tunnel

This guide covers deploying the Referee Scheduler application using your existing Traefik reverse proxy with Cloudflare Tunnel.

## Prerequisites

- Docker and Docker Compose installed
- Traefik running with `traefik_default` network
- Cloudflare Tunnel configured
- Domain configured in Cloudflare

## Quick Start

### 1. Clone Repository

```bash
git clone <your-repo-url> /opt/referee-scheduler
cd /opt/referee-scheduler
```

### 2. Configure Environment

Create `.env.production` from template:

```bash
cp .env.production.example .env.production
```

Edit `.env.production` and configure:

```bash
# Domain Configuration
DOMAIN=ref-sched.yourdomain.com
FRONTEND_URL=https://ref-sched.yourdomain.com
VITE_API_URL=https://ref-sched.yourdomain.com/api

# Database - Generate strong password
POSTGRES_USER=referee_scheduler
POSTGRES_PASSWORD=<generate-strong-password>
POSTGRES_DB=referee_scheduler

# Session Secret - Generate with: openssl rand -base64 32
SESSION_SECRET=<generate-random-secret>

# Google OAuth2
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=https://ref-sched.yourdomain.com/api/auth/google/callback

# Backups
BACKUP_RETENTION_DAYS=30
```

### 3. Generate Secure Secrets

```bash
# Generate database password
openssl rand -base64 32 | tr -d "=+/" | cut -c1-32

# Generate session secret
openssl rand -base64 32
```

### 4. Configure Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create OAuth 2.0 Client ID (or use existing)
3. Add authorized redirect URI:
   ```
   https://ref-sched.yourdomain.com/api/auth/google/callback
   ```
4. Copy Client ID and Secret to `.env.production`

### 5. Configure Cloudflare Tunnel

Add your domain to Cloudflare Tunnel configuration:

```yaml
# In your Cloudflare Tunnel config
ingress:
  - hostname: ref-sched.yourdomain.com
    service: http://traefik:80
```

Or configure via Cloudflare Zero Trust dashboard:
- Public Hostname: `ref-sched.yourdomain.com`
- Service: `http://traefik:80` (or your Traefik entrypoint)

### 6. Deploy Application

```bash
# Create backups directory
mkdir -p backups

# Build and start services
docker-compose -f docker-compose.prod.yml up -d --build

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 7. Verify Deployment

The application should now be accessible at: `https://ref-sched.yourdomain.com`

Check Traefik dashboard to verify routes are registered.

## Configuration Details

### Traefik Labels

The application uses these Traefik labels:

**Frontend:**
```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.ref-sched-frontend.rule=Host(`${DOMAIN}`)"
  - "traefik.http.routers.ref-sched-frontend.entrypoints=websecure"
  - "traefik.http.services.ref-sched-frontend.loadbalancer.server.port=3000"
```

**Backend (API):**
```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.ref-sched-backend.rule=Host(`${DOMAIN}`) && PathPrefix(`/api`)"
  - "traefik.http.routers.ref-sched-backend.entrypoints=websecure"
  - "traefik.http.services.ref-sched-backend.loadbalancer.server.port=8080"
```

### Network Configuration

The application connects to your existing `traefik_default` network:

```yaml
networks:
  traefik_default:
    external: true
```

Make sure this network exists:
```bash
docker network ls | grep traefik
```

If not, create it:
```bash
docker network create traefik_default
```

## Database Management

### Backups

Create backup script executable:
```bash
chmod +x scripts/backup-database.sh
```

Run backup:
```bash
./scripts/backup-database.sh
```

Backups are stored in `./backups/` directory.

### Automated Backups

Set up daily backups with cron:

```bash
crontab -e
```

Add line (adjust path as needed):
```
0 2 * * * cd /opt/referee-scheduler && ./scripts/backup-database.sh >> /var/log/ref-sched-backup.log 2>&1
```

### Restore from Backup

```bash
chmod +x scripts/restore-database.sh
./scripts/restore-database.sh ./backups/referee_scheduler_YYYYMMDD_HHMMSS.sql.gz
```

## Maintenance

### View Logs

```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f backend
docker-compose -f docker-compose.prod.yml logs -f frontend
docker-compose -f docker-compose.prod.yml logs -f db
```

### Restart Services

```bash
# Restart all
docker-compose -f docker-compose.prod.yml restart

# Restart specific service
docker-compose -f docker-compose.prod.yml restart backend
```

### Update Application

```bash
# Pull latest code
git pull

# Rebuild and restart
docker-compose -f docker-compose.prod.yml up -d --build

# Verify migrations ran
docker-compose -f docker-compose.prod.yml logs backend | grep -i migration
```

### Database Access

Connect to PostgreSQL:
```bash
docker-compose -f docker-compose.prod.yml exec db \
  psql -U referee_scheduler -d referee_scheduler
```

## Troubleshooting

### Application Not Accessible

1. **Check container status:**
   ```bash
   docker-compose -f docker-compose.prod.yml ps
   ```

2. **Check Traefik routing:**
   - Visit Traefik dashboard
   - Verify routers are registered for your domain
   - Check service health

3. **Check logs:**
   ```bash
   docker-compose -f docker-compose.prod.yml logs backend
   docker-compose -f docker-compose.prod.yml logs frontend
   ```

4. **Verify network:**
   ```bash
   docker network inspect traefik_default
   ```
   
   Both `referee-scheduler-backend-prod` and `referee-scheduler-frontend-prod` should be listed.

### Google OAuth Not Working

1. **Verify redirect URI** in Google Cloud Console:
   ```
   https://ref-sched.yourdomain.com/api/auth/google/callback
   ```

2. **Check environment variables:**
   ```bash
   docker-compose -f docker-compose.prod.yml exec backend env | grep GOOGLE
   ```

3. **Check frontend URL:**
   ```bash
   docker-compose -f docker-compose.prod.yml exec backend env | grep FRONTEND_URL
   ```

### Database Connection Issues

1. **Check database container:**
   ```bash
   docker-compose -f docker-compose.prod.yml exec db pg_isready
   ```

2. **Verify credentials:**
   ```bash
   docker-compose -f docker-compose.prod.yml exec backend env | grep DATABASE_URL
   ```

3. **Check database logs:**
   ```bash
   docker-compose -f docker-compose.prod.yml logs db
   ```

### Backend Can't Connect to Database

Ensure containers are on same network:
```bash
docker network inspect traefik_default | grep -A 5 "referee-scheduler"
```

Both `db` and `backend` should appear.

## Cloudflare Configuration

### DNS Settings

In Cloudflare DNS:
- Set your subdomain to **Proxied** (orange cloud)
- Cloudflare Tunnel will handle the connection

### Cloudflare Tunnel

Verify tunnel configuration includes:
```yaml
ingress:
  - hostname: ref-sched.yourdomain.com
    service: http://traefik:80
  # ... other routes
  - service: http_status:404
```

## Security Considerations

### Database Security

- Database is **NOT** exposed to host (no port mapping)
- Only accessible via Docker network
- Use strong passwords (32+ characters)

### Session Security

- Session secret should be cryptographically random
- Regenerate periodically
- Never commit to version control

### Environment Variables

- Keep `.env.production` secure
- Set proper file permissions:
  ```bash
  chmod 600 .env.production
  ```
- Exclude from version control (already in `.gitignore`)

### Backups

- Store backups offsite
- Encrypt sensitive backups
- Test restore procedures regularly

## Performance Tuning

### Database

For larger deployments, tune PostgreSQL:

```yaml
# In docker-compose.prod.yml under db service
environment:
  POSTGRES_MAX_CONNECTIONS: 100
  POSTGRES_SHARED_BUFFERS: 256MB
```

### Backend

Set resource limits:

```yaml
backend:
  deploy:
    resources:
      limits:
        cpus: '1.0'
        memory: 1G
      reservations:
        memory: 512M
```

## Monitoring

### Health Checks

Check service health:
```bash
# Database
docker-compose -f docker-compose.prod.yml exec db pg_isready

# Backend (should return JSON)
curl http://localhost:8080/api/health

# Frontend (should return 200)
curl -I http://localhost:3000
```

### Resource Usage

```bash
# Container stats
docker stats

# Disk usage
docker system df

# Logs size
du -sh /var/lib/docker/containers/*
```

## Backup Strategy

### What to Backup

1. **Database** (automated via script)
2. **Environment file** (`.env.production`)
3. **Docker volumes** (if needed)

### Backup Locations

Recommended:
- Local: `./backups/` (for quick restore)
- Offsite: Cloud storage (S3, Backblaze, etc.)
- Keep at least 3 copies in different locations

### Restore Testing

Test restore procedure quarterly:
```bash
# 1. Create fresh backup
./scripts/backup-database.sh

# 2. Restore to test database
./scripts/restore-database.sh ./backups/latest.sql.gz

# 3. Verify data integrity
docker-compose -f docker-compose.prod.yml exec db \
  psql -U referee_scheduler -d referee_scheduler -c "SELECT COUNT(*) FROM users;"
```

## Cost Considerations

Self-hosting with Traefik + Cloudflare:

| Component | Cost |
|-----------|------|
| VPS (2GB RAM, 2 CPU) | $5-15/month |
| Cloudflare Tunnel | Free |
| Domain | $10-15/year |
| Backup Storage (optional) | $0-5/month |
| **Total** | **$5-20/month** |

## Next Steps

After deployment:

1. ✅ Verify application is accessible
2. ✅ Test Google OAuth login
3. ✅ Create first admin user
4. ✅ Import match schedule
5. ✅ Set up automated backups
6. ✅ Configure monitoring
7. ✅ Document your specific configuration

---

**Deployment with Traefik Complete! 🎉**

Your referee scheduler is now running with Traefik reverse proxy and Cloudflare Tunnel.
