---
description: Delete a Task (todo state only), reason required, file/DB/audit log synchronized. Use when the user says "delete this task", "remove the wrong task", or wants to discard a draft Task.
argument-hint: "<task-id> --reason \"...\""
allowed-tools: Bash(hstl-oss:*), Read, Glob
---

# Delete Task: $ARGUMENTS

> **Command-first entry point**

> **Command First STRICT**: This Command is the official entry point for Task deletion.
> Calling `hstl-oss task delete <ID>` directly via Bash auto-records a `ceremony.bypass`
> event in the audit log. Delete is destructive — no bypass. Policy SSOT: `CLAUDE.md`
> §Command First mapping table.

Permanently deletes a Task via the CLI.

## CLI usage

```
Bash("hstl-oss task delete <task-id> --reason \"<reason ≥10 chars>\"")
```

Returns: `{ "task_id": "...", "deleted_at": "...", "file_path": "..." }`

## Automated steps

| # | Item | Behavior |
|---|------|------|
| 1 | reason validation | Must be ≥10 chars, BLOCK otherwise |
| 2 | status validation | **Only `todo` allowed**. `in-progress` / `done` BLOCK |
| 3 | DB row delete | `tasks` + dependent `harness_items` cascade |
| 4 | File delete | Remove the Task file under `works/...` |
| 5 | BACKLOG.md update | Auto — remove the row |
| 6 | Audit log | `task.deleted` + reason persisted |

## When to use

- Tasks created in error (typos, duplicates, wrong type)
- Todo Tasks no longer needed due to canceled requirements
- Replacing a Task with split Tasks (delete the original)

## Caveats

- **Cannot delete done / in-progress directly** — done Tasks require `task reopen`
  -> `task delete` (mistake prevention). For in-progress, revert status to todo first.
- **Deletion is irreversible** — only recoverable from git history. Be careful.
- **Sprint-assigned Tasks can be deleted too** — the file in the Sprint folder is also removed

## Related Commands

- `/hstl-oss:task:reopen` — done -> todo transition (precursor to delete)
- `/hstl-oss:task:unassign` — to remove from a Sprint, use unassign (not delete)

## Output

JSON: `{status: ok, data: {task_id, deleted: true, reason, file_path: "(removed)"}}` — deleted task_id + reason (permanent audit record).

Errors:
- Task not found: `error_category=NOT_FOUND`
- status != todo (in-progress / done): `error_category=REJECTED` + recovery_hint (`/hstl-oss:task:reopen` first)
- reason missing or < 10 chars: `error_category=INVALID_INPUT`
