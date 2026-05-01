---
description: Assign a Task to a Sprint (BACKLOG -> Sprint), supports multiple IDs, auto-moves files. Use when the user says "assign task to sprint", "move tasks into the sprint", or "include this in sprint N".
argument-hint: "<task-id[,id2,...]> --sprint <sprint-id>"
allowed-tools: Bash(hstl-oss:*), Read, Glob
---

# Assign Task to Sprint: $ARGUMENTS

> **Command-first entry point**

> **Command First STRICT**: This Command is the official entry point for Sprint assignment.
> Calling `hstl-oss task assign <ID> --sprint ...` directly via Bash auto-records a
> `ceremony.bypass` event in the audit log. Bulk assignment is an exception — Policy SSOT:
> `CLAUDE.md` §Command First mapping table.

Assigns a Backlog Task to an active/waiting Sprint via the CLI.

## CLI usage

```
Bash("hstl-oss task assign <task-id> --sprint <sprint-id>")
```

Bulk assignment:
```bash
hstl-oss task assign T01,T02,T03 --sprint sprint-01 | jq '.data'
```

## Automated steps

| # | Item | Behavior |
|---|------|------|
| 1 | DB sprint column update | Set `tasks.sprint_id` |
| 2 | File move | `works/tasks/T*.md` -> `works/sprints/<sprint>/tasks/T*.md` |
| 3 | Git tracking | `git mv` preferred, `os.Rename` as fallback |
| 4 | BACKLOG.md update | Auto-sync — display Sprint column |
| 5 | CURRENT-FOCUS.md update | Reflected automatically when Sprint is active |
| 6 | Audit log | `task.assigned` event |

## When to use

- Plan an existing Backlog Task into the next Sprint
- Inject an additional Task into an in-flight Sprint (allowed but not recommended)

## Caveats

- **Cannot reassign done Tasks** — the sprint column is locked once done
- **Missing sprint** — BLOCKED if the Sprint is not in the backlog/active folder
- **Do not edit the frontmatter sprint field with Write/Edit** — causes DB drift
- **Partial failure with multiple IDs**: successful ones still commit; check the response

## Related Commands

- `/hstl-oss:task:unassign` — Sprint -> Backlog
- `/hstl-oss:task:create` — assign at create time via `--sprint`
- `/hstl-oss:sprint:start` — first-Task assignment flow after Sprint activation

## Output

JSON: `{status: ok, data: {task_id, sprint_id, file_path, previous_path}}` — single ID. With multiple IDs (comma-separated), `{assigned: [{task_id, sprint_id, file_path}], failed: [...], count}`.

Errors:
- Task not found: `error_category=NOT_FOUND`
- Sprint not found / status=completed: `error_category=REJECTED`
- Already assigned to the same Sprint: `error_category=CONFLICT` + recovery_hint
