# Provider Resilience Architecture

## Overview

The platform implements a multi-layered resilience strategy for market data providers, ensuring continuous data flow even when individual providers fail.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Ingestor Worker                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Fallback Orchestrator                    │   │
│  │                                                     │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │   │
│  │  │ Provider │  │ Circuit  │  │  Rate Limit      │  │   │
│  │  │ Selector │  │ Breakers │  │  Tracker         │  │   │
│  │  └────┬─────┘  └────┬─────┘  └────────┬─────────┘  │   │
│  │       │              │                  │            │   │
│  │       ▼              ▼                  ▼            │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │           Retry Strategy                     │    │   │
│  │  │  (exponential backoff + jitter)              │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
│                          │                                  │
│              ┌───────────┼───────────┐                      │
│              ▼           ▼           ▼                      │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐       │
│  │  CoinGecko   │ │   CoinCap    │ │  Mock (dev)  │       │
│  │  Adapter     │ │   Adapter    │ │  Adapter     │       │
│  └──────────────┘ └──────────────┘ └──────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

## Components

### Provider Selector
- Maintains ordered list of eligible providers
- Respects disablement configuration
- Returns providers in priority order

### Circuit Breaker (per provider)
- **Closed**: Normal operation, counting failures
- **Open**: Blocking requests after threshold (5 failures)
- **Half-Open**: Probing after timeout (30s)
- Closes after 2 consecutive successes in half-open

### Rate Limit Tracker
- Tracks 429 responses per provider
- Parses `Retry-After` header (numeric + HTTP-date)
- Prevents requests during cooldown period

### Retry Strategy
- Exponential backoff with full jitter
- Max 3 attempts (configurable)
- Only retries transient errors
- Respects context cancellation

### Fallback Orchestrator
- Coordinates all components
- Iterates eligible providers in order
- Records provider switches and fallback events
- Exposes status for operational endpoint

## Data Flow

1. Scheduler triggers ingestion cycle
2. Orchestrator gets eligible providers from selector
3. For each provider:
   a. Check circuit breaker state (skip if open)
   b. Check rate limit (skip if cooling down)
   c. Attempt fetch with retry
   d. On success: record, update active provider, return
   e. On failure: record in circuit breaker, try next
4. If all fail: return error, scheduler retries next cycle

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PROVIDER_PRIMARY` | coingecko | Primary provider name |
| `PROVIDER_FALLBACK` | coincap | Comma-separated fallback list |
| `PROVIDER_DISABLED` | (empty) | Comma-separated disabled providers |
| `CIRCUIT_BREAKER_FAILURE_THRESHOLD` | 5 | Failures to open circuit |
| `CIRCUIT_BREAKER_OPEN_DURATION` | 30s | Time before half-open |
| `CIRCUIT_BREAKER_SUCCESS_THRESHOLD` | 2 | Successes to close |
| `RETRY_MAX_ATTEMPTS` | 3 | Max retry attempts |
| `RETRY_BASE_DELAY` | 1s | Base delay for backoff |
| `RETRY_MAX_DELAY` | 30s | Maximum delay cap |

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `circuit_breaker_state` | Gauge | 0=closed, 1=open, 2=half-open |
| `circuit_breaker_transitions_total` | Counter | State transitions |
| `provider_fallback_total` | Counter | Fallback events |
| `provider_switch_total` | Counter | Provider switches |
| `provider_rate_limited_total` | Counter | Rate limit events |
| `retry_attempts_total` | Counter | Retry attempts |
| `provider_active` | Gauge | Currently active provider |
