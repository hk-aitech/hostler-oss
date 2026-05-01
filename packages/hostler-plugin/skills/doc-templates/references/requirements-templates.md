# Requirements document templates

> Four templates for SRS, FRS, NFR, and RTM, plus an ID scheme guide

## Table of contents

1. [SRS template](#srs-system-requirements-specification-template)
2. [FRS template](#frs-functional-requirements-specification-template)
3. [NFR template](#nfr-non-functional-requirements-template)
4. [RTM template](#rtm-requirements-traceability-matrix-template)
5. [Requirements ID scheme](#requirements-id-scheme)

## SRS (System Requirements Specification) template

```markdown
# {Project name} — System Requirements Specification

> Version: {N.N.N}
> Last updated: {YYYY-MM-DD}

## 1. Scope

| Item | Content |
|------|---------|
| System | {system description} |
| Users | {user types} |
| Purpose | {core purpose} |

## 2. Stakeholder requirements

| ID | Requirement | Priority | Notes |
|----|-------------|----------|-------|
| SR-001 | {requirement} | P0 | |

## 3. System constraints

| Constraint | Description |
|------------|-------------|
| {constraint name} | {description} |

## 4. KPIs

| Metric | Target | Current |
|--------|--------|---------|
| {metric name} | {target} | {current} |
```

---

## FRS (Functional Requirements Specification) template

```markdown
# {Project name} — Functional Requirements Specification

> Version: {N.N.N}
> Last updated: {YYYY-MM-DD}

## Functional requirements

### {Area code}: {area name}

| ID | Feature | Description | Acceptance criteria | Status |
|----|---------|-------------|---------------------|--------|
| FR-{area}-001 | {feature} | {description} | {acceptance} | implemented / pending |

## Acceptance-criteria authoring guide

GIVEN {precondition}
WHEN {action}
THEN {expected outcome}

### Example

GIVEN the surge-signal score is 0.8 or higher
WHEN the Slack alert dispatch condition is satisfied
THEN a Slack message is received within 30 seconds
```

---

## NFR (Non-Functional Requirements) template

```markdown
# {Project name} — Non-Functional Requirements

> Version: {N.N.N}
> Last updated: {YYYY-MM-DD}

| ID | Category | Requirement | Target | Measurement |
|----|----------|-------------|--------|-------------|
| NFR-PF-001 | Performance | {requirement} | {target} | {measurement} |
| NFR-RL-001 | Reliability | {requirement} | {target} | {measurement} |
| NFR-SE-001 | Security | {requirement} | {target} | {measurement} |
| NFR-MT-001 | Maintainability | {requirement} | {target} | {measurement} |
| NFR-SC-001 | Scalability | {requirement} | {target} | {measurement} |

## Category codes

| Code | Meaning |
|------|---------|
| PF | Performance |
| RL | Reliability |
| SE | Security |
| MT | Maintainability |
| SC | Scalability |
```

---

## RTM (Requirements Traceability Matrix) template

```markdown
# {Project name} — Requirements Traceability Matrix

> Version: {N.N.N}
> Last updated: {YYYY-MM-DD}

| Requirement ID | Requirement | Design | Implementation file | Test | Status |
|----------------|-------------|--------|---------------------|------|--------|
| FR-SCAN-001 | {requirement} | {design doc} | `{file path}` | `{test file}` | implemented / pending |
```

---

## Requirements ID scheme

| Type | Pattern | Example |
|------|---------|---------|
| Functional | FR-{area}-{number} | FR-SCAN-001 |
| Non-functional | NFR-{category}-{number} | NFR-PF-001 |
| System | SR-{number} | SR-001 |
| Use case | UC-{number} | UC-003 |

### Area code examples (adjust per project)

| Code | Area |
|------|------|
| SCAN | Symbol scanning |
| FEAT | Feature computation |
| ALRT | Alert dispatch |
| DATA | Data ingestion |
| AUTH | Auth / security |
| MON | Monitoring |
