// path: docs/adr/003-mtls-and-secrets-management.md
# ADR 003: Mutual TLS (mTLS) with SPIFFE/Istio and Secrets Architecture

## Status
**Accepted**

## Context
Artisan Market is a distributed microservices platform handling payment flows (Safaricom Daraja M-Pesa STK Push, Stripe), catalogue inventory with concurrency locks, and customer PII (Kenyan phone numbers, email receipts). 

Relying on perimeter security alone (e.g. firewalls or private subnets) leaves the platform vulnerable to lateral movement if a container or node is compromised. Additionally, transmitting plaintext gRPC traffic within the cluster risks credential leakage.

## Decision
1. **Enforce Mutual TLS (mTLS) Mesh-Wide**:
   - In production (Kubernetes), deploy Istio Service Mesh with `PeerAuthentication` in `STRICT` mode. All pod-to-pod TCP/gRPC traffic is encrypted with TLS 1.3 and verified via SPIFFE identities (`cluster.local/ns/artisan-market/sa/<service>-sa`).
   - In local development (Docker Compose / host), provide an automated development Certificate Authority (`scripts/generate-dev-certs.sh` and `scripts/generate-certs.go`) that generates root CA and per-service X.509 certificates with Subject Alternative Names (SANs) for `localhost`, `127.0.0.1`, and service DNS names.
2. **Fine-Grained Authorization Policies (Least Privilege)**:
   - Deploy a default `DENY ALL` Istio `AuthorizationPolicy`.
   - Explicitly authorize `gateway-sa` to invoke downstream services (`user`, `product`, `order`, `payment`).
   - Authorize `order-sa` to invoke `product` service solely for inventory reservations.
3. **Secrets Management**:
   - Zero hard-coded credentials. All secrets are passed via environment variables adhering to twelve-factor app methodology.
   - External third-party integrations (Daraja, Stripe, Africa's Talking) provide deterministic zero-credential mock engines activated via `USE_MOCKS=true`.

## Consequences
- **Positive**:
  - End-to-end cryptographic confidentiality and integrity across all microservices.
  - Prevention of unauthorized lateral service-to-service calls.
  - Compliance with the Kenyan Data Protection Act 2019 and Central Bank of Kenya cybersecurity guidelines.
- **Trade-offs & Mitigations**:
  - *Trade-off*: Local certificate generation can be cumbersome on heterogeneous developer machines (e.g. Windows without OpenSSL).
  - *Mitigation*: Implemented `scripts/generate-certs.go` in pure Go standard library (`crypto/x509`), allowing 1-command generation on any operating system without external dependencies.

