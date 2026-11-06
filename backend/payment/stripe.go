// path: backend/payment/stripe.go
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StripeConfig holds Stripe API secret keys and webhook secrets.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	BaseURL       string // Defaults to "https://api.stripe.com"
	UseMocks      bool
}

// StripeClient handles PaymentIntent creation.
type StripeClient interface {
	CreatePaymentIntent(ctx context.Context, orderID string, amountCents int64, currency, customerEmail string) (*StripeIntentResponse, error)
}

type stripeClientImpl struct {
	config     StripeConfig
	httpClient *http.Client
}

// StripeIntentResponse represents the Stripe PaymentIntent result.
type StripeIntentResponse struct {
	ID           string `json:"id"`
	ClientSecret string `json:"client_secret"`
	Status       string `json:"status"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
}

// StripeWebhookEvent represents the parsed Stripe webhook payload.
type StripeWebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID               string            `json:"id"`
			Amount           int64             `json:"amount"`
			Currency         string            `json:"currency"`
			Status           string            `json:"status"`
			Metadata         map[string]string `json:"metadata"`
			LastPaymentError *struct {
				Message string `json:"message"`
			} `json:"last_payment_error,omitempty"`
		} `json:"object"`
	} `json:"data"`
}

// NewStripeClient constructs a StripeClient implementation.
func NewStripeClient(cfg StripeConfig) StripeClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.stripe.com"
	}
	return &stripeClientImpl{
		config:     cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *stripeClientImpl) CreatePaymentIntent(ctx context.Context, orderID string, amountCents int64, currency, customerEmail string) (*StripeIntentResponse, error) {
	if s.config.UseMocks || s.config.SecretKey == "" {
		mockID := fmt.Sprintf("pi_mock_%s", uuid.New().String()[:16])
		return &StripeIntentResponse{
			ID:           mockID,
			ClientSecret: fmt.Sprintf("%s_secret_%s", mockID, uuid.New().String()[:8]),
			Status:       "requires_payment_method",
			Amount:       amountCents,
			Currency:     strings.ToLower(currency),
		}, nil
	}

	data := url.Values{}
	data.Set("amount", strconv.FormatInt(amountCents, 10))
	data.Set("currency", strings.ToLower(currency))
	data.Set("metadata[order_id]", orderID)
	if customerEmail != "" {
		data.Set("receipt_email", customerEmail)
	}

	endpoint := fmt.Sprintf("%s/v1/payment_intents", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.config.SecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var intent StripeIntentResponse
	if err := json.Unmarshal(bodyBytes, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse stripe payment intent: %w", err)
	}

	return &intent, nil
}

// VerifyStripeWebhookSignature validates the Stripe-Signature header using HMAC-SHA256.
func VerifyStripeWebhookSignature(payload []byte, sigHeader, secret string, tolerance time.Duration) error {
	if sigHeader == "" {
		return fmt.Errorf("missing Stripe-Signature header")
	}

	// In mock development mode, accept special test signature
	if secret == "" && strings.Contains(sigHeader, "mock_signature") {
		return nil
	}

	var timestampStr string
	var signatures []string

	pairs := strings.Split(sigHeader, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "t":
				timestampStr = parts[1]
			case "v1":
				signatures = append(signatures, parts[1])
			}
		}
	}

	if timestampStr == "" || len(signatures) == 0 {
		return fmt.Errorf("malformed Stripe-Signature header")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid signature timestamp: %w", err)
	}

	eventTime := time.Unix(timestamp, 0)
	if tolerance > 0 && time.Since(eventTime).Abs() > tolerance {
		return fmt.Errorf("webhook timestamp outside tolerance window")
	}

	signedPayload := fmt.Sprintf("%s.%s", timestampStr, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expectedSignature)) {
			return nil
		}
	}

	return fmt.Errorf("stripe signature verification failed")
}

