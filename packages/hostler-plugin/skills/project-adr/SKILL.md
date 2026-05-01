---
name: project-adr
description: Standard skill for ADR (Architecture Decision Record) authoring, operations, and governance. Provides three template options (Michael Nygard original / MADR 4.0 / Y-statement), 11 anti-patterns, 7 fallacies, and lifecycle conventions. Use this skill whenever the user asks "should I write an ADR", "is this worth turning into an ADR", "draft ADR-NNN", "is my ADR correct", "audit our ADR usage", "supersede this ADR", "ADR vs RFC", invokes `/project-adr`, or whenever an accept / superseded / deprecated state transition is needed. A general-purpose governance skill that prevents ADR overuse and misuse in any project. Be a little pushy — invoke this even if the user is just musing about whether something deserves an ADR.
when_to_use: |
  Required — before writing an ADR (Decision Threshold check), while drafting (template selection), after drafting (anti-pattern review), at superseded/deprecated state transitions, and when auditing the entire ADR directory.
  Forbidden — recording iteration / task-level decisions (use a learning/retrospective skill), drafting a new RFC / design doc, creating knowledge base cards.
compatibility:
  tools: [Read, Grep, Glob, Bash, Write, Edit]
argument-hint: "[check|new|review|supersede <ADR-NNN>]"
paths: ["docs/02-architecture/**/*", "docs/architecture/**/*", "**/adr/**/*.md", "**/decisions/**/*.md"]
user-invocable: true
trigger_commands: []
---

# project-adr — ADR Authoring · Operations · Governance Standard

**Trigger phrases**: "should I write an ADR", "review this ADR", "ADR governance check",
"supersede ADR", "ADR lifecycle", "ADR anti-pattern check", "draft ADR-NNN",
"is my ADR correct", "audit our ADR usage", "deprecate this ADR".

## Skill Positioning

This is a general-purpose governance skill that **prevents ADR overuse and misuse in any project**.
It runs the candidate decision through three sequential gates:

```
┌─ Gate 1 ─────────────────────────────────────────────┐
│  Decision Threshold                                   │
│  Does this decision really need an ADR?               │
│  → If not: redirect to commit message / task note    │
└───────────────────────────────────────────────────────┘
                  ↓ pass
┌─ Gate 2 ─────────────────────────────────────────────┐
│  Template Selection                                    │
│  Choose Nygard / MADR 4.0 / Y-statement               │
└───────────────────────────────────────────────────────┘
                  ↓ pass
┌─ Gate 3 ─────────────────────────────────────────────┐
│  Anti-Pattern Review                                   │
│  Run all 11 anti-patterns + 7 fallacies               │
│  → On violation: redirect to fix; Block if severe     │
└───────────────────────────────────────────────────────┘
                  ↓ pass
      Accept → commit to project ADR directory
```

