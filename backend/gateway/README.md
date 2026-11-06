# Gateway Service

The gateway is a Fastify‑based REST API that forwards client requests to the Go gRPC services.

## Running locally

```sh
# Install dependencies (only needed once)
npm --prefix backend/gateway install

# Development mode (auto‑reload via ts-node)
make run-gateway
```

The service reads configuration from the `.env` file (or `.env.example`). It listens on `$GATEWAY_PORT` (default `3000`). JWT authentication uses the same secret as the User service.
