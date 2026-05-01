# Knowledge document templates

> Two templates: lessons learned and analysis reports

---

## Table of contents

1. [Lessons Learned](#1-lessons-learned-template)
2. [Analysis report](#2-analysis-report-template)

---

## 1. Lessons Learned template

```markdown
# L-{NNN}: {title}

## Overview

| Item | Content |
|------|---------|
| ID | L-{NNN} |
| Title | {title} |
| Category | Technical / Process / Collaboration / Tooling |
| Date | {YYYY-MM-DD} |
| Author | {author} |
| Related Task | {TASK-XXX} |

---

## 1. Context

{Describe the situation where the lesson surfaced}

### Background

- {background 1}
- {background 2}

### Goal

{What you were trying to achieve at the time}

---

## 2. Problem

{Problem or challenge that arose}

### Symptoms

- {symptom 1}
- {symptom 2}

### Cause

- **Direct cause:** {direct cause}
- **Root cause:** {root cause}

### Impact

- {impact 1}
- {impact 2}

---

## 3. Solution

{How the problem was solved}

### Attempts

| Attempt | Result | Notes |
|---------|--------|-------|
| {attempt 1} | failed / succeeded | {notes} |
| {attempt 2} | failed / succeeded | {notes} |

### Final solution

{The solution that worked}

```
{code or command example}
```

---

## 4. Lessons

### Headline

> {headline lesson in one sentence}

### Detailed lessons

1. **{lesson 1}**
   - {description}

2. **{lesson 2}**
   - {description}

### Wrong assumptions

- {wrong assumption 1}
- {wrong assumption 2}

---

## 5. Application

### Apply now

| Action | Target | Status |
|--------|--------|--------|
| {action 1} | {target} | ⬜ pending |
| {action 2} | {target} | ⬜ pending |

### Apply later

| Action | When | Owner |
|--------|------|-------|
| {action 1} | {when} | {owner} |

### Documentation / guide updates

| Document | Update |
|----------|--------|
| {doc 1} | {update} |

---

## 6. Related

### Related documents

- {document 1}
- {document 2}

### Related lessons

- L-{NNN}: {related lesson}

### Tags

`{tag1}` `{tag2}` `{tag3}`

---

## Change history

| Date | Change | Author |
|------|--------|--------|
| {date} | Initial draft | {author} |
```

---

## 2. Analysis report template

```markdown
# Analysis report: {topic}

## Overview

| Item | Content |
|------|---------|
| Title | {analysis topic} |
| Date | {YYYY-MM-DD} |
| Author | {author} |
| Version | 1.0 |
| Status | Draft / In review / Final |

---

## 1. Executive Summary

{3–5 sentence summary of the analysis}

### Key findings

1. {finding 1}
2. {finding 2}
3. {finding 3}

### Recommendations

1. {recommendation 1}
2. {recommendation 2}

---

## 2. Purpose

### Background

{Why the analysis is needed}

### Goals

1. {goal 1}
2. {goal 2}

### Scope

**Included:**
- {included}

**Excluded:**
- {excluded}

---

## 3. Methodology

### Approach

{Describe the analysis approach}

### Data sources

| Source | Type | Period |
|--------|------|--------|
| {source 1} | {type} | {period} |
| {source 2} | {type} | {period} |

### Tools

- {tool 1}
- {tool 2}

---

## 4. Current-state analysis

### 4.1 {area 1}

**Current state:**
{description}

**Data:**

| Item | Value | Notes |
|------|-------|-------|
| {item 1} | {value} | {notes} |
| {item 2} | {value} | {notes} |

**Analysis:**
{analysis result}

---

### 4.2 {area 2}

**Current state:**
{description}

**Analysis:**
{analysis result}

---

## 5. Findings

### 5.1 {finding 1}

**Description:**
{finding detail}

**Evidence:**
- {evidence 1}
- {evidence 2}

**Impact:**
{impact of this finding}

---

### 5.2 {finding 2}

**Description:**
{description}

**Impact:**
{impact}

---

## 6. Comparative analysis

### {comparison item}

| Item | Option A | Option B | Option C |
|------|----------|----------|----------|
| {criterion 1} | {value} | {value} | {value} |
| {criterion 2} | {value} | {value} | {value} |
| **Total score** | {score} | {score} | {score} |

---

## 7. Risks and opportunities

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| {risk 1} | high/med/low | high/med/low | {mitigation} |

### Opportunities

| Opportunity | Likelihood | Value | Approach |
|-------------|------------|-------|----------|
| {opportunity 1} | high/med/low | high/med/low | {approach} |

---

## 8. Conclusion

### Overall assessment

{overall assessment}

### Key insights

1. {insight 1}
2. {insight 2}

---

## 9. Recommendations

### Short-term (1–2 weeks)

| Recommendation | Priority | Owner | Expected outcome |
|----------------|----------|-------|------------------|
| {rec 1} | High | {owner} | {outcome} |

### Mid-term (1–3 months)

| Recommendation | Priority | Owner | Expected outcome |
|----------------|----------|-------|------------------|
| {rec 1} | Medium | {owner} | {outcome} |

### Long-term (3–6 months)

| Recommendation | Priority | Owner | Expected outcome |
|----------------|----------|-------|------------------|
| {rec 1} | Low | {owner} | {outcome} |

---

## 10. Next steps

- [ ] {next step 1}
- [ ] {next step 2}
- [ ] {next step 3}

---

## Appendix

### A. Detailed data

{detailed data or links}

### B. References

- {reference 1}
- {reference 2}

---

## Change history

| Version | Date | Change | Author |
|---------|------|--------|--------|
| 1.0 | {date} | Initial draft | {author} |
```

---

## Lesson category guide

| Category | Description | Examples |
|----------|-------------|----------|
| **Technical** | Tech decisions, implementation, debugging | Library choice, performance tuning |
| **Process** | Development process, workflow | Code review, deployment procedure |
| **Collaboration** | Team collaboration, communication | Meeting efficiency, doc sharing |
| **Tooling** | Development tools, environment | IDE setup, CI/CD configuration |

---

## Tag examples

- Technical: `architecture`, `performance`, `security`, `testing`
- Process: `deployment`, `code-review`, `planning`
- Collaboration: `communication`, `documentation`
- Tooling: `docker`, `ci-cd`, `monitoring`
