# path: backend/user/Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
ENV GOWORK=off
COPY proto/ proto/
COPY backend/user/ backend/user/

WORKDIR /app/backend/user
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/user .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Africa/Nairobi

WORKDIR /app
COPY --from=builder /app/bin/user /app/user

EXPOSE 50051
USER 10001:10001
ENTRYPOINT ["/app/user"]
