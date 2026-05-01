# Gate 1 — Decision Threshold Detailed Guide

The 5-question checklist for deciding **whether to write an ADR**. Detailed appendix to SKILL.md §Gate 1.

## Full Expansion of the 5 Questions

### Q1. Architecturally Significant?

**Does this decision affect the system structure, key quality attributes (performance / security / availability / scalability / maintainability), or have hard-to-reverse impact?**

- ✅ **Yes examples**:
  - Decision to split monolith → microservices
  - Primary database choice (SQLite vs Postgres vs distributed)
  - Authentication architecture (session vs JWT vs OAuth)
  - Core language / runtime choice (Go vs Rust vs C#)
  - Fundamental change in deployment topology
- ❌ **No examples**:
  - Variable naming convention — → coding standards doc
  - Unifying log format — → development guide
  - Library minor version upgrade — → CHANGELOG

**Verdict heuristic**: To reverse the decision in 2 years, would you have to touch **5+ files / 2+ modules**?

### Q2. Irreversible Enough?

**To roll back, would you have to change multiple files / modules? Is data migration required?**

- ✅ **Yes examples**: DB schema change, locking down an external API spec, splitting services
- ❌ **No examples**: Replacing an internal utility function, single-module refactor

**Verdict heuristic**: ADR-worthy if Cost of Change > 1 person-day.

### Q3. Future Team Needs Context?

**Will a developer 6 months from now ask "why did we do it this way?"**

- ✅ **Yes examples**:
  - "Why did we pick SQLite over Redis?" (need the single-user baseline context)
  - "Why is UUIDv7 mandated for every ID?" (need the distributed-environment context)
- ❌ **No examples**: A decision a 1-line code comment fully covers

**Verdict heuristic**: Yes if the decision is non-obvious and **trade-offs exist**.

### Q4. Not Covered Elsewhere?

**Is it not already documented in standards / policies / existing ADRs / official docs?**

- ❌ **No (do not duplicate) examples**:
  - ADR-N already mandates UUID. Do not write a separate "use UUIDv7" ADR
  - Do not restate Go's official style guide as an ADR
  - Do not restate already-finalized conventions in the project's CLAUDE.md / README as an ADR

**Verdict heuristic**: Run `grep -ril <keyword> <project ADR dir> <standards dir>` — if you find an existing document, **update that document** or **link via relates_to in the new ADR**.

**Common mistake — duplicate ADRs for the same topic**: Two ADRs sharing the
same kebab-case title (e.g. ADR-XX + ADR-YY both `cli-output-philosophy`)
is the typical overuse pattern. Before drafting a new one, always:

```bash
ADR_DIR=${ADR_DIR:-docs/architecture}
ls "$ADR_DIR"/ADR-*.md | awk -F'/' '{print $NF}' | awk -F- '{for(i=3;i<=NF;i++) printf "%s-",$i; print ""}' | sort | uniq -d
```

### Q5. Not Temporary?

**Is it not a workaround / PoC / experiment / 1-sprint stop-gap?**

- ❌ **No examples**:
  - "Hard-coded value because we ran out of time in this sprint" — Task result section
  - "PoC shows Redis is faster" — KB card
  - "Temporary workaround applied for a production issue" — hotfix commit message
- ✅ **Yes examples**: Permanently adopted tech stack / pattern / convention

**Verdict heuristic**: Yes if the **decision lifetime > 1 iteration AND the intent is permanent**.

## Alternatives by Verdict

| Verdict | Need an ADR? | Alternative |
|-----|----------|------|
| 5/5 Yes | ★ write it | Move to Gate 2 |
| 3–4 Yes | △ conditional | Architecture Advice Process → re-evaluate |
| 0–2 Yes | ✗ unnecessary | Choose from the alternatives below |

### Alternative-Document Matrix when ADR Is Unnecessary

| Decision nature | Recommended location |
|----------|--------------|
| 1-Task code decision | Task / Ticket body's result · decision section |
| Iteration-level policy | Iteration plan goal section |
| Recurring gotcha | Knowledge base / learning card |
| Operational procedure | Runbook (docs/runbooks/) |
| Team agreement convention | Standard doc (docs/standards/ or style guide) |
| Incident root cause / response | Incident post-mortem |
| Library / dependency change | CHANGELOG + dependency manifest |
| Temporary workaround | Commit message + hotfix branch description |

## 7 Common "Don't Use ADR" Situations (detailed)

### 1. Implementation policy scoped to a single iteration

**Case**: "Implement this iteration's 7 Tasks via approach A"

**Why not an ADR**: Lifetime is bounded by the iteration. No reason to persist after it ends. Recording in the iteration plan's goal / approach section is enough.

### 2. Single-Task refactor

**Case**: "Refactor the Logger class in one ticket"

**Why not an ADR**: Single-file, single-task scope. Commit message `refactor: introduce Logger abstraction` + the task result section is enough.

### 3. Library version bump

**Case**: "Switch zap → zerolog"

**Why not an ADR**: When a library swap is not an **architectural-boundary change** but a tool swap, CHANGELOG + dependency manifest comments are enough. **However**, if it changes the logging "strategy" (e.g. introducing structured logging), it is ADR-worthy.

### 4. Documentation style convention

**Case**: "Unify markdown header levels"

**Why not an ADR**: Not an architectural decision — a **style convention**. The project style guide (e.g. `docs/standards/`) suffices.

### 5. Proof of Concept results

**Case**: "Redis vs Memcached benchmark"

**Why not an ADR**: A PoC is an **experiment**, not a **decision**. Record the results themselves as a benchmark report or knowledge base card — write the ADR only when you **decide on adoption** based on the PoC.

### 6. Meta-ADR (an ADR to approve another ADR)

**General symptom**: Creating a separate ADR like `ADR-XXX-acceptance-of-ADR-YYY.md` to approve another ADR. Common mistake.

**Why not an ADR**: The transition `status: proposed` → `accepted` of the existing ADR is fully expressed by a frontmatter edit + a 1-line Preamble. A separate ADR only creates noise.

**Fix guidance**: Mark the meta-ADR as **deprecated + deprecated_reason: "acceptance-ADR anti-pattern"**, and instead fill in the original ADR's accepted_date.

### 7. Operational procedure / runbook

**Case**: "12-step production deployment procedure"

**Why not an ADR**: ADR = **why/decision**; runbook = **how-to-execute**. Use a runbook document (e.g. `docs/runbooks/`) for repeatedly executed material.

## Practical Checklist (1-minute self-check)

```
□ Q1 Architecturally Significant?  ___
□ Q2 Irreversible Enough?          ___
□ Q3 Future Team Needs Context?    ___
□ Q4 Not Covered Elsewhere?        ___
□ Q5 Not Temporary?                ___

Verdict:
  Yes count: ___/5
  → 5      : Proceed to ADR (Gate 2)
  → 3–4    : Advice Process first
  → 0–2    : Choose an alternative document
```
