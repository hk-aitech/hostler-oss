---
description: Unassign a Task from a Sprint (Sprint -> BACKLOG), auto-restores file location. Use when the user says "remove task from sprint", "move it back to backlog", or wants to defer a Task.
argument-hint: "<task-id>"
allowed-tools: Bash(hstl-oss:*), Read, Glob
---

# Unassign Task: $ARGUMENTS

> **Command-first entry point**

Returns a Sprint-assigned Task to the Backlog via the CLI.

## CLI usage

```
Bash("hstl-oss task unassign <task-id>")
```

## Automated steps

| # | Item | Behavior |
|---|------|------|
| 1 | DB sprint column clear | `tasks.sprint_id = NULL` |
| 2 | File restore | `works/sprints/<sprint>/tasks/T*.md` -> `works/tasks/T*.md` |
| 3 | Git tracking | `git mv` preferred, `os.Rename` as fallback |
| 4 | BACKLOG.md update | Auto — clear the Sprint column |
| 5 | CURRENT-FOCUS.md update | Reflected automatically when the Sprint was active |
| 6 | Audit log | `task.unassigned` event |

## When to use

- Sprint scope rebalance — Sprint is too big and some Tasks should slip to the next Sprint
- Demote a Task to backlog after priority re-evaluation
- Restore a mistakenly assigned Task

## Caveats

- **Cannot restore done Tasks** — completed Tasks' sprint column is locked (history preservation)
- **In-progress Tasks** — can be unassigned but it affects Sprint progress. Recommended:
  use `task update` to revert status to todo, then unassign
- **Do not edit the frontmatter sprint field with Write/Edit**

## Related Commands

- `/hstl-oss:task:assign` — opposite direction (Backlog -> Sprint)
- `/hstl-oss:task:delete` — permanent delete (use unassign if you only want to remove from a Sprint)

## Output

JSON: `{status: ok, data: {task_id, file_path, previous_sprint, previous_path}}` — Sprint -> Backlog restore result.

Errors:
- Task not found: `error_category=NOT_FOUND`
- Already in Backlog (Sprint-unassigned): `error_category=CONFLICT`
- status != todo (in-progress / done): `error_category=REJECTED` (active tasks cannot be unassigned)
