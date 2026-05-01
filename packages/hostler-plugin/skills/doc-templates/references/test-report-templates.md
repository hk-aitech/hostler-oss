# Test report templates

> Three templates for recording Phase / Sprint / Feature test results

## Table of contents

1. [Phase Test Report](#1-phase-test-report-template)
2. [Sprint Test Report](#2-sprint-test-report-template)
3. [Feature Test Report](#3-feature-test-report-template)

---

## 1. Phase Test Report template

**Location**: `docs/06-reports/testing/phase{N}-test-report.md`

```markdown
---
phase: {N}
test_date: YYYY-MM-DD
status: passed | failed | partial
---

# Phase {N} Test Report

## Overview

| Item | Content |
|------|---------|
| Phase | {N} |
| Test date | YYYY-MM-DD |
| Status | {passed/failed/partial} |

## Test environment

- OS:
- Runtime:
- Database:

## Test result summary

| Category | Total | Pass | Fail | Skip |
|----------|-------|------|------|------|
| Unit     |       |      |      |      |
| Integration |    |      |      |      |
| E2E      |       |      |      |      |

## Detailed results

### Passed
- [ ] item 1
- [ ] item 2

### Failed
- [ ] item (reason: )

## Issues found

| ID | Severity | Description | Status |
|----|----------|-------------|--------|
|    |          |             |        |

## Recommended actions

1. action 1
2. action 2
```

---

## 2. Sprint Test Report template

**Location**: `docs/06-reports/testing/p{N}-s{M}-test-report.md`

```markdown
---
phase: {N}
sprint: {M}
test_date: YYYY-MM-DD
---

# Sprint P{N}-S{M} Test Report

## Sprint info

| Item | Content |
|------|---------|
| Phase | {N} |
| Sprint | {M} |
| Period | YYYY-MM-DD ~ YYYY-MM-DD |

## Tasks under test

| Task | Description | Result |
|------|-------------|--------|
| TASK-NNN | description | Pass/Fail |

## Blockers / issues

- issue 1
- issue 2
```

---

## 3. Feature Test Report template

**Location**: `docs/06-reports/testing/{feature}-test-report.md`

```markdown
# {Feature name} Test Report

## Overview

| Item | Content |
|------|---------|
| Feature | {feature name} |
| Test date | YYYY-MM-DD |
| Status | {passed/failed/partial} |

## Test scenarios

| # | Scenario | Input | Expected | Actual | Pass/Fail |
|---|----------|-------|----------|--------|-----------|
| 1 | {scenario} | {input} | {expected} | {actual} | {result} |

## Screenshots / evidence

{screenshots or log excerpts}

## Conclusion

{test conclusion + follow-ups}
```
