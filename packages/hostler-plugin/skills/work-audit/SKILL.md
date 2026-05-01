---
name: work-audit
description: Audit Sprint/Task implementation completeness — D1-D7 implementation, build/test, W1-W7 workflow compliance, and antipattern detection. Read-only diagnostic. Use this skill whenever the user asks about sprint/task completion checks, "is this Task done?", workflow compliance, audit reports, or wants to verify a sprint before completing it — and run it automatically as Phase 3 of `sprint:complete`. Do NOT use for code review (use `code-review`) or for documentation review (use `doc-review`).
compatibility:
  tools: [Read, Write, Edit, Bash, Glob, Grep, Agent]
argument-hint: "[--session|--sprint|--repo] [--since=N days] [path]"
paths: ["works/**/*", "docs/**/*"]
---

# Work Audit — Implementation Completeness + Workflow Compliance Check

Read-only audit of Sprint/Task implementation completeness (D1-D7) and hostler workflow compliance (W1-W7). Audit is diagnosis, not treatment.

> **work-audit itself does not invoke doc-review or code-review.**
> It runs as a standalone skill or as an independent phase of the
> sprint:complete 10-Phase Cascade. Inside sprint:complete, the order is
> Phase 1 (doc-review) → Phase 2 (code-review) → **Phase 3 (work-audit)**.
> For the full ordering, see the "10-Phase" section of
> `skills/sprint-management/SKILL.md`.

## Table of Contents

1. Core Principles
2. Scope
3. Three-Phase Inspection Structure
4. Phase Details
5. Report Output
6. Suggested Fixes
7. Workflow
8. Reference Files

---

## Core Principles

> **audit = read-only inspection.** Never modify code, run lint auto-fix (`--fix`), or change files.
> Findings are emitted as a report; the user performs fixes as a separate task.

### Pre-flight integrity check

It is recommended to verify DB integrity before running work-audit.
If `hstl-oss context` reports any warnings, run `hstl-oss backlog sync`
first and then run the audit.

## Scope

With no argument, `--sprint` is the default — Sprint quality gating is the primary use case.

| Option | Behavior |
|--------|----------|
| `--sprint` (default) | Inspect implementation completeness + workflow over the active Sprint |
| `--session` | Limit to changes in the current session |
| `--repo` | Inspect the entire repository |
| `--since=N` | Limit to commits in the last N days (composable with the others) |

Scope determines whether the audit covers the Sprint, the session, or the whole repository.

---

## Three-Phase Inspection Structure

```
work-audit --sprint
  ├─ Phase 1: Implementation completeness (built-in)   ← D1-D7
  ├─ Phase 2: Workflow (built-in)                      ← W1-W7
  └─ Phase 3: Antipattern audit                         ← Q1-Q25 (Guide Appendix B)
```

work-audit covers **implementation completeness + workflow compliance + antipattern audit**.
doc-review and code-review are not its responsibility:
- **Standalone invocation**: the user runs `/doc-review` and `/code-review` separately
- **Inside sprint:complete**: the sprint:complete 10-Phase Cascade runs Phase 1 (doc-review) →
  Phase 2 (code-review) first, then invokes work-audit directly as Phase 3

> The structure where work-audit internally delegated to doc-review/code-review has been removed.
> This avoids orchestration duplication between sprint:complete and work-audit.
> For the 10-phase order, see the "10-Phase" section of `skills/sprint-management/SKILL.md`.

**Q axis — antipattern audit**: automatically detects and reports the 25
antipatterns from Guide Appendix B.1 (10 Skill antipatterns) + Appendix B.2
(15 CLI subcommand antipatterns).
For detection methods and recommended fixes, see `references/antipattern-check.md`.

**Core principle**: Q-axis checks are **read-only**. Detect only; fixes are handled by other skills or the user after confirmation.

---

## Phase Details

### Phase 1: Implementation Completeness — Task plan vs actual implementation (built-in)

Verify the actual implementation against each Task file's completion criteria and results section.

| # | Check | How | Grade |
|---|-------|-----|-------|
| D1 | **"Changed files" exist in results** | Verify the file paths recorded in the Task actually exist | BLOCK |
| D2 | **Completion-criteria checkboxes** | Compute the `- [x]` ratio for done tasks | WARN |
| D3 | **Sprint Task count match** | SPRINT.md Task table row count == file count under `tasks/`. **Pattern**: `grep -c "^| T[0-9]"` (cell-start only; `^|.*T[0-9]` causes false positives on TR codes — do NOT use). | WARN |
| D4 | **Sprint Task status match** | Status text in SPRINT.md == status in each Task's frontmatter | WARN |
| D5 | **Tests pass** | Run `dotnet test`, `pytest`, `npm test`, etc. | BLOCK |
| D6 | **Build succeeds** | Run `dotnet build`, `ruff check`, etc. (read-only — `--fix` is forbidden) | WARN |
| D7 | **UX → implementation mapping refreshed** | If a UI-changing Task exists, verify the UX mapping doc was updated | WARN |

