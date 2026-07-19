# Disaster Recovery Strategy

## Overview

This document defines the disaster recovery (DR) strategy for the Crypto Market Platform running on AWS EKS. It covers loss scenarios, recovery objectives, and high-level recovery approaches.

## Recovery Objectives

| Component       | RPO       | RTO        | Strategy                     |
|-----------------|-----------|------------|------------------------------|
| API Service     | 0         | 5 min      | Multi-AZ, auto-restart       |
| Realtime        | 0         | 5 min      | Multi-AZ, auto-restart       |
| Ingestor        | 5 min     | 10 min     | Restart + replay from Redis  |
| PostgreSQL      | 5 min     | 30 min     | RDS Multi-AZ + PITR          |
| Redis           | 1 hour    | 15 min     | ElastiCache replication      |
| EKS Cluster     | 0         | 60 min     | Terraform rebuild            |
| Full Region     | 24 hours  | 4 hours    | Cross-region restore         |

## Loss Scenarios

### 1. Single Pod Failure

**Impact**: Minimal — Kubernetes auto-restarts
**Detection**: Liveness probe failure
**Recovery**: Automatic (kubelet restarts container)
**Duration**: < 30 seconds

### 2. API Service Degradation

**Impact**: Elevated error rates, increased latency
**Detection**: Prometheus alerts (error rate > 1%, p99 > 500ms)
**Recovery**:
1. Check pod logs: `kubectl logs -l app.kubernetes.io/name=market-api`
2. Scale up: `kubectl scale deployment/market-api --replicas=5`
3. If bad deploy: `kubectl rollout undo deployment/market-api`
**Duration**: 2-5 minutes

### 3. Redis Unavailable

**Impact**: Realtime delivery paused, caching disabled
**Detection**: ElastiCache CloudWatch alarm, pod readiness failures
**Recovery**:
1. ElastiCache automatic failover (Multi-AZ): ~60s
2. If cluster loss: Restore from snapshot
3. Ingestor buffers to PostgreSQL during outage
**Duration**: 1-15 minutes

### 4. PostgreSQL Unavailable

**Impact**: API read failures, ingestion halted
**Detection**: RDS CloudWatch alarm, connection errors
**Recovery**:
1. RDS Multi-AZ automatic failover: ~60-120s
2. If corruption: PITR to last known good state
3. If full loss: Restore from daily backup + WAL replay
**Duration**: 2-30 minutes

### 5. EKS Node Group Failure

**Impact**: Pods evicted, rescheduled to healthy nodes
**Detection**: Node NotReady condition
**Recovery**:
1. Kubernetes reschedules pods to remaining nodes
2. Cluster Autoscaler provisions replacement nodes
3. If AZ failure: Pods spread across remaining AZs
**Duration**: 5-15 minutes

### 6. Full AZ Failure

**Impact**: ~50% capacity loss, brief disruption
**Detection**: Multiple node NotReady, RDS failover
**Recovery**:
1. Pods reschedule to healthy AZ (topology spread constraints)
2. RDS fails over to standby in different AZ
3. ElastiCache promotes replica
4. HPA scales up to compensate
**Duration**: 5-15 minutes

### 7. Full Region Failure

**Impact**: Complete service outage
**Detection**: All health checks fail, Route53 health check
**Recovery**:
1. Provision infrastructure in DR region via Terraform
2. Restore RDS from cross-region snapshot
3. Restore Redis from S3 backup
4. Deploy Helm charts to new EKS cluster
5. Update DNS (Route53 failover routing)
**Duration**: 2-4 hours

## Prevention Measures

| Measure                    | Protects Against              |
|----------------------------|-------------------------------|
| Multi-AZ deployment        | AZ failure                    |
| Pod topology spread        | Node/AZ failure               |
| RDS Multi-AZ               | Database node failure         |
| ElastiCache replication    | Redis node failure            |
| Cross-region snapshots     | Region failure                |
| GitOps (Terraform/Helm)    | Configuration loss            |
| Automated backups          | Data corruption               |
| PDBs                       | Voluntary disruptions         |

## DR Testing Schedule

| Test                          | Frequency   | Owner    |
|-------------------------------|-------------|----------|
| Backup restore verification   | Weekly      | SRE      |
| AZ failover simulation        | Quarterly   | SRE      |
| Full DR rebuild               | Bi-annually | Platform |
| Tabletop exercise             | Quarterly   | All      |

## Communication During Incidents

1. **Detect** → Automated alert fires
2. **Assess** → On-call evaluates severity (SEV1-4)
3. **Communicate** → Update status page for SEV1/SEV2
4. **Recover** → Execute recovery procedure
5. **Post-mortem** → Document within 48 hours
