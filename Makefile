.PHONY: setup up down logs build run-api run-ingestor run-realtime run-frontend migrate seed test test-integration test-frontend test-e2e lint lint-frontend fmt vet openapi-validate smoke smoke-realtime build-frontend demo clean test-resilience test-provider-fallback mock-provider-up mock-provider-down alert-test prometheus-check slo-check incident-demo incident-reset reconcile load-test-resilience test-python

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
	CGO_ENABLED=0 go build -o bin/mockprovider ./cmd/mockprovider

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
	@echo "Grafana at: http://localhost:3001 (admin/admin)"
	@echo "Alertmanager at: http://localhost:9093"

## clean: Remove build artifacts and stop containers
clean:
	rm -rf bin/
	$(DOCKER_COMPOSE) down -v --remove-orphans 2>/dev/null || true

# ─── Phase 3: Resilience Engineering Targets ─────────────────────────────────

## test-resilience: Run resilience-specific unit tests
test-resilience:
	CGO_ENABLED=0 go test -short -count=1 ./internal/resilience/... ./internal/provider/...

## test-provider-fallback: Run provider fallback tests
test-provider-fallback:
	CGO_ENABLED=0 go test -short -count=1 -run "Fallback|Selector|CircuitBreaker" ./internal/...

## test-python: Run Python SRE toolkit tests
test-python:
	cd sre-toolkit && python -m pytest tests/ -v

## mock-provider-up: Start mock provider in success mode
mock-provider-up:
	$(DOCKER_COMPOSE) up -d mock-provider
	@echo "Mock provider running at http://localhost:8082 (mode=success)"

## mock-provider-down: Stop mock provider
mock-provider-down:
	$(DOCKER_COMPOSE) stop mock-provider
	@echo "Mock provider stopped"

## alert-test: Send test alert to Alertmanager
alert-test:
	@echo "Sending test alert to Alertmanager..."
	@curl -sf -X POST http://localhost:9093/api/v2/alerts \
		-H 'Content-Type: application/json' \
		-d '[{"labels":{"alertname":"TestAlert","severity":"warning","service":"test"},"annotations":{"summary":"Test alert from Makefile"}}]' \
		&& echo "Test alert sent successfully" \
		|| echo "Failed to send test alert (is Alertmanager running?)"

## prometheus-check: Validate Prometheus configuration
prometheus-check:
	@echo "Validating Prometheus configuration..."
	@command -v promtool >/dev/null 2>&1 && \
		promtool check config monitoring/prometheus/prometheus.yml && \
		promtool check rules monitoring/prometheus/recording-rules.yml && \
		promtool check rules monitoring/prometheus/alerts.yml \
		|| echo "promtool not installed - skipping validation"

## slo-check: Check current SLO status
slo-check:
	@echo "Checking SLO status..."
	@curl -sf 'http://localhost:9090/api/v1/query?query=api:success_ratio:5m' | \
		python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('data',{}).get('result',[]); print(f'API Success Ratio: {float(r[0][\"value\"][1]):.4f}' if r else 'No data available')" 2>/dev/null \
		|| echo "Prometheus not available"

## incident-demo: Run the incident demonstration script
incident-demo:
	@ALLOW_FAILURE_INJECTION=true ./scripts/incident-demo.sh

## incident-reset: Reset all injected failures
incident-reset:
	@export ALLOW_FAILURE_INJECTION=true; \
	python3 sre-toolkit/inject_failures.py --scenario provider_429 --cleanup 2>/dev/null || true; \
	python3 sre-toolkit/inject_failures.py --scenario provider_500 --cleanup 2>/dev/null || true; \
	python3 sre-toolkit/inject_failures.py --scenario redis_failure --cleanup 2>/dev/null || true; \
	python3 sre-toolkit/inject_failures.py --scenario stale_data --cleanup 2>/dev/null || true; \
	echo "All failures cleaned up"

## reconcile: Run price reconciliation between providers
reconcile:
	cd sre-toolkit && python3 reconcile_prices.py --format text

## load-test-resilience: Run k6 load and resilience tests
load-test-resilience:
	@command -v k6 >/dev/null 2>&1 || { echo "k6 not installed. See: https://k6.io/docs/getting-started/installation/"; exit 1; }
	k6 run load-tests/resilience.js
