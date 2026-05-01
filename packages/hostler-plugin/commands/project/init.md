---
description: Initialize a project — choice-based 5-stage conversation that scaffolds folder structure, documents, and CLAUDE.md. Use when the user says "start a new project", "initialize this repo", or asks to bootstrap a fresh codebase.
allowed-tools: Bash(hstl-oss:*), Bash(git:config), Bash(git:remote), Read, Write, Edit, Glob, AskUserQuestion
argument-hint: "[project description]"
---

# Project Init — Interactive Initialization

The AI extracts information from the user prompt and gathers the rest through
**at most 5 choice-based conversation stages**, then creates the project structure
and initial documents in one go.

**Design principles** (per `docs/04-guides/development/interactive-ux-principles.md`):

- **Principle 1**: Choice-based, no open-ended questions — every conversation Phase uses choices
- **Principle 2**: At most 4 choices + "Other (free input)" — Phases 2/3/5 follow this
- **Principle 3**: At most 5 conversation stages — Phases 1~5 are conversational, Steps 0/6/7 are silent
- **Principle 4**: Defaults required (= AI recommendation) — every question's option A is the recommendation
- **Principle 5**: Escape paths ("you decide" / "skip") — Phase 4 `*`, Phase 5 D `cancel`

> **Structure note**: Steps 0/6/7 are automatic stages performed by the AI without user input.
> **Conversation Phases** are at most 5 (Phase 1~5) and never violate Principle 3.

## AI default table (when Step 0 extraction fails)

| Item | Default when extraction fails |
|------|-------------------|
| project_name | Folder name (`basename $PWD`) |
| project_type | Phase 2 B) "web app" by default ("API" / "CLI" keywords in the prompt promote to C/D) |
| tech_stack | Filesystem scan result (`package.json` -> JS, `go.mod` -> Go, `pyproject.toml` -> Python, `*.csproj` -> .NET) |
| users | "internal dev team" (PDD draft marks this as TODO) |
| critical_rules | Phase 4 default `*` (top 5 chosen automatically) |

These defaults yield a useful PDD draft even when the user picks "Quick" in Phase 1.

## Step 0: Automatic scan (zero user input)

The AI does the following internally:

