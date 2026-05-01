# 04 — Guides

User-facing material for installing, learning, and operating hostler-oss.

## Onboarding (read in order)

| File | What it covers |
|---|---|
| [`getting-started.md`](getting-started.md) | Five-minute walkthrough — clone → build → first Sprint + Task → complete loop. |
| [`concepts.md`](concepts.md) | Sprint / Task / Harness primitives explained on one page. |
| [`installation.md`](installation.md) | Production install paths, env vars, config, troubleshooting. |
| [`cli-reference.md`](cli-reference.md) | Flat command + subcommand index with output / exit-code contract. |
| [`hands-on-lab.md`](hands-on-lab.md) | 60–90 min guided lab: take a brand-new idea through research, design, and a complete Sprint to a working SvelteKit app. |

## Authoritative SSOT

When this page disagrees with the CLI, the CLI wins:

```bash
hstl-oss --help
hstl-oss <command> --help
hstl-oss --manifest        # JSON tree of every command
hstl-oss --schema          # draft-07 schema
```

## What lives elsewhere

- `02-architecture/` — ADRs and architecture decisions.
- `08-references/standards/` — formal contracts (CLI output, manifest schema, KB reference format).
- `07-knowledge/` — accumulated lessons / mistakes / patterns (KB cards).
- `06-reports/` — spike notes and operational reports.
