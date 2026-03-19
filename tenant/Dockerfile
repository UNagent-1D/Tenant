# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/tenant-service ./cmd/server/main.go

# Run stage
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/bin/tenant-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./tenant-service"]
