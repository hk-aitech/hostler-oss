# Project Integration — Frontmatter / Naming / Commit / Adjacent Artifacts

A practical convention set for keeping ADRs consistent with the project's other
artifacts (Tasks, Sprints, Feature catalogs, Knowledge Bases, runbooks).
General-purpose guidance, not tied to any specific tool or CLI.

## 1. Full Frontmatter Convention

### Required Fields (5)

```yaml
---
id: ADR-NNN                  # 3-digit zero-padded. Matches the filename prefix
title: ADR-NNN <title>        # Fixed title format
status: <state>               # proposed | accepted | rejected | superseded | deprecated
date: YYYY-MM-DD              # Proposal date (ISO 8601)
deciders: <name / role>       # Final decider(s)
---
```

### Optional but Recommended Fields

| Field | Meaning | Recommended when |
|------|------|---------|
| `uuid` | Machine identifier (UUIDv7 recommended) | Project requires graph linkage / external references |
| `supersedes` | Array of older ADRs replaced by this one | A supersede relationship arises |
| `relates_to` | Array of related ADRs (references that are not supersede) | Logical link across multiple ADRs |
| `tags` | Classification (architecture / persistence / api, etc.) | Once ADR count exceeds 20, this becomes essential |
| `review_due` | Re-review deadline (YYYY-MM-DD) | F7 Time Dimension mitigation |
| `consulted` | Advice recipients (MADR-compatible) | When using an Architecture Advice Process |
| `informed` | Decision notification list (MADR-compatible) | Broad-impact ADRs |

### Required Fields per State

| State | Additional fields |
|------|----------|
| proposed | — |
| accepted | `accepted_date: YYYY-MM-DD` |
| rejected | `rejected_date`, `rejected_reason` |
| superseded | `superseded_by: ADR-MMM`, `superseded_date` |
| deprecated | `deprecated_date`, `deprecated_reason` |

### Frontmatter Example (generic)

```yaml
---
id: ADR-N
title: ADR-N API Rate Limiting Strategy — Token Bucket
status: accepted
date: 2026-05-15
accepted_date: 2026-05-20
deciders: @lead-backend, @sre-on-call
supersedes: []
relates_to: [ADR-012, ADR-M]
tags: [api, rate-limiting, reliability]
review_due: 2027-05-15
---
```

## 2. Filename Convention

```
ADR-NNN-<kebab-case-title>.md
    ↑        ↑
    │        └─ lowercase + hyphens. No spaces / uppercase / underscores
    └─ 3-digit zero-padded (ADR-007 OK, ADR-7 NOT OK)
```

**Forbidden patterns**:
- `ADR-007_issue_hub.md` (underscores instead of hyphens)
- `ADR-7-foo.md` (missing padding)
- `ADR-007-Foo_Bar.md` (mixed case)
- `ADR-007.md` (missing title)

**Automated check**:
```bash
ADR_DIR=docs/architecture   # adjust to project convention
ls $ADR_DIR/ADR-*.md | \
  grep -vE "^$ADR_DIR/ADR-[0-9]{3}-[a-z0-9-]+\.md$" && \
  echo "WARN: filename convention violated"
```

## 3. Index File Synchronization

The README / INDEX in the ADR directory lists every ADR. Update it on each new draft and state transition.

```markdown
| ID | Title | Status |
|----|------|------|
| [ADR-044](ADR-044-api-rate-limiting.md) | API Rate Limiting Strategy | Accepted |
```

**Verification script** (README vs actual files):
```bash
ADR_DIR=docs/architecture

# ADRs listed in README
grep -oE '\[ADR-[0-9]+\]' "$ADR_DIR/README.md" | \
  sort -u > /tmp/adr-readme.txt

# Actual files
ls "$ADR_DIR"/ADR-*.md | \
  grep -oE 'ADR-[0-9]+' | sort -u > /tmp/adr-files.txt

diff /tmp/adr-readme.txt /tmp/adr-files.txt
```

## 4. Commit Message Convention

Match the project's commit convention, but recommend a pattern that makes the ADR state explicit.

### New / Transition (Conventional Commits style)

```
docs(adr): ADR-N proposed — <one-line summary>
docs(adr): ADR-N status proposed → accepted — <approval reason>
docs(adr): ADR-N status proposed → rejected — <one-line reason>
docs(adr): ADR-N status accepted → superseded — by ADR-M
docs(adr): ADR-N status accepted → deprecated — <one-line reason>
```

### Supersede (bidirectional commit)

```
docs(adr): ADR-M supersedes ADR-N — <new decision summary>
```

### General Recommendations

- **One ADR change per commit** — do not mix multiple ADRs (review difficulty)
- **Keep new-body and state-transition commits separate** — separate commits each
- If the project requires Task / Ticket IDs as commit prefixes, comply

## 5. Task / Sprint / Iteration Integration

### Task for Authoring an ADR

Splitting ADR authoring into a **separate Task** keeps work tracking clean.

```yaml
# Example: tasks/TASK-100-adr-044-rate-limiting.md
---
id: TASK-100
type: design            # or docs / research
title: "Author ADR-044 — API Rate Limiting strategy"
---

## Done Criteria
- [ ] Pass Gate 1 Decision Threshold (5 questions)
- [ ] Gate 2 template selected + frontmatter filled in
- [ ] Pass Gate 3 Anti-pattern check
- [ ] ADR Index README updated
- [ ] Commit in proposed state
```

