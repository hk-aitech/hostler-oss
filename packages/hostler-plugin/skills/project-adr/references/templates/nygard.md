# Nygard Original Template (2011)

The original from Michael Nygard — *Documenting Architecture Decisions* (2011).
The most universal format, adopted as the baseline by most ADR repos and guides.

## Structure

```markdown
---
id: ADR-NNN
title: ADR-NNN <title>
status: proposed
date: YYYY-MM-DD
deciders: <name / role>
supersedes: []
relates_to: []
---

# ADR-NNN — <title>

## Status

proposed  (→ accepted / rejected / superseded / deprecated)

## Context

<Describe the technical, political, social, and project-local forces.
Surface the tensions honestly. Neutral tone.>

## Decision

<State the response decision in a complete sentence, in the active voice.
"We will ..." style is recommended.>

## Consequences

<Describe the context that emerges after applying the decision.
Include positive, negative, and neutral consequences.>
```

## Characteristics

- **Length**: 1–2 pages (80–250 lines) recommended
- **Tone**: Journalistic. No cookbook / imperative tone (avoid A8 Blueprint/Policy)
- **Tense**: Context is at authoring time. Decision is past/active "we have decided".
  Consequences is future/narrative "once applied, X becomes ...".

## When to Use Nygard

- The decision is **simple** and **alternative comparison is not the focus**
- The project does not specify an ADR template (the most common default)
- Speed matters (more rationale than Y-statement, lighter than MADR)

## When to Use a Different Template

- You need to **explicitly compare 2+ alternatives** → use **MADR 4.0**
- You need a **1-sentence draft** to convey the core during the Architecture Advice Process → use **Y-statement**

## Faithful Example (generic — monolith decomposition decision)

```markdown
---
id: ADR-N
title: ADR-N Progressive Monolith Extraction Strategy
status: accepted
date: 2026-04-10
accepted_date: 2026-04-11
deciders: @tech-lead
supersedes: []
relates_to: [ADR-013, ADR-M]
---

# ADR-N — Progressive Monolith Extraction

## Status

accepted (approved on 2026-04-11)

## Context

As the product grew, the single deploy unit hit limits. A failed deploy
takes down everything, and per-team release cadences are hard to operate
independently. Full microservices is too costly relative to team size.

## Decision

We will reorganize the current structure into a **modular monolith**, and then
**progressively extract independent services** starting from traffic-hotspot modules.
New features will be written respecting the module boundaries.

## Consequences

- **Positive**: failed-deploy blast radius shrinks to a module
- **Positive**: per-team release cadences become realistically independent
- **Negative**: refactoring cost for code that violates module boundaries (1–2 months)
- **Negative**: deployment risk persists until services are split out
- **Neutral**: selection criteria (traffic / team boundaries) for what to extract require periodic re-evaluation
```

## Authoring Tips

1. **Don't write "we picked this" in Context** — that belongs in Decision
2. **Don't write "because" in Decision** — reasons go in Context
3. **Always include at least 1 "negative" in Consequences** — prevents Fairy Tale (A1)
4. **Keep code blocks limited to the Decision's core APIs** — prevents Mega-ADR (A9). Watch out beyond 3 blocks

## References

- Original: https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions
- joelparkerhenderson/architecture-decision-record repo (includes Korean Nygard templates)
