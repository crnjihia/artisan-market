// path: backend/common/telemetry/kafka.go
package telemetry

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// KafkaRecordHeaderCarrier adapts franz-go Record headers to OpenTelemetry TextMapCarrier.
type KafkaRecordHeaderCarrier struct {
	record *kgo.Record
}

func NewKafkaRecordHeaderCarrier(record *kgo.Record) *KafkaRecordHeaderCarrier {
	return &KafkaRecordHeaderCarrier{record: record}
}

func (c *KafkaRecordHeaderCarrier) Get(key string) string {
	for _, h := range c.record.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *KafkaRecordHeaderCarrier) Set(key, value string) {
	for i, h := range c.record.Headers {
		if h.Key == key {
			c.record.Headers[i].Value = []byte(value)
			return
		}
	}
	c.record.Headers = append(c.record.Headers, kgo.RecordHeader{
		Key:   key,
		Value: []byte(value),
	})
}

func (c *KafkaRecordHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.record.Headers))
	for _, h := range c.record.Headers {
		keys = append(keys, h.Key)
	}
	return keys
}

// InjectKafkaTrace injects active trace context into a Kafka record before dispatch.
func InjectKafkaTrace(ctx context.Context, record *kgo.Record) {
	carrier := NewKafkaRecordHeaderCarrier(record)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractKafkaTrace extracts trace context from an incoming Kafka record.
func ExtractKafkaTrace(ctx context.Context, record *kgo.Record) context.Context {
	carrier := NewKafkaRecordHeaderCarrier(record)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// StartKafkaConsumerSpan extracts W3C trace context from the record and starts a consumer span.
func StartKafkaConsumerSpan(ctx context.Context, record *kgo.Record, serviceName string) (context.Context, trace.Span) {
	parentCtx := ExtractKafkaTrace(ctx, record)
	tracer := GetTracer(serviceName)

	spanCtx, span := tracer.Start(
		parentCtx,
		"kafka.consume "+record.Topic,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("kafka"),
			semconv.MessagingDestinationNameKey.String(record.Topic),
			attribute.Int("messaging.message_payload_size_bytes", len(record.Value)),
		),
	)

	return spanCtx, span
}

// FinishSpan sets status and closes the span.
func FinishSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}
	span.End()
}
