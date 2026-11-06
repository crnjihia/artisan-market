// path: backend/payment/server.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	paymentpb "github.com/yourorg/artisan-market/proto/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PaymentServer implements the gRPC PaymentServiceServer contract.
type PaymentServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	db           *sql.DB
	daraja       DarajaClient
	stripe       StripeClient
	stripeSecret string
	useMocks     bool
}

// NewPaymentServer constructs a PaymentServer instance.
func NewPaymentServer(db *sql.DB, daraja DarajaClient, stripe StripeClient, stripeSecret string, useMocks bool) *PaymentServer {
	return &PaymentServer{
		db:           db,
		daraja:       daraja,
		stripe:       stripe,
		stripeSecret: stripeSecret,
		useMocks:     useMocks,
	}
}

// CreatePayment initiates an M-Pesa STK Push or Stripe PaymentIntent and persists the record.
func (s *PaymentServer) CreatePayment(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.GetAmountCents() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount_cents must be positive")
	}
	method := req.GetMethod()
	if method != "mpesa" && method != "stripe" {
		return nil, status.Error(codes.InvalidArgument, "method must be 'mpesa' or 'stripe'")
	}

	curr := req.GetCurrency()
	if curr == "" {
		curr = "KES"
	}

	// Idempotency: return existing payment if already created for this order and method
	var existingID, existingStatus, existingRef sql.NullString
	checkQuery := `SELECT id, status, provider_reference FROM payments WHERE order_id = $1 AND method = $2`
	err := s.db.QueryRowContext(ctx, checkQuery, req.GetOrderId(), method).Scan(&existingID, &existingStatus, &existingRef)
	if err == nil && existingID.Valid {
		return &paymentpb.CreatePaymentResponse{
			PaymentId:         existingID.String,
			Status:            existingStatus.String,
			ProviderReference: existingRef.String,
		}, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "database lookup error: %v", err)
	}

	paymentID := uuid.New().String()
	var providerRef string

	// Trigger external provider
	switch method {
	case "mpesa":
		phone := req.GetCustomerPhone()
		if phone == "" {
			return nil, status.Error(codes.InvalidArgument, "customer_phone required for mpesa")
		}
		stkResp, err := s.daraja.InitiateSTKPush(ctx, req.GetOrderId(), req.GetAmountCents(), phone)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "mpesa stk push failed: %v", err)
		}
		providerRef = stkResp.CheckoutRequestID

	case "stripe":
		intentResp, err := s.stripe.CreatePaymentIntent(ctx, req.GetOrderId(), req.GetAmountCents(), curr, req.GetCustomerEmail())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "stripe intent failed: %v", err)
		}
		providerRef = intentResp.ID
	}

	// Persist payment record and outbox event atomically
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "transaction error: %v", err)
	}
	defer tx.Rollback()

	insertQuery := `
		INSERT INTO payments (id, order_id, amount_cents, currency, method, status, provider_reference, customer_phone, customer_email)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7, $8)
	`
	if _, err := tx.ExecContext(ctx, insertQuery, paymentID, req.GetOrderId(), req.GetAmountCents(), curr, method, providerRef, req.GetCustomerPhone(), req.GetCustomerEmail()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert payment: %v", err)
	}

	outboxPayload := map[string]any{
		"payment_id":         paymentID,
		"order_id":           req.GetOrderId(),
		"amount_cents":       req.GetAmountCents(),
		"currency":           curr,
		"payment_method":     method,
		"provider_reference": providerRef,
		"customer_phone":     req.GetCustomerPhone(),
		"customer_email":     req.GetCustomerEmail(),
	}
	correlationID := uuid.New().String()
	if err := QueueOutboxEvent(ctx, tx, req.GetOrderId(), "payment.requested", "payment.events", outboxPayload, correlationID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to queue outbox event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit payment: %v", err)
	}

	return &paymentpb.CreatePaymentResponse{
		PaymentId:         paymentID,
		Status:            "pending",
		ProviderReference: providerRef,
	}, nil
}

