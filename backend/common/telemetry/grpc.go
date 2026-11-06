// path: backend/common/telemetry/grpc.go
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MetadataCarrier adapts gRPC metadata.MD to OpenTelemetry TextMapCarrier.
type MetadataCarrier metadata.MD

func (m MetadataCarrier) Get(key string) string {
	values := metadata.MD(m).Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (m MetadataCarrier) Set(key, value string) {
	metadata.MD(m).Set(key, value)
}

func (m MetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// UnaryServerInterceptor extracts W3C trace context from incoming gRPC metadata and traces handler execution.
func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	tracer := GetTracer(serviceName)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		propagator := otel.GetTextMapPropagator()
		extractedCtx := propagator.Extract(ctx, MetadataCarrier(md))

		ctx, span := tracer.Start(
			extractedCtx,
			info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String(serviceName),
				semconv.RPCMethodKey.String(info.FullMethod),
			),
		)
		defer span.End()

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "OK")
		}

		return resp, err
	}
}

// UnaryClientInterceptor traces outgoing gRPC requests and injects W3C trace context into outgoing metadata.
func UnaryClientInterceptor(clientServiceName string) grpc.UnaryClientInterceptor {
	tracer := GetTracer(clientServiceName)

	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, span := tracer.Start(
			ctx,
			method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCMethodKey.String(method),
			),
		)
		defer span.End()

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		otel.GetTextMapPropagator().Inject(ctx, MetadataCarrier(md))
		ctx = metadata.NewOutgoingContext(ctx, md)

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "OK")
		}

		return err
	}
}
