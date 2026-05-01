---
description: Start a Task — Task context briefing + reminders. Use when the user says "start task T01", "begin work", or to kick off the next Task.
argument-hint: [task-id]
allowed-tools: Bash(hstl-oss:*), Bash(git:status), Read, Glob, Grep
---

# Start Task: $ARGUMENTS

> **Command-first entry point**: This Command is the official entry for Task start.
> When the user says "start task" or "begin the work", do not call the CLI directly —
> first call this Command to run the briefing, reminders, and automatic checks, then
> call the CLI from inside the Command.

> **Command First STRICT**: This Command is a required workflow entry point.
> Calling `hstl-oss task start <ID>` directly via Bash auto-records `ceremony.bypass` /
> `*.retried` events in the audit log and counts as a violation in Phase 5 retro.
> Policy SSOT: `CLAUDE.md` §Command First mapping table. Bypass only with explicit user
> approval and `HOSTLER_OSS_*_POLICY=warn|off`.

Starts a Task via the CLI.

## CLI usage

```
Bash("hstl-oss task start $ARGUMENTS --with-ceremony")
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss task start T01 --with-ceremony | jq '.ceremony.briefing'
```

Parse `ceremony.briefing` and `ceremony.reminders` from the CLI JSON response and
render them in the briefing format below.

## Output format

When `task_start` runs, **brief the Task context** in the format below. Switch
immediately without confirmation and start the work.

```markdown
=============================================
  Task {id} start
=============================================

  Title:     {title}
  Type:      {type} -> commit prefix: {feat:|fix:|docs:|...}
  Size:      {estimate}
  Priority:  {priority}
  Sprint:    {sprint_id}

  Purpose:
  {first 2 lines of the "Purpose" section in the Task file}

  Requirements: {N}
  - [ ] {first requirement}
  - [ ] {second}
  ...

  Completion criteria: {N}
  - [ ] {first criterion}
  ...

  Dependency Tasks:
  {depends_on} -> all done OK / WARN {unfinished IDs}

  Related files (estimated):
  - src/...
  - tests/...

  Starting work.
=============================================
```

### Reminder output

If the CLI response includes a `reminders` array, print it as a blockquote below the briefing:

```markdown
> **Reminders**
> - 1 Task = 1 Commit — commit once at Task completion, no intermediate commits
> - task:complete procedure required
> - ...
```

Reminders are loaded from the `## Reminders` -> `### task-start` section of
`{project_root}/.hostler/project-config.md`.

### Pre-start checks

| # | Check | Block level |
|---|------|------|
| 1 | Placeholder remnants in Task body | BLOCK (strict) / WARN (warn) |
| 2 | Unfinished dependency Tasks | BLOCK |
| 3 | Another Task already in-progress | WARN (concurrent work caution) |

### Agent mapping

| Task type | Owning agent |
|-----------|--------------|
| Feature implementation, bug fix | developer |
| Infrastructure, Docker, CI/CD | devops |
| Documentation, guides | tech-writer |
| Test writing, quality verification | qa-engineer |
| Design, architecture review | architect |

### Auto-attach DB schema (opt-in)

**Background**: A historical mistake had SQL written against an assumed schema
(`is_active = TRUE`) when the column did not actually exist, resulting in a
LocalDev 500 + a wasted image rebuild. When the Task body contains DB-schema
change keywords (ALTER/CREATE/DROP TABLE / ADD/DROP COLUMN), the task:start
briefing auto-attaches the LocalDev psql `\d` result to prevent the same mistake.

**Opt-in activation** (off by default — enable for DB-backed projects):

```bash
export HOSTLER_OSS_TASK_START_DB_SCHEMA_ATTACH=true
export HOSTLER_OSS_LOCALDEV_DSN="postgres://user:pass@localhost:5432/db"
```

You may also inject env via `.hostler/project-config.yaml` (per-project convention;
this Command only reads the env vars).

**AI instruction** — when opt-in is active, run this immediately after Reading the Task file:

```bash
bash packages/hostler-plugin/commands/task/scripts/attach_db_schema.sh "<task-file-path>"
```

If the script produces output, append it as-is below the "Related files (estimated)"
section. If output is empty (no keywords detected or DSN not set), keep the briefing as-is.

**Security rules**:

- **Read-only SQL only** — whitelist of `\d` / `\dt` / `\dv`. Arbitrary SQL is blocked
  by regex inside the `run_psql_meta` function.
- **Table-name sanitization** — only ASCII alphanumerics + underscore
  (`^[A-Za-z_][A-Za-z0-9_]{0,62}$`). Unsafe input is WARN + skip.
- **Graceful degradation** — psql missing / DSN unset / connection failure all yield
  a single stderr WARN line + normal exit 0 (briefing flow preserved).

**Self-test** (regression check after script changes):

```bash
bash packages/hostler-plugin/commands/task/scripts/attach_db_schema.sh --self-test
```

**Scope limit**: only PostgreSQL psql is supported for now. MySQL/SQLite extensions
and migration-SQL file scanning are for a future Sprint.
