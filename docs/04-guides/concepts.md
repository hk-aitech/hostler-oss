# Concepts — Sprint / Task / Harness in one page

hostler-oss is a **file-based, SQLite-indexed lifecycle state machine** for AI-collaborative development. Three primitives carry every workflow.

## Task

A unit of work. One Markdown file under `works/tasks/T###-slug.md` (or, when assigned, under `works/sprints/<sprint>/tasks/`). The frontmatter holds machine state (status, priority, estimate, depends_on); the body holds human intent (Purpose, Requirements, Done Criteria, Type, References) and post-completion records (Result → Design Decisions / Artifacts / Verification).

State machine:

```
todo  ──start──▶  in-progress  ──complete──▶  done
  ▲                   │
  └────reopen─────────┘
```

Every transition writes to the audit log and is gated by Harness items (see below). Tasks are file-SSOT — if the SQLite index disagrees, the file wins.

## Sprint

A time-boxed bundle of Tasks. One folder under `works/sprints/{backlog,active,completed}/<sprint-id>/` with a `SPRINT.md` (planning doc) and a `tasks/` subfolder.

State machine:

```
backlog  ──start──▶  active  ──complete──▶  completed
```

A Sprint moves between three top-level folders as state changes — `os.Rename`-driven, atomic. The folder itself is the state record.

## Harness Gate

A checklist that gates lifecycle transitions. Every `task start` / `task complete` / `sprint start` / `sprint complete` consults a template (`task:feature`, `sprint:default`, etc.) that defines required + optional items (`criteria_checked`, `build_passed`, `tests_passed`, `phase5_retro`, …).

Items can be:

- **deterministic** — auto-checked by the Go side (e.g. `build_passed` runs `go build ./...`).
- **subjective** — human (or AI agent) confirms via `hstl-oss harness check task T001 <item> --evidence "<commit-sha>"`.

`task complete --auto-check-criteria` ticks every deterministic item plus `criteria_checked` if every `- [ ]` in `## Done Criteria` became `- [x]`. Required items that stay open BLOCK the transition.

## Why files-first?

- Diff-friendly. Every state change is a Markdown patch reviewable on GitHub.
- Branch / worktree / merge work naturally on Sprint state.
- The SQLite index is rebuilt from files via `hstl-oss backlog sync`, never the reverse.
- AI agents can read the state without an API.

## What the plugin layer adds

The `hstl-oss:` Claude Code plugin wires the CLI into Claude:

- **Skills** auto-trigger on relevant prompts ("write a retro" → `retro` skill).
- **Commands** (`/hstl-oss:task:start T001`) provide first-class lifecycle entry points.
- **Agents** orchestrate multi-step work (developer / qa-engineer / architect / tech-writer).
- **Session hooks** inject project context at session start so subagents know the active Sprint/Task without explicit briefing.

The plugin is optional — the CLI alone covers every state transition.

## Knowledge Base, ADR, Audit log

Three supporting persistence layers:

- **KB** (`hstl-oss kb`) — capture lessons, rebuild an index. Cards live under `docs/07-knowledge/` and link back to Tasks.
- **ADR** (`hstl-oss adr`) — Architecture Decision Records under `docs/02-architecture/ADR-NNN-*.md`. Frontmatter status field is validated.
- **Audit log** (`hstl-oss audit`) — every state change with hash chain (tamper evidence). Stored under `~/.hostler/data/<project-key>/`.

## Reading order

- [`getting-started.md`](getting-started.md) — five-minute hands-on walk.
- [`installation.md`](installation.md) — install paths, env vars.
- [`cli-reference.md`](cli-reference.md) — flat command map.
- `hstl-oss --help` and `hstl-oss <cmd> --help` — authoritative.
