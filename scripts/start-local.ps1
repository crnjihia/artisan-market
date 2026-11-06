# path: scripts/start-local.ps1
# ==============================================================================
# Artisan Market - Local Development Stack Launcher (PowerShell)
# ==============================================================================
$ErrorActionPreference = "Stop"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "       Artisan Market - Local Stack Launcher              " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Ensure Docker is running
Write-Host "`n>>> [1/4] Checking Docker status..." -ForegroundColor Yellow
try {
    $version = docker info --format '{{.ServerVersion}}'
    Write-Host "Docker is running (v$version)" -ForegroundColor Green
} catch {
    Write-Error "Docker Desktop is not running. Please open Docker Desktop and try again."
    exit 1
}

# 2. Ensure auxiliary containers are up
Write-Host "`n>>> [2/4] Starting Docker Compose backing services..." -ForegroundColor Yellow
docker compose up -d
docker compose ps

# 3. Ensure dev certs exist
Write-Host "`n>>> [3/4] Checking development mTLS certificates..." -ForegroundColor Yellow
if (-not (Test-Path "certs/ca.crt")) {
    Write-Host "Generating development mTLS certificates..." -ForegroundColor Yellow
    go run scripts/generate-certs.go
} else {
    Write-Host "mTLS certificates are present." -ForegroundColor Green
}

# 4. Instructions for Microservices & Storefront
Write-Host "`n==========================================================" -ForegroundColor Green
Write-Host " Backing services are LIVE! Ready to start applications: " -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Green

Write-Host "`nTo start the platform services, run each in a new terminal tab:" -ForegroundColor White
Write-Host "  Tab 1 (User Service):     " -NoNewline; Write-Host "cd backend/user; go run ." -ForegroundColor Cyan
Write-Host "  Tab 2 (Product Service):  " -NoNewline; Write-Host "cd backend/product; go run ." -ForegroundColor Cyan
Write-Host "  Tab 3 (Order Service):    " -NoNewline; Write-Host "cd backend/order; go run ." -ForegroundColor Cyan
Write-Host "  Tab 4 (Payment Service):  " -NoNewline; Write-Host "cd backend/payment; go run ." -ForegroundColor Cyan
Write-Host "  Tab 5 (Notification):     " -NoNewline; Write-Host "npm --prefix backend/notification run dev" -ForegroundColor Cyan
Write-Host "  Tab 6 (API Gateway):      " -NoNewline; Write-Host "npm --prefix backend/gateway run dev" -ForegroundColor Cyan
Write-Host "  Tab 7 (Storefront):       " -NoNewline; Write-Host "cd frontend; npm run dev" -ForegroundColor Cyan

Write-Host "`nDeveloper Web Consoles:" -ForegroundColor White
Write-Host "  Storefront UI:   http://localhost:3050" -ForegroundColor Yellow
Write-Host "  API Gateway:     http://localhost:4000/api/health" -ForegroundColor Yellow
Write-Host "  Mailpit Emails:  http://localhost:8025" -ForegroundColor Yellow
Write-Host "  Jaeger Tracing:  http://localhost:16686" -ForegroundColor Yellow
Write-Host "  Redpanda Kafka:  http://localhost:8081" -ForegroundColor Yellow

