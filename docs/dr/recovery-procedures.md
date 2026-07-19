# Recovery Procedures

## Overview

Step-by-step recovery procedures for each failure scenario. Execute in order, verify at each step.

## Prerequisites

- `kubectl` configured with cluster access
- `aws` CLI configured with appropriate permissions
- `helm` v3 installed
- Access to PagerDuty/Slack for communication

---

## Procedure 1: Pod CrashLoopBackOff

**Symptoms**: Pod restarting repeatedly, `CrashLoopBackOff` status

```bash
# 1. Identify failing pods
kubectl get pods -n cryptomarket-prod | grep -i crash

# 2. Check logs
kubectl logs <pod-name> -n cryptomarket-prod --previous

# 3. Check events
kubectl describe pod <pod-name> -n cryptomarket-prod

# 4. Common fixes:
#    - Config error: Update ConfigMap/Secret, restart deployment
kubectl rollout restart deployment/market-api -n cryptomarket-prod

#    - Bad image: Rollback to previous
kubectl rollout undo deployment/market-api -n cryptomarket-prod

#    - Resource exhaustion: Increase limits in Helm values
helm upgrade cryptomarket deploy/helm/cryptomarket \
  -n cryptomarket-prod -f deploy/helm/cryptomarket/values-prod.yaml \
  --set api.resources.limits.memory=512Mi
```

**Verification**: `kubectl get pods -n cryptomarket-prod` shows Running/Ready

---

## Procedure 2: Database Failover

**Symptoms**: Connection refused, `pg_isready` fails, RDS alarm

```bash
# 1. Check RDS status
aws rds describe-db-instances \
  --db-instance-identifier cryptomarket-prod \
  --query 'DBInstances[0].DBInstanceStatus'

# 2. If Multi-AZ failover in progress, wait (60-120s)
aws rds describe-db-instances \
  --db-instance-identifier cryptomarket-prod \
  --query 'DBInstances[0].StatusInfos'

# 3. Restart affected deployments to re-establish connections
kubectl rollout restart deployment/market-api -n cryptomarket-prod
kubectl rollout restart deployment/market-ingestor -n cryptomarket-prod

# 4. Verify connectivity
kubectl exec -it deploy/market-api -n cryptomarket-prod -- \
  wget -qO- http://localhost:8080/ready
```

**If RDS is unrecoverable:**
```bash
# Restore from PITR
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier cryptomarket-prod \
  --target-db-instance-identifier cryptomarket-prod-restored \
  --use-latest-restorable-time

# Update connection string in Secrets Manager
aws secretsmanager put-secret-value \
  --secret-id cryptomarket/prod/postgres \
  --secret-string '{"host":"cryptomarket-prod-restored.xxx.rds.amazonaws.com","port":"5432","username":"cryptouser","password":"...","dbname":"cryptomarket"}'

# Restart deployments
kubectl rollout restart deployment/market-api -n cryptomarket-prod
```

---

## Procedure 3: Redis Failure

**Symptoms**: Realtime delivery stopped, cache misses at 100%

```bash
# 1. Check ElastiCache status
aws elasticache describe-cache-clusters \
  --cache-cluster-id cryptomarket-prod-redis \
  --show-cache-node-info

# 2. If automatic failover triggered, wait (~60s)

# 3. Restart realtime service
kubectl rollout restart deployment/market-realtime -n cryptomarket-prod

# 4. Verify Redis connectivity
kubectl exec -it deploy/market-realtime -n cryptomarket-prod -- \
  wget -qO- http://localhost:8081/health
```

**If ElastiCache is unrecoverable:**
```bash
# Restore from snapshot
aws elasticache create-cache-cluster \
  --cache-cluster-id cryptomarket-prod-redis-restore \
  --snapshot-name cryptomarket-prod-redis-snapshot \
  --cache-node-type cache.r6g.large \
  --engine redis

# Update connection in Secrets Manager, restart services
```

---

## Procedure 4: EKS Node Failure

**Symptoms**: Nodes NotReady, pods Pending

```bash
# 1. Check node status
kubectl get nodes
kubectl describe node <node-name>

# 2. Check for pending pods
kubectl get pods -n cryptomarket-prod --field-selector=status.phase=Pending

# 3. Cluster Autoscaler should provision new nodes (wait 5 min)
kubectl get nodes -w

# 4. If autoscaler not working, manually scale node group
aws eks update-nodegroup-config \
  --cluster-name cryptomarket-prod \
  --nodegroup-name default \
  --scaling-config minSize=3,maxSize=12,desiredSize=4

# 5. Verify pods rescheduled
kubectl get pods -n cryptomarket-prod -o wide
```

---

## Procedure 5: Full Cluster Rebuild

**Symptoms**: EKS cluster unrecoverable, control plane failure

```bash
# 1. Provision new cluster
cd deploy/terraform/environments/prod
terraform apply -target=module.networking
terraform apply -target=module.eks
terraform apply -target=module.rds
terraform apply -target=module.elasticache
terraform apply

# 2. Configure kubectl
aws eks update-kubeconfig --name cryptomarket-prod --region us-east-1

# 3. Install ingress controller
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace

# 4. Deploy application
helm install cryptomarket deploy/helm/cryptomarket \
  -n cryptomarket-prod --create-namespace \
  -f deploy/helm/cryptomarket/values-prod.yaml

# 5. Verify
kubectl get pods -n cryptomarket-prod
curl -s https://api.cryptomarket.io/health
```

---

## Procedure 6: Bad Deployment Rollback

**Symptoms**: Error rate spike after deployment

```bash
# 1. Immediate rollback
kubectl rollout undo deployment/market-api -n cryptomarket-prod

# 2. Or via Helm
helm rollback cryptomarket <previous-revision> -n cryptomarket-prod

# 3. Verify
kubectl rollout status deployment/market-api -n cryptomarket-prod
curl -s https://api.cryptomarket.io/health

# 4. If canary was in progress
helm uninstall cryptomarket-canary -n cryptomarket-prod
```

---

## Escalation Matrix

| Severity | Response Time | Escalation Path              |
|----------|---------------|------------------------------|
| SEV1     | 5 min         | On-call → Engineering Lead   |
| SEV2     | 15 min        | On-call → Team Lead          |
| SEV3     | 1 hour        | On-call (next business day)  |
| SEV4     | 4 hours       | Backlog                      |
