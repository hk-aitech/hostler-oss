---
description: Get a single Task's details — frontmatter + dependencies + harness status. Use when the user asks "show me task T123", "what's in this task?", or wants metadata for a single Task.
argument-hint: "<task-id>"
allowed-tools: Bash(hstl-oss:*), Read, Glob
---

# Get Task: $ARGUMENTS

> **Command-first entry point**

Queries detailed information for a single Task via the CLI.

## CLI usage

```
Bash("hstl-oss task get <task-id>")
```

JSON parsing (jq):
```bash
hstl-oss task get T01 | jq '{id,title,status,sprint,depends_on}'
```

Returned fields:
- `id`, `title`, `type`, `status`, `priority`, `estimate`
- `sprint` (parent Sprint ID or null)
- `depends_on` (predecessor Task ID array)
- `file_path` (frontmatter SSOT path)
- `created_at`, `updated_at`

## Automated steps

| # | Item |
|---|------|
| 1 | DB row lookup (sqlite `tasks` table) |
| 2 | Verify consistency with frontmatter cache (warn on drift) |
| 3 | Summary of dependent harness_items (optional) |

## When to use

- Verify metadata before writing the Task body
- Inspect dependency Tasks
- Single-record info for script automation
- Quick AI brief of Task context

## Caveats

- **Nonexistent ID** -> NOT_FOUND error + recovery_hint with similar IDs
- **Drift detected** -> verify with `hstl-oss backlog sync --dry-run`, recover with sync
- For full body, call `Read <file_path>` separately

## Related Commands

- `/hstl-oss:task:list` — multi-record listing (with filters)
- `/hstl-oss:task:start` — auto-briefs at start
- `/hstl-oss:task:update` — modify metadata fields

## Output

JSON: `{status: ok, data: TaskFull: {task_id, title, type, estimate, priority, status, sprint, depends_on, file_path, work_ticket, completion_hmac?, harness_items: [...]}}`.

Errors:
- Task not found: `error_category=NOT_FOUND` + recovery_hint (`hstl-oss task list`)
