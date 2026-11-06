import { FastifyInstance } from 'fastify';
import { createOrderClient, createPaymentClient, invokeGrpc } from '../grpc';

export async function checkoutRoutes(app: FastifyInstance) {
  const orderClient = createOrderClient();
  const paymentClient = createPaymentClient();

  // POST /orders (Buyer checkout submission)
  app.post(
    '/orders',
    {
      schema: {
        body: {
          type: 'object',
          required: ['items', 'payment_method', 'customer_phone'],
          properties: {
            user_id: { type: 'string' },
            items: {
              type: 'array',
              minItems: 1,
              items: {
                type: 'object',
                required: ['product_id', 'quantity'],
                properties: {
                  product_id: { type: 'string' },
                  quantity: { type: 'integer', minimum: 1 },
                },
              },
            },
            payment_method: { type: 'string', enum: ['mpesa', 'stripe'] },
            customer_phone: { type: 'string', pattern: '^(\\+254|0)[17]\\d{8}$' },
            customer_email: { type: 'string', format: 'email' },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const body = request.body as any;
        let userId = body.user_id;
        try {
          // Optional JWT enrichment if logged in
          await request.jwtVerify();
          userId = (request.user as any).sub;
        } catch {
          // Allow guest checkout if not signed in
          if (!userId) {
            userId = 'guest-' + Math.random().toString(36).substring(2, 10);
          }
        }

        const res = await invokeGrpc(orderClient.CreateOrder.bind(orderClient), {
          user_id: userId,
          items: body.items,
          payment_method: body.payment_method,
          customer_phone: body.customer_phone,
          customer_email: body.customer_email || '',
        });

        return reply.code(201).send(res);
      } catch (err: any) {
        return reply.code(mapGrpcErrorToHttp(err)).send({
          statusCode: mapGrpcErrorToHttp(err),
          error: err.details || err.message,
        });
      }
    }
  );

  // GET /orders/:id
  app.get('/orders/:id', async (request, reply) => {
    try {
      const { id } = request.params as any;
      const res = await invokeGrpc(orderClient.GetOrder.bind(orderClient), {
        order_id: id,
      });
      return reply.code(200).send(res);
    } catch (err: any) {
      return reply.code(mapGrpcErrorToHttp(err)).send({
        statusCode: mapGrpcErrorToHttp(err),
        error: err.details || err.message,
      });
    }
  });

  // GET /orders (List orders for buyer)
  app.get('/orders', async (request, reply) => {
    try {
      const query = request.query as any;
      const res = await invokeGrpc(orderClient.ListOrders.bind(orderClient), {
        user_id: query.user_id || '',
        limit: query.limit || 20,
        offset: query.offset || 0,
      });
      return reply.code(200).send(res);
    } catch (err: any) {
      return reply.code(mapGrpcErrorToHttp(err)).send({
        statusCode: mapGrpcErrorToHttp(err),
        error: err.details || err.message,
      });
    }
  });

  // POST /orders/:id/cancel
  app.post('/orders/:id/cancel', async (request, reply) => {
    try {
      const { id } = request.params as any;
      const body = (request.body as any) || {};
      const res = await invokeGrpc(orderClient.CancelOrder.bind(orderClient), {
        order_id: id,
        reason: body.reason || 'buyer_requested',
      });
      return reply.code(200).send(res);
    } catch (err: any) {
      return reply.code(mapGrpcErrorToHttp(err)).send({
        statusCode: mapGrpcErrorToHttp(err),
        error: err.details || err.message,
      });
    }
  });

  // POST /payments (Initiate M-Pesa STK push or Stripe intent)
  app.post(
    '/payments',
    {
      schema: {
        body: {
          type: 'object',
          required: ['order_id', 'amount_cents', 'method'],
          properties: {
            order_id: { type: 'string' },
            amount_cents: { type: 'integer', minimum: 1 },
            currency: { type: 'string', default: 'KES' },
            method: { type: 'string', enum: ['mpesa', 'stripe'] },
            customer_phone: { type: 'string', pattern: '^(\\+254|0)[17]\\d{8}$' },
            customer_email: { type: 'string' },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const body = request.body as any;
        const res = await invokeGrpc(paymentClient.CreatePayment.bind(paymentClient), {
          order_id: body.order_id,
          amount_cents: body.amount_cents,
          currency: body.currency || 'KES',
          method: body.method,
          customer_phone: body.customer_phone || '',
          customer_email: body.customer_email || '',
        });
        return reply.code(201).send(res);
      } catch (err: any) {
        return reply.code(mapGrpcErrorToHttp(err)).send({
          statusCode: mapGrpcErrorToHttp(err),
          error: err.details || err.message,
        });
      }
    }
  );

  // GET /payments/:id (Polling endpoint for checkout modal)
  app.get('/payments/:id', async (request, reply) => {
    try {
      const { id } = request.params as any;
      const res = await invokeGrpc(paymentClient.GetPayment.bind(paymentClient), {
        payment_id: id,
      });
      return reply.code(200).send(res);
    } catch (err: any) {
      return reply.code(mapGrpcErrorToHttp(err)).send({
        statusCode: mapGrpcErrorToHttp(err),
        error: err.details || err.message,
      });
    }
  });
}

function mapGrpcErrorToHttp(err: any): number {
  switch (err.code) {
    case 3: // INVALID_ARGUMENT
      return 400;
    case 5: // NOT_FOUND
      return 404;
    case 9: // FAILED_PRECONDITION
      return 412;
    case 8: // RESOURCE_EXHAUSTED
      return 409;
    case 16: // UNAUTHENTICATED
      return 401;
    default:
      return 500;
  }
}
