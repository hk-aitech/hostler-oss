# Getting started — 5-minute walkthrough

Goal: from zero to a running Sprint with one Task in five minutes. By the end you'll have a real `.hstl/` state directory, a Sprint folder under `works/sprints/`, and a Task you can start, complete, and inspect.

## Prerequisites

- **Go 1.24+** to build the CLI (the toolchain auto-fetches via `GOTOOLCHAIN=auto` if your local Go is older).
- **git** for project-root detection.
- **Claude Code** (CLI / IDE / web) if you want the plugin side.

The CLI alone is fully functional without Claude Code — the plugin is a layer on top, not a hard dependency.

## 1. Install the CLI

```bash
curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/latest/download/install.sh | sh
```

The installer auto-detects your OS / arch, downloads the pre-built binary, verifies its SHA256, and drops `hstl-oss` into `~/.local/bin/`. Make sure that directory is on your `PATH`.

Verify:

```bash
hstl-oss version
```

You should see a JSON envelope like `{"data":{"version":"…","commit":"…","date":"…"},"status":"ok"}`.

> Alternatives — `go install …@latest` (when you have Go), or `git clone + make install` (when you want to compile locally). See [`installation.md`](installation.md#cli) for the full menu.

## 2. Initialise a project

In any directory you want to manage with hostler-oss:

```bash
mkdir my-project && cd my-project
git init
hstl-oss project init-skeleton
```

`init-skeleton` creates the minimum tree: `.hstl/project-config.yaml`, `works/sprints/{backlog,active,completed}/`, `works/tasks/`, and a `BACKLOG.md`. Files are the source of truth — the SQLite index under `.hstl/` is just a cache.

## 3. Create your first Sprint

```bash
hstl-oss sprint create \
  --id sprint-01 \
  --title "Initial walkthrough" \
  --goal "Verify the hostler-oss lifecycle"
```

This drops a Sprint folder at `works/sprints/backlog/sprint-01/` with a `SPRINT.md` template you can edit.

Start it:

```bash
hstl-oss sprint start sprint-01
```

The folder moves to `works/sprints/active/sprint-01/`.

## 4. Create and run a Task

```bash
hstl-oss task create \
  --title "First task" \
  --type chore \
  --priority p3 \
  --estimate XS \
  --summary "Verify the create→start→complete loop end-to-end"
```

Edit the body of the Task file under `works/tasks/T001-first-task.md` — fill in `## Purpose`, `## Requirements`, `## Done Criteria` with actual content (the strict summary heuristic rejects placeholder bodies on `task start`).

Attach to the Sprint:

```bash
hstl-oss task assign T001 sprint-01
```

Start it:

```bash
hstl-oss task start T001
```

Now do the work. When done, fill in `## Result` (Design Decisions / Artifacts / Verification sub-headings) and:

```bash
hstl-oss task complete T001 --auto-check-criteria
```

`--auto-check-criteria` ticks the `criteria_checked` Harness Gate item if every `## Done Criteria` checkbox is `- [x]`.

## 5. Inspect

```bash
hstl-oss task list
hstl-oss sprint progress sprint-01
hstl-oss harness get task T001
hstl-oss audit list --entity-type task --entity-id T001
```

Every state transition was recorded in the audit log. The Sprint can now be completed:

```bash
hstl-oss sprint complete sprint-01
```

## 6. (Optional) Activate the Claude Code plugin

The plugin lives in `packages/hostler-plugin/`. The fastest install — from inside Claude Code:

```
/plugin marketplace add https://github.com/hk-aitech/hostler-oss
/plugin install hstl-oss@hostler-oss
```

Re-launch Claude Code. The `hstl-oss:` namespace registers ~7 command groups, ~21 skills, 4 agents, and 3 session hooks. See [`installation.md`](installation.md#plugin) for symlink / local-path alternatives.

Now Claude can drive the same lifecycle through commands like `/hstl-oss:task:start T001`, and the skills auto-trigger on relevant prompts.

## Where to next

- [`concepts.md`](concepts.md) — what Sprint / Task / Harness actually mean here.
- [`installation.md`](installation.md) — production install paths, env vars, troubleshooting.
- [`cli-reference.md`](cli-reference.md) — every command at a glance.
- `hstl-oss --help` — authoritative SSOT, regenerated from code.
- `hstl-oss manifest --schema` — full JSON Schema for tool-use / MCP integration.
