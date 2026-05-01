# Task Checklists

## Pre-review of Design Documents

Before authoring a Task, always check the related design documents:

| Task type | Documents to check | What to look for |
|----------|----------|----------|
| Feature implementation | `docs/03-design/feature/` | Detailed feature spec, API spec |
| Domain logic | `docs/03-design/domain/` | Domain model, business rules |
| UI/UX | `docs/03-design/ux/` | Screen design, user flow |
| Infra / deploy | `docs/03-design/deployment/` | Deploy topology, environment config |
| Architecture | `docs/02-architecture/` | Tech decisions, constraints |

### Review Procedure

```
1. Identify the design document(s) related to the Task
2. Extract requirements from the design document
3. Author the completion criteria based on the design document
4. List the referenced documents in the tech notes
```

---

## Task Creation Checklist (CLI-based)

- [ ] Call `hstl-oss task create --title ... --type ...` (Tasks are always created in BACKLOG)
- [ ] CLI atomically issues the ID, creates the file, updates BACKLOG, and writes the audit log
- [ ] Author the body of the created file (purpose / requirements / completion criteria / scope limits / nature) — placeholder bodies are rejected for assign
- [ ] Completion-criteria checkboxes (at least 3, in measurable form)
- [ ] Sprint inclusion is a separate step after the body is written: `hstl-oss task assign <ID> --sprint <sprint-id>` (the "create → write body → assign" 3-step ceremony is enforced)

---

## Task Completion Checklist (required to transition to done)

- [ ] All completion-criteria checkboxes 100% checked
- [ ] Build passes
- [ ] All tests pass
- [ ] Result section authored (completion date, changed files, key decisions)
- [ ] SPRINT.md status synced (Sprint Tasks only)
- [ ] UX mapping update verified (Tasks involving UI changes only)

---

## Sprint Start Checklist (CLI-based)

- [ ] Call `hstl-oss sprint start <id> --with-ceremony` (moves backlog → active folder + transitions state)
- [ ] doc-review passes
- [ ] Design readiness verified
- [ ] CURRENT-FOCUS.md updated (automatic)

---

## Sprint Completion Checklist

The Sprint ceremony state is owned by `hstl-oss harness get sprint <id>` (SSOT) — confirm all 11 phases are
`[✓]` before calling `hstl-oss sprint complete`.

---

## Task Body Backtick-path Convention

Backtick-wrapped file paths (`path/to/file`) in the Task body are validated for
existence by the same convention in both the design_readiness check of
`sprint start` and the stale_path check of `doc-review`. WARN is emitted when
the stale ratio exceeds the threshold (default 30%). The threshold is tunable
via the project's sprint-start stale-percentage environment variable.

**Authoring guidelines**:

- **Files planned for future creation** are caught as stale even if written as backtick paths.
  The WARN at Sprint planning time is normal (the file does not exist yet);
  it goes away once the file is actually created and the check is rerun.
- **Code identifiers in narrative context** (`func Foo()`, `const Bar`, etc.)
  contain no slash and are auto-excluded — no false positives.
- **Command patterns** (`go test ./...`) include spaces and trailing `...` and are auto-excluded.

The prefix auto-discovery list (e.g. `pkg/`, `cmd/`, `skills/`, `commands/`)
is owned by the project's CLI implementation as a single source.

---

## Interface Signature Change Grep Checklist

A refactor Task changing a Go interface method signature (parameters / return
values / name) requires touching **every implementation** as well as the
definition. Do not use build failures as a discovery loop — list the affected
implementations up front when authoring the Task:

```bash
# 1) Same-named function implementations outside the interface definition
grep -rn "func .*) MethodName(" <repo>/

# 2) mockXxx — test-mock struct methods
grep -rn "func .*mock.*) MethodName(" <repo>/

# 3) ServiceAdapter / composition — delegation layer
grep -rn "MethodName(" <repo>/internal/app/
```

Register the result as `## Requirements` checkboxes (with file path + line). Under the **1 Task = 1 Commit**
principle, all implementations must be modified together in one commit. Do not leave intermediate commits
in a build-broken state.

---

## Pre-refactor Import-graph Grep Checklist

For a refactor Task (package move / field rename / new import / module path
change, etc.), measure the current state of the import graph before starting:

