# ADR Lifecycle — State Transitions and Immutability

ADRs are **logs, not living documents**. Once accepted, the body is immutable;
transitions only flip `status`.

## State Graph

```
                   ┌───────────────┐
                   │   proposed    │   ← default state for a new draft
                   └───────┬───────┘
                           │
           ┌───────────────┼───────────────┐
           │ approved       │ rejected
           ▼               ▼
    ┌──────────┐    ┌──────────┐
    │ accepted │    │ rejected │ ← keep the file even on rejection (no number gaps)
    └────┬─────┘    └──────────┘
         │
   ┌─────┼─────────┐
   │ replaced by    │ context change
   │ a new ADR      │
   ▼              ▼
 ┌────────────┐  ┌────────────┐
 │ superseded │  │ deprecated │
 └────────────┘  └────────────┘
```

## Definitions per State

| State | Meaning | Frontmatter changes |
|------|------|----------------|
| **proposed** | Decision under proposal. Approver not finalized. | `status: proposed` |
| **accepted** | Approved by deciders. Implementation / adoption can proceed. | `status: accepted` + `accepted_date: YYYY-MM-DD` |
| **rejected** | Rejected by deciders. Preserved as history. | `status: rejected` + `rejected_date` + `rejected_reason` |
| **superseded** | Replaced by a newer ADR. Body retains historical value only. | `status: superseded` + `superseded_by: ADR-MMM` + `superseded_date` |
| **deprecated** | Context is gone (product wind-down, tech EOL). No replacement. | `status: deprecated` + `deprecated_date` + `deprecated_reason` |

## Transition Scenarios

### T1 — proposed → accepted (normal approval)

**Condition**: Approval message from `deciders`.

**Actions**:
1. frontmatter `status: proposed` → `accepted`
2. Add `accepted_date: YYYY-MM-DD`
3. **A 1-line Preamble is recommended** — "> Accepted YYYY-MM-DD — <reason / PR link / approval message>"
4. Body edits forbidden (typo fixes are allowed exceptions, with the git log preserving history)

**Preamble example**:
```markdown
> **Accepted 2026-05-20** — @lead-backend approved + accepted condition to issue a load-test Task.
```

**Commit message example**:
```
docs(adr): ADR-NNN status proposed → accepted — <approval reason>
```

### T2 — proposed → rejected

**Condition**: Decider rejects.

**Actions**:
1. `status: rejected`
2. `rejected_date: YYYY-MM-DD`
3. `rejected_reason: "<one-line reason>"`
4. **Do not delete the file** — preserve as a record of considered alternatives

**Why no delete**: When the same topic is proposed again, prior rejection reasons are needed to avoid repeated debate.

### T3 — accepted → superseded (replaced by a new ADR)

**Condition**: A new decision **overrides** the existing decision.

**Actions for the old ADR**:
1. `status: accepted` → `superseded`
2. Add `superseded_by: ADR-MMM`
3. `superseded_date: YYYY-MM-DD`

**Actions for the new ADR**:
1. Add `supersedes: [ADR-NNN]` to frontmatter
2. The Context section must contain **2–3 sentences explaining "why ADR-NNN's <...> was replaced"**

**Bidirectional link verification**:
```bash
# If a new ADR claims to supersede an old ADR,
# the old ADR must point back via superseded_by
ADR_DIR=${ADR_DIR:-docs/architecture}
for new in "$ADR_DIR"/ADR-*.md; do
  supersedes=$(grep -oE '^supersedes: \[ADR-[0-9]+' "$new" | grep -oE 'ADR-[0-9]+')
  for old_id in $supersedes; do
    old_file=$(ls "$ADR_DIR"/${old_id}-*.md 2>/dev/null | head -1)
    grep -q "superseded_by: ADR-" "$old_file" || echo "MISSING: $old_file lacks superseded_by"
  done
done
```

