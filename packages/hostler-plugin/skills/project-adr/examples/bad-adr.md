# Cautionary Tale — ADR Containing 7 Anti-patterns

This example is **deliberately badly written**. Use it to learn the anti-patterns.

## Original (the bad ADR)

```markdown
---
title: ADR-N We will use Redis!
status: accepted
---

# ADR-N — Adopting Redis

## Context

We need faster data access. Everyone uses Redis.

## Decision

We will use Redis for **all caching, session management, queues, rate limiting,
and leaderboards**. Redis is industry-leading and Netflix, Twitter, and GitHub
all use it.

## Implementation Steps

1. Install Redis 7.4
2. Edit docker-compose.yml
3. Add the redis client to the application code
4. Convert existing DB queries → Redis cache
5. Move session storage → Redis
6. Add rate-limiting middleware
7. WebSocket pub/sub via Redis
8. Implement leaderboards via Redis ZSET
9. Deploy

## Consequences

- All API responses become **dramatically** faster
- Scalability improves **10x**
- Operations become **much** easier

## Scoring

| Option | Score |
|--------|-------|
| Redis  | 9.5   |
| Memcached | 3.2 |
| Do nothing | 1.0 |

Score basis: overall satisfaction.
```

---

## Detailed Violation Analysis

### Violation 1 — A1 Fairy Tale (only upsides)

**Symptom**: Consequences has only 3 positive items. No negatives, neutrals, or trade-offs.

**Evidence**: "dramatically / 10x / much" are all positive.

**Fix direction**:
- Increased operational complexity (Redis process management, persistence strategy, memory monitoring)
- Adds another single point of failure (without HA)
- Risk of large-scale adoption without quantified trade-off measurements

### Violation 2 — A2 Sales Pitch (marketing language)

**Symptom**: "Industry-leading", "Netflix, Twitter, and GitHub all use it", "dramatically", "10x".

**Evidence**: Persuasion via emphasis terms with no quantitative benchmarks or sources.

**Fix direction**:
- Specific numbers + sources (e.g. "Redis 7 benchmark 1M QPS" + source link)
- "Used by Netflix" → "Netflix Tech Blog YYYY, role as caching layer" (source link)

### Violation 3 — A4 Dummy Alternative

**Symptom**: The Scoring table includes a "Do nothing" alternative.

**Evidence**: "Do nothing 1.0" is not an actually-considered alternative — it is intentional weakening.

**Fix direction**: List only real competing candidates — Redis / Memcached / KeyDB / in-process cache.

### Violation 4 — A8 Blueprint or Policy in Disguise (cookbook style)

**Symptom**: The "Implementation Steps" section lists 9 implementation steps.

**Evidence**: ADRs record decisions, not runbooks.

**Fix direction**:
- Remove the section → split into a runbook (e.g. `docs/runbooks/redis-deployment.md`)
- Keep only **what was decided + why** in the ADR

### Violation 5 — A10 Novel / Epic (excessive scope)

**Symptom**: 5 decisions packed into a single ADR — caching + session + queue + rate limiting + leaderboard.

**Evidence**: Decisions with different trade-offs are bundled into one document.

**Fix direction**:
- Split per use case:
  - ADR-N Session storage — Redis
  - ADR-M Rate limiting backend — Redis
  - ADR-O Real-time leaderboard — Redis ZSET
- Link via `relates_to`, group via `tags: [redis]`

### Violation 6 — A11 Magic Tricks (pseudo-quantitative scoring)

**Symptom**: Scoring table with "9.5 / 3.2 / 1.0" with no rubric.

**Evidence**: Lists only "overall satisfaction" with no actual scoring criteria.

**Fix direction**: Specify a rubric:
```
| Option | Speed (w=3) | Operational complexity (w=2) | Cost (w=1) | Total |
|--------|-----------|-------------------|------------|-------|
| Redis  | 4 → 12    | 2 → 4             | 3 → 3      | 19    |
| ...
```

