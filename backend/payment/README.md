// path: backend/payment/README.md
# Payment Service (`artisan-market`)

Production-grade payment processing microservice for the Kenyan Artisan Marketplace. Supports Safaricom Daraja M-Pesa (Lipa Na M-Pesa Online STK Push) in KES and Stripe PaymentIntents (test mode), with zero-credential mock adapters (`USE_MOCKS=true`), dual gRPC/HTTP callback handlers, transactional outbox pattern, and idempotent saga event consumers.

## Architecture

```mermaid
flowchart LR
    KafkaIn[Kafka: payment.events] -->|payment.requested| Consumer[Kafka Consumer]
    Consumer -->|Initiate STK Push / Intent| Provider[Daraja / Stripe / Mock]
    Provider -.->|Asynchronous Callback| HTTP[HTTP :8084]
    HTTP -->|Process Callback| Srv[Payment Server]
    Srv -->|Atomic Commit| DB[(PostgreSQL: payment_db)]
    DB -->|Outbox Table| OutboxRelay[Transactional Outbox Relay]
    OutboxRelay -->|payment.succeeded / payment.failed| KafkaOut[Kafka: payment.events]
```

## Features
- **Safaricom Daraja STK Push**: Initiates `CustomerPayBillOnline` STK push to Kenyan carrier numbers (`+2547...` / `+2541...`).
- **Daraja Callback Webhook**: `POST /mpesa/callback` parses Safaricom callback JSON payload, validates `ResultCode` (0 = Success, 1032 = User Cancelled), extracts M-Pesa receipt, and queues outbox domain events.
- **Stripe PaymentIntents**: Creates PaymentIntents with currency matching (KES) and verifies inbound `POST /stripe/webhook` signatures via HMAC-SHA256 (`Stripe-Signature`).
- **Transactional Outbox Relay**: Polls `outbox` table with `FOR UPDATE SKIP LOCKED` and publishes `payment.succeeded` or `payment.failed` to Kafka topic `payment.events`.
- **Idempotent Consumer & Handler**: Uses `idempotent_events` table to guarantee that duplicate callbacks or duplicate Kafka messages are executed at most once.
- **Zero-Credential Mock Mode**: With `USE_MOCKS=true`, generates realistic `ws_CO_MOCK_...` checkout IDs and `pi_mock_...` Stripe intents, allowing 100% end-to-end local testing.

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `PAYMENT_DB_DSN` | `postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable` | PostgreSQL connection string |
| `PAYMENT_SERVICE_ADDR` | `:50054` | gRPC service listening address |
| `PAYMENT_HTTP_ADDR` | `:8084` | HTTP callback/webhook listening address |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka/Redpanda broker seed list |
| `OUTBOX_POLL_INTERVAL` | `2s` | Outbox polling frequency |
| `USE_MOCKS` | `true` | When true, enables zero-credential mock mode |
| `DARAJA_CONSUMER_KEY` | `""` | Safaricom Daraja Consumer Key |
| `DARAJA_CONSUMER_SECRET` | `""` | Safaricom Daraja Consumer Secret |
| `DARAJA_PASSKEY` | `""` | Safaricom Daraja Lipa Na M-Pesa Passkey |
| `DARAJA_SHORTCODE` | `174379` | Safaricom Paybill / Shortcode |
| `DARAJA_CALLBACK_URL` | `http://localhost:8084/mpesa/callback` | Callback URL registered with Safaricom |
| `STRIPE_SECRET_KEY` | `""` | Stripe secret API key (`sk_test_...`) |
| `STRIPE_WEBHOOK_SECRET` | `""` | Stripe webhook signing secret (`whsec_...`) |

## Running Tests

```bash
cd backend/payment
go test -tags purego -v ./... -count=1
```
