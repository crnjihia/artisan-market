// path: backend/notification/src/index.ts

import http from 'http';
import dotenv from 'dotenv';
import { NotificationConsumer } from './kafkaConsumer';

dotenv.config();
dotenv.config({ path: '../../.env' });

const HTTP_PORT = parseInt(process.env.NOTIFICATION_HTTP_PORT || '3001', 10);
const consumer = new NotificationConsumer();

const server = http.createServer((req, res) => {
  if (req.url === '/healthz' || req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok', service: 'notification' }));
    return;
  }

  res.writeHead(404, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'Not Found' }));
});

server.listen(HTTP_PORT, () => {
  console.log(`[Notification Service] Health check HTTP server listening on :${HTTP_PORT}`);
});

consumer.start().catch((err) => {
  console.error('[Notification Service] Failed to start Kafka consumer:', err.message);
});

const shutdown = async () => {
  console.log('[Notification Service] Shutting down gracefully...');
  server.close();
  try {
    await consumer.stop();
  } catch (err: any) {
    console.error('[Notification Service] Error stopping consumer:', err.message);
  }
  process.exit(0);
};

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
