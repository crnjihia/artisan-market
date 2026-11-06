import { FastifyInstance } from 'fastify';
import { createProductClient, invokeGrpc } from '../grpc';

export async function productRoutes(app: FastifyInstance) {
  const productClient = createProductClient();

  // POST /products (Protected artisan endpoint)
  app.post(
    '/products',
    {
      preValidation: [app.authenticate],
      schema: {
        body: {
          type: 'object',
          required: ['name', 'price_cents', 'inventory', 'category'],
          properties: {
            name: { type: 'string', minLength: 2 },
            description: { type: 'string' },
            price_cents: { type: 'integer', minimum: 0 },
            inventory: { type: 'integer', minimum: 0 },
            category: { type: 'string' },
            image_url: { type: 'string' },
            origin_region: { type: 'string' },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const sellerId = (request.user as any).sub;
        const body = request.body as any;
        const res = await invokeGrpc(productClient.CreateProduct.bind(productClient), {
          seller_id: sellerId,
          name: body.name,
          description: body.description || '',
          price_cents: body.price_cents,
          inventory: body.inventory,
          category: body.category,
          image_url: body.image_url || '',
          origin_region: body.origin_region || 'Kenya',
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

  // GET /products/:id (Public)
  app.get('/products/:id', async (request, reply) => {
    try {
      const { id } = request.params as any;
      const res = await invokeGrpc(productClient.GetProduct.bind(productClient), {
        product_id: id,
      });
      return reply.code(200).send(res);
    } catch (err: any) {
      return reply.code(mapGrpcErrorToHttp(err)).send({
        statusCode: mapGrpcErrorToHttp(err),
        error: err.details || err.message,
      });
    }
  });

  // GET /products (Public catalogue listing with search & category filters)
  app.get(
    '/products',
    {
      schema: {
        querystring: {
          type: 'object',
          properties: {
            category: { type: 'string' },
            query: { type: 'string' },
            limit: { type: 'integer', minimum: 1, maximum: 100, default: 20 },
            offset: { type: 'integer', minimum: 0, default: 0 },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const query = request.query as any;
        const res = await invokeGrpc(productClient.ListProducts.bind(productClient), {
          category: query.category || '',
          query: query.query || '',
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
    }
  );

  // PUT /products/:id (Protected)
  app.put(
    '/products/:id',
    {
      preValidation: [app.authenticate],
      schema: {
        body: {
          type: 'object',
          required: ['name', 'price_cents', 'inventory', 'category'],
          properties: {
            name: { type: 'string', minLength: 2 },
            description: { type: 'string' },
            price_cents: { type: 'integer', minimum: 0 },
            inventory: { type: 'integer', minimum: 0 },
            category: { type: 'string' },
            image_url: { type: 'string' },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const { id } = request.params as any;
        const body = request.body as any;
        const res = await invokeGrpc(productClient.UpdateProduct.bind(productClient), {
          product_id: id,
          name: body.name,
          description: body.description || '',
          price_cents: body.price_cents,
          inventory: body.inventory,
          category: body.category,
          image_url: body.image_url || '',
        });
        return reply.code(200).send(res);
      } catch (err: any) {
        return reply.code(mapGrpcErrorToHttp(err)).send({
          statusCode: mapGrpcErrorToHttp(err),
          error: err.details || err.message,
        });
      }
    }
  );

  // DELETE /products/:id (Protected)
  app.delete('/products/:id', { preValidation: [app.authenticate] }, async (request, reply) => {
    try {
      const { id } = request.params as any;
      const res = await invokeGrpc(productClient.DeleteProduct.bind(productClient), {
        product_id: id,
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
    case 8: // RESOURCE_EXHAUSTED
      return 409;
    case 16: // UNAUTHENTICATED
      return 401;
    default:
      return 500;
  }
}
