# hostler-oss

A **Sprint / Task workflow toolkit for Claude Code**, distributed as a Claude Code plugin paired with a Go CLI. Together they let you run an opinionated, file-based project-management loop — sprints, tasks, harness gating, ADRs, retrospectives, and a knowledge base — directly from Claude Code.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go ≥ 1.24](https://img.shields.io/badge/Go-%E2%89%A5%201.24-00ADD8.svg)](packages/hostler-cli/go.mod)

## What you get

| Component | What it is |
|---|---|
| **`packages/hostler-plugin`** | Claude Code plugin (namespace `hstl-oss:`) with 7 command groups, 20 user-invocable skills, 4 agents, and 3 session hooks. |
| **`packages/hostler-cli`** | Go CLI (`hstl-oss`) that owns the file-based state for sprints, tasks, backlog, registry, audit log, and ceremony artefacts. |

The plugin orchestrates the Claude Code experience. The CLI persists, validates, and gates state transitions.

## Quick start

### 1. Install the CLI

One-line install (no Go required — pre-built binary):

```bash
curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/latest/download/install.sh | sh
hstl-oss version
```

The installer auto-detects OS / arch (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64), verifies the SHA256 against `SHA256SUMS`, and drops `hstl-oss` into `~/.local/bin/`. Set `HSTL_INSTALL_DIR` for a different target.

Other paths (`go install`, build from source) are documented in [`docs/04-guides/installation.md`](docs/04-guides/installation.md).

### 2. Install the plugin

From inside Claude Code:

```
/plugin marketplace add https://github.com/hk-aitech/hostler-oss
/plugin install hstl-oss@hostler-oss
```

The first line registers this repo as a marketplace (it ships `.claude-plugin/marketplace.json` at the root). The second installs the `hstl-oss` plugin — slash commands appear as `/hstl-oss:sprint:create`, `/hstl-oss:task:start`, etc. CLI alternative + symlink-for-development paths are documented in [`docs/04-guides/installation.md`](docs/04-guides/installation.md#plugin).

### 3. Initialize a project

```bash
cd /path/to/your/project
hstl-oss project init
```

This scaffolds the standard `docs/` (00–08 prefixed), `works/sprints/`, `works/tasks/`, and `.hstl-oss/` layout.

### 4. Run the lifecycle

The standard one-task sprint, end to end:

```bash
hstl-oss sprint create --id sprint-01 --title "First sprint" --goal "Ship the smoke loop"
hstl-oss task create --title "First task" --type feature --estimate S --priority p2 --summary "..."
# author the task body — purpose / requirements / done criteria
hstl-oss task assign T001 --sprint sprint-01
hstl-oss sprint start sprint-01
hstl-oss task start T001
# ... do the work ...
hstl-oss task complete T001
hstl-oss sprint complete sprint-01
hstl-oss kb create --category mistakes --subcategory ... --title "..." --problem "..." --solution "..."
```

A complete walk-through of every gate (summary heuristic, sprint tier, harness items, context-ack hash) lives in [`docs/06-reports/spikes/smoke-e2e-lifecycle.md`](docs/06-reports/spikes/smoke-e2e-lifecycle.md).

## Repository layout

```
hostler-oss/
├── packages/
│   ├── hostler-plugin/     # Claude Code plugin (skills, commands, agents, hooks)
│   └── hostler-cli/        # Go CLI: cmd/hstl-oss, pkg/, internal/, migrations/
├── docs/
│   ├── 00-project/         # PDD, roadmap, system overview
│   ├── 02-architecture/    # ADRs (e.g. ADR-001-oss-extraction.md)
│   ├── 06-reports/         # Audits, smoke runs, spikes
│   └── ...                 # 01, 03, 04, 05, 07, 08 prefixed standard sections
├── works/                  # Sprint / task working artefacts (CURRENT-FOCUS, BACKLOG, sprints/)
├── archive/                # Retired material
├── go.work                 # Go workspace
├── LICENSE                 # Apache-2.0
├── NOTICE
└── CONTRIBUTING.md
```

## Documentation

- [`docs/00-project/pdd.md`](docs/00-project/pdd.md) — Project Definition Document (scope, non-goals, success criteria)
- [`docs/00-project/roadmap.md`](docs/00-project/roadmap.md) — milestone plan
- [`docs/02-architecture/ADR-001-oss-extraction.md`](docs/02-architecture/ADR-001-oss-extraction.md) — why the OSS surface is what it is
- [`docs/06-reports/spikes/smoke-e2e-lifecycle.md`](docs/06-reports/spikes/smoke-e2e-lifecycle.md) — verified end-to-end lifecycle run

## Scope

**In scope.** File-based sprint and task lifecycle, harness gating, knowledge-base capture, ADR support, hotfix workflow, audit log, project-structure scaffolding, the `hstl-oss:` plugin surface (commands, skills, agents, hooks).

**Out of scope.** HMAC / GPG document signing, multi-AI orchestration, daemon-based worker coordination, message-bus integrations, universal-feature catalogues, cross-domain trace registries, multi-track program management, faceted project-context layers, persona-specific tooling. These were present in the originating internal toolkit and have been removed deliberately to keep this release focused, auditable, and easy to adopt.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). The short version:

1. Open an MR against `main` from a topic branch.
2. `cd packages/hostler-cli && go build ./... && go test ./...` must be green.
3. Follow the standard `docs/` and `works/` layout (see [`packages/hostler-plugin/skills/project-structure`](packages/hostler-plugin/skills/project-structure)).

## License

Apache License 2.0 — see [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
