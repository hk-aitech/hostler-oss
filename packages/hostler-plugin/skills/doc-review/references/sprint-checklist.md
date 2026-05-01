# SPRINT.md Quality Checklist

## Frontmatter (BLOCK)

| Field | Type | Required |
|------|------|:---:|
| id | string | ✅ |
| title | string | ✅ |
| status | enum (backlog/active/completed) | ✅ |
| start | date | ✅ |
| end | date | ✅ |

### Frontmatter Example

```yaml
---
id: sprint-NN
title: "Conditional search + supply/demand deep-dive"
status: active
start: 2026-03-20
end: 2026-04-02
---
```

## Required Sections (BLOCK)

### `## Metadata`
Table format: Sprint ID, period, Task count, estimated workload

```markdown
## Metadata

| Item | Value |
|------|-------|
| Sprint ID | sprint-NN |
| Period | 2026-03-20 ~ 2026-04-02 |
| Task count | 12 |
| Estimated workload | 24 SP |
```

### `## Goal`
1–3 lines of goal description + a list of key goals

```markdown
## Goal

Lift the hit rate above 20% by putting conditional search into production and deep-analyzing supply/demand data.

- Integrate Kiwoom conditional search over REST + WS
- Finish backfilling the KIS investor-flow dataset
- Retrain LightGBM (60-feature variant)
```

### `## Task List`
Table: ID, title, size, priority, status

```markdown
## Task List

| ID | Title | Size | Priority | Status |
|----|-------|------|----------|--------|
| | Configure conditional search HTS | S | P1 | todo |
| | Improve surge-detection logic    | M | P0 | in-progress |
```

## Recommended Sections (WARN)

### `## Done Criteria`
Checkbox list

```markdown
## Done Criteria

- [ ] Conditional-search API live in production
- [ ] hit rate ≥ 20%
- [ ] LightGBM retraining complete
```

### `## Dependency Graph`
ASCII or markdown graph

### `## Risks`
Table: risk, impact, mitigation

```markdown
## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Kiwoom API rate-limit exceeded | Conditional search unavailable | Tune polling interval |
```

## Cross-validation (WARN)

| Item | Check |
|------|----------|
| Task count consistency | Number of table rows == number of T*.md files in tasks/ |
| Task sprint field | Each Task's sprint field == SPRINT.md `id` |
| Date order | start <= end |
| Status valid values | One of backlog/active/completed |

## Auto-fixable Items

| Issue | Fix |
|------|----------|
| Frontmatter field missing | Add with default |
| `## Done Criteria` missing | Insert empty section stub |

## Not Auto-fixable

| Issue | Reason |
|------|------|
| Task count mismatch | Requires either adding/removing files or editing the table |
| `## Goal` content missing | Must be authored by a human |
| sprint field mismatch | Cannot determine which side is correct |
