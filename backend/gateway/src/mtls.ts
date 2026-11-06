// path: backend/gateway/src/mtls.ts

import fs from 'fs';
import path from 'path';
import * as grpc from '@grpc/grpc-js';

export function getGrpcClientCredentials(serviceName: string): grpc.ChannelCredentials {
  const enableTls = (process.env.GRPC_ENABLE_TLS || 'false').toLowerCase() === 'true';

  if (!enableTls) {
    return grpc.credentials.createInsecure();
  }

  const certsDir = process.env.CERTS_DIR || path.resolve(__dirname, '../../../certs');
  const caPath = path.join(certsDir, 'ca.crt');
  const clientCertPath = path.join(certsDir, 'gateway.crt');
  const clientKeyPath = path.join(certsDir, 'gateway.key');

  if (fs.existsSync(caPath) && fs.existsSync(clientCertPath) && fs.existsSync(clientKeyPath)) {
    try {
      const rootCert = fs.readFileSync(caPath);
      const clientCert = fs.readFileSync(clientCertPath);
      const clientKey = fs.readFileSync(clientKeyPath);

      console.log(`[mTLS] Initialized mutual TLS credentials for service "${serviceName}"`);
      return grpc.credentials.createSsl(rootCert, clientKey, clientCert);
    } catch (err: any) {
      console.warn(`[mTLS] Error loading certificates from ${certsDir}: ${err.message}. Falling back to insecure.`);
      return grpc.credentials.createInsecure();
    }
  }

  console.warn(`[mTLS] Certificates not found in ${certsDir}. Using insecure credentials.`);
  return grpc.credentials.createInsecure();
}

