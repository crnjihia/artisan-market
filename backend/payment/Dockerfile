# path: backend/payment/Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.work ./
COPY proto/ proto/
COPY backend/payment/ backend/payment/

WORKDIR /app/backend/payment
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/payment .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Africa/Nairobi

WORKDIR /app
COPY --from=builder /app/bin/payment /app/payment

EXPOSE 50054 8084
USER 10001:10001
ENTRYPOINT ["/app/payment"]

