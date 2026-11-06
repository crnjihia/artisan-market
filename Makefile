# ==============================================================================
# Artisan Market - Makefile
# ==============================================================================
.PHONY: up down status buf-lint buf-gen test lint clean seed help

SHELL := /bin/bash
BUF ?= $(shell which buf 2>/dev/null || echo "npx --yes @bufbuild/buf")

help: ## Display available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

up: ## Start all auxiliary services (Redpanda, Postgres, Redis, Jaeger, Mailpit, MinIO)
	docker compose up -d
	@echo "Services started. Jaeger at http://localhost:16686, Mailpit at http://localhost:8025, MinIO at http://localhost:9001"

down: ## Stop and remove auxiliary containers
	docker compose down -v

status: ## Show running container health
	docker compose ps

buf-lint: ## Run Buf linter on Protobuf definitions
	$(BUF) lint

buf-gen: ## Generate Go and client stubs from Protobuf
	$(BUF) generate

test: ## Run unit tests across all Go and Node workspaces
	@echo "=== Running Go Microservice Tests ==="
	cd backend/user && go test -v ./... -count=1
	cd backend/product && go test -v ./... -count=1
	cd backend/order && go test -v ./... -count=1
	cd backend/payment && go test -v ./... -count=1
	@echo "=== Running Node Tests ==="
	npm --prefix backend/gateway test
	npm --prefix backend/notification test

test-saga: ## Run Distributed Saga E2E Integration and Chaos tests
	@echo "=== Running Distributed Saga E2E Tests ==="
	cd backend/order && go test -v -run "TestSagaE2E_" -count=1

test-contracts: ## Verify Protobuf contracts and breaking change detection
	bash scripts/verify-contracts.sh

lint: ## Run Go vet and TypeScript checks
	@echo "=== Linting Go Services ==="
	cd backend/user && go vet ./...
	cd backend/product && go vet ./...
	cd backend/order && go vet ./...
	cd backend/payment && go vet ./...
	@echo "=== Typechecking TypeScript Services ==="
	npm --prefix backend/gateway run build
	npm --prefix backend/notification run build

clean: ## Clean built binaries and temporary test artifacts
	rm -rf bin/ dist/ backend/*/dist frontend/.next
