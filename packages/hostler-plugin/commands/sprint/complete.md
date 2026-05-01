---
description: Complete a Sprint — performance briefing + 10-Phase Cascade + KB registration + reminders. Use when the user says "complete the sprint", "wrap up the sprint", or finishes the active Sprint cycle.
argument-hint: [sprint-id]
allowed-tools: Bash(hstl-oss:*), Bash(git:checkout), Bash(git:branch), Bash(git:merge), Bash(git:push), Bash(git:commit), Bash(git:status), Bash(git:diff), Bash(go:test), Read, Write, Edit, Glob, Grep
---

# Complete Sprint: $ARGUMENTS

> **Command-first entry point**: This Command is the official entry for Sprint completion.
> When the user requests Sprint completion, do not call the CLI directly — first call this
> Command to run the 10-Phase Cascade, performance briefing, and KB registration procedure,
> then call the CLI from inside the Command.
>
> **10-Phase SSOT**: The Phase list is single-sourced in `.hostler/project-config.yaml`'s
> `sprint_ceremony.complete` section. The table in this Command must stay in sync with that config.
>
> **Branch policy**: Whether and when to merge `dev -> main` at Sprint completion follows the shared
> template `skills/branch-workflow/references/branch-policy.md` (default: do not auto-promote each
> Sprint, manual main merge). A project's own SSOT takes precedence.

> **Command First STRICT**: This Command is the official entry point for Sprint completion.
> Calling `hstl-oss sprint complete <ID>` directly via Bash auto-records `ceremony.bypass` /
> `*.retried` events in the audit log and counts as a violation in Phase 5 retro. The
> 10-Phase Cascade must always go through this Command. Policy SSOT: `CLAUDE.md`
> §Command First mapping table.

Completes a Sprint via the CLI.

## No automatic registration

> **Behavioral rule — KB/Task registration in Phases 6/9 must always wait for user selection.**
>
> The AI **must never** call `hstl-oss kb create` / `hstl-oss task create` immediately while
> performing Phases 6/9. Skipping the user-confirmation step has caused unintended KB cards
> and Tasks to be left in the Sprint record, generating cleanup cost later.
>
> **Required order (no exceptions)**:
>
> 1. Phase 6: record only the **candidate table** in SPRINT.md (Actual Selection column = `TODO`)
> 2. Phase 9: record only the **candidate table** in SPRINT.md (Actual Selection column = `TODO`)
> 3. Run all phases through Phase 10 (skill update check)
> 4. **Print the final user-confirmation prompt** -> wait for response
> 5. Based on the response, call `hstl-oss kb create` / `hstl-oss task create` (registration happens here)
> 6. Update the Actual Selection column in SPRINT.md (`Registered (ID)` / `Excluded (reason)` / `Merged (ID)`)
> 7. `hstl-oss sprint complete` -> stamp

## CLI usage

```
Bash("hstl-oss sprint complete $ARGUMENTS --with-ceremony")
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss sprint complete <id> --with-ceremony | jq '.ceremony.harness'
```

Parse `ceremony.harness` and `ceremony.kb_candidates` from the CLI JSON response and
preserve the existing completion-ceremony rendering.

Returns:
- Success: `{ "sprint_id": "...", "status": "completed", "ceremony": { ... } }`
- Gate BLOCKED: exit code 3 + BLOCKED items printed

## Automatic side effects

`hstl-oss sprint complete` atomically updates the following beyond the 10-Phase Cascade —
the user/AI does not touch them separately:

- **`works/sprints/active/CURRENT-FOCUS.md`**: when the active Sprint ends,
  it is auto-reset to idle (`UpdateCurrentFocus`).
- **`works/tasks/BACKLOG.md`**: state changes (done) of the Sprint's Tasks
  are reflected automatically.
- Consistency check: `hstl-oss backlog sync --dry-run`. Do not edit with Write/Edit.

## Output format

`sprint:complete` is a **quality ceremony performed alongside the user**. Run all Phases
in an unbroken sequence, and ask for user confirmation **only once at the end**.

### Phase 0: Performance briefing

```markdown
=============================================
  Sprint {id} performance briefing
=============================================

  Period: {started} ~ {completed} ({N} days)
  Tasks: {done}/{total} done

  Goal achievement:
  - {goal item 1} — basis for completion
  - {goal item 2} — basis for completion
  - {goal item 3} — partially done (reason)

  Key artifacts:
  1. {artifact 1} (file path or commit)
  2. {artifact 2}

  Numbers:
  - Commits: {N}
  - Code changes: +{N} / -{N} lines
  - Build: {result}
  - Tests: {pass}/{total}

  Acknowledged debt / follow-ups:
  - {debt 1} -> {follow-up Task or Sprint}

=============================================
```

If reminders exist, print them as a blockquote right after the briefing.

