#!/bin/bash
# Setup Script for Traefik + Cloudflare Deployment
# This script helps configure the application for Traefik reverse proxy

set -e

echo "═══════════════════════════════════════════════════════"
echo "  Referee Scheduler - Traefik Production Setup"
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

# Check if Traefik network exists
if docker network ls | grep -q "traefik_default"; then
    echo "✓ Traefik network found"
else
    echo "⚠ Traefik network not found"
    read -p "Create traefik_default network? (yes/no): " create_network
    if [ "$create_network" = "yes" ]; then
        docker network create traefik_default
        echo "✓ Network created"
    else
        echo "Please create the network manually or update docker-compose.prod.yml"
    fi
fi

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
    read -p "Enter your domain name (e.g., ref-sched.example.com): " domain

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

# Domain Configuration (used by Traefik routing)
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

# Backup Configuration
BACKUP_RETENTION_DAYS=30
EOF

    echo "✓ .env.production created"
fi

# Load environment
export $(cat .env.production | grep -v '^#' | xargs)

# Create necessary directories
echo ""
echo "Creating directories..."
mkdir -p backups
echo "✓ Directories created"

# Make scripts executable
echo ""
echo "Setting script permissions..."
chmod +x scripts/*.sh
echo "✓ Scripts are now executable"

echo ""
echo "═══════════════════════════════════════════════════════"
echo "  Setup Complete!"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "Next steps:"
echo ""
echo "1. Verify .env.production has correct values:"
echo "   cat .env.production"
echo ""
echo "2. Configure Cloudflare Tunnel for domain: $DOMAIN"
echo "   Point to: http://traefik:80"
echo ""
echo "3. Add Google OAuth redirect URI:"
echo "   https://$DOMAIN/api/auth/google/callback"
echo ""
echo "4. Deploy the application:"
echo "   docker-compose -f docker-compose.prod.yml up -d --build"
echo ""
echo "5. Set up automated backups:"
echo "   crontab -e"
echo "   Add: 0 2 * * * cd $(pwd) && ./scripts/backup-database.sh"
echo ""
echo "Configuration summary:"
echo "  Domain: $DOMAIN"
echo "  Database: referee_scheduler"
echo "  Network: traefik_default"
echo ""
