#!/bin/bash

# Redeploy script: tears down docker compose, pulls latest changes, and starts containers again

set -e  # Exit on any error

echo "🛑 Stopping containers..."
docker compose down

echo "📥 Pulling latest changes..."
git pull

echo "🚀 Starting containers..."
docker compose up -d --build

echo "✅ Redeployment complete!"
