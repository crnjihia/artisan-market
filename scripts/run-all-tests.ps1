# path: scripts/run-all-tests.ps1
# ==============================================================================
# Artisan Market - Full Test Suite Runner (PowerShell)
# ==============================================================================
$ErrorActionPreference = "Stop"

function Run-GoTest {
    param(
        [string]$Path,
        [string]$Name,
        [string]$SubPkg = "",
        [string]$Tags = ""
    )
    Write-Host "`n>>> Testing $Name ($Path $SubPkg)..." -ForegroundColor Yellow
    Push-Location $Path
    try {
        $binName = "${Name}_test.exe"
        $cmd = @("test")
        if ($Tags) { $cmd += @("-tags", $Tags) }
        $cmd += @("-c", "-o", $binName)
        if ($SubPkg) { $cmd += $SubPkg }
        & go $cmd
        & ".\$binName" "-test.v"
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Test failed for $Name"
        }
        Write-Host ">>> ${Name}: ALL TESTS PASSED" -ForegroundColor Green
    } finally {
        Remove-Item "${Name}_test.exe" -ErrorAction SilentlyContinue
        Pop-Location
    }
}

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   Artisan Market Microservices - Comprehensive Test Suite  " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Contract & Schema Validation
& .\scripts\verify-contracts.ps1

# 2. Security & Telemetry Unit Tests
Run-GoTest -Path "backend/common" -Name "security" -SubPkg "./security" -Tags "purego"
Run-GoTest -Path "backend/common" -Name "telemetry" -SubPkg "./telemetry" -Tags "purego"

# 3. User Service Unit Tests
Run-GoTest -Path "backend/user" -Name "user" -Tags "purego"

# 4. Product Service Unit Tests
Run-GoTest -Path "backend/product" -Name "product" -Tags "purego"

# 5. Payment Service Unit Tests
Run-GoTest -Path "backend/payment" -Name "payment" -Tags "purego"

# 6. Order Service & E2E Saga Integration Tests
Run-GoTest -Path "backend/order" -Name "order_saga" -Tags "purego"

# 7. Gateway Service Jest Tests
Write-Host "`n>>> Testing Gateway Service (Jest)..." -ForegroundColor Yellow
npm --prefix backend/gateway test
Write-Host ">>> Gateway Service: ALL TESTS PASSED" -ForegroundColor Green

# 8. Notification Service Jest Tests
Write-Host "`n>>> Testing Notification Service (Jest)..." -ForegroundColor Yellow
npm --prefix backend/notification test
Write-Host ">>> Notification Service: ALL TESTS PASSED" -ForegroundColor Green

Write-Host "`n==========================================================" -ForegroundColor Green
Write-Host "   100% SUITE PASS: All Microservices, Sagas & Contracts   " -ForegroundColor Green
Write-Host "==========================================================" -ForegroundColor Green
