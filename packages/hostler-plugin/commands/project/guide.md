---
description: Plugin usage guide — command list, prompt examples, workflow walkthrough, template directory. Use when the user asks "how do I use this plugin?", "what commands are available?", or "show me example prompts".
allowed-tools: Read
argument-hint: "[topic]"
---

# Guide - Plugin Guide

## Arguments

- `topic`: optional topic
  - `commands` — command list
  - `prompts` — prompt examples
  - `workflow` — workflow guide
  - `templates` — template list

---

## Default help output

```markdown
# hostler plugin

A Claude Code plugin that supports a structured project development process.

## How the plugin is used

### Commands (slash commands)
Invoked directly by the user. **Always requires the `/hstl-oss:` prefix.**

### Skills
Claude **invokes them automatically** based on context. The user does not need to call them.

### Agents
Claude **selects them automatically** based on the nature of the work.

## Quick start

### 1. Start a new project
Describe the project naturally and the structure and documents are generated automatically:

> /hstl-oss:project:init I want to build a user authentication API using Python
> and FastAPI. We will use JWT-based authentication with PostgreSQL.

### 2. Start a session
> /hstl-oss:session:start

### 3. Start a task
> /hstl-oss:task:start Implement user registration API

## Key commands

| Command | Description |
|--------|------|
| `/hstl-oss:project:init <description>` | Initialize a project (interactive) |
| `/hstl-oss:session:start` | Start a session |
| `/hstl-oss:session:end` | End a session |
| `/hstl-oss:session:catchup` | Restore context |
| `/hstl-oss:task:start [id]` | Start a task |
| `/hstl-oss:task:next` | Recommend the next task |
| `/hstl-oss:task:complete` | Complete the current task |
| `/hstl-oss:project:guide [topic]` | Guide |

## Workflow

```
Start project
    |
    v
/hstl-oss:project:init -------------------+
    |                                     |
    v                                     |
[Gather information conversationally]     |
    |                                     |
    v                                     |
[Auto-generate structure + documents]     |
    |                                     |
    v                                     |
/hstl-oss:session:start <-----------------+
    |
    v
/hstl-oss:task:start
    |
    v
[Do the work]
    |
    v
/hstl-oss:task:complete
    |
    v
/hstl-oss:session:end
```

## Detailed guides

- `/hstl-oss:project:guide commands` — command details
- `/hstl-oss:project:guide prompts` — prompt examples
- `/hstl-oss:project:guide workflow` — workflow guide
- `/hstl-oss:project:guide templates` — template list
```

---

## topic: commands

```markdown
# Command details

## Project management

### /hstl-oss:project:init <project description>
Initializes a project. Extracts info from your description and asks for what's missing.

**Generated:**
- Folder structure (docs, src, tests, deploy, scripts, tools, works/)
- README.md, CLAUDE.md
- PDD (Project Definition Document) draft
- Roadmap draft
- Architecture Overview draft
- First ADR
- User Persona draft
- Phase 1 specification draft

## Session management

### /hstl-oss:session:start
Starts a new work session.
- Check current project state
- Load in-progress work
- Check git status

### /hstl-oss:session:end
Ends the session.
- Persist work state
- Update worklog (works/worklogs/YYYY-MM-DD.md)
- Save context for the next session

### /hstl-oss:session:catchup
Restores context from the previous session.
- Read CURRENT-FOCUS.md
- Inspect recent worklogs (works/worklogs/)
- Determine current work state

## Task management

### /hstl-oss:task:start [task-id]
Starts a task.
- With task-id: starts that task
- Without task-id: checks the current task or creates a new one

### /hstl-oss:task:next
Recommends the next task by priority.
- Picks the highest-priority task from the backlog
- Checks dependencies

### /hstl-oss:task:complete
Completes the current task.
- Updates task frontmatter status (done)
- On Sprint completion, moves works/sprints/active/ to completed/
- Updates CURRENT-FOCUS.md
```

---

## topic: prompts

