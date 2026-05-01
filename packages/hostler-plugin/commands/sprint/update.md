---
description: Update Sprint metadata fields — title / goal (status transitions go through start/complete). Use when the user asks to "rename a sprint", "rewrite the sprint goal", or fix a typo in Sprint metadata.
argument-hint: "<sprint-id> [--title \"...\"] [--goal \"...\"]"
allowed-tools: Bash(hstl-oss:*), Read, Glob
---

# Update Sprint: $ARGUMENTS

> **Command-first entry point**

Updates Sprint metadata fields (title, goal) via the CLI. Status transitions are
performed only by the ceremony Commands (start/complete).

## CLI usage

```
Bash("hstl-oss sprint update <sprint-id> [--title \"...\"] [--goal \"...\"]")
```

Examples:
```bash
# Change title
hstl-oss sprint update sprint-01 --title "Sprint/Task operations consolidation"

# Change goal
hstl-oss sprint update sprint-01 --goal "Address missing Commands, document automatic behavior, run Dev regression"

# Update both at once
hstl-oss sprint update sprint-01 --title "..." --goal "..."
```

## Automated steps

| # | Item |
|---|------|
| 1 | Atomic DB sprint row update |
| 2 | SPRINT.md frontmatter sync (no Write/Edit) |
| 3 | Auto-update of BACKLOG.md / CURRENT-FOCUS.md |
| 4 | Audit log (`sprint.updated` + changed fields) |

## When to use

- Adjust title/goal after Sprint start due to scope changes
- Refine vague goal phrasing during retro
- Fix typos

## Caveats

- **Cannot change status** — use `/hstl-oss:sprint:start` or `/hstl-oss:sprint:complete`
- **Cannot change id** — sprint id is bound to the folder name and DB key
- **Cannot change Task list** — use Task assign/unassign separately
- **Do not edit SPRINT.md frontmatter with Write/Edit** — causes DB drift

## Related Commands

- `/hstl-oss:sprint:list` — list all
- `/hstl-oss:sprint:progress` — progress + Task details
- `/hstl-oss:sprint:create` — new creation (initial title/goal)
- `/hstl-oss:sprint:start` / `/hstl-oss:sprint:complete` — status transitions

## Output

JSON: `{status: ok, data: {sprint_id, updated_fields: [...], previous_values: {...}}}` — updated field list + prior values (for audit).

Errors:
- Sprint not found: `error_category=NOT_FOUND`
- Direct status update attempt: `error_category=REJECTED` (status is start/complete only)
- Both title/goal missing: `error_category=INVALID_INPUT`
