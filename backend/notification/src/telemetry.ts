// path: backend/notification/src/telemetry.ts

import crypto from 'crypto';

export interface TraceContext {
  traceId: string;
  spanId: string;
  traceparent: string;
}

export function extractKafkaTraceparent(headers?: Record<string, any>): TraceContext {
  let raw: string | undefined;

  if (headers) {
    if (headers['traceparent']) {
      raw = headers['traceparent'].toString();
    } else if (headers['Traceparent']) {
      raw = headers['Traceparent'].toString();
    }
  }

  if (raw && /^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$/i.test(raw)) {
    const parts = raw.split('-');
    return {
      traceId: parts[1],
      spanId: parts[2],
      traceparent: raw,
    };
  }

  const traceId = crypto.randomBytes(16).toString('hex');
  const spanId = crypto.randomBytes(8).toString('hex');
  return {
    traceId,
    spanId,
    traceparent: `00-${traceId}-${spanId}-01`,
  };
}

