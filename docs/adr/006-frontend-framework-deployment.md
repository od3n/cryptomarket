# ADR 006: Frontend Framework and Deployment Model

## Status

Accepted

## Context

Phase 2 requires a market dashboard frontend. We evaluated:

1. **Next.js** (React framework with SSR/SSG)
2. **Vite + React** (SPA)
3. **Plain React** (CRA or manual setup)

## Decision

We chose **Next.js 14** with:

- **App Router** for file-based routing
- **TypeScript** for type safety
- **Tailwind CSS** for styling
- **Standalone output** for Docker deployment
- **Reverse proxy** via Next.js rewrites (no CORS)

## Rationale

### Why Next.js

- **Built-in API proxying**: `rewrites()` in `next.config.ts` proxies `/api/*` and `/events/*` to backend services, avoiding CORS entirely
- **Production-ready**: Optimized builds, image optimization, code splitting
- **Standalone output**: Single `server.js` for minimal Docker images
- **TypeScript-first**: Excellent TS support out of the box
- **Large ecosystem**: Recharts, Testing Library, Playwright all integrate well

### Deployment Model

```
Browser → Next.js (port 3000)
              ├── /api/* → proxy → market-api (port 8080)
              ├── /events/* → proxy → realtime-gateway (port 8081)
              └── /* → Next.js pages
```

### Why not Vite/CRA

- Would require explicit CORS configuration on backends
- No built-in server-side proxying
- More configuration for production deployment

## Consequences

### Positive

- No CORS configuration needed
- Single origin for all requests
- SSR capability for SEO (if needed later)
- Well-documented deployment patterns

### Negative

- Larger bundle than minimal SPA (acceptable for dashboard)
- Node.js runtime required in production
- Learning curve for App Router conventions

## Docker Strategy

```dockerfile
# Multi-stage build
FROM node:20-alpine AS builder
# ... npm ci, npm run build

FROM node:20-alpine AS runner
# Copy standalone output
# Run as non-root user
```

## Environment Configuration

| Variable | Purpose |
|----------|---------|
| `API_URL` | Backend API URL (server-side) |
| `REALTIME_URL` | Realtime gateway URL (server-side) |
| `NEXT_PUBLIC_FRESH_THRESHOLD` | Freshness threshold (client-side) |
| `NEXT_PUBLIC_STALE_THRESHOLD` | Stale threshold (client-side) |