### Violation 7 — F2 Following the Crowd

**Symptom**: Uses "Netflix, Twitter, and GitHub all use it" as the rationale.

**Evidence**: argumentum ad populum — no compatibility verification with our own requirements.

**Fix direction**: State "why Redis is needed in our project's baseline (scale / traffic / team size)". Netflix runs millions of QPS — we run hundreds — acknowledge the context difference.

### Additional Violation — Missing Required Frontmatter Fields

**Symptom**: Missing `id`, `date`, `deciders`. `status: accepted` but no `accepted_date`.

**Evidence**: Violates the minimum ADR frontmatter convention. Cannot pass a CI gate.

**Fix direction**:
```yaml
---
id: ADR-N
title: ADR-N <title>
status: accepted
date: 2026-05-20
accepted_date: 2026-05-25
deciders: <name / role>
---
```

---

## Violation Count & Verdict

| Category | Violations | Severity |
|---------|--------|--------|
| Content anti-patterns | A1, A2, A4, A8, A10, A11 = 6 | ★★★ |
| Fallacies | F2 = 1 | ★★ |
| Frontmatter convention | 3 missing fields | ★★★ |

**Verdict**: **Gate 3 Block** — 2+ violations. Mandatory rewrite.

---

## Rewritten ADR (a learning rewrite)

Reduced to 1 use case (session storage) + MADR 4.0 format + anti-patterns removed.

```markdown
---
id: ADR-N
title: ADR-N Session Storage — Redis over In-Process
status: proposed
date: 2026-05-20
deciders: @lead-backend
relates_to: []
tags: [persistence, session]
---

# ADR-N — Session Storage

## Context and Problem Statement

As we scaled to multiple instances, the in-process session store began to
depend on sticky-session routing. Re-deploys force frequent re-logins on users.

## Decision Drivers

- Cross-instance session sharing (no-downtime redeploy)
- Read/write latency under 5 ms
- Tolerable additional operational process count
- Team operational capacity (1 person)

## Considered Options

1. **In-process + sticky session (status quo)**
2. **Redis key-value shared store**
3. **DB table (reuse the existing RDBMS)**

## Decision Outcome

Chosen option: "Redis key-value shared store", because:
- Achievable no-downtime redeploy goal (driver #1)
- Measured p99 latency at 2–3 ms (driver #2)
- Limited to 1 additional process — Redis only (driver #3)

### Consequences

- Good, because sticky-session routing is removed → LB configuration simpler
- Good, because logins persist across redeploys
- Bad, because 1 new operational process is added — backups / monitoring required
- Bad, because Redis outage causes universal re-login (HA mandatory)
- Neutral, because compared to a DB table, latency is better but operational scope differs

### Confirmation

- Load-test Task: report p99 latency at 10k concurrent sessions
- HA configuration (2-node sentinel) build-out Task as a prerequisite

## Pros and Cons of the Options

### Option 1. In-process + sticky (status quo)

- Good: simple operations, no new process
- Bad: no-downtime redeploy impossible — does not meet the goal

### Option 2. Redis (selected)

- Good: low latency + no-downtime redeploy
- Bad: additional process operations burden

### Option 3. DB table

- Good: reuses an existing process
- Neutral: latency 10–20 ms range
- Bad: tuning burden when hot rows arise

## More Information

- [Stripe blog — scaling session storage](https://stripe.com/blog/)
- Reference for Redis Sentinel HA configuration
```

---

## Learning Points

1. **Reduced scope** — 5 decisions → 1 decision
2. **Status quo is also a valid Decision** — must always be included as an option
3. **At least 2 Bads** — prevents A1 Fairy Tale
4. **Remove rubric-less scores** — prevents A11 Magic Tricks
5. **Be explicit about context** — not "Netflix uses it", but "in our baseline"
6. **Confirmation section** — measurable success conditions
