# ADR-001: Go for Services, Python for SRE Tooling

## Status

Accepted

## Context

The platform requires two distinct types of software:
1. High-performance backend services (API, ingestion, realtime)
2. Operational and SRE tooling (backup verification, reconciliation, incident reporting)

## Decision

Use **Go** for all backend services and **Python** for SRE/operational tooling.

## Rationale

### Go for services:
- Excellent concurrency model (goroutines) for concurrent ingestion and API serving
- Single static binary deployment simplifies container images
- Strong standard library for HTTP, JSON, and networking
- Low memory footprint suitable for containerized workloads
- Built-in race detector and profiling tools
- Fast compilation supports rapid CI/CD

### Python for SRE tooling:
- Rich ecosystem for data analysis (pandas, numpy)
- Excellent libraries for cloud provider SDKs (boto3)
- Faster development for scripts and one-off operational tasks
- Strong support for report generation and visualization
- Lower barrier for SRE team members who may not be Go specialists

## Consequences

- Two language runtimes in the repository
- Go services share a single module; Python tooling lives in `sre-toolkit/`
- CI must handle both Go and Python toolchains
- Clear boundary: Python never serves production traffic
