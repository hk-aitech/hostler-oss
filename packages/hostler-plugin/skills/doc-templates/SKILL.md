---
name: doc-templates
description: Provide hostler's standard document templates — 40+ templates spanning ADR, design, Sprint/Task, review, KB, runbook, test report, and more. Use this skill whenever the user mentions creating a new doc from a template, asks for an "ADR template", "design doc template", "runbook template", "frontmatter schema", or says things like "scaffold a doc with the right frontmatter" — even when they don't explicitly say "doc-templates". Do NOT use for reviewing existing docs (use doc-review).
compatibility:
  tools: [Read, Write, Edit, Glob]
paths: ["docs/**/*", "**/*.md"]
---

# Document Templates Skill

> Document template guide

---

## Table of contents

1. Template categories
2. Project templates
3. ADR template
4. Design templates
5. Sprint / Task templates
6. Operations templates
7. Test Report templates
8. Review templates
9. Knowledge templates
10. File location quick reference
11. Reference Files
12. Related skills

---

## Template categories

| Category | Templates | Purpose | Details |
|----------|-----------|---------|---------|
| Project | 3 | Project definition | `references/project-templates.md` |
| ADR | 1 | Architecture decisions | `references/adr-template.md` |
| Design | 7 | Design documents | `references/design-templates.md` |
| Sprint | 6 | Sprint management | See `sprint-management` skill |
| Task | 1 | Task definition | See `task-management` skill |
| Operations | 5 | Operations docs | `references/operations-templates.md` + `operability` skill templates/ |
| Test Report | 3 | Test reports | `references/test-report-templates.md` |
| Review | 5 | Review reports | See `review-management` skill |
| Knowledge | 2 | Knowledge base | `references/knowledge-templates.md` |
| Requirements | 4 | Requirements | `references/requirements-templates.md` |
| Research | 3 | Research reports | `references/research-report-templates.md` |
| Worklog | 3 | Worklogs | `references/worklog-templates.md` |

> **UX design**: 6 dedicated templates in the `ux-design` skill
> **Troubleshooting**: see the `troubleshooting` skill

---

## Project templates

Define project purpose, goals, scope, and roadmap. Detailed templates in `references/project-templates.md`.

| Template | Location | Key sections |
|----------|----------|--------------|
| PDD | docs/00-project/ | Vision, goals, scope, constraints, tech stack |
| Roadmap | docs/00-project/ | Phase overview, milestones, dependencies |
| Phase Spec | docs/00-project/phases/ | Goals, features, Sprint plan, Done criteria |

---

## ADR template

Records architecture decisions. Detailed template and authoring guide: `references/adr-template.md`.

**Location**: docs/02-architecture/adrs/

**Required sections**: Status, Context, Considered options, Decision, Consequences (positive / negative / mitigation), References

**Status**: proposed → accepted → deprecated / superseded

---

## Design templates

7 design-doc templates. Details: `references/design-templates.md`.

| Template | Location | Key sections |
|----------|----------|--------------|
| Architecture Overview | docs/03-design/ | Architecture principles, components, communication, security |
| Feature Design | docs/03-design/features/ | Requirements, architecture, data model, API |
| Domain Design | docs/03-design/domain/ | Aggregate, Entity, Value Object, Events |
| API Design | docs/03-design/api/ | Endpoints, auth, error codes |
| Config Design | docs/03-design/ | Config structure, per-environment, secrets |
| Deploy Design | docs/03-design/ | Environments, containers, CI/CD, rollback |
| IaC Design | docs/03-design/ | Modules, resources, state management |

---

## Sprint / Task templates

Sprint and Task documents are managed by dedicated skills:

- **Sprint**: `sprint-management` skill — SPRINT.md, Review, Retrospective
- **Task**: `task-management` skill — Task Definition (8 frontmatter fields)

**Sprint location**: `works/sprints/{active|backlog|completed}/sprint-NN/SPRINT.md`
**Task location**: `works/sprints/.../sprint-NN/tasks/TNN-title.md`

---

## Operations templates

3 operations-doc templates. Details: `references/operations-templates.md`.

| Template | Location | Key sections |
|----------|----------|--------------|
| Runbook | docs/05-runbooks/ | Start/stop, healthcheck, incident response |
| Playbook | docs/05-runbooks/ | Scenarios, decision flow, escalation |
| Monitoring | docs/05-runbooks/ | Metrics, dashboards, alert rules |

> **Operability**: BDM/FMA/PRR/SLO live in the `operability` skill

