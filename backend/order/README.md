# Order Service

The **order** microservice implements the order aggregate, persisting orders to PostgreSQL and publishing domain events via an outbox table to Kafka (`order.events`).

## Running locally

```sh
# Install dependencies (if you haven't already)
make up                 # start dependent containers (Postgres, Redpanda, etc.)
make run-order          # starts the gRPC order service on $ORDER_SERVICE_ADDR (default :50053)
```

The service reads the following env vars (see `.env.example`):
- `ORDER_DB_DSN` – PostgreSQL DSN for the order database.
- `ORDER_SERVICE_ADDR` – gRPC bind address.
- `KAFKA_BROKERS` – comma‑separated list of Kafka broker addresses.
- `OUTBOX_POLL_INTERVAL` – how often the outbox processor polls for unpublished events (e.g. `5s`).

## API (gRPC)

- `CreateOrder` – creates a new order and writes an `OrderCreated` event to the outbox.
- `GetOrder` – fetches a single order.
- `ListOrders` – returns all orders.
- `UpdateOrderStatus` – updates the order status and writes an `OrderStatusUpdated` event.

## Persistence

- **orders** table – stores order aggregates.
- **outbox** table – stores events to be reliably published to Kafka. The background processor reads rows where `published = false`, produces them to the `order.events` topic, and marks them as published.

## Tests

*Tests are not yet implemented – add unit tests in `backend/order` and run via `make test`.*
