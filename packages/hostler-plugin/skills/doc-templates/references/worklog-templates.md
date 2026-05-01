# Worklog + session log templates

> Templates for daily work logs, session logs, and sprint retrospectives

## Table of contents

1. [Daily worklog](#daily-worklog)
2. [Session worklog (long sessions)](#session-worklog-long-sessions)
3. [Sprint retrospective](#sprint-retrospective)

## Daily worklog

```markdown
# Worklog YYYY-MM-DD

> Session: HH:MM ~ HH:MM KST (~N hours)
> Commits: **N**
> Tests: N passed

---

## Completed work

| Task | Description | Status |
|------|-------------|--------|
| TNN | {description} | ✅ |

## Bug fixes

| Fix | Description |
|-----|-------------|
| {module} | {fix detail} |

## Key decisions

- {decision 1}

## Lessons logged

- {lesson 1}

## Tomorrow

1. {task 1}
```

## Session worklog (long sessions)

```markdown
# Worklog YYYY-MM-DD Session N

> Session: HH:MM ~ HH:MM KST (~N hours)
> Commits: **N**
> Tests: N→M passed (+K)
> KB: N→M cards (+K)
> FeatureVector: N→M kinds (+K)

---

## Completed Sprints

| Sprint | Task | Outcome |
|--------|------|---------|
| NN | {key outcome} | ✅ |

## Sprint NN progress (N/M done)

| Task | Description | Status |
|------|-------------|--------|

## Docs authored

| Doc | Content |
|-----|---------|

## Lessons logged (N items)

| ID | Category | Content |
|----|----------|---------|

## Tomorrow

1. {task 1}
```

## Sprint retrospective

```markdown
# Sprint NN Retrospective

> Scope: {Sprint scope}
> Period: YYYY-MM-DD ~ YYYY-MM-DD
> Commits: N

---

## 1. What was achieved

{Quantitative outcomes}

## 2. What went well

### ✅ {item}
{description}

## 3. What didn't go well

### ❌ {item}
{description + lesson}

## 4. What we learned

| # | Lesson | KB |
|---|--------|-----|

## 5. What to do differently

| # | Improvement | When to apply |
|---|-------------|---------------|

## 6. Numeric summary

| Metric | Start | End | Change |
|--------|-------|-----|--------|
```
