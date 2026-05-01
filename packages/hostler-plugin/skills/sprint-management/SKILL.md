---
name: sprint-management
description: Sprint lifecycle management — create + task decomposition, start + design readiness, complete + 10-phase quality audit. Use this skill whenever the user mentions sprint planning, sprint lifecycle, sprint progress, "create a sprint", "start the sprint", "complete the sprint", or asks to break a sprint into tasks — even when they don't say "sprint-management". Do NOT use for individual task operations (use `task-management`) or for editing SPRINT.md directly.
compatibility:
  tools: [Read, Write, Edit, Bash, Glob, Grep]
paths: ["works/sprints/**/*"]
trigger_commands: ["sprint.*", "task.assign", "task.unassign"]
---


# Sprint Management

Use the **hostler CLI** for the Sprint lifecycle.

> **Branch policy**: For Sprint-branch fork/merge conventions, follow
> `../branch-workflow/references/branch-policy.md` (the shared default template).
> Default assumption: `sprint-{id}` branches off `dev` and, on Sprint completion,
> merges back to `dev` and is deleted locally. If the project defines its own SSOT,
> the project SSOT wins.

## CLI usage

Inspect CLI commands and parameters via the manifest:

```bash
hstl-oss-oss --manifest --skill sprint-management
```

The 6 ceremony-bearing commands (create / start / complete) MUST go through the Command interface (`/hstl-oss:sprint:*`).

## Automatic CURRENT-FOCUS.md management

`works/sprints/active/CURRENT-FOCUS.md` is updated automatically by the `hostler` CLI
(`UpdateCurrentFocus` in `cli/pkg/fileutil/fileutil.go`).

**Auto triggers**:
- `hstl-oss sprint start <id>` — switch to sprint-active state, regenerate the current-Task focus table
- `hstl-oss sprint complete <id>` — switch to idle state
- `hstl-oss task start/complete` — when an active Sprint exists, update the focus row

**Rule (absolute)**: never edit `CURRENT-FOCUS.md` directly with Write/Edit.
If you suspect a mismatch, check the diff with `hstl-oss backlog sync --dry-run` and run
`hstl-oss backlog sync`. Same SSOT mechanism as BACKLOG.md.

## Cascade Gate (sprint complete)

`hstl-oss sprint complete` triggers the 10-Phase sequential validation:
1. doc-review — 2. code-review — 3. work-audit — 4. doc-cross-check (conditional)
5. retro — 6. learned — 7. Dev environment deploy validation
8. Binary build + install (conditional) — 9. Follow-up Task derivation — 10. Skill update review

### Phase 6 / 9: no auto-registration + actual-selection column convention

In Phase 6 (lessons) / Phase 9 (follow-up Tasks), record only the **candidate table** in SPRINT.md.
`hstl-oss kb create` / `hstl-oss task create` MUST be invoked **only after the final user confirmation**.

**Table structure** (auto-injected by EnsureCeremonySections):
- Phase 6: `| # | Candidate | Category | Source Task | Recommended | Actual selection | Rationale |`
- Phase 9: `| # | Proposal | type | est | pri | Recommended | Actual selection | Rationale |`

**Actual-selection value convention**: `Registered (ID)` / `Excluded (reason ≥10 chars)` / `Merged (existing ID)`.
For zero-item cases, a single "N/A" row plus an explicit exclusion reason is mandatory.

### Phase 9: follow-up Task derivation — 3-source unified scan
- **KPT Try**: Phase 5 retro Try items → Task candidates
- **Lesson keywords**: regex-scan each Task's `### Lessons` block
  (`separate Task|outside this Task scope|follow-up Task|future Task candidate`)
- **Scope limits**: collect every bullet from each Task's `### Next Sprint or later` block
- Record in the unified 3-source mapping table → user confirmation → `hstl-oss task create` (final step)

### Phase 10: skill update review
- Map files / features changed in the Sprint to relevant skills
- Project skills: fix immediately (bugs) or describe an improvement proposal
- Shared skills: file an issue via `/report-issue`

If anything is BLOCKED, return `{"status": "BLOCKED", "phase": N, ...}`. Resolve and retry.

## Principles