- Inspect folder name, git remote, and any existing README/CLAUDE.md/pyproject.toml/package.json/go.mod/*.csproj
- Extract candidate tech stack/type/key features from the free-form `$ARGUMENTS`
- Synthesize a draft of the mode/project type/tech stack candidates (not displayed)

## Phase 1: Mode selection (1 question)

```
Choose project initialization mode:

  A) Quick   — single confirmation based on AI inference, then generate immediately (~1 min)
  B) Confirm — confirm the 3 essentials (~3 min)  <- default
  C) Detail  — confirm 5 items + tune Critical Rules (~5 min)

Choose (A/B/C, Enter=B):
```

- **A** -> jump to Phase 5 draft approval
- **B** -> Phases 2~3 + Phase 5
- **C** -> all Phases 2~5

## Phase 2: Project type (1 question, 4 choices)

The AI places the most likely type from auto-scan as option A.

```
Q. What type of project?

  A) {AI recommendation — based on scan}  <- default
  B) Web app (frontend-centric)
  C) API / backend service
  D) CLI tool / library
  *) Other — type a single line

Choose (A/B/C/D or *, Enter=A):
```

## Phase 3: Tech stack (1 question, 4 choices)

Cross-references the Phase 2 choice and auto-scan (go.mod / package.json /
pyproject.toml etc.) to present the top 3 stacks + "Other".

```
Q. Main tech stack?

  A) {AI recommendation 1 — from filesystem}  <- default
  B) {AI recommendation 2}
  C) {AI recommendation 3}
  D) Mixed / multiple stacks
  *) Other — type a single line (e.g. "Rust + Tauri")

Choose (A/B/C/D or *, Enter=A):
```

## Phase 4: Critical Rules (Detail mode only, 1 question, multi-select)

Quick/Confirm modes auto-select the top 5 and skip Phase 4. Only Detail mode
asks for user confirmation.

```
Q. Critical Rule candidates to embed in CLAUDE.md. Pick the ones that "must
   never be broken" (multi-select, e.g. "1,3,5"):

  1) Korean commit messages
  2) No magic numbers (const required)
  3) No merge without tests
  4) {stack-specific candidate}
  5) {stack-specific candidate}

  *) AI recommendation — auto-select top 5  <- default

Choose (comma-separated numbers or *, Enter=*):
```

## Phase 5: Draft approval (1 question)

Display the full draft synthesized from Phases 1~4 and ask for a single approval.

```
=== Files to be generated and content summary ===
{project name / type / tech stack / 5 Critical Rules / 7 generated documents}
=================================================

Generate as is?

  A) Confirm — generate all files and finish  <- default
  B) Edit    — provide a "section: replacement" line
  C) Restart — go back to Phase 2 and redo
  D) Cancel  — generate nothing

Choose (A/B/C/D, Enter=A):
```

## Step 6: Generation (zero user input)

### 6-1. Auto-generate skeleton

First create base directories and index files via CLI:

```bash
hstl-oss project init-skeleton
```

This command is idempotent — it creates the following without overwriting:

- `works/tasks/` + `works/sprints/{backlog,active,completed}/` directories
- `works/tasks/BACKLOG.md` skeleton
- `works/CURRENT-FOCUS.md` (initial idle state)
- `.hostler/project-config.yaml` (auto-pins the project key)

#### Project key YAML pinning behavior

Internally, `init-skeleton` calls `DetectProjectKey` to determine a stable key
in priority order, then persists it to `project.key` in `.hostler/project-config.yaml`:

1. If YAML already has `project.key`, use it (SSOT)
2. Otherwise: `git config --get remote.origin.url` -> normalize -> SHA1
3. If no git remote: SHA1 of `pwd` absolute path (fallback)
4. The computed key is auto-pinned to YAML so subsequent runs hit step (1)

**Why this matters**: if the key shifts when the binary is updated or the hash
algorithm changes, the DB cache sees the project as a different one — risking
data loss. YAML pinning ensures the key persists once decided, guaranteeing
cross-version stability.

**Verify**: `cat .hostler/project-config.yaml | grep '^  key:'`

### 6-2. Document file generation (AI manual)

Following the standard structure of the `project-structure` skill, generate the
files below. Directory structure and naming conventions are defined in that
skill's `references/init-guide.md` and are not duplicated here.

Generated files:

| # | File | Content |
|---|------|------|
| 1 | `README.md` | Project overview + getting-started guide (draft) |
| 2 | `CLAUDE.md` | Claude Code main context + Command-first policy (see below) |
| 3 | `docs/00-project/pdd.md` | PDD — purpose/type/stack/constraints draft + success-metric TODO |
| 4 | `docs/00-project/roadmap.md` | Default 3-Phase roadmap draft |
| 5 | `docs/03-design/architecture/overview.md` | Tech-stack-based architecture draft |
| 6 | `docs/02-architecture/adrs/ADR-001-{stack}-selection.md` | Tech-selection ADR |
| 7 | `works/CURRENT-FOCUS.md`, `works/worklogs/` | Work management scaffolding |

### Required CLAUDE.md sections

Generated CLAUDE.md must include the following section:

```markdown
## Plugin Command-first policy

This project uses the plugin. Task/Sprint state changes must always go through
the Commands below — the AI must not call the CLI directly.

| Operation | Command | Underlying CLI |
|------|---------|---------|
| Create Task | `/hstl-oss:task:create` | `hstl-oss task create` |
| Start Task | `/hstl-oss:task:start` | `hstl-oss task start --with-ceremony` |
| Complete Task | `/hstl-oss:task:complete` | `hstl-oss task complete --with-ceremony` |
| Create Sprint | `/hstl-oss:sprint:create` | `hstl-oss sprint create` |
| Start Sprint | `/hstl-oss:sprint:start` | `hstl-oss sprint start --with-ceremony` |
| Complete Sprint | `/hstl-oss:sprint:complete` | `hstl-oss sprint complete --with-ceremony` |

**Principle**: when a Command exists, the Command takes precedence. Run the
Command's briefing/reminders/validation ceremony first, then have it call the
underlying CLI. Read-only commands (`hstl-oss task list`, `hstl-oss brief`,
`hstl-oss sprint progress`) may be called directly.
```

## Step 7: Completion report

```markdown
## Project initialization complete: {project-name}

### Generated files
{File list table — Status: created}

### Recommended next steps
1. Review and refine `docs/00-project/pdd.md`
2. `/hstl-oss:project:guide` — confirm plugin usage
3. `/hstl-oss:sprint:create` — create the first Sprint
4. `/hstl-oss:task:start` — start the first Task
```

## Prompt handling examples

### Example 1 (Confirm mode)

> "Python FastAPI-based REST API service. JWT auth, PostgreSQL."

- Phase 2 A) = "API / backend service" (prompt contains "REST API")
- Phase 3 A) = "Python + FastAPI + PostgreSQL" (extracted from prompt)
- Phase 4 skipped
- Phase 5 generate after draft approval

### Example 2 (Quick mode)

> "React + TypeScript dashboard"

- Phase 2 skipped — "web app" auto-selected
- Phase 3 skipped — "React + TypeScript" auto-selected
- Phase 5 draft approval only

## Notes

- Existing files are not overwritten (collisions are reported to the user)
- All generated documents are "draft" status with TODO markers
- When information is unclear, fall back to reasonable defaults and mark TODO in the document
