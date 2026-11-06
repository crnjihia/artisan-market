# path: backend/gateway/Dockerfile
FROM node:20-alpine AS builder

WORKDIR /app
COPY proto/ proto/
COPY backend/gateway/package*.json backend/gateway/
COPY backend/gateway/tsconfig.json backend/gateway/
COPY backend/gateway/src/ backend/gateway/src/

WORKDIR /app/backend/gateway
RUN npm install
RUN npm run build

FROM node:20-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV TZ=Africa/Nairobi

COPY proto/ /app/proto/
COPY backend/gateway/package*.json /app/
RUN npm install --only=production

COPY --from=builder /app/backend/gateway/dist /app/dist

EXPOSE 3000
USER node
CMD ["node", "dist/index.js"]

