# Phase 7 — Infrastructure Plan: Terraform, Delivery, Capacity, Cost, Security

**Status:** Proposed — pending infrastructure gate approval
**Parent:** [Program Plan](./program-plan.md)
**Scope:** Deliverables 5, 19, 20, 21, 22, 23, 24, 28, 35, 36, 37

---

## D5. Global Traffic Management Strategy

The chosen approach is recorded in [ADR-020](../adr/020-multi-region-architecture.md); this section contains the full evaluation and per-traffic-class behavior.

### D5.1 Option evaluation

| Criterion | Route 53 | AWS Global Accelerator | Cloudflare (LB + traffic steering) | Hybrid (Cloudflare edge + Route 53 origin) |
|-----------|----------|------------------------|------------------------------------|---------------------------------------------|
| Failover routing | Health-checked failover records; 3-region health checkers | Endpoint health checks, automatic failover | Health-checked LB pools with steering policies | Both layers available |
| Latency-based routing | Yes (latency records) | Inherent (anycast + AWS backbone) | Yes (latency steering) | Yes |
| Weighted routing | Yes (0-100 weights, gradual shifts) | Traffic dials per endpoint | Yes (pool weights) | Yes |
| Failover speed | DNS TTL-bounded (60s TTL + resolver caching; worst case minutes) | Seconds (anycast, no DNS dependency) | TTL-bounded (proxy mode can be near-instant) | Mixed |
| Stale DNS cache exposure | YES — clients with cached records hit dead region until TTL expires | NO — anycast IPs are stable | Reduced (proxy absorbs) | Partial |
| SSE/long-lived connections | No awareness; connections die with region, client reconnects via DNS | Connection survives until endpoint death; new connections route correctly | Same as DNS unless proxied streams | Mixed |
| Connection draining | ALB-level only (target deregistration delay) | Endpoint-level, graceful | ALB-level | ALB-level |
| AWS integration depth | Native (health checks → ALB, CloudWatch, Terraform `aws_route53_*`) | Native but separate resource set | External; requires origin exposure | Complex |
| Cost | ~$30/mo (zone + ~50 health checks) | ~$90/mo per accelerator + LCU | Tier-dependent; LB $5+/mo + steering add-on | Both |
| Operational complexity | Low (well-understood, Terraform-managed) | Low-medium | Medium (second control plane) | High (two steering authorities to keep consistent) |
| Vendor lock-in | AWS | AWS | Cloudflare | Both |

### D5.2 Decision: Route 53 (with TTL discipline and client-side resilience)

Rationale:
1. **Single control plane.** The entire platform is AWS-native; adding Cloudflare or Global Accelerator introduces a second traffic authority whose state must be reconciled with Terraform and the failover orchestrator. At this scale, one steering system is a safety property, not just a cost saving.
2. **The DNS weakness is mitigated by the application design.** Stale DNS caches send clients to a dead region — but every client class already tolerates that: browsers retry, the frontend re-fetches on error, and SSE clients auto-reconnect (existing `useMarketStream` retry behavior). The failover RTO (15 min) is much larger than realistic DNS convergence (<2 min with 60s TTL), so Global Accelerator's seconds-level advantage buys nothing against our targets.
3. **Terraform-native.** Failover and weighted records live in the `global/dns` state (D19.4); a traffic shift is a plan-apply, fully auditable and idempotent — matching the orchestration design (failover-strategy.md D16).
4. **Cost is negligible** relative to the program envelope.

Global Accelerator is documented as the upgrade path if a future workload requires sub-minute traffic convergence or non-browser clients with aggressive DNS caching. Cloudflare remains a future CDN option for static asset acceleration only (out of Phase 7 scope; no CDN exists today — capability map §4.1).

### D5.3 Per-traffic-class behavior

