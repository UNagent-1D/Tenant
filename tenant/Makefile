.PHONY: run build test migrate-up migrate-down docker-up docker-down tidy

APP_NAME=tenant-service
MAIN=./cmd/server/main.go
MIGRATE_BIN=migrate
DB_URL?=$(shell grep DATABASE_URL .env | cut -d '=' -f2-)

# Run the server (loads .env automatically via godotenv)
run:
	go run $(MAIN)

# Build binary
build:
	go build -o bin/$(APP_NAME) $(MAIN)

# Run tests
test:
	go test ./... -v -race -count=1

# Tidy dependencies
tidy:
	go mod tidy

# Apply global migrations
migrate-up:
	$(MIGRATE_BIN) -path migrations/global -database "$(DB_URL)" up

# Rollback global migrations
migrate-down:
	$(MIGRATE_BIN) -path migrations/global -database "$(DB_URL)" down 1

# Start PostgreSQL via Docker Compose
docker-up:
	docker compose up -d

# Stop PostgreSQL
docker-down:
	docker compose down