**Generic example**: If ADR-N (introduce Issue Hub) is replaced by ADR-M (switch to native Git-hosting issues), then:
- ADR-N: `status: superseded`, `superseded_by: ADR-M`
- ADR-M: `supersedes: [ADR-N]`

### T4 — accepted → deprecated (retired without replacement)

**Condition**: The context itself disappears (product wind-down, tech EOL) — **no replacement ADR**.

**Actions**:
1. `status: deprecated`
2. `deprecated_date: YYYY-MM-DD`
3. `deprecated_reason: "<reason>"`

**deprecated vs superseded**:
- **superseded** — a new ADR **re-decides** the same problem
- **deprecated** — the problem itself is gone (no decision needed any longer)

**Generic example**: A legacy compatibility ADR (e.g. "legacy-format-compatibility") — when the legacy system has been fully removed and the compatibility issue itself no longer exists, mark it deprecated.

## Immutability Conventions

### Allowed Edits

- **Typos / spelling** — allowed if traceable in git history
- **Broken-link fixes** — for URL rot
- **Status transitions** — only the `status` frontmatter field
- **Preamble additions** — 1–2 lines justifying acceptance
- **superseded_by / supersedes** — to maintain bidirectional links

### Forbidden Edits

- **Context / Decision / Consequences body**
  — if changes are needed, supersede with a new ADR
- **accepted_date** — immutable once recorded
- **deciders** — cannot retroactively change the deciders
- **UUID** — preserved permanently

### Absolutely No Deletion

- **Never delete the file in any state** — preserve superseded / deprecated / rejected alike
- **No number reuse** — keep the slot of a retired number empty (number gaps are a common operational mistake; preserve the slot via the rejected/deprecated file)
- **No git history force-push**

## Number Management

### New Number Allocation

```bash
ADR_DIR=${ADR_DIR:-docs/architecture}
NEXT=$(ls "$ADR_DIR"/ADR-*.md 2>/dev/null | \
  grep -oE 'ADR-[0-9]+' | sort -u | \
  awk -F- '{print $2+0}' | sort -n | tail -1 | \
  awk '{print $1+1}')
printf "ADR-%03d\n" "$NEXT"
```

### Number Gap Diagnosis

```bash
ADR_DIR=${ADR_DIR:-docs/architecture}
ls "$ADR_DIR"/ADR-*.md | grep -oE 'ADR-[0-9]+' | sort -u | \
  awk -F- '{print $2+0}' | sort -n | \
  awk 'NR>1 && $1 != prev+1 { for(i=prev+1;i<$1;i++) print "MISSING: ADR-"i } {prev=$1}'
```

**Verdict**: Any missing number must correspond to one of:
- A rejected file (verify via `ls $ADR_DIR/ADR-NNN-*.md`)
- A deprecated file
- A superseded file

If none exist, it is an **historical deletion mistake** — if not recoverable, note "ADR-NNN: historical, no file" in the Index (README).

## Periodic Review (Time Dimension, F7 mitigation)

### Optional `review_due` Field

```yaml
---
status: accepted
accepted_date: 2026-04-22
review_due: 2027-04-22     # re-review in 1 year
---
```

**Re-review triggers**:
- `review_due` reached
- Major release of related tech (e.g. Go 2.0, .NET 15)
- Business context change (product wind-down / expansion)

**Re-review outcomes**:
- Still valid → update `last_reviewed: YYYY-MM-DD` + extend `review_due` by 1 year
- Invalid → supersede (with a new ADR) or deprecate

## Transition Commit-message Templates (Conventional Commits)

```
docs(adr): ADR-NNN proposed (new proposal)
docs(adr): ADR-NNN status proposed → accepted — <reason>
docs(adr): ADR-NNN status proposed → rejected — <reason>
docs(adr): ADR-NNN status accepted → superseded — by ADR-MMM
docs(adr): ADR-NNN status accepted → deprecated — <reason>
docs(adr): ADR-MMM supersedes ADR-NNN — <one-line summary>
```

If the project requires a task/ticket ID prefix, adjust to that convention.
