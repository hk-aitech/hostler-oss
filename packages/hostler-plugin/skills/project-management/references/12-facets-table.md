---
uuid: 019dc434-eb65-7564-879c-d1e466a48b4f
title: "12 Core Facet definitions — 5W1H axes + Layer + source path"
type: reference
related_skill: context-engineering
---

# 12 Core Facet definitions

> Exact definitions of the 12 core facets. The detailed expansion of the SKILL.md body's compact table (`Layer 1 Stable Core 7 / Dynamic 5 / On-Demand 3+`).

## Table of contents

1. Layer 1 — Stable Core (7 facets)
2. Layer 2 — Dynamic Context (5 facets)
3. Layer 3 — On-Demand Resource (3+ facets, plugin)
4. 5W1H axis mapping summary
5. source path conventions (mapping on the user project side)

## 1. Layer 1 — Stable Core (7 facets)

Auto-loaded at session start. Change rate: months to years. Referenced in every
session and every action.

| facet | 5W1H axis | Meaning | Standard source path (convention) |
|-------|-----------|---------|-----------------------------------|
| `core.objective` | WHY | Why the project exists | `docs/00-project/pdd.md` or a vision/mission doc |
| `core.domain` | WHAT-structural | Domain / Bounded Context structure | a domain-model ADR or a domain-model doc |
| `core.environment` | WHERE-deploy | Deployment environment (LocalDev/Dev/Prod) | Task frontmatter `environment` field, or `.env` switching |
| `core.persona` | WHO | User / AI-pair identity | `CLAUDE.md` (or an equivalent pair-definition doc) |
| `core.policy` | HOW-constrained | Constraints / mandatory conventions (commit format, forbidden actions, etc.) | `CLAUDE.md` (policy section) or `docs/08-references/standards/*.md` |
| `core.roadmap` | WHEN-plan | Time plan | `docs/00-project/roadmap.md` |
| `core.knowledge_index` | WHAT-learned | Knowledge index (KB card list) | `docs/07-knowledge/INDEX.md` |

## 2. Layer 2 — Dynamic Context (5 facets)

Refreshed at Task time. Change rate: days to weeks. Referenced on every Task
invocation.

| facet | 5W1H axis | Meaning | Standard source path (convention) |
|-------|-----------|---------|-----------------------------------|
| `core.milestone` | WHAT-outcome | Milestone deliverable (a unit of goal achievement) | the milestone section of `docs/00-project/roadmap.md` |
| `core.track` | WHERE-execution | Execution unit (a topical bundle, multiple Sprints) | `subprocess: hstl-oss track current -o json` |
| `core.sprint` | HOW-batch | A 1–3 day bundle | `subprocess: hstl-oss sprint list --status active -o json` |
| `core.task` | HOW-atomic | A single unit of work | Task frontmatter (`task-frontmatter` resolver) |
| `core.market_state` | WHEN-external | External state (domain-specific — market / traffic / season, etc.) | `plugin:market-state` (delegatable to a Layer 3 plugin) |

## 3. Layer 3 — On-Demand Resource (3+ facets, plugin)

JIT-loaded. Change rate: minutes to days. Called when needed. **Defined as
plugins** — the user project registers its own domain facets.

| facet pattern | Meaning | Examples |
|---------------|---------|----------|
| `plugin.<project>.<name>` | The project's own domain facet | `plugin.<project>.<custom_facet>` (e.g. `broker_api` / `market_hours` for a trading app, `dev_server_state` for a web app) |

Plugin transport: stdio JSON-RPC v2.0. The user project registers by
authoring `.hstl-oss/plugins/<name>/manifest.yaml + entry executable`.

Details: `domain-facet-creation.md`.

## 4. 5W1H axis mapping summary

| Axis | Facet |
|------|-------|
| **WHY** | core.objective |
| **WHAT-structural** | core.domain |
| **WHAT-outcome** | core.milestone |
| **WHAT-learned** | core.knowledge_index |
| **WHO** | core.persona |
| **WHERE-deploy** | core.environment |
| **WHERE-execution** | core.track |
| **WHEN-plan** | core.roadmap |
| **WHEN-external** | core.market_state |
| **HOW-constrained** | core.policy |
| **HOW-batch** | core.sprint |
| **HOW-atomic** | core.task |

The 5W1H 12-axis mapping is 1:1 with the 12 core facets (Layer 3 plugin facets
are domain-specific and unrelated to 5W1H).

## 5. source path conventions (mapping on the user project side)

When the user project imports hostler-plugin, the source path of each of the
7 stable-core facets above varies with the project's docs/ structure. The
standard defaults (e.g. `core.objective` → `docs/00-project/pdd.md`) are
provided by `CoreDefaultDefinitions()`, but if the project uses its own
paths, override them in the `project.facets` section of
`.hstl-oss/project-config.yaml`.

```yaml
project:
  facets:
    core.objective:
      source: "file:my-vision/mission.md"   # custom path instead of default
      layer: 1
```

source forms (resolver prefixes):
- `file:<path>` — read the file as-is
- `subprocess:<command>` — stdout from running a shell command
- `task-frontmatter[.<field>]` — active Task frontmatter (Task-time only)
- `plugin:<plugin-name>` — Layer 3 plugin call (stdio JSON-RPC)

Detailed schema + resolver: see hostler-plugin's facet-spec output (`docs/08-references/standards/facet-spec.md` in the user project).
