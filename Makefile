# Variables
MCP_DIR = mcp
API_DIR = api
BINARY_NAME = server-mcp
GO_BINARY = $(MCP_DIR)/$(BINARY_NAME)
API_PORT = 8000

# Default target
.PHONY: all
all: build run

# Build the MCP server
.PHONY: build
build:
	@echo "🔨 Building MCP server..."
	cd $(MCP_DIR) && go build -o ../$(BINARY_NAME) cmd/main.go
	@echo "✅ MCP server built successfully"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(GO_BINARY)
	@echo "✅ Clean completed"

# Install Python dependencies
.PHONY: install-deps
install-deps:
	@echo "📦 Installing Python dependencies..."
	cd $(API_DIR) && pip install -r requirements.txt || pip install fastapi uvicorn agents
	@echo "✅ Dependencies installed"

# Run the API server
.PHONY: run
run: build
	@echo "🚀 Starting API server on port $(API_PORT)..."
	@echo "📡 MCP server binary: $(GO_BINARY)"
	cd $(API_DIR) && uvicorn main:app --port $(API_PORT) --host 0.0.0.0

# Run API in development mode with auto-reload
.PHONY: dev
dev: build
	@echo "🔄 Starting API server in development mode..."
	cd $(API_DIR) && uvicorn main:app --reload --port $(API_PORT) --host 0.0.0.0

# Run only the MCP server (for testing)
.PHONY: run-mcp
run-mcp: build
	@echo "🔧 Running MCP server directly..."
	$(GO_BINARY)

# Check if Go is installed
.PHONY: check-go
check-go:
	@which go > /dev/null || (echo "❌ Go is not installed. Please install Go first." && exit 1)
	@echo "✅ Go is installed: $$(go version)"

# Check if Python is installed
.PHONY: check-python
check-python:
	@which python3 > /dev/null || (echo "❌ Python3 is not installed. Please install Python3 first." && exit 1)
	@echo "✅ Python3 is installed: $$(python3 --version)"

# Setup everything
.PHONY: setup
setup: check-go check-python install-deps build
	@echo "🎉 Setup completed! You can now run 'make run' to start the API"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Build and run the API (default)"
	@echo "  build        - Build the MCP server"
	@echo "  run          - Build and run the API server"
	@echo "  dev          - Run API in development mode with auto-reload"
	@echo "  run-mcp      - Run only the MCP server"
	@echo "  setup        - Check dependencies and build everything"
	@echo "  clean        - Clean build artifacts"
	@echo "  install-deps - Install Python dependencies"
	@echo "  check-go     - Check if Go is installed"
	@echo "  check-python - Check if Python3 is installed"
	@echo "  help         - Show this help message"

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker-compose build
	@echo "✅ Docker image built successfully"

.PHONY: docker-run
docker-run:
	@echo "🚀 Starting Docker container..."
	docker-compose up -d
	@echo "✅ Container started. API available at http://localhost:8000"

.PHONY: docker-stop
docker-stop:
	@echo "🛑 Stopping Docker container..."
	docker-compose down
	@echo "✅ Container stopped"

.PHONY: docker-logs
docker-logs:
	@echo "📋 Showing Docker logs..."
	docker-compose logs -f

.PHONY: docker-clean
docker-clean:
	@echo "🧹 Cleaning Docker resources..."
	docker-compose down -v --rmi all
	docker system prune -f
	@echo "✅ Docker cleanup completed"

.PHONY: docker-dev
docker-dev:
	@echo "🔄 Starting Docker in development mode..."
	docker-compose up --build
