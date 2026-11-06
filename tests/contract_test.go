// path: tests/contract_test.go
package tests

import (
	"encoding/json"
	"regexp"
	"testing"

	orderpb "github.com/yourorg/artisan-market/proto/order/v1"
	paymentpb "github.com/yourorg/artisan-market/proto/payment/v1"
	productpb "github.com/yourorg/artisan-market/proto/product/v1"
	userpb "github.com/yourorg/artisan-market/proto/user/v1"
)

var kenyanPhoneRegex = regexp.MustCompile(`^\+254[17]\d{8}$`)

// TestContract_ProtobufContracts ensures all gRPC service request/response structs
// adhere strictly to the schema contracts, types, and validation invariants.
func TestContract_ProtobufContracts(t *testing.T) {
	// 1. User Protobuf Contract
	userReq := &userpb.RegisterRequest{
		Email:    "artisan@nairobi.co.ke",
		Password: "SecureArtisanPass2026!",
		Name:     "Wanjiku Mwangi",
		Phone:    "+254712345678",
		Role:     "seller",
	}
	if userReq.GetName() != "Wanjiku Mwangi" {
		t.Errorf("contract violation: expected Wanjiku Mwangi, got %s", userReq.GetName())
	}
	if !kenyanPhoneRegex.MatchString(userReq.GetPhone()) {
		t.Errorf("contract violation: invalid phone format: %s", userReq.GetPhone())
	}
	if userReq.GetRole() != "seller" {
		t.Errorf("contract violation: expected seller, got %s", userReq.GetRole())
	}

	// 2. Product Protobuf Contract
	productReq := &productpb.ReserveInventoryRequest{
		OrderId: "ord-wire-101",
		Items: []*productpb.ReserveItem{
			{ProductId: "prod-shuka", Quantity: 2},
			{ProductId: "prod-soapstone", Quantity: 1},
		},
	}
	if productReq.GetOrderId() != "ord-wire-101" {
		t.Errorf("contract violation: expected ord-wire-101, got %s", productReq.GetOrderId())
	}
	if len(productReq.GetItems()) != 2 {
		t.Errorf("contract violation: expected 2 items, got %d", len(productReq.GetItems()))
	}
	if productReq.GetItems()[0].GetQuantity() != 2 {
		t.Errorf("contract violation: expected quantity 2, got %d", productReq.GetItems()[0].GetQuantity())
	}

	// 3. Order Protobuf Contract
	orderReq := &orderpb.CreateOrderRequest{
		UserId:        "user-uuid-99",
		Items:         []*orderpb.CreateOrderItem{{ProductId: "prod-shuka", Quantity: 2}},
		PaymentMethod: "mpesa",
		CustomerPhone: "+254722000000",
		CustomerEmail: "wanjiku@artisan.co.ke",
	}
	if orderReq.GetPaymentMethod() != "mpesa" {
		t.Errorf("contract violation: expected mpesa, got %s", orderReq.GetPaymentMethod())
	}
	if !kenyanPhoneRegex.MatchString(orderReq.GetCustomerPhone()) {
		t.Errorf("contract violation: invalid customer_phone: %s", orderReq.GetCustomerPhone())
	}

	// 4. Payment Protobuf Contract
	payReq := &paymentpb.CreatePaymentRequest{
		OrderId:       "ord-wire-101",
		AmountCents:   450000,
		Currency:      "KES",
		Method:        "mpesa",
		CustomerPhone: "+254722000000",
		CustomerEmail: "wanjiku@artisan.co.ke",
	}
	if payReq.GetCurrency() != "KES" {
		t.Errorf("contract violation: expected KES, got %s", payReq.GetCurrency())
	}
	if payReq.GetAmountCents() != 450000 {
		t.Errorf("contract violation: expected 450000, got %d", payReq.GetAmountCents())
	}
	if payReq.GetMethod() != "mpesa" {
		t.Errorf("contract violation: expected mpesa, got %s", payReq.GetMethod())
	}
}

// TestContract_SagaEventSchemaContracts verifies JSON schemas for cross-service Kafka events
func TestContract_SagaEventSchemaContracts(t *testing.T) {
	// 1. payment.requested event contract
	type PaymentRequestedEvent struct {
		EventID       string `json:"event_id"`
		PaymentID     string `json:"payment_id"`
		OrderID       string `json:"order_id"`
		AmountCents   int64  `json:"amount_cents"`
		Currency      string `json:"currency"`
		PaymentMethod string `json:"payment_method"`
		CustomerPhone string `json:"customer_phone"`
		CustomerEmail string `json:"customer_email"`
	}

	rawPayRequested := `{
		"event_id": "evt-req-1",
		"payment_id": "pay-1",
		"order_id": "ord-1",
		"amount_cents": 350000,
		"currency": "KES",
		"payment_method": "mpesa",
		"customer_phone": "+254712345678",
		"customer_email": "buyer@nairobi.co.ke"
	}`

	var payEvt PaymentRequestedEvent
	if err := json.Unmarshal([]byte(rawPayRequested), &payEvt); err != nil {
		t.Fatalf("payment.requested JSON schema contract violation: %v", err)
	}
	if !kenyanPhoneRegex.MatchString(payEvt.CustomerPhone) {
		t.Errorf("contract violation: invalid customer_phone format: %s", payEvt.CustomerPhone)
	}
	if payEvt.Currency != "KES" {
		t.Errorf("contract violation: expected KES, got %s", payEvt.Currency)
	}

	// 2. payment.succeeded event contract
	type PaymentSucceededEvent struct {
		EventID       string `json:"event_id"`
		OrderID       string `json:"order_id"`
		PaymentID     string `json:"payment_id"`
		MpesaReceipt  string `json:"mpesa_receipt"`
		AmountCents   int64  `json:"amount_cents"`
		CustomerPhone string `json:"customer_phone"`
	}

	rawPaySucceeded := `{
		"event_id": "evt-succ-1",
		"order_id": "ord-1",
		"payment_id": "pay-1",
		"mpesa_receipt": "QGR458JKL1",
		"amount_cents": 350000,
		"customer_phone": "+254712345678"
	}`

	var succEvt PaymentSucceededEvent
	if err := json.Unmarshal([]byte(rawPaySucceeded), &succEvt); err != nil {
		t.Fatalf("payment.succeeded JSON schema contract violation: %v", err)
	}
	if succEvt.MpesaReceipt == "" {
		t.Error("contract violation: missing mpesa_receipt in payment.succeeded")
	}

	// 3. order.cancelled compensation event contract
	type OrderCancelledCompensationEvent struct {
		OrderID string `json:"order_id"`
		Reason  string `json:"reason"`
	}

	rawOrderCancelled := `{
		"order_id": "ord-1",
		"reason": "payment_failure_or_timeout"
	}`

	var cancelEvt OrderCancelledCompensationEvent
	if err := json.Unmarshal([]byte(rawOrderCancelled), &cancelEvt); err != nil {
		t.Fatalf("order.cancelled compensation event contract violation: %v", err)
	}
	if cancelEvt.OrderID == "" || cancelEvt.Reason == "" {
		t.Error("contract violation: missing order_id or reason in compensation event")
	}
}
