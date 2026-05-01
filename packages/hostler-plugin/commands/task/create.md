---
description: Create a Task and register it in BACKLOG / Sprint. Use when the user says "create a task", "add a backlog item", or wants to file a unit of work.
argument-hint: [task-title]
allowed-tools: Bash(hstl-oss:*), Bash(git:status), Read, Write, Edit, Glob, Grep
---

# Create Task: $ARGUMENTS

> **Command-first entry point**: This Command is the official entry for Task creation.
> When the user requests Task creation, do not call the CLI directly — first call this
> Command to follow the parameter guidance, validation, and default rules, then call
> the CLI from inside the Command.

> **Command First STRICT**: This Command is the official entry point for Task creation.
> Calling `hstl-oss task create ...` directly via Bash auto-records `ceremony.bypass` /
> `*.retried` events in the audit log and counts as a violation in Phase 5 retro. CI /
> bulk creation is exceptionally allowed — Policy SSOT: `CLAUDE.md` §Command First mapping table.

Creates a Task via the CLI.

## CLI usage

```
Bash("hstl-oss task create --title '$ARGUMENTS' --summary 'one-line summary' --type feature --priority p2 --estimate M")
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss task create --title 'API endpoint' --summary 'New REST API endpoint to support external integration' --type feature | jq '.data.task_id'
```

> **Required `--summary` flag**: To preserve the context at creation time,
> `--summary` (minimum 10 chars, no upper limit) is required. It is inserted as a
> `## Summary` section in the Task file. If the user does not provide a summary,
> the AI infers Task intent and writes it.

> **Single-path summary semantic check via the CLI**:
> `hstl-oss task create` validates the summary in the order below. The AI just
> writes the summary and **calls the CLI directly** (no Task-tool pre-check needed).
>
>   1. Length check — minimum 10 chars (no upper limit)
>   2. Static heuristics — word diversity, banned phrases, purpose connectors. Hard-block on failure.
>   3. LLM subagent — only when heuristics pass, headless call to
>      `claude -p --system-prompt-file agents/summary-judge.md`. Session UUID
>      (`.hostler/.summary-judge-session-id`) is reused so the call costs ~$0.005.
>      Block with verdict + suggestion on failure.
>
> **Bypass paths** (urgent CI):
> `--skip-summary-validation` — skips all layers. Audit log records the bypass.
> `HOSTLER_OSS_SUMMARY_POLICY=moderate|lenient|off` — relaxes heuristics.
> `HOSTLER_OSS_SUMMARY_SUBAGENT=off` — skips only the subagent (heuristics still run).

> **`--sprint` flag is deprecated**: `hstl-oss task create` always creates Tasks in
> BACKLOG. Move into a Sprint as a separate step with
> `hstl-oss task assign <ID> --sprint sprint-NN` after writing the body. This protects
> the principle that placeholder-body Tasks must not be moved into a Sprint (assigning
> at create time would route around the body-writing ceremony).

Returns: `{ "task_id": "TNN", "file_path": "...", "created_at": "..." }`

## Automatic side effects

`hstl-oss task create` atomically updates the following beyond Task file creation —
the user/AI does not touch them separately:

- **`works/tasks/BACKLOG.md`**: auto-created on the first call, gains a Task row on
  each subsequent call. Also auto-synced on Sprint assign/unassign and completion.
- **`works/sprints/active/CURRENT-FOCUS.md`**: when an active Sprint exists, the
  current-Task focus table is updated alongside.
- Consistency check: `hstl-oss backlog sync --dry-run` -> if diffs exist, run
  `hstl-oss backlog sync` to reconcile.

> Editing `works/tasks/BACKLOG.md` or `CURRENT-FOCUS.md` directly with Write/Edit
> causes DB drift. Always go through the CLI.

## Parameters

| Parameter | Description | Default |
|---------|------|--------|
| title | Task title (start with a verb) | Required |
| summary | One-line summary (≥10 chars, what/why/success criteria) | Required |
| type | feature/bugfix/refactor/infra/docs/test/chore | Required |
| priority | p0/p1/p2/p3 | p2 |
| estimate | XS/S/M/L/XL | M |
| depends_on | Predecessor Task ID list | [] |

## CLI only

`hstl-oss task create` atomically performs ID issuance + file creation +
BACKLOG/SPRINT update + audit logging, so there is no fallback path. If the CLI is
not installed, recover with `cd cli && make install` first.

## Body-writing guidance after creation

The Task file includes a `## Type` section. When writing the body, check at least
one of the four options — clarifying the Task's character at Sprint planning time
prevents scope confusion.

- New structure / Update existing structure / Measurement, verification, deployment / Mixed

This cross-references the Sprint kickoff briefing's design-readiness section.

> **Tasks without a body cannot enter a Sprint**: when `hstl-oss task assign` detects
> a placeholder body, it **hard-blocks** with `failed[].reason = "placeholder_body"`.
> Write the body and retry. Emergency bypass:
> `HOSTLER_OSS_ASSIGN_ALLOW_PLACEHOLDER=1` (for marker false-positives, not recommended).

**Completion criteria vs goals**:
- `## Completion Criteria` = measurable minimum (acceptance threshold). Below this -> cannot transition to done.
- Aspirational goals ("≥N%", "achieve top score", etc.) belong in `## Notes`. Do not mix with completion criteria.

Details: `skills/task-management/references/detailed-guide.md`

## Output

JSON: `{status: ok, data: {task_id, file_path, work_ticket, title, type, estimate, priority, status: "todo"}}` — auto-issued task_id (T###) + work_ticket (WT-T###-<rand>).

With `--with-ceremony`, the response includes `ceremony.briefing` (body-writing guidance) + `ceremony.next_action`.

Errors:
- Missing required flag (--title): `error_category=INVALID_INPUT`
- Summary heuristic failure: `error_category=INVALID_SUMMARY_HEURISTIC` + recovery_hint
- Sprint specified but not found: `error_category=NOT_FOUND`
