---
title: E2E lifecycle smoke run
status: pass
last_run: 2026-04-30
binary: hstl-oss (commit dev, version dev)
---

# E2E Lifecycle Smoke Run

A standalone test of the full Sprint / Task / KB lifecycle against a fresh
git workspace using the locally-built `hstl-oss` binary. Confirms the OSS
extraction is functionally complete for a one-task sprint.

## Workspace setup

```bash
SMOKE=/tmp/hstl-oss-smoke
rm -rf "$SMOKE" && mkdir -p "$SMOKE" && cd "$SMOKE"
git init -q
mkdir -p works/tasks works/sprints/{active,backlog,completed} \
         docs/07-knowledge/{mistakes,operations}
git add -A && git commit -q -m "init"
```

## Lifecycle steps verified

| # | Command | Result |
|---|---|---|
| 1 | `hstl-oss version` | returns JSON envelope `{status: ok, ...}`. |
| 2 | `hstl-oss --help` | lists 22 top-level subcommands; out-of-scope subcommands (multi-track, worker orchestration, persona-specific) are absent as expected. |
| 3 | `hstl-oss sprint create --id sprint-01 --title ...` | sprint folder + SPRINT.md created under `works/sprints/backlog/sprint-01/`. |
| 4 | `hstl-oss task create --title ... --summary ... --skip-summary-validation` | T001 created at `works/tasks/T001-smoke-task-1.md`. |
| 5 | task body authored (purpose / requirements / done criteria / result) | placeholder gate cleared. |
| 6 | `hstl-oss task assign T001 --sprint sprint-01` | task moved into the sprint. |
| 7 | `hstl-oss sprint start sprint-01` | sprint folder transitions backlog → active, status=active. |
| 8 | `hstl-oss task start T001` | task transitions todo → in-progress. |
| 9 | `hstl-oss harness check task T001 <item>` × 7 items | harness items recorded; `context_acknowledged` requires `hstl-oss context --ticket <work_ticket>` first to obtain a `sha256:...` evidence. |
| 10 | `hstl-oss task complete T001 --auto-check-go=false` | task transitions in-progress → done. |
| 11 | `hstl-oss sprint progress sprint-01` | reports 1/1 done (100%). |
| 12 | `hstl-oss harness check sprint sprint-01 <phase>` × 12 phases | sprint:default template phases recorded. |
| 13 | `hstl-oss sprint complete sprint-01` | sprint folder transitions active → completed; SPRINT.md + CURRENT-FOCUS.md updated; side_effects emitted. |
| 14 | `hstl-oss kb create --category mistakes --subcategory persistence ...` | KB card M001 written at `docs/07-knowledge/mistakes/persistence.md`. |
| 15 | `hstl-oss kb list` | M001 surfaced in JSON list output. |

## Notes for first-run users

- **Default policies are strict.** Out of the box `task create` enforces the
  summary heuristic (preset=strict). Pass `--skip-summary-validation` for
  CI-style smoke runs or relax via `HOSTLER_OSS_SUMMARY_POLICY=moderate|lenient|off`.
- **Sprint Tier solo gate.** `sprint start` blocks unless the sprint contains
  at least one task. Either assign a task first or pass `--force`.
- **Harness Gate is template-aware.** The available `<item_id>` set differs
  per task type (`task:feature` vs `task:infra` ...). Always run
  `hstl-oss harness get task <id>` to discover valid item IDs before checking.
- **`context_acknowledged` requires a real hash.** Run
  `hstl-oss context --ticket <work_ticket>` to obtain a SHA-256 evidence
  string and submit it as `--evidence sha256:<hex>`.
- **Sprint complete needs every required phase checked.** `sprint:default`
  has 12 phases; missing items return `BLOCKED` with `missing_phases`.

## Status

The OSS extraction is functionally green for the standard one-task sprint
lifecycle. Full smoke coverage including dependency chains, reopen, discard,
and multi-task sprints is left to a richer end-to-end test suite.
