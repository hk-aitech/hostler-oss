---
name: project-structure
description: SSOT for the standard project folder structure — `docs/` (00–08 prefix per the architecture decision record),
  `src/`, `tests/`, `works/` (Sprints/Tasks), `archive/`, `.claude/`. Use this skill whenever the user asks where a file
  should live, how to set up folders, what a numeric prefix means, or any phrasing like "what's the folder layout",
  "where do reports go", or "set up the docs structure". Be a little pushy — invoke this even when the user only loosely
  hints at file placement. Do NOT use for archiving obsolete files (use the archive skill).
compatibility:
  tools: [Read, Glob]
paths: ["docs/**/*", "works/**/*", "archive/**/*"]
user-invocable: false
---

# Project Structure Skill

> Project folder structure guide

---

## Table of Contents

1. Core principles
2. Standard project structure
3. Directory naming rules
4. Folder usage
5. File naming rules
6. Project initialization procedure
7. Language conventions
8. Related documents
9. Additional resources

---

## Core Principles

```
1. Consistent structure for easy project navigation
2. Numeric prefixes that make document order explicit
3. Place configuration where it is used (Locality of Configuration)
```

---

## Standard Project Structure

```
{project-name}/
├── docs/                         # Documentation
│   ├── index.md                  # Documentation entry point
│   ├── 00-project/               # Project definition
│   │   ├── index.md
│   │   ├── pdd.md                # Project Definition Document
│   │   ├── roadmap.md            # Roadmap (per-Phase milestones)
│   │   ├── implementation-plan.md # Implementation plan (Task allocation per Sprint)
│   │   ├── system-overview.md    # System overview
│   │   ├── quality-attributes.md # Quality attributes
│   │   └── domain-glossary.md    # Domain glossary
│   ├── 01-requirements/          # Requirements
│   │   ├── SRS.md                # System requirements
│   │   ├── FRS.md                # Functional requirements
│   │   ├── NFR.md                # Non-functional requirements
│   │   └── RTM.md                # Requirements Traceability Matrix
│   ├── 02-architecture/          # Architecture decisions + views
│   │   ├── index.md
│   │   └── ADR-NNN-title.md      # Architecture Decision Records
│   ├── 03-design/                # Design documents
│   │   ├── index.md
│   │   ├── architecture/         # Architectural design
│   │   ├── domain/               # Domain design
│   │   ├── features/             # Feature design
│   │   ├── ux/                   # UX design
│   │   ├── api/                  # API design
│   │   └── deployment/           # Deployment design
│   ├── 04-guides/                # Guides (development + operations)
│   │   ├── index.md
│   │   ├── development/          # Development / setup how-to
│   │   ├── operations/           # Operations guides (BDM scenarios, PRR reports)
│   │   ├── troubleshooting/      # Troubleshooting (absorbs former 06-troubleshooting)
│   │   ├── runbooks/             # Operational runbooks (single process)
│   │   └── playbooks/            # Operational playbooks (compound process)
│   ├── 05-operations/            # Operations specs (SLO, FMA, Runbook, monitoring)
│   │   ├── slo-definitions.md    # SLO definitions
│   │   ├── failure-mode-analysis.md
│   │   ├── monitoring/           # Monitoring strategy
│   │   ├── data-management/      # Data retention / backup policy
│   │   └── runbooks/             # Operational runbooks (separately operated from 04-guides)
│   ├── 06-reports/               # Reports (analysis, measurements, audits, spikes)
│   │   ├── analysis/             # Analysis / verification reports
│   │   ├── audits/               # Audit reports
│   │   ├── measurements/         # Performance / metrics measurements
│   │   ├── checklists/           # Checklist reports
│   │   └── spikes/               # Research / comparison / spike results
│   ├── 07-knowledge/             # Knowledge base
│   │   ├── api/                  # API lessons
│   │   ├── architecture/         # Design decisions
│   │   ├── domain/               # Domain knowledge
│   │   ├── operations/           # Operations lessons
│   │   └── mistakes/             # Recurrence prevention
│   └── 08-references/            # External reference materials (API docs, specs, external mappings)
│       ├── api/                  # External API documents
│       └── research/             # External research / domain materials
│
├── src/                          # Source code
├── tests/                        # Tests
├── config/                       # Configuration files (yaml, etc.)
├── scripts/                      # Scripts
├── db/                           # DB migrations
│
├── works/                        # Work management
│   ├── CURRENT-FOCUS.md          # Current progress (CLI-managed)
│   ├── worklogs/                 # Daily worklogs (optional)
│   │   └── YYYY-MM-DD.md
│   ├── tasks/                    # Sprint-unassigned Tasks
│   │   ├── BACKLOG.md            # Unassigned Task index
│   │   ├── T*.md                 # Active unassigned Tasks (todo, in-progress)
│   │   └── completed/            # Completed hotfix / ad-hoc Tasks (done)
│   │       └── T*.md
│   └── sprints/                  # Sprint management
│       ├── active/
│       ├── backlog/
│       └── completed/
│
├── archive/                      # Archive (project root)
│   ├── docs/                     # Old document versions
│   ├── src/                      # Retired source code
│   ├── config/                   # Old configuration files
│   ├── data/                     # Old data, previous models
│   └── ...                       # Mirror of the project root structure
│
├── .claude/                      # Claude configuration
│   ├── commands/
│   ├── skills/
│   ├── agents/
│   └── context/
│
├── user-project-guide            # Claude Code main context
├── README.md                     # Project README
└── .gitignore
```

