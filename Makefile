.PHONY: help build run test clean lint docker-build docker-run

# Default target
help:
	@echo "Available commands:"
	@echo "  build       - Build the application"
	@echo "  run         - Run the application"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  lint        - Run linter"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"

# Build the application
build:
	@echo "Building application..."
	go build -o bin/server cmd/server/main.go

# Run the application
run:
	@echo "Running application..."
	go run cmd/server/main.go

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t portfolio-zen-backend .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8000:8000 --env-file .env portfolio-zen-backend

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Generate mock files (if using gomock)
mocks:
	@echo "Generating mocks..."
	mockgen -source=internal/services/broker.go -destination=internal/mocks/mock_broker.go
	mockgen -source=internal/database/client.go -destination=internal/mocks/mock_database.go

# Run with hot reload (requires air)
dev:
	@echo "Running with hot reload..."
	air

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	gosec ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...
