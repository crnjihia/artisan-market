// path: backend/gateway/src/telemetry.ts

import crypto from 'crypto';
import * as grpc from '@grpc/grpc-js';
import { FastifyRequest, FastifyReply } from 'fastify';

export interface TraceContext {
  traceId: string;
  spanId: string;
  traceparent: string;
}

// Parses W3C traceparent header: 00-{traceId}-{spanId}-{flags}
export function parseTraceparent(raw?: string): TraceContext {
  if (raw && /^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$/i.test(raw)) {
    const parts = raw.split('-');
    return {
      traceId: parts[1],
      spanId: parts[2],
      traceparent: raw,
    };
  }

  // Generate new W3C compliant trace ID and span ID
  const traceId = crypto.randomBytes(16).toString('hex');
  const spanId = crypto.randomBytes(8).toString('hex');
  const traceparent = `00-${traceId}-${spanId}-01`;

  return { traceId, spanId, traceparent };
}

// Injects active W3C traceparent into gRPC metadata for downstream Go services
export function injectTraceMetadata(traceCtx?: TraceContext): grpc.Metadata {
  const md = new grpc.Metadata();
  if (traceCtx?.traceparent) {
    md.set('traceparent', traceCtx.traceparent);
  }
  return md;
}

// Fastify plugin / hook ensuring all incoming requests have a trace context
export function attachTraceContext(req: FastifyRequest, reply: FastifyReply) {
  const incoming = (req.headers['traceparent'] as string) || (req.headers['x-trace-id'] as string);
  const traceCtx = parseTraceparent(incoming);

  // Attach to request object for route handlers
  (req as any).traceContext = traceCtx;

  // Echo traceparent header in HTTP response for browser visibility
  reply.header('traceparent', traceCtx.traceparent);
  reply.header('x-trace-id', traceCtx.traceId);
}