// GetPayment fetches a payment record by ID.
func (s *PaymentServer) GetPayment(ctx context.Context, req *paymentpb.GetPaymentRequest) (*paymentpb.GetPaymentResponse, error) {
	if req.GetPaymentId() == "" {
		return nil, status.Error(codes.InvalidArgument, "payment_id is required")
	}

	query := `
		SELECT id, order_id, amount_cents, currency, method, status, provider_reference, customer_phone, failure_reason, created_at, updated_at
		FROM payments
		WHERE id = $1 OR order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var (
		id, orderID, currency, method, pStatus string
		amountCents                            int64
		providerRef, phone, failureReason      sql.NullString
		createdAt, updatedAt                   time.Time
	)

	err := s.db.QueryRowContext(ctx, query, req.GetPaymentId()).Scan(
		&id, &orderID, &amountCents, &currency, &method, &pStatus,
		&providerRef, &phone, &failureReason, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "payment not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &paymentpb.GetPaymentResponse{
		Payment: &paymentpb.Payment{
			Id:                id,
			OrderId:           orderID,
			AmountCents:       amountCents,
			Currency:          currency,
			Method:            method,
			Status:            pStatus,
			ProviderReference: providerRef.String,
			CustomerPhone:     phone.String,
			FailureReason:     failureReason.String,
			CreatedAtUnixMs:   createdAt.UnixMilli(),
			UpdatedAtUnixMs:   updatedAt.UnixMilli(),
		},
	}, nil
}

// ProcessMpesaCallback handles asynchronous STK push responses from Safaricom.
func (s *PaymentServer) ProcessMpesaCallback(ctx context.Context, req *paymentpb.ProcessMpesaCallbackRequest) (*paymentpb.ProcessMpesaCallbackResponse, error) {
	if req.GetCheckoutRequestId() == "" {
		return nil, status.Error(codes.InvalidArgument, "checkout_request_id required")
	}

	eventID := fmt.Sprintf("mpesa-cb-%s", req.GetCheckoutRequestId())

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db tx failed: %v", err)
	}
	defer tx.Rollback()

	// Idempotency check
	var existing string
	err = tx.QueryRowContext(ctx, "SELECT event_id FROM idempotent_events WHERE event_id = $1", eventID).Scan(&existing)
	if err == nil {
		return &paymentpb.ProcessMpesaCallbackResponse{
			Acknowledged: true,
			Status:       "already_processed",
		}, nil
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO idempotent_events (event_id, consumer_name) VALUES ($1, $2)", eventID, "mpesa-callback-handler"); err != nil {
		return nil, status.Errorf(codes.Internal, "idempotency insert failed: %v", err)
	}

	// Find matching payment record
	var paymentID, orderID string
	var amountCents int64
	findQuery := `SELECT id, order_id, amount_cents FROM payments WHERE provider_reference = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, findQuery, req.GetCheckoutRequestId()).Scan(&paymentID, &orderID, &amountCents)
	if errors.Is(err, sql.ErrNoRows) {
		// Callback for unknown reference; acknowledge safely to avoid repeated retries from Safaricom
		_ = tx.Commit()
		return &paymentpb.ProcessMpesaCallbackResponse{
			Acknowledged: true,
			Status:       "payment_not_found",
		}, nil
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query payment: %v", err)
	}

	correlationID := uuid.New().String()
	finalStatus := "failed"

	if req.GetResultCode() == 0 {
		finalStatus = "succeeded"
		updateQuery := `
			UPDATE payments
			SET status = 'succeeded', mpesa_receipt = $1, updated_at = now()
			WHERE id = $2
		`
		if _, err := tx.ExecContext(ctx, updateQuery, req.GetMpesaReceiptNumber(), paymentID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update payment: %v", err)
		}

		succeededPayload := map[string]any{
			"payment_id":         paymentID,
			"order_id":           orderID,
			"amount_cents":       amountCents,
			"payment_method":     "mpesa",
			"provider_reference": req.GetMpesaReceiptNumber(),
			"completed_at":       time.Now().UnixMilli(),
		}
		if err := QueueOutboxEvent(ctx, tx, orderID, "payment.succeeded", "payment.events", succeededPayload, correlationID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to queue outbox event: %v", err)
		}
	} else {
		finalStatus = "failed"
		updateQuery := `
			UPDATE payments
			SET status = 'failed', failure_reason = $1, updated_at = now()
			WHERE id = $2
		`
		if _, err := tx.ExecContext(ctx, updateQuery, req.GetResultDesc(), paymentID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update payment: %v", err)
		}

		failedPayload := map[string]any{
			"payment_id":     paymentID,
			"order_id":       orderID,
			"reason":         req.GetResultDesc(),
			"payment_method": "mpesa",
			"failed_at":      time.Now().UnixMilli(),
		}
		if err := QueueOutboxEvent(ctx, tx, orderID, "payment.failed", "payment.events", failedPayload, correlationID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to queue outbox event: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit callback: %v", err)
	}

	return &paymentpb.ProcessMpesaCallbackResponse{
		Acknowledged: true,
		PaymentId:    paymentID,
		Status:       finalStatus,
	}, nil
}

