# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tenant .

# Run stage
FROM alpine:3.20

RUN adduser -D -u 10001 app
WORKDIR /app

COPY --from=builder /out/tenant .

USER app
EXPOSE 8080

CMD ["./tenant"]
