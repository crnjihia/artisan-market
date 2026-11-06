package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	userpb "github.com/yourorg/artisan-market/proto/user/v1"
	"golang.org/x/crypto/argon2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Kenyan mobile phone format: Safaricom & Airtel Kenya prefixes (+2547..., +2541..., 07..., 01...)
var kenyanPhoneRegex = regexp.MustCompile(`^(?:\+254|0)?([17]\d{8})$`)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

// UserServiceServer implements userpb.UserServiceServer.
type UserServiceServer struct {
	userpb.UnimplementedUserServiceServer
	db         *sql.DB
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewUserServiceServer constructs a configured UserServiceServer instance.
func NewUserServiceServer(db *sql.DB, jwtSecret string, accessTTL, refreshTTL time.Duration) *UserServiceServer {
	return &UserServiceServer{
		db:         db,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// UserClaims defines custom JWT claims for authorization across services.
type UserClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// Register validates buyer or artisan credentials, hashes passwords with Argon2id,
// records the user in PostgreSQL, and generates an access token + rotating refresh token.
func (s *UserServiceServer) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.GetEmail()))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid email address format")
	}

	if len(req.GetPassword()) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	phone, err := normalizeKenyanPhone(req.GetPhone())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid Kenyan phone number: %v", err)
	}

	role := strings.ToLower(strings.TrimSpace(req.GetRole()))
	if role == "" {
		role = "buyer"
	}
	if role != "buyer" && role != "seller" {
		return nil, status.Error(codes.InvalidArgument, "role must be either 'buyer' or 'seller'")
	}

	passwordHash, err := hashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
	}

	var userID string
	var createdAt time.Time
	query := `
		INSERT INTO users (email, password_hash, name, phone, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err = s.db.QueryRowContext(ctx, query, email, passwordHash, name, phone, role).Scan(&userID, &createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, status.Error(codes.AlreadyExists, "an account with this email already exists")
		}
		return nil, status.Errorf(codes.Internal, "database error inserting user: %v", err)
	}

	accessToken, err := s.generateAccessToken(userID, email, name, role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}

	refreshToken, err := s.generateAndStoreRefreshToken(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to issue refresh token: %v", err)
	}

	profile := &userpb.UserProfile{
		Id:              userID,
		Email:           email,
		Name:            name,
		Phone:           phone,
		Role:            role,
		CreatedAtUnixMs: createdAt.UnixMilli(),
	}

	return &userpb.RegisterResponse{
		UserId:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Profile:      profile,
	}, nil
}

// Login validates user credentials against stored Argon2id hashes and issues fresh tokens.
func (s *UserServiceServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.GetEmail()))
	if email == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	var (
		userID       string
		storedHash   string
		name         string
		phone        string
		role         string
		artisanBio   sql.NullString
		region       sql.NullString
		mpesaNumber  sql.NullString
		createdAt    time.Time
	)

	query := `
		SELECT id, password_hash, name, phone, role, artisan_bio, region, mpesa_number, created_at
		FROM users
		WHERE email = $1
	`
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&userID, &storedHash, &name, &phone, &role, &artisanBio, &region, &mpesaNumber, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	if !verifyPassword(storedHash, req.GetPassword()) {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	accessToken, err := s.generateAccessToken(userID, email, name, role)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}

	refreshToken, err := s.generateAndStoreRefreshToken(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to issue refresh token: %v", err)
	}

	profile := &userpb.UserProfile{
		Id:              userID,
		Email:           email,
		Name:            name,
		Phone:           phone,
		Role:            role,
		ArtisanBio:      artisanBio.String,
		Region:          region.String,
		MpesaNumber:     mpesaNumber.String,
		CreatedAtUnixMs: createdAt.UnixMilli(),
	}

	return &userpb.LoginResponse{
		UserId:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Profile:      profile,
	}, nil
}

// GetProfile retrieves user profile details.
func (s *UserServiceServer) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var (
		email       string
		name        string
		phone       string
		role        string
		artisanBio  sql.NullString
		region      sql.NullString
		mpesaNumber sql.NullString
		createdAt   time.Time
	)

	query := `
		SELECT email, name, phone, role, artisan_bio, region, mpesa_number, created_at
		FROM users
		WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, req.GetUserId()).Scan(
		&email, &name, &phone, &role, &artisanBio, &region, &mpesaNumber, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user profile not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &userpb.GetProfileResponse{
		Profile: &userpb.UserProfile{
			Id:              req.GetUserId(),
			Email:           email,
			Name:            name,
			Phone:           phone,
			Role:            role,
			ArtisanBio:      artisanBio.String,
			Region:          region.String,
			MpesaNumber:     mpesaNumber.String,
			CreatedAtUnixMs: createdAt.UnixMilli(),
		},
	}, nil
}

