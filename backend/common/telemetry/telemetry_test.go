// path: backend/common/telemetry/telemetry_test.go
package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

func TestInitTracer_Offline(t *testing.T) {
	shutdown, err := InitTracer(context.Background(), "artisan-test-service", "disabled")
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tr := GetTracer("test")
	ctx, span := tr.Start(context.Background(), "TestOperation")
	if !span.SpanContext().IsValid() {
		t.Errorf("expected valid span context")
	}
	span.End()
	_ = ctx
}

func TestGRPCMetadataPropagation(t *testing.T) {
	shutdown, err := InitTracer(context.Background(), "artisan-gateway", "disabled")
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tr := GetTracer("artisan-gateway")
	parentCtx, parentSpan := tr.Start(context.Background(), "Gateway.RouteCheckout")
	defer parentSpan.End()

	parentTraceID := parentSpan.SpanContext().TraceID().String()

	// 1. Inject into outgoing gRPC metadata
	outgoingMD := metadata.New(nil)
	otel.GetTextMapPropagator().Inject(parentCtx, MetadataCarrier(outgoingMD))

	// 2. Extract in downstream service (Order Service)
	extractedCtx := otel.GetTextMapPropagator().Extract(context.Background(), MetadataCarrier(outgoingMD))

	orderTracer := GetTracer("artisan-order")
	_, orderSpan := orderTracer.Start(extractedCtx, "OrderService.CreateOrder", trace.WithSpanKind(trace.SpanKindServer))
	defer orderSpan.End()

	orderTraceID := orderSpan.SpanContext().TraceID().String()

	if parentTraceID != orderTraceID {
		t.Errorf("Trace context propagation failed over gRPC! Parent: %s, Child: %s", parentTraceID, orderTraceID)
	}
}

func TestKafkaHeaderPropagation(t *testing.T) {
	shutdown, err := InitTracer(context.Background(), "artisan-order", "disabled")
	if err != nil {
		t.Fatalf("InitTracer failed: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	tr := GetTracer("artisan-order")
	orderCtx, orderSpan := tr.Start(context.Background(), "OrderService.DispatchOutbox")
	defer orderSpan.End()

	expectedTraceID := orderSpan.SpanContext().TraceID().String()

	// 1. Inject into Kafka record headers
	record := &kgo.Record{
		Topic: "order.events",
		Key:   []byte("ord-12345"),
		Value: []byte(`{"order_id":"ord-12345","amount_cents":350000}`),
	}
	InjectKafkaTrace(orderCtx, record)

	hasTraceparent := false
	for _, h := range record.Headers {
		if h.Key == "traceparent" {
			hasTraceparent = true
			break
		}
	}
	if !hasTraceparent {
		t.Fatalf("expected 'traceparent' header in Kafka record")
	}

	// 2. Extract in Kafka consumer (Payment Service)
	consumerCtx, consumerSpan := StartKafkaConsumerSpan(context.Background(), record, "artisan-payment")
	defer consumerSpan.End()

	consumerTraceID := consumerSpan.SpanContext().TraceID().String()
	if consumerTraceID != expectedTraceID {
		t.Errorf("Trace context propagation failed over Kafka! Expected: %s, Got: %s", expectedTraceID, consumerTraceID)
	}
	_ = consumerCtx
}

func TestREDMetrics(t *testing.T) {
	// Should not panic and should register metrics cleanly
	RecordREDMetrics("order-service", "CreateOrder", 25*time.Millisecond, nil)
	RecordREDMetrics("order-service", "CreateOrder", 45*time.Millisecond, errors.New("out of stock"))
	RecordREDMetrics("payment-service", "ProcessMpesaCallback", 120*time.Millisecond, nil)
}

