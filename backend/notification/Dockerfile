# path: backend/notification/Dockerfile
FROM node:20-alpine AS builder

WORKDIR /app
COPY package*.json ./
RUN npm install
COPY tsconfig.json ./
COPY src/ src/
RUN npm run build

FROM node:20-alpine
ENV NODE_ENV=production
ENV TZ=Africa/Nairobi
RUN apk --no-cache add tzdata

WORKDIR /app
COPY package*.json ./
RUN npm install --omit=dev
COPY --from=builder /app/dist ./dist

EXPOSE 3001
USER node
CMD ["node", "dist/index.js"]

