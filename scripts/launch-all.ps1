# path: scripts/launch-all.ps1
# ==============================================================================
# Artisan Market - One-Click Service Launcher for Windows
# Launches all microservices and Next.js frontend in separate PowerShell tabs/windows
# ==============================================================================
$ErrorActionPreference = "Stop"

$rootDir = Split-Path -Parent $PSScriptRoot
Set-Location $rootDir

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "       Artisan Market - Multi-Service Launcher            " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Check Docker Containers
Write-Host "`n>>> [1/3] Verifying Docker backing services..." -ForegroundColor Yellow
$containers = docker compose ps --format '{{.Service}}: {{.Status}}'
if (-not $containers) {
    Write-Host "Starting Docker containers..." -ForegroundColor Yellow
    docker compose up -d
} else {
    Write-Host "Backing containers are active." -ForegroundColor Green
}

# 2. Check Certificates
Write-Host "`n>>> [2/3] Checking TLS certificates..." -ForegroundColor Yellow
if (-not (Test-Path "certs/ca.crt")) {
    go run scripts/generate-certs.go
} else {
    Write-Host "Certificates are verified." -ForegroundColor Green
}

# 3. Launch Services
Write-Host "`n>>> [3/3] Launching platform services..." -ForegroundColor Yellow

$services = @(
    @{ Name = "Artisan Storefront"; Path = "frontend"; Command = "npm run dev" },
    @{ Name = "Artisan Gateway"; Path = "backend/gateway"; Command = "npm run dev" },
    @{ Name = "User Service"; Path = "backend/user"; Command = "go run ." },
    @{ Name = "Product Service"; Path = "backend/product"; Command = "go run ." },
    @{ Name = "Order Service"; Path = "backend/order"; Command = "go run ." },
    @{ Name = "Payment Service"; Path = "backend/payment"; Command = "go run ." },
    @{ Name = "Notification Service"; Path = "backend/notification"; Command = "npm run dev" }
)

$hasWT = (Get-Command wt -ErrorAction SilentlyContinue) -ne $null

if ($hasWT) {
    Write-Host "Windows Terminal detected. Launching tabbed workspace..." -ForegroundColor Green
    $wtArgs = @()
    for ($i = 0; $i -lt $services.Count; $i++) {
        $svc = $services[$i]
        $fullPath = Join-Path $rootDir $svc.Path
        if ($i -eq 0) {
            $wtArgs += "-d `"$fullPath`" --title `"$($svc.Name)`" powershell -NoExit -Command `"$($svc.Command)`""
        } else {
            $wtArgs += "; new-tab -d `"$fullPath`" --title `"$($svc.Name)`" powershell -NoExit -Command `"$($svc.Command)`""
        }
    }
    $joinedArgs = $wtArgs -join " "
    Start-Process wt -ArgumentList $joinedArgs
} else {
    Write-Host "Launching services in separate PowerShell windows..." -ForegroundColor Green
    foreach ($svc in $services) {
        $fullPath = Join-Path $rootDir $svc.Path
        Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd `"$fullPath`"; Host.UI.RawUI.WindowTitle = '$($svc.Name)'; $($svc.Command)"
    }
}

Write-Host "`n==========================================================" -ForegroundColor Green
Write-Host " All services have been launched!                         " -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Green
Write-Host "  Storefront:     http://localhost:3050" -ForegroundColor Cyan
Write-Host "  API Gateway:    http://localhost:4000/api/health" -ForegroundColor Cyan
Write-Host "  Mailpit Emails: http://localhost:8025" -ForegroundColor Cyan
Write-Host "  Jaeger Tracing: http://localhost:16686" -ForegroundColor Cyan
Write-Host "  Redpanda Kafka: http://localhost:8081" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Green

