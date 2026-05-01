# Troubleshooting document templates

> Topic guide and worklog templates

---

## Table of contents

1. [index.md template](#indexmd-template)
2. [Topic guide template](#topic-guide-template)
3. [Worklog template](#worklog-template)
4. [Simple incident-record template](#simple-incident-record-template)

---

## index.md template

The hub page template for the troubleshooting area.

```markdown
# Troubleshooting Guide

This document collects problems found during {project name} development and their solutions.

## Topic guides

Curated troubleshooting documents:

| Topic | Document | Last updated |
|-------|----------|--------------|
| {topic 1} | [{filename}](./{filename}.md) | YYYY-MM-DD |
| {topic 2} | [{filename}](./{filename}.md) | YYYY-MM-DD |

## Worklogs

Complex incident investigations recorded by date:

| Date | Problem | Log |
|------|---------|-----|
| YYYY-MM-DD | {summary} | [{filename}](./worklogs/YYYY-MM-DD-{topic}.md) |
| YYYY-MM-DD | {summary} | [{filename}](./worklogs/YYYY-MM-DD-{topic}.md) |

---

## Common problems and fixes

### 1. {problem title}

**Symptoms**
- {symptom 1}
- {symptom 2}

**Cause**
{Brief cause}

**Fix**
```bash
{fix command or code}
```

**Prevention**
- {prevention 1}
- {prevention 2}

---

### 2. {problem title}

**Symptoms**
{symptom description}

**Fix**
{fix description}

---

## Verification

### Verify {feature 1}

```bash
# State check
{command}

# Run tests
{command}
```

### Verify {feature 2}

```bash
{command}
```

---

## Related files

| File | Description |
|------|-------------|
| `{file path}` | {description} |
| `{file path}` | {description} |

---

## Lessons Learned

1. **{lesson 1}**: {description}
2. **{lesson 2}**: {description}
3. **{lesson 3}**: {description}

---

## Dev-environment diagnostic tool

### Using {tool name}

```bash
# Status check
{command}

# Diagnose errors
{command}

# Auto recovery
{command}
```

**Checks**
- {item 1}
- {item 2}
- {item 3}

**Details**: [{runbook link}](../05-runbooks/...)
```

---

## Topic guide template

A structured guide template for complex problems.

```markdown
# {Problem topic} Troubleshooting

**Status**: complete / in progress
**Created**: YYYY-MM-DD
**Last updated**: YYYY-MM-DD
**Related commits**: `{commit-hash}`, `{commit-hash}`

---

## 1. Overview

{Overall description of the problem}

### 1.1 Versions

- {framework}: {version}+
- {runtime}: {version}
- {other}: {version}

### 1.2 Related technologies

- {tech 1}
- {tech 2}

---

## 2. {Problem type 1}

### 2.1 Symptoms

```
{error message or log}
```

- {symptom 1}
- {symptom 2}
- {symptom 3}

### 2.2 Cause analysis

#### Cause 1: {cause title}

{Detailed cause description}

```{language}
// Example of the offending code
{code}
```

#### Cause 2: {cause title}

{Detailed cause description}

### 2.3 Solutions

#### Solution 1: {solution title}

{Solution description}

```{language}
// Example of the fixed code
{code}
```

**Caveats:**
- {caveat 1}
- {caveat 2}

#### Solution 2: {solution title}

{Solution description}

```{language}
{code}
```

---

## 3. {Problem type 2}

### 3.1 Symptoms

```
{error message}
```

- {symptom description}

### 3.2 Cause

{Cause description}

### 3.3 Workaround (e.g. disable plugin)

```{language}
{workaround code}
```

### 3.4 Permanent fixes

1. **{fix 1}**: {description}
2. **{fix 2}**: {description}
3. **{fix 3}**: {description}

---

## 4. Related files

### 4.1 Modified files

| File | Change |
|------|--------|
| `{file path}` | {change} |
| `{file path}` | {change} |
| `{file path}` | {change} |

### 4.2 Configuration

{Description of related configuration}

```yaml
# {config file}
{config contents}
```

---

## 5. Debugging guide

### 5.1 Log inspection

```bash
# Inspect related errors in {log type}
{command}
```

### 5.2 Direct API tests

```bash
# Test {endpoint 1}
{curl command}

# Test {endpoint 2}
{curl command}
```

### 5.3 {Other debugging method}

```bash
{command}
```

---

## 6. Checklist

Checklist for {problem type}:

- [ ] {check item 1}
- [ ] {check item 2}
- [ ] {check item 3}
- [ ] {check item 4}

---

## 7. Related docs

- [{doc 1 title}]({link})
- [{doc 2 title}]({link})
- [{doc 3 title}]({link})

---

*Created: YYYY-MM-DD*
*Author: {author}*
```

---

## Worklog template

A real-time record template of an incident investigation.

```markdown
# {Problem topic} Troubleshooting Log

**Date**: YYYY-MM-DD
**Status**: resolved / in progress / on hold
**Severity**: Critical / High / Medium / Low
**Time spent**: {duration}

---

## Problem context

### Symptoms

{What was wrong}

```
{error message or log}
```

### Reproduction

1. {step 1}
2. {step 2}
3. {step 3}

### Environment

- OS: {os}
- Runtime: {runtime and version}
- Framework: {framework and version}

---

## Investigation

### HH:MM - Initial inspection

{What you checked first}

```bash
{commands run}
```

Result: {description of result}

### HH:MM - {investigation step 2}

{Details}

**Hypothesis**: {hypothesis}

**Validation**:
```bash
{validation command}
```

Result: {result}

### HH:MM - {investigation step 3}

{Details}

Findings:
- {finding 1}
- {finding 2}

---

## Attempts

### Attempt 1: {solution description}

```{language}
{applied code}
```

Result: {success/failure} - {description}

### Attempt 2: {solution description}

{Attempt details}

Result: {success/failure} - {description}

---

## Final resolution

### Root cause

{Root-cause description}

### Resolution

{Applied resolution}

```{language}
{fix code}
```

### Verification

```bash
{verification command}
```

Result: {normal operation confirmed}

---

## Related commits

- `{commit-hash}`: {commit message}
- `{commit-hash}`: {commit message}

---

## Lessons

1. {lesson 1}
2. {lesson 2}
3. {lesson 3}

---

## References

- {reference doc/link 1}
- {reference doc/link 2}

---

*Authored: YYYY-MM-DD*
*Author: {author}*
```

---

## Simple incident-record template

A simple format you can paste directly into index.md.

```markdown
### {number}. {problem title}

**Problem**
- {one-line problem description}

**Symptoms**
```
{error message}
```

**Cause**
{cause description}

**Fix**
```bash
{fix command}
```

**Prevention**
- {prevention measure}

---
```

---

## Template selection guide

| Situation | Template |
|-----------|----------|
| Simple problem, quick fix | Add the simple format inline in index.md |
| Complex problem, investigation worth recording | Worklog template (worklogs/) |
| Likely to recur, worth sharing | Topic guide template |
| Bundle of related problems | Topic guide template |

---

## File-naming conventions

### Topic guides

```
{kebab-case-topic}.md

Examples:
- backend-auth-issues.md
- database-connection.md
- performance-optimization.md
- docker-container-issues.md
```

### Worklogs

```
YYYY-MM-DD-{kebab-case-topic}.md

Examples:
- 2026-01-03-auth-fix.md
- 2026-01-02-nodejs-version.md
- 2026-01-02-gitlab-tab.md
```

---

## Authoring tips

### Traits of a good document

1. **Reproducible**: someone else can reproduce the problem
2. **Searchable**: include the error message verbatim
3. **Copy-pasteable**: the fix command can be copied and used directly
4. **Chronological**: investigation is recorded in order

### Avoid

1. Summarizing error messages (keep originals)
2. Omitting failed attempts (they help others)
3. Missing environment info (essential for repro)
4. "I'll write it later" (record it now)
