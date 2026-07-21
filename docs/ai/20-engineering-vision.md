# Engineering Vision

## The Evolution of an AI-Assisted Operational Platform

---

## 1. How the Platform Evolved

### Phase Progression

This platform did not begin as an AI project. It became AI-ready through deliberate engineering maturation:

**Phases 1-2: Foundation.** A Go monolith (single module, clear package boundaries) ingesting market data from external providers, persisting to PostgreSQL, caching in Redis, delivering via REST API. Key early decisions — string-based prices, interface-driven providers, UTC everywhere — established discipline over cleverness. The Next.js frontend and Redis Streams realtime delivery (SSE, per ADR-003) completed the user-facing surface.

**Phase 3: Resilience.** A production incident (postmortem 001: 47 minutes of stale data from CoinGecko rate limiting) drove the resilience program: multi-provider fallback (ADR-007), circuit breakers (ADR-009), retry ownership clarity (ADR-008), rate limit handling, and a failure injection toolkit with safeguards (ADR-014). This phase taught the lesson that defines the AI program: *the incident was resolved by a human following a runbook — AI's role is to compress the 19 minutes it took to identify the root cause, not to replace the identification.*

**Phases 4-5: SRE and Security.** SLOs defined with measurable SLIs (ADR-011), burn-rate alerting (ADR-012), error budget policy with deployment gates, on-call structure, severity matrix, 12 runbooks linked directly from alert annotations. Security model (ADR-015), supply chain security with signed builds (ADR-017), threat model, least-privilege IAM.

**Phase 6: Operational Excellence.** CI/CD with canary stages and SLO gates, chaos testing in staging (ADR-018), performance gates, Terraform drift detection, backup/restore verification, load testing to 5000 VU, incident demonstration scripts. The platform became *demonstrably* reliable — not just designed for reliability.

**Phase 7: Scale Planning.** Multi-region architecture (ADR-020) with data strategy, failover planning, and execution roadmap — planning for scale before needing it, documenting trade-offs honestly.

**Phase 8: AI Augmentation.** This phase. The platform's maturity — structured observability, linked runbooks, documented decisions, automated gates — is exactly what makes AI augmentation safe. The AI layer is designed as advisory infrastructure: read-only, removable, evidence-citing, human-gated.

### The Pattern

Each phase increased the platform's **legibility**:

| Phase | Legibility gained |
|-------|------------------|
| Foundation | Code structure, interfaces, contracts |
| Resilience | Failure modes, causal chains, recovery procedures |
| SRE | Reliability targets, budget semantics, alert meanings |
| Security | Threat model, access boundaries, data classification |
| Operational | Deployment safety, verification procedures, tool outputs |
| Scale | Growth constraints, capacity relationships |
| AI | Knowledge graph, retrieval architecture, evaluation scenarios |

AI augmentation is the beneficiary of this accumulated legibility. An illegible platform cannot be safely augmented — the AI would have nothing to ground against.

---

## 2. Engineering Lessons Learned

### Lesson 1: Incidents are curriculum

Postmortem 001 did not just produce fixes — it produced the causal chain model that powers alert correlation, the scenario format that powers evaluation, and the timeline structure that the incident assistant drafts against. Every incident should leave the platform more legible than it found it.

### Lesson 2: Safeguards enable speed

The failure injection toolkit exists *because* of ADR-014's safeguards (`ALLOW_FAILURE_INJECTION` guard). Chaos testing runs in CI *because* it cannot touch production. SLO gates make deployments routine *because* they make dangerous deployments impossible. The AI program inherits this pattern: kill switches and approval gates are what make advisory AI safe to ship early.

### Lesson 3: Documentation is infrastructure

The runbook annotations in alerts.yml are not documentation decoration — they are the edges of the knowledge graph. The ADR corpus is not bureaucracy — it is the constraint set for architecture review. Documentation written for humans turns out to be exactly what AI needs for grounding. The investment in docs (70+ files, consistent structure) pays compound interest.

### Lesson 4: Determinism is a feature

The platform's most reliable components are its most deterministic: recording rules, deployment gates, canary stages. The AI program deliberately preserves this: pinned prompt versions, fixed retrieval parameters, reproducible evaluation, gradeable-equivalent outputs. AI that behaves differently every time is not operational infrastructure — it is a novelty.

### Lesson 5: Measure before automating

No AI capability ships without evaluation scenarios. This discipline comes from watching automation without measurement become liability: untested alert rules create fatigue, unverified runbooks create false confidence. The evaluation framework exists because "it seems to work" is not an operational claim.

### Lesson 6: Small surface, deep integration

The platform resisted microservices (ADR-001: single module) and resisted WebSockets (ADR-003: SSE) — choosing the smaller surface that meets requirements. The AI architecture follows suit: thin assistants over existing tools, no new stateful dependencies, model swappable, everything removable. Complexity budget is finite.

---

## 3. Architecture Decisions That Shaped This Phase

