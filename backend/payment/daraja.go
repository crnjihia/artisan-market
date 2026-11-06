// path: backend/payment/daraja.go
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DarajaConfig holds Safaricom Daraja API credentials and settings.
type DarajaConfig struct {
	ConsumerKey    string
	ConsumerSecret string
	Passkey        string
	ShortCode      string
	CallbackURL    string
	BaseURL        string // Defaults to sandbox "https://sandbox.safaricom.co.ke"
	UseMocks       bool
}

// DarajaClient abstracts STK push requests for real Daraja or local mocks.
type DarajaClient interface {
	InitiateSTKPush(ctx context.Context, orderID string, amountCents int64, phone string) (*DarajaSTKResponse, error)
}

type darajaClientImpl struct {
	config     DarajaConfig
	httpClient *http.Client
}

// DarajaSTKRequest is the JSON payload sent to Daraja Lipa Na M-Pesa Online.
type DarajaSTKRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int64  `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// DarajaSTKResponse is the response from Daraja processrequest endpoint.
type DarajaSTKResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// DarajaCallbackPayload represents the inbound STK callback body from Safaricom.
type DarajaCallbackPayload struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int32  `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  *struct {
				Item []struct {
					Name  string `json:"Name"`
					Value any    `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata,omitempty"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

// NewDarajaClient constructs a DarajaClient based on configuration.
func NewDarajaClient(cfg DarajaConfig) DarajaClient {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://sandbox.safaricom.co.ke"
	}
	if cfg.ShortCode == "" {
		cfg.ShortCode = "174379" // Daraja standard sandbox shortcode
	}
	return &darajaClientImpl{
		config:     cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// NormalizeKenyanPhoneDaraja formats phone numbers to 2547XXXXXXXX or 2541XXXXXXXX.
func NormalizeKenyanPhoneDaraja(raw string) (string, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	if strings.HasPrefix(cleaned, "+254") {
		cleaned = cleaned[1:]
	} else if strings.HasPrefix(cleaned, "07") || strings.HasPrefix(cleaned, "01") {
		cleaned = "254" + cleaned[1:]
	}

	re := regexp.MustCompile(`^254(7|1)\d{8}$`)
	if !re.MatchString(cleaned) {
		return "", fmt.Errorf("invalid Kenyan phone number for Daraja: %s (must be 2547XXXXXXXX or 2541XXXXXXXX)", raw)
	}

	return cleaned, nil
}

func (c *darajaClientImpl) InitiateSTKPush(ctx context.Context, orderID string, amountCents int64, phone string) (*DarajaSTKResponse, error) {
	formattedPhone, err := NormalizeKenyanPhoneDaraja(phone)
	if err != nil {
		return nil, err
	}

	// In Mock Mode or if credentials are missing, return a deterministic mock response
	if c.config.UseMocks || c.config.ConsumerKey == "" || c.config.ConsumerSecret == "" {
		mockCheckoutID := fmt.Sprintf("ws_CO_MOCK_%s", uuid.New().String()[:18])
		return &DarajaSTKResponse{
			MerchantRequestID:   fmt.Sprintf("MOCK-REQ-%s", uuid.New().String()[:8]),
			CheckoutRequestID:   mockCheckoutID,
			ResponseCode:        "0",
			ResponseDescription: "Success. Request accepted for processing (MOCK)",
			CustomerMessage:     fmt.Sprintf("Success. STK Push sent to %s for KES %d", formattedPhone, amountCents/100),
		}, nil
	}

	// 1. Fetch OAuth Access Token
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("daraja auth failed: %w", err)
	}

	// 2. Compute Timestamp & Password
	timestamp := time.Now().UTC().Format("20060102150405")
	rawPass := fmt.Sprintf("%s%s%s", c.config.ShortCode, c.config.Passkey, timestamp)
	password := base64.StdEncoding.EncodeToString([]byte(rawPass))

	kesAmount := amountCents / 100
	if kesAmount <= 0 {
		kesAmount = 1
	}

	reqBody := DarajaSTKRequest{
		BusinessShortCode: c.config.ShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            kesAmount,
		PartyA:            formattedPhone,
		PartyB:            c.config.ShortCode,
		PhoneNumber:       formattedPhone,
		CallBackURL:       c.config.CallbackURL,
		AccountReference:  orderID[:min(len(orderID), 12)],
		TransactionDesc:   "Artisan Market Order",
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/mpesa/stkpush/v1/processrequest", c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("daraja stk request failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var stkRes DarajaSTKResponse
	if err := json.Unmarshal(bodyBytes, &stkRes); err != nil {
		return nil, fmt.Errorf("failed to parse daraja response: %s", string(bodyBytes))
	}

	if stkRes.ResponseCode != "0" {
		return &stkRes, fmt.Errorf("daraja error: code=%s desc=%s", stkRes.ResponseCode, stkRes.ResponseDescription)
	}

	return &stkRes, nil
}

func (c *darajaClientImpl) getAccessToken(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/oauth/v1/generate?grant_type=client_credentials", c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.config.ConsumerKey + ":" + c.config.ConsumerSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("daraja oauth status %d: %s", resp.StatusCode, string(b))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

