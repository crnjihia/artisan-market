// path: backend/payment/consumer.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

// PaymentRequestPayload models the inbound payment.requested event.
type PaymentRequestPayload struct {
	EventID       string `json:"event_id"`
	PaymentID     string `json:"payment_id"`
	OrderID       string `json:"order_id"`
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	CustomerPhone string `json:"customer_phone"`
	CustomerEmail string `json:"customer_email"`
}

// StartPaymentConsumer polls Kafka for payment.requested events to initiate STK push or Stripe intent.
func StartPaymentConsumer(ctx context.Context, db *sql.DB, client *kgo.Client, daraja DarajaClient, stripe StripeClient, useMocks bool) {
	if client == nil {
		return
	}

	log.Println("[Payment Consumer] Subscribing to payment.events topic...")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			fetches := client.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					if err.Err != context.Canceled {
						log.Printf("[Payment Consumer] Fetch error: %v", err.Err)
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

				if strings.Contains(eventType, "payment.requested") || eventType == "PaymentRequested" {
					if err := handlePaymentRequested(ctx, db, record.Key, record.Value, daraja, stripe, useMocks); err != nil {
						log.Printf("[Payment Consumer] Failed to process %s: %v", eventType, err)
					}
				}
			})
		}
	}
}

func handlePaymentRequested(ctx context.Context, db *sql.DB, key, value []byte, daraja DarajaClient, stripe StripeClient, useMocks bool) error {
	var payload PaymentRequestPayload
	if err := json.Unmarshal(value, &payload); err != nil {
		return err
	}

	if payload.OrderID == "" {
		payload.OrderID = string(key)
	}
	if payload.PaymentID == "" {
		payload.PaymentID = uuid.New().String()
	}

	eventID := payload.EventID
	if eventID == "" {
		eventID = payload.PaymentID + "-payment.requested"
	}

	// Idempotency check
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

	if _, err := tx.ExecContext(ctx, "INSERT INTO idempotent_events (event_id, consumer_name) VALUES ($1, $2)", eventID, "payment-saga-consumer"); err != nil {
		return err
	}

	// Insert or fetch existing payment record
	var existingPaymentID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM payments WHERE order_id = $1 AND method = $2", payload.OrderID, payload.PaymentMethod).Scan(&existingPaymentID)
	if err == sql.ErrNoRows {
		insertQuery := `
			INSERT INTO payments (id, order_id, amount_cents, currency, method, status, customer_phone, customer_email)
			VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)
		`
		curr := payload.Currency
		if curr == "" {
			curr = "KES"
		}
		if _, err := tx.ExecContext(ctx, insertQuery, payload.PaymentID, payload.OrderID, payload.AmountCents, curr, payload.PaymentMethod, payload.CustomerPhone, payload.CustomerEmail); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		payload.PaymentID = existingPaymentID
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// External provider dispatch
	var providerRef string
	var reqErr error

	switch payload.PaymentMethod {
	case "mpesa":
		resp, err := daraja.InitiateSTKPush(ctx, payload.OrderID, payload.AmountCents, payload.CustomerPhone)
		if err != nil {
			reqErr = err
		} else {
			providerRef = resp.CheckoutRequestID
		}
	case "stripe":
		resp, err := stripe.CreatePaymentIntent(ctx, payload.OrderID, payload.AmountCents, payload.Currency, payload.CustomerEmail)
		if err != nil {
			reqErr = err
		} else {
			providerRef = resp.ID
		}
	}

	// Update DB with provider reference or failure
	if reqErr != nil {
		log.Printf("[Payment Consumer] Provider dispatch failed for order %s: %v", payload.OrderID, reqErr)
		txFail, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			return err
		}
		defer txFail.Rollback()

		_, _ = txFail.ExecContext(ctx, "UPDATE payments SET status = 'failed', failure_reason = $1, updated_at = now() WHERE id = $2", reqErr.Error(), payload.PaymentID)

		failPayload := map[string]any{
			"payment_id":     payload.PaymentID,
			"order_id":       payload.OrderID,
			"reason":         reqErr.Error(),
			"payment_method": payload.PaymentMethod,
			"failed_at":      time.Now().UnixMilli(),
		}
		_ = QueueOutboxEvent(ctx, txFail, payload.OrderID, "payment.failed", "payment.events", failPayload, uuid.New().String())
		return txFail.Commit()
	}

	if providerRef != "" {
		_, _ = db.ExecContext(ctx, "UPDATE payments SET provider_reference = $1, updated_at = now() WHERE id = $2", providerRef, payload.PaymentID)
	}

	return nil
}
