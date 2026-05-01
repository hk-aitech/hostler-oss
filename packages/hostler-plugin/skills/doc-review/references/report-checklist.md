# Report / ADR / Requirements Quality Checklist

## Reports (docs/06-reports/)

| Item | Severity | Check |
|------|------|----------|
| # title (h1) | BLOCK | `^# ` pattern exists in the file |
| Authoring date | BLOCK | `Authored:`, `Date:`, `date:`, `YYYY-MM-DD` pattern present |
| Table of contents | WARN | Recommend `## Table of Contents` or TOC when there are 5+ sections (##) |
| ## Conclusion section | WARN | `## Conclusion` exists |
| Substantive content | INFO | 50+ total lines |

### Report Authoring-Date Pattern Examples

```markdown
> Authored: 2026-03-19
```

or

```markdown
| Date | 2026-03-19 |
```

## ADR (docs/02-architecture/)

| Item | Severity | Check |
|------|------|----------|
| ADR number in # title | BLOCK | `^# ADR-\d+` or `^# \[ADR` pattern |
| Status | BLOCK | One of Accepted / Deprecated / Superseded / Proposed |
| ## Context section | BLOCK | `## Context` exists |
| ## Decision section | BLOCK | `## Decision` exists |
| ## Consequences section | WARN | `## Consequences` exists |
| ## Alternatives section | INFO | `## Alternatives` exists |

### ADR Status Notation Example

```markdown
**Status**: Accepted (2026-03-19)
```

or

```markdown
> Status: Accepted
```

The literal token `Status` is intentional — it is one of the patterns the parser
matches when extracting ADR status (alongside `status:`).

### Minimum ADR Structure Example

```markdown
# ADR: TimescaleDB selected

**Status**: Accepted (2026-01-15)

## Context

We need to store and query time-series OHLCV data efficiently.
We prefer the direction that minimizes operational overhead through PostgreSQL extensions.

## Decision

Adopt TimescaleDB. Since it is PostgreSQL-based, asyncpg works as-is, and
hypertables handle time-series partitioning automatically.

## Consequences

- Positive: time-series query performance 10x improvement, automatic compression
- Negative: larger Docker image size, PostgreSQL version constraints
```

## Requirements (docs/requirements/)

| Item | Severity | Check |
|------|------|----------|
| Version info | BLOCK | `Version:`, `version:`, `v\d+\.\d+` patterns |
| Requirement ID format | WARN | FR-XX-NNN, NFR-XX-NNN, SR-NNN patterns |
| Acceptance criteria per requirement | WARN | `Acceptance Criteria`, `GIVEN`, `WHEN`, `THEN` patterns |
| Priority | INFO | P0/P1/P2/P3 or HIGH/MED/LOW notation |

### Requirements ID Scheme

| Type | Pattern | Example |
|------|------|------|
| Functional requirements | FR-{area}-{number} | FR-SCAN-001 |
| Non-functional requirements | NFR-{category}-{number} | NFR-PF-001 |
| System requirements | SR-{number} | SR-001 |
| Use cases | UC-{number} | UC-003 |

### Category Codes

| Code | Meaning |
|------|------|
| PF | Performance |
| RL | Reliability |
| SE | Security |
| MT | Maintainability |
| SC | Scalability |

## Design Documents (docs/design/)

| Item | Severity | Check |
|------|------|----------|
| # title | BLOCK | `^# ` pattern |
| Date or version | BLOCK | Date pattern or `v\d+` |
| ## Overview section | WARN | `## Overview` exists |
| ## Architecture diagram | INFO | Code block or image link |

## Auto-fixable Items

| Doc type | Issue | Fix |
|----------|------|----------|
| Report | No `## Conclusion` | Insert empty section stub |
| ADR | No `## Consequences` | Insert empty section stub |
| Requirements | No version | Insert `> Version: 1.0.0` |

## Not Auto-fixable

| Issue | Reason |
|------|------|
| Report has no authoring date | The exact date cannot be inferred |
| ADR has no status | The decision status cannot be guessed |
| Requirements have no ID scheme | Renumbering existing IDs is risky |