### Phase 2: hostler workflow compliance (built-in)

| # | Check | How | Grade |
|---|-------|-----|-------|
| W1 | **task:start was invoked** | The git history shows the in-progress transition for done tasks | WARN |
| W2 | **Results section exists** | done tasks have a `## Result` section + a completion date | WARN |
| W3 | **Commit-message format** | Compliance rate with the `{type}: {description}` pattern | INFO |
| W4 | **Sprint completion handling** | Sprints whose Tasks are all done have moved to `completed/` | WARN |
| W5 | **CURRENT-FOCUS up to date** | Sprint progress matches the actual Task statuses | WARN |
| W6 | **Retrospective recorded** | The most recently completed Sprint has a `## Retro` section with Keep/Problem/Try content. **Pattern**: capture from the retro heading to end-of-file with `sed -n '/^## Retro/,$p'` then count item lines with `grep -c "^- "`. Do NOT use `sed -n '/^## Retro/,/^$/p'` — it stops at the first blank line. If missing, suggest running `/retro`. | WARN |
| W7 | **Worklog commit coverage** | Verify yesterday's commits are recorded in the worklog | WARN |

### Phase 3: Antipattern Audit (Q axis)

Detect the 25 antipatterns from Guide Appendix B. For the detailed checklist
and detection methods, see `references/antipattern-check.md`.

**Q1-Q10 (Skill antipatterns)** — uses the output of `{project-root}/scripts/measure-skill-quality.py` (manual review when the script is missing):
- Q1 vague description
- Q2 missing negative condition
- Q3 description longer than 250 characters
- Q4 SKILL.md longer than 500 lines
- Q5 hardcoded lookup tables
- Q6 inspection skill performs auto-fix
- Q7 10+ ALWAYS/NEVER occurrences
- Q8 trigger-eval.json missing
- Q9 project-specific language
- Q10 model-recommended verbs ("can perform")

**Q11-Q25 (CLI subcommand antipatterns, archive)** — manual review (measurement script removed):
- Q11 1-sentence description
- Q12 enum-like options listed as a string (no enum field)
- Q13 missing Annotations
- Q14 inconsistent error format
- Q15 missing error_category
- Q16 missing recovery_hint
- Q17 missing L1 Server Instructions
- Q18 hardcoded section heading in the validator
- Q19 cross-cutting concern injected per handler
- Q20 direct file editing allowed
- Q21 advertise = dispatch identical list
- Q22 policy weakened on validation failure
- Q23 SDK-upgrade regression test skipped
- Q24 ID counter not advanced
- Q25 worktree absolute path stored in DB

**Q26 (Legacy duplicate file detection)** — `scripts/audit-legacy-duplicates.sh`:
- Detects schema/config files whose basename appears in 2+ locations within the repo
- Target globs: `cli/**/schemas/*.json`, `**/*.config.yaml`,
  `**/harness*.{json,yaml}`, `**/briefing*.yaml`, etc.
- Allowlist reference: `references/legacy-duplicate-allowlist.json` (excludes
  intentional shadow copies)
- Goal: prevent recurrence of past schema/config duplicate-entry-point cases
- Result: WARN (manual review required) — never deletes automatically (Q-axis is read-only)

**Detection automation**: reuse existing measurement scripts to avoid duplicate
implementation. Q1-Q16 can be automated by script, Q17-Q25 require partial
manual review, and Q26 is fully scripted.

### Existing debt outside Sprint scope vs newly introduced regression

When the D/W/Q checks of work-audit return FAIL, you must distinguish between
**a regression newly introduced inside the Sprint scope** and **pre-existing
technical debt outside the Sprint scope**. Treating everything as BLOCK
repeatedly delays sprint completion; ignoring everything lets debt
accumulate.

**Decision checklist** (for each FAIL):

1. Did this FAIL exist before the Sprint's first commit?
   → Use `git log --oneline <Sprint start SHA>..HEAD -- <path>` to trace back.
   If it existed before Sprint start, it is **existing debt**.
2. Is fixing the issue explicitly in the Sprint scope (the SPRINT.md Task list)?
   → If yes, it is a **new regression** (treat as failure).
3. Does fixing it require one or more Tasks?
   → If yes, **split it out as a follow-up Task** and let the current Sprint pass.

**Standard report template for separately recorded items** (added by the audit runner):

```markdown
### Existing technical debt outside Sprint scope (separated)

- D1 violation: {path} — origin Sprint/commit {SHA}, follow-up Task: T{NNN}
- W6 violation: {file} — origin Sprint/commit {SHA}, follow-up Task: T{NNN}
- Q6 violation: {skill} — origin Sprint/commit {SHA}, follow-up Task: T{NNN}
```

