# Game Day Template

> Structured failure exercises to validate resilience and team readiness.
> Frequency: Quarterly | Duration: 2-4 hours | Environment: Staging

## Game Day: [Title]

**Date**: YYYY-MM-DD
**Facilitator**: [Name]
**Participants**: [Names]
**Environment**: Staging

---

## Objectives

1. Validate [specific resilience mechanism]
2. Verify team can detect, diagnose, and recover within SLO time limits
3. Identify gaps in runbooks, alerting, or tooling
4. Practice communication protocols under pressure

## Scope

### In Scope
- [Specific services/failures to test]

### Out of Scope
- Production environment
- [Other exclusions]

### Safety Constraints
- All experiments run in staging ONLY
- CHAOS_ENV=staging safety gate enforced
- Automatic cleanup after each experiment
- Abort criteria: [define when to stop]

## Scenarios

### Scenario 1: [Name]

| Field | Value |
|-------|-------|
| **Failure** | [What breaks] |
| **Expected Detection** | [Alert that should fire, time to detect] |
| **Expected Behavior** | [How system should degrade] |
| **Recovery Target** | [Time to full recovery] |
| **Runbook** | [Link to relevant runbook] |

**Steps**:
1. Record baseline metrics
2. Inject failure: `[command]`
3. Observe for [N] minutes
4. Team responds (no hints for first 10 min)
5. Remove failure: `[command]`
6. Verify recovery

**Success Criteria**:
- [ ] Alert fired within [N] minutes
- [ ] On-call acknowledged within [N] minutes
- [ ] Correct runbook identified
- [ ] Service recovered within [N] minutes
- [ ] No data loss

### Scenario 2: [Name]
[Repeat structure]

## Timeline

| Time | Activity |
|------|----------|
| T+0 | Brief participants, review safety |
| T+15 | Scenario 1 injection |
| T+45 | Scenario 1 debrief |
| T+60 | Scenario 2 injection |
| T+90 | Scenario 2 debrief |
| T+120 | Full retrospective |

## Roles

| Role | Person | Responsibility |
|------|--------|---------------|
| Facilitator | | Injects failures, enforces timeline |
| Observer | | Documents timeline, takes notes |
| On-Call | | Responds as if real incident |
| Scribe | | Records decisions and actions |

## Results

### Detection

| Scenario | Alert Fired | Time to Detect | Target | Pass? |
|----------|-------------|---------------|--------|-------|
| 1 | | | | |
| 2 | | | | |

### Recovery

| Scenario | Time to Mitigate | Time to Resolve | Target | Pass? |
|----------|-----------------|-----------------|--------|-------|
| 1 | | | | |
| 2 | | | | |

### Findings

| # | Finding | Severity | Action Item | Owner | Due |
|---|---------|----------|-------------|-------|-----|
| 1 | | | | | |
| 2 | | | | | |

## Retrospective

### What went well
-

### What didn't go well
-

### Action Items
- [ ]

## Sign-Off

- [ ] All scenarios executed
- [ ] Findings documented
- [ ] Action items assigned with owners
- [ ] Runbooks updated (if gaps found)
- [ ] Results shared with team
