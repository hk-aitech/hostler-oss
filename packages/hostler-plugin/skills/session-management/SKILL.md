---
name: session-management
description: Diagnose and tune Claude Code session mechanics — SessionStart hook architecture, the CURRENT-FOCUS.md SSOT, context engineering (JIT loading, compaction, pruning), and auto-memory integration. Use this skill whenever the user mentions "session start", "Hook debug", "CURRENT-FOCUS", "context loss", "JIT loading", "compaction", session restart issues, or anything about how Claude Code loads project context — even when they don't explicitly ask. Do NOT use for project / sprint status (use `project-management`).
compatibility:
  tools: [Read, Glob]
paths: ["works/**/*", ".hstl-oss/**/*", ".claude/**/*"]
user-invocable: false
---

# Session Management Skill

> Claude Code session context-engineering guide. Focuses on the SessionStart hook
> architecture, CURRENT-FOCUS.md management, and context strategy.
> For project / sprint / task status queries, use `/hstl-oss:project-management`.

---

## SessionStart hook architecture

`hooks/session-context.sh` runs automatically at the start of every session. The hook is bound to Claude Code's `SessionStart` event and prints the project state without user intervention.

### Execution order

The hook runs the following six steps in order:

1. **Verify works/ directory exists** — confirm the hostler work structure has been initialized. Emits `[WARN]` if missing.
2. **Print Git status** — show current branch, uncommitted change count, and remote sync state (ahead/behind). When uncommitted files exist, list the top five. Interpret the `[project] branch:<name> uncommitted:<N> ahead:<N> behind:<N>` line per the branch policy (shared template: `skills/branch-workflow/references/branch-policy.md`) — e.g. a large ahead count signals it's time to push `dev` or merge `dev → main`.
3. **Find current Task** — search Task files under `works/sprints/active/` for items with `status: in-progress` and print the ID and title. If nothing is in progress, surface the `/task:start` hint.
4. **Sprint progress** — print active / backlog / completed Sprint counts and the done/total ratio of each active Sprint.
5. **Workflow issue detection** — detect Sprints whose Tasks are all done but that haven't been moved to `completed/`, and surface the corresponding `git mv` command.
6. **Disk space check** — warn if available space drops below 100MB.

### Safety guards

The hook never blocks session startup. It carries three protections:

- **Recursion guard**: `HSTL_HOOK_RUNNING` environment variable prevents repeat invocations.
- **Timeout**: terminates automatically if it doesn't finish within 8 seconds.
- **Error handling**: an `ERR` trap records the failure point (line number, exit code) and `exit 0` keeps the session running.

---

## Context engineering rationale

### Why inject context automatically at session start

Each Claude Code session starts blank. It doesn't know the prior session's conversation, the in-flight work, or the project state. The SessionStart hook solves this "cold start" by automatically providing the **smallest high-signal token set**. The user shouldn't have to ask "where were we?" every time.

### Criteria for valuable information

Information emitted by the hook is selected by:

| Criterion | Description | Example |
|-----------|-------------|---------|
| Actionable | Information that lets you decide the next action immediately | "5 uncommitted changes" |
| Current focus | The Task / Sprint to focus on right now | "T### in progress" |
| Anomaly detection | A state that drifted from the normal flow | "Sprint complete but not moved" |

Conversely, the hook deliberately **does not** print:
- Detailed history of completed Sprints (token waste)
- The full backlog Task list (information overload)
- Code change diffs (the user can fetch when needed)

### Three core techniques

1. **Context Pruning** — remove completed information and keep only the current state. Replace stale error messages, resolved issues, and completed Task details with summaries.
2. **Compaction** — once context utilization hits 70%, run `/compact`. Update CURRENT-FOCUS.md before compacting and capture key decisions.
3. **Just-in-Time Loading** — don't read every file up front; load only when needed. Use `Read` for a specific file, `Grep` for related code, `Glob` for a file listing.

For details, see `references/context-engineering.md`.

---

## CURRENT-FOCUS.md management

`works/CURRENT-FOCUS.md` is the **single source of truth** for the current Sprint and Task state.

### Role

- Summarizes the active Sprint's goal and progress.
- Records the in-progress Task's checklist and decisions.
- Holds session notes that let the next session quickly recover context.

### When to update