| Traffic class | Routing policy | Health check | Failover behavior | TTL | Rollback |
|---------------|----------------|--------------|-------------------|-----|----------|
| **Frontend** (browser users) | Failover record `cryptomarket.example.com` → regional ALB; weighted during drills | HTTPS `/` 200 from 3 Route53 regions, 30s interval, 3/3 threshold | Automated F1 on region-unavailable; browsers retry through reconnect; static+SSR is stateless so shift is safe at any time | 60s (lowered from 300s automatically when region enters failover-candidate) | Re-apply previous weight/failover state via Terraform |
| **REST API** | Failover record `api.cryptomarket.example.com` | HTTPS `/ready` (deep check: Redis ping + DB reachable — existing endpoint semantics in `internal/api/router.go`), 30s interval | Automated F1; clients with cached DNS get connection errors and retry (API clients must implement retry — documented in API contract); no sticky state | 60s | Same |
| **Realtime (SSE)** | Failover record `realtime.cryptomarket.example.com` | HTTPS `/ready` (Redis-backed, `internal/realtime/server.go`) | Automated F1; long-lived connections DIE with the region (by design — SSE cannot be migrated); clients reconnect with `Last-Event-ID`; post-failover the new region's stream is fresh, so reconnecting clients receive a stream-reset and resync from latest-value cache (data-strategy.md D11.4) | 60s | Same |
| **Admin/observability** (Grafana, Alertmanager, operational endpoints) | NO cross-region failover records. Each region's operational surface is addressed by region-qualified hostnames (`grafana.use1.…`, `grafana.usw2.…`) | Regional only | Operators deliberately choose which region's console to use during an incident; the aggregation-point Grafana (D25, outside primary) is the incident-time surface. Failover of admin DNS during an incident would amplify confusion | 300s | n/a |

### D5.4 Cross-cutting rules

1. **TTL discipline:** normal TTL 300s; automatically lowered to 60s when any region enters `failover-candidate` (D6 state), restored 24h after returning to `healthy`. This bounds stale-cache exposure during the risk window without paying query-cost penalties in steady state.
2. **Health checks are deep, not liveness.** `/ready` semantics (dependency-checked) for API and realtime; a region returning 200 from a live process with a dead database is NOT healthy (explicit charter requirement).
3. **Controlled failover = weighted records.** Drills and planned shifts use weighted records (10/25/50/100) with health gates between stages (T2); emergency failover uses failover records (instant). Both record types coexist per hostname with the failover record as parent.
4. **Rollback is state re-application.** Traffic configuration is fully Terraform-managed; "rollback" is applying the prior known-good state — never console-click drift. Emergency console changes (break-glass) must be reconciled into Terraform within 24h (drift detection, D20).
5. **Frontend assets:** no CDN today; frontend is served by regional pods. During a regional outage, users shifted to the standby receive that region's build — version parity is guaranteed by the deployment lock (D14.2 #4), so both regions always run the same frontend version.

---

## D19. Multi-Region Terraform Program

### D19.1 Current state (evidence)

- 13 modules under `deploy/terraform/modules/` (networking, eks, rds, elasticache, s3, iam, kms, dns, acm, monitoring, secrets, secrets-rotation, waf).
- Per-environment roots (`dev`, `staging`, `prod`), each with ONE state file (`prod/terraform.tfstate` in S3 + DynamoDB lock, `deploy/terraform/environments/prod/backend.tf`).
- Single provider block per root; region variable defaults to `us-east-1`.
- No `global/` layer, no ECR module, no Route53 health-check/failover resources, no secondary-region roots.

### D19.2 Target structure

```
deploy/terraform/
├── global/                          # NEW — exactly-once, region-independent resources
│   ├── dns/                         # Hosted zone, failover+weighted records, health checks
│   ├── artifact-replication/        # ECR source repo + replication config
│   ├── iam/                         # Cross-account/cross-region CI roles, break-glass
│   └── global-observability/        # Aggregation point (remote-write target, global Grafana)
├── modules/                         # EXISTING — extended
│   ├── regional-network/            # (rename/extend of networking: adds TGW/peering hooks)
│   ├── regional-eks/                # (extend eks: adds region label, IRSA per region)
│   ├── regional-rds/                # (extend rds: adds replica creation + promotion support)
│   ├── regional-redis/              # (extend elasticache: parameterized, no global datastore)
│   ├── regional-ecr/                # NEW — regional replica repos (pull-through)
│   ├── regional-observability/      # (extend monitoring: remote-write, region labels)
│   └── regional-secrets/            # (extend secrets: replica blocks, multi-region KMS)
└── environments/
    ├── dev/                         # single-region (unchanged)
    ├── staging/
    │   ├── primary/                 # us-east-1 (refactored from staging/)
    │   └── secondary/               # us-west-2 — NEW
    └── prod/
        ├── primary/                 # us-east-1 (refactored from prod/)
        └── secondary/               # us-west-2 — NEW
```

