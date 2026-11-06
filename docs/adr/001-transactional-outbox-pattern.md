# path: docs/adr/001-transactional-outbox-pattern.md
# ADR 001: Transactional Outbox Pattern for Reliable Distributed Event Publishing

- **Status**: Accepted
- **Date**: 2026-11-06
- **Author**: Christopher Njihia (Nairobi, Kenya)
- **Scope**: `order`, `payment`, and all event-driven microservices

---

## 1. Context and Problem Statement

In the Artisan Market e-commerce platform, operations such as order placement and payment status transitions require two side effects:
1. Mutating local state in PostgreSQL (e.g., creating an order record or recording an M-Pesa transaction).
2. Emitting an event to Kafka/Redpanda (e.g., `order.created`, `payment.succeeded`) to drive the distributed checkout saga and customer notifications.

Performing these two operations sequentially in application memory ("dual writes") introduces severe failure modes:
- **Case A (Publish then Commit)**: The Kafka event is emitted, but the PostgreSQL commit fails due to a constraint violation or connection reset. Downstream services reserve stock or charge M-Pesa for a non-existent order.
- **Case B (Commit then Publish)**: PostgreSQL commits successfully, but Kafka is temporarily unreachable or the container dies before publishing. The event is permanently dropped, resulting in orphan orders where customer payments are never collected.

Two-Phase Commit (2PC) and distributed XA transactions are unsupported by Kafka and introduce heavy locking that impairs availability under high concurrency.

---

## 2. Decision Drivers

- **Zero Data Loss**: Orders, inventory reservations, and M-Pesa callbacks must never be lost during network partitions or broker downtime.
- **Atomic Local Transactions**: Event emission must be strictly coupled to the success of the database transaction.
- **High Concurrency & Low Contention**: Multiple worker pods must be able to drain events concurrently without deadlocks or row locks blocking new user writes.
- **Simplicity & Operational Independence**: Avoid heavy external CDC infrastructure (such as Debezium / Kafka Connect) for developer-first ease of maintenance and zero-dependency deployments.

---

## 3. Considered Options

1. **Dual Writes in Application Layer**: Directly write to Postgres and then publish to Kafka.
2. **Change Data Capture (CDC) via Debezium**: Stream Postgres WAL to Kafka via Kafka Connect.
3. **Transactional Outbox with PostgreSQL Polling (`SKIP LOCKED`)**: Write events into an `outbox` table in the same transaction as domain entities, drained by an asynchronous Go worker.

---

## 4. Decision Outcome

Chosen Option: **Transactional Outbox with PostgreSQL `SKIP LOCKED`**.

### Architecture & Implementation Details

1. **Transactional Outbox Table**: Each microservice schema (`artisan_order`, `artisan_payment`) contains a dedicated outbox table:
   ```sql
   CREATE TABLE outbox (
       id UUID PRIMARY KEY,
       aggregate_id VARCHAR(64) NOT NULL,
       event_type VARCHAR(128) NOT NULL,
       topic VARCHAR(128) NOT NULL,
       payload JSONB NOT NULL,
       correlation_id VARCHAR(64) NOT NULL,
       status VARCHAR(32) NOT NULL DEFAULT 'pending',
       created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
       published_at TIMESTAMPTZ
   );
   CREATE INDEX idx_outbox_pending ON outbox (created_at) WHERE status = 'pending';
   ```

2. **Atomic Ingestion**: Domain entities and outbox events are written in a single ACID transaction (`BeginTx` -> `Insert Order` -> `Insert Outbox` -> `Commit`).

3. **High-Throughput Concurrent Relay Worker**:
   An asynchronous Go goroutine (`StartOutboxRelay`) polls the pending events every 1–2 seconds:
   ```sql
   SELECT id, aggregate_id, event_type, topic, payload, correlation_id
   FROM outbox
   WHERE status = 'pending'
   ORDER BY created_at ASC
   LIMIT 50
   FOR UPDATE SKIP LOCKED;
   ```
   - `SKIP LOCKED` allows multiple replicas of `order-service` to read and drain distinct outbox batches simultaneously without lock contention or serial blocking.
   - Upon successful Kafka acknowledgment, the worker marks `status = 'published'` and `published_at = now()`.

4. **Idempotent Consumption Guarantee**:
   Because network retries can cause at-least-once message delivery, every consumer maintains an `idempotent_events` table:
   ```sql
   INSERT INTO idempotent_events (event_id, consumer_name) VALUES ($1, $2);
   ```
   If the unique constraint is violated, the duplicate event is silently acknowledged without duplicating business actions.

---

## 5. Consequences

### Positive
- **Guaranteed Consistency**: State updates and event publishing are 100% atomic.
- **Resilience to Broker Downtime**: If Kafka experiences an outage, user orders continue to be accepted; outbox records accumulate safely in PostgreSQL and automatically drain upon recovery.
- **Zero Third-Party Daemon Dependencies**: Runs entirely within Go microservice processes without needing external CDC containers.
- **Traceability**: Outbox records preserve `correlation_id` and OpenTelemetry W3C baggage across the saga lifecycle.

### Negative & Mitigations
- **At-Least-Once Delivery**: Downstream consumers must handle duplicate events. *Mitigation*: Enforced via database-level `idempotent_events` table across all consumers.
- **Table Growth**: Successfully published rows accumulate over time. *Mitigation*: Partitioning or a scheduled cron purging published records older than 7 days (`DELETE FROM outbox WHERE status = 'published' AND published_at < now() - interval '7 days'`).

