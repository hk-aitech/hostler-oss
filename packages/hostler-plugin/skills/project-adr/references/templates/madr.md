# MADR 4.0 Template (released 2024-09-17)

**MADR** = Markdown **Any** Decision Records (renamed to "Any" in 2023 — the broadened ADR movement).

MADR 4.0 was released on 2024-09-17 and ships in two variants: **full** and **minimal**.

## Full Template

Use when comparing multiple alternatives is the core. Decision drivers + options + consequences analysis are structured.

```markdown
---
id: ADR-NNN
title: ADR-NNN <title>
status: proposed                     # proposed | rejected | accepted | deprecated | superseded by ADR-MMM
date: YYYY-MM-DD                      # last modified date
deciders: <name / role>               # final approvers
consulted: <optional — advice recipients>
informed: <optional — notification recipients>
supersedes: []
relates_to: []
---

# ADR-NNN <short decision title — present tense>

## Context and Problem Statement

<2–3 sentences on the problem context and background. Neutral, free form or an illustrative story.>

## Decision Drivers

- <driver 1 — e.g. speed · cost · team skills>
- <driver 2>
- <...>

## Considered Options

- <Option 1>
- <Option 2>
- <Option 3>

## Decision Outcome

Chosen option: "<Option N>", because <justification — link to the drivers>.

### Consequences

- Good, because <consequence 1>
- Good, because <consequence 2>
- Bad, because <consequence 3>

### Confirmation

<How to confirm the decision is correctly implemented — tests, metrics, audits, etc.>

## Pros and Cons of the Options

### Option 1

<brief description>

- Good, because <advantage>
- Neutral, because <neutral>
- Bad, because <disadvantage>

### Option 2

<...>

## More Information

<related links · discussions · follow-up ADRs>
```

## Minimal Template

Use when there is no alternative comparison or the decision is simple. Lightweight and similar to Nygard.

```markdown
---
id: ADR-NNN
title: ADR-NNN <title>
status: proposed
date: YYYY-MM-DD
deciders: <name / role>
---

# ADR-NNN <decision title>

## Context and Problem Statement

<...>

## Considered Options

- <Option 1>
- <Option 2>

## Decision Outcome

Chosen option: "<Option N>", because <...>.
```

## When to Use MADR

- You must compare **2+ alternatives** explicitly
- There are **clearly multiple** decision drivers (speed + cost + security, etc.)
- **Team-wide buy-in before sharing** matters — a comparison table aids persuasion
- The project CI runs a markdown linter and you want enforced compliance

## Major Changes in MADR 4.0 (vs 3.0)

- Element renaming (some template elements)
- **Minimal variant introduced** — version with optional elements removed
- Two formats provided: annotated / bare (with or without comments)
- Conformance with Semantic Versioning 2.0.0 + CHANGELOG

## MADR 3.0 → 4.0 Migration Checklist

| 3.0 section | In 4.0 | Action |
|--------------|---------|------|
| Consequences (split Positive/Negative) | Merged (Good/Bad/Neutral inline) | Re-write merged |
| Links | More Information | Renamed section |

## Authoring Tips

1. **Write Decision Drivers first** — without drivers, option comparison is vacuous
2. **Confirmation section is optional but recommended** — can integrate with CI gates / tests / benchmarks
3. **Aim for 2–5 options** — 1 risks Dummy Alternative (A4); 6+ leads to analysis paralysis
4. **At least 1 Good / 1 Bad per option** — prevents Free Lunch Coupon (A3)

## References

- Official: https://adr.github.io/madr/
- Repo: https://github.com/adr/madr
- Annotated version: https://github.com/adr/madr/blob/develop/template/adr-template.md
- Zimmermann commentary: https://ozimmer.ch/practices/2022/11/22/MADRTemplatePrimer.html
