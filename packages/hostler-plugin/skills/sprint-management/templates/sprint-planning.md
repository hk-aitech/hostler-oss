# Sprint Planning: {Sprint ID}

## Overview

| Item | Value |
|------|-------|
| Sprint ID | {p{N}-s{M}} |
| Name | {Sprint name} |
| Phase | Phase {N} |
| Period | {start} ~ {end} |
| Status | 📋 Planning |

---

## 1. Sprint goals

### Headline goal

{Sprint's headline goal in 1–2 sentences}

### Detailed goals

1. {goal 1}
2. {goal 2}
3. {goal 3}

---

## 2. Scope

### In Scope

- {included 1}
- {included 2}

### Out of Scope

- {excluded 1}
- {excluded 2}

---

## 3. Task plan

### 3.1 Task list

| ID | Title | Size | Owner | Priority | Dependencies |
|----|-------|------|-------|----------|--------------|
| TASK-XXX | {title} | S/M/L | developer | P0 | - |
| TASK-YYY | {title} | S/M/L | devops | P1 | TASK-XXX |

### 3.2 Workload estimate

| Size | Count | Hours |
|------|-------|-------|
| XS | X | X h |
| S | X | X h |
| M | X | X h |
| L | X | X h |
| **Total** | **X** | **X h** |

---

## 4. Dependency analysis

### 4.1 Dependency graph

```
TASK-XXX
    │
    ▼
TASK-YYY ──▶ TASK-ZZZ
```

### 4.2 External dependencies

| Dependency | Owner | Status | Expected resolution |
|------------|-------|--------|---------------------|
| {dependency 1} | {owner} | pending / done | {date} |

---

## 5. Capacity Analysis

| Item | Value |
|------|-------|
| Total available time | {N} h |
| Previous Sprint measured velocity | {N} tasks/day |
| Recommended capacity (80–85%) | {N} h |
| Buffer (uncertainty 15–20%) | {N} h |

> **Best Practice**: plan a Sprint's workload at 80–85% of total available time. Allocate the remaining 15–20% to unexpected issues, code review, meetings, etc.

---

## 6. Risk analysis

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| {risk 1} | high/med/low | high/med/low | {mitigation} |
| {risk 2} | high/med/low | high/med/low | {mitigation} |

---

## 7. Metrics to track

| Metric | Measurement | Target |
|--------|-------------|--------|
| Velocity | Completed Tasks / Sprint days | {N} tasks/day |
| Burndown | Remaining Tasks per day | Linear decline |
| Cycle Time | Average Task start → done | ≤ {N} h |
| Blocked Time | Cumulative time in blocked state | ≤ 10% of total |
| Code coverage | Test coverage (at Sprint completion) | ≥ {N}% |

---

## 8. Referenced design documents

> **Required**: list the design documents referenced for Sprint planning

| Document type | Path | What to check |
|---------------|------|---------------|
| Phase spec | `docs/00-project/phases/phase{N}-spec.md` | Basis for the Sprint goal |
| Feature design | `docs/03-design/feature/{feature}.md` | Feature details |
| Domain design | `docs/03-design/domain/{domain}.md` | Domain model |
| ADR | `docs/02-architecture/adrs/ADR-{NNN}.md` | Tech decisions |

---

## 9. Done-criteria checklist

### Functional verification
- [ ] All Tasks complete
- [ ] {functional criterion 1}
- [ ] {functional criterion 2}

### Quality verification
- [ ] Build green
- [ ] Tests pass (coverage ≥ {X}%)
- [ ] Code review complete

### Design consistency verification
- [ ] Implementation matches the referenced design docs
- [ ] ADR decisions adhered to
- [ ] Domain model consistency confirmed

### Documentation verification
- [ ] Changed design docs updated
- [ ] API docs updated (where applicable)
- [ ] Sprint result recorded

---

## 10. Schedule

| Week | Plan |
|------|------|
| Week 1 | {plan} |
| Week 2 | {plan} |
