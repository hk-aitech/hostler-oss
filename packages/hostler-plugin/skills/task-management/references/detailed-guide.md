---
type: reference
audience: task-management users (optional reference)
purpose: Supplementary reference that does not fit into SKILL.md — Task file template / naming / priority / compression issuance / migrate heuristic
---

# Task Management Detailed Guide

> Supplementary reference that augments the core workflow in SKILL.md. The
> body cross-links into this material. Duplication is avoided — the SSOT for
> material covered by SKILL.md is SKILL.md itself (this document only carries
> the additional content).

## Table of Contents

1. Priority Definitions
2. Naming Rules
3. Full Task File Template
4. Good Task Title Examples
5. Compression-issue Pattern — Bulk Migration
6. migrate / cutover Heuristic Details

> The items SKILL.md covers (CLI entrypoints / Frontmatter Canonical / Harness
> Gate / Estimate / preflight-scope / result-section sub-headings, etc.) have
> SKILL.md as their SSOT. They are not restated here.

---

## 1. Priority Definitions

| Priority | Description | When to act | Examples |
|----------|------|----------|------|
| **P0** | Critical — blocker | Immediately | Build failure, service outage |
| **P1** | High — Sprint mandatory | Within the current Sprint | Core feature implementation |
| **P2** | Medium — improvement | When time permits | Refactor, optimization |
| **P3** | Low — nice-to-have | Backlog | Documentation, code cleanup |

---

## 2. Naming Rules

| Item | Rule | Examples |
|------|------|------|
| Task ID | Project convention, globally unique | `T001`, `T042`, `TASK-001` |
| Filename | `{ID}-{kebab-case-title}.md` | `T010-user-auth.md` |
| Sprint folder | `sprint-NN` (2 digits) | `sprint-01`, `sprint-12` |

---

## 3. Full Task File Template

```markdown
---
id: TNN
title: "{title}"
sprint: sprint-NN
status: todo
priority: p1
estimate: M
depends_on: []
created: YYYY-MM-DD
type: feature
---

# TNN: {title}

## Nature

- [ ] Brand new structure
- [ ] Update to existing structure
- [ ] Measurement / verification / deployment
- [ ] Mixed (some new + some updated)

## Purpose

{Background — why this Task is needed}

## Requirements

{Concrete requirements — measurement is recommended}

### Measurement (YYYY-MM-DD)

- Target grep command: `grep -rn "<pattern>" <path>/`
- Result: N hits (with actual file paths)
- Scope of impact: X packages / Y files

## Done Criteria

- [ ] Measurable criterion 1 (with verify command)
- [ ] Measurable criterion 2

## Scope Limits

### Included in this Sprint
- {confirmed items}

### Deferred to a future Sprint
- {deferred items — write "none" if there are no deferrals}

## References

- {related documents / track / preceding Tasks}

## Result

(Author at task complete time — using all 3 required sub-headings)

### Design Decisions
{Key choices and trade-offs}

### Artifacts
- `{full file path}` (new/modified/deleted)

### Verification
- Build: {result}
- Tests: {result}

### Commit
- {commit SHA}: {message}
```

---

## 4. Good Task Title Examples

**Good**:
- "Implement user login API"
- "Add JWT token expiration setting"
- "Fix race condition in InventoryService.UpdateStock"

**Bad**:
- "Login" (vague)
- "Bug fix" (not specific)
- "Refactor" (target unclear)

**Principle**: start with a verb (implement, add, fix, delete, refactor) + a
target noun + concrete action. Excellent if "what / why / how" is conveyed in
a single sentence.

---

## 5. Compression-issue Pattern — Bulk Migration

Apply when migrating / inducting a large set of Tasks (10–30+) at once:
migrating leftover Tasks from an external repo, reflecting audit-report
groupings, etc.

**Pre-condition**: At least one audit report or equivalent group-mapping
basis. Each group is bundled by the criterion "can be judged done together
when 1 Task is done".

### Procedure

1. **Pre-audit + group mapping** — secure the audit report's §group-mapping table
2. **Bulk body generation** — define title/type/est/purpose/reqs/criteria per
   group as dicts + author N entries from a single template string. N times
   faster than individual authoring and ensures consistency
3. **Issue via `hstl-oss task create`** — allocator issues next-id → overwrite with the authored body
4. **Scope separation: issuance ≠ implementation** — this ceremony Task only
   issues. Each integration Task is implemented when assigned to a future Sprint.
   Separating issuance / implementation minimizes the burden on the current Sprint

### Compression Criteria

| Condition | Can be combined |
|------|----------|
| Same category (planning rule / linter / HMAC, etc.) | ✅ |
| Can be judged done together when one Task is done | ✅ |
| Different ownership areas (code / docs / infra) | ❌ (split) |
| Strict implementation-order dependency | ❌ (split out) |

---

## 6. migrate / cutover Heuristic Details

Tasks containing keywords like `migrate` / `migration` / `cutover` /
`rename` / `move to` / `relocate` are structurally biased toward
underestimation. When `task create` detects the keyword combined with
`--estimate XS`, it emits a stderr WARN (exit 0, not a block).

### Recommended Minimum Estimates

| Work type | Recommended min estimate | Rationale |
|----------|-------------------|------|
| Simple rename (filename only) | S | A single grep + sed round-trip suffices; limited impact |
| Writer-code conversion (1–3 files) | S | Structural change within a single module |
| migration + home migration + writer conversion | M | 3+ areas crossed, dependency verification required |
| Module path migration / archive migration | L | Import-graph impact + cleanup of leftovers |

### Verdict Procedure

1. **Detect the keyword** — does the `task create` summary / title contain the keywords above?
2. **Measure impact via grep** — get the actual change count via `grep -rn "<old_pattern>" <repo>/ | wc -l`
   (no subjective estimation)
3. **Decide the estimate** — based on the table above + actual count. Adjust upward when WARN fires
4. **If L/XL is warranted, split** — for L+ Tasks, consider phase 1/2 split or split into separate Tasks
