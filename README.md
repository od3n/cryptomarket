# Crypto Market Data Platform

A production-grade platform demonstrating backend engineering, SRE practices, infrastructure automation, and observability. Ingests cryptocurrency market data from public providers, normalizes and persists it, and exposes REST APIs with full monitoring. Phase 2 adds realtime delivery via Server-Sent Events and a market dashboard.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│  CoinGecko  │────▶│  Ingestor    │────▶│  PostgreSQL  │
│  Provider   │     │  (Go worker) │     │  (history)   │
└─────────────┘     └──────┬───────┘     └──────────────┘
                           │
                           ▼
                    ┌──────────────┐     ┌──────────────┐
                    │    Redis     │◀────│   Market API │
                    │ (cache+stream)│     │   (Go HTTP)  │
                    └──────┬───────┘     └──────────────┘
                           │
                           ▼
                    ┌──────────────┐     ┌──────────────┐
                    │  Realtime    │────▶│  Dashboard   │
                    │  Gateway(SSE)│     │  (Next.js)   │
                    └──────────────┘     └──────────────┘
```

- **market-api**: REST API serving latest and historical market data
- **market-ingestor**: Periodic worker fetching provider data, validating, persisting, caching, and publishing events
- **realtime-gateway**: SSE service consuming Redis Streams and broadcasting to browser clients
- **frontend**: Next.js market dashboard with live price updates

## Prerequisites

- Go 1.21+
- Node.js 20+
- Docker & Docker Compose
- Make

## Quick Start

```bash
# 1. Copy environment config
cp .env.example .env
cp frontend/.env.example frontend/.env.local

# 2. Start the full stack
make up

# 3. Verify
make smoke
make smoke-realtime
```

The stack starts: PostgreSQL, Redis, migrations, API (port 8080), Ingestor, Realtime Gateway (port 8081), Frontend (port 3000), and Prometheus (port 9090).

### Local URLs

| Service | URL |
|---------|-----|
| Dashboard | http://localhost:3000 |
| Market API | http://localhost:8080 |
| Realtime Gateway | http://localhost:8081 |
| Prometheus | http://localhost:9090 |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Application environment |
| `SERVICE_NAME` | `market-api` | Service identifier |
| `HTTP_PORT` | `8080` | API listen port |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `POSTGRES_DSN` | *(required)* | PostgreSQL connection string |
| `REDIS_ADDRESS` | `localhost:6379` | Redis host:port |
| `REDIS_PASSWORD` | *(empty)* | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `INGESTION_INTERVAL` | `60s` | Time between ingestion cycles |
| `PROVIDER_BASE_URL` | `https://api.coingecko.com/api/v3` | Provider API base URL |
| `PROVIDER_TIMEOUT` | `10s` | Provider HTTP timeout |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness check |
| GET | `/ready` | Readiness check (PostgreSQL + Redis) |
| GET | `/markets` | Latest cached market values |
| GET | `/coins` | List supported active coins |
| GET | `/coins/{symbol}` | Latest data for a coin |
| GET | `/coins/{symbol}/history` | Paginated historical snapshots |
| GET | `/providers/status` | Provider operational status |
| GET | `/metrics` | Prometheus metrics |

### Realtime Endpoint

| Method | Path | Description |
|--------|------|-------------|
| GET | `/events/markets` | SSE stream of market price updates |

See [Realtime Delivery Documentation](docs/architecture/realtime-delivery.md) for details.

### History Query Parameters

- `limit` (default 50, max 500): Number of results
- `before` (RFC3339): Snapshots before this time
- `from` (RFC3339): Snapshots from this time
- `to` (RFC3339): Snapshots up to this time

## Development

```bash
make setup          # Download dependencies
make build          # Build binaries
make run-api        # Run API locally (requires .env)
make run-ingestor   # Run ingestor locally (requires .env)
make run-realtime   # Run realtime gateway locally
make run-frontend   # Run frontend dev server
make migrate        # Run database migrations
make seed           # Seed initial coin data
make test           # Run unit tests
make test-frontend  # Run frontend tests
make lint           # Run linter
make lint-frontend  # Run frontend linting
make fmt            # Format code
make vet            # Run go vet
make demo           # Start full stack with dashboard
```

## Metrics

Available at `/metrics` in Prometheus format:

### Market API
- `http_request_duration_seconds` - HTTP request latency
- `http_responses_total` - HTTP response counts
- `ingestion_duration_seconds` - Ingestion cycle duration
- `ingestion_success_total` - Successful ingestion cycles
- `ingestion_failure_total` - Failed ingestion cycles
- `provider_request_duration_seconds` - Provider API latency
- `data_freshness_seconds` - Time since last successful ingestion

### Realtime Gateway
- `realtime_active_connections` - Current SSE connections
- `realtime_connections_total` - Total connections since start
- `realtime_events_consumed_total` - Events read from stream
- `realtime_events_broadcast_total` - Events sent to clients
- `realtime_event_validation_failures_total` - Malformed events
- `realtime_dropped_messages_total` - Messages dropped (slow clients)

## Current Limitations

- Single provider (CoinGecko) with no fallback
- No authentication or rate limiting
- No circuit breaker (basic retry only)
- Local development focus; no production deployment yet
- Frontend tests require manual verification of SSE behavior

## Next Milestones

1. Multiple provider support with fallback
2. Circuit breaker and resilience patterns
3. Kubernetes deployment with Helm
4. Terraform infrastructure (AWS)
5. Grafana dashboards, Loki logs, Tempo traces
6. k6 load testing
7. SRE toolkit (Python): backup verification, reconciliation, incident reporting

## Architecture Decision Records

- [ADR 001: Go plus Python](docs/adr/001-go-plus-python.md)
- [ADR 002: PostgreSQL and Redis Streams](docs/adr/002-postgresql-redis-streams.md)
- [ADR 003: SSE over WebSockets](docs/adr/003-sse-over-websockets.md)
- [ADR 004: Redis Streams Consumer Group](docs/adr/004-redis-streams-consumer-group.md)
- [ADR 005: Latest-State Delivery Policy](docs/adr/005-latest-state-delivery-policy.md)
- [ADR 006: Frontend Framework and Deployment](docs/adr/006-frontend-framework-deployment.md)

## License

MIT
