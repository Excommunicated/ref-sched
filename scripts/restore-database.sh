#!/bin/bash
# Database Restore Script for Referee Scheduler
# Usage: ./restore-database.sh <backup-file>

set -e

# Check arguments
if [ $# -eq 0 ]; then
    echo "Usage: $0 <backup-file>"
    echo ""
    echo "Example: $0 ./backups/referee_scheduler_20260422_120000.sql.gz"
    echo ""
    echo "Available backups:"
    ls -1 ./backups/referee_scheduler_*.sql.gz 2>/dev/null || echo "  No backups found"
    exit 1
fi

BACKUP_FILE="$1"

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Load environment variables
if [ -f .env.production ]; then
    export $(cat .env.production | grep -v '^#' | xargs)
fi

CONTAINER_NAME="referee-scheduler-db-prod"

echo "WARNING: This will restore the database from backup!"
echo "Backup file: $BACKUP_FILE"
echo "Database: ${POSTGRES_DB}"
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Restore cancelled."
    exit 0
fi

echo ""
echo "Starting database restore..."

# Restore backup
gunzip -c "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}"

if [ $? -eq 0 ]; then
    echo "✓ Database restored successfully"
else
    echo "✗ Restore failed!"
    exit 1
fi

echo ""
echo "Restore process complete!"