| Trigger | Update |
|---------|--------|
| Task started | Add current Task ID, title, and Done Criteria |
| Task completed | Mark complete and roll forward to the next Task |
| Sprint switched | Update Sprint goal and Task list |
| Before `/compact` | Refresh progress and decisions |
| Before session end | Leave notes for the next session |

### Format recommendations

- Keep only the current Sprint / Task state. Refer to `works/sprints/completed/` for completed Sprint detail.
- Carry only the essentials so the file stays small. The hook reads this file at every session start; an oversized file kills token efficiency.

### Automatic management

CURRENT-FOCUS.md is no longer updated manually. `hstl-oss sprint start`
/ `hstl-oss sprint complete` invoke `fileutil.UpdateCurrentFocus`
to refresh it automatically. New projects get an idle initial state created
automatically by `hstl-oss project init-skeleton`. Rich custom content authored manually
by the user / AI may be overwritten on the next sprint transition, so durable
records belong in `works/sprints/<sid>/SPRINT.md`.

### Session validation

When a Task is in progress at session start, run `hstl-oss harness auto-check task <ID>`
to bulk-validate the Go-deterministic items (build/test/lint) and quickly
clear stale results. Recommended as a single integrated command in the SessionStart hook.

---

## Relationship to auto memory

Claude Code ships a built-in **auto memory** feature that manages cross-session memory. Distinguish its role from the session-context hook clearly.

| Aspect | auto memory | session-context hook |
|--------|-------------|----------------------|
| Owner | Claude Code built-in | hostler plugin |
| Stored content | User preferences, coding style, recurring patterns | Project state, Sprint/Task progress |
| Update mechanism | Claude decides automatically | Hook script runs every session |
| Scope | Cross-project (global memory) | Current project only |

**Principle**: auto memory remembers "how you work"; the session-context hook tells you "what you're doing right now". They are complementary, not overlapping.

---

## Cross-session state strategy

hostler maintains cross-session state in three layers:

| Layer | File / mechanism | Role | Lifetime |
|-------|------------------|------|----------|
| 1. Immediate restore | `works/CURRENT-FOCUS.md` | Current work context, checklist, decisions | Per Task / Sprint |
| 2. History tracking | `git log`, Task frontmatter | Permanent record of work history | Permanent |
| 3. Lessons accrual | Knowledge Base (`/learned`) | Recurring-reference lesson cards | Permanent |

A separate worklog file (`works/worklogs/`) is optional. `git log` already captures enough work history; reach for worklogs only when you need to record decision context or experiment results that don't fit naturally in git.

---

## Subagent context-ack auto-injection

When you spawn a subagent via Claude Code's `Agent` tool, the hostler PreToolUse hook
(`hooks/subagent-context-ack.sh`) fires automatically and records a context ack
on the work_ticket of the active Task.

### Trigger conditions

- PreToolUse matcher = `Agent` (just before the Claude Code Agent tool is called)
- Calls `hstl-oss context --ticket <ticket>` only when there is a Task in `hstl-oss task list --status in-progress` and `work_ticket` is populated
- No active Task / no ticket / jq missing / hostler missing → **silent skip**
  (fail-soft — never blocks the subagent invocation itself)

### Effect

- Even if the AI forgets to call `hstl-oss context --ticket` manually, an ack
  is recorded at subagent invocation time, so the `context_acknowledged`
  item in the task complete Gate passes
- Combined with TTL expiry: the ack is refreshed on each subagent call,
  reducing stale-ack issues during long Tasks

### Limitations

- Targets only the single active ticket (when the ticket is rotated by checkpoint,
  the next subagent call records an ack on the new ticket)
- On hook failure, only a stderr warning is emitted; the subagent proceeds normally

### Debug

When the environment variable `HSTL_SUBAGENT_ACK_RUNNING` is set, the script
exits immediately for recursion protection. Run `bash hooks/subagent-context-ack.sh`
manually to verify that one ack is recorded for the active Task.

Related: change history `references/changelog.md` (maintainers only), feedback `defensive-design-principle`.

## Session resumption pattern

When you resume a session after a long break, the SessionStart hook output alone
sometimes lacks "which Task and how far". This section standardizes context
recovery on resumption.

### Resumption detection

If any of the following holds, treat it as a resumed session and **call `hstl-oss brief` or
`/hstl-oss:project-management` before the first response**:

