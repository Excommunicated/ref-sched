# Self-Hosted Production Deployment Guide

This guide walks you through deploying the Referee Scheduler application on your own server.

## Prerequisites

- Linux server (Ubuntu 22.04+ recommended)
- Docker and Docker Compose installed
- Domain name pointing to your server
- Ports 80 and 443 open in firewall
- Minimum 2GB RAM, 20GB disk space

## Quick Start

### 1. Install Docker

If Docker is not already installed:

```bash
# Update package list
sudo apt update

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add your user to docker group (optional, to run without sudo)
sudo usermod -aG docker $USER

# Log out and back in for group change to take effect
```

### 2. Clone Repository

```bash
git clone <your-repo-url> /opt/referee-scheduler
cd /opt/referee-scheduler
```

### 3. Run Initial Setup

```bash
chmod +x scripts/*.sh
./scripts/initial-setup.sh
```

This script will:
- Create `.env.production` with your configuration
- Generate secure passwords
- Set up directory structure
- Configure nginx

You'll be prompted for:
- Your domain name
- Email address (for SSL certificates)
- Google OAuth credentials

### 4. Configure DNS

Point your domain to your server's IP address:

```
Type: A Record
Name: @  (or your subdomain)
Value: <your-server-ip>
TTL: 3600

Type: A Record  
Name: www
Value: <your-server-ip>
TTL: 3600
```

Wait for DNS propagation (can take up to 48 hours, usually much faster).

Verify with: `dig your-domain.com`

### 5. Obtain SSL Certificates

```bash
./scripts/ssl-setup.sh
```

This will:
- Obtain Let's Encrypt SSL certificates
- Configure automatic renewal
- Set up HTTPS

### 6. Deploy Application

```bash
# Build and start all services
docker-compose -f docker-compose.prod.yml up -d --build

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 7. Verify Deployment

Visit your domain: `https://your-domain.com`

You should see the login page.

## Configuration

### Environment Variables

All configuration is in `.env.production`. Key variables:

| Variable | Description |
|----------|-------------|
| `DOMAIN` | Your domain name |
| `POSTGRES_PASSWORD` | Database password (auto-generated) |
| `SESSION_SECRET` | Session encryption key (auto-generated) |
| `GOOGLE_CLIENT_ID` | Google OAuth Client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth Client Secret |
| `LETSENCRYPT_EMAIL` | Email for SSL certificate notifications |

### Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create a new OAuth 2.0 Client ID
3. Add authorized redirect URI:
   - `https://your-domain.com/api/auth/google/callback`
4. Copy Client ID and Secret to `.env.production`

## Database Management

### Backups

Create a backup:
```bash
./scripts/backup-database.sh
```

Backups are stored in `./backups/` and automatically cleaned up after 30 days.

### Automated Backups with Cron

Set up daily backups at 2 AM:

```bash
crontab -e
```

Add:
```
0 2 * * * cd /opt/referee-scheduler && ./scripts/backup-database.sh >> /var/log/ref-sched-backup.log 2>&1
```

### Restore from Backup

```bash
./scripts/restore-database.sh ./backups/referee_scheduler_20260422_120000.sql.gz
```

## Maintenance

### View Logs

```bash
# All services
docker-compose -f docker-compose.prod.yml logs -f

# Specific service
docker-compose -f docker-compose.prod.yml logs -f backend
docker-compose -f docker-compose.prod.yml logs -f frontend
docker-compose -f docker-compose.prod.yml logs -f nginx
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

# Check migrations ran successfully
docker-compose -f docker-compose.prod.yml logs backend | grep -i migration
```

### SSL Certificate Renewal

Certificates auto-renew via the certbot container. To manually renew:

```bash
docker-compose -f docker-compose.prod.yml run --rm certbot renew
docker-compose -f docker-compose.prod.yml restart nginx
```

## Security Hardening

### Firewall Configuration

Using UFW (Ubuntu Firewall):

```bash
# Install UFW
sudo apt install ufw

# Allow SSH (important - don't lock yourself out!)
sudo ufw allow 22/tcp

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

### Disable Database Port Exposure

The production config does NOT expose PostgreSQL port 5432 to the host. The database is only accessible to backend container via Docker network.

### Regular Updates

```bash
# Update system packages
sudo apt update && sudo apt upgrade -y

# Update Docker images
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d
```

## Monitoring

### Check Container Health

```bash
docker-compose -f docker-compose.prod.yml ps
```

All containers should show "Up" status.

### Database Connection

```bash
docker-compose -f docker-compose.prod.yml exec db psql -U referee_scheduler -d referee_scheduler
```

### Disk Space

```bash
# Check disk usage
df -h

# Check Docker disk usage
docker system df

# Clean up old images
docker system prune -a
```

## Troubleshooting

### Application Won't Start

1. Check logs:
   ```bash
   docker-compose -f docker-compose.prod.yml logs
   ```

2. Verify environment variables:
   ```bash
   cat .env.production
   ```

3. Check database connection:
   ```bash
   docker-compose -f docker-compose.prod.yml exec backend env | grep DATABASE
   ```

### SSL Certificate Issues

1. Verify DNS is correct:
   ```bash
   dig your-domain.com
   ```

2. Check nginx configuration:
   ```bash
   docker-compose -f docker-compose.prod.yml exec nginx nginx -t
   ```

3. View certbot logs:
   ```bash
   docker-compose -f docker-compose.prod.yml logs certbot
   ```

### Google OAuth Not Working

1. Verify redirect URI in Google Console matches:
   ```
   https://your-domain.com/api/auth/google/callback
   ```

2. Check environment variables are loaded:
   ```bash
   docker-compose -f docker-compose.prod.yml exec backend env | grep GOOGLE
   ```

### Database Performance Issues

1. Check connection count:
   ```bash
   docker-compose -f docker-compose.prod.yml exec db \
     psql -U referee_scheduler -c "SELECT count(*) FROM pg_stat_activity;"
   ```

2. Optimize database:
   ```bash
   docker-compose -f docker-compose.prod.yml exec db \
     psql -U referee_scheduler -d referee_scheduler -c "VACUUM ANALYZE;"
   ```

## Scaling Considerations

### Vertical Scaling

Increase resources in Docker Compose:

```yaml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
```

### Database Optimization

For larger deployments, consider:
- Separate database server
- Connection pooling (PgBouncer)
- Read replicas

## Backup Strategy

### Full System Backup

Include in your backup:
1. Database backups (automated via script)
2. `.env.production` (contains secrets)
3. SSL certificates (`certbot/conf/`)
4. Uploaded files (if any)

### Disaster Recovery

1. Keep latest backup offsite
2. Document recovery procedure
3. Test restore process periodically

## Cost Estimation

For a typical deployment:

| Component | Specs | Monthly Cost |
|-----------|-------|--------------|
| VPS | 2GB RAM, 2 CPU, 40GB SSD | $5-15 |
| Domain | Annual registration | $1-2/month |
| Backup Storage | Optional, for offsite | $0-5 |
| **Total** | | **$6-20/month** |

Popular VPS providers:
- DigitalOcean
- Linode
- Vultr
- Hetzner

## Support

For issues:
1. Check logs first
2. Review troubleshooting section
3. Check GitHub issues
4. Review documentation

## Next Steps

After deployment:
1. Create your first admin user
2. Import match schedule
3. Invite referees
4. Set up regular backups
5. Monitor application health
6. Plan for updates

---

**Production Deployment Complete! 🎉**

Your referee scheduler is now self-hosted and ready to use.
