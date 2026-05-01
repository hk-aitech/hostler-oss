---
description: Start a Sprint — backlog -> active transition + briefing + design readiness check + reminders. Use when the user says "start the sprint", "begin sprint N", or to kick off a planned Sprint.
argument-hint: [sprint-id]
allowed-tools: Bash(hstl-oss:*), Bash(git:checkout), Bash(git:commit), Bash(git:status), Bash(git:ls-files), Read, Write, Edit, Glob, Grep
---

# Start Sprint: $ARGUMENTS

> **Command-first entry point**: This Command is the official entry for Sprint start.
> When the user requests Sprint start, do not call the CLI directly — first call this
> Command to run the briefing, design-readiness check, and reminders, then call the
> CLI from inside the Command.
>
> **Branch policy**: Sprint branch points (`dev` / `main`) and merge conventions follow
> the shared template `skills/branch-workflow/references/branch-policy.md`. A project's
> own SSOT takes precedence.

> **Command First STRICT**: This Command is the official entry point for Sprint start.
> Calling `hstl-oss sprint start <ID>` directly via Bash auto-records `ceremony.bypass` /
> `*.retried` events in the audit log and counts as a violation in Phase 5 retro.
> Policy SSOT: `CLAUDE.md` §Command First mapping table.

Starts a Sprint via the CLI.

## CLI usage

```
Bash("hstl-oss sprint start $ARGUMENTS --with-ceremony")
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss sprint start sprint-01 --with-ceremony | jq '.ceremony.briefing'
```

Parse `ceremony.briefing`, `ceremony.design_readiness`, and `ceremony.reminders` from the
CLI JSON response and render them in the briefing format below.

## Output format

When `sprint_start` runs, print the **structured briefing** in this format and then
proceed. Run all checks contiguously and start the Sprint automatically.

```markdown
=============================================
  Sprint {id} kickoff briefing
=============================================

  Title: {title}
  Goal:  {goal}

  Tasks ({N}):
  | # | ID | Title | Size | Priority | Depends |
  |---|-----|------|------|---------|------|
  | 1 | T01 | ... | M | p0 | — |
  | 2 | T02 | ... | S | p1 | T01 |

  Dependency graph:
  T01 -> T02 -> T03
  T04 -> T05
  T06 (independent)

  Size estimate:
  XS x {n} + S x {n} + M x {n} + L x {n} + XL x {n} = {total points}

  Design readiness:
  - Zero placeholder remnants in Task bodies
  - All dependency Tasks done
  - WARN: body under 200 chars (suspected underwriting)
  - WARN: "new vs update" identification missing (must be stated in body)

  Task precondition classification:
  - New (file missing): T02, T05
  - Update (file exists): T01, T03, T04
    -> If recent Sprints touched related files, reconfirm context

  Recommended starting Task: T01 (no dependencies, p0, no preconditions)
=============================================
```

### Reminder output

If the CLI response includes a `reminders` array, print it as a blockquote below the briefing:

```markdown
> **Reminders**
> - {reminder 1}
> - {reminder 2}
```

Reminders are loaded from the `## Reminders` -> `### sprint-start` section of
`{project_root}/.hostler/project-config.md`.

### Design readiness checks

| # | Check | Block level |
|---|------|------|
| 1 | Placeholder remnants in Task bodies | WARN |
| 2 | Unfinished dependency Tasks | INFO |
| 3 | Dev-environment deployment verification Task included | WARN (if missing) |
| 4 | Last Task's `depends_on` includes prior Tasks | WARN |
| 5 | "New vs update" classification per Task | INFO |

**Task precondition classification**:

"New vs update" is determined by:

- **New**: the main file referenced by the Task does not yet exist (per `git ls-files`).
  The Task body should describe what to create.
- **Update**: the main file already exists. The Task body should specify which sections
  to change and why. If a recent Sprint (current Sprint -3 to -1) modified that file,
  reconfirm context for possible conflicts.

The AI performs this classification at Sprint start. If the body lacks an explicit
designation, only print a WARN (not BLOCK). This is a preventive measure derived from retros.

## Internal execution order

1. Read the Sprint SPRINT.md file -> extract briefing data
2. Run the design-readiness checks
3. Print the briefing (per the format above)
4. `Bash("hstl-oss sprint start <id> --with-ceremony")` — backlog -> active folder move + state transition + audit log
5. **Create Sprint-dedicated branch**: `git checkout -b sprint-{id}` (branch from dev)
6. **CURRENT-FOCUS.md auto-update** — `hstl-oss sprint start` automatically creates/updates
   `works/sprints/active/CURRENT-FOCUS.md` to the sprint-active state (`UpdateCurrentFocus`).
   Do not edit with Write/Edit. (CEREMONY.md is deprecated — `harness_items` is the SSOT.)
7. git commit (`chore: switch {sprint_id} to active`)

> **Proceed automatically without user confirmation.** After printing the briefing, immediately call the CLI.
> **Branch policy**: at Sprint start, branch off from `dev` to create `sprint-{id}` and work there.
