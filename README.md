# Crypto Market Data Platform

A production-grade platform demonstrating backend engineering, SRE practices, infrastructure automation, and observability. Ingests cryptocurrency market data from public providers, normalizes and persists it, and exposes REST APIs with full monitoring.

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
                    └──────────────┘     └──────────────┘
```

- **market-api**: REST API serving latest and historical market data
- **market-ingestor**: Periodic worker fetching provider data, validating, persisting, caching, and publishing events

## Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make

## Quick Start

```bash
# 1. Copy environment config
cp .env.example .env

# 2. Start the full stack
make up

# 3. Verify
make smoke
```

The stack starts: PostgreSQL, Redis, migrations, API (port 8080), Ingestor, and Prometheus (port 9090).

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
| GET | `/metrics` | Prometheus metrics |

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
make migrate        # Run database migrations
make seed           # Seed initial coin data
make test           # Run unit tests
make lint           # Run linter
make fmt            # Format code
make vet            # Run go vet
```

## Metrics

Available at `/metrics` in Prometheus format:

- `http_request_duration_seconds` - HTTP request latency
- `http_responses_total` - HTTP response counts
- `ingestion_duration_seconds` - Ingestion cycle duration
- `ingestion_success_total` - Successful ingestion cycles
- `ingestion_failure_total` - Failed ingestion cycles
- `provider_request_duration_seconds` - Provider API latency
- `data_freshness_seconds` - Time since last successful ingestion

## Current Limitations

- Single provider (CoinGecko) with no fallback
- No authentication or rate limiting
- No WebSocket/SSE realtime delivery (planned)
- No circuit breaker (basic retry only)
- Local development focus; no production deployment yet

## Next Milestones

1. Realtime gateway (WebSocket/SSE via Redis Streams)
2. Multiple provider support with fallback
3. Circuit breaker and resilience patterns
4. Kubernetes deployment with Helm
5. Terraform infrastructure (AWS)
6. Grafana dashboards, Loki logs, Tempo traces
7. k6 load testing
8. SRE toolkit (Python): backup verification, reconciliation, incident reporting

## License

MIT
