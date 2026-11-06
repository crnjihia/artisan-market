// path: backend/payment/server_test.go
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	paymentpb "github.com/yourorg/artisan-market/proto/payment/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNormalizeKenyanPhoneDaraja(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{input: "+254712345678", expected: "254712345678", wantErr: false},
		{input: "0712345678", expected: "254712345678", wantErr: false},
		{input: "254712345678", expected: "254712345678", wantErr: false},
		{input: "0112345678", expected: "254112345678", wantErr: false},
		{input: "+254 712 345 678", expected: "254712345678", wantErr: false},
		{input: "0712-345-678", expected: "254712345678", wantErr: false},
		{input: "020123456", expected: "", wantErr: true},    // landline
		{input: "+12025550123", expected: "", wantErr: true}, // US number
		{input: "invalid", expected: "", wantErr: true},
	}

	for _, tc := range tests {
		actual, err := NormalizeKenyanPhoneDaraja(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("NormalizeKenyanPhoneDaraja(%q) expected error, got nil", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("NormalizeKenyanPhoneDaraja(%q) unexpected error: %v", tc.input, err)
		}
		if actual != tc.expected {
			t.Errorf("NormalizeKenyanPhoneDaraja(%q) = %q; want %q", tc.input, actual, tc.expected)
		}
	}
}

func TestDarajaClient_MockMode(t *testing.T) {
	client := NewDarajaClient(DarajaConfig{UseMocks: true})
	resp, err := client.InitiateSTKPush(context.Background(), "ord-123", 250000, "+254712345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ResponseCode != "0" {
		t.Errorf("expected ResponseCode 0, got %s", resp.ResponseCode)
	}
	if len(resp.CheckoutRequestID) < 10 {
		t.Errorf("expected non-empty CheckoutRequestID, got %s", resp.CheckoutRequestID)
	}
}

func TestStripeClient_MockMode(t *testing.T) {
	client := NewStripeClient(StripeConfig{UseMocks: true})
	resp, err := client.CreatePaymentIntent(context.Background(), "ord-123", 250000, "kes", "buyer@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID == "" || resp.ClientSecret == "" {
		t.Fatalf("expected non-empty ID and ClientSecret")
	}
	if resp.Status != "requires_payment_method" {
		t.Errorf("expected requires_payment_method, got %s", resp.Status)
	}
}