### Cascade Gate — sequential 10-Phase execution

Each Phase prints `=== Phase N/10: ... ===` at start/end. The Phase list and order
are sourced from `.hostler/project-config.yaml`'s `sprint_ceremony.complete` section.

<!-- SPRINT_PHASE_TABLE:BEGIN (SSOT: cli/pkg/db/schemas/harness_defaults.json sprint:default) -->
| Phase | Content | Block condition |
|-------|------|---------|
| 1 | Phase 1: doc-review --sprint | When BLOCK items remain |
| 2 | Phase 2: code-review --sprint | When Critical/High remain |
| 3 | Phase 3: work-audit (implementation + workflow) | — |
| 4 | Phase 4: doc-cross-check | When design changed (conditional) |
| 5 | Phase 5: retro (KPT) | Cannot complete without retro |
| 6 | Phase 6: learned (KB) | Even 0 cards must be acknowledged |
| 7 | Phase 7: Dev deployment verification | When Dev deploy fails |
| 8 | Phase 8: binary build + install | Conditional on production Sprint |
| 9 | Phase 9: follow-up Task derivation | New work from retro must be added to BACKLOG, **required** |
| 10 | Phase 10: skill update check | Skill modification or `inbox_submit`, **required** |
| 11 | Phase 11: Deprecation Cleanup Gate (archive residue check) | When archive changes are detected (conditional) |
| 12 | Phase 12: dev/main gap check | WARN 24/BLOCK 30 — `hstl-oss rules exec precommit.branch.sync` |
<!-- SPRINT_PHASE_TABLE:END -->

> **SSOT**: The "Content" column above is single-sourced from
> `cli/pkg/db/schemas/harness_defaults.json` `sprint:default[].name`. Drift is
> guarded by `scripts/check-sprint-phase-drift.sh` in pre-commit.

### Phase 5: retro visibility

```markdown
=== Phase 5/10: retro (KPT) ===

Sprint {id} analysis of {N} Tasks:

[Keep] (N items)
  K1. {title} — {basis}

[Problem] (N items)
  P1. {title} — {impact} — {when found}

[Try] (N items)
  T1. {title} — {concrete approach} — {when to apply}
```

### Phase 6: learned visibility

**No automatic registration**: in this Phase, only record candidates and recommendations
in the SPRINT.md `## Lessons` section. `hstl-oss kb create` is called **only after the
final user confirmation**.

```markdown
=== Phase 6/10: learned (KB) ===

Lesson candidates from Sprint {id} (record in SPRINT.md `## Lessons` table):

| # | Candidate | Category | Source Task | Recommendation | Actual Selection | Basis |
|---|------|---------|----------|-----|---------|------|
| 1 | {lesson title} | architecture | T123 | Recommended | TODO (after user confirmation) | New principle established |
| 2 | {lesson title} | mistakes | T124 | Optional | TODO (after user confirmation) | doc-review missed |
| 3 | {lesson title} | operations | T125 | Excluded | TODO (after user confirmation) | Already in KB |

**Actual Selection column convention**:
- `Registered (M###)` — KB card ID
- `Excluded (reason ≥10 chars)` — concrete reason for not selecting
- `Merged (existing M###)` — absorbed into an existing card
- Even 0-candidate cases require one row "N/A | — | — | — | Excluded (reason) | basis"

Selection criteria:
- Mistakes likely to recur -> Recommended
- New principles/patterns introduced to the project -> Recommended
- Already covered by existing KB -> Excluded
- One-off issues specific to a single Sprint -> Excluded
```

### Phase 9: Follow-up Task derivation — three-source unified scan

**Core principle**: KPT collection and Task registration are separate stages, and the
ceremony must connect them. If Try items recorded in Phase 5 are not registered as
backlog Tasks, a "collect ≠ act" gap forms and the same improvement opportunities
are repeatedly lost.

Scanning only KPT Try misses follow-ups recorded in Task result-section "Lesson"
blocks or "Out-of-scope -> Future Sprints" blocks. Scan three sources together:

| Source | Scan target | Pattern |
|------|----------|------|
| **KPT Try** | Try items collected in Phase 5 retro | Mapping table existing behavior |
| **Lesson keywords** | Each Task's `### Lessons` block in the result section | Regex: `separate\s*(refactor\s*)?Task\|outside\s*current\s*Task\s*scope\|follow-up\s*Task\|future\s*Task\s*candidate` |
| **Scope limits** | Each Task's `### Future Sprints` block | All bullet items (exclusion declaration = follow-up candidate) |

**Phase 9 scan procedure** (AI executes):

