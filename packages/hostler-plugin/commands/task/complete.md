---
description: Complete a Task — performance summary + Harness verification + commit guide + reminders. Use when the user says "complete the task", "finish this task", or "mark task done".
argument-hint: "[task-id]"
allowed-tools: Bash(hstl-oss:*), Bash(git:commit), Bash(git:diff), Bash(git:status), Bash(git:mv), Bash(go:build), Bash(go:test), Bash(dotnet:build), Bash(dotnet:test), Read, Write, Edit, Glob
---

# Complete Current Task

> **Command-first entry point**: This Command is the official entry for Task completion.
> When the user requests Task completion, do not call the CLI directly — first call this
> Command to run the performance summary, Harness verification, and commit guide, then
> call the CLI from inside the Command.

> **Command First STRICT**: This Command is a required workflow entry point.
> Calling `hstl-oss task complete <ID>` directly via Bash auto-records `ceremony.bypass` /
> `*.retried` events in the audit log and counts as a violation in Phase 5 retro.
> Policy SSOT: `CLAUDE.md` §Command First mapping table. Bypass only with explicit user
> approval and `HOSTLER_OSS_*_POLICY=warn|off`.

Completes a Task via the CLI.

## Pre-check — avoid missing `git add`

The result-section validation in `hstl-oss task complete` is strict by default — if a file
listed under `### Artifacts` is **missing from both staged and unstaged git diff**, the
command BLOCKs. Files that exist on disk but are untracked or not staged trigger the same
BLOCK, so verify once before calling the CLI.

```bash
# Just before entering the Command body
git status --short                    # are untracked (??) rows aligned with the artifacts?
git diff --cached --name-only         # do staged files cover the artifact list?
```

**Note — `git add` paths are repo-root relative**:
- Inside a subdirectory (`cli/`, `skills/*/`), `git add some.go` uses a path relative to
  that subdirectory and may not match the repo-root view. Adding from the root with
  **repo-root relative paths** like `git add cli/some.go` is safer.
- New files only need `git add -N` to pass strict validation (unstaged new files are
  included in diff). To commit them, run `git add` to actually stage.

## CLI usage

```
Bash("hstl-oss task complete TNN --with-ceremony")
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss task complete TNN --with-ceremony | jq '.ceremony.harness_gate'
```

Parse `ceremony.changed_files`, `ceremony.harness_gate`, and `ceremony.result_section`
from the CLI JSON response and preserve the existing completion-validation rendering.

## Output format

When `task_complete` runs, **summarize the work** in the format below. Run everything
contiguously and complete automatically.

```markdown
=============================================
  Task {id} completion check
=============================================

  Title:    {title}
  Type:     {type} -> commit prefix: {prefix}
  Sprint:   {sprint_id}

  Work summary:
  - Changed files: {N} (+{added} / -{deleted} lines)
  - New files: {list}
  - Modified files: {list}
  - Tests: {N new added} / {M existing modified}

  Completion criteria coverage:
  - {criterion 1} — {evidence}
  - {criterion 2} — {evidence}
  - WARN: {criterion 3} — partially met ({reason})

  Harness Gate:
  - criteria_checked — {evidence summary}
  - build_passed — 0 errors / 0 warnings
  - tests_passed — {N} pass / 0 fail
  - code_review — {verification summary}

  Suggested commit message:
  {type_prefix}({task_id}): {commit message suggestion}

  -> Auto-completing the Task and creating the commit
=============================================
```

### Reminder output

If the CLI response includes a `reminders` array, print it as a blockquote before the verification:

```markdown
> **Reminders**
> - Use full file paths in the result section (no `src/.../` shortcuts)
> - Do not bypass the Harness Gate
```

### Automated steps

| # | Item | Method |
|---|------|------|
| 1 | Aggregate changed files | `git diff --stat` |
| 2 | Verify completion-criteria checkboxes | Parse Task file (`- [x]` vs `- [ ]`) |
| 3 | Build verification | `dotnet build` (when code changed) |
| 4 | Test verification | `dotnet test` or `pytest` (when code changed) |
| 5 | Full Harness Gate | Auto-call `harness_check` |
| 6 | Suggest commit message | type + task_id + change summary |
| 7 | **Auto-move Sprint-unassigned Task folder** | `works/tasks/T*.md` -> `works/tasks/completed/T*.md` + DB `file_path` update |

### Summary of automatic side effects

When `hstl-oss task complete` performs the done transition, the **actual** automatic
side effects are (per implementation in `pkg/task/task.go` `Complete` +
`transitionTask` + `TransitionTaskAtomic`):

