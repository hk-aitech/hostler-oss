---
description: Create a Sprint — folder + SPRINT.md + Task list design + BACKLOG.md update. Use when the user asks to "create a sprint", "plan a sprint", or wants to start a new development cycle with grouped Tasks.
argument-hint: [sprint-id] [sprint-title]
allowed-tools: Bash(hstl-oss:*), Bash(git:status), Bash(git:diff), Read, Write, Edit, Glob, Grep
---

# Create Sprint: $ARGUMENTS

> **Command-first entry point**: This Command is the official entry for Sprint creation.
> When the user requests Sprint creation, do not call the CLI directly — first call this
> Command to apply the Task design guidance and required-rule validation, then call the
> CLI from inside the Command.

> **Command First STRICT**: This Command is the official entry point for Sprint creation.
> Calling `hstl-oss sprint create ...` directly via Bash auto-records `ceremony.bypass` /
> `*.retried` events in the audit log. Policy SSOT: `CLAUDE.md` §Command First mapping table.

Creates a Sprint via the CLI.

## CLI usage

```
Bash("hstl-oss sprint create --id sprint-NN --title 'Sprint title' --goal 'Sprint goal'")
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss sprint create --id sprint-NN --title 'Quality debt cleanup' | jq '.data.sprint_id'
```

Returns: `{ "sprint_id": "sprint-NN", "task_count": N, "file_path": "..." }`

Tasks are added individually via `hstl-oss task create` after Sprint creation.

## Generated files

- `works/sprints/backlog/sprint-NN/SPRINT.md` — Sprint metadata + Task list
- `works/sprints/backlog/sprint-NN/tasks/TNN-*.md` — Task files

> The CEREMONY.md file hierarchy is deprecated. Sprint ceremony state is single-sourced
> in the `harness_items` DB and queried via `hstl-oss harness get sprint <id>`.

## User memory rule conflict detection

**Background**: when refactoring "Dev environment deployment verification" historically
clashed with the user memory `feedback_no_prod_dev_change_without_explicit.md`, manual
re-edits became repetitive. `sprint:create` now auto-scans the Sprint goal and Task
candidates for Dev/Prod keywords and provides a WARN + redefinition suggestion.

**AI instruction** — run the script below right before calling `hstl-oss sprint create`:

```bash
memdir="$HOME/.claude/projects/$(pwd | sed 's|/|-|g')/memory"
bash packages/hostler-plugin/commands/sprint/scripts/scan_memory_rules.sh \
    "$memdir" \
    "<Sprint goal>" \
    "<Task 1 title>" "<Task 2 title>" ...
```

(memory path follows the Claude Code default: absolute path
`$HOME/.claude/projects/<slugified-cwd>/memory`)

If the script output is **non-empty**, display the WARN block above the briefing as-is
and ask the user for `[y/edit/cancel]`:

- `y` -> keep the original plan and proceed with `hstl-oss sprint create`. The audit log
  records a `memory.warning.bypassed` event (track via `hstl-oss audit log --event memory.warning.bypassed`).
- `edit` -> propose replacing Task titles/summaries/completion criteria with phrasing
  like "LocalDev integrated verification — Dev/Prod deploy deferred to a separate stage
  with explicit user approval" and re-prompt. The user can also provide their own wording.
- `cancel` -> abort `hstl-oss sprint create` and return to the Task design step.

If the script output is **empty** (rule inactive or no keywords detected), continue with
the existing flow — no WARN.

**Non-Claude-Code environments**: when the memory directory is missing, the script
silently skips (exit 0 + empty stdout). No effect on existing CI / headless executions.

**Self-test** (regression check after script changes):

```bash
bash packages/hostler-plugin/commands/sprint/scripts/scan_memory_rules.sh --self-test
```

**Scope limit**: only the `feedback_no_prod_dev_change_without_explicit` rule is in
scope for now. Generalized pattern-action mapping is for a future Sprint.

