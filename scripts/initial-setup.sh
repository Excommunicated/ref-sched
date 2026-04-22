#!/bin/bash
# Initial Setup Script for Self-Hosted Deployment
# This script helps configure the application for first-time deployment

set -e

echo "═══════════════════════════════════════════════════════"
echo "  Referee Scheduler - Production Setup"
echo "═══════════════════════════════════════════════════════"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo "Warning: Running as root. It's recommended to run as a regular user with sudo access."
    echo ""
fi

# Check for required commands
echo "Checking prerequisites..."
command -v docker >/dev/null 2>&1 || { echo "Error: docker is not installed"; exit 1; }
command -v docker-compose >/dev/null 2>&1 || command -v docker compose >/dev/null 2>&1 || { echo "Error: docker-compose is not installed"; exit 1; }
echo "✓ Docker is installed"

# Check if .env.production exists
if [ -f .env.production ]; then
    echo ""
    echo "Warning: .env.production already exists!"
    read -p "Do you want to overwrite it? (yes/no): " overwrite
    if [ "$overwrite" != "yes" ]; then
        echo "Using existing .env.production"
    else
        rm .env.production
    fi
fi

# Create .env.production if it doesn't exist
if [ ! -f .env.production ]; then
    echo ""
    echo "Creating .env.production..."

    # Prompt for domain
    read -p "Enter your domain name (e.g., ref-scheduler.example.com): " domain

    # Prompt for email
    read -p "Enter your email for Let's Encrypt SSL: " email

    # Generate secure passwords
    echo ""
    echo "Generating secure passwords..."
    db_password=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-32)
    session_secret=$(openssl rand -base64 32)

    # Prompt for Google OAuth
    echo ""
    echo "Google OAuth Configuration:"
    echo "(Get these from: https://console.cloud.google.com/apis/credentials)"
    read -p "Google Client ID: " google_client_id
    read -p "Google Client Secret: " google_client_secret

    # Create .env.production file
    cat > .env.production <<EOF
# Production Environment Configuration
# Generated on $(date)

# Domain Configuration
DOMAIN=$domain
FRONTEND_URL=https://$domain
VITE_API_URL=https://$domain/api

# Database Configuration
POSTGRES_USER=referee_scheduler
POSTGRES_PASSWORD=$db_password
POSTGRES_DB=referee_scheduler

# Session Secret
SESSION_SECRET=$session_secret

# Google OAuth2 Configuration
GOOGLE_CLIENT_ID=$google_client_id
GOOGLE_CLIENT_SECRET=$google_client_secret
GOOGLE_REDIRECT_URL=https://$domain/api/auth/google/callback

# Email for Let's Encrypt SSL certificates
LETSENCRYPT_EMAIL=$email

# Backup Configuration
BACKUP_RETENTION_DAYS=30
BACKUP_PATH=/backups
EOF

    echo "✓ .env.production created"
fi

# Load environment
export $(cat .env.production | grep -v '^#' | xargs)

# Create necessary directories
echo ""
echo "Creating directories..."
mkdir -p nginx/conf.d
mkdir -p certbot/conf
mkdir -p certbot/www
mkdir -p backups
echo "✓ Directories created"

# Create nginx config from template
echo ""
echo "Configuring nginx..."
if [ -f nginx/conf.d/ref-sched.conf ]; then
    echo "  nginx config already exists, skipping..."
else
    sed "s/DOMAIN/$DOMAIN/g" nginx/conf.d/ref-sched.conf.template > nginx/conf.d/ref-sched.conf
    echo "✓ nginx configuration created"
fi

# Make scripts executable
echo ""
echo "Setting script permissions..."
chmod +x scripts/*.sh
echo "✓ Scripts are now executable"

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  Initial Setup Complete!"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "Next steps:"
echo ""
echo "1. Verify .env.production has correct values"
echo "2. Configure your domain's DNS to point to this server"
echo "3. Run: ./scripts/ssl-setup.sh to obtain SSL certificates"
echo "4. Run: docker-compose -f docker-compose.prod.yml up -d"
echo "5. Run: ./scripts/backup-database.sh (set up as a cron job)"
echo ""
echo "Configuration summary:"
echo "  Domain: $DOMAIN"
echo "  Email: $LETSENCRYPT_EMAIL"
echo "  Database: referee_scheduler"
echo ""
