---
uuid: 019dc434-eb6a-774f-b069-6a430e06ba19
title: "3 Layer separation rationale — change rate × reference frequency matrix"
type: reference
related_skill: context-engineering
---

# 3 Layer separation rationale

> Why split the 12 core facets across three layers instead of a flat single list? Efficiency matrix + anti-patterns + cache policy.

## Table of contents

1. Separation criterion — change rate × reference frequency
2. 3-Layer efficiency matrix
3. Anti-patterns — failure modes if you use a flat single list
4. Loading-time policy
5. Cache / TTL policy (Phase 4 territory)

## 1. Separation criterion

Facets are classified along two axes (change rate × reference frequency):

| | Referenced every session | Referenced every Task | Referenced as needed |
|---|---|---|---|
| **Months–years change** | Layer 1 Stable Core | (rare)  | (rare) |
| **Days–weeks change** | (split needed) | Layer 2 Dynamic | (split needed) |
| **Minutes–days change** | (wasteful) | (wasteful) | Layer 3 On-Demand |

A facet's Layer is decided by where it sits on these two axes.

## 2. 3-Layer efficiency matrix

| Layer | Loading frequency | Cache policy | Average token cost |
|-------|-------------------|--------------|---------------------|
| Layer 1 (Stable Core) | once at session start | retained throughout the session | loaded once, reused |
| Layer 2 (Dynamic) | at Task start | retained for the Task | refreshed at Task time |
| Layer 3 (On-Demand) | at explicit invocation | short TTL (e.g. 60s) | cost paid at call time |

If an average session processes N Tasks:
- Layer 1: 1 load cost (reused over N tasks)
- Layer 2: N load costs (refreshed per Task)
- Layer 3: M load costs (number of explicit calls, M ≤ N)

A flat single list rereading 12 facets on every call costs 12 × N × (call frequency); the 3-layer split reduces this to 1 + 12 × N (Layer 2 only) + M × Layer 3 cost.

## 3. Anti-patterns — failure modes with a flat list

### Anti-pattern 1: refresh every facet on every call

```
Task start → read all 12 facets → AI invocation
```

**Problem**: rereading a monthly-changing facet like `core.objective` on every Task = waste.

**Fix**: load Layer 1 once at session start (SessionStart hook).

### Anti-pattern 2: ignore Stable Core on every call

```
Session start → no SessionStart hook → AI re-reads PDD on every call
```

**Problem**: re-reading `docs/00-project/pdd.md` every call — token waste.

**Fix**: auto-load Layer 1 + cache for the session.

### Anti-pattern 3: register an on-demand facet at Layer 1

Example: registering `plugin.<project>.market_hours` (a domain facet that changes every minute) as Layer 1.

**Problem**: reading minute-rate-of-change external state once at session start and using it stale for the rest of the session → accident.

**Fix**: register at Layer 3 + short TTL (60s).

## 4. Loading-time policy

| Time | Loaded |
|------|--------|
| Session start (Claude Code session begins) | Layer 1 (7 facets) — auto via `SessionStart` hook |
| Task start (`hstl-oss task start <ID>`) | Layer 2 (5 facets) refresh — Task-time context composition |
| Explicit call (`hstl-oss context --facets <list>`) | the explicit list (any layer) |
| Explicit call (`hstl-oss context --layer <N>`) | all of Layer N |
| Explicit call (`hstl-oss context --all`) | all facets (debug) |
| Plugin facet call (Layer 3) | at call time + short TTL cache |

## 5. Cache / TTL policy (Phase 4 territory)

> Phase 4 (JIT optimization) is the leftover item of this track (+ planned separately). The following is design intent — actual implementation lands in Phase 4.

| Layer | TTL | Cache key |
|-------|-----|-----------|
| Layer 1 | unlimited for the session | (facet, session_id) |
| Layer 2 | for the Task | (facet, task_id) |
| Layer 3 | per-facet refresh definition (e.g. 60s) | (facet, project, timestamp-bucket) |

Budget-based auto-pruning: if the call exceeds the context budget, lower-priority (less referenced) facets are auto-excluded first.

Detailed cache policy: see the upstream ADR §D8 or the Phase 4 plan.
