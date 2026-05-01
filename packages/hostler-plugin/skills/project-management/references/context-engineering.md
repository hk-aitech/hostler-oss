---
uuid: 019dc434-da16-71e7-8ae6-77d34e1f22d8
title: "Context Engineering — Project Context 12-Facet × 3-Layer Model (concept and design intent)"
type: reference
parent_skill: project-management
created: 2026-04-25
updated: 2026-04-25
---

# Context Engineering — Project Context 12-Facet × 3-Layer Model

> **Project Context isn't a single linear hierarchy (Objective → Roadmap → Sprint → Task) — it's an M:N grid of 12 core facets × 3 Layers** (M:N meaning: a single facet applies to many Tasks, and a single Task references many facets at once).
>
> This reference explains the **concept and design intent** behind the shared OSS standard. For usage, see the parent skill (`project-management`) body + the `/hstl-oss:project:context` command. For self-registering custom facets, see `domain-facet-creation.md`.
>
> **History**: this reference was originally a standalone skill
> (`context-engineering`) but was demoted to a parent-skill reference to
> mitigate skill inflation and reduce UX friction. Details: see the
> maintainer change history (`changelog.md`).

## Concept

The linear-hierarchy intuition — Objective → Roadmap → Milestone → Track → Sprint → Task — does not adequately express the context that should be injected into the AI. Roadmap / Milestone / Track sit on the same horizontal line but are different axes (WHEN / WHAT-outcome / WHERE-execution), and Persona / Environment / Knowledge are axes orthogonal to the hierarchy above.

This model formalizes that fact via two principles:

1. **Make the 12 core facets explicit** — promote previously implicit facets (Environment / Market-state / Knowledge / Persona / Quota / Policy, etc.) to first-class citizens. The number 12 maps 1:1 to the 5W1H 12 branches (WHY / WHAT-structural / WHAT-outcome / WHAT-learned / WHO / WHERE-deploy / WHERE-execution / WHEN-plan / WHEN-external / HOW-constrained / HOW-batch / HOW-atomic) — not arbitrary (`references/12-facets-table.md` §4).
2. **3-Layer separation** — split facets into Stable Core (months–years) / Dynamic (days–weeks) / On-Demand (minutes–days) using a change-rate × reference-frequency matrix. Refreshing every facet on every call wastes tokens and time (detailed rationale: `references/3-layer-rationale.md`).

## Why this exists (background)

After one user project in production demonstrated 6 incidents caused by missing facets, this model was proposed and accepted upstream → Tier 1 (Hostler CLI) Phase 1~3 implemented; Phase 4 (JIT optimization) remains.

Representative incident patterns:
- Persona facet missing → AI answered in a generic SW-developer tone instead of the domain tone
- Environment facet not specified → assumed LocalDev SQL, leading to a Prod migration failure
- Knowledge facet under-compressed → AI inferred stale code as SSOT

The linear hierarchy alone provides no slot to express or inject those facets. For concrete cases (which project / Sprint number), see `references/domain-facet-creation.md` §5 dogfood section.

## 12 Core Facets at a glance

| Layer | # facets | Change rate | Reference frequency | Loading time |
|-------|----------|-------------|---------------------|--------------|
| **Layer 1 Stable Core** | 7 | months–years | every session | auto via SessionStart hook |
| **Layer 2 Dynamic** | 5 | days–weeks | every Task | refreshed at Task time |
| **Layer 3 On-Demand** | 3+ (plugin) | minutes–days | as needed | JIT |

Exact name / 5W1H axis / source path of each facet: `references/12-facets-table.md`.

## 3 Layer separation — why three layers

Reading every facet again on every call wastes tokens and time. Facets with different change rates can't share refresh cadence. Stable Core is loaded once and reused; Dynamic refreshes at Task time; On-Demand uses a short TTL cache.

Separation rationale + efficiency matrix + anti-patterns: `references/3-layer-rationale.md`.

## How the facet model integrates (responsibility mapping)

This skill **focuses on concept explanation**. Actual command invocation / options / usage are delegated to:

| Intent | Delegation target |
|--------|-------------------|
| Look up facet values / `hstl-oss context --facets/--layer/--all` options | `project-management` skill |
| Direct command invocation (compressed 200 tokens for AI) | `/hstl-oss:project:context` command |
| Procedure for registering custom domain facets | `references/domain-facet-creation.md` |
| Layer 3 plugin SDK transport (gRPC vs stdio JSON-RPC) | upstream ADR direct reference |

This skill body only points to *where each responsibility lives*; concrete usage is found at the delegation target.

## When to invoke this skill

| User question / context | This skill | Delegation target |
|-------------------------|------------|-------------------|
| "What's a facet?" / "Why 12 facets?" / "Explain the facet model" | ✅ | |
| "What's the difference between Layer 1 / 2 / 3?" / "Stable Core / Dynamic / On-Demand meaning?" | ✅ | |
| "Why 3-layer separation?" / "context engineering principles" | ✅ | |
| "How do I add a facet to my project?" | ✅ (delegates to references/domain-facet-creation.md) | |
| Usage / options for `hstl-oss context --facets <list>` | ❌ | `project-management` skill |
| Actually call a command to look up facet values | ❌ | `/hstl-oss:project:context` command |
| Layer 3 plugin transport decision (gRPC vs stdio) | ❌ | upstream ADR direct |

This skill is **not invoked** for usage questions — even if a trigger word ("facet" etc.) matches, if the user's intent is *command invocation / options* rather than *concept / rationale / registration procedure*, delegate immediately.

## Related

- Upstream ADR — Project Context 12-Facet × 3-Layer Model (decision SSOT, 8 decision points)
- Upstream ADR — Plugin SPI stdio JSON-RPC v2.0 (Layer 3 transport)
- facet-spec — Phase 1 standard (facet declaration schema, resolver interface)
- `project-management` skill — delegation for `hstl-oss context` / `hstl-oss brief` usage
- `/hstl-oss:project:context` command — entry point for actually retrieving facet values

The exact paths of the documents above vary with the user project's docs/ structure. The hostler-plugin standard identifies them by skill name (`project-management` / `context-engineering`); each user project finds them by grepping its own docs location.