### D19.3 State boundaries and blast radius

| State | Contents | Lock | Blast radius |
|-------|----------|------|--------------|
| `global/dns` | Hosted zone, records, health checks | DynamoDB `global-tf-lock` | Routing (guarded: deletion protection + `prevent_destroy`) |
| `global/artifacts` | ECR source + replication | same | Image availability |
| `global/iam` | CI/deploy roles, break-glass | same | Credentials |
| `global/observability` | Aggregation infra | same | Visibility |
| `env/region/primary` | All regional compute+data | Per-state DynamoDB table | One region only |
| `env/region/secondary` | All regional compute+data | Separate table | One region only |

Rules:
1. **No single state controls both regions.** A bad plan in one state cannot destroy the other region.
2. Provider aliases: regional roots use one provider (their region). `global/dns` uses `us-east-1` (Route53 is global but the provider needs a home). Cross-region references are via outputs + `terraform_remote_state` data sources (read-only), never shared state.
3. Dependency outputs: secondary region consumes primary's outputs (replica source DB identifier, ECR source registry) via remote-state data sources with explicit `depends_on` documentation. The dependency is one-way (secondary reads primary's outputs); primary never reads secondary.
4. Destroy protection: `prevent_destroy` on RDS instances (both regions), S3 backup buckets, hosted zone. `deletion_protection = true` on prod RDS (existing) extended to replica. Regional roots require `-var allow_regional_destroy=true` for any destructive plan against data resources (plan-time gate).
5. State migration: existing `prod/terraform.tfstate` is split via `terraform state mv` into `prod/primary/` — rehearsed in staging first, executed with a frozen deployment window.

### D19.4 Promotion-safe outputs

Each regional root exports a stable output contract consumed by `global/dns` and the failover orchestrator:

```
alb_dns_name, alb_hosted_zone_id, api_health_endpoint,
realtime_health_endpoint, frontend_origin, rds_endpoint, rds_status,
region_role (primary|secondary), deployed_version
```

Failover = repoint DNS records at `global/dns` state using the secondary's outputs. No regional state mutation is required for a traffic-only failover (F1), keeping the most common action blast-radius-minimal.

---

## D20. Regional Environment Parity

### D20.1 Parity contract

Both regions MUST match on: EKS version and add-on versions; Helm chart version and values (except documented regional overrides); IAM role shapes; secret KEYS (values may differ per region for endpoints); provider configuration (primary/fallback chain); scaling policy shapes (min/max may differ per capacity plan); observability stack versions and rule sets; network policy sets; RDS parameter groups; ElastiCache parameter groups; TLS certificate coverage; DNS record shapes.

Regional differences MUST be explicit, in exactly two places: `values-{env}-secondary.yaml` (documented overrides with rationale comments) and a `regional-overrides.md` manifest checked by CI.

### D20.2 Drift detection

| Layer | Mechanism | Frequency |
|-------|-----------|-----------|
| Terraform | Scheduled `terraform plan -detailed-exitcode` per state in CI; nonzero = drift page | Daily + pre-failover |
| Kubernetes | `helm diff` between deployed release and chart+values from Git; ArgoCD-style diff (even without ArgoCD, the diff job runs) | Daily |
| Secrets | Key-shape comparison (not values) across regions via read-only audit role | Weekly |
| Parameters | RDS/ElastiCache parameter group dump → diff against golden file in Git | Weekly |
| Versions | EKS/addon/chart version manifest compared across regions; skew > allowed = alert | Continuous (CI on deploy) |

Drift findings are tickets with SLO (fix or document within 5 business days). Undocumented drift blocks failover readiness (checked in the pre-failover validation step, D16 step 2).

---

## D21. Container and Artifact Replication

### D21.1 Artifact inventory and replication

