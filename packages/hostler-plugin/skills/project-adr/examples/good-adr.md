# Exemplary ADR — passes all of Gates 1–3

This example uses a hypothetical project's "API rate limiting strategy" decision scenario.

## Authoring Process (3 Gates)

### Gate 1 — 5 Decision Threshold Questions

```
Topic: "Introduce rate limiting on the public API; adopt the token bucket algorithm"

Q1 Architecturally Significant?  ✅ (shared middleware layer; affects every endpoint)
Q2 Irreversible Enough?           ✅ (rate values will be published in client / partner contracts)
Q3 Future Team Needs Context?     ✅ (someone will ask "why token bucket and not sliding window?")
Q4 Not Covered Elsewhere?         ✅ (existing ADRs / standards / runbooks all checked)
Q5 Not Temporary?                 ✅ (permanent strategy, not a one-off event response)

→ 5/5 Yes. Proceed to Gate 2.
```

### Gate 2 — Template Selection

```
Comparing 3 alternatives is core (token bucket / sliding window / leaky bucket).
Drivers are multi-axis (accuracy / memory / burst tolerance).
→ Choose MADR 4.0.
```

### Gate 3 — Anti-Pattern Validation (post-authoring)

```
A1 Fairy Tale        : ✅ 3 Bad items in Consequences (memory growth / tuning complexity / burst abuse)
A2 Sales Pitch       : ✅ 0 marketing terms. Benchmark sources cited
A3 Free Lunch Coupon : ✅ Long-term maintenance perspective stated (3 tuning parameters to manage)
A4 Dummy Alternative : ✅ All 3 alternatives are real competing candidates
A5 Sprint            : ✅ 3m / 1y / 3y outcomes described
A6 Tunnel Vision     : ✅ Operator / partner / QA perspectives included
A7 Maze              : ✅ Title and body keywords align
A8 Blueprint         : ✅ 0 implementation-step lists (separated into a runbook)
A9 Mega-ADR          : ✅ 175 lines, 1 code-block pair
A10 Novel            : ✅ 9 sections
A11 Magic Tricks     : ✅ No scoring used

F1–F7 fallacies      : ✅ all pass
F8 AI over-confidence: ✅ Decision sentence finalized by the user
```

---

## Actual ADR Body (hypothetical ADR-N)

```markdown
---
id: ADR-N
title: ADR-N API Rate Limiting Strategy — Token Bucket
status: proposed
date: 2026-05-15
deciders: @lead-backend, @sre-on-call
relates_to: [ADR-012, ADR-M]
review_due: 2027-05-15
tags: [api, rate-limiting, reliability]
---

# ADR-N — API Rate Limiting Strategy

## Context and Problem Statement

Monthly call volume on the public REST API grew 3.5x quarter over quarter.
Beyond malicious bot traffic, partner integrations are also producing an
average of 2 over-call incidents per week. To preserve service stability
and manage partner SLAs, a rate-limiting layer is needed.

## Decision Drivers

- Stable handling of up to 1,000 req/s + brief bursts allowed
- Low memory footprint (deployed on every API gateway instance)
- Simple tuning parameters (operable by 1 person)
- Easy partner communication (the rate calculation is intuitive)

## Considered Options

1. **Token Bucket** ← selected
2. **Sliding Window Counter**
3. **Leaky Bucket**

## Decision Outcome

**Chosen option: "Token Bucket"**, because:

- Burst tolerance is naturally expressed (1 bucket-capacity parameter)
- Memory is O(1) per client — at 100k active clients, single-digit MB (driver #2)
- Easy to communicate to partners as "X req/s + up to Y burst" (driver #4)

### Consequences

- **Good**, because burst patterns are served without trimming (matches partner integration reality)
- **Good**, because memory footprint is smaller than sliding window
- **Bad**, because malicious burst patterns may abuse the bucket — capacity tuning needs care
- **Bad**, because misconfigured refill rate × capacity makes user-perceived rate diverge from the stated rate (tuning documentation is mandatory)
- **Neutral**, because the bucket is not shared across API instances — cluster-level accuracy is to be re-evaluated later (whether to adopt a Redis shared bucket)

### Confirmation

- Post-implementation load-test Task: a verification report confirming the
  bucket drains correctly up to 1.2x the nominal rate
- Confirm correct 429 responses in integration tests with 2 partner companies

## Pros and Cons of the Options

### Option 1. Token Bucket (selected)

- Good: intuitive burst tolerance
- Good: O(1) memory per client
- Bad: capacity tuning errors cause divergence between stated and actual rate

### Option 2. Sliding Window Counter

- Good: high long-window average accuracy
- Neutral: medium implementation complexity
- Bad: stores last-N-second request arrays → memory pressure
- Bad: hard to express bursts naturally

### Option 3. Leaky Bucket

- Good: strictly limits output rate → protects downstream
- Bad: cannot absorb bursts → degrades partner experience under normal patterns
- Bad: queue management overhead

## Time Dimension

| Time | Expected outcome |
|------|---------|
| +3 months | 2 partner integrations. Target 0 over-calls per week |
| +1 year | Re-evaluate need for cluster-shared bucket (Redis) |
| +3 years | Could evolve into per-API-plan-tier differentiated rate strategy |

## More Information

- ADR-M (API Gateway Selection) — basis for the middleware deployment point
- ADR-O (Observability Stack) — how rate-limit metrics are collected
- [System Design Interview §Rate Limiter](https://example.org/refs/rate-limiter)
- [Stripe rate-limit blog](https://stripe.com/blog/rate-limiters)
```

---

## Why This Is a "Good Example"

| Criterion | Score | Basis |
|-----|-----|-----|
| Gate 1 | 5/5 | Every question Yes, with rationale |
| Number of alternatives | 3 | Avoids A4 Dummy Alternative |
| Consequences | 2 pos / 2 neg / 1 neutral | Avoids A1 Fairy Tale / A3 Free Lunch |
| Length | 175 lines | Avoids A9 Mega-ADR (≤400) |
| Tense | Past / present | No RFC-like "should" language |
| Time axis | 3m / 1y / 3y | Mitigates A5 Sprint / F7 |
| Confirmation | Load test + integration test | Measurable success conditions |
| relates_to | 2 links | Not an isolated ADR |
| review_due | 1 year | F7 Time Dimension mitigation |

## Post-Approval Transition Example

### Upon Receiving the Decider's Approval

```yaml
---
id: ADR-N
title: ADR-N API Rate Limiting Strategy — Token Bucket
status: accepted                            # proposed → accepted
date: 2026-05-15
accepted_date: 2026-05-20                    # new field
deciders: @lead-backend, @sre-on-call
relates_to: [ADR-012, ADR-M]
review_due: 2027-05-15
tags: [api, rate-limiting, reliability]
---

> **Accepted 2026-05-20** — @lead-backend approved + accepted condition to issue a load-test Task.

# ADR-N — API Rate Limiting Strategy
<... body unchanged ...>
```

### Commit Message Example

```
docs(adr): ADR-N status proposed → accepted — load-test Task condition approved
```
