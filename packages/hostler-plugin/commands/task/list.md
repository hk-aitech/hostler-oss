---
description: List Tasks — Sprint/status filter with JSON pipeline support. Use when the user asks "show all tasks", "list backlog tasks", or wants to filter Tasks by Sprint/status.
argument-hint: "[--sprint <id>] [--status todo|in-progress|done]"
allowed-tools: Bash(hstl-oss:*), Read
---

# List Tasks: $ARGUMENTS

> **Command-first entry point**

Lists Tasks via the CLI. JSON output supports jq post-processing.

## CLI usage

```
Bash("hstl-oss task list [--sprint <sprint-id>] [--status todo|in-progress|done]")
```

Examples (with jq):
```bash
# Todo Tasks in a Sprint
hstl-oss task list --sprint sprint-01 --status todo | jq '.tasks[] | {id,title}'

# p0 Tasks in the Backlog
hstl-oss task list -q | jq '.tasks[] | select(.priority == "p0" and .sprint == null)'

# Compute Sprint progress
hstl-oss task list --sprint sprint-01 | jq '[.tasks[] | .status] | group_by(.) | map({status: .[0], count: length})'
```

## Hook contract

JSON output schema:
```json
{
  "tasks": [
    {
      "task_id": "T001",
      "title": "...",
      "type": "feature",
      "status": "todo",
      "priority": "p2",
      "estimate": "M",
      "sprint": "sprint-01",
      "depends_on": []
    }
  ],
  "count": 1
}
```

Scripts/Hooks rely on these field names — they form a stable contract. Changes require migration.

## Filter options

| Option | Behavior |
|------|------|
| `--sprint <id>` | Tasks in a specific Sprint (e.g. `sprint-01`) |
| `--sprint backlog` | Sprint-unassigned Tasks only |
| `--status <s>` | Status filter (todo / in-progress / done) |
| no option | All |

## When to use

- Review the backlog before Sprint start
- Track progress during a Sprint (`--sprint <id>`)
- Extract Task ID lists in hooks/scripts
- AI uses it together with `task next` when recommending the next Task

## Related Commands

- `/hstl-oss:task:get` — single-record detail
- `/hstl-oss:task:next` — recommend the next Task
- `/hstl-oss:sprint:complete` — auto-aggregates Sprint done/total counts

## Output

JSON: `{status: ok, data: {tasks: [TaskSummary], count}}` — TaskSummary = `{task_id, title, type, estimate, priority, status, sprint?, depends_on}`.

`count` reflects the result after applying filters (--sprint / --status). Zero results still produce `{count: 0, tasks: []}`.
