---
uuid: 019dd338-7617-79bf-8e51-13c15f6a1f56
title: hostler-cli — Hostler OSS CLI
status: active
created: 2026-04-28
doc_type: readme
---

# hostler-cli — Hostler OSS CLI (Go)

> The Sprint / Task / Harness lifecycle CLI for the hostler-oss workflow toolkit.

This package is the Go implementation of the `hstl-oss` binary — a file-based
Sprint / Task / Harness state machine for AI-collaborative development.

## Overview

- **Language**: Go 1.24+ (workspace 1.25 toolchain compatible)
- **Module path**: `github.com/hk-aitech/hostler-oss/packages/hostler-cli`
- **Binary**: `hstl-oss`
- **Architecture**: 3-layer hexagonal (cmd / pkg / internal) with Port-Adapter
  ports under `internal/ports`.

## Build / install

```bash
cd packages/hostler-cli
go build -o ~/.local/bin/hstl-oss ./cmd/hstl-oss
```

Or via the Makefile:

```bash
cd packages/hostler-cli && make install
```

`make install` builds and copies `hstl-oss` into `~/.local/bin`.

### Verify

```bash
go test ./...               # full unit test suite
go vet ./...                # static analysis
hstl-oss version            # prints the ldflags-injected version
```

## Main commands

Authoritative SSOT: `hstl-oss --help`. JSON Schema output for tool-use /
MCP-style integration: `hstl-oss manifest --schema`.

### Sprint / Task / Harness lifecycle

```bash
hstl-oss sprint create --id sprint-NN --title "..." --goal "..."
hstl-oss sprint start sprint-NN --with-ceremony
hstl-oss sprint complete sprint-NN --with-ceremony

hstl-oss task create --summary "WHAT/WHY/SUCCESS" --type bugfix --estimate XS --priority p1
hstl-oss task start TNNN --with-ceremony
hstl-oss task complete TNNN --with-ceremony

hstl-oss harness get task TNNN
hstl-oss harness check task TNNN <item_id> --evidence "..."
hstl-oss harness auto-check task TNNN  # batch-verify deterministic Go items
```

### KB / ADR / supporting commands

```bash
hstl-oss kb list / search / create / rebuild / seal
hstl-oss adr validate                  # frontmatter status verification
hstl-oss context                       # project state JSON
hstl-oss backlog sync                  # auto-reconcile BACKLOG.md
hstl-oss rules exec <rule_id>          # run a single pre-commit rule
```

## Architecture

### 3-layer (hexagonal)

```
cmd/hstl-oss/cmd/   — CLI entry points (cobra) + per-command handlers
pkg/                — public domain / pure functions / Rule Engine
internal/           — adapters / output formatter / brand constants
```

### Port-Adapter pattern

- **Ports**: `internal/ports` (domain interfaces)
- **FS adapter**: file SSOT for Sprint/Task `.md` and frontmatter
- **SQLite adapter**: secondary index that mirrors file state
- **DB lock**: `internal/dblock` — `flock` based global mutex
- **Brand constants**: `internal/brand`

### Core design principles

- **Files are the SSOT** — SQLite is a secondary index; on conflict the file wins.
- **`os.Rename`-centric transitions** — most state changes are atomic renames.
- **SQLite-less baseline** — even without SQLite the CLI's basic features work.
- **CLI Output Contract** — stdout=JSON / stderr=diagnostic.

## Development guide

### Tooling

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Tests / static analysis

```bash
go test ./... -count=1
go vet ./...
staticcheck ./...
```

### Pre-commit (auto-block)

`hstl-oss rules list` shows the registered rules. Highlights:
- `precommit.kb.reference-format` — block bare KB references
- `precommit.uuid.frontmatter` — require UUID v7 frontmatter
- (and more — consult `hstl-oss rules list`)

### Manifest / Schema output

```bash
hstl-oss --manifest               # full subcommand manifest
hstl-oss --schema                 # draft-07 JSON Schema
hstl-oss <subcommand> --schema    # partial-tree schema
```

## Directory layout

```
packages/hostler-cli/
├── cmd/hstl-oss/cmd/   — cobra command handlers (sprint, task, harness, kb, adr, ...)
├── pkg/                — public domain
│   ├── apperr/         — error category enum
│   ├── audit/          — audit-event hash chain
│   ├── config/         — project-config schema
│   ├── db/             — SQLite schema + harness defaults
│   ├── kb/             — Knowledge Base card CRUD
│   ├── rules/          — pre-commit Rule Engine + registered rules
│   ├── sprint/, task/  — lifecycle domain
│   └── ...             — additional domain packages
├── internal/           — adapters / output / brand
│   ├── adapters/       — fs, sqlite (secondary index)
│   ├── app/            — application service (DI entry point)
│   ├── brand/          — package-level brand constants
│   ├── dblock/         — flock mutex
│   └── output/         — JSON / text / console formatter
├── migrations/         — SQLite schema migrations
├── go.mod              — module definition
└── Makefile            — install / test / tools shortcuts
```

## Operating environment

- **OS**: Linux (aarch64 + x86_64), macOS (Apple Silicon)
- **Go**: 1.24 minimum
- **SQLite**: 3.44+ (used as a secondary index; the CLI degrades gracefully without it)

## License

Apache-2.0 — see the repo-root `LICENSE`.