**Skill scope**: file **creation proposal + validation + state transition guidance** —
actual file writes happen only after user approval. The skill does NOT auto-allocate
the "ADR-NNN" number (numbering is decided manually after Gate 3, comparing against
the project's ADR Index).

**ADR directory path**: varies per project (`docs/architecture/`,
`docs/adr/`, `adr/`, user project docs, etc.). The skill registers the
common conventions in the `paths` frontmatter, and all command examples
use the `$ADR_DIR` variable.

## Before Writing — Gate 1: Decision Threshold

**5 core questions**:

| # | Question | Yes → ADR needed | No → ADR not needed |
|---|------|---------------|-----------------|
| 1 | **Architecturally Significant?** — Does this affect system structure, key quality attributes, or have hard-to-reverse impact? | ✅ | ❌ |
| 2 | **Irreversible Enough?** — To roll back, would you have to change multiple files / modules? | ✅ | ❌ |
| 3 | **Future Team Needs Context?** — Will a developer 6 months from now ask "why did we do it this way?" | ✅ | ❌ |
| 4 | **Not Covered Elsewhere?** — Is it not already documented in standards / policies / existing ADRs / official docs? | ✅ | ❌ (no duplicates) |
| 5 | **Not Temporary?** — Is it not a workaround / PoC / experiment / 1-iteration ad-hoc fix? | ✅ | ❌ |

**Verdict rules**:

- **5/5 Yes** → proceed to ADR (move to Gate 2)
- **3–4 Yes** → run an Architecture Advice Process first (gather stakeholder input) → re-evaluate
- **0–2 Yes** → no ADR needed. Choose one of:
  - The decision/result section of a Task / Ticket
  - A Knowledge base / learning card (if the project has one)
  - A 1-line code comment (only the WHY)
  - An iteration / sprint retrospective entry

### 7 Common Situations Where ADRs **Should NOT Be Written**

1. **Implementation policy scoped to 1 iteration** — record in the iteration plan / task body
2. **A code refactor finished in a single task** — commit message + task result section is enough
3. **Library version bump** — CHANGELOG / dependency-file comment
4. **Documentation style convention** — add to a standard (style guide)
5. **Proof of Concept results** — knowledge base card or experiment report
6. **A meta-ADR to ratify an existing ADR** — just flip the existing ADR's status from proposed → accepted. Don't create a separate ADR ("acceptance-ADR" anti-pattern)
7. **Operational procedures / runbooks** — separate runbook documents

> Detailed anti-patterns + generalized violation cases: [references/anti-patterns.md](references/anti-patterns.md)

## During Writing — Gate 2: Template Selection

### Comparison and Selection Guide for the 3 Templates

| Criterion | Nygard (original) | MADR 4.0 | Y-statement |
|------|---------------|----------|-------------|
| Released | 2011 | 2024-09-17 | 2012 (SATURN) / Zimmermann |
| Length | 1–2 pages | 1–3 pages (full) / half page (minimal) | 1 sentence (+ extension) |
| Strengths | Simple, universal | Structured, rich metadata, linter support | Conversational, decision-focused |
| Weaknesses | Weak alternative analysis | Frontmatter learning curve | Rationale can be thin |
| **Default recommendation** | ★★★ (most common baseline) | ★★ (when comparing alternatives matters) | ★ (Advice Process draft) |

**Selection algorithm** (concise):

```
if comparing multiple alternatives is core → MADR 4.0
elif simple "just record the decision" → Nygard
elif Advice Process route + follow-up ADR planned → Y-statement (draft)
else → Nygard (default)
```

> Template bodies: [references/templates/nygard.md](references/templates/nygard.md) /
> [references/templates/madr.md](references/templates/madr.md) /
> [references/templates/y-statement.md](references/templates/y-statement.md)

### Standard Frontmatter Convention (minimum common set)

```yaml
---
id: ADR-NNN                  # 3-digit zero-padded
title: ADR-NNN <title>
status: proposed             # proposed → accepted → superseded|deprecated (or rejected)
date: YYYY-MM-DD             # proposal date (ISO 8601)
accepted_date: YYYY-MM-DD    # added on accepted transition
deciders: <name / role>      # final approver(s)
supersedes: []               # array of older ADRs replaced by this one
relates_to: []               # related ADRs (references that are not supersede)
---
```

Optional fields like `uuid`, `tags`, `review_due`, `consulted`, `informed`
can be added per project convention — details: [references/project-integration.md](references/project-integration.md).

**ID Issuance Rules**:
- Filename: `ADR-NNN-<kebab-case-title>.md` (3-digit zero-padded)
- No number gaps from re-use: never reuse a number when retiring an ADR; keep rejected / deprecated files
- Avoid duplicate topics: grep filename / title before drafting a new one

## After Writing — Gate 3: Anti-Pattern Review

The completed ADR must pass the **11 anti-patterns + 7 fallacies checklist**.
**Two or more violations → block creation**.

### 11 Anti-Patterns (summary)

**Subjectivity family**:
- A1 **Fairy Tale** — only lists upsides (downsides omitted)
- A2 **Sales Pitch** — marketing tone + exaggeration
- A3 **Free Lunch Coupon** — ignores long-term consequences
- A4 **Dummy Alternative** — only obviously inferior alternatives are presented

**Time & Context family**:
- A5 **Sprint** (myopia) — only considers 2–3 iterations
- A6 **Tunnel Vision** — operations / maintenance perspective absent
- A7 **Maze** — title and content do not match

**Size family**:
- A8 **Blueprint/Policy** — imperative / cookbook style (not journalistic)
- A9 **Mega-ADR** — too much implementation detail + diagrams + code
- A10 **Novel/Epic** — entire SAD compressed into a single ADR

**Deception family**:
- A11 **Magic Tricks** — fake urgency / pseudo-quantitative scoring

### 7 Decision-Making Fallacies (summary)

- F1 Blind Flight — skips requirements analysis
- F2 Following the Crowd — blind imitation
- F3 Anecdotal Evidence — single-case rationale
- F4 Blending Whole and Part — whole/part confusion
- F5 Abstraction Aversion — concept and tech choice mixed
- F6 Golden Hammer — one-size-fits-all misuse
- F7 Time Dimension Ignored — lifecycle ignored

> Detailed definitions + detection methods + generic symptom examples: [references/anti-patterns.md](references/anti-patterns.md)

### Automated Validation Checklist (grep-based)

```bash
# $ADR_DIR — the project's ADR directory (e.g. docs/architecture, docs/adr)
ADR_DIR=${ADR_DIR:-docs/architecture}

# A1 Fairy Tale detection — no "downside/risk/drawback/trade-off" in Consequences section
grep -A 20 "^## Consequences" <ADR.md> | grep -iE "downside|risk|drawback|trade-off|negative" || echo "WARN: A1 Fairy Tale suspected"

# A9 Mega-ADR detection — file > 400 lines OR 3+ diagrams
[ $(wc -l < <ADR.md>) -gt 400 ] && echo "WARN: A9 Mega-ADR suspected"
[ $(grep -c '```' <ADR.md>) -gt 6 ] && echo "WARN: A9 Mega-ADR — more than 3 code-block pairs"

# Duplicate detection — similar filenames
ls "$ADR_DIR"/ADR-*.md | awk -F- '{for(i=3;i<=NF;i++) printf "%s-",$i; print ""}' | sort | uniq -d

# Number gap detection
ls "$ADR_DIR"/ADR-*.md | grep -oE 'ADR-[0-9]+' | sort -u | awk -F- '{print $2+0}' | awk 'NR>1 && $1 != prev+1 {print "HOLE between ADR-"prev" and ADR-"$1} {prev=$1}'
```

## Lifecycle — State Transition Convention

```
       proposed ──(approved)──▶ accepted ──(replaced by new ADR)──▶ superseded
          │                        │
          └──(rejected)──▶ rejected └──(context change)──▶ deprecated
```

**Transition Rules**:

| Transition | Condition | Changes |
|------|------|----------|
| proposed → accepted | Decider approves | `status: accepted` + add `accepted_date` |
| proposed → rejected | Decider rejects | `status: rejected` + add `rejected_reason` (do not delete) |
| accepted → superseded | New ADR replaces this one | New ADR's `supersedes: [ADR-NNN]` + old ADR's `superseded_by: ADR-MMM` + `status: superseded` |
| accepted → deprecated | Context change (product wind-down, tech EOL) | `status: deprecated` + `deprecated_date` + `deprecated_reason` |

**Immutability rules**:
- **Do not edit the body of an Accepted ADR** — supersede with a new ADR if changes are needed
- A status transition may modify the frontmatter only, plus a single Preamble line
- **Never delete** — superseded / deprecated / rejected are all preserved as history

> Detailed transition scenarios + bidirectional link verification scripts: [references/lifecycle.md](references/lifecycle.md)

## Scope — ADR vs Other Document Types

| Document type | Purpose | Tense | Scope | Storage location (convention) |
|----------|------|-----|------|----------------|
| **ADR** | **Record** of decisions already made | past | architectural | `docs/architecture/` or `docs/adr/` |
| RFC | **Proposal** for input gathering | future | feature-level proposal | `docs/rfcs/` |
| Design Doc / TDR | Implementation **plan** | future | implementation detail | `docs/design/` |
| Knowledge Base card | Lessons · gotchas · insights | past | personal / team learning | `docs/knowledge/` or team wiki |
| Task / Ticket body | Single-task spec | present | scope of 1 task | issue tracker / `tasks/` |
| Runbook | Operational procedure | imperative | repeated execution | `docs/runbooks/` |

**Workflow Relationships**:

```
 RFC (proposal · discussion) ──▶ Decision Meeting ──▶ ADR (record)
                                                       │
                                                       ▼
                                               Design Doc (impl plan)
                                                       │
                                                       ▼
                                                Task (1 commit)
                                                       │
                                                       ▼
                                          KB / Learning card (lessons)
```

> Detailed decision tree: [references/scope-adr-vs-others.md](references/scope-adr-vs-others.md)

## Operating Modes (skill invocation)

### Mode: `check` (default)

```
/project-adr check <ADR file path>
```

Runs Gates 1–3 in full. Outputs pass/block verdict + violation list.

### Mode: `new`

```
/project-adr new "<ADR title>"
```

1. Run the Gate 1 Decision Threshold interview (5 questions)
2. On pass, prompt for Template selection
3. Generate frontmatter + section skeleton from user answers
4. Allocate a new number (max ID + 1 from grep against the ADR directory)
5. Show the file creation location (write happens only after user approval)

### Mode: `review`

```
/project-adr review <ADR directory path>
```

Audit the entire ADR directory:
- Detect number gaps / duplicate filenames
- Warn about long-stale `proposed` (>30 days)
- Verify supersede integrity (bidirectional links)
- List Mega-ADR candidates (>400 lines)

### Mode: `supersede`

```
/project-adr supersede ADR-NNN --with "<one-line summary of new decision>"
```

1. Propose changing the existing ADR-NNN status from accepted → superseded
2. Auto-fill the new ADR's supersedes field
3. Guide bidirectional reference consistency in frontmatter

## Project Integration Conventions

### Task / Iteration Linkage

- Recommend type: `design` or `docs` for an ADR-authoring task
- Between ADR proposed → accepted, a **separate task** is recommended (visualizes pending approval)
- The accepted transition happens **only when the decider's explicit approval message exists**

### Commit Message Convention (adjust to project conventions)

Conventional Commits style examples:

```
docs(adr): ADR-NNN proposed — <title>                   # new proposal
docs(adr): ADR-NNN status proposed → accepted — <reason>  # state transition
docs(adr): ADR-NNN → ADR-MMM supersede                  # supersede
docs(adr): ADR-NNN deprecated — <reason>                 # deprecated
```

Follow the project's task ID prefix convention if required.

### Feature Catalog Linkage (when present)

- If an ADR finalizes new Feature boundaries, **add an entry to the feature catalog**
- A redesign ADR for an existing Feature should mark the catalog entry `status: revised`

### Relationship with Knowledge Base / Learning Cards (important)

- **ADRs record decisions**; **KB / learning cards record lessons · gotchas · insights**
- A topic may have both an ADR and a KB — ADR for why/decision, KB for how-to-avoid-pitfall
- Gotchas discovered while writing an ADR should **not be mixed into the ADR body** — file them as separate KB cards

> Detailed integration guide: [references/project-integration.md](references/project-integration.md)

## Quality Bar

| Criterion | Minimum | Recommended |
|------|-----|------|
| Gate 1 Yes count | 3/5 | 5/5 |
| Alternatives analyzed (MADR) | 2 | 3–5 |
| Consequences — positive + negative | 1 each | 3 each |
| File line count | 80–300 target | ≤ 400 (A9 Mega-ADR block threshold) |
| Source / rationale links | 1 | 3 |
| Anti-pattern violations | ≤ 1 | 0 |

## Edge Cases

- **Decision already made but no ADR exists**: writing a retrospective ADR is fine. Note in the Context section "decided N years ago, recorded later" and use measured data in Consequences
- **Multiple ADRs in flight**: accept them one at a time — connect mutually dependent ones via `relates_to`
- **Team disagreement**: route through Architecture Advice Process → Y-statement draft → consensus → promote to MADR
- **AI tries to auto-author an ADR**: **forbidden**. The skill only diagnoses Gates 1–3 and offers templates. The final Decision sentence must be authored by the user

## Reference Files

### References
- **[references/when-to-adr.md](references/when-to-adr.md)** — Gate 1 Decision Threshold 5 questions in detail + alternatives when not needed
- **[references/templates/nygard.md](references/templates/nygard.md)** — Nygard original template (Context/Decision/Consequences)
- **[references/templates/madr.md](references/templates/madr.md)** — MADR 4.0 full / minimal templates
- **[references/templates/y-statement.md](references/templates/y-statement.md)** — Y-statement 1 sentence + extension sections
- **[references/anti-patterns.md](references/anti-patterns.md)** — 11 anti-patterns + 7 fallacies in detail + generic symptom examples
- **[references/lifecycle.md](references/lifecycle.md)** — Transition scenarios + frontmatter change conventions + bidirectional supersede linking
- **[references/scope-adr-vs-others.md](references/scope-adr-vs-others.md)** — Decision tree for ADR / RFC / Design Doc / KB / Task / runbook
- **[references/project-integration.md](references/project-integration.md)** — Frontmatter / filename / commit / Task / CI gate integration conventions

### Examples
- **[examples/good-adr.md](examples/good-adr.md)** — Exemplary ADR (passes all of Gates 1–3)
- **[examples/bad-adr.md](examples/bad-adr.md)** — Cautionary tale containing 7 anti-patterns + corrective guide

## Related Skills (when present in the project)

- Knowledge base / learning skills — capture lessons / gotchas as KB cards, not ADRs
- Retrospective / sprint skills — iteration-level retrospectives
- Document review skills — generic doc rule validation after ADR drafting
- Cross-document check skills — link consistency between ADRs
- Feature catalog skills — sync the catalog when an ADR shifts Feature boundaries

## Provenance

This skill's standards rest on the following references — standards, papers, and practitioner guides:

1. **Michael Nygard (2011)** — *Documenting Architecture Decisions* (the original)
2. **Zdun et al. (2013)** — *Sustainable Architectural Decisions* (Y-statement)
3. **MADR 4.0 (2024-09-17)** — Markdown Any Decision Records
4. **Olaf Zimmermann (2023, 2025)** — *How (not) to create ADRs* + *7 ADM Fallacies*
5. **AWS Prescriptive Guidance** — ADR process + best practices
6. **Microsoft Azure Well-Architected Framework** — Maintain an ADR
7. **ThoughtWorks** — Lightweight technology governance + Architecture Advice Process
8. **Anthropic Claude Code Skills v2.0** — frontmatter compatibility for this SKILL.md

Detailed citations + URLs: [references/provenance.md](references/provenance.md)
