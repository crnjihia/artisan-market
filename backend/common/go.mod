// path: backend/common/go.mod
module github.com/crnjihia/artisan-market/backend/common

go 1.24.0

require (
	github.com/prometheus/client_golang v1.19.1
	github.com/twmb/franz-go v1.17.1
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
	go.opentelemetry.io/otel/sdk v1.28.0
	go.opentelemetry.io/otel/trace v1.28.0
	google.golang.org/grpc v1.65.0
)
