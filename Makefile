# MCP Memory Server Makefile

.PHONY: build run stop logs clean setup help docker-build docker-up docker-down

# Default target
help: ## Show this help message
	@echo "MCP Memory Server - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

# Local builds
build: ## Build the Go binaries locally
	go build -o mcp-memory-server cmd/server/main.go
	go build -o mcp-memory-reporter cmd/reporting/main.go

run: ## Run the MCP server locally (alias for run-server)
	./mcp-memory-server

run-server: ## Run the MCP server locally
	./mcp-memory-server

run-reporter: ## Run the reporting server locally  
	MCP_ENABLE_ENCRYPTION=true MCP_ENCRYPTION_KEY_PATH=$(HOME)/.mcp-memory/encryption.key ./mcp-memory-reporter

# Docker commands
docker-build: ## Build Docker containers
	docker-compose build

docker-up: ## Start all services with Docker Compose
	docker-compose up -d

docker-down: ## Stop all services
	docker-compose down

docker-logs: ## Show logs from all services
	docker-compose logs -f

docker-logs-server: ## Show logs from MCP server only
	docker-compose logs -f mcp-memory-server

docker-logs-reporter: ## Show logs from reporter only
	docker-compose logs -f mcp-memory-reporter

docker-restart: ## Restart all services
	docker-compose restart

docker-ps: ## Show status of all services
	docker-compose ps

# Setup and management
setup: ## Complete setup with Docker
	./scripts/setup.sh

clean: ## Clean up local binaries and Docker resources
	rm -f mcp-memory-server mcp-memory-reporter
	docker-compose down -v
	docker system prune -f

test: ## Run tests
	go test ./...

format: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

# Development helpers
dev-up: ## Start services in development mode
	docker-compose up

dev-shell-server: ## Get shell access to server container
	docker-compose exec mcp-memory-server sh

dev-shell-reporter: ## Get shell access to reporter container
	docker-compose exec mcp-memory-reporter sh

# Production commands  
prod-up: ## Start services in production mode
	docker-compose -f docker-compose.yml up -d

prod-down: ## Stop production services
	docker-compose -f docker-compose.yml down

prod-logs: ## Show production logs
	docker-compose -f docker-compose.yml logs -f