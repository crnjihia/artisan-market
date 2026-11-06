// path: backend/payment/main.go
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/twmb/franz-go/pkg/kgo"
	paymentpb "github.com/yourorg/artisan-market/proto/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	dbDSN := getEnv("PAYMENT_DB_DSN", "postgres://postgres:postgres@localhost:5432/payment_db?sslmode=disable")
	grpcAddr := getEnv("PAYMENT_SERVICE_ADDR", ":50054")
	httpAddr := getEnv("PAYMENT_HTTP_ADDR", ":8084")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	outboxPollInterval := getDurationEnv("OUTBOX_POLL_INTERVAL", 2*time.Second)
	useMocks := getEnv("USE_MOCKS", "true") == "true"

	darajaConsumerKey := getEnv("DARAJA_CONSUMER_KEY", "")
	darajaConsumerSecret := getEnv("DARAJA_CONSUMER_SECRET", "")
	darajaPasskey := getEnv("DARAJA_PASSKEY", "")
	darajaShortcode := getEnv("DARAJA_SHORTCODE", "174379")
	darajaCallbackURL := getEnv("DARAJA_CALLBACK_URL", "http://localhost:8084/mpesa/callback")
	stripeSecretKey := getEnv("STRIPE_SECRET_KEY", "")
	stripeWebhookSecret := getEnv("STRIPE_WEBHOOK_SECRET", "")

	log.Printf("[Payment Service] Starting (useMocks=%v, grpc=%s, http=%s)", useMocks, grpcAddr, httpAddr)

	// Database initialization
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		log.Fatalf("[Payment Service] Failed to open DB: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 1; i <= 5; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("[Payment Service] Waiting for PostgreSQL (attempt %d/5)...", i)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[Payment Service] Warning: cannot ping DB yet: %v", err)
	} else {
		if err := runMigrations(db); err != nil {
			log.Fatalf("[Payment Service] Migration failed: %v", err)
		}
		log.Printf("[Payment Service] Database migrations applied successfully")
	}

	// Providers
	darajaClient := NewDarajaClient(DarajaConfig{
		ConsumerKey:    darajaConsumerKey,
		ConsumerSecret: darajaConsumerSecret,
		Passkey:        darajaPasskey,
		ShortCode:      darajaShortcode,
		CallbackURL:    darajaCallbackURL,
		UseMocks:       useMocks,
	})

	stripeClient := NewStripeClient(StripeConfig{
		SecretKey:     stripeSecretKey,
		WebhookSecret: stripeWebhookSecret,
		UseMocks:      useMocks,
	})

	server := NewPaymentServer(db, darajaClient, stripeClient, stripeWebhookSecret, useMocks)

	// Kafka client & background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	brokerList := strings.Split(kafkaBrokers, ",")
	kgoClient, err := kgo.NewClient(
		kgo.SeedBrokers(brokerList...),
		kgo.ConsumerGroup("payment-saga-service-group"),
		kgo.ConsumeTopics("payment.events"),
	)
	if err != nil {
		log.Printf("[Payment Service] Warning: Kafka client init error: %v", err)
	} else {
		defer kgoClient.Close()
		go StartOutboxRelay(ctx, db, kgoClient, outboxPollInterval)
		go StartPaymentConsumer(ctx, db, kgoClient, darajaClient, stripeClient, useMocks)
	}

	// HTTP Callback & Webhook Server
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"payment"}`))
	})
	httpMux.HandleFunc("/mpesa/callback", makeMpesaHTTPHandler(server))
	httpMux.HandleFunc("/stripe/webhook", makeStripeHTTPHandler(server))

	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: httpMux,
	}
	go func() {
		log.Printf("[Payment Service] HTTP callbacks listening on %s", httpAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Payment Service] HTTP server error: %v", err)
		}
	}()

	// gRPC Server
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("[Payment Service] Failed to listen on %s: %v", grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("[Payment Service] gRPC listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[Payment Service] gRPC server failed: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("[Payment Service] Shutting down gracefully...")
	cancel()
	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)

	log.Println("[Payment Service] Stopped.")
}

func makeMpesaHTTPHandler(server *PaymentServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var cb DarajaCallbackPayload
		if err := json.Unmarshal(bodyBytes, &cb); err != nil {
			http.Error(w, "Invalid callback JSON", http.StatusBadRequest)
			return
		}

		stk := cb.Body.StkCallback
		var receiptNum, phone string
		var amountCents int64

		if stk.CallbackMetadata != nil {
			for _, item := range stk.CallbackMetadata.Item {
				switch item.Name {
				case "MpesaReceiptNumber":
					receiptNum = fmt.Sprintf("%v", item.Value)
				case "PhoneNumber":
					phone = fmt.Sprintf("%v", item.Value)
				case "Amount":
					switch v := item.Value.(type) {
					case float64:
						amountCents = int64(v * 100)
					case int64:
						amountCents = v * 100
					case string:
						if f, err := strconv.ParseFloat(v, 64); err == nil {
							amountCents = int64(f * 100)
						}
					}
				}
			}
		}

		req := &paymentpb.ProcessMpesaCallbackRequest{
			CheckoutRequestId:  stk.CheckoutRequestID,
			ResultCode:         stk.ResultCode,
			ResultDesc:         stk.ResultDesc,
			MpesaReceiptNumber: receiptNum,
			AmountCents:        amountCents,
			PhoneNumber:        phone,
		}

		resp, err := server.ProcessMpesaCallback(r.Context(), req)
		if err != nil {
			log.Printf("[Payment HTTP] ProcessMpesaCallback error: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ResultCode": 0,
			"ResultDesc": "Accepted",
			"status":     resp.GetStatus(),
		})
	}
}

func makeStripeHTTPHandler(server *PaymentServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		sigHeader := r.Header.Get("Stripe-Signature")
		req := &paymentpb.ProcessStripeWebhookRequest{
			PayloadJson:     string(bodyBytes),
			SignatureHeader: sigHeader,
		}

		resp, err := server.ProcessStripeWebhook(r.Context(), req)
		if err != nil {
			log.Printf("[Payment HTTP] ProcessStripeWebhook error: %v", err)
			http.Error(w, fmt.Sprintf("Webhook processing error: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"received": true,
			"status":   resp.GetStatus(),
		})
	}
}

func runMigrations(db *sql.DB) error {
	ddl := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS payments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		order_id UUID NOT NULL,
		amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
		currency VARCHAR(3) NOT NULL DEFAULT 'KES',
		method VARCHAR(32) NOT NULL CHECK (method IN ('mpesa', 'stripe')),
		status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'succeeded', 'failed', 'timed_out')),
		provider_reference VARCHAR(255),
		mpesa_receipt VARCHAR(64),
		customer_phone VARCHAR(32),
		customer_email VARCHAR(255),
		failure_reason TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
	CREATE INDEX IF NOT EXISTS idx_payments_provider_reference ON payments(provider_reference);

	CREATE TABLE IF NOT EXISTS outbox (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aggregate_id VARCHAR(255) NOT NULL,
		event_type VARCHAR(255) NOT NULL,
		topic VARCHAR(255) NOT NULL,
		payload JSONB NOT NULL,
		headers JSONB,
		published BOOLEAN NOT NULL DEFAULT false,
		retry_count INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		published_at TIMESTAMPTZ
	);

	CREATE INDEX IF NOT EXISTS idx_payment_outbox_unpublished ON outbox(created_at ASC) WHERE published = false;

	CREATE TABLE IF NOT EXISTS idempotent_events (
		event_id VARCHAR(255) PRIMARY KEY,
		consumer_name VARCHAR(255) NOT NULL,
		processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);
	`
	_, err := db.Exec(ddl)
	return err
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
