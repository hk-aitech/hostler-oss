---
name: getting-started
description: Overview of the entire hostler plugin — initial setup, day-to-day workflow, project:init, Sprint/Task lifecycle, and the catalog of commands/skills/agents. Use this skill whenever the user is new to hostler, asks "how do I get started", "what does this plugin do", "which commands are there", "show me the workflow", or generally needs the plugin overview — even if they don't ask explicitly. Do NOT use for flag configuration (use `flags`) or for session mechanics (use `session-management`).
compatibility:
  tools: [Read, Glob]
paths: ["**/*"]
---

# Getting Started Guide

## Table of Contents

1. About the plugin
2. How to use the plugin
3. Work hierarchy (5-Level Hierarchy)
4. Quick start
5. Prompt examples
6. Workflow overview
7. Key commands
8. Generated project structure
9. Tips
10. Guides
11. Related docs

---

## About the Plugin

**hostler** is a Claude Code plugin that supports a structured project development process.

### Core capabilities

1. **Conversational project initialization**: describe the project in natural language and get the structure and documents generated automatically
2. **Phase / Sprint / Task hierarchy management**: structured work classification
3. **Session-based context management**: continuity across work sessions
4. **32 document templates**: standardized deliverables

---

## How to Use the Plugin

### Commands (slash commands)
The user **invokes them directly**. **The `/hstl-oss:` prefix is required.**

```bash
/hstl-oss:project:init <description>
/hstl-oss:session:start
/hstl-oss:task:start
```

### Skills (8)
Claude **invokes them automatically based on context**. The user does not need to call them directly.

### Agents (5)
Claude **picks one automatically based on the nature of the work**.

---

## Work Hierarchy (5-Level Hierarchy)

hostler manages projects in five hierarchical levels:

```
Project (6-12 months)
└── Release (2-4 months)
    └── Phase (4-8 weeks)
        └── Sprint (2 weeks)
            └── Task (30 min - 8 hours)
```

### Level summary

| Level | Duration | Description |
|-------|---------|-------------|
| **Project** | 6-12 months | Top-level unit for an entire product or major initiative |
| **Release** | 2-4 months | Bundle of shippable features |
| **Phase** | 4-8 weeks | Goal-focused development phase (2-4 Sprints) |
| **Sprint** | 2 weeks | Iterative development cycle (5-15 Tasks, 20-40 hours) |
| **Task** | 30 min - 8 hours | Single clear deliverable, owned by a single agent |

### Task size classification

| Size | Estimated time | Note |
|------|---------------|------|
| **XS** | ≤ 30 min | Simple edit |
| **S** | 30 min - 2 hours | Changes within one file |
| **M** | 2-4 hours | Changes across several files |
| **L** | 4-8 hours | Day-long focused work |
| **XL** | 1-2 days | **Must be split** |

> For detailed size criteria, priorities (P0-P3), state-transition rules, and
> carry-over procedures, see the `work-units` skill (the pre-archive content
> is preserved under `skills/work-units/`).

---

## Quick Start

### Step 1: Initialize the project

Describe your project naturally:

```
/hstl-oss:project:init I'd like to build a user authentication API in Python with FastAPI.
It will use JWT-based auth and PostgreSQL. It's part of a microservice
architecture and other services will call it.
```

**What gets generated automatically:**
- Folder structure (docs, src, tests, etc.)
- PDD (Project Definition Document) draft
- Roadmap draft
- Architecture Overview
- First ADR
- User Persona draft

### Step 2: Review and complete the docs

Review the generated documents and fill in TODO items:
- The user's project docs
- The user's project docs

### Step 3: Start a session

```
/hstl-oss:session:start
```

### Step 4: Start working

```
/hstl-oss:task:start Implement user registration API
```

---

## Prompt Examples

### Good prompt (information-rich)

```
/hstl-oss:project:init I want to build a real-time stock trading dashboard
for individual investors using React and TypeScript. It receives real-time
quotes via WebSocket and needs portfolio management, trade history,
and notification settings. The backend is being built by a separate team
and we have its API spec.
```

**Information extracted:**
- Type: web app (dashboard)
- Purpose: real-time stock trading monitoring
- Tech stack: React, TypeScript, WebSocket
- Users: individual investors
- Core features: portfolio management, trade history, notifications
- Constraint: separate backend (API integration)

### Sparse prompt (will trigger follow-up questions)

```
/hstl-oss:project:init build a todo app
```

**Auto-asked questions:**
- Web app? Mobile? CLI?
- What tech stack?
- What core features?

---

## Workflow Overview

### Daily work loop

```
1. Session start (SessionStart Hook runs automatically — Git/Task/Sprint state)
   ↓
2. /hstl-oss:task:start [task-id or description]
   ↓
3. Do the work
   ↓
4. /hstl-oss:task:complete
   ↓
5. /hstl-oss:task:next (next work) or /hstl-oss:learned (record a lesson)
```

### Project phases