| # | Side effect | Condition | Implementation |
|---|-------------|------|------|
| 1 | **Sprint-unassigned file move** | `sprintID == ""` | `works/tasks/T*.md` -> `works/tasks/completed/T*.md` (`git mv` preferred, `os.Rename` fallback). Idempotent |
| 2 | **DB `tasks.file_path` update** | When file moves | Via `TransitionTaskAtomic` callback return (atomic) |
| 3 | **frontmatter `status` update** | Always | `UpdateTaskStatus` — `in-progress` -> `done` |
| 4 | **Append `## Status History`** | Always | `AppendStatusHistory` |
| 5 | **completion HMAC signature** | When frontmatter exists | `StampTaskHMAC` (fail-soft) |
| 6 | **Auto-insert `## Result` stub** | When section is missing | `EnsureResultSectionStub` — runs before HMAC computation |
| 7 | **Auto-strip blockquote** | When stub not inserted | `StripTaskBlockquote` |
| 8 | **SPRINT.md status column update** | `sprintID != ""` | `UpdateSprintMD` (non-atomic, recoverable) |
| 9 | **`task.completed` audit event** | Always | `audit.LogEvent` |
| 10 | **Reminder injection** | When config exists | `config.GetReminders("task.complete")` |
| 11 | **post_actions** | type=hotfix/infra | follow-up recommendation strings |

**What `hstl-oss task complete` does NOT do** (commonly misunderstood — clarification):

- **No BACKLOG.md update**. Task complete/reopen does not call `UpdateBacklogMD` /
  `RemoveFromBacklogMD`. BACKLOG.md Task-state reflection converges at
  **the bulk sync at sprint complete** or via `hstl-oss backlog sync`. This is
  intentional — task complete is the tight-loop path; bulk BACKLOG.md updates
  are a Sprint-boundary responsibility.
- **No CURRENT-FOCUS.md update**. `UpdateCurrentFocus` is called only by
  `hstl-oss sprint start` / `complete` / `project delete`. Task transitions do
  not touch CURRENT-FOCUS.md.

Sprint-unassigned file-move details: `skills/project-structure/SKILL.md` "Task
move flow" section. Design rationale: KB `architecture/decisions.md` A02.

### Result-section file-existence policy

`hstl-oss task complete` verifies that files listed in `## Result -> ### Artifacts`
actually appear in `git diff`. The policy has three levels, prioritized as
**env var -> `.hostler/project-config.yaml` -> default (strict)**.

| Policy | Behavior | Env value |
|------|------|------------|
| `strict` (default) | BLOCK on missing files | `HOSTLER_OSS_TASK_RESULT_CHECK_POLICY=strict` |
| `warn` | Print warning and pass | `HOSTLER_OSS_TASK_RESULT_CHECK_POLICY=warn` |
| `off` | Skip the check | `HOSTLER_OSS_TASK_RESULT_CHECK_POLICY=off` |

Configurable in `.hostler/project-config.yaml`:
```yaml
task_result_check:
  policy: warn  # strict | warn | off
```

> **Avoiding the check is not recommended**: use `warn` / `off` only as a temporary
> escape hatch. The real fix is to list the actual changed files accurately in the
> result section.

### Auto-writing the Task result section

> **Proceed automatically without user confirmation.** When validation passes, immediately run task_complete + git commit.