**retro coupling**: when existing debt is identified, record it under the
Sprint retro Problem section as "Existing debt found: N items", and connect
a Try item that says "Spawn follow-up Task T???". sprint:complete Phase 9
(deriving follow-up Tasks) must turn that Try into an actual Task
registration so accumulated debt is managed.

**Principles**:
- work-audit provides the **decision criteria** but **must not perform automatic separation** — read-only
- Separation decisions are made by the audit report author (AI or user)
- FAILs separated as "existing debt" are excluded from the Sprint quality
  grade calculation (subtract them from the BLOCK/WARN count when computing
  the combined Phase 1+2+3 grade and aggregate them in a separate section)

### Quality grading (combined Phase 1+2+3)

Combine the results of Phase 1 (implementation completeness) and Phase 2 (workflow) to grade the Sprint.

| Grade | Criteria |
|-------|----------|
| **A** | BLOCK 0, WARN ≤ 2, all tests pass, commit format ≥ 90% |
| **B** | BLOCK 0, WARN ≤ 5 |
| **C** | BLOCK 0, WARN > 5 |
| **D** | BLOCK ≥ 1 |

---

## Report Output

```
═══════════════════════════════════════════════
  WORK AUDIT REPORT — {date} (scope: --sprint)
═══════════════════════════════════════════════

Phase 1: Implementation completeness (built-in)
  ✅ D1: changed files exist
  ✅ D3: Sprint Task count matches
  ✅ D5: build succeeded + tests passed

Phase 2: Workflow (built-in)
  ✅ W3: 95% of commit messages follow the format

═══════════════════════════════════════════════
  Combined grade: A | BLOCK: 0 | WARN: 1 | PASS: 10
═══════════════════════════════════════════════

> doc-review and code-review results are not included in the work-audit report.
> When called standalone, run them separately; in sprint:complete they run as their own phases.
```

## Suggested Fixes

When the audit reports WARN/BLOCK, the report ends with concrete remediation commands.
The audit itself does not run them. Run them only when the user replies "please fix".

### Fix-suggestion principles

| Principle | Description |
|-----------|-------------|
| **Concrete commands** | Include file paths, sed commands, agent-delegation phrasing |
| **Preview first** | For code edits, suggest `--diff` preview commands first |
| **Confirm before applying** | Ask "Apply fixes?" first |
| **Selective fixes** | Allow per-item selection |

### Fixes that delegate to skills (mandatory)

| WARN item | Delegated skill | Reason |
|-----------|----------------|--------|
| **W6 missing retrospective** | `/hstl-oss:retro` | retro performs KPT extraction + delegates to learned to record lessons |

> When fixing W6, do not write the retrospective into SPRINT.md directly. Always invoke the retro skill.

### Items that cannot be fixed by the audit

- Retroactively adding results sections / checkboxes (the implementer must write them)
- Architectural decisions (ADR)
- Rewriting commit history

## Workflow

### Bash execution rules (mandatory)

Phase checks may run in parallel, but every Bash command must terminate with exit 0.

**Defensive bash style**:

1. Do not use `set -e`
2. Defend against empty variables in arithmetic: use `${var:-0}`
3. Absorb `grep -c` results: use `|| true` (do NOT use `|| echo 0` — produces "0\n0" arithmetic errors)
4. End every Bash command with `echo "done"`

### Execution order

1. Determine scope (parse args → default `--sprint`)
2. Phase 1: built-in implementation completeness checks (D1-D7)
3. Phase 2: built-in workflow checks (W1-W7)
4. Compute quality grade and emit report
5. If WARN/BLOCK exists, suggest fixes and ask the user to confirm

> work-audit does not call doc-review/code-review.
> The user runs them separately when needed, or sprint:complete runs them as separate phases.

## Reference Files

- **`references/deliverable-checklist.md`** — detailed implementation-completeness checks (D1-D7)
- **`references/workflow-checklist.md`** — hostler workflow compliance checks (W1-W7)
- **`references/changelog.md`** — change log (maintainers only)

## Use of Auto-Verification

In the Task completion verification flow, the Go-deterministic items
(`build_passed`, `tests_passed`, `lint_passed`) are batched automatically by
`hstl-oss harness auto-check`. After work-audit checks D1-D7 implementation
issues, the recommended order is:

```bash
hstl-oss-oss harness auto-check task <ID>            # 3 Go-deterministic items
hstl-oss-oss harness check task <ID> criteria_checked --evidence "..."
hstl-oss-oss harness check task <ID> code_review --evidence "..."
hstl-oss-oss task complete <ID> --with-ceremony
```

Design rationale: the user's project docs (auto-check safety).

## Per-Project Extensions (auto-detected)

At runtime, this skill auto-detects `.hostler/project-config.md` in the project root.
If that file exists and contains a `## Per-skill extensions` → `### work-audit` section,
the project-specific rules are applied **in addition to** the shared rules.

- Detection path: `{project_root}/.hostler/project-config.md`
- If absent, only the shared rules are applied (backward compatible)
