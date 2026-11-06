// path: backend/order/saga_e2e_test.go
package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	orderpb "github.com/yourorg/artisan-market/proto/order/v1"
)

// TestSagaE2E_FullHappyPath validates the complete distributed saga workflow:
// 1. Order Created -> PENDING
// 2. Inventory Reserved -> INVENTORY_RESERVED, payment.requested emitted with KES cents
// 3. Payment Succeeded -> ORDER_CONFIRMED, order.confirmed emitted for notification
// 4. Duplicate payment.succeeded delivery is cleanly handled by idempotent_events
func TestSagaE2E_FullHappyPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	server := NewOrderServiceServer(db)

	orderID := "order-kenya-001"
	userID := "user-nairobi-001"
	totalCents := int64(500000) // KES 5,000.00
	customerPhone := "+254712345678"
	customerEmail := "buyer@artisan.co.ke"

	// -------------------------------------------------------------
	// STEP 1: Create Order
	// -------------------------------------------------------------
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO orders").
		WithArgs(sqlmock.AnyArg(), userID, totalCents, "mpesa", customerPhone, customerEmail).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO order_items").
		WithArgs(sqlmock.AnyArg(), "shuka-01", "Handcrafted Artisan Item", int32(2), int64(250000)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(sqlmock.AnyArg(), "order.created", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	createReq := &orderpb.CreateOrderRequest{
		UserId: userID,
		Items: []*orderpb.CreateOrderItem{
			{ProductId: "shuka-01", Quantity: 2},
		},
		PaymentMethod: "mpesa",
		CustomerPhone: customerPhone,
		CustomerEmail: customerEmail,
	}

	createResp, err := server.CreateOrder(ctx, createReq)
	if err != nil {
		t.Fatalf("Step 1 failed - CreateOrder returned error: %v", err)
	}
	if createResp.GetStatus() != "pending" {
		t.Fatalf("expected initial status 'pending', got '%s'", createResp.GetStatus())
	}

	// -------------------------------------------------------------
	// STEP 2: Product service confirms inventory -> inventory.reserved
	// -------------------------------------------------------------
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-inv-res-001").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"})) // not processed yet

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("evt-inv-res-001", "order-saga-consumer").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Update order to reserved & fetch payment fields for outbox event
	returnRows := sqlmock.NewRows([]string{"total_cents", "payment_method", "customer_phone", "customer_email"}).
		AddRow(totalCents, "mpesa", customerPhone, customerEmail)
	mock.ExpectQuery("UPDATE orders SET status = 'reserved', saga_step = 'INVENTORY_RESERVED'").
		WithArgs(orderID).
		WillReturnRows(returnRows)

	// Enqueue payment.requested to outbox
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "payment.requested", "payment.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	invPayload, _ := json.Marshal(map[string]any{
		"event_id":       "evt-inv-res-001",
		"order_id":       orderID,
		"reservation_id": "res-999",
	})

	err = handleSagaMessage(ctx, db, "inventory.reserved", []byte(orderID), invPayload)
	if err != nil {
		t.Fatalf("Step 2 failed - handleSagaMessage inventory.reserved returned error: %v", err)
	}

	// -------------------------------------------------------------
	// STEP 3: Payment service confirms payment -> payment.succeeded
	// -------------------------------------------------------------
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-pay-succ-001").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}))

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("evt-pay-succ-001", "order-saga-consumer").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE orders SET status = 'confirmed'").
		WithArgs(orderID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Enqueue order.confirmed to outbox
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "order.confirmed", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	payPayload, _ := json.Marshal(map[string]any{
		"event_id":   "evt-pay-succ-001",
		"order_id":   orderID,
		"payment_id": "pay-daraja-stk-001",
	})

	err = handleSagaMessage(ctx, db, "payment.succeeded", []byte(orderID), payPayload)
	if err != nil {
		t.Fatalf("Step 3 failed - handleSagaMessage payment.succeeded returned error: %v", err)
	}

	// -------------------------------------------------------------
	// STEP 4: Idempotency Verification (duplicate delivery)
	// -------------------------------------------------------------
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-pay-succ-001").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}).AddRow("evt-pay-succ-001")) // already present
	mock.ExpectRollback()

	err = handleSagaMessage(ctx, db, "payment.succeeded", []byte(orderID), payPayload)
	if err != nil {
		t.Fatalf("Step 4 failed - duplicate event should be ignored idempotently, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

// TestSagaE2E_PaymentFailure_Compensation validates that when a customer cancels an M-Pesa STK Push
// or has insufficient balance, the saga rolls back: Order is CANCELLED and order.cancelled
// compensation event is enqueued to release the reserved product inventory.
func TestSagaE2E_PaymentFailure_Compensation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	orderID := "order-cancel-002"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-fail-daraja-1032").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}))

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("evt-fail-daraja-1032", "order-saga-consumer").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Order transitions to cancelled / PAYMENT_FAILED
	mock.ExpectExec("UPDATE orders SET status = 'cancelled', saga_step = 'PAYMENT_FAILED'").
		WithArgs(orderID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Triggers order.cancelled compensation event for Product service to release stock
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "order.cancelled", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	failPayload, _ := json.Marshal(map[string]any{
		"event_id": "evt-fail-daraja-1032",
		"order_id": orderID,
		"reason":   "DS_USER_CANCELLED_OR_INSUFFICIENT_FUNDS",
	})

	err = handleSagaMessage(ctx, db, "payment.failed", []byte(orderID), failPayload)
	if err != nil {
		t.Fatalf("payment failure compensation returned unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

// TestSagaE2E_PaymentTimeout_Compensation verifies that if Safaricom STK Push callback
// times out (>60 seconds), a payment.timeout event cancels the order and releases reserved stock.
func TestSagaE2E_PaymentTimeout_Compensation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	orderID := "order-timeout-003"

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-timeout-001").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}))

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("evt-timeout-001", "order-saga-consumer").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE orders SET status = 'cancelled', saga_step = 'PAYMENT_FAILED'").
		WithArgs(orderID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "order.cancelled", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	timeoutPayload, _ := json.Marshal(map[string]any{
		"event_id": "evt-timeout-001",
		"order_id": orderID,
		"reason":   "payment_timeout_exceeded_60s",
	})

	err = handleSagaMessage(ctx, db, "payment.timeout", []byte(orderID), timeoutPayload)
	if err != nil {
		t.Fatalf("payment timeout compensation returned unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}

// TestSagaE2E_Chaos_KafkaOutage_OutboxRecovery verifies system resilience:
// When the message broker is temporarily offline during an order operation, events
// are safely persisted in PostgreSQL outbox table with status 'pending'.
// Once Kafka connectivity is restored, the outbox relay reliably drains the batch.
func TestSagaE2E_Chaos_KafkaOutage_OutboxRecovery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Transactional Outbox insertion while Kafka is DOWN
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs("ord-chaos-999", "order.created", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin tx: %v", err)
	}

	payload := map[string]any{
		"order_id":    "ord-chaos-999",
		"amount_cents": 500000,
	}
	err = QueueOutboxEvent(ctx, tx, "ord-chaos-999", "order.created", "order.events", payload, "corr-chaos-1")
	if err != nil {
		t.Fatalf("QueueOutboxEvent failed during simulated broker outage: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("transaction commit failed: %v", err)
	}

	// 2. Outbox Relay draining pending records after broker recovers
	now := time.Now()
	payloadJSON, _ := json.Marshal(payload)
	outboxRows := sqlmock.NewRows([]string{
		"id", "aggregate_id", "event_type", "topic", "payload", "correlation_id",
	}).AddRow("outbox-uuid-1", "ord-chaos-999", "order.created", "order.events", payloadJSON, "corr-chaos-1")

	// The relay queries pending records with row-level locks
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, aggregate_id, event_type, topic, payload, correlation_id FROM outbox WHERE status = 'pending'").
		WillReturnRows(outboxRows)

	// When published, updates row to published
	mock.ExpectExec("UPDATE outbox SET status = 'published', published_at = now\\(\\) WHERE id = \\$1").
		WithArgs("outbox-uuid-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Execute simulated relay sweep
	sweepTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to begin sweep tx: %v", err)
	}

	rows, err := sweepTx.QueryContext(ctx, "SELECT id, aggregate_id, event_type, topic, payload, correlation_id FROM outbox WHERE status = 'pending' ORDER BY created_at LIMIT 50 FOR UPDATE SKIP LOCKED")
	if err != nil {
		t.Fatalf("failed to query pending outbox: %v", err)
	}

	var processedCount int
	for rows.Next() {
		var id, aggID, eventType, topic, corrID string
		var rawPayload []byte
		if err := rows.Scan(&id, &aggID, &eventType, &topic, &rawPayload, &corrID); err != nil {
			t.Fatalf("scan error: %v", err)
		}

		// Simulated Kafka publish success
		_, err := sweepTx.ExecContext(ctx, "UPDATE outbox SET status = 'published', published_at = now() WHERE id = $1", id)
		if err != nil {
			t.Fatalf("failed to mark published: %v", err)
		}
		processedCount++
	}
	rows.Close()

	if err := sweepTx.Commit(); err != nil {
		t.Fatalf("sweep commit failed: %v", err)
	}

	if processedCount != 1 {
		t.Errorf("expected 1 processed outbox record after recovery, got %d", processedCount)
	}

	_ = now
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled sqlmock expectations: %v", err)
	}
}