On completion, write the `## Result` section in the Task file using this sub-heading
convention. The parser scans for file paths only under `### Artifacts`; other
sub-headings (### Design Decisions, ### Verification, etc.) are treated as narrative
and excluded from scanning.

```markdown
## Result

### Design Decisions

{Describe key choices and trade-offs. File names may be mentioned — the parser treats them as narrative and skips them}

### Artifacts

- `{full file path}` (new/modified/deleted)
- `{full file path}` (new/modified/deleted)

### Verification

- Build: {result}
- Tests: {result}
- Other: {command + result}

### Commit

- {commit SHA}: {message}
```

**Sub-heading parser convention**:
- Only `### Artifacts` / `### Changed files` / `### Files` are scanned for file paths
- Other sub-headings (### Design Decisions, ### Verification, ### Follow-up Tasks, etc.)
  are treated as narrative — file-name mentions inside them are not false positives
- **Backward compat**: if you write everything directly under `## Result` without
  any sub-heading, the parser scans the whole section (compatible with older Tasks)
- For clear separation, use explicit `### Artifacts` and `### Design Decisions`

### Anti-patterns

**Mistake 1 — backtick paths in narrative under backward-compat mode**:

```markdown
## Result
[Direct prose without sub-headings]

The key finding was the missing version field in `.hostler/project-config.yaml`...
```

Without sub-headings under `## Result`, the parser uses backward-compat mode and scans
everything. The backticked path above is captured and the command BLOCKs. **Fix**:
wrap path-mentioning narrative in an explicit sub-heading like `### Design Decisions`,
or remove the leading `.` from the path to avoid the file-existence check.

**Mistake 2 — listing artifacts under an arbitrary heading like `### Changes` instead of `### Artifacts`**:

```markdown
## Result

### Changes      <!-- not recognized by the parser -->
- `src/foo.go` (modified)
```

`### Changes` is not in the artifact-keyword list and is treated as narrative -> the actual
files are not scanned and "missing_files" may be triggered under strict policy. **Fix**:
use one of `### Artifacts` / `### Files` / `### Changed files`.

**Good example** — explicit sub-heading separation:

```markdown
## Result

### Design Decisions
Used the global `.hostler/project-config.yaml` store. Alternatives considered ...

### Artifacts
- `cli/pkg/config/v2.go` (new)
- `cli/pkg/config/v2_test.go` (new)

### Verification
- `go test ./pkg/config/...`: PASS (12 cases)
```

Mentioning `.hostler/project-config.yaml` in narrative is ignored by the parser
(sub-heading mode is active).

## Handling BLOCKED Harness Gates

**Principle: before calling `harness_check`, always run `harness_get` to discover the available `item_id`s.**

Harness items differ per Task type (per `cli/schemas/harness_defaults.json`):

| Task Type | Required Items |
|-----------|----------------|
| `feature` / `refactor` | `criteria_checked`, `build_passed`, `tests_passed`, `code_review` |
| `bugfix` | `criteria_checked`, `reproduction`, `root_cause`, `build_passed`, `tests_passed` |
| `infra` | `criteria_checked`, `deploy_verified`, `rollback_plan` |
| `docs` | `criteria_checked`, `doc_review` |
| `test` | `criteria_checked`, `tests_passed` |
| `chore` | `criteria_checked` |

Reflexively checking `build_passed` produces `NOT_FOUND` errors on `infra`/`docs`/`chore` Tasks. Use only `items[].id` from the `harness_get` response. Details: `skills/task-management/SKILL.md` — "Mandatory harness_get prelude" section.

### Auto-verification recommended

Go-deterministic items (`build_passed`, `tests_passed`, `lint_passed`) can be batch-verified
with a separate command. Instead of calling `harness check` 6 times, prefer:

```
hstl-oss harness auto-check task TNN              # 1. Auto-verify and check 3 Go-deterministic items
hstl-oss harness check task TNN criteria_checked --evidence "..."  # 2. Manual for human-judgment items
hstl-oss harness check task TNN code_review --evidence "..."
hstl-oss task complete TNN --with-ceremony        # 3. Complete
```

`auto-check` does not touch human-judgment items (`criteria_checked`, `code_review`,
`reproduction`, `root_cause`, `deploy_verified`), preserving Harness Gate safety.
Detailed design: KB `architecture/decisions.md` A01.

### --auto-check-criteria

The `hstl-oss task complete --auto-check-criteria` flag parses checkboxes in the
Task body's `## Completion Criteria` section: when all are `[x]`, it auto-checks
the Harness `criteria_checked` item; if any are unchecked, it BLOCKs. This skips
manual entry of `criteria_checked` and shortens the ceremony.

```bash
hstl-oss task complete TNN --auto-check-criteria --with-ceremony
hstl-oss task complete TNN --auto-check-criteria --dry-run  # validation only
```

Detailed behavior, scan targets, JSON samples: `skills/task-management/SKILL.md`
— "--auto-check-criteria usage guide".

### --dry-run / --interactive

- `--dry-run`: only run Harness / result-section validation, no completion transition
- `--interactive`: per-BLOCKED unchecked item, prompt `[y/n/s]` (auto-disabled when not on a TTY)

### Manual invocation (when handling all items directly)

```
harness_get("task", "TNN")                              # 1. List available item_ids
harness_check("task", "TNN", items[i].id, evidence)     # 2. Use only the returned ids
Bash("hstl-oss task complete TNN --with-ceremony")      # 3. Retry via CLI
```

On NOT_FOUND, `recovery_hint` contains the available `item_id` list for that Task type.

## Reverting an incorrectly completed Task

If you need to undo a done state due to additional requirements, inaccurate verification,
or test failures, use `/hstl-oss:task:reopen`. Reopen requires a reason (≥10 chars),
auto-resets the Harness Gate, and auto-returns the folder for Sprint-unassigned Tasks.
A done Task cannot be deleted with a single command — the two-step `reopen -> delete`
sequence is enforced (mistake prevention).
