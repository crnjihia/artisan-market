# Contributing to Artisan Market

Thank you for your interest in contributing to **Artisan Market**! This document provides guidelines and instructions for setting up your local environment, developing across Go and Node.js microservices, and running our automated test suites.

---

## 1. Prerequisites

Ensure you have the following installed on your workstation:
- **Go**: `1.24+` (managed via `go.work` multi-module workspace)
- **Node.js**: `20.x+` and `npm 10.x+`
- **Docker & Docker Compose**: `v2.20+`
- **Buf CLI**: Installed globally or executed via `npx --yes @bufbuild/buf`
- **Git**

---

## 2. Quick Local Setup

```bash
# 1. Clone the repository
git clone https://github.com/yourorg/artisan-market.git
cd artisan-market

# 2. Configure environment variables
cp .env.example .env

# 3. Start auxiliary infrastructure (Kafka/Redpanda, Multi-DB Postgres, Redis, Jaeger, Mailpit, MinIO)
make up

# 4. Generate local mutual TLS (mTLS) development certificates
go run scripts/generate-certs.go
```

Auxiliary services will be available at:
- **Jaeger Tracing UI**: [http://localhost:16686](http://localhost:16686)
- **Mailpit Email UI**: [http://localhost:8025](http://localhost:8025)
- **MinIO S3 Console**: [http://localhost:9001](http://localhost:9001)
- **Redpanda Console**: [http://localhost:8081](http://localhost:8081)

---

## 3. Architecture & Repository Structure

Artisan Market is organized as an enterprise polyglot monorepo:

```text
├── backend/
│   ├── common/         # Shared mTLS security, OpenTelemetry tracing & RED metrics
│   ├── gateway/        # Fastify REST-to-gRPC API gateway with JWT & rate limiting
│   ├── user/           # User authentication, Argon2id, rotating refresh tokens (Go)
│   ├── product/        # Catalogue CRUD, row-level locks, oversell prevention (Go)
│   ├── order/          # Order aggregate, Transactional Outbox, Saga Coordinator (Go)
│   ├── payment/        # Safaricom M-Pesa Daraja STK Push & Stripe webhooks (Go)
│   └── notification/   # Bilingual SMS (Africa's Talking) & Email worker (Node.js)
├── frontend/           # Next.js 14 App Router e-commerce storefront
├── proto/              # Protobuf v3 service contracts (Buf v2)
├── deploy/k8s/         # Kubernetes manifests, Istio ingress, STRICT mTLS & canary
├── scripts/            # Database init, cert generators, contract & test runners
└── tests/              # Cross-service Protobuf wire & Kafka event contract tests
```

---

## 4. Development Workflow

### Working with Protobuf Contracts
Whenever updating `.proto` definitions in `proto/`:
```bash
# Lint Protobuf schemas
npx @bufbuild/buf lint

# Check for breaking API changes against baseline
npx @bufbuild/buf breaking --against .

# Generate Go stubs and client packages
npx @bufbuild/buf generate
```

### Zero-Credential Offline Development
When developing offline or without live API credentials for Safaricom or Africa's Talking:
- Keep `USE_MOCKS=true` in `.env`.
- Payment Service simulates Safaricom STK prompt acceptance after 500ms and issues an automated callback.
- Notification Service logs mock SMS to the console and routes mock emails cleanly.

---

## 5. Running Tests

We maintain a strict 100% test pass policy across all services before merging pull requests.

### Comprehensive Test Runner
Run all unit, integration, saga, contract, and Jest test suites in one command:

```powershell
# Windows PowerShell
powershell -ExecutionPolicy Bypass -File .\scripts\run-all-tests.ps1
```

```bash
# Linux / macOS
bash scripts/verify-contracts.sh
make test
```

### Windows WDAC / PureGo Flag Note
On Windows systems with AppLocker or Windows Defender Application Control (WDAC), temporary binary execution in `AppData\Local\Temp` is restricted. Tests should be compiled locally using:
```powershell
go test -tags purego -c -o service_test.exe
.\service_test.exe "-test.v"
Remove-Item "service_test.exe"
```

---

## 6. Coding Standards & Conventions

### Go Services
- Adhere to `go vet ./...` and `gofmt -s -w`.
- Maintain strict database transaction boundaries (`BeginTx` -> `defer tx.Rollback()` -> `tx.Commit()`).
- Always queue domain events via the **Transactional Outbox** pattern; never perform dual writes.

### TypeScript / Node.js
- Enable TypeScript `strict: true`.
- Zero `any` types where possible; use explicit domain interfaces.
- ESLint and Prettier must pass with 0 errors.

### Commit Messages
Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:
- `feat(scope): add feature description`
- `fix(scope): fix bug description`
- `test(scope): add test cases`
- `docs(scope): update documentation`
- `refactor(scope): refactor code without behavior change`

---

## 7. Submitting Pull Requests

1. Create a feature branch: `git checkout -b feat/my-new-feature`.
2. Commit your changes with conventional messages.
3. Verify that `powershell -File .\scripts\run-all-tests.ps1` passes with 100% success.
4. Push your branch: `git push origin feat/my-new-feature`.
5. Open a Pull Request detailing the problem solved, design decisions made, and screenshots/logs of verification.
