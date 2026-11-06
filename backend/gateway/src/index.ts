import Fastify, { FastifyInstance } from 'fastify';
import fastifyJwt from '@fastify/jwt';
import fastifyRateLimit from '@fastify/rate-limit';
import fastifyCors from '@fastify/cors';
import dotenv from 'dotenv';
import { userRoutes } from './routes/user';
import { productRoutes } from './routes/product';
import { checkoutRoutes } from './routes/checkout';

declare module 'fastify' {
  interface FastifyInstance {
    authenticate: (request: any, reply: any) => Promise<void>;
  }
}

dotenv.config();
dotenv.config({ path: '../../.env' });

const PORT = process.env.GATEWAY_PORT ? Number(process.env.GATEWAY_PORT) : 4000;
const JWT_SECRET = process.env.JWT_SECRET || 'super-secret-artisan-jwt-key-change-in-production';

export function buildApp(): FastifyInstance {
  const app = Fastify({
    logger: process.env.NODE_ENV === 'test' ? false : { level: 'info' },
  });

  // CORS support for Next.js storefront
  app.register(fastifyCors, {
    origin: true,
    credentials: true,
  });

  // JWT Authentication
  app.register(fastifyJwt, {
    secret: JWT_SECRET,
  });

  // Rate Limiting (100 req/minute per IP)
  app.register(fastifyRateLimit, {
    max: 100,
    timeWindow: '1 minute',
  });

  // Authentication Hook Decorator
  app.decorate('authenticate', async (request: any, reply: any) => {
    try {
      await request.jwtVerify();
    } catch (err: any) {
      reply.code(401).send({
        statusCode: 401,
        error: 'Unauthorized',
        message: 'Missing or invalid authentication token',
      });
    }
  });

  // Health Check
  app.get('/api/health', async () => {
    return {
      status: 'ok',
      service: 'artisan-gateway',
      timestamp: new Date().toISOString(),
    };
  });

  // REST API Routes
  app.register(userRoutes, { prefix: '/api' });
  app.register(productRoutes, { prefix: '/api' });
  app.register(checkoutRoutes, { prefix: '/api' });

  return app;
}

// Start standalone server if run directly
if (require.main === module) {
  const app = buildApp();
  app.listen({ port: PORT, host: '0.0.0.0' }, (err, address) => {
    if (err) {
      console.error('Gateway startup error:', err);
      process.exit(1);
    }
    console.log(`[Artisan Gateway] Running on ${address}`);
  });
}
