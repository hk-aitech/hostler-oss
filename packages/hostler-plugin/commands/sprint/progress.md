---
description: Query Sprint progress — Task done/total + Phase checklist state. Use when the user asks "how is the sprint going?", "what's left in this sprint?", or "show sprint progress".
argument-hint: "<sprint-id>"
allowed-tools: Bash(hstl-oss:*), Read
---

# Sprint Progress: $ARGUMENTS

> **Command-first entry point**

Queries a Sprint's progress and Phase checklist state via the CLI.

## CLI usage

```
Bash("hstl-oss sprint progress <sprint-id>")
```

JSON parsing (jq):
```bash
hstl-oss sprint progress sprint-01 | jq '{
  done: .task_done,
  total: .task_total,
  pct: (.task_done * 100 / .task_total),
  remaining: .tasks[] | select(.status != "done") | .id
}'
```

Returned fields:
- `sprint_id`, `title`, `goal`, `status`
- `task_total`, `task_done`, `task_in_progress`, `task_todo`
- `tasks[]` — Task-level status / type / priority
- `harness_summary` — Sprint-level Harness item progress
- `started_at`, `completed_at`

## Automated steps

| # | Item |
|---|------|
| 1 | DB sprint row + tasks join lookup |
| 2 | Tally Tasks by status |
| 3 | Sprint Harness item progress summary (10-Phase Cascade state) |

## When to use

- Daily standup progress check
- Identify remaining Tasks near completion
- Per-Phase incomplete-item review (e.g. Phase 9 follow-up Task derivation)
- AI brief of Sprint context when recommending the next Task

## Related Commands

- `/hstl-oss:sprint:list` — multi-record listing
- `/hstl-oss:sprint:complete` — completion (10-Phase Cascade)
- `/hstl-oss:task:list` — Task-level listing with `--sprint <id>`

## Output

JSON: `{status: ok, data: {sprint_id, done, total, percent, in_progress, todo, harness: [...]}}` — Task counts + 12-Phase harness_items state.

Errors:
- Sprint not found: `error_category=NOT_FOUND` + recovery_hint (`hstl-oss sprint list`)
