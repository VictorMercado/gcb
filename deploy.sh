#!/bin/bash


set -e  # Exit on any error

echo "🛑 Stopping containers..."
docker compose down

echo "📥 Pulling latest changes..."
git pull

echo "� Copying .env file..."
cp /opt/gcb/.env .

echo "�🚀 Starting containers..."
docker compose up -d --build

echo "✅ Redeployment complete!"
