package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

// SagaPayload represents the generic domain payload received from Kafka topics.
type SagaPayload struct {
	EventID        string `json:"event_id"`
	OrderID        string `json:"order_id"`
	PaymentID      string `json:"payment_id"`
	ReservationID  string `json:"reservation_id"`
	Reason         string `json:"reason"`
	AmountCents    int64  `json:"amount_cents"`
	PaymentMethod  string `json:"payment_method"`
	CustomerPhone  string `json:"customer_phone"`
	CustomerEmail  string `json:"customer_email"`
}

// StartSagaConsumer listens to inventory and payment topics and steps through the saga state machine.
func StartSagaConsumer(ctx context.Context, db *sql.DB, client *kgo.Client) {
	if client == nil {
		return
	}

	log.Println("[Order Saga Consumer] Subscribing to saga topics: inventory.events, payment.events...")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			fetches := client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					if err.Err != context.Canceled {
						log.Printf("[Order Saga Consumer] Fetch error: %v", err.Err)
					}
				}
				continue
			}

			fetches.EachRecord(func(record *kgo.Record) {
				eventType := ""
				for _, h := range record.Headers {
					if h.Key == "event_type" {
						eventType = string(h.Value)
						break
					}
				}
				if eventType == "" {
					eventType = string(record.Topic)
				}

				if err := handleSagaMessage(ctx, db, eventType, record.Key, record.Value); err != nil {
					log.Printf("[Order Saga Consumer] Failed to process %s: %v", eventType, err)
				}
			})
		}
	}
}

func handleSagaMessage(ctx context.Context, db *sql.DB, eventType string, key, value []byte) error {
	var payload SagaPayload
	if err := json.Unmarshal(value, &payload); err != nil {
		return err
	}

	if payload.OrderID == "" {
		payload.OrderID = string(key)
	}

	eventID := payload.EventID
	if eventID == "" {
		eventID = string(key) + "-" + eventType
	}

	// Idempotency check: prevent duplicate event processing
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existing string
	err = tx.QueryRowContext(ctx, "SELECT event_id FROM idempotent_events WHERE event_id = $1", eventID).Scan(&existing)
	if err == nil {
		// Already processed
		return nil
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO idempotent_events (event_id, consumer_name) VALUES ($1, $2)", eventID, "order-saga-consumer"); err != nil {
		return err
	}

	correlationID := uuid.New().String()

	switch {
	case strings.Contains(eventType, "inventory.reserved") || eventType == "InventoryReserved":
		// Step 2: Inventory reserved successfully -> Trigger payment request
		query := `UPDATE orders SET status = 'reserved', saga_step = 'INVENTORY_RESERVED', updated_at = now() WHERE id = $1 RETURNING total_cents, payment_method, customer_phone, customer_email`
		var totalCents int64
		var method, phone, email string
		if err := tx.QueryRowContext(ctx, query, payload.OrderID).Scan(&totalCents, &method, &phone, &email); err != nil {
			return err
		}

		paymentRequestedPayload := map[string]any{
			"payment_id":     uuid.New().String(),
			"order_id":       payload.OrderID,
			"amount_cents":   totalCents,
			"currency":       "KES",
			"payment_method": method,
			"customer_phone": phone,
			"customer_email": email,
		}
		if err := QueueOutboxEvent(ctx, tx, payload.OrderID, "payment.requested", "payment.events", paymentRequestedPayload, correlationID); err != nil {
			return err
		}

	case strings.Contains(eventType, "inventory.reservation_failed") || eventType == "InventoryReservationFailed":
		// Step 2b: Stock unavailable -> Fail order saga
		query := `UPDATE orders SET status = 'cancelled', saga_step = 'INVENTORY_FAILED', updated_at = now() WHERE id = $1`
		if _, err := tx.ExecContext(ctx, query, payload.OrderID); err != nil {
			return err
		}

		cancelledPayload := map[string]any{
			"order_id": payload.OrderID,
			"reason":   payload.Reason,
		}
		if err := QueueOutboxEvent(ctx, tx, payload.OrderID, "order.cancelled", "order.events", cancelledPayload, correlationID); err != nil {
			return err
		}

	case strings.Contains(eventType, "payment.succeeded") || eventType == "PaymentSucceeded":
		// Step 3a: Payment succeeded -> Confirm order saga
		query := `UPDATE orders SET status = 'confirmed', saga_step = 'ORDER_CONFIRMED', updated_at = now() WHERE id = $1`
		if _, err := tx.ExecContext(ctx, query, payload.OrderID); err != nil {
			return err
		}

		confirmedPayload := map[string]any{
			"order_id":   payload.OrderID,
			"payment_id": payload.PaymentID,
		}
		if err := QueueOutboxEvent(ctx, tx, payload.OrderID, "order.confirmed", "order.events", confirmedPayload, correlationID); err != nil {
			return err
		}

	case strings.Contains(eventType, "payment.failed") || strings.Contains(eventType, "payment.timeout") || eventType == "PaymentFailed" || eventType == "PaymentTimeout":
		// Step 3b: Payment failed or timed out -> Cancel order and trigger product compensation (inventory release)
		query := `UPDATE orders SET status = 'cancelled', saga_step = 'PAYMENT_FAILED', updated_at = now() WHERE id = $1`
		if _, err := tx.ExecContext(ctx, query, payload.OrderID); err != nil {
			return err
		}

		cancelledPayload := map[string]any{
			"order_id": payload.OrderID,
			"reason":   "payment_failure_or_timeout",
		}
		// Publishing order.cancelled triggers the product service to release inventory
		if err := QueueOutboxEvent(ctx, tx, payload.OrderID, "order.cancelled", "order.events", cancelledPayload, correlationID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
