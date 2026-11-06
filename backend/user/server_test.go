package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	userpb "github.com/yourorg/artisan-market/proto/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRegister_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewUserServiceServer(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("artisan@kisii.co.ke", sqlmock.AnyArg(), "Mogaka Nyabuto", "+254712345678", "seller").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("user-uuid-1", time.Now()))

	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs("user-uuid-1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := &userpb.RegisterRequest{
		Email:    "artisan@kisii.co.ke",
		Password: "SecurePassword123!",
		Name:     "Mogaka Nyabuto",
		Phone:    "0712345678", // Should normalize to +254712345678
		Role:     "seller",
	}

	resp, err := server.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.GetUserId() != "user-uuid-1" {
		t.Errorf("expected user id user-uuid-1, got %s", resp.GetUserId())
	}
	if resp.GetAccessToken() == "" {
		t.Error("expected non-empty access token")
	}
	if resp.GetRefreshToken() == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.GetProfile().GetPhone() != "+254712345678" {
		t.Errorf("expected normalized phone +254712345678, got %s", resp.GetProfile().GetPhone())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

func TestRegister_InvalidPhone(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewUserServiceServer(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	req := &userpb.RegisterRequest{
		Email:    "buyer@nairobi.co.ke",
		Password: "SecurePassword123!",
		Name:     "Wanjiru Mwangi",
		Phone:    "+15551234567", // Non-Kenyan number
		Role:     "buyer",
	}

	_, err = server.Register(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid phone, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument code, got %v", st.Code())
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewUserServiceServer(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("artisan@kisii.co.ke", sqlmock.AnyArg(), "Mogaka Nyabuto", "+254712345678", "seller").
		WillReturnError(errors.New(`ERROR: duplicate key value violates unique constraint "users_email_key"`))

	req := &userpb.RegisterRequest{
		Email:    "artisan@kisii.co.ke",
		Password: "SecurePassword123!",
		Name:     "Mogaka Nyabuto",
		Phone:    "+254712345678",
		Role:     "seller",
	}

	_, err = server.Register(context.Background(), req)
	if err == nil {
		t.Fatal("expected AlreadyExists error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists status code, got %v", st.Code())
	}
}

func TestLogin_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewUserServiceServer(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	hashedPassword, err := hashPassword("MyValidPassword123")
	if err != nil {
		t.Fatalf("failed to create test hash: %v", err)
	}

	rows := sqlmock.NewRows([]string{
		"id", "password_hash", "name", "phone", "role", "artisan_bio", "region", "mpesa_number", "created_at",
	}).AddRow(
		"user-uuid-1", hashedPassword, "Mogaka Nyabuto", "+254712345678", "seller",
		"Master soapstone carver from Tabaka, Kisii", "Kisii", "+254712345678", time.Now(),
	)

	mock.ExpectQuery("SELECT id, password_hash, name, phone, role").
		WithArgs("artisan@kisii.co.ke").
		WillReturnRows(rows)

	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs("user-uuid-1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := &userpb.LoginRequest{
		Email:    "artisan@kisii.co.ke",
		Password: "MyValidPassword123",
	}

	resp, err := server.Login(context.Background(), req)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if resp.GetUserId() != "user-uuid-1" {
		t.Errorf("expected user id user-uuid-1, got %s", resp.GetUserId())
	}
	if resp.GetProfile().GetRegion() != "Kisii" {
		t.Errorf("expected region Kisii, got %s", resp.GetProfile().GetRegion())
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewUserServiceServer(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	hashedPassword, err := hashPassword("CorrectPassword123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	rows := sqlmock.NewRows([]string{
		"id", "password_hash", "name", "phone", "role", "artisan_bio", "region", "mpesa_number", "created_at",
	}).AddRow(
		"user-uuid-1", hashedPassword, "Mogaka Nyabuto", "+254712345678", "seller",
		nil, nil, nil, time.Now(),
	)

	mock.ExpectQuery("SELECT id, password_hash, name, phone, role").
		WithArgs("artisan@kisii.co.ke").
		WillReturnRows(rows)

	req := &userpb.LoginRequest{
		Email:    "artisan@kisii.co.ke",
		Password: "WrongPassword!",
	}

	_, err = server.Login(context.Background(), req)
	if err == nil {
		t.Fatal("expected Unauthenticated error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated code, got %v", st.Code())
	}
}

func TestGetProfile_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewUserServiceServer(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	rows := sqlmock.NewRows([]string{
		"email", "name", "phone", "role", "artisan_bio", "region", "mpesa_number", "created_at",
	}).AddRow(
		"artisan@kisii.co.ke", "Mogaka Nyabuto", "+254712345678", "seller",
		"Soapstone artisan", "Kisii", "+254712345678", time.Now(),
	)

	mock.ExpectQuery("SELECT email, name, phone, role").
		WithArgs("user-uuid-1").
		WillReturnRows(rows)

	resp, err := server.GetProfile(context.Background(), &userpb.GetProfileRequest{UserId: "user-uuid-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetProfile().GetName() != "Mogaka Nyabuto" {
		t.Errorf("expected Mogaka Nyabuto, got %s", resp.GetProfile().GetName())
	}
	if resp.GetProfile().GetRegion() != "Kisii" {
		t.Errorf("expected Kisii, got %s", resp.GetProfile().GetRegion())
	}
}

func TestUpdateSellerProfile_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewUserServiceServer(db, "test-jwt-secret", 15*time.Minute, 24*time.Hour)

	rows := sqlmock.NewRows([]string{
		"email", "name", "phone", "role", "artisan_bio", "region", "mpesa_number", "created_at",
	}).AddRow(
		"artisan@kisii.co.ke", "Mogaka Nyabuto", "+254712345678", "seller",
		"Handmade Kisii soapstone chess sets and bowls", "Kisii", "+254712345678", time.Now(),
	)

	mock.ExpectQuery("UPDATE users").
		WithArgs("Handmade Kisii soapstone chess sets and bowls", "Kisii", "+254712345678", "user-uuid-1").
		WillReturnRows(rows)

	req := &userpb.UpdateSellerProfileRequest{
		UserId:      "user-uuid-1",
		ArtisanBio:  "Handmade Kisii soapstone chess sets and bowls",
		Region:      "Kisii",
		MpesaNumber: "0712345678", // normalizes to +254712345678
	}

	resp, err := server.UpdateSellerProfile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetProfile().GetMpesaNumber() != "+254712345678" {
		t.Errorf("expected normalized M-Pesa number +254712345678, got %s", resp.GetProfile().GetMpesaNumber())
	}
}

func TestPasswordHashing_Roundtrip(t *testing.T) {
	password := "KenyaMamboYote2026!"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword failed: %v", err)
	}

	if !verifyPassword(hash, password) {
		t.Fatal("verifyPassword failed for correct password")
	}

	if verifyPassword(hash, "WrongPassword") {
		t.Fatal("verifyPassword succeeded for incorrect password")
	}
}
