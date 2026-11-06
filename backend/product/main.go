package main

import (
	"database/sql"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	productpb "github.com/yourorg/artisan-market/proto/product/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	dbDSN := getEnv("PRODUCT_DB_DSN", "postgres://postgres:postgres@localhost:5432/product_db?sslmode=disable")
	serviceAddr := getEnv("PRODUCT_SERVICE_ADDR", ":50052")

	log.Printf("[Product Service] Connecting to database...")
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		log.Fatalf("[Product Service] Failed to open DB: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 1; i <= 5; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("[Product Service] Waiting for PostgreSQL (attempt %d/5)...", i)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[Product Service] Warning: cannot ping DB yet: %v", err)
	} else {
		if err := runMigrations(db); err != nil {
			log.Fatalf("[Product Service] Migration failed: %v", err)
		}
		log.Printf("[Product Service] Database migrations applied successfully")
	}

	srv := grpc.NewServer()
	productServer := NewProductServiceServer(db)
	productpb.RegisterProductServiceServer(srv, productServer)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", serviceAddr)
	if err != nil {
		log.Fatalf("[Product Service] Failed to bind to %s: %v", serviceAddr, err)
	}

	go func() {
		log.Printf("[Product Service] gRPC listening on %s", serviceAddr)
		if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("[Product Service] gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[Product Service] Shutting down gracefully...")
	srv.GracefulStop()
	log.Println("[Product Service] Stopped")
}

func runMigrations(db *sql.DB) error {
	ddl := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS products (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		seller_id UUID NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
		inventory INT NOT NULL CHECK (inventory >= 0),
		category TEXT NOT NULL,
		image_url TEXT,
		origin_region TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
	CREATE INDEX IF NOT EXISTS idx_products_seller ON products(seller_id);
	CREATE INDEX IF NOT EXISTS idx_products_region ON products(origin_region);

	CREATE TABLE IF NOT EXISTS inventory_reservations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		order_id UUID NOT NULL,
		product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
		quantity INT NOT NULL CHECK (quantity > 0),
		status TEXT NOT NULL DEFAULT 'reserved',
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_inventory_reservations_order ON inventory_reservations(order_id);
	CREATE INDEX IF NOT EXISTS idx_inventory_reservations_product ON inventory_reservations(product_id);
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
