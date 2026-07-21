# Contributing to Crypto Market Data Platform

Thank you for contributing! This guide will get you from clone to first PR in under 30 minutes.

## Quick Start

```bash
# 1. Clone and enter the repository
git clone <repo-url> && cd cryptomarket

# 2. Install dependencies
make setup

# 3. Copy environment config
cp .env.example .env
cp frontend/.env.example frontend/.env.local

# 4. Start the full stack
make up

# 5. Verify everything works
make smoke
make smoke-realtime
```

The platform is now running:
- Dashboard: http://localhost:3002
- API: http://localhost:8080
- Realtime: http://localhost:8081
- Grafana: http://localhost:3001 (admin/admin)
- Prometheus: http://localhost:9090

## Development Workflow

### Branch Strategy

We use trunk-based development with short-lived feature branches:

```bash
git checkout -b feat/my-feature    # New feature
git checkout -b fix/my-bugfix      # Bug fix
git checkout -b chore/cleanup      # Maintenance
git checkout -b docs/update-readme # Documentation
```

Branch naming: `feat/`, `fix/`, `chore/`, `docs/`, `perf/`, `test/`, `ci/`, `build/`, `revert/`

### Commit Conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

[optional body — explain WHY, not what]

[optional footer]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

**Examples**:
```
feat(provider): add Binance as tertiary data source

The platform needs a third provider for improved resilience
during CoinGecko/CoinCap outages.

Closes #142
```

```
fix(api): prevent panic on nil market data response

Circuit breaker was returning nil slice that handlers
didn't guard against.

Fixes #156
```

### Making Changes

1. Create a feature branch from `main`
2. Make your changes
3. Run checks locally:
   ```bash
   make test           # Go unit tests
   make test-frontend  # Frontend tests
   make lint           # Go linter
   make lint-frontend  # Frontend linter
   make fmt            # Format Go code
   make vet            # Go vet
   ```
4. Commit with a conventional commit message
5. Push and open a PR against `main`
6. Ensure CI passes (all green)
7. Get at least 1 approval
8. Merge (squash preferred)

## Project Structure

```
cryptomarket/
├── cmd/                    # Service entry points
│   ├── api/               # Market API server
│   ├── ingestor/          # Data ingestion worker
│   ├── realtime/          # SSE realtime gateway
│   └── mockprovider/      # Mock provider for testing
├── internal/              # Private application code
│   ├── api/              # HTTP handlers, middleware, routing
│   ├── cache/            # Redis cache and stream operations
│   ├── config/           # Environment-based configuration
│   ├── market/           # Domain models and validation
│   ├── provider/         # Provider interface and adapters
│   ├── realtime/         # SSE server implementation
│   ├── repository/       # Database access layer
│   ├── resilience/       # Circuit breaker, retry, rate limit
│   ├── scheduler/        # Periodic execution
│   ├── stream/           # Redis Streams consumer
│   ├── subscriber/       # SSE client hub
│   ├── telemetry/        # Logging and metrics
│   └── worker/           # Ingestion orchestration
├── frontend/             # Next.js dashboard
├── deploy/               # Infrastructure
│   ├── docker/          # Dockerfiles
│   ├── helm/            # Helm charts
│   ├── kubernetes/      # K8s manifests
│   └── terraform/       # AWS infrastructure
├── monitoring/           # Prometheus, Grafana, Alertmanager
├── docs/                 # Documentation and ADRs
├── scripts/              # Operational scripts
├── sre-toolkit/          # Python SRE tools
├── load-tests/           # k6 load tests
└── e2e/                  # Playwright E2E tests
```

## Coding Standards

### Go

- Format: `gofmt -s` (enforced by CI)
- Lint: `golangci-lint` with strict config (`.golangci.yml`)
- Errors: Wrap with `fmt.Errorf("context: %w", err)`
- Logging: `log/slog` structured — never `fmt.Println`
- Testing: Table-driven tests; race detector enabled
- Concurrency: Always propagate `context.Context`

### TypeScript (Frontend)

- Strict TypeScript; no `any`
- ESLint with next/core-web-vitals
- Jest + Testing Library for tests
- Test user behavior, not implementation

### Python (SRE Toolkit)

- Format/lint: `ruff`
- Testing: `pytest`
- Type hints required

## Testing

```bash
make test               # Go unit tests (fast, no deps)
make test-integration   # Integration tests (requires Docker)
make test-frontend      # Frontend unit tests
make test-e2e           # E2E tests (requires running stack)
make test-resilience    # Resilience-specific tests
make test-python        # Python SRE toolkit tests
make load-test-resilience  # k6 load tests
```

### Writing Tests

- **Unit tests**: Fast (< 1s), isolated, deterministic
- **Naming**: `TestFunction_Scenario_Expected`
- **Table-driven**: Use subtests for multiple cases
- **Race detector**: Always run with `-race`

## Architecture Decision Records (ADRs)

ADRs are required for:
- New services or data stores
- Protocol changes
- Security model changes
- Significant refactors

Format: `docs/adr/NNN-title.md` with Status → Context → Decision → Consequences.

Use the ADR issue template when proposing a new decision.

## Documentation

- Update README.md if behavior changes
- Add ADR for architectural decisions
- Update runbooks for operational changes
- Update OpenAPI spec (`api/openapi.yaml`) for API changes

## Getting Help

- Check existing [documentation](docs/)
- Review [ADRs](docs/adr/) for design decisions
- Check [runbooks](docs/runbooks/) for operational procedures
- Open an issue with the appropriate template

## Code of Conduct

Be respectful, constructive, and assume good intent. Focus on the work, not the person.
