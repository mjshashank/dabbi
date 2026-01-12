.PHONY: all build ui clean install test dev

# Variables
BINARY_NAME=dabbi
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

all: ui build

# Build the UI (React + Vite)
ui:
	@echo "Building UI..."
	cd ui && npm install && npm run build

# Build the Go binary
build: ui
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/dabbi

# Build without rebuilding UI (faster for Go-only changes)
build-go:
	@echo "Building $(BINARY_NAME) (Go only)..."
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/dabbi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf ui/dist
	rm -rf ui/node_modules

# Install to system
install: build
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)..."
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed! Run 'dabbi --help' to get started."

# Run all tests
test: test-backend test-frontend

# Run backend tests
test-backend:
	@echo "Running Go tests..."
	go test -v ./...

# Run backend tests with coverage
test-backend-cover:
	@echo "Running Go tests with coverage..."
	go test -v -race -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run frontend tests
test-frontend:
	@echo "Running UI tests..."
	cd ui && npm run test

# Run frontend tests with coverage
test-frontend-cover:
	@echo "Running UI tests with coverage..."
	cd ui && npm run test:coverage

# Run tests with coverage (legacy alias)
test-cover: test-backend-cover

# Development mode - run UI dev server and Go server separately
dev-ui:
	@echo "Starting UI dev server..."
	cd ui && npm run dev

dev-server:
	@echo "Starting Go server..."
	go run ./cmd/dabbi serve --port 8080

# Lint
lint:
	@echo "Linting..."
	golangci-lint run ./...
	cd ui && npm run lint

# Format code
fmt:
	@echo "Formatting..."
	go fmt ./...
	cd ui && npm run format

# Check if multipass is available
check-multipass:
	@which multipass > /dev/null || (echo "Error: multipass not found. Install from https://multipass.run" && exit 1)
	@echo "multipass is available"

# Help
help:
	@echo "dabbi"
	@echo ""
	@echo "Usage:"
	@echo "  make              - Build UI and Go binary"
	@echo "  make build        - Build UI and Go binary"
	@echo "  make build-go     - Build Go binary only (faster)"
	@echo "  make ui           - Build UI only"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make install      - Install to /usr/local/bin"
	@echo ""
	@echo "Testing:"
	@echo "  make test                - Run all tests (backend + frontend)"
	@echo "  make test-backend        - Run Go tests only"
	@echo "  make test-backend-cover  - Run Go tests with coverage report"
	@echo "  make test-frontend       - Run UI tests only"
	@echo "  make test-frontend-cover - Run UI tests with coverage report"
	@echo ""
	@echo "Development:"
	@echo "  make dev-ui       - Start UI dev server (port 5173)"
	@echo "  make dev-server   - Start Go server (port 8080)"
	@echo "  make lint         - Run linters"
	@echo "  make fmt          - Format code"
	@echo ""
	@echo "Requirements:"
	@echo "  - Go 1.22+"
	@echo "  - Node.js 18+"
	@echo "  - multipass"
