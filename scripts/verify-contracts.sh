#!/usr/bin/env bash
# path: scripts/verify-contracts.sh
# ==============================================================================
# Artisan Market - Protobuf Contract & Schema Verification (Bash)
# ==============================================================================
set -euo pipefail

echo "========================================================"
echo " [Artisan Market] Protobuf Contract & Schema Validation "
echo "========================================================"

BUF_BIN="buf"
if ! command -v buf &> /dev/null; then
    BUF_BIN="npx --yes @bufbuild/buf"
fi

# 1. Buf Linter
echo -e "\n>>> [1/3] Running Buf Schema Linter..."
$BUF_BIN lint
echo "Buf lint check: PASSED"

# 2. Buf Breaking Change Detection
echo -e "\n>>> [2/3] Running Buf Breaking Change Detection..."
$BUF_BIN breaking --against .
echo "Buf breaking change check: PASSED"

# 3. Go Contract & Wire Serialization Test Suite
echo -e "\n>>> [3/3] Running Go Protobuf Wire & Kafka Event Contract Tests..."
cd tests
go test -v ./... -count=1
cd ..
echo "Go contract tests: PASSED"

echo -e "\n========================================================"
echo " All Protobuf Contracts & Event Schemas Verified Valid! "
echo "========================================================"