| Artifact | Replication mechanism | Standby availability during primary outage |
|----------|----------------------|---------------------------------------------|
| Container images (api, ingestor, realtime, frontend) | ECR cross-region replication (registry-level, configured at repo creation; replicates on push) | YES — replica registry is regional |
| Helm charts | Git (repository) + chart OCI artifact pushed to ECR alongside images | YES — Git and replicated OCI |
| SBOMs | Generated in CI (Syft), attached as OCI artifact, replicated with images | YES |
| Signed attestations | cosign signatures on images; signatures replicate with tags | YES |
| Migration binaries | Migrations are embedded in the api image (`go run ./cmd/api migrate` pattern, `internal/repository/migrate.go`) — no separate artifact | YES (via image) |
| Operational scripts | Git (`scripts/`, `sre-toolkit/`) | YES (Git) |
| Frontend assets | Built into frontend image (Next.js standalone) — no separate CDN bucket in current design | YES (via image) |
| Terraform plans | Not replicated as artifacts; plans are regenerable from Git + state | REGENERABLE |

### D21.2 Rules

1. The standby region NEVER pulls from the primary region's registry at deploy time: deployments in the secondary reference the replicated registry URL (per-region `imageRegistry` in Helm values — extend `global.imageRegistry`).
2. Replication lag check: CI verifies image digest exists in the secondary registry before marking a release promotion-ready (images typically replicate in seconds; the check is a gate, not a wait-loop).
3. Image immutability: ECR tag immutability enabled in both regions; only immutable, signed, attested images are deployable (admission check documented; enforcement via existing supply-chain ADR-017 patterns).

---

## D22. Secret Replication Strategy

### D22.1 Current state (evidence)

- Secrets Manager secrets per environment, KMS-encrypted with a regional key (`deploy/terraform/modules/secrets/main.tf`).
- RDS uses managed master password (`manage_master_user_password = true`).
- Rotation module exists (`modules/secrets-rotation/`), single-region.
- No `replica` blocks; no multi-region KMS keys.

### D22.2 Target design

| Secret class | Replication | Encryption | Rotation behavior |
|--------------|-------------|------------|-------------------|
| App config (API keys, Redis password) | Secrets Manager multi-region replica (automatic, near-real-time) | Multi-region KMS key (same key material, regional endpoints) | Rotate in primary → replica updates automatically; verify replica freshness alarm |
| DB credentials | RDS managed master password is REGIONAL per instance; replica instance gets its own managed password post-promotion | Regional KMS | Post-promotion: app DSN secret in secondary updated to the promoted instance's credential (automated step in D16 workflow) |
| Redis auth token | Replicated | Multi-region KMS | Manual rotation window (existing doc) + replication verification |
| Provider API keys | Replicated | Multi-region KMS | Provider-dashboard rotation → update primary secret → replication |
| CI/CD deploy credentials | NOT replicated via Secrets Manager; per-region OIDC roles (GitHub Actions OIDC → regional IAM role) — no long-lived cross-region secret exists | n/a | Role policy review quarterly |

### D22.3 Failure and emergency semantics

- **Replication failure:** alarm on replica age >15 min (Secrets Manager replication is typically <1 min). Failover with stale secrets: break-glass procedure grants temporary credential creation in the secondary (audited, auto-expiring role, IC approval).
- **Revocation:** revoking in primary propagates via replication; emergency revocation in one region uses regional API directly (documented).
- **Consistency timing:** rotation runbook requires post-rotation verification that both regions' apps can read the new value (rolling restart sequencing: secondary first, then primary — inverse of deploy order, so a bad secret never takes down the active region first).
- **No primary dependency:** the secondary's secret access path is Secrets Manager regional endpoint + regional KMS — neither requires the primary region to be alive. This is verified in the DR exercise (secret read with primary region simulated down).

---

## D23. Regional CI/CD Strategy

### D23.1 Current state (evidence)

- `.github/workflows/` contains NO files. The pipeline in `docs/architecture/overview.md` (lint → security scan → build → Terraform plan → staging → smoke → canary 5/25/50 → prod) is documentation, not implementation. Release-please config exists (`.release-please-config.json`).
- This is a critical finding: Phase 7's regional delivery depends on a pipeline that must first be BUILT (single-region), then extended.

### D23.2 Deployment model decision

**Sequential, secondary-first (canary region), with primary promotion gated on secondary verification.**

Rationale: the secondary region serves no production traffic under normal topology — it is the ideal canary. A bad build detonates in the standby, never in the active region. This inverts the "primary-first" instinct deliberately.

```
local + CI validation (lint, test, SAST, image scan, SBOM, sign)
→ deploy secondary region (full rollout, it is the canary)
→ secondary verification (smoke, synthetic, SLO window 30 min)
→ deploy primary region canary (5% via existing canary template)
→ primary verification (15 min)
→ primary 25% → 50% → 100% (15 min stages, health-gated)
→ global promotion complete (release tagged, manifest updated)
```

