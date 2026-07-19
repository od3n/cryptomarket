.PHONY: setup up down logs build run-api run-ingestor migrate seed test test-integration lint fmt vet openapi-validate smoke clean

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

## run-api: Run the API server locally
run-api:
	go run ./cmd/api

## run-ingestor: Run the ingestor locally
run-ingestor:
	go run ./cmd/ingestor

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

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

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
	@curl -sf http://localhost:$(HTTP_PORT)/metrics | grep -q "http_request_duration" || (echo "FAIL: /metrics" && exit 1)
	@echo "All smoke tests passed."

## clean: Remove build artifacts and stop containers
clean:
	rm -rf bin/
	$(DOCKER_COMPOSE) down -v --remove-orphans 2>/dev/null || true
