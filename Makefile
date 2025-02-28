.PHONY: build run test lint migrate-up migrate-down migrate-version clean

# Go related variables
BINARY_NAME=document-api
GO=$(shell which go)

# Docker related variables
DOCKER_COMPOSE=$(shell which -v docker-compose)

# Build the application
build:
	$(GO) build -o ./bin/$(BINARY_NAME) ./cmd/api

# Run the application
run: build
	./bin/$(BINARY_NAME)

# Run all tests
test:
	$(GO) test ./...

# Run linter
lint:
	$(shell which golangci-lint) run

# Run database migrations up
migrate-up:
	$(GO) run scripts/migrate.go -up

# Run database migrations down
migrate-down:
	$(GO) run scripts/migrate.go -down

# Show migration version
migrate-version:
	$(GO) run scripts/migrate.go -version

# Start docker services (PostgreSQL and Redis)
docker-up:
	$(DOCKER_COMPOSE) up -d

# Stop docker services
docker-down:
	$(DOCKER_COMPOSE) down

# Clean build files
clean:
	rm -rf ./bin

# Install dependencies
deps:
	$(GO) mod download

# Initialize development environment
init: deps docker-up migrate-up