---

## Directory Naming Rules

### docs/ Subfolders

| Rule | Format | Example |
|------|------|------|
| Order prefix | `NN-` (2-digit number) | `00-`, `01-`, `99-` |
| Word separator | hyphen `-` | `01-architecture` |
| Case | lowercase kebab-case | `user-guide` |

### Prefix Allocation Standard (per ADR-002)

> The SSOT for the numeric semantics is the user project docs.
> This table is a summary; on conflict, the ADR wins.

| Prefix | Folder | Purpose |
|--------|------|------|
| `00-` | project | Project definition, roadmap, specifications |
| `01-` | requirements | Requirements (SRS, FRS, NFR, RTM) |
| `02-` | architecture | Architecture decisions + views (includes ADRs) |
| `03-` | design | Design documents |
| `04-` | guides | Development / operations guides (includes troubleshooting) |
| `05-` | operations | Operations specs (SLO, FMA, Runbook, monitoring) |
| `06-` | reports | Reports (analysis, measurements, audits, spikes) |
| `07-` | knowledge | Knowledge base |
| `08-` | references | External reference materials (API docs, specs, external mappings) |

> **archive/**: lives at the project root. Stores **every kind of file**
> (documents, source, configuration, data) that needs to be retired or kept.
> Mirrors the project-root directory structure so the original location can be
> traced back. Examples: user project docs → `archive/<original-path>`,
> `src/old_module/` → `archive/src/old_module/`.

---

## Folder Usage

### docs/ — Documentation

| Folder | Purpose | Key files |
|------|------|----------|
| `00-project/` | Project definition | pdd.md, roadmap.md, implementation-plan.md |
| `01-requirements/` | Requirements | SRS.md, FRS.md, NFR.md, RTM.md |
| `02-architecture/` | Architecture decisions + views | ADR-NNN-title.md |
| `03-design/` | Design documents | architecture, domain, feature design, UX, API |
| `04-guides/` | Guides in general | development/, operations/, troubleshooting/, runbooks/ |
| `05-operations/` | Operations specs | slo-definitions.md, runbooks/, monitoring/, data-management/ |
| `06-reports/` | Reports | analysis/, audits/, measurements/, checklists/, spikes/ |
| `07-knowledge/` | Knowledge base | API, design, domain, operations, recurrence prevention |
| `08-references/` | External references | external API docs, external research / domain materials |

### `<docs>/04-guides/` (or equivalent guide directory) — substructure example

```
<docs>/guides/
├── development/      # Development / setup how-to (API integration, tool setup, etc.)
├── operations/       # Operations guides (BDM scenarios, PRR reports)
├── runbooks/         # Operational runbooks (single process — start / stop commands)
└── playbooks/        # Operational playbooks (compound process — deployment, incident response)
```

### `<docs>/05-operations/` (or equivalent operations directory) — substructure example

```
<docs>/operations/
├── slo-definitions.md      # SLO definitions
├── failure-mode-analysis.md
├── alert-slo-verification.md
├── monitoring/             # Monitoring strategy
│   └── monitoring-strategy.md
├── data-management/        # Data retention / backup policy
│   ├── data-retention-backup-policy.md
│   └── infra-data-management.md
└── runbooks/               # Operational runbooks
    └── {process}-runbook.md
```

### `<docs>/06-reports/` (or equivalent reports directory) — substructure example

```
<docs>/reports/
├── analysis/         # Analysis / measurement / verification reports
│   ├── {topic}-analysis-YYYY-MM-DD.md
│   └── {scope}-verification-YYYY-MM-DD.md
├── audits/           # Audit reports (deep-audit, gap-audit, etc.)
│   └── {scope}-audit-YYYY-MM-DD.md
├── measurements/     # Performance / metrics measurements
│   └── {metric}-measurement-YYYY-MM-DD.md
├── checklists/       # Checklist reports
│   └── {scope}-checklist-YYYY-MM-DD.md
└── spikes/           # Research / comparison / spike results
    └── {topic}-research-YYYY-MM-DD.md
```

### user project docs — Troubleshooting (absorbs former 06-troubleshooting)

> Troubleshooting documents are integrated into the user project docs.
> The former `06-troubleshooting/` numeric slot was reused for `06-reports` (ADR-002).

```
user project docs
├── index.md                        # Resolved-issue summary (Quick Reference)
└── YYYY-MM-DD-{topic}.md           # Detailed troubleshooting log
```

### works/ — Work management

| File / folder | Purpose |
|----------|------|
| `CURRENT-FOCUS.md` | Current progress (auto-updated by sprint:start/complete) |
| `worklogs/YYYY-MM-DD.md` | Daily worklog (optional; with hostler, prefer the result section of the Task body) |
| `tasks/BACKLOG.md` | Unassigned Task index (CLI-generated, regenerated by `hstl-oss backlog rebuild-md`) |
| `tasks/T*.md` | **Active unassigned Tasks** (todo, in-progress) |
| `tasks/completed/T*.md` | **Completed unassigned hotfix / ad-hoc Tasks** (auto-moved) |
| `sprints/active/sprint-NN/` | Active Sprint (SPRINT.md + tasks/) |
| `sprints/backlog/sprint-NN/` | Future Sprint |
| `sprints/completed/sprint-NN/` | Completed Sprint |

### .claude/ — Claude configuration

| Folder | Purpose |
|------|------|
| `commands/` | Custom slash commands |
| `skills/` | Project-specific skills |
| `agents/` | Agent definitions |
| `context/` | Context documents |

---

## File Naming Rules

### Documentation Files

| Type | Format | Example |
|------|------|------|
| Index file | `index.md` | 1 per section, required |
| Project doc | `kebab-case.md` | `pdd.md`, `roadmap.md` |
| ADR | `ADR-NNN-title.md` | `ADR-001-platform-selection.md` |
| Phase spec | `phase{N}-spec.md` | `phase1-spec.md` |
| Guide | `NNN-title.md` | `001-overview.md` |
| Feature design | `feature-name.md` | `user-authentication.md` |
| Learning entry | `L-NNN-title.md` | `L-001-lesson-title.md` |

### Work Files

| Type | Format | Example |
|------|------|------|
| Sprint folder | `sprint-NN` | `sprint-NN` (2-digit zero-padding recommended) |
| Task file | `TNN-title.md` (project conventions respected) | `T001-auth.md` |
| Sprint overview | `SPRINT.md` | - |

### Task Move Flow

> **This section is the SSOT for the Task move flow.** The
> "Sprint-unassigned Task auto folder move" section in
> `skills/task-management/SKILL.md` is a summary of the behaviour at
> `hstl-oss task complete/reopen` time; if they conflict, this section wins.

Task file moves split into **two flows depending on Sprint assignment**.

**1. Sprint-assigned Task** (works/sprints/{backlog|active|completed}/sprint-NN/tasks/T*.md)

- Moves with the Sprint folder. Sprint-level git mv, automatically performed by sprint:start / sprint:complete ceremonies.
- Individual Tasks are never moved alone. Updating the status frontmatter is enough.

```bash
# At Sprint completion (automatic, hstl-oss sprint complete)
git mv works/sprints/active/sprint-NN works/sprints/completed/sprint-NN
```

**2. Sprint-unassigned hotfix / ad-hoc Task** (works/tasks/T*.md)

- On `hstl-oss task complete`, individually moved to `works/tasks/completed/T*.md` along with the done transition (automatic).
- On `hstl-oss task reopen`, moved back to `works/tasks/T*.md`.

```bash
# Automatic (hstl-oss task complete)
works/tasks/T###-fix-bug.md → works/tasks/completed/T###-fix-bug.md
```

The **status frontmatter and folder location must always agree**. If they
diverge from manual editing, `hstl-oss backlog sync` will reconcile.

---

## Project Initialization Procedure

At project initialization, create the standard directory structure defined in this skill.
The `/hstl-oss:project:init` command references this skill, and the list of directories /
files to create is detailed in `references/init-guide.md`.

> **Numeric semantics SSOT**: user project docs — the prefix-purpose mapping is governed by that ADR.
> **File placement and initialization SSOT**: this SKILL.md — no other command/skill should redefine the file placement rules.

---

## Language Conventions

| Item | Language |
|------|------|
| Document body | Project default language |
| Code examples | English (variable / class names) |
| Technical terms | Original term recommended |
| File names | English kebab-case |

---

## Related Documents

- Document templates: `doc-templates` skill
- UX design: `ux-design` skill

---

## Tech-stack-specific Structure Standards

This skill auto-detects files at the project root to apply additional rules per tech stack.

| Detection file | Tech stack | Additional rules |
|----------|---------|---------|
| `*.sln` or `*.slnx` | .NET | [dotnet-structure-standard.md](references/dotnet-structure-standard.md) |
| `package.json` + `tsconfig.json` | Node.js / TypeScript | (to be added) |
| `pyproject.toml` or `setup.py` | Python | (to be added) |
| `go.mod` | Go | (to be added) |

> If the project has a `.hostler/project-config.md`, additional rules from there take precedence.

## Per-project-type Manifests

Depending on the project type, reference one of three manifest YAMLs to apply required directories and file structure.

| Type | Manifest | Characteristics |
|------|-----------|------|
| **Enterprise** | `references/manifest-enterprise.yaml` | Large team, operations specs (SLO/FMA/DR) required, ADR required |
| **Plugin** | `references/manifest-plugin.yaml` | Claude Code plugins / tools, skills/commands structure, applies to hostler-plugin |
| **Mobile** | `references/manifest-mobile.yaml` | iOS / Android / Flutter, UX-first, API / release-centric |

**Manifest structure**: each YAML has `required_dirs`, `optional_dirs`, `template_ref`.
- `template_ref: doc-templates` — content standards live in the `doc-templates` skill
- `pattern: true` — existence is verified by file pattern (glob), not exact filename
- `optional_dirs` — directories that may be omitted depending on project characteristics

**On project init**: read the appropriate manifest and create the directories + files in `required_dirs`.
A future unified `project init` CLI will automate from this manifest (today this is split between
`hstl-oss project init-skeleton` and `hstl-oss project init-docs`).

## Additional Resources

- Detailed initialization guide: `references/init-guide.md` — initialization procedure, list of files to create, recommended `.gitignore` entries
- Changelog (maintainer-only): `references/changelog.md`
- .NET project structure standard: `references/dotnet-structure-standard.md` — Clean Architecture 10 rules
- Enterprise project manifest: `references/manifest-enterprise.yaml`
- Plugin project manifest: `references/manifest-plugin.yaml`
- Mobile project manifest: `references/manifest-mobile.yaml`

## Per-project Extensions (auto-detected)

This skill auto-detects `.hostler/project-config.md` at the project root at runtime.
If that file exists and contains a `## Per-skill Extensions` → `### project-structure` section,
project-specific rules are applied **in addition** to the shared rules.

- Detection path: `{project_root}/.hostler/project-config.md`
- If absent, only the shared rules apply (backward compatible)