## Sprint scope pre-review ceremony

> **Background** (from a prior sprint retro). When a Sprint goal includes large scope
> changes — such as persona-boundary changes or external system integrations — without
> explicit user approval, scope-out occurs right before Sprint start, wasting work.
> Detect this heuristically before Sprint creation and require user confirmation.

### Three detection heuristics

If any of the following signals appear in the goal/Task list, the AI **must not** call
`hstl-oss sprint create` without explicit user confirmation.

| Category | Example keywords | Risk |
|------|---------|-----|
| **scope_boundary** | `another system`, `external domain`, `separate project`, `upper layer` | Work beyond this project's scope — possible Sprint scope violation |
| **external_integration** | `gitlab`, `external api`, `mcp`, `k8s`, `traefik`, `infra` | External system dependency — verification/deployment complexity spikes |
| **new_binary** | `new binary`, `add binary`, `split CLI` | Affects deployment pipeline — may exceed a single Sprint |

> Additional heuristic: **two or more L+ estimated Tasks**. Estimates are finalized at
> Task creation time; at create time, substitute the AI's prior judgment.

### User confirmation prompt template

When a signal is detected, the AI asks the user first using this format:

```markdown
=============================================
  Sprint scope pre-review needed
=============================================

  Sprint ID:    sprint-NN
  Title:        {title}
  Goal:         {goal}

  Detected signals:
  - [scope_boundary] "another system" — work beyond project scope included
  - [external_integration] "external api" — external infrastructure dependency

  This Sprint may conflict with the project's current stabilization stance.
  Proceed? [y/N/edit]

=============================================
```

User response:
- `y` / `yes` / `proceed` — AI calls `hstl-oss sprint create ... --force`
- `N` / `edit` — request a narrower goal
- no response — cannot proceed; ask again

### CLI enforcement flag (`--require-review`)

To enforce the heuristic in automation/CI, use `--require-review`. If at least one
signal is detected, exit code 1 (BLOCKED) — bypass with `--force` only after explicit
user approval.

```bash
# Manual: require review, pass when no signals
hstl-oss sprint create --id sprint-NN --title "..." --goal "stabilization" --require-review

# Signal detected -> BLOCKED — bypass after user approval
hstl-oss sprint create --id sprint-NN --title "..." --goal "extract module" --require-review --force
```

Implementation: `packages/hostler-cli/pkg/sprint/scope_review.go`.

## Required rules for code-changing Sprints

Last Task = Dev environment deployment verification (`type: infra`, `priority: p1`).
The `## Requirements` section must list every implementation Task in the Sprint along
with concrete verification methods.

### Boundary for Dev-verification Task completion criteria

**Items that sprint:complete will check must not appear in the Dev-verification Task's
completion criteria.** Because Task complete precedes sprint:complete in time, checking
events that have not yet occurred at the Task moment causes a "cannot complete" paradox.

Forbidden: items owned by sprint:complete such as "SPRINT.md Phase 5/6/9 records" /
"N KB cards registered" / "Sprint HMAC signature complete".

Correct placement: move such items to `SPRINT.md ## Completion Criteria` or sprint
`harness_items`, and leave only a "responsibility delegated to sprint:complete" comment
in the Dev-verification Task body. Detailed rules: `.claude/rules/task-development.md`
§sprint:complete responsibility migration.

## Output

JSON: `{status: ok, data: SprintCreateResult: {sprint_id, title, goal, status: "backlog", folder_path}}`. With `--with-ceremony`, the `ceremony` object includes the candidate Task matrix + reminders.

Errors:
- Duplicate ID: `error_category=CONFLICT` + recovery_hint (`hstl-oss sprint list`)
- Invalid ID format: `error_category=INVALID_INPUT` (regex `^sprint-\d+$`)
- Scope Review BLOCK (--require-review): `error_category=REJECTED` + heuristic signals
