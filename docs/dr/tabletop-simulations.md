# Tabletop Simulations

## Overview

Tabletop exercises validate disaster recovery procedures through guided walkthroughs. Each simulation tests team readiness, procedure accuracy, and communication effectiveness.

## Simulation Schedule

| Exercise                    | Frequency   | Duration | Participants         |
|-----------------------------|-------------|----------|----------------------|
| Database failure            | Quarterly   | 60 min   | SRE, Backend, DBA    |
| AZ failure                  | Quarterly   | 60 min   | SRE, Platform        |
| Full region failure         | Bi-annually | 90 min   | All engineering      |
| Security incident           | Quarterly   | 60 min   | SRE, Security, Legal |
| Deployment failure          | Monthly     | 30 min   | SRE, Backend         |

---

## Simulation 1: PostgreSQL Corruption

### Scenario

At 14:30 UTC, a failed migration corrupts the `ticker_data` table. The API returns 500 errors for all ticker endpoints.

### Inject

```bash
# Simulate: Drop table in staging
kubectl exec -it deploy/market-api -n cryptomarket-staging -- \
  psql "$DATABASE_URL" -c "DROP TABLE ticker_data;"
```

### Expected Response Timeline

| Time   | Action                                          | Owner    |
|--------|-------------------------------------------------|----------|
| T+0    | Alert fires (error rate > 5%)                   | Auto     |
| T+2min | On-call acknowledges                            | SRE      |
| T+5min | Root cause identified (missing table)           | SRE      |
| T+10min| Decision: PITR vs restore from backup           | SRE+Lead |
| T+15min| Restore initiated                               | SRE      |
| T+30min| Service restored, data verified                 | SRE      |
| T+45min| Post-incident review scheduled                  | Lead     |

### Success Criteria

- [ ] Alert fired within 2 minutes
- [ ] On-call acknowledged within 5 minutes
- [ ] Correct procedure identified (PITR)
- [ ] Service restored within RTO (30 min)
- [ ] Data loss within RPO (5 min)
- [ ] Communication to stakeholders sent

---

## Simulation 2: AZ Failure

### Scenario

AWS reports degraded infrastructure in us-east-1a. 50% of EKS nodes become NotReady. RDS primary is in the affected AZ.

### Inject

```bash
# Simulate: Cordon nodes in one AZ
kubectl cordon node-1 node-2  # Nodes in us-east-1a
kubectl drain node-1 --ignore-daemonsets --delete-emptydir-data
```

### Expected Response Timeline

| Time   | Action                                          | Owner    |
|--------|-------------------------------------------------|----------|
| T+0    | Node NotReady alerts fire                       | Auto     |
| T+1min | Pods reschedule to healthy AZ                   | K8s      |
| T+2min | RDS Multi-AZ failover initiates                 | AWS      |
| T+3min | On-call acknowledges                            | SRE      |
| T+5min | Verify pods running in healthy AZs              | SRE      |
| T+8min | RDS failover complete, connections re-establish | AWS+App  |
| T+10min| Service fully operational                       | —        |

### Success Criteria

- [ ] Pods rescheduled without manual intervention
- [ ] RDS failover completed within 120s
- [ ] No data loss
- [ ] Total recovery < 15 minutes
- [ ] HPA compensated for reduced capacity

---

## Simulation 3: Bad Deployment

### Scenario

A new API version introduces a memory leak. After canary promotion to 50%, pods begin OOMKilling.

### Inject

```bash
# Simulate: Deploy with artificially low memory limit
helm upgrade cryptomarket deploy/helm/cryptomarket \
  -n cryptomarket-staging \
  --set api.resources.limits.memory=32Mi
```

### Expected Response Timeline

| Time   | Action                                          | Owner    |
|--------|-------------------------------------------------|----------|
| T+0    | Canary verification gate detects elevated errors| CI/CD    |
| T+1min | Automatic rollback triggered                    | CI/CD    |
| T+2min | Canary release uninstalled                      | CI/CD    |
| T+3min | Stable release unaffected                       | —        |
| T+5min | Team notified of failed deployment              | Auto     |

### Success Criteria

- [ ] Canary gate caught the issue before 100%
- [ ] Automatic rollback executed
- [ ] Stable traffic unaffected
- [ ] No user-facing impact
- [ ] Deployment blocked until fix

---

## Simulation 4: Redis Cluster Loss

### Scenario

ElastiCache Redis cluster experiences complete failure. Realtime SSE delivery stops. Ingestion queue backs up.

### Expected Response Timeline

| Time   | Action                                          | Owner    |
|--------|-------------------------------------------------|----------|
| T+0    | Redis connection alerts fire                    | Auto     |
| T+2min | On-call acknowledges                            | SRE      |
| T+5min | ElastiCache restore from snapshot initiated     | SRE      |
| T+10min| New Redis cluster available                     | AWS      |
| T+12min| Services restarted with new endpoint            | SRE      |
| T+15min| Realtime delivery resumed                       | —        |

### Success Criteria

- [ ] API continued serving cached/DB data (degraded mode)
- [ ] Ingestor buffered to PostgreSQL
- [ ] Redis restored within RTO (15 min)
- [ ] No permanent data loss

---

## Debrief Template

After each simulation:

1. **Timeline accuracy**: Did events match expected timeline?
2. **Procedure gaps**: Were runbooks sufficient?
3. **Tooling gaps**: Were tools available and working?
4. **Communication**: Was escalation effective?
5. **Action items**: What needs improvement?

| Action Item | Owner | Due Date | Status |
|-------------|-------|----------|--------|
|             |       |          |        |