Emergency path (SEV fix): IC may authorize primary-first or simultaneous deploy; recorded; post-hoc secondary deploy within 4h mandatory to restore parity.

### D23.3 Workflow inventory (to implement)

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `ci.yml` | PR | Lint, unit tests, Terraform validate/plan (read-only), OpenAPI lint, frontend tests |
| `build-sign.yml` | merge to main | Build images, SBOM, cosign sign, push to primary ECR, verify replication to secondary |
| `deploy-staging.yml` | merge to main | Staging both regions, e2e (Playwright, `e2e/` exists) |
| `deploy-prod.yml` | release tag | The D23.2 sequence with manual approval gates (GitHub environments) |
| `regional-rollback.yml` | manual | Per-region `helm rollback` + traffic weight adjustment |
| `dr-drill.yml` | manual (scheduled) | Standby validation suite (readiness for failover) |
| `terraform.yml` | path filter | Plan on PR, apply on merge with per-state locking |

### D23.4 Rollback semantics

- Regional rollback is independent: `helm rollback` in the affected region; version manifest updated; skew alarm suppressed during declared rollback window.
- Database schema is NEVER rolled back (expand/contract only — D8.3); code rollback must tolerate the newer schema.
- Failover configuration (DNS weights) is versioned in Terraform; "rollback" of a traffic shift = re-apply previous weight state (idempotent).

---

## D24. Multi-Region Progressive Delivery

### D24.1 Version skew policy

| Boundary | Maximum skew | Enforcement |
|----------|--------------|-------------|
| Regional application versions | 1 minor release (N and N-1) | Version manifest comparison in CI; alarm at skew; block at N-2 |
| Event contract versions | N and N-1 (v2 consumers accept v1 events per D12.1) | Contract test in CI (`internal/provider/contract_test.go` pattern extended to events) |
| Schema vs code | Code N must run against schema N and N+1 (expand/contract) | Migration dry-run against previous release in CI |
| Frontend ↔ API | API must support frontend N and N-1 | OpenAPI compatibility check |

### D24.2 Rules for safe global rollout

1. Migrations are expand-only in the deploy that introduces them; contract (cleanup) migrations ship in the NEXT release after all regions run N+1.
2. No deploy may contain an incompatible database-dependent change AND a region-simultaneous rollout — such changes are flagged in PR review by the migration check and forced into the expand/contract split.
3. Realtime clients: event contract v1/v2 dual-parse (D12) means mixed-version regions during rollout never break clients; SSE reconnection absorbs pod churn.
4. Health verification gates every stage (D23.2); a failed gate halts promotion and holds both regions at their current versions (skew within policy is the stable failure state).

---

## D28. Regional Capacity Planning

### D28.1 Load model (evidence-based)

Current production sizing (`deploy/terraform/environments/prod/main.tf`, `docs/cost/estimates.md`): EKS 4× t3.xlarge (3-12 autoscale), RDS db.r6g.large Multi-AZ, ElastiCache cache.r6g.large ×3, HPA: api 2-10, realtime 2-8, frontend 2-6, ingestor 1.

### D28.2 Capacity states

| State | Compute | Database | Redis | Notes |
|-------|---------|----------|-------|-------|
| Normal (per region, active serving) | 4 nodes, HPAs at typical 30-50% | db.r6g.large | 3-node | Baseline |
| Failover (secondary absorbs 100%) | Autoscale to 8 nodes (HPA max already allows 10/8/6 pods) | Same instance class handles read load; write load = same as normal (single writer either way) | Same | Validated by load test at 2× (D29) |
| Peak traffic (2× normal) | 8-10 nodes | Burst capacity + read path stays cache-first | Headroom in maxmemory policy | Load-tested (existing `load-tests/scale.js` 5000 VU pattern) |
| Degraded provider operation | Normal | Normal | Normal | Fallback provider may be slower → cycle time up; no capacity change; freshness SLO pressure |
| Regional maintenance (one region down for patching) | Other region at failover sizing | n/a | n/a | Maintenance = rehearsed mini-failover |

### D28.3 Headroom targets and rules

