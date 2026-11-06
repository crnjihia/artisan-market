# path: backend/order/Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.work ./
COPY proto/ proto/
COPY backend/order/ backend/order/

WORKDIR /app/backend/order
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/order .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Africa/Nairobi

WORKDIR /app
COPY --from=builder /app/bin/order /app/order

EXPOSE 50053
USER 10001:10001
ENTRYPOINT ["/app/order"]