// ProcessStripeWebhook handles Stripe PaymentIntent events with signature verification.
func (s *PaymentServer) ProcessStripeWebhook(ctx context.Context, req *paymentpb.ProcessStripeWebhookRequest) (*paymentpb.ProcessStripeWebhookResponse, error) {
	payload := []byte(req.GetPayloadJson())
	if err := VerifyStripeWebhookSignature(payload, req.GetSignatureHeader(), s.stripeSecret, 5*time.Minute); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid stripe signature: %v", err)
	}

	var event StripeWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "malformed stripe json: %v", err)
	}

	eventID := fmt.Sprintf("stripe-evt-%s", event.ID)
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db tx failed: %v", err)
	}
	defer tx.Rollback()

	// Idempotency check
	var existing string
	err = tx.QueryRowContext(ctx, "SELECT event_id FROM idempotent_events WHERE event_id = $1", eventID).Scan(&existing)
	if err == nil {
		return &paymentpb.ProcessStripeWebhookResponse{
			Acknowledged: true,
			Status:       "already_processed",
		}, nil
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO idempotent_events (event_id, consumer_name) VALUES ($1, $2)", eventID, "stripe-webhook-handler"); err != nil {
		return nil, status.Errorf(codes.Internal, "idempotency insert failed: %v", err)
	}

	piID := event.Data.Object.ID
	orderID := event.Data.Object.Metadata["order_id"]

	var paymentID string
	var amountCents int64
	var dbOrderID string

	findQuery := `SELECT id, order_id, amount_cents FROM payments WHERE provider_reference = $1 OR order_id = $2 FOR UPDATE`
	err = tx.QueryRowContext(ctx, findQuery, piID, orderID).Scan(&paymentID, &dbOrderID, &amountCents)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Commit()
		return &paymentpb.ProcessStripeWebhookResponse{
			Acknowledged: true,
			Status:       "payment_not_found",
		}, nil
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query payment: %v", err)
	}

	correlationID := uuid.New().String()
	finalStatus := "pending"

	switch event.Type {
	case "payment_intent.succeeded":
		finalStatus = "succeeded"
		_, err := tx.ExecContext(ctx, "UPDATE payments SET status = 'succeeded', updated_at = now() WHERE id = $1", paymentID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "update payment failed: %v", err)
		}

		succeededPayload := map[string]any{
			"payment_id":         paymentID,
			"order_id":           dbOrderID,
			"amount_cents":       amountCents,
			"payment_method":     "stripe",
			"provider_reference": piID,
			"completed_at":       time.Now().UnixMilli(),
		}
		if err := QueueOutboxEvent(ctx, tx, dbOrderID, "payment.succeeded", "payment.events", succeededPayload, correlationID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to queue outbox: %v", err)
		}

	case "payment_intent.payment_failed":
		finalStatus = "failed"
		reason := "card_payment_failed"
		if event.Data.Object.LastPaymentError != nil && event.Data.Object.LastPaymentError.Message != "" {
			reason = event.Data.Object.LastPaymentError.Message
		}
		_, err := tx.ExecContext(ctx, "UPDATE payments SET status = 'failed', failure_reason = $1, updated_at = now() WHERE id = $2", reason, paymentID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "update payment failed: %v", err)
		}

		failedPayload := map[string]any{
			"payment_id":     paymentID,
			"order_id":       dbOrderID,
			"reason":         reason,
			"payment_method": "stripe",
			"failed_at":      time.Now().UnixMilli(),
		}
		if err := QueueOutboxEvent(ctx, tx, dbOrderID, "payment.failed", "payment.events", failedPayload, correlationID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to queue outbox: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit webhook tx: %v", err)
	}

	return &paymentpb.ProcessStripeWebhookResponse{
		Acknowledged: true,
		PaymentId:    paymentID,
		Status:       finalStatus,
	}, nil
}
