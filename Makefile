# Makefile for CCTV Backend

.PHONY: help build run test clean docker-build docker-up docker-down docker-logs setup db-init db-reset db-backup dev install-dev-deps deploy monitor

# Load environment variables from .env
include .env
export

# Default target
help:
	@echo "Available commands:"
	@echo "  setup        - Setup the project (install dependencies, create directories)"
	@echo "  build        - Build the Go application"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-up    - Start Docker Compose services"
	@echo "  docker-down  - Stop Docker Compose services"
	@echo "  docker-logs  - View Docker Compose logs"
	@echo "  db-init      - Initialize database with schema"
	@echo "  db-reset     - Reset database"
	@echo "  db-backup    - Backup database"
	@echo "  dev          - Run in development mode with hot reload"
	@echo "  deploy       - Deploy to production"
	@echo "  monitor      - Monitor running services"

# Setup project
setup:
	@echo "Setting up project..."
	go mod download
	mkdir -p uploads/clips
	mkdir -p temp_clips
	mkdir -p logs
	@echo "Setup complete!"

# Build the application
build:
	@echo "Building application..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/cctv-backend .
	@echo "Build complete!"

# Run the application locally
run:
	@echo "Running application..."
	go run .

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf uploads/clips/*
	rm -rf temp_clips/*
	@echo "Cleanup complete!"

# Docker commands
docker-build:
	@echo "Building Docker images..."
	docker compose build

docker-up:
	@echo "Starting Docker Compose services..."
	docker compose up -d
	@echo "Services started!"
	@echo "Backend available at http://localhost:$(SERVER_PORT)"
	@echo "Database available at localhost:$(DB_PORT)"
	@echo "Adminer available at http://localhost:8080"

docker-down:
	@echo "Stopping Docker Compose services..."
	docker compose down

docker-logs:
	@echo "Showing Docker Compose logs..."
	docker compose logs -f

# Initialize database
db-init:
	@echo "Initializing database..."
	docker compose exec cctv_postgres psql -U $(DB_USER) -d $(DB_NAME) -f /docker-entrypoint-initdb.d/01-schema.sql
	@echo "Database initialized!"

# Reset database
db-reset:
	@echo "Resetting database..."
	docker compose exec cctv_postgres psql -U $(DB_USER) -c "DROP DATABASE IF EXISTS $(DB_NAME);"
	docker compose exec cctv_postgres psql -U $(DB_USER) -c "CREATE DATABASE $(DB_NAME);"
	docker compose exec cctv_postgres psql -U $(DB_USER) -d $(DB_NAME) -f /docker-entrypoint-initdb.d/01-schema.sql
	@echo "Database reset complete!"

# Backup database
db-backup:
	@echo "Creating database backup..."
	docker compose exec cctv_postgres pg_dump -U $(DB_USER) $(DB_NAME) > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Database backup created!"

# Development mode with hot reload (requires air)
dev:
	@echo "Starting development mode..."
	@echo "Make sure you have 'air' installed: go install github.com/cosmtrek/air@latest"
	air

# Install development dependencies
install-dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/cosmtrek/air@latest
	@echo "Development dependencies installed!"

# Production deployment
deploy:
	@echo "Deploying to production..."
	docker compose -f docker-compose.prod.yml up -d --build
	@echo "Production deployment complete!"

# Monitor services
monitor:
	@echo "Monitoring services..."
	watch -n 2 'docker compose ps && echo "" && docker stats --no-stream'
