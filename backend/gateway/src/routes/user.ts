import { FastifyInstance } from 'fastify';
import { createUserClient, invokeGrpc } from '../grpc';

export async function userRoutes(app: FastifyInstance) {
  const userClient = createUserClient();

  // POST /users/register
  app.post(
    '/users/register',
    {
      schema: {
        body: {
          type: 'object',
          required: ['email', 'password', 'name', 'phone'],
          properties: {
            email: { type: 'string', format: 'email' },
            password: { type: 'string', minLength: 8 },
            name: { type: 'string', minLength: 1 },
            phone: { type: 'string', pattern: '^(\\+254|0)[17]\\d{8}$' },
            role: { type: 'string', enum: ['buyer', 'seller'] },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const body = request.body as any;
        const res = await invokeGrpc(userClient.Register.bind(userClient), {
          email: body.email,
          password: body.password,
          name: body.name,
          phone: body.phone,
          role: body.role || 'buyer',
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

  // POST /users/login
  app.post(
    '/users/login',
    {
      schema: {
        body: {
          type: 'object',
          required: ['email', 'password'],
          properties: {
            email: { type: 'string', format: 'email' },
            password: { type: 'string', minLength: 1 },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const body = request.body as any;
        const res = await invokeGrpc(userClient.Login.bind(userClient), {
          email: body.email,
          password: body.password,
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

  // GET /users/profile (Protected)
  app.get(
    '/users/profile',
    { preValidation: [app.authenticate] },
    async (request, reply) => {
      try {
        const userId = (request.user as any).sub;
        const res = await invokeGrpc(userClient.GetProfile.bind(userClient), {
          user_id: userId,
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

  // PUT /users/seller-profile (Protected)
  app.put(
    '/users/seller-profile',
    {
      preValidation: [app.authenticate],
      schema: {
        body: {
          type: 'object',
          required: ['artisan_bio', 'region'],
          properties: {
            artisan_bio: { type: 'string', minLength: 5 },
            region: { type: 'string', minLength: 2 },
            mpesa_number: { type: 'string', pattern: '^(\\+254|0)[17]\\d{8}$' },
          },
        },
      },
    },
    async (request, reply) => {
      try {
        const userId = (request.user as any).sub;
        const body = request.body as any;
        const res = await invokeGrpc(userClient.UpdateSellerProfile.bind(userClient), {
          user_id: userId,
          artisan_bio: body.artisan_bio,
          region: body.region,
          mpesa_number: body.mpesa_number,
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
}

function mapGrpcErrorToHttp(err: any): number {
  switch (err.code) {
    case 3: // INVALID_ARGUMENT
      return 400;
    case 5: // NOT_FOUND
      return 404;
    case 6: // ALREADY_EXISTS
      return 409;
    case 7: // PERMISSION_DENIED
      return 403;
    case 16: // UNAUTHENTICATED
      return 401;
    default:
      return 500;
  }
}