### Splitting Approval into Its Own Task (recommended)

Splitting **proposal Task** and **approval Task** makes user/team approval wait time visible.

```
TASK-100  ADR-N proposed authored        (type: design)
TASK-101  ADR-N approval review          (type: review, blocked_by: TASK-100)
TASK-102  ADR-N accepted transition      (type: docs, blocked_by: TASK-101)
```

### Iteration / Sprint Goal Wording

If the iteration includes ADR work, include it in the goal document:

```yaml
# sprint-plan / iteration-goals
goal: "Finalize ADR-044 API Rate Limiting + complete a basic implementation PoC"
```

## 6. Feature Catalog Integration (when present)

If the project maintains a Feature Catalog (feature list, product backlog, capability map, etc.):

### When an ADR Finalizes a New Feature Boundary

1. After ADR is accepted, add an entry to the Feature Catalog
2. Cross-reference back via `adr_id: ADR-NNN` on the catalog entry
3. Optionally add `feature_id: <catalog ID>` to the ADR frontmatter

### When an ADR Redesigns an Existing Feature

Mark the Feature Catalog entry as revised:

```yaml
- feature_id: FT-042
  status: revised        # active → revised
  revised_by_adr: ADR-N
```

## 7. Relationship with Knowledge Base / Learning Cards

### Restating the Difference Between ADR and KB/Learning

- **ADR**: why this decision was made (decision · rationale · consequences)
- **KB / Learning card**: the gotcha / lesson found while implementing this decision

### Two-artifact Pattern for One Topic

```
ADR-019: Mandatory UUIDv7 for all IDs
  └─ decision rationale + trade-offs

KB Card (category: mistakes):
  "UUIDv7 timestamp bit shift — Go vs JavaScript compatibility gotcha"
```

**Rule**: **Do not mix implementation gotchas into the ADR body** — ADRs are
architecture-level; gotchas are implementation-level. If you find a gotcha
while writing the ADR, file it as a separate KB card.

## 8. Relationship with Runbooks

**ADR is "why"; the runbook is "how to execute"**.

- ADR-N: "Adopt Blue-Green deployment strategy" (decision)
- Runbook: "12-step procedure for executing Blue-Green deployment" (execution)

ADRs reference runbooks by link. Do not copy runbook content into the ADR body (avoids A8 Blueprint).

## 9. Quality Gate Integration

If the project runs CI / commit hooks / pre-merge gates, register this skill's checklist as gate rules:

### Recommended Gate Rules

```
G1. All 5 required frontmatter fields present
G2. Filename matches the convention (ADR-NNN-<kebab>.md)
G3. Registered in the Index file
G4. Anti-pattern A1 (Fairy Tale) auto-detection — warn if Consequences has 0 Bad items
G5. Anti-pattern A8 (Blueprint) auto-detection — warn if there are more than 10 numbered items
G6. Anti-pattern A9 (Mega-ADR) — block if file exceeds 400 lines
G7. Bidirectional supersede link integrity
G8. Number-gap detection (when retired, a rejected file must exist)
```

### Gate Implementation Example (simplest shell-based)

```bash
#!/bin/bash
# pre-commit hook example: when ADRs change, run this skill's checks
ADR_DIR=docs/architecture
CHANGED=$(git diff --cached --name-only | grep "^$ADR_DIR/ADR-.*\.md$")
[ -z "$CHANGED" ] && exit 0

for FILE in $CHANGED; do
  # G1 Frontmatter
  for field in id title status date deciders; do
    grep -q "^$field:" "$FILE" || { echo "BLOCK: $FILE frontmatter '$field' missing"; exit 1; }
  done

  # G6 Mega-ADR
  LC=$(wc -l < "$FILE")
  [ "$LC" -gt 400 ] && { echo "BLOCK: $FILE has $LC lines (Mega-ADR)"; exit 1; }
done

exit 0
```

## 10. AI Agent Behaviour (no auto-generation)

This skill's **core principle** — AI does not unilaterally issue ADRs:

1. **Number issuance happens after user confirmation** — even on `/project-adr new`, the AI only proposes the next number
2. **The Decision sentence is finalized by the user** — AI offers a draft; the final Decision is authored by the user
3. **Accepted transition requires an explicit user approval message** — no auto-accept
4. **No automated anti-pattern fixing** — only detection; human judgement decides the fix

**Detection**: In `/project-adr review` mode, an "AI-authored suspect" flag will fire. If the author cannot be confirmed, consider transitioning to rejected.

## 11. Document Index (Provenance Chain)

The project's CLAUDE.md / README / standard documents / architecture views
reference ADRs by **link**. **Do not duplicate the citation** — the ADR is
the source of truth (SSOT).

```
CLAUDE.md / project README
  └─ Key architecture decisions: "see ADR-N (Language Selection)"
     └─ docs/architecture/architecture-view.md (view document)
        └─ Body summarizes ADR-N + links. Do not copy the ADR body.
```

When duplication is allowed: **a 1-sentence Y-statement summary** (when a link
alone would lose context). Otherwise, link only.