func TestVerifyStripeWebhookSignature(t *testing.T) {
	payload := []byte(`{"id":"evt_123","type":"payment_intent.succeeded"}`)
	secret := "whsec_test_secret_123"
	now := time.Now().Unix()

	signedPayload := fmt.Sprintf("%d.%s", now, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	sigHeader := fmt.Sprintf("t=%d,v1=%s", now, sig)

	// Valid signature
	err := VerifyStripeWebhookSignature(payload, sigHeader, secret, 5*time.Minute)
	if err != nil {
		t.Fatalf("expected signature to be valid, got: %v", err)
	}

	// Tampered payload
	tampered := []byte(`{"id":"evt_123","type":"payment_intent.failed"}`)
	err = VerifyStripeWebhookSignature(tampered, sigHeader, secret, 5*time.Minute)
	if err == nil {
		t.Fatal("expected tampered signature to fail, got nil")
	}

	// Missing header
	err = VerifyStripeWebhookSignature(payload, "", secret, 5*time.Minute)
	if err == nil {
		t.Fatal("expected empty header error, got nil")
	}
}

func TestCreatePayment_Success_Mpesa(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer db.Close()

	daraja := NewDarajaClient(DarajaConfig{UseMocks: true})
	stripe := NewStripeClient(StripeConfig{UseMocks: true})
	srv := NewPaymentServer(db, daraja, stripe, "test-secret", true)

	orderID := "3b333a92-b43a-4228-a400-9856ad8ce294"

	mock.ExpectQuery("SELECT id, status, provider_reference FROM payments WHERE order_id = \\$1 AND method = \\$2").
		WithArgs(orderID, "mpesa").
		WillReturnError(sql.ErrNoRows)

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO payments").
		WithArgs(sqlmock.AnyArg(), orderID, int64(350000), "KES", "mpesa", sqlmock.AnyArg(), "+254712345678", "").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "payment.requested", "payment.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &paymentpb.CreatePaymentRequest{
		OrderId:       orderID,
		AmountCents:   350000,
		Currency:      "KES",
		Method:        "mpesa",
		CustomerPhone: "+254712345678",
	}

	resp, err := srv.CreatePayment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}

	if resp.PaymentId == "" {
		t.Errorf("expected non-empty payment ID")
	}
	if resp.Status != "pending" {
		t.Errorf("expected status 'pending', got %s", resp.Status)
	}
	if resp.ProviderReference == "" {
		t.Errorf("expected non-empty provider reference")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestProcessMpesaCallback_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer db.Close()

	daraja := NewDarajaClient(DarajaConfig{UseMocks: true})
	stripe := NewStripeClient(StripeConfig{UseMocks: true})
	srv := NewPaymentServer(db, daraja, stripe, "test-secret", true)

	checkoutID := "ws_CO_MOCK_123456789"
	paymentID := "9e81b674-0f2c-497d-a1c6-218151f1ff96"
	orderID := "3b333a92-b43a-4228-a400-9856ad8ce294"

	mock.ExpectBegin()

	mock.ExpectQuery("SELECT event_id FROM idempotent_events WHERE event_id = \\$1").
		WithArgs("mpesa-cb-" + checkoutID).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("mpesa-cb-"+checkoutID, "mpesa-callback-handler").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT id, order_id, amount_cents FROM payments WHERE provider_reference = \\$1 FOR UPDATE").
		WithArgs(checkoutID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "amount_cents"}).AddRow(paymentID, orderID, int64(350000)))

	mock.ExpectExec("UPDATE payments").
		WithArgs("NLJ7RT61SV", paymentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "payment.succeeded", "payment.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &paymentpb.ProcessMpesaCallbackRequest{
		CheckoutRequestId:  checkoutID,
		ResultCode:         0,
		ResultDesc:         "The service request is processed successfully.",
		MpesaReceiptNumber: "NLJ7RT61SV",
		AmountCents:        350000,
		PhoneNumber:        "254712345678",
	}

	resp, err := srv.ProcessMpesaCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessMpesaCallback failed: %v", err)
	}

	if !resp.Acknowledged {
		t.Errorf("expected Acknowledged to be true")
	}
	if resp.Status != "succeeded" {
		t.Errorf("expected status 'succeeded', got %s", resp.Status)
	}
	if resp.PaymentId != paymentID {
		t.Errorf("expected payment ID %s, got %s", paymentID, resp.PaymentId)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestProcessMpesaCallback_CancelledByUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer db.Close()

	daraja := NewDarajaClient(DarajaConfig{UseMocks: true})
	stripe := NewStripeClient(StripeConfig{UseMocks: true})
	srv := NewPaymentServer(db, daraja, stripe, "test-secret", true)

	checkoutID := "ws_CO_MOCK_USER_CANCEL"
	paymentID := "9e81b674-0f2c-497d-a1c6-218151f1ff96"
	orderID := "3b333a92-b43a-4228-a400-9856ad8ce294"

	mock.ExpectBegin()

	mock.ExpectQuery("SELECT event_id FROM idempotent_events WHERE event_id = \\$1").
		WithArgs("mpesa-cb-" + checkoutID).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("mpesa-cb-"+checkoutID, "mpesa-callback-handler").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT id, order_id, amount_cents FROM payments WHERE provider_reference = \\$1 FOR UPDATE").
		WithArgs(checkoutID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "amount_cents"}).AddRow(paymentID, orderID, int64(350000)))

	mock.ExpectExec("UPDATE payments").
		WithArgs("Request cancelled by user.", paymentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "payment.failed", "payment.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &paymentpb.ProcessMpesaCallbackRequest{
		CheckoutRequestId: checkoutID,
		ResultCode:        1032,
		ResultDesc:        "Request cancelled by user.",
	}

	resp, err := srv.ProcessMpesaCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessMpesaCallback failed: %v", err)
	}

	if !resp.Acknowledged {
		t.Errorf("expected Acknowledged to be true")
	}
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %s", resp.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestProcessStripeWebhook_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer db.Close()

	secret := "whsec_test_secret_123"
	srv := NewPaymentServer(db, nil, nil, secret, true)

	piID := "pi_3MtwBwLkdIwHu7ix28a3tqPa"
	paymentID := "9e81b674-0f2c-497d-a1c6-218151f1ff96"
	orderID := "3b333a92-b43a-4228-a400-9856ad8ce294"

	payload := fmt.Sprintf(`{"id":"evt_test_123","type":"payment_intent.succeeded","data":{"object":{"id":"%s","amount":250000,"currency":"kes","status":"succeeded","metadata":{"order_id":"%s"}}}}`, piID, orderID)

	now := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.%s", now, payload)))
	sig := hex.EncodeToString(mac.Sum(nil))
	sigHeader := fmt.Sprintf("t=%d,v1=%s", now, sig)

	mock.ExpectBegin()

	mock.ExpectQuery("SELECT event_id FROM idempotent_events WHERE event_id = \\$1").
		WithArgs("stripe-evt-evt_test_123").
		WillReturnError(sql.ErrNoRows)

	mock.ExpectExec("INSERT INTO idempotent_events").
		WithArgs("stripe-evt-evt_test_123", "stripe-webhook-handler").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT id, order_id, amount_cents FROM payments WHERE provider_reference = \\$1 OR order_id = \\$2 FOR UPDATE").
		WithArgs(piID, orderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "amount_cents"}).AddRow(paymentID, orderID, int64(250000)))

	mock.ExpectExec("UPDATE payments SET status = 'succeeded'").
		WithArgs(paymentID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO outbox").
		WithArgs(orderID, "payment.succeeded", "payment.events", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &paymentpb.ProcessStripeWebhookRequest{
		PayloadJson:     payload,
		SignatureHeader: sigHeader,
	}

	resp, err := srv.ProcessStripeWebhook(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessStripeWebhook failed: %v", err)
	}

	if !resp.Acknowledged {
		t.Errorf("expected Acknowledged to be true")
	}
	if resp.Status != "succeeded" {
		t.Errorf("expected status 'succeeded', got %s", resp.Status)
	}
	if resp.PaymentId != paymentID {
		t.Errorf("expected payment ID %s, got %s", paymentID, resp.PaymentId)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}

func TestGetPayment_Found_And_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock error: %v", err)
	}
	defer db.Close()

	srv := NewPaymentServer(db, nil, nil, "", true)

	paymentID := "9e81b674-0f2c-497d-a1c6-218151f1ff96"
	orderID := "3b333a92-b43a-4228-a400-9856ad8ce294"
	now := time.Now()

	// Found query
	mock.ExpectQuery("SELECT id, order_id, amount_cents, currency, method, status, provider_reference, customer_phone, failure_reason, created_at, updated_at FROM payments").
		WithArgs(paymentID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "order_id", "amount_cents", "currency", "method", "status",
			"provider_reference", "customer_phone", "failure_reason", "created_at", "updated_at",
		}).AddRow(paymentID, orderID, int64(150000), "KES", "mpesa", "succeeded", "NLJ7RT61SV", "+254712345678", nil, now, now))

	res, err := srv.GetPayment(context.Background(), &paymentpb.GetPaymentRequest{PaymentId: paymentID})
	if err != nil {
		t.Fatalf("GetPayment failed: %v", err)
	}
	if res.Payment.Id != paymentID || res.Payment.Status != "succeeded" {
		t.Errorf("unexpected payment payload: %+v", res.Payment)
	}

	// Not Found query
	mock.ExpectQuery("SELECT id, order_id, amount_cents, currency, method, status, provider_reference, customer_phone, failure_reason, created_at, updated_at FROM payments").
		WithArgs("non-existent-id").
		WillReturnError(sql.ErrNoRows)

	_, err = srv.GetPayment(context.Background(), &paymentpb.GetPaymentRequest{PaymentId: "non-existent-id"})
	if err == nil {
		t.Fatalf("expected error for non-existent payment, got nil")
	}
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected NotFound code, got %v", status.Code(err))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sql expectations: %v", err)
	}
}
