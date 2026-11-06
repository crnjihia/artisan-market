import { buildApp } from '../index';
import { FastifyInstance } from 'fastify';

describe('Artisan Market Gateway API Tests', () => {
  let app: FastifyInstance;

  beforeAll(async () => {
    app = buildApp();
    await app.ready();
  });

  afterAll(async () => {
    await app.close();
  });

  test('GET /api/health returns status ok', async () => {
    const response = await app.inject({
      method: 'GET',
      url: '/api/health',
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.status).toBe('ok');
    expect(body.service).toBe('artisan-gateway');
  });

  test('POST /api/users/register rejects invalid Kenyan phone format', async () => {
    const response = await app.inject({
      method: 'POST',
      url: '/api/users/register',
      payload: {
        email: 'artisan@tabaka.co.ke',
        password: 'ValidPassword123!',
        name: 'Mogaka Nyabuto',
        phone: '+15551234567', // Non-Kenyan number
        role: 'seller',
      },
    });

    expect(response.statusCode).toBe(400);
  });

  test('POST /api/users/register rejects invalid email address', async () => {
    const response = await app.inject({
      method: 'POST',
      url: '/api/users/register',
      payload: {
        email: 'not-an-email',
        password: 'ValidPassword123!',
        name: 'Mogaka Nyabuto',
        phone: '+254712345678',
        role: 'seller',
      },
    });

    expect(response.statusCode).toBe(400);
  });

  test('GET /api/users/profile returns 401 when token is missing', async () => {
    const response = await app.inject({
      method: 'GET',
      url: '/api/users/profile',
    });

    expect(response.statusCode).toBe(401);
    const body = JSON.parse(response.payload);
    expect(body.error).toBe('Unauthorized');
  });

  test('POST /api/products returns 401 when token is missing', async () => {
    const response = await app.inject({
      method: 'POST',
      url: '/api/products',
      payload: {
        name: 'Kisii Soapstone Bowl',
        price_cents: 200000,
        inventory: 5,
        category: 'Carvings',
      },
    });

    expect(response.statusCode).toBe(401);
  });
});

