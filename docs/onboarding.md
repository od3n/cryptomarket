# Developer Onboarding Guide

> **Goal**: Get a new engineer from zero to productive in under 30 minutes.

## Prerequisites

Install these before starting:

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.21+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20+ | [nodejs.org](https://nodejs.org/) |
| Docker + Compose | Latest | [docker.com](https://www.docker.com/products/docker-desktop/) |
| Make | Any | Pre-installed on macOS/Linux |
| golangci-lint | v1.56+ | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56` |

## Step-by-Step Onboarding

### 1. Clone and Setup (2 min)

```bash
git clone <repo-url> && cd cryptomarket
make setup
cp .env.example .env
cp frontend/.env.example frontend/.env.local
```

### 2. Start the Stack (1 min)

```bash
make up
```

Wait ~30 seconds for all services to become healthy.

### 3. Verify (1 min)

```bash
make smoke
make smoke-realtime
```

All tests should pass. Open http://localhost:3002 to see the dashboard.

### 4. Explore the Architecture (10 min)

Read these in order:
1. [Architecture Overview](architecture/overview.md) — system components and data flow
2. [ADR-001: Go plus Python](adr/001-go-plus-python.md) — language choices
3. [ADR-002: PostgreSQL and Redis Streams](adr/002-postgresql-redis-streams.md) — data layer
4. [ADR-003: SSE over WebSockets](adr/003-sse-over-websockets.md) — realtime delivery
5. [ADR-009: Circuit Breaker Pattern](adr/009-circuit-breaker.md) — resilience

### 5. Make Your First Change (10 min)

Try this exercise:
1. Create a branch: `git checkout -b feat/add-health-detail`
2. Open `internal/api/handlers.go`
3. Find the `Health` handler
4. Add a `"version": "1.0.0"` field to the health response
5. Run `make test` — fix any failures
6. Run `make lint` — fix any warnings
7. Commit: `git commit -am "feat(api): add version to health endpoint"`
8. Push and open a PR

### 6. Understand the CI Pipeline (5 min)

Review `.github/workflows/ci.yml`:
- **lint**: gofmt, go vet, golangci-lint
- **test**: unit tests with race detector + coverage
- **build**: compile all binaries
- **frontend**: typecheck, lint, test, build
- **docker**: build all images
- **security**: govulncheck

## Key Commands Reference

| Command | Purpose |
|---------|---------|
| `make up` | Start full stack |
| `make down` | Stop all services |
| `make test` | Run Go unit tests |
| `make lint` | Run Go linter |
| `make fmt` | Format Go code |
| `make build` | Build all binaries |
| `make run-api` | Run API locally |
| `make run-ingestor` | Run ingestor locally |
| `make run-frontend` | Run frontend dev server |
| `make smoke` | Verify running stack |
| `make demo` | Start stack + show URLs |
| `make incident-demo` | Run incident simulation |

## Service Endpoints (Local)

| Service | URL | Notes |
|---------|-----|-------|
| Dashboard | http://localhost:3002 | Next.js frontend |
| Market API | http://localhost:8080 | REST API |
| Realtime | http://localhost:8081 | SSE gateway |
| Prometheus | http://localhost:9090 | Metrics |
| Grafana | http://localhost:3001 | admin/admin |
| Alertmanager | http://localhost:9093 | Alert routing |
| Mock Provider | http://localhost:8082 | Test provider |

## Key Concepts

### Data Flow
```
CoinGecko/CoinCap → Ingestor → PostgreSQL (history)
                             → Redis (cache + stream)
                             → Realtime Gateway (SSE) → Browser
```

### Resilience Pattern
```
Request → Circuit Breaker → Retry (exponential backoff) → Provider
                ↓ (open)
         Fallback Provider → Cache → Degraded Response
```

### Key Files
| File | Purpose |
|------|---------|
| `internal/provider/provider.go` | Provider interface |
| `internal/resilience/circuitbreaker.go` | Circuit breaker |
| `internal/api/router.go` | HTTP routing + middleware |
| `internal/api/middleware.go` | Security, rate limit, metrics |
| `internal/stream/consumer.go` | Redis Streams consumer |
| `internal/config/config.go` | All configuration |

## Common Issues

| Problem | Solution |
|---------|----------|
| Port 5432 in use | Docker maps to 5433; check `.env` |
| Frontend can't reach API | Ensure `make up` completed; check `frontend/.env.local` |
| Tests fail with "connection refused" | Start Docker services first: `make up` |
| golangci-lint not found | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56` |

## Next Steps

- Read [CONTRIBUTING.md](../CONTRIBUTING.md) for PR workflow
- Browse [runbooks](runbooks/) for operational procedures
- Review [SLOs](sre/slos.md) for reliability targets
- Check [SRE Scorecard](sre/scorecard.md) for current metrics
