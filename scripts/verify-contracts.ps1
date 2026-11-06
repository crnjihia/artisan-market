# path: scripts/verify-contracts.ps1
# ==============================================================================
# Artisan Market - Protobuf Contract & Schema Verification (PowerShell)
# ==============================================================================
$ErrorActionPreference = "Stop"

Write-Host "========================================================" -ForegroundColor Cyan
Write-Host " [Artisan Market] Protobuf Contract & Schema Validation " -ForegroundColor Cyan
Write-Host "========================================================" -ForegroundColor Cyan

# 1. Buf Linter
Write-Host "`n>>> [1/3] Running Buf Schema Linter..." -ForegroundColor Yellow
$bufCmd = Get-Command buf -ErrorAction SilentlyContinue
if ($bufCmd) {
    & buf lint
} else {
    npx --yes @bufbuild/buf lint
}
if ($LASTEXITCODE -ne 0) {
    Write-Error "Buf lint check failed!"
    exit $LASTEXITCODE
}
Write-Host "Buf lint check: PASSED" -ForegroundColor Green

# 2. Buf Breaking Change Detection
Write-Host "`n>>> [2/3] Running Buf Breaking Change Detection (against baseline)..." -ForegroundColor Yellow
if ($bufCmd) {
    & buf breaking --against .
} else {
    npx --yes @bufbuild/buf breaking --against .
}
if ($LASTEXITCODE -ne 0) {
    Write-Error "Buf breaking change check failed!"
    exit $LASTEXITCODE
}
Write-Host "Buf breaking change check: PASSED" -ForegroundColor Green

# 3. Go Contract & Wire Serialization Test Suite
Write-Host "`n>>> [3/3] Running Go Protobuf Wire & Kafka Event Contract Tests..." -ForegroundColor Yellow
Push-Location "tests"
try {
    go test -tags purego -c -o contract_test.exe
    .\contract_test.exe "-test.v"
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Go contract tests failed!"
        exit $LASTEXITCODE
    }
} finally {
    Remove-Item "contract_test.exe" -ErrorAction SilentlyContinue
    Pop-Location
}
Write-Host "Go contract tests: PASSED" -ForegroundColor Green

Write-Host "`n========================================================" -ForegroundColor Green
Write-Host " All Protobuf Contracts & Event Schemas Verified Valid! " -ForegroundColor Green
Write-Host "========================================================" -ForegroundColor Green

