---
description: User-facing project briefing — real-time status report (verbose by default). Use when the user asks "what's the project status?", "give me a briefing", "where are we now?", or wants a rich snapshot of git/Sprint/Task state.
argument-hint: ""
allowed-tools: Bash(hstl-oss:*)
---

# Project Brief

Provides a rich briefing of the project's current state to the user. Returns
git status (+ recent commits + branch topology + branch_gaps), Sprint progress,
Task status (+ backlog todos + recent P2 Tasks), and issues collected in real time.

> The default is verbose so a plain `hstl-oss brief` call yields a rich output
> in line with brief's intent ("user-friendly rich briefing"). For automation/CI
> scenarios that need the compact form, pass `--verbose=false`.

## CLI usage

```
Bash("hstl-oss brief")                    # default verbose
Bash("hstl-oss brief --verbose=false")    # compact (backward compat — previous default)
```

Use `jq` for JSON parsing (no python3):
```bash
hstl-oss brief | jq '.active_sprints'
hstl-oss brief | jq '.git.recent_commits'        # verbose-only field
hstl-oss brief | jq '.git.branch_gaps'           # verbose-only field
hstl-oss brief | jq '.backlog_todo_tasks'        # verbose-only field
hstl-oss brief | jq '.tracks'                    # Track Management integration
hstl-oss brief | jq '.tracks.current'            # current worktree's Track + next_action
hstl-oss brief | jq '.tracks.active'             # active Track list (status only)
```

## Output format

Parse the `hstl-oss brief` response and render it as follows.

```markdown
=============================================
  {project_name} — Project Brief ({date})
=============================================

  Version: {version}

Git
  Branch:      {branch}
  Uncommitted: {N}
  Ahead:       {N}
  Behind:      {N}

  Recent commits (verbose, 5):                  <- default
    {sha}  {message}  {date}

  Branch topology (verbose):                    <- default
    Local:  {local branches}
    Remote: {remote branches}
    HEAD:   {head}

  Branch gaps (verbose):                        <- default
    main..dev:   {N}
    dev..HEAD:   {N}
    HEAD..main:  {N}

Track (Track Management integration)
  Current worktree:  {track_id} [{status}] — {title}     <- .track file / HOSTLER_TRACK_ID env / state json
    next:            {next_action}                       <- e.g. /hstl-oss:task:start ... or /hstl-oss:sprint:start ...
  Active Tracks ({count}):
    · {track_id} — {title}
    · ...
  (When unassigned: single line "Track (current worktree): unassigned — hstl-oss track create / HOSTLER_TRACK_ID env")

Sprint
  [{status}] {sprint_id}  {title}
    {done}/{total} done ({percent}%)
    Goal: {goal}
    Tasks:
    [x] {task_id}  {title}
    [>] {task_id}  {title}  (in-progress)
    [ ] {task_id}  {title}

Recently completed Sprints (3)
  {sprint_id}  {completed_at}  {done}/{total} done
    Title: {title}
    Goal:  {goal}

Recently completed Tasks (P0~P2, up to 10)             <- included by default
  {task_id}  {completed_at}  {priority}  {type} [{sprint}]
    {title}

Backlog Todo Tasks (verbose, top 10)                   <- default
  {task_id}  {priority}  {estimate}  {type}
    {title}

Task summary
  Total: {total}  Done: {done}  In-Progress: {in_progress}  Todo: {todo}

Current Task
  {task_id}: {title}  or  none

Next Task
  {task_id}: {title}  or  none

Issues
  [WARN] {message}
  none

=============================================
```

### Section order (fixed)

1. Header (project name + date)
2. Git status (+ verbose: recent commits, branch topology, branch_gaps)
3. **Track** — current worktree assignment + active Track list
4. Active Sprint (progress + Task list + goal/summary)
5. Recently completed Sprints (3, with goal/summary)
6. Recently completed P2+ Tasks (10)
7. Backlog Todo Tasks (verbose, 10)
8. Task summary
9. Current Task + Next Task
10. Issues

### Rendering rules

- If no Sprint: print "Sprint: none (no active Sprint)"
- Task list markers: `[x]` done, `[>]` in-progress, `[ ]` todo
- If no Issues: print "none"
- No ASCII graphics (e.g. progress bars) — text only
- If verbose extra fields (recent_commits / branches / branch_gaps / backlog_todo_tasks) are
  null or empty, omit the section entirely (e.g. fresh repo, brief --verbose=false)
- **Track section** — if `tracks` is null or active=0 + current=null,
  omit the section. If only current is null, print a single "unassigned" line. If active is 0, omit the active row.

## References

- AI-facing context: `/hstl-oss:project:context`
- Skill guide: `skills/project-management/SKILL.md`
