# path: backend/product/Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.work ./
COPY proto/ proto/
COPY backend/product/ backend/product/

WORKDIR /app/backend/product
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/product .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Africa/Nairobi

WORKDIR /app
COPY --from=builder /app/bin/product /app/product

EXPOSE 50052
USER 10001:10001
ENTRYPOINT ["/app/product"]
