# Y-Statement Template (Zimmermann, 2012)

**Y-statement** = WH(**Y**) format that compresses the "why" of a decision into one sentence.

First proposed by Olaf Zimmermann at SATURN 2012, formalized in Zdun et al.
*Sustainable Architectural Decisions* (2013). Validated for over 10 years at
HSR/OST university and ABB practice.

## 1-Sentence Template

```
In the context of <use case / user story>,
facing <concern c> (and <concern d> (and ...)),
we decided for <option / pattern> and against <other options>,
to achieve <system qualities / desired consequences>,
accepting <downside d / negative consequences>.
```

Korean paraphrase (kept for projects that prefer Korean prose):

```
In the context of <use case>,
facing <quality requirement / constraint>,
we adopt <chosen option> instead of <alternative A>
to achieve <target system qualities>,
accepting <trade-off>.
```

## Examples

### Example 1 — Session State Management

> In the context of **the Web shop service**, facing the need to **keep user
> session data consistent and current across shop instances**, we decided for
> **the Database Session State Pattern** (and against **Client Session State or
> Server Session State**) to achieve **cloud elasticity**, accepting that **a
> session database needs to be designed, implemented, and replicated**.

### Example 2 — Generic application (Dual Identifier)

> In the context of **the project's identifier system**, facing the need to
> **simultaneously achieve machine-friendly global uniqueness and human-friendly
> memorability**, we decided for **Dual Identifier (UUIDv7 + human-readable
> AltID)** (and against **UUID alone or AltID alone**) to achieve **collision
> prevention in a distributed environment plus document traceability**, accepting
> **the synchronization overhead between the two identifiers and increased
> frontmatter complexity**.

## Extension (Y-statement + body)

When 1 sentence is not enough rationale, use the Y-statement as a **Summary** and append Nygard sections below:

```markdown
---
id: ADR-NNN
title: ADR-NNN <title>
status: proposed
date: YYYY-MM-DD
deciders: <name / role>
---

# ADR-NNN <title>

## Summary (Y-statement)

In the context of ..., facing ..., we decided for ... and against ...,
to achieve ..., accepting ... .

## Context

<detailed background>

## Decision

<expansion of the Y-statement's "we decided for">

## Consequences

<expansion of the Y-statement's "to achieve" + "accepting">
```

## When to Use Y-Statement

- The early stage of an **Architecture Advice Process** (for advice gathering)
- When you want to share **just the core of the decision quickly**
- **Common adoption scenario**: early iteration design discussion → advice gathering →
  Y-statement draft → promote to MADR or Nygard after approval
- For catalog summaries (an index that lists Y-statements only, instead of full ADR bodies)

## When NOT to Use Y-Statement

- Complex decisions that **need a lot of rationale explanation** → Nygard / MADR
- Decisions that **need detailed Consequences analysis** → MADR
- Decisions that **need pros/cons comparison across multiple alternatives** → MADR

## Authoring Tips

1. **Don't omit any of the 5 elements** — context / concern / choice / quality / trade-off
2. **"accepting" is the most important** — prevents A1 Fairy Tale
3. **If the 1 sentence becomes too long** (>40 words), promote to Nygard / MADR
4. **Only list actually-considered alternatives in "and against"** — watch out for A4 Dummy Alternative

## Derived Variants

- **cards42 ADR cards** — compresses Y-statement to a card size (for team workshops)
- **e-ADR** (embedded ADR) — embeds Y-statement in Java code comments
  (Zimmermann: github.com/adr/e-adr)

## References

- Original: https://medium.com/olzzio/y-statements-10eb07b5a177
- Zimmermann blog: https://ozimmer.ch/practices/2020/04/27/ArchitectureDecisionMaking.html
- Design Practice Repository: https://socadk.github.io/design-practice-repository/artifact-templates/DPR-ArchitecturalDecisionRecordYForm.html
- Zdun et al. 2013 *Sustainable Architectural Decisions* (IEEE Software)
