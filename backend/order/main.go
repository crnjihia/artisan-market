package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/twmb/franz-go/pkg/kgo"
	orderpb "github.com/yourorg/artisan-market/proto/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	dbDSN := getEnv("ORDER_DB_DSN", "postgres://postgres:postgres@localhost:5432/order_db?sslmode=disable")
	serviceAddr := getEnv("ORDER_SERVICE_ADDR", ":50053")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	outboxPoll := getDurationEnv("OUTBOX_POLL_INTERVAL", 2*time.Second)

	log.Printf("[Order Service] Connecting to database...")
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		log.Fatalf("[Order Service] Failed to open DB: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 1; i <= 5; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("[Order Service] Waiting for PostgreSQL (attempt %d/5)...", i)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[Order Service] Warning: cannot ping DB yet: %v", err)
	} else {
		if err := runMigrations(db); err != nil {
			log.Fatalf("[Order Service] Migration failed: %v", err)
		}
		log.Printf("[Order Service] Database migrations applied successfully")
	}

	// Initialize Kafka client
	var kgoClient *kgo.Client
	brokerList := strings.Split(kafkaBrokers, ",")
	kgoClient, err = kgo.NewClient(
		kgo.SeedBrokers(brokerList...),
		kgo.ConsumerGroup("order-saga-service-group"),
		kgo.ConsumeTopics("inventory.events", "payment.events"),
	)
	if err != nil {
		log.Printf("[Order Service] Warning: Kafka client initialization error: %v", err)
	} else {
		defer kgoClient.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch background outbox relay & saga consumer
	go StartOutboxRelay(ctx, db, kgoClient, outboxPoll)
	go StartSagaConsumer(ctx, db, kgoClient)

	// gRPC Server setup
	srv := grpc.NewServer()
	orderServer := NewOrderServiceServer(db)
	orderpb.RegisterOrderServiceServer(srv, orderServer)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", serviceAddr)
	if err != nil {
		log.Fatalf("[Order Service] Failed to bind to %s: %v", serviceAddr, err)
	}

	go func() {
		log.Printf("[Order Service] gRPC listening on %s", serviceAddr)
		if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("[Order Service] gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[Order Service] Shutting down gracefully...")
	cancel()
	srv.GracefulStop()
	log.Println("[Order Service] Stopped")
}

func runMigrations(db *sql.DB) error {
	ddl := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS orders (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id TEXT NOT NULL,
		total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
		currency TEXT NOT NULL DEFAULT 'KES',
		status TEXT NOT NULL DEFAULT 'pending',
		payment_method TEXT NOT NULL,
		customer_phone TEXT NOT NULL,
		customer_email TEXT,
		saga_step TEXT NOT NULL DEFAULT 'ORDER_CREATED',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

	CREATE TABLE IF NOT EXISTS order_items (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
		product_id TEXT NOT NULL,
		product_name TEXT NOT NULL,
		quantity INT NOT NULL CHECK (quantity > 0),
		unit_price_cents BIGINT NOT NULL CHECK (unit_price_cents >= 0),
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);

	CREATE TABLE IF NOT EXISTS outbox (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		aggregate_type TEXT NOT NULL DEFAULT 'Order',
		aggregate_id UUID NOT NULL,
		event_type TEXT NOT NULL,
		topic TEXT NOT NULL DEFAULT 'order.events',
		payload JSONB NOT NULL,
		headers JSONB,
		published BOOLEAN NOT NULL DEFAULT false,
		retry_count INT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		published_at TIMESTAMPTZ
	);

	CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox(created_at) WHERE published = false;

	CREATE TABLE IF NOT EXISTS idempotent_events (
		event_id TEXT PRIMARY KEY,
		consumer_name TEXT NOT NULL,
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
