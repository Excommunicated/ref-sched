#!/bin/bash
# Database Backup Script for Referee Scheduler
# This script creates a PostgreSQL dump and manages backup retention

set -e

# Load environment variables
if [ -f .env.production ]; then
    export $(cat .env.production | grep -v '^#' | xargs)
fi

# Configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="referee_scheduler_${TIMESTAMP}.sql.gz"
CONTAINER_NAME="referee-scheduler-db-prod"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

echo "Starting database backup..."
echo "Backup file: $BACKUP_FILE"

# Create backup using docker exec
docker exec "$CONTAINER_NAME" pg_dump \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" \
    --clean --if-exists \
    | gzip > "$BACKUP_DIR/$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "✓ Backup completed successfully"
    echo "  Location: $BACKUP_DIR/$BACKUP_FILE"
    echo "  Size: $(du -h "$BACKUP_DIR/$BACKUP_FILE" | cut -f1)"
else
    echo "✗ Backup failed!"
    exit 1
fi

# Clean up old backups
echo ""
echo "Cleaning up backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "referee_scheduler_*.sql.gz" -type f -mtime +$RETENTION_DAYS -delete
echo "✓ Cleanup complete"

# List recent backups
echo ""
echo "Recent backups:"
ls -lh "$BACKUP_DIR"/referee_scheduler_*.sql.gz | tail -5

echo ""
echo "Backup process complete!"
