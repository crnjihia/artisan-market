module github.com/yourorg/artisan-market/backend/user

go 1.24.0

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/yourorg/artisan-market/proto v0.0.0
	golang.org/x/crypto v0.24.0
	google.golang.org/grpc v1.65.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/yourorg/artisan-market/proto => ../../proto
