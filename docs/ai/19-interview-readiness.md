# Interview Readiness Matrix

## Purpose

Map AI-assisted capabilities implemented on this platform to interview topics, with concrete evidence, documentation, demonstrations, and likely questions. This matrix enables engineers to speak credibly about AI-assisted platform engineering based on implemented work — not theory.

---

## Topic 1: AI-Assisted SRE

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | Incident assistant design (alert summarization, correlation, RCA hypotheses); failure injection toolkit (`sre-toolkit/inject_failures.py`, 4 scenarios); SLO deployment gate (`scripts/check-slo-gate.sh`); burn-rate alerting (multi-window pattern in `monitoring/prometheus/alerts.yml`) |
| **Documentation** | `docs/ai/06-incident-assistant.md`, `docs/sre/slos.md`, `docs/runbooks/` (12 runbooks), `docs/postmortems/001` |
| **Demonstrations** | Demo 1: rate-limit cascade replay with correlation and hypothesis generation |
| **Likely questions** | "How do you prevent AI from misdirecting an investigation?" — Confidence levels, multiple hypotheses, counter-evidence requirement, evidence citations; human confirms before acting. "How do you measure if it helps?" — Time-to-understand metric, acceptance rate, scenario replay evaluation. "What if AI is wrong during a SEV-1?" — Advisory only; on-call retains authority; kill switch disables in <1 min; post-incident review of AI involvement. |

---

## Topic 2: Operational Copilots

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | Assistant architecture with 7 defined components ([04-architecture.md](04-architecture.md)); capability maturity map covering 68 activities across 12 domains; integration via Alertmanager webhooks and CI workflows |
| **Documentation** | `docs/ai/04-architecture.md`, `docs/ai/02-capability-map.md`, `docs/ai/03-use-case-portfolio.md` (18 use cases prioritized) |
| **Demonstrations** | Demo 1 (incident copilot), Demo 2 (deployment copilot) |
| **Likely questions** | "How do you decide what AI should vs. shouldn't do?" — HITL tier model: T1 human decision through T4 recommendation-only; decision tree based on production impact. "How do copilots avoid becoming crutches?" — Intentionally-manual activities list; tier downgrade rules; acceptance tracking with rejection analysis. "Build vs. buy?" — Assistants are thin orchestration over existing tools (Prometheus, GitHub, runbooks); model is swappable; no vendor lock on reasoning layer. |

---

## Topic 3: Prompt Engineering

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | Prompt standards: 4-layer architecture (system/task/context/input); versioned templates with changelogs; adversarial test suite (injection resistance); structured output schemas with confidence reporting |
| **Documentation** | `docs/ai/12-prompt-standards.md` (versioning lifecycle, prohibited patterns, testing requirements) |
| **Demonstrations** | Adversarial test cases: PR description injection, log injection, authority escalation attempts |
| **Likely questions** | "How do you version prompts?" — Semver-inspired; eval gate on every version; shadow period for major changes; instant rollback via version pin. "How do you prevent prompt injection from PR content?" — Input treated as data; safety constraints in system prompt; adversarial suite in CI; assistants have no action authority to abuse. "How do you test prompts?" — Scenario-based eval with REQUIRED/BONUS grading; regression suite across all assistants; human review for tone/clarity. |

---

## Topic 4: Retrieval-Augmented Systems

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | Retrieval strategy: 17 source types with chunking strategy; hybrid search (semantic + keyword + structured graph lookup); mandatory citation with provenance (file, section, line, commit); secrets scrubbing pipeline |
| **Documentation** | `docs/ai/10-retrieval-strategy.md`, `docs/ai/05-knowledge-model.md` (entity-relationship model derived from parseable artifacts) |
| **Demonstrations** | Demo 6: onboarding Q&A with citation verification; citation validation catches hallucinated paths |
| **Likely questions** | "How do you handle hallucination?" — Grounding guarantee: claims without retrieved evidence marked ungrounded; citation validator checks file existence; 'no relevant sources' is a valid answer. "How do you keep the index fresh?" — Push-triggered incremental re-index; weekly full rebuild; staleness flags on old chunks. "How do you prevent secrets reaching the model?" — Pre-index exclusion (gitleaks patterns), pre-injection scrubbing, runtime output scan, zero-retention model configuration. |

---

## Topic 5: Human-in-the-Loop Automation

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | 4-tier HITL model with 40+ classified activities; existing controls preserved (SLO gate, canary stages, ADR-014 injection safeguards); escalation/downgrade rules; compliance verification via audit |
| **Documentation** | `docs/ai/14-human-in-the-loop.md`, `docs/ai/13-governance.md` (approval matrix) |
| **Demonstrations** | Kill switch exercise: all assistants disabled, zero platform impact; T2 approval record audit |
| **Likely questions** | "Where do you draw the automation line?" — Anything that could cause a production outage requires explicit human authorization; tier decision tree based on write access + business judgment. "How do you handle automation creep?" — Quarterly tier compliance audit; T3 activities gaining influence get reclassified; 90-day clean record required for upgrades. "What's your fastest path to manual control?" — Kill switch <1 min, any engineer can trigger, no approval needed to make safe. |

---

## Topic 6: AI Governance

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | Governance framework: change approval matrix; audit logging (fail-closed); model selection criteria; prompt ownership via CODEOWNERS; quarterly review checklist; privacy data classification |
| **Documentation** | `docs/ai/13-governance.md`, `docs/ai/17-quality-gates.md` (7 gates, 2 non-exceptable) |
| **Demonstrations** | Audit completeness verification; gate failure → assistant suspension behavior |
| **Likely questions** | "Who's accountable when AI gives bad advice?" — Human approver owns the decision; AI output is advisory with evidence for verification; audit trail records both. "How do you audit AI decisions?" — Every interaction logged: inputs, prompt version, model, output, human decision; append-only; 12-month retention. "What stops unauthorized AI capability expansion?" — Governance sign-off for activation; scope compliance checked quarterly; feature flags as enforcement mechanism. |

