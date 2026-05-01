# CLAUDE.md — hostler-oss contributor guide

**Version**: 0.1.0

This file is loaded automatically by Claude Code when the working
directory is inside this repository. Read it once at session start.

## What this repo is

`hostler-oss` ships two halves under one git tree:

- `packages/hostler-cli/` — Go CLI, binary name `hstl-oss`. Owns the
  on-disk Sprint / Task state, audit log, and harness gate.
- `packages/hostler-plugin/` — Claude Code plugin (namespace `hstl-oss:`).
  Slash commands, skills, agents, three session hooks.

The plugin orchestrates the user-facing workflow; the CLI persists and
validates everything. They share the file format defined in `docs/`.

License: Apache-2.0. Public OSS release at `hk-aitech/hostler-oss`.

## Where things live

| Path | What |
|---|---|
| `packages/hostler-cli/cmd/hstl-oss/` | CLI entrypoint and Cobra command tree |
| `packages/hostler-cli/pkg/` | rules, harness, schemas — public-shaped Go packages |
| `packages/hostler-cli/internal/` | implementation details (output, fsutil, …) |
| `packages/hostler-plugin/skills/` | one directory per skill, each with `SKILL.md` |
| `packages/hostler-plugin/commands/` | slash command markdown files |
| `packages/hostler-plugin/hooks/` | three session hooks (POSIX shell) |
| `docs/00-project/` | PDD + roadmap |
| `docs/02-architecture/` | ADRs |
| `docs/04-guides/` | installation, getting started, hands-on lab, CLI reference |
| `docs/07-knowledge/` | KB cards (lessons-learned) |
| `works/{sprints,tasks,worklogs}/` | Sprint / Task working artefacts (greenfield in OSS) |
| `scripts/install.sh` | one-line installer published with each Release |
| `.github/workflows/{ci,release}.yml` | CI on push/PR; Release on `v*` tag |

## Common commands

```bash
# CLI — run from packages/hostler-cli/
make build          # → bin/hstl-oss
make test           # go test ./...
make lint           # vet + staticcheck + golangci-lint (when installed)
make install        # build + copy to ~/.local/bin/hstl-oss
make cross          # ARM64 + AMD64 binaries under bin/
make check-version-sync   # plugin.json / marketplace.json / CLAUDE.md must match
```

```bash
# Reproduce CI checks locally before push
shellcheck scripts/install.sh                                   # strict
shellcheck -e SC2016 packages/hostler-plugin/hooks/*.sh         # SC2016 allowed
python3 -c "import yaml; yaml.safe_load(open('PATH'))"          # YAML validity
```

CI pins `shellcheck 0.9.0` and `Go 1.24` (with `GOTOOLCHAIN=auto` so
the 1.25 toolchain is auto-fetched for `modernc.org/sqlite`). The Go
test step pins `CGO_ENABLED=1` because `-race` requires cgo; build
and vet stay on `CGO_ENABLED=0`.

## Conventions

- **Language**: English only across the entire repo — source, commits,
  PRs, issues, docs, KB cards. This is a public OSS project and
  consistency with the audience matters.
- **Commit messages**: conventional prefix (`fix:`, `chore:`, `ci:`,
  `docs:`, …). Reference any related Task ID (e.g. `T###`) when one
  exists.
- **One Task = one commit** when working through the Sprint workflow.
- **Use `jq` over hand-rolled JSON parsing** in shell.
- **`const` over `var`** in Go where the value is fixed.
- **Command First** — prefer invoking the CLI over editing files by
  hand. The SQLite index is rebuildable via `hstl-oss backlog sync`;
  never lose data by hand-editing.

## Release flow

A release is one action: push a `v*` annotated tag.

```bash
git tag -a v0.1.1 -m "v0.1.1 — <summary>"
git push origin v0.1.1
```

`.github/workflows/release.yml` cross-builds 5 platforms (linux/darwin
× amd64/arm64 + windows/amd64), generates `SHA256SUMS`, attaches
`scripts/install.sh`, and creates a GitHub Release with auto-generated
notes. CI green on `main` is not gated by the workflow itself — a
broken release is hard to retract, so check `gh run list` (or the API)
before tagging.

Version-bump checklist (enforced by `make check-version-sync`):

1. `packages/hostler-plugin/.claude-plugin/plugin.json` `.version`
2. `.claude-plugin/marketplace.json` `.plugins[hstl-oss].version`
3. `CLAUDE.md` `**Version**: …` line at the top of this file

## Things to know

- **Local state lives outside git**: `~/.hostler/data/<project-key>/`
  holds the SQLite index and audit log. The repo's
  `.hstl/project-config.yaml` carries the per-clone project key and is
  gitignored.
- **`works/` is mostly empty** in the public release — dogfood
  artefacts stayed in the prep repo. Treat new Sprint / Task work as
  greenfield.
- **`docs/07-knowledge/`** is the canonical place for lessons learned.
  Use `/hstl-oss:learned` (or `hstl-oss kb create`) rather than
  dropping notes elsewhere.
- **`hstl:_frozen:review:*` skills** are intentionally frozen — don't
  extend them; treat as legacy until a successor lands.
- **Push protection**: GitHub secret-scanning blocks pushes that look
  like real keys. Tests with secret-shaped fixtures (e.g.
  `pkg/rules/samples_commit_test.go`) split high-entropy literals at
  compile time (`"sk_" + "live_" + …`) so the runtime string is intact
  but the scanner doesn't match. Keep that pattern.

## Pointers

- Installation: `docs/04-guides/installation.md`
- Getting started (project init → first Task): `docs/04-guides/getting-started.md`
- Hands-on lab (full lifecycle walkthrough): `docs/04-guides/hands-on-lab.md`
- CLI reference: `docs/04-guides/cli-reference.md`
- Concepts (Sprint, Task, harness gate, audit log): `docs/04-guides/concepts.md`
- Project definition: `docs/00-project/pdd.md`
- ADRs: `docs/02-architecture/`
