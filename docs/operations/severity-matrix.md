# Incident Severity Matrix

> Defines severity levels, response times, escalation paths, and communication requirements for all incidents.

## Severity Levels

| Severity | Name | Definition | Examples |
|----------|------|-----------|----------|
| **SEV-1** | Critical | Complete service outage or data loss affecting all users | API fully down, database corrupted, all providers unavailable, security breach |
| **SEV-2** | High | Major degradation affecting >50% of users or core functionality | API error rate >10%, realtime delivery failing, single provider down with no fallback |
| **SEV-3** | Medium | Partial degradation affecting <50% of users or non-core features | Elevated latency (p99 >1s), stale data >5min, single coin missing data |
| **SEV-4** | Low | Minor issue with negligible user impact | Non-critical alert, cosmetic bug, documentation error, single non-critical metric missing |

## Response Time Requirements

| Severity | Acknowledge | Mitigate | Resolve | Postmortem |
|----------|-------------|----------|---------|------------|
| **SEV-1** | 5 min | 30 min | 4 hours | Required (48h) |
| **SEV-2** | 15 min | 1 hour | 8 hours | Required (72h) |
| **SEV-3** | 1 hour | 4 hours | 24 hours | Optional |
| **SEV-4** | 4 hours | Next business day | 1 week | Not required |

## Escalation Path

```
Level 1: On-Call Engineer (initial responder)
    │
    │ (no progress in 15 min for SEV-1/2, 1 hour for SEV-3)
    ▼
Level 2: Service Owner / Tech Lead
    │
    │ (no progress in 30 min for SEV-1, 2 hours for SEV-2)
    ▼
Level 3: Engineering Manager / Platform Lead
    │
    │ (SEV-1 unresolved after 2 hours)
    ▼
Level 4: VP Engineering / Executive Sponsor
```

## Roles During an Incident

| Role | Responsibility | Assigned To |
|------|---------------|-------------|
| **Incident Commander (IC)** | Coordinates response, makes decisions, manages timeline | On-call engineer (SEV-3/4) or Tech Lead (SEV-1/2) |
| **Technical Lead** | Diagnoses root cause, implements fix | Most relevant domain expert |
| **Communications Lead** | Updates stakeholders, manages status page | IC or designated engineer |
| **Scribe** | Documents timeline, actions, decisions | Available engineer |

## Communication Requirements

### Internal

| Severity | Channel | Frequency | Audience |
|----------|---------|-----------|----------|
| SEV-1 | #incidents + war room | Every 15 min | All engineering |
| SEV-2 | #incidents | Every 30 min | Engineering leads |
| SEV-3 | #incidents | At start + resolution | On-call + service owner |
| SEV-4 | Ticket only | At resolution | Reporter + assignee |

### External (Status Page)

| Severity | Update | Timing |
|----------|--------|--------|
| SEV-1 | "Investigating" → "Identified" → "Fixing" → "Monitoring" → "Resolved" | Every 15 min during active incident |
| SEV-2 | "Investigating" → "Resolved" | At start and resolution |
| SEV-3/4 | No status page update unless user-reported | — |

## Severity Classification Guide

### Is it SEV-1?
- [ ] API returns 5xx for >10% of requests
- [ ] Complete data pipeline failure (no fresh data >10 min)
- [ ] Security breach or data exposure
- [ ] All providers unavailable simultaneously
- [ ] Database or Redis complete outage

### Is it SEV-2?
- [ ] API error rate 5-10%
- [ ] Realtime delivery completely failing
- [ ] Primary provider down, fallback struggling
- [ ] Data freshness >5 minutes
- [ ] Deployment rollback required

### Is it SEV-3?
- [ ] API latency p99 >1s for >5 minutes
- [ ] Single coin or subset of data stale
- [ ] Non-critical alert firing persistently
- [ ] Grafana dashboard unavailable
- [ ] Single non-critical endpoint failing

### Is it SEV-4?
- [ ] Intermittent non-critical errors
- [ ] Documentation inaccuracy
- [ ] Non-user-facing metric missing
- [ ] Cosmetic UI issue
- [ ] Development environment issue

## Auto-Escalation Rules

| Condition | Action |
|-----------|--------|
| SEV-1 not acknowledged in 5 min | Auto-page Level 2 |
| SEV-1 not mitigated in 30 min | Auto-page Level 3 |
| SEV-2 not acknowledged in 15 min | Auto-page Level 2 |
| Any SEV recurs within 24h | Escalate one level |
| Error budget burn >14.4x | Treat as SEV-2 minimum |

## Post-Incident Review

Required for SEV-1 and SEV-2:
1. Schedule blameless postmortem within 48h (SEV-1) or 72h (SEV-2)
2. Use template: `docs/postmortems/`
3. Include: timeline, root cause, impact, action items
4. Action items tracked with owners and deadlines
5. Review in weekly engineering meeting

## Related Documents

- [On-Call Guide](on-call.md)
- [Runbooks](../runbooks/)
- [SLOs](../sre/slos.md)
- [Alerting Strategy](../sre/alerting-strategy.md)
