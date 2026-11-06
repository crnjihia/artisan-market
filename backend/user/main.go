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
	userpb "github.com/yourorg/artisan-market/proto/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	dbDSN := getEnv("USER_DB_DSN", getEnv("DB_DSN", "postgres://postgres:postgres@localhost:5432/user_db?sslmode=disable"))
	serviceAddr := getEnv("USER_SERVICE_ADDR", ":50051")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-artisan-jwt-key-change-in-production")
	accessTTL := getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	refreshTTL := getDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour)

	log.Printf("[User Service] Connecting to database...")
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		log.Fatalf("[User Service] Failed to open DB: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Attempt connection with simple retry loop
	for i := 1; i <= 5; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("[User Service] Waiting for PostgreSQL (attempt %d/5)...", i)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("[User Service] Warning: cannot ping DB yet: %v", err)
	} else {
		if err := runMigrations(db); err != nil {
			log.Fatalf("[User Service] Migration failed: %v", err)
		}
		log.Printf("[User Service] Database migrations applied successfully")
	}

	srv := grpc.NewServer()
	userServer := NewUserServiceServer(db, jwtSecret, accessTTL, refreshTTL)
	userpb.RegisterUserServiceServer(srv, userServer)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", serviceAddr)
	if err != nil {
		log.Fatalf("[User Service] Failed to bind to %s: %v", serviceAddr, err)
	}

	go func() {
		log.Printf("[User Service] gRPC listening on %s", serviceAddr)
		if err := srv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("[User Service] gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[User Service] Shutting down gracefully...")
	srv.GracefulStop()
	log.Println("[User Service] Stopped")
}

func runMigrations(db *sql.DB) error {
	ddl := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT NOT NULL,
		phone TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'buyer',
		artisan_bio TEXT,
		region TEXT,
		mpesa_number TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TIMESTAMPTZ NOT NULL,
		revoked BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
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