```
Phase (8-16 weeks) ─────────────────────────┐
├── Sprint 1 (1-2 weeks)                    │
│   ├── Task 1 (30 min - 8 hours)           │
│   ├── Task 2                              │
│   └── Task 3                              │
├── Sprint 2                                │
│   └── ...                                 │
└── Sprint 3                                │
    └── ...                                 │
────────────────────────────────────────────┘
```

---

## Key Commands

| Command | Description |
|---------|-------------|
| `/hstl-oss:project:init <description>` | Initialize a project (interactive) |
| `/hstl-oss:task:start [id]` | Start work |
| `/hstl-oss:task:next` | Recommend the next task |
| `/hstl-oss:task:complete` | Complete work |
| `/hstl-oss:learned` | Record a lesson |
| `/hstl-oss:project-management` | Look up project context |
| `/hstl-oss:project:guide [topic]` | Show guides |

> Session start/end is handled automatically by the SessionStart Hook (no separate command needed).

---

## Generated Project Structure

```
project/
├── docs/
│   ├── project/          # PDD, Roadmap, Phase specs
│   │   └── phases/
│   ├── adr/              # Architecture Decision Records
│   ├── design/           # design docs
│   │   ├── architecture/
│   │   ├── domain/
│   │   ├── features/
│   │   ├── ux/
│   │   ├── api/
│   │   └── deployment/
│   ├── guides/           # development / operations guides
│   ├── runbooks/         # ops runbooks
│   ├── playbooks/        # response playbooks
│   └── knowledge/        # lessons, reports
├── src/                  # source code
├── tests/                # tests
├── deploy/               # deployment configuration
├── scripts/              # scripts
├── tools/                # development tools
├── works/                # work management
│   ├── CURRENT-FOCUS.md  # current progress + roadmap
│   ├── worklogs/         # daily worklogs (YYYY-MM-DD.md)
│   └── sprints/
│       ├── active/       # in-progress Sprints
│       ├── backlog/      # future Sprints
│       └── completed/    # completed Sprints
├── .claude/              # Claude config
├── The user's project guide
├── README.md
└── .gitignore
```

> 📖 Details: see the `project-structure` skill.

---

## Tips

### More information is better
Including the following in your project description produces a more
complete first draft:
- Project purpose and goals
- Tech stack
- Primary users
- Core features
- Constraints / requirements

### Documents are "drafts"
Auto-generated documents are starting points. Fill in TODO items and refine.

### The SessionStart Hook runs automatically
At the start of every session, Git/Task/Sprint state is printed automatically. If you need a deeper check, use `/hstl-oss:project-management`.

---

## Guides

Further reading:
- `/hstl-oss:project:guide` - full guide
- `/hstl-oss:project:guide commands` - command details
- `/hstl-oss:project:guide prompts` - prompt examples
- `/hstl-oss:project:guide templates` - template list

---

## Using the hostler CLI

The hostler plugin uses the `hostler` CLI for Task/Sprint/Harness Gate operations.

Install:
```bash
cd ${CLAUDE_PLUGIN_ROOT}/cli && make install
```

(Installs `~/.local/bin/hostler` + `~/.local/bin/hostler`.)

Discover CLI commands and parameters via the manifest:
```bash
hstl-oss-oss --manifest                           # full command list
hstl-oss-oss --manifest --skill task-management   # Task-related commands
hstl-oss-oss --manifest --group sprint            # Sprint-related commands
```

### Issue Inbox — cross-project reporting

When you find a plugin bug or improvement in another project:
1. `hstl-oss mailbox submit --target-plugin hostler --type bug --title "..." --symptom "..."` — store the issue
2. From the hostler project: `hstl-oss mailbox scan` — check unprocessed issues
3. `hstl-oss mailbox review --issue-id ISS-... --action accept` — auto-create a Task

Issues are stored as files at `~/.hostler/inbox/{target_plugin}/ISS-{date}-{seq}.md`.

---

## Branch Topology (summary)

Three default tiers + a hotfix offshoot:

```
sprint-{id}  →  dev  →  main         (+ hotfix-{ISSUE}-{slug})
(work)         (integration)  (Prod)
```

- At Sprint start, branch `sprint-{id}` from `dev`
- At Sprint completion, merge into `dev` and delete the local branch
- `dev → main` merge is manual at external-release time (do not auto-merge every Sprint)
- hotfix branches from `main` and **must** merge back into both `main` and `dev`

> For exact conventions, CI triggers, and separators, see the shared template
> `skills/branch-workflow/references/branch-policy.md`. If a project defines
> its own SSOT, that project's SSOT wins.

## Related Docs

- Project structure: `project-structure` skill
- Work hierarchy detail: see §3 (work-units content folded in)
- Session Hook architecture: `session-management` skill
- Project status / briefing: `project-management` skill
- **Branch policy (shared template)**: `skills/branch-workflow/references/branch-policy.md`

---

## Additional Resources

- First-workflow example: `examples/sample-workflow.md` — full flow from initialization through retrospective
