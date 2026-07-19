# Ingress Configuration

## Overview

The Crypto Market Platform uses NGINX Ingress Controller for external traffic routing with TLS termination, rate limiting, security headers, and compression.

## Architecture

```
Internet → AWS ALB/NLB → NGINX Ingress Controller → Service → Pods
```

## Ingress Rules

| Host                    | Path    | Backend Service   | Port |
|-------------------------|---------|-------------------|------|
| `api.cryptomarket.io`   | `/`     | market-api        | 80   |
| `ws.cryptomarket.io`    | `/`     | market-realtime   | 80   |
| `app.cryptomarket.io`   | `/`     | market-frontend   | 80   |

## TLS Configuration

- Certificates managed via AWS ACM + cert-manager
- Automatic HTTPS redirect (301)
- TLS 1.2+ only
- HSTS enabled (max-age=31536000)

```yaml
annotations:
  cert-manager.io/cluster-issuer: letsencrypt-prod
  nginx.ingress.kubernetes.io/ssl-redirect: "true"
  nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
```

## Rate Limiting

| Endpoint         | Rate Limit     | Burst | Scope     |
|------------------|----------------|-------|-----------|
| `/api/v1/*`      | 100 req/s      | 200   | Per IP    |
| `/api/v1/ticker` | 50 req/s       | 100   | Per IP    |
| `/events` (SSE)  | 10 connections | 20    | Per IP    |
| Default          | 200 req/s      | 400   | Per IP    |

```yaml
annotations:
  nginx.ingress.kubernetes.io/limit-rps: "100"
  nginx.ingress.kubernetes.io/limit-burst-multiplier: "2"
  nginx.ingress.kubernetes.io/limit-connections: "10"
```

## Security Headers

All responses include:

```yaml
annotations:
  nginx.ingress.kubernetes.io/configuration-snippet: |
    more_set_headers "X-Frame-Options: DENY";
    more_set_headers "X-Content-Type-Options: nosniff";
    more_set_headers "X-XSS-Protection: 1; mode=block";
    more_set_headers "Referrer-Policy: strict-origin-when-cross-origin";
    more_set_headers "Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'";
    more_set_headers "Permissions-Policy: camera=(), microphone=(), geolocation=()";
```

## Compression

```yaml
annotations:
  nginx.ingress.kubernetes.io/enable-compression: "true"
  nginx.ingress.kubernetes.io/compression-level: "6"
  nginx.ingress.kubernetes.io/proxy-body-size: "10m"
```

Enabled for: `application/json`, `text/html`, `text/css`, `application/javascript`, `text/event-stream`

## Caching Headers

| Path Pattern     | Cache-Control              | Rationale              |
|------------------|----------------------------|------------------------|
| `/api/v1/assets` | `public, max-age=300`      | Asset list rarely changes |
| `/api/v1/ticker` | `no-cache`                 | Real-time data         |
| `/_next/static/` | `public, max-age=31536000` | Immutable frontend assets |
| `/events`        | `no-cache, no-store`       | SSE stream             |

## Timeouts

```yaml
annotations:
  nginx.ingress.kubernetes.io/proxy-connect-timeout: "5"
  nginx.ingress.kubernetes.io/proxy-read-timeout: "60"
  nginx.ingress.kubernetes.io/proxy-send-timeout: "60"
```

For SSE (realtime):
```yaml
  nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
  nginx.ingress.kubernetes.io/proxy-buffering: "off"
```

## WebSocket/SSE Support

The realtime service uses Server-Sent Events (SSE). NGINX Ingress handles this with:
- Disabled proxy buffering for streaming responses
- Extended read timeout (1 hour)
- Connection upgrade headers

## Monitoring

Ingress metrics exposed via NGINX Ingress Controller Prometheus exporter:
- `nginx_ingress_controller_requests` — Total requests by host/path/status
- `nginx_ingress_controller_request_duration_seconds` — Latency histogram
- `nginx_ingress_controller_bytes_sent` — Bandwidth usage
- `nginx_ingress_controller_nginx_process_connections` — Active connections

## Environment Differences

| Setting        | Dev              | Staging          | Production       |
|----------------|------------------|------------------|------------------|
| TLS            | Self-signed      | Let's Encrypt    | ACM              |
| Rate limit     | 1000 req/s       | 200 req/s        | 100 req/s        |
| Replicas       | 1                | 2                | 3                |
| Domain         | dev.local        | staging.crypto.io| cryptomarket.io  |