```markdown
# Prompt examples

## Project initialization

### Example 1: API service
> /hstl-oss:project:init I want to build a Python FastAPI-based REST API
> for user authentication and authorization. It will use JWT-based authentication
> and PostgreSQL as the database. It is part of a microservice architecture
> and other services will call it.

**Extracted information:**
- Type: REST API (microservice)
- Purpose: user authentication and authorization
- Stack: Python, FastAPI, JWT, PostgreSQL
- Users: other services

### Example 2: Web application
> /hstl-oss:project:init I want to build a real-time stock trading dashboard
> with React and TypeScript. Individual investors should be able to manage
> their portfolios and monitor market data. Real-time data will arrive via
> WebSocket.

**Extracted information:**
- Type: web app (dashboard)
- Purpose: stock trading monitoring and portfolio management
- Stack: React, TypeScript, WebSocket
- Users: individual investors
- Key features: portfolio management, market data monitoring, real-time updates

### Example 3: CLI tool
> /hstl-oss:project:init I want to build a project scaffolding CLI tool for
> developers in Go. It should support templates for various languages and
> frameworks and be customizable through a config file.

**Extracted information:**
- Type: CLI tool
- Purpose: project scaffolding
- Stack: Go
- Users: developers
- Key features: project templates, customization

### Example 4: Minimal start
> /hstl-oss:project:init build a todo app

**Follow-up questions:**
- Web app? Mobile? CLI?
- Tech stack?
- Main features?

## Starting a session

### Example: with context
> /hstl-oss:session:start I stopped yesterday in the middle of the user registration API

### Example: fresh start
> /hstl-oss:session:start

## Starting a task

### Example: specific task
> /hstl-oss:task:start Implement user registration API including email validation

### Example: continue an existing task
> /hstl-oss:task:start TASK-003
```

---

## topic: workflow

```markdown
# Workflow guide

## Project lifecycle

### Step 1: Project definition
```
/hstl-oss:project:init <detailed description>
```
- Auto-generates PDD, Roadmap, and Architecture drafts
- Asks for additional information as needed

### Step 2: Design refinement
Review and refine generated documents:
- `docs/00-project/pdd.md` — add success metrics
- `docs/00-project/roadmap.md` — flesh out milestones
- `docs/03-design/architecture/overview.md` — detail the components

### Step 3: Sprint planning
```
Phase 1 -> Sprint 1, 2, 3 ...
```
- Create Sprint + Task in `works/sprints/active/` or `works/sprints/backlog/`
- Set priorities

### Step 4: Development cycle
```
/hstl-oss:session:start
    v
/hstl-oss:task:start
    v
[Development work]
    v
/hstl-oss:task:complete
    v
/hstl-oss:session:end
```

## Daily workflow

### Morning: start session
```
/hstl-oss:session:start
or
/hstl-oss:session:catchup  (when there is in-progress work)
```

### During work
```
/hstl-oss:task:start [task-id]
... work ...
/hstl-oss:task:complete
/hstl-oss:task:next
```

### Evening: end session
```
/hstl-oss:session:end
```

## Documentation workflow

### ADR
1. When a technical decision is needed
2. Create `docs/02-architecture/adrs/ADR-NNN-title.md`
3. Template: `templates/adr/ADR-TEMPLATE.md`

### Feature design
1. Before developing a new feature
2. Create `docs/03-design/features/feature-name.md`
3. Template: `templates/design/feature-design.md`

### Sprint documents
1. Sprint start: `sprint-planning.md`
2. During Sprint: update `sprint-progress.md`
3. Sprint end: `sprint-review.md`, `sprint-retrospective.md`
```

---

## topic: templates

```markdown
# Template list

## Project (3)
| Template | Purpose |
|--------|------|
| `templates/project/pdd.md` | Project Definition Document |
| `templates/project/roadmap.md` | Roadmap |
| `templates/project/phase-spec.md` | Phase specification |

## ADR (1)
| Template | Purpose |
|--------|------|
| `templates/adr/ADR-TEMPLATE.md` | Architecture Decision Record |

## Design (7)
| Template | Purpose |
|--------|------|
| `architecture-overview.md` | Architecture overview |
| `domain-design.md` | Domain design |
| `feature-design.md` | Feature design |
| `api-design.md` | API design |
| `deployment-design.md` | Deployment design |
| `configuration-design.md` | Configuration management |
| `iac-design.md` | IaC design |

## UX (6)
| Template | Purpose |
|--------|------|
| `user-persona.md` | User persona |
| `user-scenario.md` | User scenario |
| `task-flow.md` | Task flow |
| `interaction-pattern.md` | Interaction pattern |
| `wireframe.md` | Wireframe |
| `usability-checklist.md` | Usability checklist |

## Sprint (6)
| Template | Purpose |
|--------|------|
| `sprint-planning.md` | Sprint planning |
| `sprint-backlog.md` | Sprint backlog |
| `sprint-progress.md` | Progress |
| `sprint-review.md` | Sprint review |
| `sprint-retrospective.md` | Retrospective |
| `sprint-completion.md` | Completion report |

## Operations (4)
| Template | Purpose |
|--------|------|
| `runbook.md` | Operations runbook |
| `playbook.md` | Response playbook |
| `incident-report.md` | Incident report |
| `sla-slo.md` | SLA/SLO definition |

## Review (2)
| Template | Purpose |
|--------|------|
| `code-review-report.md` | Code review report |
| `design-review.md` | Design review |

## Knowledge (2)
| Template | Purpose |
|--------|------|
| `lessons-learned.md` | Lessons learned |
| `analysis-report.md` | Analysis report |
```
