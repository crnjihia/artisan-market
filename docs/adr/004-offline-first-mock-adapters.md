# path: docs/adr/004-offline-first-mock-adapters.md
# ADR 004: Zero-Credential Offline-First Mock Adapters for Local Development & CI

- **Status**: Accepted
- **Date**: 2026-11-06
- **Author**: Christopher Njihia (Nairobi, Kenya)
- **Scope**: Payment Service (Safaricom Daraja, Stripe) & Notification Service (Africa's Talking, Mailpit)

---

## 1. Context and Problem Statement

Artisan Market deeply integrates with African and global financial and communication infrastructure:
- **Safaricom Daraja API**: STK Push (`Lipa na M-Pesa Online`) requiring OAuth credentials, Passkey, shortcode, and a publicly reachable HTTPS callback URL.
- **Africa's Talking**: SMS delivery across Kenyan networks (Safaricom, Airtel, Telkom) requiring username and API key.
- **Stripe**: Card processing requiring secret keys and webhook endpoints.
- **SMTP**: Email delivery requiring transactional email credentials.

In onboarding scenarios, continuous integration (CI) runners, and offline developer environments (e.g., developing on a laptop during power or ISP outages), developers cannot always supply live third-party API credentials or expose local webhooks to the public internet via reverse proxies (e.g., ngrok).

If services fail to start or crash when credentials are absent, local development velocity drops to zero and automated CI test suites become brittle.

---

## 2. Decision Drivers

- **Zero-Friction Developer Experience**: A developer should clone the repo, run `make up` and `powershell -File .\scripts\run-all-tests.ps1`, and have 100% of tests and services pass without entering a single API key or credit card.
- **High-Fidelity Simulation**: The mock mode must not merely be a `return nil` stub; it must validate Kenyan phone formats (`+254...`), simulate network latencies, generate authentic M-Pesa receipts (e.g. `QGR458JKL1`), trigger real HTTP callbacks, and respect error scenarios (e.g. user cancellations).
- **Runtime Toggle**: Switching between live third-party gateways and offline simulation must be controlled via a single environment variable: `USE_MOCKS=true|false`.

---

## 3. Decision Outcome

Chosen Option: **Interface-Driven Adapter Pattern with Dual Implementations (Live & Mock)**.

### Implementation Architecture

1. **Payment Service (`backend/payment`)**:
   - `DarajaClient` interface:
     ```go
     type DarajaClient interface {
         InitiateSTKPush(ctx context.Context, phone, amount, orderID string) (*DarajaSTKResponse, error)
     }
     ```
   - Dual implementations:
     - `LiveDarajaClient`: Authenticates with Safaricom OAuth sandbox, encodes password, issues HTTP POST to Daraja endpoint.
     - `MockDarajaClient`: Validates Kenyan phone prefix (+2547, +2541), simulates Safaricom STK prompt response, and schedules an automatic background HTTP callback to `http://localhost:8084/mpesa/callback` with ResultCode 0 after a configurable delay (500ms).

2. **Notification Service (`backend/notification`)**:
   - `SMSProvider`:
     - Live: Sends SMS via Africa's Talking REST API.
     - Mock: Logs SMS delivery to console with simulated message ID (`AT-MOCK-...`) and validates E.164 phone normalization.
   - `EmailProvider`:
     - Live: Dispatches SMTP messages to local Mailpit container (`localhost:1025`), providing a web UI at `http://localhost:8025` to inspect rendered HTML emails.
     - Mock: Dispatches mock delivery when Mailpit is unavailable.

---

## 4. Consequences

### Positive
- **Instant Local Bootstrapping**: Any engineer or reviewer can verify the entire microservices stack in under 60 seconds with zero third-party accounts.
- **Hermetic CI Pipelines**: GitHub Actions runs the complete test matrix hermetically without requiring external network access or managing secret rotations for sandbox accounts.
- **Chaos Testing**: The mock simulators enable controlled chaos testing (simulating STK prompt cancellation ResultCode 1032, network timeout, or insufficient balance ResultCode 1001) deterministically in unit and integration tests.

### Negative & Mitigations
- **Divergence Risk**: The mock adapter could diverge from upstream API changes (e.g. Safaricom updating Daraja callback payload). *Mitigation*: Unit contract tests (`contract_test.go`) and annual contract verification against Safaricom's official developer documentation.

