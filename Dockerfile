# Multi-stage build for Go MCP server and Python FastAPI app
FROM golang:1.24-alpine AS go-builder

# Install git for Go modules
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy Go module files
COPY mcp/go.mod mcp/go.sum ./

# Download dependencies
RUN go mod download

# Copy Go source code
COPY mcp/ ./

# Build the MCP server binary
RUN go build -o server-mcp cmd/main.go

# Python stage
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy Python requirements
COPY api/requirements.txt ./

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the built Go binary from the previous stage
COPY --from=go-builder /app/server-mcp ./

# Copy Python application
COPY api/ ./

# Create a non-root user
RUN useradd -m -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

# Expose the port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8000/docs || exit 1

# Run the FastAPI application
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
