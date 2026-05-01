---
description: Partially update Task metadata — title/type/priority/estimate/depends_on. Use when the user says "change task priority", "update the estimate", "rename a task", or any non-status metadata change.
argument-hint: "<task-id> [--title ...] [--type ...] [--priority ...] [--estimate ...]"
allowed-tools: Bash(hstl-oss:*), Read, Glob
---

# Update Task: $ARGUMENTS

> **Command-first entry point**

Partially updates Task metadata via the CLI. Status transitions go through the
ceremony Commands (start/complete/reopen).

## CLI usage

```
Bash("hstl-oss task update <task-id> [flags]")
```

Supported flags:
| Flag | Description |
|--------|------|
| `--title <text>` | Change Task title |
| `--type <type>` | feature/bugfix/refactor/infra/docs/test/chore/spike/hotfix |
| `--priority <p0\|p1\|p2\|p3>` | Priority |
| `--estimate <XS\|S\|M\|L\|XL>` | Expected size (alias: `/hstl-oss:task:estimate`) |
| `--depends-on <id1,id2>` | Dependency Task ID list (full replace) |

Multiple flags can be applied at once:
```bash
hstl-oss task update T01 --priority p1 --estimate L
```

## Automated steps

| # | Item |
|---|------|
| 1 | Atomic DB row update |
| 2 | Frontmatter sync (no Write/Edit) |
| 3 | BACKLOG.md / CURRENT-FOCUS.md update |
| 4 | Audit log (`task.updated` + changed fields) |

## When to use

- Priority re-evaluation
- Re-estimation (S -> M, etc.)
- Add/remove dependencies
- Correct a wrong type (note: changing type also changes Harness Gate items, so be careful)

## Caveats

- **Cannot change status** — use the `start` / `complete` / `reopen` ceremony commands
- **When type changes** Harness Gate items shift — already-checked items are preserved
  but new required items are added as unchecked
- **depends_on changes are full replace** — not add/remove; the new list overrides
- **Do not edit frontmatter with Write/Edit** — causes DB drift

## Related Commands

- `/hstl-oss:task:get` — current state
- `/hstl-oss:task:estimate` — `--estimate` alias entry
- `/hstl-oss:task:start` / `/hstl-oss:task:complete` / `/hstl-oss:task:reopen` — status transitions

## Output

JSON: `{status: ok, data: {task_id, updated_fields: [...], previous_values: {...}}}` — updated fields + prior values.

Errors:
- Task not found: `error_category=NOT_FOUND`
- Direct status update attempt: `error_category=REJECTED` (status is start/complete/reopen only)
- Zero changed fields: `error_category=INVALID_INPUT`
