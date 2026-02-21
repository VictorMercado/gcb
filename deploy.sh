#!/bin/bash
# Google Cloud Bucket service deploy script
LOG_FILE="/etc/gcb/deploy.log"
ENV_FILE="/etc/gcb/.env"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

WORK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P)

if [ ! -f "$LOG_FILE" ]; then
    touch "$LOG_FILE"
    echo "Created: $LOG_FILE"
else
    echo "LOG File already exists."
fi
echo "$(date) - Deployment started" >> $LOG_FILE
set -e  # Exit on any error

# 2. Verify env file exists at /etc/gcb/.env
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}⚠️  No .env file found at $ENV_FILE${NC}"
    echo "$(date) - Deployment failed: No .env file found at $ENV_FILE" >> $LOG_FILE
    exit 1
fi
echo -e "${GREEN}✅ Env file found at $ENV_FILE${NC}"

echo -e "${GREEN}🚀 Starting Google Cloud Bucket service deployment in $WORK_DIR...${NC}"

echo -e "${YELLOW}📥 Pulling latest changes...${NC}"

git pull --rebase origin main

echo -e "${YELLOW}🔨 Building and starting containers...${NC}"

# Removed 'down' to prevent downtime during build
if [[ "$1" == "--no-cache" ]]; then
    docker compose up -d --build --no-cache --remove-orphans
else
    docker compose up -d --build --remove-orphans
fi

# Step 5: Cleanup - This is crucial for your disk space
echo -e "${YELLOW}🧹 Cleaning up old images...${NC}"
docker image prune -f

# Log deployment timestamp
echo "$(date) - Deployment complete" >> $LOG_FILE

echo -e "${GREEN}✅ Deployment complete!${NC}"
docker compose ps

# Step 6: Show recent logs
echo -e "${YELLOW}📋 Recent logs:${NC}"
docker compose logs --tail=20

echo -e "${GREEN}✅ Deployment complete!${NC}"
echo -e "View full logs with: ${YELLOW}docker compose logs -f${NC}"
