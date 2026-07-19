.PHONY: setup up down logs build run-api run-ingestor run-realtime run-frontend migrate seed test test-integration test-frontend test-e2e lint lint-frontend fmt vet openapi-validate smoke smoke-realtime build-frontend demo clean

# Variables
APP_ENV ?= development
HTTP_PORT ?= 8080
DOCKER_COMPOSE := docker compose

## setup: Prepare the local project (install tools, download deps)
setup:
	go mod download
	go mod tidy
	@echo "Setup complete. Copy .env.example to .env and adjust values."

## up: Start the full stack via Docker Compose
up:
	$(DOCKER_COMPOSE) up --build -d

## down: Stop and remove all containers
down:
	$(DOCKER_COMPOSE) down

## logs: Tail logs from all services
logs:
	$(DOCKER_COMPOSE) logs -f

## build: Build all Go binaries
build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api
	CGO_ENABLED=0 go build -o bin/ingestor ./cmd/ingestor
	CGO_ENABLED=0 go build -o bin/realtime ./cmd/realtime

## run-api: Run the API server locally
run-api:
	go run ./cmd/api

## run-ingestor: Run the ingestor locally
run-ingestor:
	go run ./cmd/ingestor

## run-realtime: Run the realtime gateway locally
run-realtime:
	HTTP_PORT=8081 go run ./cmd/realtime

## run-frontend: Run the frontend dev server
run-frontend:
	cd frontend && npm run dev

## migrate: Run database migrations
migrate:
	go run ./cmd/api migrate

## seed: Seed initial coin data
seed:
	go run ./cmd/api seed

## test: Run unit tests
test:
	go test -short -race -count=1 ./...

## test-integration: Run integration tests (requires Docker)
test-integration:
	go test -run Integration -race -count=1 ./...

## test-frontend: Run frontend unit tests
test-frontend:
	cd frontend && npm test

## test-e2e: Run end-to-end tests (requires running stack)
test-e2e:
	cd frontend && npx playwright test

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## lint-frontend: Run frontend linting
lint-frontend:
	cd frontend && npm run lint && npm run typecheck

## fmt: Format all Go code
fmt:
	gofmt -s -w .

## vet: Run go vet
vet:
	go vet ./...

## openapi-validate: Validate the OpenAPI specification
openapi-validate:
	@command -v npx >/dev/null 2>&1 || { echo "npx required for OpenAPI validation"; exit 1; }
	npx --yes @redocly/cli lint api/openapi.yaml

## smoke: Run smoke tests against a running stack
smoke:
	@echo "Running smoke tests..."
	@curl -sf http://localhost:$(HTTP_PORT)/health || (echo "FAIL: /health" && exit 1)
	@curl -sf http://localhost:$(HTTP_PORT)/ready || (echo "FAIL: /ready" && exit 1)
	@curl -sf http://localhost:$(HTTP_PORT)/coins || (echo "FAIL: /coins" && exit 1)
	@curl -sf http://localhost:$(HTTP_PORT)/markets || (echo "FAIL: /markets" && exit 1)
	@curl -sf http://localhost:$(HTTP_PORT)/providers/status || (echo "FAIL: /providers/status" && exit 1)
	@curl -sf http://localhost:$(HTTP_PORT)/metrics | grep -q "http_request_duration" || (echo "FAIL: /metrics" && exit 1)
	@echo "All smoke tests passed."

## smoke-realtime: Run smoke tests for the realtime gateway
smoke-realtime:
	@echo "Running realtime smoke tests..."
	@curl -sf http://localhost:8081/health || (echo "FAIL: realtime /health" && exit 1)
	@curl -sf http://localhost:8081/ready || (echo "FAIL: realtime /ready" && exit 1)
	@curl -sf http://localhost:8081/metrics | grep -q "realtime_active_connections" || (echo "FAIL: realtime /metrics" && exit 1)
	@echo "Realtime smoke tests passed."

## build-frontend: Build the frontend for production
build-frontend:
	cd frontend && npm run build

## demo: Start the full stack and open the dashboard
demo: up
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Dashboard available at: http://localhost:3000"
	@echo "API available at: http://localhost:8080"
	@echo "Realtime gateway at: http://localhost:8081"
	@echo "Prometheus at: http://localhost:9090"

## clean: Remove build artifacts and stop containers
clean:
	rm -rf bin/
	$(DOCKER_COMPOSE) down -v --remove-orphans 2>/dev/null || true