1. Collect all Sprint Task files via `Glob("works/sprints/active/{sprint-id}/tasks/T*.md")`
2. For each file, read `## Result` -> `### Lessons` block -> match keyword regex
3. For each file, read `## Scope Limits` -> `### Future Sprints` block -> collect bullets
4. Combine KPT Try + lesson keywords + scope limits into a unified mapping table
5. **Backlog duplicate check**: extract 3 main nouns from each candidate title ->
   compare with `hstl-oss task list -o json | jq` against existing backlog Task titles ->
   suspected duplicates: mark `Possible duplicate: T### <title>` in the basis column.
   If the user chooses "Merge", record `Merged (existing T###)` instead of `Registered (T###)`.
6. Present the table to the user with a confirmation prompt

**Why deduplicate**: similar backlog Tasks have repeatedly been re-issued even when
candidates already existed. The pre-check prevents duplicate issuance.

**No automatic registration**: in this Phase, only record candidates and recommendations
in the SPRINT.md `## Follow-up Tasks` section. `hstl-oss task create` is called **only
after the final user confirmation**.

**Required Phase 9 briefing**:

```markdown
=== Phase 9/10: Follow-up Task derivation (three-source unified scan) ===

Sprint {id} follow-up Task candidate mapping (record in SPRINT.md `## Follow-up Tasks` table):

| # | Proposal | type | est | pri | Recommendation | Actual Selection | Basis (source / origin Task) |
|---|------|------|-----|-----|-----|---------|---------------------|
| 1 | {concrete Task title} | refactor | S | p2 | Recommended | TODO (after user confirmation) | KPT Try T1 |
| 2 | {concrete Task title} | docs | XS | p3 | Recommended | TODO (after user confirmation) | KPT Try T2 |
| 3 | {Task title} | chore | S | p3 | Recommended | TODO (after user confirmation) | Lesson |
| 4 | {Task title} | refactor | M | p3 | Optional | TODO (after user confirmation) | Scope limit |
| 5 | — | — | — | — | Excluded | TODO (after user confirmation) | KPT Try T3 — KB sufficient |