---

## Topic 7: Observability (AI-Augmented)

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | Full observability stack: Prometheus (18 alerts with runbook links, 201-line recording rules), 5 Grafana dashboards, Loki, Tempo, synthetic checks; SLO framework with 5 objectives and burn-rate alerting; AI layer: anomaly explanation, noise detection, coverage gap analysis |
| **Documentation** | `docs/sre/slos.md`, `docs/sre/alerting-strategy.md`, `docs/operations/tracing.md`, `monitoring/` |
| **Demonstrations** | Demo 1 (alert correlation using metric evidence); Demo 5 (utilization analysis from metrics) |
| **Likely questions** | "How does AI add value beyond existing alerts?" — Explanation layer (why is this anomalous?), correlation (3 alerts = 1 incident), coverage gaps (what's NOT monitored); alerts detect, AI contextualizes. "How do you distinguish anomaly from incident?" — Statistical anomaly ≠ confirmed incident; AI classifies with evidence; human confirms before incident declaration (T1 activity). "How do you handle alert fatigue?" — Tuning review cadence; noise detection from firing history; correlation reduces pages-per-incident. |

---

## Topic 8: Platform Engineering

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | Golden path tooling: Makefile (30+ targets), devcontainer, docker-compose full stack, kind + Helm local deploy, Tilt; CI/CD: 12 workflows with SLO gates, canary stages, preview environments; self-service: failure injection toolkit, chaos experiments, backup/restore automation |
| **Documentation** | `docs/onboarding.md`, `docs/deployment/`, `Makefile`, `.devcontainer/` |
| **Demonstrations** | `make demo` → full platform in minutes; Demo 7 (drift detection as platform capability) |
| **Likely questions** | "How do you measure platform success?" — Time-to-first-contribution (onboarding), deployment frequency, lead time, recovery time; developer feedback loop. "Where does AI fit in platform engineering?" — Toil reduction (PR summaries, review assistance), safety amplification (deployment advisory, infra review), knowledge access (onboarding Q&A); platform stays deterministic, AI is advisory layer. "How do you keep the platform from becoming a bottleneck?" — Self-service first (injection toolkit, preview envs); AI assistants are independently removable; zero lock-in. |

---

## Topic 9: Reliability Engineering

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | SLO framework (5 SLOs, error budgets, burn-rate policy); resilience patterns: circuit breaker, retry with backoff, provider fallback, rate limit handling (15 test files covering these); DR strategy with tabletop simulations; chaos testing program (ADR-018) with safeguards (ADR-014); postmortem culture |
| **Documentation** | `docs/sre/`, `docs/dr/`, `docs/testing/`, `docs/adr/009-circuit-breaker.md`, `docs/adr/014-failure-injection-safeguards.md` |
| **Demonstrations** | `make incident-demo` (failure injection with live SLO impact); `make test-resilience` |
| **Likely questions** | "How do you use error budgets?" — Deployment gate (budget <10% blocks), policy zones (25/50% thresholds), burn-rate alerting (14.4x fast, 6x slow); AI forecasts exhaustion. "How do you test resilience?" — Unit (circuit breaker, retry), integration (contract tests), chaos (4 injection scenarios in staging), load (k6: benchmark, resilience, 5000-VU scale), game days (template + DR exercises). "How does AI change incident response?" — Time-to-understand reduction via correlation + hypotheses; but severity declaration, remediation, and communication remain human (T1). |

---

## Topic 10: Architecture Evolution

| Attribute | Detail |
|-----------|--------|
| **Implemented evidence** | 20 ADRs documenting evolution: monolith-first (001), data layer (002), realtime (003-005), resilience (007-010), SRE (011-014), security (015, 017), performance (016), release (018-019), multi-region (020); AI layer designed as removable advisory capability (not architectural change) |
| **Documentation** | `docs/adr/` (20 decisions), `docs/architecture/`, `docs/multi-region/` (6 planning docs) |
| **Demonstrations** | Demo 3: AI checks PR against ADR constraints (architecture consistency enforcement) |
| **Likely questions** | "How do you make architecture decisions?" — ADR process: context → decision → consequences; AI assists with constraint checking and gap detection but decisions are human (T1). "How do you prevent architecture erosion?" — AI review flags ADR violations on PRs; monthly architecture review; drift detection between docs and implementation. "What's your evolution strategy?" — Monolith-first with clear boundaries; multi-region planned but not premature (ADR-020); AI capabilities are additive and removable — no architectural lock-in. "How do you decide when to evolve?" — Evidence-driven: SLO pressure, capacity forecasts, cost data; AI provides analysis, humans decide timing. |

---

## Cross-Cutting Narrative

**The story this platform tells:**

1. **Maturity first, AI second** — 7 phases of platform engineering (services, resilience, SRE, security, DR, multi-region planning) created the foundation that makes AI augmentation safe and valuable.
2. **AI as advisory layer** — Every assistant is read-only, removable, evidence-citing, and human-gated. The platform works identically with all AI disabled.
3. **Measurement over hype** — Eval scenarios, precision/recall targets, acceptance tracking, and time-saved metrics replace anecdotal claims.
4. **Governance as enabler** — Approval matrices, audit logs, and kill switches make it safe to move fast with AI capabilities.
5. **Honest limitations** — Every design documents what AI cannot do; every demo documents its limitations; every recommendation carries confidence.

**Evidence of each principle is in the repository** — interviewers can be pointed to specific files, workflows, and documents.