- **Use the CLI** — never change Sprint state with manual `git mv` or `sed`
- **Don't bypass the Cascade Gate** — when BLOCKED, fix the offending Phase and retry
- **Code-changing Sprint**: the final Task = build + verify Task (`type: infra`, `priority: p1`)
- **Never skip the sprint:complete 10 phases** — verify all phase items are checked via `hstl-oss harness get sprint <id>` (harness_items SSOT)
- **SPRINT.md 7-column standard**: `ID | Title | type | estimate | priority | status | Dependencies`

## Consistency warnings

Sprint create/start responses may include a `consistency_warnings` field.
On detected file/DB drift, the response returns WARN/INFO-level warnings.
When warnings appear, recommend `hstl-oss backlog sync` to reconcile.

## ADR deferral expiry check on Sprint start

Include an **ADR deferral expiry check** in the Sprint-start brief. Goal: structurally
prevent ambiguous deferral expressions (e.g. an ADR body saying `Sprint N+` /
`next Sprint`) from being neglected across multiple Sprints.

### AI procedure

1. Parse the entry sprint number (e.g. sprint-NN) when the Sprint-start skill is invoked
2. Scan the `deferred_until` field in the frontmatter of project ADR documents
3. If the entry sprint number ≥ `deferred_until`, add a WARN to the brief:
   ```
   ⚠️ Deferral expired: ADR-XXX (deferred_until=sprint-NN) — current Sprint MM is starting. Review needed.
   ```
4. For ADRs without `deferred_until` but with vague body expressions like "Sprint N+" / "next Sprint",
   cross-check the open-deferral catalog in the project docs §2
   and emit a WARN based on AI judgment

### Convention SSOT

The `deferred_until` field convention and the current open-deferral catalog live in
[the project docs](../../../../docs)
as the single source of truth.

## Sanity-check expected values in the validation matrix

> **Background**: When the validation matrix in a Sprint plan carries an "unattainable
> expected value" (e.g. `orphan_entry=0` — unrealistic because the catalog scope is broader
> than the code area), and the gap is only discovered during the Dev validation right
> before Sprint completion, you pay matrix-rewrite cost + a partial PASS verdict.
> KB grounding for this section: project docs (validation-matrix expected-value sanity check).

When a Sprint plan's validation matrix (e.g. PLAN-XXX §6) or the Done Criteria of a Dev
validation Task uses an "expected = measured" matrix, **enforce a one-time dry-run sanity check
at the planning stage**.

### Procedure (3 steps)

1. **Author the matrix** — for each item, specify the validation command and expected value
   (e.g. `hstl-oss trac trace ... | jq '.data.orphan_entry'` → expected `0`).
2. **Run a dry run as soon as one dependency Task completes** — execute one of the matrix's
   validation commands once in the real environment to measure **the current baseline**.
   The most accurate moment is right after the dependency Task completes.
3. **Adjust the matrix if expected ≠ measured** — when the measurement can't reach the expected:
   - Adjust the expected value to the realistic baseline (e.g. `orphan_entry=0` → `orphan_entry≤140`
     + a follow-up plan to reduce gradually)
   - Or narrow the catalog scope (e.g. `--context X`)
   - Update the plan §6 + record the change rationale in the Sprint retrospective

### Example

- Matrix item: `hstl-oss trac trace pkg/<module> --context <CTX>` → expected `orphan_entry=0`
- Measured: 32 — the Context catalog also includes Features beyond the code module
  (HMAC Chain, Audit Log, etc.), so it's unrealistic
- Adjustment: narrow the expected value to `pkg/<module>'s entries: orphan_entry=0`,
  or change to baseline + 0 variance

### Anti-patterns to avoid

- ❌ Plan only specifies "ideal goals" + measurement happens right before Sprint completion → rewrite cost
- ❌ Assume "we're sure" about per-item validation commands without running them
- ❌ The plan author and the Dev-validation runner are different people, and the sanity check is skipped

## Sprint redesign workflow

When an existing Sprint's goal changes or its Task composition needs adjustment:

1. `hstl-oss task unassign` to release existing Tasks → returns to backlog
2. `hstl-oss sprint update` to change goal / title
3. `hstl-oss task assign` to assign new Tasks
4. `hstl-oss task delete` to discard Tasks (todo only)