| Decision | Consequence for AI Program |
|----------|---------------------------|
| ADR-002: PostgreSQL + Redis Streams | Clean data flow = retrievable state; stream events = incident timeline source |
| ADR-005: Latest-state delivery | Degraded mode semantics (ADR-013) define what "stale" means — AI can classify freshness precisely |
| ADR-011-012: SLOs + burn rates | Budget mathematics gives deployment advisor its vocabulary |
| ADR-014: Injection safeguards | Safeguard pattern reused for AI kill switches |
| ADR-017: Supply chain security | Provenance discipline extends to prompt/model versioning |
| ADR-019: Release strategy | Conventional commits + release-please = PR summary raw material |
| ADR-020: Multi-region planning | Honest "planned, not built" status = AI must handle future-state docs without treating them as current |

The AI architecture (04-architecture.md) introduces no decisions that conflict with ADRs 001-020. Where it touches existing territory, it defers: the SLO gate remains the only deployment blocker; canary stages proceed per policy; runbooks remain human-executed.

---

## 4. Future Roadmap

### Near-term (0-3 months)

1. **Governance + evaluation foundation** (AI-10, AI-9): Audit logging, kill switches, scenario library, CI gates
2. **First assistant: Incident core** (AI-2): Alert summarization + correlation + runbook recommendation — the highest-value use case per portfolio analysis
3. **Retrieval infrastructure** (AI-8): Indexed grounding with mandatory citation

Success = one assistant in production with measured precision >85%, zero safety violations, and on-call feedback confirming time savings.

### Medium-term (3-9 months)

4. **Deployment advisor** (AI-3): Advisory assessments in production workflow
5. **Repository intelligence** (AI-1): PR summaries + ADR-constraint review
6. **Documentation assistant** (AI-4): Drift detection operational
7. **Incident depth** (AI-7): RCA hypotheses + timeline drafting (requires incident corpus growth)

Success = 4 assistants operational, weekly quality reviews running, scenario library covers all incidents, operator acceptance >60%.

### Long-term (9-18 months)

8. **Cost and capacity advisors** (AI-5, AI-6): Quarterly planning inputs
9. **Multi-region readiness**: If ADR-020 execution begins, AI assists with failover verification and cross-region consistency checking
10. **Knowledge compounding**: Each incident enriches the scenario library; each drift fix strengthens the index; the system's legibility compounds

### What We Will Not Do

- Autonomous remediation (no timeline exists for this; human approval is permanent)
- AI-driven architecture decisions (ADRs remain human-authored)
- Predictive incident prevention claims (we detect and contextualize; prevention remains engineering work)
- Custom model training (commodity models + our retrieval + our evaluation = defensible without ML research)

---

## 5. Remaining Trade-offs

### Trade-off 1: Grounding vs. Coverage

Mandatory citation means the assistant cannot help where documentation is absent. A novel failure mode with no runbook, no postmortem precedent, and no relevant metric gets an honest "insufficient data" rather than a speculative answer. We accept reduced coverage in exchange for trust. The alternative — ungrounded helpfulness — destroys trust on the first confident wrong answer during a SEV-1.

### Trade-off 2: Determinism vs. Capability

Pinned prompts, fixed retrieval parameters, and reproducible evaluation constrain assistant flexibility. A more creative system might occasionally produce brilliant insights. It would also produce unexplainable variance. For operational infrastructure, explainable consistency beats occasional brilliance.

### Trade-off 3: Speed vs. Governance

Shadow periods, approval matrices, and quarterly reviews slow capability delivery. The first assistant takes ~10 weeks including foundation work when the code alone is ~4 weeks. We accept this because the alternative — shipping AI into incident response without governance — risks the operational trust that took 7 phases to build.

### Trade-off 4: Single-tenant Simplicity vs. Platform Ambition

These assistants are designed for this platform, this team, this scale. They are not a product. Generalization (multi-tenant, arbitrary repositories) would triple complexity for zero current value. If the patterns prove out, generalization is a future decision — with its own ADR.

### Trade-off 5: AI Investment vs. Direct Engineering

Every week on AI capabilities is a week not on features or multi-region execution. The bet: operational toil reduction compounds. If incident response time drops 50% and review cycles halve, the investment pays back in engineering capacity. The evaluation framework exists to test this bet honestly — and the kill switch exists to cut losses if it fails.

---

## 6. Conclusion

This platform's AI strategy is not "add AI." It is: *make the platform legible enough that AI can be safely useful, then add AI where evidence shows value, measured by outcomes, governed by humans.*

The engineering vision is an operational platform where:
- Every alert arrives with context, correlation, and a recommended next step
- Every deployment carries an evidence-based risk assessment
- Every infrastructure change is reviewed against security and cost constraints
- Every document stays current or is flagged as stale
- Every incident makes the next one easier to resolve

And where every one of those capabilities is: human-approved, deterministic, reproducible, observable, testable, explainable, secure, and reversible.

The platform remains what it has always been: a system engineers can trust at 3 AM. AI makes the 3 AM response faster and better-informed. It does not — and must not — replace the engineer.