1. **Identify the target pkg / struct field**
2. **Check reverse references in affected packages** — confirm whether source
   pkgs that already import the target pkg might also be referenced in the
   reverse direction. If both directions exist → cycle risk → consider
   migrating to a shared util first
3. **Identify JSON tags / API response surfaces** — when renaming a struct field,
   confirm external exposure; keep deprecated aliases when needed
4. **Identify DB column / schema impact**
5. **Register results as `## Requirements` checkboxes**

---

## Section-boundary Preservation on Edit

When editing the Task body or other markdown via Edit / replace_all, **regressions
where section headers (## / ###) disappear** can happen.

**Before Edit**:
- Read the target file first; note the section header list (`grep -E "^##|^###"`)
- Confirm `replace_all=false` (default) — `replace_all` may unintentionally affect other sections
- Confirm `old_string` lives inside a single section (split if it spans multiple)

**After Edit**:
- Re-confirm the section header list with `grep -E "^##|^###" <file>` — no headers should have disappeared
- Compare checkbox count (`grep -c "^- \[" <file>`) before vs after (only intended changes)
- Confirm sections outside the change scope are preserved verbatim

---

## Pre-`task:complete` Artifact Verification

Under the strict policy (default), `task complete` BLOCKS if files listed under
the result section's `### Artifacts` are not visible in `git diff`. Cross-check
the following before calling complete:

```bash
git diff --cached --name-only           # staged
git status --short                      # includes untracked
```

**Checklist**:
- Each bullet in `### Artifacts` has a path that exists in `git diff --cached --name-only` or `git status --short`
- New files are registered in the index via `git add` or `git add -N`
- If you add from a subdirectory, are the paths relative to the **repo root**?

Verification policy levels (env `HOSTLER_OSS_TASK_RESULT_CHECK_POLICY`):
- `strict` (default) — BLOCK if a missing file is found
- `warn` — warn and pass
- `off` — skip the check

Use `warn` / `off` only as a temporary mitigation. The real fix is to record changed files accurately in the result section.

---

## `task update --status` Escape Hatch — Forbidden for AI

`hstl-oss task update --status <s>` is a **drift-recovery-only escape hatch**.
The everyday Task lifecycle must always go through the normal paths
(`task start` / `complete` / `reopen`).

**AI guidance — strictly enforced**:

1. **No automatic retry** — when a normal command fails, do not call `task update --status`
   as a retry to recover from drift. Analyze the failure and report to the user first
2. **Do not invoke without explicit user approval** — even when drift is confirmed, only call
   it once after asking the user "may I force the transition via the escape hatch?"
3. **One-call rule** — if the issue recurs, isolate the root cause (DB drift /
   frontmatter drift / transition-logic bug) into a separate Task. Repeating the escape
   hatch hides symptoms
4. **Audit trail required** — after invocation, verify the trace via the audit
   event `task.status.force_updated`. Do not ignore the warnings array in the CLI response

**Correct usage**:

```bash
# After reporting to the user, with approval, call once
hstl-oss-oss task update <ID> --status done
hstl-oss-oss audit log --event-type task.status.force_updated | grep <ID>
```

---

## E2E Verification Temp Task Pattern

For E2E lifecycle verification in a dev environment, create / complete /
delete a temp Task as follows:

```bash
# 1. Create the temp Task (registered only in BACKLOG)
hstl-oss-oss task create --title "Temp Task for lifecycle verification" --type chore \
  --summary "E2E lifecycle verification — to be deleted after completion"

# 2. Author the minimum Task body (avoid placeholder_body)
#    Fill in the purpose / requirements / completion criteria sections in works/tasks/T*-*.md

# 3. Verify lifecycle behaviour
hstl-oss-oss task start <ID>
hstl-oss-oss task complete <ID> --with-ceremony

# 4. Permanently delete after completion (2 steps)
hstl-oss-oss task reopen <ID> --reason "Delete temp E2E verification Task"
hstl-oss-oss task delete <ID> --reason "Cleanup after E2E verification"
```

**Notes**:
- Creating / deleting without sprint assignment is simpler and safer (sprint-assigned Tasks face result-section validation)
- `task delete` works only in todo state — `reopen` first is required