1. **Failover headroom:** secondary region node group min = 4 (same as primary), max = 12 (same); HPA maxes identical. Standby is NOT scaled to zero — hot standby means warm capacity. Cost saving comes from traffic absence, not capacity absence.
2. **Pre-provisioned vs rapid scaling:** node group scales in ~2-3 min (Cluster Autoscaler); acceptable within 15-min RTO. Reserved capacity: 4 nodes reserved per region (cost model D35); burst beyond is on-demand (never spot for prod).
3. **Database capacity:** failover does not change DB write volume (single writer); read volume shifts entirely to the promoted instance — db.r6g.large validated at 2× read load in staging. Promotion to a LARGER class is a documented contingency (`db.r6g.xlarge` pre-approved for emergency resize, ~20 min, within extended RTO).
4. **Load balancer limits:** ALB scales transparently; pre-warm by routing standby synthetic traffic (health checks + scheduled smoke) — already warm under hot-standby design.
5. **Cold-cache protection:** post-failover, all API reads hit PostgreSQL until the first ingestion cycle completes (~60s). DB must absorb 100% cache-miss load for ≤2 min — validated by the cold-cache load scenario (D29).
6. **Quarterly capacity review:** compare actual peak vs headroom; adjust node maxes and instance classes; update cost model.

---

## D35. Cost Model

Baseline: production ~$1,588/month single region (`docs/cost/estimates.md`).

### D35.1 Incremental cost of hot standby (Option B), us-west-2

| Item | Specification | Monthly |
|------|---------------|---------|
| Second EKS cluster | 1 cluster | $73 |
| Standby compute | 4× t3.xlarge reserved (same min as primary) | ~$460 (on-demand equivalent; reserved pricing assumed per optimization roadmap) |
| Secondary RDS (cross-region replica) | db.r6g.large, single-AZ (Multi-AZ optional: +$350) | $290 |
| Replication traffic (DB) | ~50GB/mo cross-region @ $0.02/GB | $1-5 |
| Secondary ElastiCache | cache.r6g.large ×3 | $280 |
| NAT gateways (secondary) | 2× | $64 |
| ALB (secondary) | 1× | $25 |
| Route53 health checks + queries | ~50 checks | $30 |
| ECR replication + storage | Replicated images | $5 |
| Cross-region data transfer (app-level: none by design; secrets/artifacts negligible) | — | $5 |
| Duplicated observability (regional Prometheus/Loki storage) | EBS + retention | $40 |
| Secrets Manager replication | Replica secrets | $2 |
| KMS multi-region keys | 2 keys | $2 |
| DR testing (staging failover drills: transient compute + transfer) | Quarterly amortized | $30 |
| **Total incremental** | | **~$1,300/month (+82%)** |

New production total: **~$2,900/month.**

### D35.2 Strategy comparison

| Strategy | Incremental cost | RTO | RPO | Cost/RTO-honesty assessment |
|----------|-----------------|-----|-----|------------------------------|
| Backup-and-restore (current) | ~$50 (cross-region copies) | 4 h | 24 h | Cheapest; unacceptable recovery for the program mission |
| Warm standby | ~$550-700 | 30-60 min | ≤5 min | Saves ~$600/mo vs hot; pays it back in 10× slower recovery and standby-rot risk |
| **Hot standby (selected)** | **~$1,300** | **≤15 min** | **≤60 s** | Best recovery-per-dollar; standby earns rent as validation environment |
| Active-active | ~$1,900-2,400 | ~0 (stateless) | 0 (with sync costs) | +50-85% over hot standby for benefits the workload cannot use (no local-write requirement); rejected (program-plan.md §5.3) |

### D35.3 Cost controls

1. Budget alerts at 80/100% of the $3,000/mo production envelope (existing Cost Explorer practice extended).
2. Cross-region transfer as a first-class budget line (replication + any accidental app-level transfer).
3. Standby uses reserved capacity at primary-matching min; burst is on-demand only during drills/failover.
4. Quarterly review: if failover drills show db.r6g.large insufficient, right-size ONCE with data, not speculatively.
5. Dev/staging remain single-region (multi-region complexity is production-only; staging gets a lightweight secondary for drills — staging secondary sized at ~$200/mo included in staging envelope growth).

---

## D36. Security Model for Multi-Region

Extends ADR-015 (security model) and `docs/security/iam-least-privilege.md`.

