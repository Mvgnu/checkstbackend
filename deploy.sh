#!/bin/bash
# OpenAnki Backend Deploy Script
# Run this on the VPS to pull latest changes and restart the server

set -e

# Configuration
APP_DIR="/home/ploi/checkst.app/checkstbackend"
SERVICE_NAME="openanki"
REPO_URL="https://github.com/Mvgnu/checkstbackend.git"
BRANCH="main"

echo "=== OpenAnki Backend Deploy ==="
echo "Time: $(date)"

# Navigate to app directory
cd "$APP_DIR"

# Pull latest changes
echo "[1/4] Pulling latest changes..."
if [ -d ".git" ]; then
    git fetch origin "$BRANCH"
    git reset --hard "origin/$BRANCH"
else
    echo "Cloning repository..."
    git clone "$REPO_URL" .
fi

# Build the server
echo "[2/4] Building server..."
CGO_ENABLED=1 go build -o server ./cmd/server

# Ensure data directory exists
mkdir -p data

# Restart the service
echo "[3/4] Restarting service..."
sudo systemctl restart "$SERVICE_NAME" || {
    echo "Service not found. Starting directly..."
    ./server &
}

# Health check
echo "[4/4] Checking health..."
sleep 2
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✓ Server is healthy!"
else
    echo "⚠ Health check failed. Check logs with: journalctl -u $SERVICE_NAME -f"
fi

echo "=== Deploy Complete ==="