1. **Time elapsed**: more than 4 hours since the previous session ended (a typical day-scale break)
2. **Sprint state mismatch**: SessionStart hook reports `[sprint]` as `none` but an active Sprint folder
   exists in `works/sprints/active/`
3. **Task in progress**: `[task]` is `none` but a Task with `status=in-progress` exists in the DB (also when
   uncommitted Task files appear in git status)
4. **User explicitly says "resume", "back at it", "where were we"**

### Invocation order

Once resumption is detected, recover context in this order:

```
1. hstl-oss brief              # Project / Sprint / Task 3-layer state + next-Task recommendation
2. hstl-oss context             # (when needed) work-ticket-based AI compressed context
3. Check recent commits: git log --oneline -5
4. Read the in-progress Task body (re-confirm requirements / Done Criteria before resuming work)
```

### What it returns

`hstl-oss brief` provides all of the following at once:
- [project] branch / uncommitted / ahead / behind
- [sprint] active Sprint ID + progress + remaining Task count
- [task] currently in-progress Task + next recommendation
- [rules] session rules summary
- [issues] Tasks with placeholders, drift warnings

### Principles

- **First response after resume = briefing only** — don't immediately run the user's request;
  share state first and reconfirm intent
- **Refresh the context hash** — the prior session's work-ticket context-ack is invalid.
  Re-issue with `hstl-oss context --ticket WT-...` in the new session
- **Ask first if a Task is in progress** — confirm intent with "shall we continue T###?"

### Scope

Currently this is a guide for the AI to follow manually. Auto resumption detection
(extending the SessionStart hook) and threshold environment variables
(`HSTL_SESSION_RESUME_THRESHOLD_MIN`) are deferred to a future Sprint.

---

## When you need a deeper inspection

When you need more detail than the SessionStart hook provides:

| Goal | Use |
|------|-----|
| Project briefing (`hstl-oss brief`) | `/hstl-oss:project-management` |
| AI compressed context (`hstl-oss context`) | `/hstl-oss:project-management` |
| Sprint / Task progress | `/hstl-oss:project-management` |
| Restore prior conversation history | Claude Code built-in `context resume` |
| Compact context | Claude Code built-in `/compact` |

---

## Additional resources

### Reference Files

- **`references/context-engineering.md`** — Detailed context-engineering strategy (Pruning, Compaction, JIT Loading, long-session guide)

### Troubleshooting: common session issues

#### When the hook doesn't run

| Symptom | Cause | Fix |
|---------|-------|-----|
| No state output at session start | Hook file missing or not executable | `ls -la hooks/session-context.sh` and then `chmod +x` |
| `[WARN] works/ not found` | hostler not initialized | Run `/hstl-oss:project:init` or `mkdir -p works/sprints/{active,backlog,completed}` |
| Hook output truncated or incomplete | 8-second timeout exceeded | If git status is slow, add large directories to `.gitignore`, or check the slow step in `hooks/session-context.sh` |
| Hook runs twice | `HSTL_HOOK_RUNNING` not set | Confirm the hook script's first line includes recursion-guard code (`[[ -n "$HSTL_HOOK_RUNNING" ]] && exit 0`) |

#### When context is lost

| Symptom | Cause | Fix |
|---------|-------|-----|
| New session doesn't know prior work | CURRENT-FOCUS.md not updated | Always update CURRENT-FOCUS.md on Task complete / transition. Verify update before `/compact` too |
| Context lost after `/compact` | CURRENT-FOCUS.md not refreshed before compact | Before `/compact`, write current progress, open decisions, and next steps into CURRENT-FOCUS.md |
| Stale Sprint info displayed | Completed Sprint not moved | Run `git mv works/sprints/active/sprint-NN works/sprints/completed/sprint-NN` |

#### CURRENT-FOCUS.md size management

When the file exceeds 100 lines, token efficiency degrades. Trim it by:
- Removing completed Task detail and summarizing as "Sprint NN: 5/7 Tasks done"
- Moving decision history to ADRs or worklogs
- Not including code snippets (record paths only)

### Related skills

- **`project-management`** — Project / Sprint / Task status queries; use of `hstl-oss brief` and `hstl-oss context`.
- **`sprint-management`** — Sprint create / start / complete and Sprint lifecycle.
- **`task-management`** — Task create, state transitions, completion handling.