### D36.1 Regional IAM separation

| Principal | Primary region | Secondary region | Cross-region |
|-----------|----------------|------------------|--------------|
| Workload IRSA roles (api/ingestor/realtime) | Scoped to primary resources | Mirror roles scoped to secondary resources | NONE — workloads never reach cross-region |
| CI deploy role (OIDC) | Can deploy primary | Separate role can deploy secondary | Roles share a permission BOUNDARY but are distinct principals; a compromise of one region's deploy path does not grant the other |
| Failover orchestrator role | Read primary RDS status; cannot promote | Can promote secondary replica; cannot write data directly | Promotion permission is the sharpest knife: scoped to `rds:PromoteReadReplica` on the replica ARN only, requires MFA/session tag, CloudTrail alarm on use |
| Observability aggregation | Push to aggregation point | Push to aggregation point | Aggregation point is read-only for regions |
| Break-glass role | Per-region emergency role | Per-region emergency role | Activated only via documented procedure; 1h expiry; alarm + mandatory post-use review |

### D36.2 KMS and encryption

- Multi-region KMS keys (same material, both regions) for secrets that must decrypt in the secondary during failover (app config, Redis auth).
- Regional keys for regional-only resources (EBS, RDS storage — RDS manages its own; S3 buckets use regional keys).
- Key policy: replication administration restricted to the secrets pipeline role; decryption granted per-region workload roles.

### D36.3 CI/CD trust

- GitHub OIDC per-environment, per-region roles (no long-lived credentials — extends ADR-017 supply-chain posture).
- Deploy approval: production deploys require GitHub environment approvers; regional failover config changes require the same.
- Pipeline identity is pinned (workflow ref by tag); `GITHUB_TOKEN` permissions minimal per job (existing dependabot + gitleaks hygiene in repo: `.gitleaks.toml`, `.github/dependabot.yml`).

### D36.4 Emergency access (break-glass)

1. Trigger: failover in progress AND automation path unavailable.
2. Mechanism: assumable role with session policy restricting to the specific remediation actions; assumption requires approval-chain evidence (incident ID as session tag).
3. Guardrails: 1-hour max session; all actions CloudTrail-logged to a tamper-evident log bucket (object lock); automatic ticket creation; mandatory review within 48h; credential path rotated after use.
4. DNS control: Route53 record changes during failover go through the orchestrator role OR break-glass; hosted zone itself has change-logging enabled and is `prevent_destroy`.

### D36.5 Database promotion permissions

- `rds:PromoteReadReplica` granted ONLY to the orchestrator role, scoped to the replica ARN, with `aws:RequestedRegion` condition = secondary region.
- No human IAM principal holds promote directly; humans act through the approved workflow (which assumes the role with session tags) — preserving audit attribution while keeping the action possible during automation failure.

---

## D37. Compliance and Data Residency Considerations

### D37.1 Applicable today (factual)

| Topic | Status |
|-------|--------|
| Personal data | The platform processes PUBLIC market data (prices, volumes, provider metadata). No user accounts, no PII collection (frontend is anonymous; no auth for market data; `internal/api/auth.go` guards operational endpoints only). |
| Region selection | us-east-1 + us-west-2 (both US). Data does not leave US jurisdiction under this design. |
| Provider licensing | CoinGecko/CoinCap terms permit display with attribution; API keys are per-terms single-consumer — dual-region ingestion would violate rate terms, reinforcing the single-leader design (D13). |
| Logging/traces | Contain no PII; IPs appear in access logs — standard operational data, retention per existing Loki policy. |
| Backups | Cross-region copies stay within US regions. |
| Retention/deletion | Snapshot lifecycle via partitions (DROP PARTITION archival, `migrations/003`); S3 lifecycle policies. No right-to-erasure pipeline needed (no PII) — documented as a deliberate boundary. |

### D37.2 Hypothetical future concerns (explicitly NOT requirements)

- If user accounts are ever added: GDPR/CCPA analysis, EU region option, data-residency routing — out of scope, flagged as a future architecture gate.
- If non-US providers are added: cross-border transfer review at that time.

### D37.3 Governance

- Region pair (us-east-1/us-west-2) recorded in the architecture decision (ADR-020) with residency rationale.
- Any change to the region pair requires re-running this assessment (documented as a review trigger).