---

## Test Report templates

3 test-report templates. Details: `references/test-report-templates.md`.

| Template | Location |
|----------|----------|
| Phase Test Report | docs/06-reports/testing/ |
| Sprint Test Report | docs/06-reports/testing/ |
| Feature Test Report | docs/06-reports/testing/ |

---

## Review templates

Detailed templates live in the `review-management` skill's `references/review-templates.md`.

| Template | File | Key sections |
|----------|------|--------------|
| Sprint review | `review-management/templates/sprint-review.md` | Completed Tasks, SOLID principles, tech debt, overall grade |
| Phase review | `review-management/templates/phase-review.md` | Sprint summary, architecture assessment, debt summary, next-Phase prep |
| Code review report | `review-management/templates/code-review-report.md` | Issue triage (Critical/Major/Minor), quality metrics, security check |
| Design review | `review-management/templates/design-review.md` | Requirements review, architecture fit, risk analysis, approval decision |
| Review folder README | `review-management/templates/review-folder-readme.md` | Folder structure, process, evaluation criteria, grading guide |

---

## Knowledge templates

2 templates for lessons learned and analysis reports. Details: `references/knowledge-templates.md`.

| Template | Location | Key sections |
|----------|----------|--------------|
| Lessons Learned | docs/07-knowledge/ | Context, problem, solution, lessons, application |
| Analysis Report | docs/06-reports/analysis/ | Methodology, findings, conclusion, recommendations |

---

## File location quick reference

| Template type | File location | Template source |
|---------------|---------------|-----------------|
| PDD | docs/00-project/pdd.md | `references/project-templates.md` |
| Roadmap | docs/00-project/roadmap.md | `references/project-templates.md` |
| Phase Spec | docs/00-project/phases/ | `references/project-templates.md` |
| ADR | docs/02-architecture/adrs/ | `references/adr-template.md` |
| Architecture | docs/03-design/ | `references/design-templates.md` |
| Feature Design | docs/03-design/features/ | `references/design-templates.md` |
| Domain Design | docs/03-design/domain/ | `references/design-templates.md` |
| API Design | docs/03-design/api/ | `references/design-templates.md` |
| Runbook | docs/05-runbooks/ | `references/operations-templates.md` |
| Playbook | docs/05-runbooks/ | `references/operations-templates.md` |
| Monitoring | docs/05-runbooks/ | `references/operations-templates.md` |
| SLA / SLO | docs/03-design/operations/ | `operability` skill `templates/slo-definition.md` |
| BDM Scenario | docs/03-design/operations/ | `operability` skill |
| PRR Report | docs/06-reports/operability/ | `operability` skill |
| Sprint Overview | `works/sprints/{active\|backlog\|completed}/sprint-NN/SPRINT.md` | `sprint-management` skill |
| Task | `works/sprints/.../sprint-NN/tasks/TNN-title.md` | `task-management` skill |
| Phase Test Report | docs/06-reports/testing/ | `references/test-report-templates.md` |
| Sprint Test Report | docs/06-reports/testing/ | `references/test-report-templates.md` |
| Analysis Report | docs/06-reports/analysis/ | `references/knowledge-templates.md` |
| Lessons Learned | docs/07-knowledge/ | `references/knowledge-templates.md` |
| Incident Report | docs/06-reports/incidents/ | `references/operations-templates.md` |

---

## Reference Files

> Detailed templates live under references/

| File | Contents |
|------|----------|
| `references/adr-template.md` | Detailed ADR template + authoring guide |
| `references/design-templates.md` | 7 design-doc templates |
| `references/project-templates.md` | PDD, Roadmap, Phase spec |
| `references/knowledge-templates.md` | Lessons learned, analysis report |
| `references/requirements-templates.md` | SRS, FRS, NFR, RTM requirements |
| `references/research-report-templates.md` | Deep research, quality comparison, overall evaluation |
| `references/operations-templates.md` | Runbooks, deployment, monitoring |
| `references/test-report-templates.md` | Phase / Sprint / Feature test reports |
| `references/worklog-templates.md` | Worklogs, session logs, retrospectives |

---

## Related skills

- `sprint-management` — Sprint document creation and management
- `task-management` — Task document creation and management
- `ux-design` — 6 UX design-doc templates
- `operability` — BDM/FMA/PRR/SLO operability docs
- `review-management` — Review process
- `troubleshooting` — Problem-solving workflow and templates
- `project-structure` — Project folder structure