// UpdateSellerProfile updates artisan seller details such as their bio, craft region, and M-Pesa number.
func (s *UserServiceServer) UpdateSellerProfile(ctx context.Context, req *userpb.UpdateSellerProfileRequest) (*userpb.UpdateSellerProfileResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	mpesaNumber := strings.TrimSpace(req.GetMpesaNumber())
	if mpesaNumber != "" {
		normalized, err := normalizeKenyanPhone(mpesaNumber)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid M-Pesa phone number: %v", err)
		}
		mpesaNumber = normalized
	}

	var (
		email       string
		name        string
		phone       string
		role        string
		artisanBio  sql.NullString
		region      sql.NullString
		mpesaOut    sql.NullString
		createdAt   time.Time
	)

	query := `
		UPDATE users
		SET artisan_bio = $1, region = $2, mpesa_number = $3, role = 'seller', updated_at = now()
		WHERE id = $4
		RETURNING email, name, phone, role, artisan_bio, region, mpesa_number, created_at
	`
	err := s.db.QueryRowContext(ctx, query, req.GetArtisanBio(), req.GetRegion(), mpesaNumber, req.GetUserId()).Scan(
		&email, &name, &phone, &role, &artisanBio, &region, &mpesaOut, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	return &userpb.UpdateSellerProfileResponse{
		Profile: &userpb.UserProfile{
			Id:              req.GetUserId(),
			Email:           email,
			Name:            name,
			Phone:           phone,
			Role:            role,
			ArtisanBio:      artisanBio.String,
			Region:          region.String,
			MpesaNumber:     mpesaOut.String,
			CreatedAtUnixMs: createdAt.UnixMilli(),
		},
	}, nil
}

// -----------------------------------------------------------------------------
// JWT Token Helpers
// -----------------------------------------------------------------------------

func (s *UserServiceServer) generateAccessToken(userID, email, name, role string) (string, error) {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "artisan-market-user-service",
		},
		Email: email,
		Name:  name,
		Role:  role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *UserServiceServer) generateAndStoreRefreshToken(ctx context.Context, userID string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Hash token before storing to prevent plaintext leakage in DB backups
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])
	expiresAt := time.Now().Add(s.refreshTTL)

	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := s.db.ExecContext(ctx, query, userID, tokenHash, expiresAt)
	if err != nil {
		return "", err
	}

	return rawToken, nil
}

// -----------------------------------------------------------------------------
// Security & Validation Utilities
// -----------------------------------------------------------------------------

func normalizeKenyanPhone(input string) (string, error) {
	cleaned := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(input), " ", ""), "-", "")
	matches := kenyanPhoneRegex.FindStringSubmatch(cleaned)
	if len(matches) != 2 {
		return "", fmt.Errorf("phone must be a valid Safaricom/Airtel number (e.g. +254712345678 or 0712345678)")
	}
	return "+254" + matches[1], nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Stored format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads, b64Salt, b64Hash), nil
}

func verifyPassword(encodedHash, password string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != saltLen {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expectedHash) != argonKeyLen {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return subtle.ConstantTimeCompare(expectedHash, actualHash) == 1
}
