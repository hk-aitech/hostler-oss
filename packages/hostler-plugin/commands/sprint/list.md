---
description: List Sprints — status filter, JSON pipeline support. Use when the user asks "what sprints exist?", "show me the backlog sprints", or wants to filter by Sprint status.
argument-hint: "[--status active|completed|backlog]"
allowed-tools: Bash(hstl-oss:*), Read
---

# List Sprints: $ARGUMENTS

> **Command-first entry point**

Lists Sprints via the CLI. Supports JSON output for jq post-processing.

## CLI usage

```
Bash("hstl-oss sprint list [--status active|completed|backlog]")
```

Examples:
```bash
# Active sprints only
hstl-oss sprint list --status active | jq '.sprints[] | {id, title, goal}'

# Completed sprints with progress
hstl-oss sprint list --status completed | jq '.sprints[] | {id, done: .task_done, total: .task_total}'

# Total sprint count
hstl-oss sprint list | jq '.sprints | length'
```

## Hook contract

JSON output schema:
```json
{
  "sprints": [
    {
      "sprint_id": "sprint-01",
      "title": "...",
      "goal": "...",
      "status": "active",
      "task_total": 9,
      "task_done": 7,
      "started_at": "...",
      "completed_at": null
    }
  ],
  "count": 1
}
```

## Filter options

| Option | Behavior |
|------|------|
| `--status active` | In-progress Sprints (works/sprints/active/) |
| `--status backlog` | Waiting Sprints (works/sprints/backlog/) |
| `--status completed` | Completed Sprints (works/sprints/completed/) |
| no option | All |

## When to use

- Check the backlog before deciding the next Sprint
- Aggregate completed-Sprint statistics for retro/audit
- Quick view of active Sprint progress

## Related Commands

- `/hstl-oss:sprint:progress` — detailed progress for a single Sprint
- `/hstl-oss:sprint:start` — backlog -> active transition
- `/hstl-oss:sprint:complete` — active -> completed transition (10-Phase Cascade)

## Output

JSON: `{status: ok, data: {sprints: [SprintRecord], count}}` — SprintRecord = `{sprint_id, title, status, folder_path, started_at, completed_at, goal, track_id?}` (track_id is omitempty).

Filter/sort results return the same envelope; only `count` reflects the filter. Zero results still produce `{count: 0, sprints: []}`.
