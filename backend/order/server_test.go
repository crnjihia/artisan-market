package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	orderpb "github.com/yourorg/artisan-market/proto/order/v1"
)

func TestCreateOrder_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewOrderServiceServer(db)

	mock.ExpectBegin()
	// Insert order
	mock.ExpectExec("INSERT INTO orders").
		WithArgs(sqlmock.AnyArg(), "user-uuid-1", int64(250000), "mpesa", "+254712345678", "buyer@nairobi.co.ke").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Insert item
	mock.ExpectExec("INSERT INTO order_items").
		WithArgs(sqlmock.AnyArg(), "prod-1", "Handcrafted Artisan Item", int32(1), int64(250000)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Queue transactional outbox event
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(sqlmock.AnyArg(), "order.created", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &orderpb.CreateOrderRequest{
		UserId: "user-uuid-1",
		Items: []*orderpb.CreateOrderItem{
			{ProductId: "prod-1", Quantity: 1},
		},
		PaymentMethod: "mpesa",
		CustomerPhone: "+254712345678",
		CustomerEmail: "buyer@nairobi.co.ke",
	}

	resp, err := server.CreateOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if resp.GetOrderId() == "" {
		t.Error("expected non-empty order ID")
	}
	if resp.GetStatus() != "pending" {
		t.Errorf("expected pending status, got %s", resp.GetStatus())
	}
	if resp.GetTotalCents() != 250000 {
		t.Errorf("expected 250000 total cents, got %d", resp.GetTotalCents())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestGetOrder_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewOrderServiceServer(db)

	now := time.Now()
	orderRows := sqlmock.NewRows([]string{
		"id", "user_id", "total_cents", "currency", "status", "payment_method", "customer_phone", "customer_email", "saga_step", "created_at", "updated_at",
	}).AddRow("order-123", "user-1", int64(500000), "KES", "confirmed", "mpesa", "+254712345678", "buyer@nairobi.co.ke", "ORDER_CONFIRMED", now, now)

	mock.ExpectQuery("SELECT id, user_id, total_cents").
		WithArgs("order-123").
		WillReturnRows(orderRows)

	itemRows := sqlmock.NewRows([]string{"product_id", "product_name", "quantity", "unit_price_cents"}).
		AddRow("prod-lion", "Kisii Soapstone Lion", int32(2), int64(250000))

	mock.ExpectQuery("SELECT product_id, product_name, quantity, unit_price_cents FROM order_items").
		WithArgs("order-123").
		WillReturnRows(itemRows)

	resp, err := server.GetOrder(context.Background(), &orderpb.GetOrderRequest{OrderId: "order-123"})
	if err != nil {
		t.Fatalf("GetOrder failed: %v", err)
	}

	if resp.GetOrder().GetId() != "order-123" {
		t.Errorf("expected order-123, got %s", resp.GetOrder().GetId())
	}
	if len(resp.GetOrder().GetItems()) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.GetOrder().GetItems()))
	}
	if resp.GetOrder().GetItems()[0].GetProductName() != "Kisii Soapstone Lion" {
		t.Errorf("expected Kisii Soapstone Lion, got %s", resp.GetOrder().GetItems()[0].GetProductName())
	}
}

func TestCancelOrder_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewOrderServiceServer(db)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE orders SET status = 'cancelled'").
		WithArgs("order-123").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(sqlmock.AnyArg(), "order.cancelled", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	resp, err := server.CancelOrder(context.Background(), &orderpb.CancelOrderRequest{
		OrderId: "order-123",
		Reason:  "buyer_requested",
	})
	if err != nil {
		t.Fatalf("CancelOrder failed: %v", err)
	}

	if !resp.GetSuccess() || resp.GetStatus() != "cancelled" {
		t.Errorf("unexpected cancel response: %v", resp)
	}
}

func TestSagaConsumer_InventoryReserved(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	// Idempotency check: not found in idempotent_events
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-1").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}))

	// Record event in idempotent_events
	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("evt-1", "order-saga-consumer").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Update order to reserved
	mock.ExpectQuery("UPDATE orders SET status = 'reserved'").
		WithArgs("order-123").
		WillReturnRows(sqlmock.NewRows([]string{"total_cents", "payment_method", "customer_phone", "customer_email"}).
			AddRow(int64(250000), "mpesa", "+254712345678", "buyer@nairobi.co.ke"))

	// Trigger payment.requested in outbox
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(sqlmock.AnyArg(), "payment.requested", "payment.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	payload, _ := json.Marshal(map[string]any{
		"event_id": "evt-1",
		"order_id": "order-123",
	})

	err = handleSagaMessage(context.Background(), db, "inventory.reserved", []byte("order-123"), payload)
	if err != nil {
		t.Fatalf("handleSagaMessage failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestSagaConsumer_PaymentSucceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-pay-1").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}))

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("evt-pay-1", "order-saga-consumer").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE orders SET status = 'confirmed'").
		WithArgs("order-123").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(sqlmock.AnyArg(), "order.confirmed", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	payload, _ := json.Marshal(map[string]any{
		"event_id":   "evt-pay-1",
		"order_id":   "order-123",
		"payment_id": "pay-xyz-1",
	})

	err = handleSagaMessage(context.Background(), db, "payment.succeeded", []byte("order-123"), payload)
	if err != nil {
		t.Fatalf("handleSagaMessage payment.succeeded failed: %v", err)
	}
}

func TestSagaConsumer_PaymentFailed_TriggersCompensation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("evt-fail-1").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}))

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("evt-fail-1", "order-saga-consumer").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE orders SET status = 'cancelled'").
		WithArgs("order-123").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Triggers order.cancelled compensation event for product service
	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(sqlmock.AnyArg(), "order.cancelled", "order.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	payload, _ := json.Marshal(map[string]any{
		"event_id": "evt-fail-1",
		"order_id": "order-123",
		"reason":   "insufficient_funds_or_user_cancelled",
	})

	err = handleSagaMessage(context.Background(), db, "payment.failed", []byte("order-123"), payload)
	if err != nil {
		t.Fatalf("handleSagaMessage payment.failed failed: %v", err)
	}
}

func TestSagaConsumer_Idempotency(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	// Event already exists
	mock.ExpectQuery("SELECT event_id FROM idempotent_events").
		WithArgs("already-processed-evt").
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}).AddRow("already-processed-evt"))

	mock.ExpectRollback()

	payload, _ := json.Marshal(map[string]any{
		"event_id": "already-processed-evt",
		"order_id": "order-123",
	})

	err = handleSagaMessage(context.Background(), db, "payment.succeeded", []byte("order-123"), payload)
	if err != nil {
		t.Fatalf("idempotent replay should not return error, got: %v", err)
	}
}

