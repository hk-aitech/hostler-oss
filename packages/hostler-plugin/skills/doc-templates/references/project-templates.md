# Project document templates

> Three templates for project management

---

## Table of contents

1. [PDD (Project Definition Document)](#1-pdd-template)
2. [Roadmap](#2-roadmap-template)
3. [Phase Spec](#3-phase-spec-template)

---

## 1. PDD template

```markdown
# Project Definition Document (PDD)

## Project overview

| Item | Content |
|------|---------|
| Project name | {project name} |
| Version | 1.0 |
| Date | {YYYY-MM-DD} |
| Author | {author} |
| Status | Draft / Review / Approved |

---

## 1. Vision and goals

### 1.1 Vision

{The ultimate vision the project pursues}

### 1.2 Goals

1. {goal 1}
2. {goal 2}
3. {goal 3}

### 1.3 Success criteria

| Criterion | Measurement | Target |
|-----------|-------------|--------|
| {criterion 1} | {measurement} | {value} |
| {criterion 2} | {measurement} | {value} |

---

## 2. Scope

### 2.1 In Scope

- {included 1}
- {included 2}

### 2.2 Out of Scope

- {excluded 1}
- {excluded 2}

---

## 3. Stakeholders

| Role | Owner | Responsibility |
|------|-------|----------------|
| Project owner | {name} | {responsibility} |
| Developer | {name} | {responsibility} |

---

## 4. Constraints

### 4.1 Technical constraints

- {tech constraint 1}

### 4.2 Business constraints

- {business constraint 1}

### 4.3 Schedule constraints

- {schedule constraint 1}

---

## 5. Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| {risk 1} | high/medium/low | high/medium/low | {mitigation} |

---

## 6. Tech stack

| Category | Technology | Version |
|----------|------------|---------|
| Language | {language} | {version} |
| Framework | {framework} | {version} |
| Database | {DB} | {version} |

---

## 7. Milestones

| Phase | Name | Goal | Estimated duration |
|-------|------|------|--------------------|
| Phase 1 | {name} | {goal} | {duration} |
| Phase 2 | {name} | {goal} | {duration} |

---

## 8. References

- [Roadmap](roadmap.md)
- [Architecture](../design/architecture/architecture-overview.md)

---

## Change history

| Version | Date | Change | Author |
|---------|------|--------|--------|
| 1.0 | {date} | Initial draft | {author} |
```

---

## 2. Roadmap template

```markdown
# Project Roadmap

## Overview

| Item | Content |
|------|---------|
| Project | {project name} |
| Version | 1.0 |
| Last updated | {YYYY-MM-DD} |

---

## Phase overview

```
Phase 1          Phase 2          Phase 3          Phase 4
    │                │                │                │
    ▼                ▼                ▼                ▼
┌────────┐      ┌────────┐      ┌────────┐      ┌────────┐
│  MVP   │ ──▶  │ Expand │ ──▶  │Mature  │ ──▶  │Stabilize│
└────────┘      └────────┘      └────────┘      └────────┘
```

---

## Phase details

### Phase 1: {Phase name}

| Item | Content |
|------|---------|
| Goal | {goal} |
| Duration | {duration} |
| Sprint count | {N} |
| Status | 📋 planned / 🔄 in progress / ✅ done |

**Key features:**
- {feature 1}
- {feature 2}

**Deliverables:**
- {deliverable 1}
- {deliverable 2}

---

### Phase 2: {Phase name}

| Item | Content |
|------|---------|
| Goal | {goal} |
| Duration | {duration} |
| Sprint count | {N} |
| Status | 📋 planned / 🔄 in progress / ✅ done |

**Key features:**
- {feature 1}
- {feature 2}

---

## Milestones

| Milestone | Phase | Date | Status |
|-----------|-------|------|--------|
| {milestone 1} | Phase 1 | {date} | ⬜/✅ |
| {milestone 2} | Phase 2 | {date} | ⬜/✅ |

---

## Dependencies

```
Phase 1 ──▶ Phase 2 ──▶ Phase 3
              │
              └──▶ Phase 4 (parallel)
```

---

## Risk and mitigation

| Risk | Phase | Mitigation |
|------|-------|------------|
| {risk 1} | Phase 1 | {mitigation} |

---

## Change history

| Version | Date | Change |
|---------|------|--------|
| 1.0 | {date} | Initial draft |
```

---

## 3. Phase Spec template

```markdown
# Phase {N} Spec: {Phase name}

## Overview

| Item | Content |
|------|---------|
| Phase | Phase {N} |
| Name | {Phase name} |
| Duration | {start} ~ {end} |
| Sprint count | {N} |
| Status | 📋 planned / 🔄 in progress / ✅ done |

---

## Goals

### Headline goal

{Phase's headline goal in 1–2 sentences}

### Detailed goals

1. {goal 1}
2. {goal 2}
3. {goal 3}

---

## Key features

| Feature | Description | Priority |
|---------|-------------|----------|
| {feature 1} | {description} | P0/P1/P2 |
| {feature 2} | {description} | P0/P1/P2 |

---

## Technical requirements

### Functional

- {FR-001}: {requirement description}
- {FR-002}: {requirement description}

### Non-functional

- {NFR-001}: {requirement description}
- {NFR-002}: {requirement description}

---

## Sprint plan

| Sprint | Name | Period | Goal | Status |
|--------|------|--------|------|--------|
| p{N}-s1 | {name} | Week 1-2 | {goal} | ⬜ |
| p{N}-s2 | {name} | Week 3-4 | {goal} | ⬜ |

---

## Deliverables

| Deliverable | Description | Owner |
|-------------|-------------|-------|
| {deliverable 1} | {description} | {owner} |
| {deliverable 2} | {description} | {owner} |

---

## Dependencies

### Preconditions

- {precondition 1}
- {precondition 2}

### Effect on subsequent Phases

- {effect 1}

---

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| {risk 1} | high/medium/low | {mitigation} |

---

## Done criteria

- [ ] All Sprints completed
- [ ] Build green
- [ ] Tests pass (coverage ≥ {X}%)
- [ ] Documentation updated
- [ ] Review complete

---

## References

- [PDD](../pdd.md)
- [Roadmap](../roadmap.md)
- [Sprint plan](phase{N}-sprints.md)
```
