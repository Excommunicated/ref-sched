#!/bin/bash
# SSL Certificate Setup Script using Let's Encrypt
# This script obtains SSL certificates for your domain

set -e

# Load environment variables
if [ ! -f .env.production ]; then
    echo "Error: .env.production not found!"
    echo "Run ./scripts/initial-setup.sh first"
    exit 1
fi

export $(cat .env.production | grep -v '^#' | xargs)

echo "═══════════════════════════════════════════════════════"
echo "  SSL Certificate Setup"
echo "═══════════════════════════════════════════════════════"
echo ""
echo "Domain: $DOMAIN"
echo "Email: $LETSENCRYPT_EMAIL"
echo ""

# Check if certificates already exist
if [ -d "certbot/conf/live/$DOMAIN" ]; then
    echo "Certificates already exist for $DOMAIN"
    read -p "Do you want to renew them? (yes/no): " renew
    if [ "$renew" != "yes" ]; then
        echo "Skipping certificate generation"
        exit 0
    fi
    RENEW_FLAG="--force-renewal"
else
    RENEW_FLAG=""
fi

# Create temporary nginx config for Let's Encrypt challenge
echo "Setting up temporary nginx configuration..."
cat > nginx/conf.d/letsencrypt.conf <<EOF
server {
    listen 80;
    server_name $DOMAIN www.$DOMAIN;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 404;
    }
}
EOF

# Start nginx temporarily for challenge
echo "Starting nginx for Let's Encrypt challenge..."
docker-compose -f docker-compose.prod.yml up -d nginx

# Wait for nginx to start
sleep 3

# Obtain certificate
echo ""
echo "Obtaining SSL certificate..."
docker-compose -f docker-compose.prod.yml run --rm certbot certonly \
    --webroot \
    --webroot-path=/var/www/certbot \
    --email "$LETSENCRYPT_EMAIL" \
    --agree-tos \
    --no-eff-email \
    $RENEW_FLAG \
    -d "$DOMAIN" \
    -d "www.$DOMAIN"

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ SSL certificates obtained successfully!"
    echo ""
    echo "Certificates are stored in: certbot/conf/live/$DOMAIN/"

    # Remove temporary config
    rm nginx/conf.d/letsencrypt.conf

    # Restart nginx with production config
    echo "Restarting nginx with production configuration..."
    docker-compose -f docker-compose.prod.yml restart nginx

    echo ""
    echo "✓ SSL setup complete!"
    echo ""
    echo "Your site should now be accessible at: https://$DOMAIN"
else
    echo ""
    echo "✗ Failed to obtain SSL certificates"
    echo ""
    echo "Troubleshooting:"
    echo "1. Verify DNS is pointing to this server"
    echo "2. Ensure ports 80 and 443 are open in firewall"
    echo "3. Check nginx logs: docker-compose -f docker-compose.prod.yml logs nginx"
    exit 1
fi