For CLI command details, see `hstl-oss --manifest --skill sprint-management`.

## Sprint discard

When redesign isn't enough (duplication / scope overlap / goal drift / abort), discard
the Sprint itself. `hstl-oss sprint discard` is the official discard path; always use this
command instead of manual workarounds (DB UPDATE / folder rename).

### When to use (decision checklist)

If **any one** of the following holds, consider discard:

1. **Scope duplication** — goal / Task set heavily overlaps another Sprint → absorb one
2. **Scope overlap** — the Sprint goal turns out to be already covered by another Sprint
3. **Low progress + priority shift** — under 30% progress + a more urgent need arose, can't reorganize
4. **Goal invalidated** — the underlying decision / assumption changed and Tasks no longer matter
5. **Design mistake** — major design defect found → full redesign and a new Sprint

**Counter-cases (do NOT discard)**:
- Simple delays → just update goal / period via `sprint update`
- Removing some Tasks → `task unassign` + `task delete` (Task-level cleanup)
- Renaming / re-statusing a completed Sprint → **forbidden** (history-area immutability principle — project docs)

### Command usage

```bash
# Default (incomplete Tasks remain in the Sprint folder — manage manually)
hstl-oss-oss sprint discard sprint-NN --reason "Absorbed into sprint-MM due to scope overlap"

# Auto-return incomplete Tasks (recommended — preserves remaining work)
hstl-oss-oss sprint discard sprint-NN --reason "30% progress + goal drift" --return-tasks
```

- `--reason` is **mandatory and ≥10 characters** (persisted in the audit record, equivalent to reopen/delete conventions)
- With `--return-tasks`, todo / in-progress Tasks return to `works/tasks/` automatically
- File move: `works/sprints/<cur>/<id>/` → `works/sprints/discarded/<id>/`
- Audit: `sprint.discarded` event + payload `{from_status, to_status, reason, returned_tasks}`
- No HMAC signing (preserves the immutability of the completed chain)

### Transition rules

| Current status | → discarded | Result |
|----------------|-------------|--------|
| backlog | ✅ allowed | Cancelled before scheduling (the most common case) |
| active | ✅ allowed | Aborted in flight — CURRENT-FOCUS returns to idle |
| completed | ❌ rejected | History-area immutability — restore then re-discard (exceptional) |
| discarded | ❌ rejected | Already discarded (idempotency not supported) |

### Restore path (exceptional case)

When the discard decision was wrong or made without review:

```bash
hstl-oss-oss sprint update sprint-NN --status backlog --force
```

- Manual restore only (a dedicated `restore` command is deferred to a future Sprint)
- The folder must be moved manually: `works/sprints/discarded/` → `works/sprints/backlog/`
- The `sprint.discarded` event remains in the audit, so capture the restore context in the commit message

### Example scenario — Sprint A at 30% progress + goal duplication discovered

```bash
# 1) Inspect the situation
hstl-oss-oss sprint progress sprint-XX         # done=3/10 (30%)
hstl-oss-oss sprint list --status active       # found goal overlap with sprint-YY

# 2) Return incomplete Tasks + discard the Sprint (one command)
hstl-oss-oss sprint discard sprint-XX \
  --reason "Goal overlap with sprint-YY discovered — absorbing into sprint-YY" \
  --return-tasks

# 3) Verify
hstl-oss-oss sprint list --status discarded     # sprint-XX shown
hstl-oss-oss task list --status todo            # confirm 7 returned tasks in backlog

# 4) When needed, assign backlog Tasks to sprint-YY
hstl-oss-oss task assign T### --sprint sprint-YY
```

### Branch handling for discarded Sprints

For the `sprint-NN` git branch handling convention, see
[the project docs](../../../../docs)
§"Branch handling for discarded Sprints". Summary: **never merge into main/dev**; if audit retention is
needed, replace with an `archive/sprint-NN-discarded-YYYY-MM-DD` tag.

## References

- CLI manifest: `hstl-oss --manifest --skill sprint-management`
- Detailed rules: `references/estimation.md`, `references/sprint-templates.md`
- Design-readiness criteria for sprint start: `references/design-readiness.md`
- Worked example of an active Sprint: `examples/sample-sprint-overview.md`
- Change history (maintainers only): `references/changelog.md`