Summary: KPT Try N + Lessons M + Scope limits K = total candidates L
Recommended A / Optional B / Excluded C
```

**Actual Selection column convention**:
- `Registered (T###)` — newly issued Task ID
- `Excluded (reason ≥10 chars)` — concrete reason for not selecting
- `Merged (existing T###)` — absorbed into an existing Task
- Even 0-candidate cases require one row "N/A | — | — | — | — | Excluded (reason) | basis"

**Deduplication guidance**: when KPT Try and a Lesson share a topic, merge them into
one row but list both sources in the basis column (e.g. `basis: KPT Try T1 + Lesson`).

**Principles**:
- Each KPT Try item is recommended by default (unless the user explicitly says "skip registration")
- **Lessons / scope-limit sources are also recommended by default** — when the
  implementer wrote "needs a separate Task", registration intent is clear
- Vague Task candidates may be filed at P3/XS, with details fleshed out when the Task starts
- Foundational rule changes such as skill improvements may be excluded if covered by KB
- Always state the decision rationale in the basis column

### Final user confirmation (once after Phase 10) — pairs with no-auto-registration

> **Required order (no exceptions)**:
>
> 1. Complete Phases 1~10 (no registration CLI calls)
> 2. SPRINT.md `## Lessons` / `## Follow-up Tasks` tables: Actual Selection column = TODO
> 3. Print the confirmation prompt below -> wait for the user response (stop here)
> 4. Parse the response -> call `hstl-oss kb create` / `hstl-oss task create`
> 5. Update the **Actual Selection** column in SPRINT.md (`Registered (ID)` / `Excluded (reason)` / `Merged (ID)`)
> 6. Stamp via `hstl-oss sprint complete <id> --with-ceremony`

```markdown
=============================================
  Sprint {id} completion summary — pre-registration confirmation
=============================================

  Phase 0 performance briefing       OK
  Phase 1 doc-review                 OK / WARN {N} issues
  Phase 2 code-review                OK
  Phase 3 work-audit                 OK
  Phase 4 doc-cross-check            OK
  Phase 5 retro                      KPT K{N}/P{N}/T{N}
  Phase 6 learned                    {N} candidates (recorded, unregistered)
  Phase 7 build & test               OK go test -count=1 ./...
  Phase 8 binary build & install     OK cd cli && make install
  Phase 9 follow-up Task derivation  {N} candidates (KPT {A} + Lessons {B} + Scope {C}, recorded, unregistered)
  Phase 10 skill / docs update       OK (state explicitly when no changes)

  Pre-registration state. Pick from below to register.

  KB registration candidates (per SPRINT.md `## Lessons` table):
  1. {Lesson 1} (architecture) — Recommended
  2. {Lesson 2} (mistakes) — Optional

  Follow-up Task registration candidates (per SPRINT.md `## Follow-up Tasks` table):
  1. {proposal title} (refactor/S/p2) Recommended — KPT Try
  2. {proposal title} (docs/XS/p3) Recommended — Lesson
  3. {proposal title} (refactor/M/p3) Optional — Scope limit

  Input options:
  - "confirm" -> register all KB recommended + all Task recommended, auto-record exclusion reasons (default)
  - "KB all / Task all" -> register everything
  - "KB 1,3 / Task 1,2" -> partial selection (unselected items recorded as `Excluded (reason)`)
  - "KB none / Task none" -> register nothing (Actual Selection column requires `Excluded (reason ≥10 chars)`)
  - "edit KPT" -> revise KPT and re-print
  - no input -> same as "confirm" (default: register all recommended)
=============================================
```

**Execution order after response handling** (AI runs automatically):

1. `hstl-oss kb create ...` — create selected KB cards. **Collect returned KB IDs** (`registered_kb_ids`)
2. `hstl-oss task create ...` — create selected follow-up Tasks. **Collect returned Task IDs** (`registered_task_ids`)
3. **Read-verification immediately after registration** — verify each collected ID actually exists in the DB. Cases of "report only, no actual row" have accumulated due to multi-worktree drift / DB write failure / response-parsing errors / auto-registration omissions (KB `docs/07-knowledge/mistakes/governance.md` M003).
   ```bash
   # Single command for combined KB + Task verification (recommended)
   hstl-oss ceremony verify-registration --sprint-id sprint-XX \
     --kb-ids "M001,A002" --task-ids "T001,T002"
   # exit code 0 = passed (audit ceremony.verification.passed)
   # exit code 1 = blocked (audit ceremony.verification.failed + missing IDs as stdout JSON)
   ```
   Even one NOT_FOUND -> BLOCK + sprint stamp blocked. The AI must treat exit code 1 as fatal during response handling (do not proceed to stamp on failure).
4. Update the **Actual Selection** column in the SPRINT.md `## Lessons` / `## Follow-up Tasks` tables:
   - Registered -> `Registered (M###)` / `Registered (T###)`
   - Not registered -> `Excluded (10+ char reason)` or `Merged (existing ID)`
5. Call `hstl-oss sprint complete <id> --with-ceremony` — stamp + active->completed move

## Branch merge

After Sprint completion, merge the Sprint branch into dev:

```bash
git checkout dev
git merge sprint-{id} --no-ff -m "merge: sprint-{id} -> dev"
git push origin dev
git branch -d sprint-{id}  # delete local branch
```

If a prod release is needed, additionally merge `dev -> main`.

## Strict rules

- **Never skip the 10 Phases** — verify all Phase items via `hstl-oss harness get sprint <id>` (Phases 9/10 in particular; the legacy CEREMONY.md file is deprecated — `harness_items` is the SSOT)
- **No automatic registration in Phases 6/9** — call `hstl-oss kb create` / `hstl-oss task create`
  **only after handling the final user-confirmation response**, not while running Phases 6/9. Record the
  Actual Selection column as TODO -> receive user response -> update with the actual ID or exclusion reason.
- **Read-verification after registration is required** — after each `hstl-oss kb create` / `hstl-oss task create`,
  immediately verify the returned IDs exist via `hstl-oss kb get` / `hstl-oss task get`.
  Even one NOT_FOUND -> BLOCK + sprint stamp blocked + audit `ceremony.verification.failed`
  recorded. Follow-up to KB `mistakes/governance.md` M003 to prevent recurrence of "registration
  reported but row missing" cases.
- **KPT Try -> follow-up Task linkage required** — Try items recorded in Phase 5 must
  be briefed as a "Task candidate mapping table" in Phase 9 and recommended for
  registration by default unless the user explicitly skips. "Collect only, do not register"
  is allowed only when explicitly refused.
- **Lessons must be recorded as KB cards** — memory notes do not substitute for KB
- **Sprint branch -> dev merge required** — Sprint completion = branch merge + folder move
- The Stop hook blocks session end when unchecked items remain

## Handling BLOCKED responses

After fixing the unfinished items in that Phase, re-run `sprint_complete`.
Inspect the current Gate state with `harness_get("sprint", "$ARGUMENTS")`.

## Auto-verification

For Phase 7 (build/test) and per-Task Harness Gate processing, `hstl-oss harness auto-check`
batch-processes Go-deterministic items (`build_passed`, `tests_passed`, `lint_passed`).
The same is recommended for the final Dev verification Task at the end of a Sprint.

```bash
# For each Sprint Task
hstl-oss harness auto-check task <ID>           # 3 Go-deterministic items auto-checked
hstl-oss harness check task <ID> criteria_checked --evidence "..."  # human judgment manual
hstl-oss harness check task <ID> code_review --evidence "..."
hstl-oss task complete <ID> --with-ceremony
```

Detailed design: KB `architecture/decisions.md` A01 — auto-check safety decision.
