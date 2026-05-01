---
name: project-management
description: The unified entry point for project briefings, status checks, context lookups, and the Context 12-Facet × 3-Layer Model usage and concepts. Use this skill whenever the user asks for project status, says "give me a status", "briefing", "where are we", "what's the next Task", "track status", "facet model", "project status", "briefing", "what's next", "current task", or types short triggers like "brf", "br", or "brief" — even when they don't explicitly ask the skill to run. Calls `hstl-oss context` (AI ~200 tokens) or `hstl-oss brief` (rich user briefing). Covers Sprint/Task/Track progress + facet model usage and concepts. Concept and registration procedures are delegated to references/. Do NOT use for sprint/task lifecycle ops (use sprint-management / task-management).
compatibility:
  tools: [Bash, Read]
paths: ["works/**/*", "docs/**/*", ".hstl-oss/**/*"]
user-invocable: false
trigger_commands: ["context.*", "brief.*", "brf", "br"]
---

# Project Management — Project Context · Briefing Reference Guide

A reference guide for the CLI commands used to look up project status. Covers
`hstl-oss context`, `hstl-oss brief`, and the Sprint/Task progress flow.

## The two commands' roles

| | `hstl-oss context` | `hstl-oss brief` |
|---|--------------------|------------------|
| **Audience** | AI / sub-agent | User (human) |
| **Purpose** | Compressed context for prompt injection | Rich project briefing |
| **Tokens** | ~200 or less | unlimited |
| **Output** | 5-line key:value text | section-by-section text |
| **Command** | `/hstl-oss:project:context` | `/hstl-oss:project:brief` |

## CLI usage

### hstl-oss context — for AI

```bash
hstl-oss-oss context           # compressed text (existing sprint/task mode)
hstl-oss-oss context -o json   # JSON
```

Output example:
```
[project] hostler-plugin branch:main uncommitted:0 ahead:0
[sprint] sprint-NN "title" 3/6 done (50%)
[task] T### "title" (in-progress) | next: T###
[rules] Korean commit, const required, jq usage, Command First, 1Task=1Commit
[issues] none
```

Sub-agent injection:
```
Agent({
  prompt: "Project context:\n" + contextOutput + "\n\nIn this context, ..."
})
```

### hstl-oss context — facet mode

Project Context Facet Model. A grid of core 12 facets × 3 Layers.
Specifying any of `--facets / --layer / --all` switches to facet mode.

```bash
hstl-oss-oss context --layer 1                   # Layer 1 Stable Core 7
hstl-oss-oss context --layer 2                   # Layer 1+2 (default)
hstl-oss-oss context --all                       # all (debug)
hstl-oss-oss context --facets objective,track    # explicit facets (shorthand: "objective" → "core.objective")
```

**3-Layer model**:

| Layer | Role | Facets (12 + plugin) |
|-------|------|----------------------|
| Stable Core (1) | Auto-loaded per session | objective/domain/environment/persona/policy/roadmap/knowledge_index (7) |
| Dynamic (2) | Refreshed at Task time | milestone/track/sprint/task/market_state (5) |
| On-Demand (3) | JIT loaded | knowledge_card / quota / feedback / collaborator (3+ plugin) |

**namespace**: `core.<name>` (12 hostler-fixed) / `plugin.<group>.<name>` (project-defined)

Sub-agent usage:
```
Agent({
  prompt: "Layer 1 stable core context:\n" + run("hstl-oss context --layer 1") + "\n\n..."
})
```

**Reference**: the facet model ADR + standard doc lives under the user project's ADR / standards directory (by convention). For the exact filename, consult the user project's INDEX.

### hstl-oss brief — for users

```bash
hstl-oss-oss brief                   # rich briefing (default verbose)
hstl-oss-oss brief --verbose=false   # compressed (for automation / CI)
hstl-oss-oss brief -o json           # JSON output
```

> **default verbose**: aligns with brief's original definition ("user-friendly
> rich briefing"). When reporting to a user, `hstl-oss brief` alone (no
> options) gives rich output. For automation / CI scenarios that need
> compressed output, explicitly use `--verbose=false`.

Information included (default verbose):
- Project name / version
- Git status (branch, uncommitted, ahead, behind) + **last 5 commits + branch topology + branch_gaps** (verbose)
- **Track Management**:
  - `tracks.current` — Track owned by the current worktree + next_action (priority: `.track` file / `HOSTLER_TRACK_ID` env / state json)
  - `tracks.active` — list of active Tracks (id + title + status; progress is queried separately via `hstl-oss track status`)
  - `tracks.count` — number of active Tracks
