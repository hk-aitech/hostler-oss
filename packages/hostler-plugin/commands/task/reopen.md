---
description: Reopen a Task — done/in-progress -> todo/in-progress, reason required, Harness Gate reset, auto folder return. Use when the user says "reopen the task", "undo completion", or "the task isn't actually done".
argument-hint: "<task-id> --reason \"...\" [--status todo|in-progress]"
allowed-tools: Bash(hstl-oss:*), Bash(git:*), Read, Glob
---

# Reopen Task: $ARGUMENTS

> **Command-first entry point**: This Command is the official entry for Task reopen.
> When the user says "reopen the task" or "revert it", do not call the CLI directly —
> first call this Command to run the briefing, reason validation, and follow-up guidance,
> then call the CLI from inside the Command.

> **Command First STRICT**: This Command is the official entry point for Task reopen.
> Calling `hstl-oss task reopen <ID> --reason ...` directly via Bash auto-records a
> `ceremony.bypass` event in the audit log and counts as a violation in retro.
> Policy SSOT: `CLAUDE.md` §Command First mapping table.

Reopens a Task via the CLI.

## CLI usage

```
Bash("hstl-oss task reopen <task-id> --reason \"<reason ≥10 chars>\" [--status todo|in-progress]")
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss task reopen T01 --reason "Reopening due to test failure" | jq '.data'
```

Returns: `{ "task_id": "...", "previous_status": "done", "new_status": "todo",
"harness_reset": true, "file_moved": "..." }`

## Output format

```markdown
=============================================
  Task {id} reopened
=============================================

  Title:    {title}
  Type:     {type}
  Sprint:   {sprint_id} or (backlog)

  Status transition:
  done -> {todo|in-progress}

  Reason:
  {reason}

  Auto-handled:
  - Harness Gate items reset (check history cleared -> re-verification required)
  - File location returned (for Sprint-unassigned: completed/T*.md -> T*.md)
  - BACKLOG.md / CURRENT-FOCUS.md auto-updated
  - audit log: task.reopened (reason persisted)

  Next steps:
  - When work is done, run `/hstl-oss:task:complete <id>` again
=============================================
```

## Automated steps

| # | Item | Behavior |
|---|------|------|
| 1 | reason validation | ≥10 chars required (CLI level) |
| 2 | Status transition | done/in-progress -> todo (default) or in-progress |
| 3 | Harness Gate reset | Delete `harness_items` rows -> re-verification required |
| 4 | File folder return | Sprint-unassigned Tasks only (sprintID guard) |
| 5 | BACKLOG/CURRENT-FOCUS update | Automatic |
| 6 | Audit log | `task.reopened` event + reason recorded |

## When to use

- **Rework needed**: after done, additional requirements / bugs discovered
- **Mistakenly completed**: Harness Gate passed but verification later proves inaccurate
- **Test failure**: failure discovered in later verification (e.g. Phase 7 sprint build/test)
- **User request**: stakeholder rejects the result

## Caveats

- **reason ≥10 chars required** — CLI validates and BLOCKs otherwise
- **Default status = todo** — to immediately resume in-progress, use `--status in-progress`
- **Path to permanently delete a done Task**: 1) reopen to todo -> 2) `task delete` (mistake prevention)
- **Decision guide**: see "Task state decision guide" in `task-management`
  SKILL.md for the difference between reopen vs delete vs update

## Related Commands

- `/hstl-oss:task:complete` — completion (the opposite direction)
- `/hstl-oss:task:start` — todo -> in-progress (resume work after reopen)
- `/hstl-oss:task:delete` — permanent delete for todo Tasks (done requires reopen first)
