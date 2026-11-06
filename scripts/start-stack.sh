#!/usr/bin/env bash
set -e

# Bring up the docker compose stack in detached mode
docker compose up -d

# Wait for services to become healthy
MAX_WAIT=30
INTERVAL=3
elapsed=0
while true; do
  if docker compose ps postgres 2>/dev/null | grep -E "healthy|Up" >/dev/null 2>&1; then
    echo "Backing services are healthy"
    break
  fi
  if [[ $elapsed -ge $MAX_WAIT ]]; then
    echo "Proceeding with stack tests..."
    break
  fi
  echo "Waiting for services... ($elapsed s)"
  sleep $INTERVAL
  elapsed=$((elapsed + INTERVAL))
done

# Install dependencies and build storefront
npm --prefix frontend install
npm --prefix frontend run build

# Launch storefront server in background
PORT=3050 npm --prefix frontend start &
FRONTEND_PID=$!

# Wait for storefront readiness
sleep 5

# Install browser binaries if needed
npx playwright install chromium 2>/dev/null || true

# Run Playwright E2E suite
BASE_URL="http://localhost:3050" npm --prefix frontend run test:e2e

# Cleanup background server
kill $FRONTEND_PID 2>/dev/null || true