- Active Sprint progress + Task list
- Last 3 completed Sprints
- Task summary (done/total)
- Current Task + Next Task
- Last 10 completed P0~P2 Tasks
- **Top 10 Backlog Todo Tasks** (verbose)
- Issues / Warnings

## When to call which

| Situation | Command |
|-----------|---------|
| Sub-agent spawn | `hstl-oss context` |
| Self-orientation by AI at session start | `hstl-oss context` |
| User says "show status" / "briefing" / **"brf"** / **"br"** / **"brief"** | `hstl-oss brief` (default verbose) |
| AI explains the state to the user | `hstl-oss brief` (default verbose) |
| Compressed brief for automation / CI | `hstl-oss brief --verbose=false` |
| Sprint progress only | `hstl-oss sprint progress` |
| Single Task lookup | `hstl-oss task get` |
| **Track status** | `hstl-oss brief` (active 5 + current) or `hstl-oss track list` / `hstl-oss track status` |

## Short trigger keywords

When the user types any of the 1–3 character abbreviations or short words
below, this skill triggers and auto-runs `/hstl-oss:project:brief`
(= `hstl-oss brief --output text`):

- `brf`, `br`, `brief`, `briefing` → run `hstl-oss brief` (user-friendly rich output)
- `status`, `current state`, `where are we`, `next Task`, `any issues` → same
- `track`, `current track`, `track status` → `hstl-oss brief` (with track section) or `hstl-oss track list`

The Anthropic Claude Code spec doesn't support short aliases for slash
commands, so this Skill handles short-keyword matching itself (via natural
language matching in the description).

## Sprint/Task status lookup flow

### Self-orientation right after session start

```bash
hstl-oss-oss context
```

This single command captures current branch, active Sprint, in-progress
Task, rules summary, and issues at once. Inject the output text into the
prompt when spawning a sub-agent.

### Status report to the user

```bash
hstl-oss-oss brief                   # default verbose
hstl-oss-oss brief --verbose=false   # compressed (for automation / CI)
```

Outputs the active Sprint's Task list (done/total), the last 3 completed
Sprints, current/next Task, **last 5 commits + branch topology + branch_gaps
+ top 10 backlog todo Tasks** (added by verbose) — formatted for human
reading.

### Sprint progress only

```bash
hstl-oss-oss sprint progress
```

### Single Task lookup

```bash
hstl-oss-oss task get <TASK-ID>
```

### Full status flow (summary)

```
session start → hstl-oss context              (compressed AI 200 tokens)
user "show status" → hstl-oss brief            (for user, default verbose)
automation / CI compressed brief → hstl-oss brief --verbose=false
Sprint progress only → hstl-oss sprint progress
Task details → hstl-oss task get <ID>
```

---

## Trigger keywords

Triggers: "show status", "project state", "where are we so far", "briefing",
"Sprint progress", "Task status", "what's the next Task", "any issues?",
"project status", "sprint progress", "current task", "what's next",
"project overview", "context", "subagent context"

---

## References

- Commands: `commands/project/context.md`, `commands/project/brief.md`
- Boundary doc: see the 'project-context vs briefing boundary' doc in the user project's guides directory
- CLI sources: `cli/cmd/cli/cmd/context.go`, `brief.go`
- Session Hook architecture: `session-management` skill
- **Branch policy (shared template)**: `skills/branch-workflow/references/branch-policy.md`
- **Project conventions SSOT** (per-project): when a project defines its own under `docs/`, it overrides the plugin's shared template

## Facet model — concept and rationale references

This skill's usage section (§ hstl-oss context — facet mode) covers command
options and behavior. The **concept, rationale, and registration
procedure** for the facet model are split across 4 references:

- `references/context-engineering.md` — the 12-Facet × 3-Layer model concept + design intent + responsibility delegation table
- `references/12-facets-table.md` — exact name / 5W1H axis / source path of each of the 12 facets
- `references/3-layer-rationale.md` — the rationale for separating by change rate × reference frequency + anti-patterns
- `references/domain-facet-creation.md` — registration procedure for user-project-side custom domain facets
- `references/changelog.md` — change history (maintainers only)

| User question | Where to go |
|---------------|-------------|
| "facet options / commands / usage" | This SKILL.md §hstl-oss-oss context — facet mode |
| "what's a facet / why 12 facets / Layer differences" | `references/context-engineering.md` |
| "how do I register a facet" | `references/domain-facet-creation.md` |
| ADR rationale | The user project's facet-model ADR directly |
