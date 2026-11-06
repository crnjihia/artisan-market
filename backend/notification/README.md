// path: backend/notification/README.md
# Notification Service (`artisan-market`)

Bilingual notification microservice (Node.js 20 + TypeScript) for the Kenyan Artisan Marketplace. Consumes saga events from Kafka (`order.events`, `payment.events`), formats localized customer communications in English and Swahili, dispatches SMS via Africa's Talking (sandbox & mock) and emails via Mailpit SMTP (sandbox & mock), featuring exponential backoff retries and Dead Letter Queue (`notification.dlq`) routing.

## Architecture

```mermaid
flowchart TD
    K[Kafka Topics: order.events, payment.events] -->|Events| C[Notification Consumer]
    C -->|Parse & Deduplicate| T[Bilingual Templates (EN/SW)]
    T -->|Rendered Payload| R[Retry Handler (Exponential Backoff)]
    R -->|Attempt 1..3| S[SMS Provider: Africa's Talking]
    R -->|Attempt 1..3| E[Email Provider: Mailpit SMTP]
    R -->|Persistent Failure| DLQ[Kafka Topic: notification.dlq]
```

## Features
- **Dual Language Support (EN & SW)**: Renders buyer communications in English or Swahili (e.g. `order.created`, `order.confirmed`, `order.cancelled`, `payment.succeeded`, `payment.failed`).
- **Kenyan Currency & Phone Formatting**: Normalizes Kenyan mobile numbers to E.164 (`+2547...` / `+2541...`) and formats prices in Kenya Shillings (`KES`).
- **Africa's Talking SMS**: Integrates with Africa's Talking Messaging API (sandbox and production), with transparent zero-credential mock mode (`USE_MOCKS=true`).
- **Mailpit SMTP Email**: Connects to Mailpit locally on `:1025` for development email inspection, supporting both HTML and plain text multi-part formatting.
- **Retry with Exponential Backoff & DLQ**: Automatically retries transient provider errors up to 3 times before routing unrecoverable payloads to `notification.dlq`.
- **Idempotency**: Message deduplication prevents double-sending notifications for redelivered Kafka messages.

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `NOTIFICATION_HTTP_PORT` | `3001` | Health check HTTP server port |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka/Redpanda broker addresses |
| `NOTIFICATION_GROUP_ID` | `notification-service-group` | Kafka consumer group ID |
| `NOTIFICATION_DLQ_TOPIC` | `notification.dlq` | Dead Letter Queue Kafka topic |
| `USE_MOCKS` | `true` | When true, logs mock deliveries without external credentials |
| `AFRICA_TALKING_USERNAME` | `sandbox` | Africa's Talking account username |
| `AFRICA_TALKING_API_KEY` | `""` | Africa's Talking API key |
| `AFRICA_TALKING_SENDER_ID` | `""` | Alphanumeric sender ID (optional) |
| `EMAIL_SMTP_HOST` | `localhost` | SMTP host (Mailpit locally) |
| `EMAIL_SMTP_PORT` | `1025` | SMTP port (Mailpit locally) |
| `EMAIL_FROM` | `Artisan Market <no-reply@artisan-market.local>` | Default email sender |

## Running Tests

```bash
cd backend/notification
npm test
```
