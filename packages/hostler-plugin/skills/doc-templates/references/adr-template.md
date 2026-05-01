# ADR (Architecture Decision Record) template

> Template for recording architecture decisions

---

## ADR template

```markdown
---
title: ADR-{NNN} {title}
status: proposed | accepted | deprecated | superseded
date: {YYYY-MM-DD}
deciders: {list of deciders}
---

# ADR-{NNN}: {title}

## Status

{proposed | accepted | deprecated | superseded by ADR-XXX}

## Date

{YYYY-MM-DD}

---

## Context

{Describe the background and situation that requires a decision.}

- Current state
- Problem
- Constraints
- Requirements

---

## Considered options

### Option 1: {option name}

**Description:**
{option description}

**Pros:**
- {pro 1}
- {pro 2}

**Cons:**
- {con 1}
- {con 2}

### Option 2: {option name}

**Description:**
{option description}

**Pros:**
- {pro 1}

**Cons:**
- {con 1}

---

## Decision

{Clearly explain the chosen decision and the reasoning behind it.}

**Chosen option:** Option {N}

**Reasoning:**
- {reason 1}
- {reason 2}

---

## Consequences

### Positive

- {positive 1}
- {positive 2}

### Negative

- {negative 1}
- {negative 2}

### Mitigation

- {how to mitigate the negatives}

---

## References

- {related doc 1}
- {related ADR}
- {external reference}

---

## Change history

| Date | Change | Author |
|------|--------|--------|
| {date} | Initial draft | {author} |
```

---

## ADR status definitions

| Status | Description |
|--------|-------------|
| **proposed** | Proposal under review |
| **accepted** | Approved and in effect |
| **deprecated** | No longer recommended |
| **superseded** | Replaced by a newer ADR |

---

## ADR authoring guide

### When to write an ADR

- A decision that affects architecture
- Tech stack selection
- Design pattern adoption
- Development process change
- Hard-to-reverse decisions

### Traits of a good ADR

1. **Clear context**: why a decision was needed
2. **Sufficient options reviewed**: compare at least 2 alternatives
3. **Reasoned decision**: why that option was chosen
4. **Expected outcomes**: capture both positive and negative consequences

### ADR file naming

```
docs/02-architecture/
├── ADR-001-use-clean-architecture.md
├── ADR-002-select-postgresql-database.md
└── ADR-003-adopt-cqrs-pattern.md
```
