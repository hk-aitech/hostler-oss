# Project Initialization Detailed Guide

> Detailed reference for the `project-structure` skill's initialization procedure.
> The `/hstl-oss:project:init` command consults this guide to bootstrap a project.

---

## On `/project:init` Execution

1. Create the project folder
2. Create the standard directory structure from the `project-structure` skill
3. Create README.md
4. Create CLAUDE.md
5. Create .gitignore
6. Initialize works/
7. Create the default docs/ structure (with numeric prefixes)

> **The Single Source of Truth for the directory structure is the `project-structure` skill's SKILL.md.**
> This document only defines the **minimum file list** to create at init time.

---

## Directories Created

The directories created at init time, derived from the `project-structure` skill's standard:

```bash
# Documentation structure (numeric prefixes mandatory,
# 10-slot ADR baseline)
mkdir -p docs/00-project
mkdir -p docs/02-architecture
# mkdir -p docs/01-requirements  # optional — requirements (FR, NFR)
mkdir -p docs/03-design/{architecture,domain,features,operations,ux,api,deployment}
mkdir -p docs/04-guides/{development,operations,troubleshooting}
mkdir -p docs/06-reports/{analysis,audits,checklists,measurements,spikes}
mkdir -p docs/07-knowledge/{api,architecture,domain,operations,mistakes,migration}
mkdir -p docs/08-references/{api,research}

# Source / tests / infra
mkdir -p src tests config scripts db

# Work management
mkdir -p works/sprints/{active,backlog,completed}
mkdir -p works/{worklogs,tasks}

# Archive
mkdir -p archive

# Claude configuration
mkdir -p .claude/context
```

## Files Created

```bash
# Project root
README.md
CLAUDE.md
.gitignore

# Documentation index
docs/index.md
docs/00-project/index.md
docs/00-project/pdd.md
docs/00-project/roadmap.md
docs/02-architecture/index.md
docs/03-design/index.md
docs/04-guides/index.md
docs/04-guides/troubleshooting/index.md

# Architecture rules
docs/03-design/architecture/banned-patterns.md

# Work management
works/CURRENT-FOCUS.md
works/worklogs/.gitkeep
works/tasks/BACKLOG.md
works/sprints/active/.gitkeep
works/sprints/backlog/.gitkeep
works/sprints/completed/.gitkeep

# Claude configuration
.claude/context/work-units.md
.claude/context/conventions.md
```

---

## Recommended `.gitignore` Entries

At project init, generate the following as the default `.gitignore`.

```gitignore
# IDE
.idea/
.vscode/
*.swp

# Build
bin/
obj/
dist/
node_modules/

# Local settings
*.local.json
*.local.md
.env.local

# OS
.DS_Store
Thumbs.db
```

---

## Post-init Checklist

After initialization, verify:

1. Fill in TODOs in `docs/00-project/pdd.md`.
2. Lay out Phase / Sprint planning in `docs/00-project/roadmap.md`.
3. Record the current goal in `works/CURRENT-FOCUS.md`.
4. Add project-specific entries to `.gitignore`.
5. Make the first commit: `git init && git add -A && git commit -m "init: project initialization"`.
