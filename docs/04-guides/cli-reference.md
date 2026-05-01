# CLI reference

Authoritative SSOT: `hstl-oss --help` and `hstl-oss <command> --help`. This page is a flat index of what's available so you can find the right entry point without running `--help` repeatedly. For machine-readable use, `hstl-oss --manifest` and `hstl-oss <subtree> --schema` emit JSON Schema (draft-07) that tools can bind to.

## Top-level commands

| Group | What it does |
|---|---|
| `task`     | Task lifecycle (list / get / create / start / complete / delete / reopen / assign / unassign / next / update / checkpoint) |
| `sprint`   | Sprint lifecycle (list / create / start / complete / discard / progress / update / reconcile) |
| `kb`       | Knowledge Base cards (list / create / rebuild / search) |
| `harness`  | Harness Gate inspection + check (get / check / check-all / auto-check) |
| `audit`    | Query the audit log + restore from git log |
| `backlog`  | BACKLOG.md sync + drift detection |
| `config`   | project-config.yaml check / fix / migrate-mode / show-effective |
| `rules`    | Rule engine — list / explain / effective / validate / run |
| `adr`      | ADR frontmatter validation |
| `registry` | Resource registry (list / allocate) |
| `gate-log` | Gate-judgement log (observability) |
| `doc-review` | Detect stale `` `path` `` references and placeholder bodies |
| `ceremony` | Helper for sprint/task start/complete ceremonies |
| `brief`    | Project briefing for human consumption |
| `context`  | Compact AI/subagent context (≈200 tokens) |
| `manifest` | Self-introspection (subcommand tree + JSON Schema) |
| `hooks`    | git hook install / status |
| `project`  | Project-level subcommands (init-skeleton, init-docs, …) |
| `version`  | Print version JSON envelope |

## Task subcommands

| Subcommand | Purpose |
|---|---|
| `task create`        | Create a Task. Required: `--title`, `--type`, `--priority`, `--estimate`, `--summary`. Optional: `--depends`, `--with-ceremony`. |
| `task list`          | List Tasks. Filters: `--sprint`, `--status`, `--priority`, `--type`, `--since`, `--until`, `--last N`, `--limit N`. |
| `task get <id>`      | Show a single Task as JSON / text / console. |
| `task next`          | Recommend the next Task to work on (priority + drift-aware). |
| `task start <id>`    | `todo` → `in-progress`. Flags: `--with-ceremony`, `--dry-run`. |
| `task complete <id>` | `in-progress` → `done`. Flags: `--with-ceremony`, `--dry-run`, `--auto-check-criteria`, `--interactive`. |
| `task reopen <id>`   | `done` / `in-progress` → previous step. Required: `--reason`. |
| `task delete <id>`   | Remove a `todo` Task. Required: `--reason`. |
| `task assign <ids> <sprint>` | Move Task(s) from backlog into a Sprint. |
| `task unassign <ids>` | Detach Task(s) back to backlog. |
| `task update <id>`   | Partial frontmatter update (`--title`, `--type`, `--priority`, `--estimate`, `--depends`). |
| `task checkpoint <id>` | Re-issue a work ticket mid-task. |

## Sprint subcommands

| Subcommand | Purpose |
|---|---|
| `sprint create`       | Required: `--id`, `--title`, `--goal`. Drops a folder under `works/sprints/backlog/`. |
| `sprint list`         | Filter by `--status`. |
| `sprint start <id>`   | `backlog` → `active`. Flag: `--with-ceremony`. |
| `sprint complete <id>`| `active` → `completed`. Flag: `--with-ceremony`. |
| `sprint discard <id>` | Mark a Sprint as discarded (scope dup / stalled / cancelled). Required: `--reason`. |
| `sprint progress <id>`| Aggregate Task progress + completion rate. |
| `sprint update <id>`  | Edit Sprint fields (`--title`, `--goal`, `--status`). |
| `sprint reconcile <id>`| Re-sync Sprint DB row to filesystem truth. |

## KB subcommands

| Subcommand | Purpose |
|---|---|
| `kb create`  | Create a KB card from input flags. |
| `kb list`    | List cards (optionally by category). |
| `kb search`  | Full-text search across title + body. |
| `kb rebuild` | Rebuild the KB index from files. |

## Harness subcommands

| Subcommand | Purpose |
|---|---|
| `harness get <task|sprint> <id>`        | Show every item with required / done state. |
| `harness check <task|sprint> <id> <item>` | Mark a single item done. Required: `--evidence`. |
| `harness check-all <task|sprint> <id>`    | Mark every unchecked item done (with caution). |
| `harness auto-check <task|sprint> <id>`   | Run Go-side deterministic items (`build_passed` / `tests_passed` / `lint_passed`) and check matching items. |

## Output contract

Every command emits a stable JSON envelope on stdout:

```json
{
  "status": "ok" | "error",
  "data": {...},
  "message": "..."
}
```

`stderr` carries diagnostics (warnings, info logs, prompt-style nudges). Tooling that pipes JSON should always read stdout only.

Output formats (`-o`):

- `-o json` (default) — single-line or pretty JSON envelope.
- `-o text` — human-friendly plain text.
- `-o console` — colorised tables (only when stdout is a TTY).

`-q` / `--quiet` suppresses the warning sidecars on stderr.

## Manifest / Schema

```bash
hstl-oss --manifest                    # full tree manifest
hstl-oss --schema                      # draft-07 JSON Schema for the whole CLI
hstl-oss task --manifest               # task subtree only
hstl-oss task start --schema           # single-leaf JSON Schema
hstl-oss manifest validate <file.json> # drift detection between cached + live
```

Use these to wire `hstl-oss` into MCP servers, Claude tool-use definitions, or shell-completion generators.

## Exit codes

| Code | Meaning |
|---|---|
| `0`  | Success |
| `1`  | Generic error (most failure paths) |
| `2`  | Invalid usage (cobra-level argument / flag error) |

## Where things live

| What | Path |
|---|---|
| Project state          | `<repo>/.hstl/project-config.yaml` + `<repo>/works/` |
| SQLite index           | `~/.hostler/data/<project-key>/hstl.db` |
| Audit log              | `~/.hostler/data/<project-key>/audit.jsonl` (append-only, hash-chained) |
| Local CLI binary       | `~/.local/bin/hstl-oss` |
| Plugin                 | `<repo>/packages/hostler-plugin/` |

## Useful one-liners

```bash
# What Task should I work on now?
hstl-oss task next

# What Tasks are blocked?
hstl-oss task list --status blocked -o json -q | jq '.data.tasks[] | .task_id'

# Show drift between DB index and files
hstl-oss backlog sync --dry-run

# Compact context for a subagent prompt
hstl-oss context

# Project briefing for stand-up
hstl-oss brief
```

For anything not listed here, `hstl-oss <command> --help` is always the source of truth.
