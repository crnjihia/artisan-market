#!/usr/bin/env bash
set -e

# Bring up the docker compose stack in detached mode
docker compose up -d

# Wait for services to become healthy (simple retry loop)
MAX_WAIT=60
INTERVAL=5
elapsed=0
while true; do
  # Check gateway health (requires API key header)
  if curl -ks "https://localhost:${GATEWAY_PORT:-3000}/api/health" -H "X-API-KEY: ${GATEWAY_API_KEY:-dev-secret-key}" >/dev/null 2>&1; then
    echo "Gateway is healthy"
    break
  fi
  if [[ $elapsed -ge $MAX_WAIT ]]; then
    echo "Timed out waiting for services"
    exit 1
  fi
  echo "Waiting for services... ($elapsed s)"
  sleep $INTERVAL
  elapsed=$((elapsed + INTERVAL))
done

# Give a little extra time for the notification consumer to start
sleep 5

# Run front‑end build (required for Playwright tests that use the built UI)
npm --prefix frontend install
npm --prefix frontend run build

# Install Playwright browsers
npx --yes playwright install

# Run Playwright tests
npm --prefix frontend run test:e2e

# If we got here, tests passed – keep containers running for debugging (optional)
#docker compose logs -f
