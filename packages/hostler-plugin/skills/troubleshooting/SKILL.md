---
name: troubleshooting
description: |
  Systematic debugging and troubleshooting via the 5-step problem-solving
  workflow (Identify, Classify, Diagnose with 5 Whys, Resolve, Close), based
  on ITIL / SRE conventions. Use this skill whenever the user mentions
  debugging, troubleshooting, incident response, root-cause analysis,
  postmortem authoring, outage recovery, bug tracing, building a reproduction
  fixture, or mapping the hotfix-task flow — even when they don't explicitly
  say "troubleshooting". Do NOT use for routine Sprint Task work
  (use `task-management`).
allowed-tools: [Read, Write, Edit, Bash, Glob, Grep]
paths: ["docs/**/*", "works/**/*"]
---

# Troubleshooting Skill

> Systematic problem-solving workflow + documentation guide. The main body covers
> only the 5-step core; detailed templates / checklists / plugin-adoption recovery
> procedures / hotfix Task mapping live under `references/`.

## Table of contents

1. Workflow branching (Task / Worklog modes)
2. Reproduce-first principle
3. The 5-step process
4. Diagnostic tools
5. Quick Reference
6. Plugin issue reporting
7. Backlog sync error response
8. Reference files
9. Related skills / docs
10. References (external best practices)

## Workflow branching

This skill offers **two workflows**. At entry, judge automatically based on
whether the project's `works/` directory exists (a sign that the hostler plugin
is in use):

### Preferred: hotfix Task workflow (when works/ exists)

For projects using the hostler plugin (`works/` + `.hstl-oss/`), **map all 5
steps to the hotfix Task body / Harness Gate**. Don't create a separate worklog
file.

| 5 steps | hotfix Task mapping |
|---------|---------------------|
| **Identify** | After `hstl-oss task create --type hotfix --title "..."`, write Task `## Purpose` + `## Requirements` |
| **Classify** | Task `priority` field (p0–p3) + execute outside the Sprint |
| **Diagnose** | Task `## Result → ### Design decision` body + `harness check task <ID> root_cause --evidence "..."` |
| **Resolve** | code commit + `harness check reproduction / build_passed / tests_passed` |
| **Close** | Task `## Rollback` body + `harness check rollback_plan / change_record` + `hstl-oss task complete <ID>` |

CLI usage flow:

```bash
# 1. Identify — create the Task
hstl-oss-oss task create --type hotfix --priority p1 --title "..."

# 2~3. Author the body (purpose / requirements / design decision / rollback)
$EDITOR works/tasks/T<ID>-...md

# 4. Resolve — code change + 1 commit
git add -A && git commit -m "hotfix(T<ID>): ..."

# 5. Close — harness items + complete
hstl-oss-oss harness auto-check task T<ID>           # build / test / lint auto
hstl-oss-oss harness check task T<ID> root_cause --evidence "..."
hstl-oss-oss harness check task T<ID> rollback_plan --evidence "..."
hstl-oss-oss harness check task T<ID> change_record --evidence "..."
hstl-oss-oss task complete T<ID> --with-ceremony
```

Why this flow:
- Auto traceability: Task ↔ commit ↔ KB card
- The Harness Gate enforces the 5-step core items (reproduction / cause / rollback)
- BACKLOG.md index syncs automatically
- Sprint-unassigned hotfix Tasks auto-relocate to `works/tasks/completed/` on done

### Fallback: worklog-file workflow (when works/ is absent)

Use only for non-hostler projects without a `works/` directory or for ad-hoc notes.
Follow the 5 steps and the documentation type sections below directly.

## Reproduce-first principle

Before starting a bugfix Task, write a reproduction fixture first. Starting a fix
without a reproduction wastes time when the cause hypothesis is wrong, and the
bug recurs.

### Reproduction-fixture template

```markdown
## Reproduction fixture

**Environment**:
- OS: {Ubuntu 24.04 / macOS 14.x / etc.}
- Versions: hstl-oss v{X.X.X} / Go {1.XX}
- Branch: {branch name}

**Reproduction commands**:
\```bash
{minimum commands needed to reproduce}
\```

**Expected output**:
\```
{expected output under normal operation}
\```

**Actual output (bug)**:
\```
{actual error output or wrong behavior}
\```

**Reproduction success**: yes / no / intermittent
```

### bugfix Task entry procedure

1. Write the reproduction fixture first (use the template above)
2. Confirm reproduction succeeds → form a cause hypothesis
3. Validate the hypothesis → make the fix
4. After the fix, regression-test against the fixture

> If reproduction fails: record "could not reproduce" in the Task body and
> describe only observed symptoms. Do not assert a definitive cause.

## The 5-step process

```
┌─────────────────────────────────────────────────────────────────┐
│  1. Identify   2. Classify   3. Diagnose   4. Resolve   5. Close │
├─────────────────────────────────────────────────────────────────┤
│     ↓              ↓              ↓              ↓          ↓    │
│  Symptom       Severity       Cause          Fix          Docs   │
│  Impact        Priority       Hypothesis     Verify       Retro  │
│  Repro         Escalation     Log analysis   Rollback     Share  │
└─────────────────────────────────────────────────────────────────┘
```

### Step 1 — Identify

**Goal**: capture the problem precisely

| Item | Question | Record |
|------|----------|--------|
| Symptom | What went wrong? | Error messages, expected vs actual |
| Impact | How wide is the impact? | Users, systems, features |
| Reproduce | How do you reproduce it? | Steps, conditions |
| Timing | When did it start? | Onset time, frequency |
| Environment | Under what environment? | OS, version, config |

### Step 2 — Classify

**Goal**: decide severity and priority

| Severity | Description | Example | Response time |
|----------|-------------|---------|---------------|
| Critical | Whole-service outage | Server down, data loss | Immediate |
| High | Major feature broken | Login broken, payment failing | Within 1 hour |
| Medium | Some features broken | UI breakage, slow response | Within 4 hours |
| Low | Minor inconvenience | Typo, mild discomfort | Next work unit |

**Escalation criterion**: if not resolved in 30 minutes, notify a senior owner and share investigation context.

### Step 3 — Diagnose

**Goal**: identify the root cause

**Diagnosis process**:
1. **Gather information** — logs (errors / access), monitoring metrics, recent changes (deploy / config)
2. **Form hypotheses** — list possible causes and rank likelihood
3. **Validate hypotheses** — start with the most likely + reproduction tests + isolation tests (narrow the problem space)

**5 Whys technique**:

```
Problem: API responses are slow
  Why 1: Why slow? → DB queries take a long time
  Why 2: Why slow queries? → Index missing
  Why 3: Why no index? → Missed in the migration
  Why 4: Why missed? → Review didn't catch it
  Why 5: Why didn't review catch it? → No review checklist

Root cause: missing review process for DB migrations
```

### Step 4 — Resolve

**Goal**: solve the problem and verify

1. **Pick a solution** — workaround vs root-cause fix; assess risk
2. **Apply the change** — verify in test env first, prepare rollback plan, log all changes
3. **Verify** — confirm the problem is solved, no side effects, monitoring is clean

**Rollback checklist**:
- [ ] Rollback command prepared
- [ ] Rollback test complete
- [ ] Rollback trigger condition defined
- [ ] Rollback owner assigned

### Step 5 — Close

**Goal**: document and share knowledge

1. **Document** — write a worklog; if recurrence is likely, write a guide
2. **Retro** — what did we learn, how do we prevent it, does the process need to change
3. **Share knowledge** — share with the team (when needed), update the Knowledge Base

## Diagnostic tools

| Tool | Use | Example |
|------|-----|---------|
| Log inspection | Trace errors | `tail -f /var/log/<app>.log` |
| Process check | Status check | `ps aux \| grep <process>` |
| Port check | Network | `lsof -i :<port>` |
| Disk check | Capacity | `df -h` |
| Memory check | Resources | `free -m` |

Project-specific tools belong under `<your-project>/scripts/`. Augment this
skill's `## Diagnostic tools` section in the project README (e.g. include a
custom diagnostic script like `dev-check.sh --status / --diagnose / --fix`).

## Quick Reference

### Document locations (worklog workflow)

| Document type | Location |
|---------------|----------|
| Hub | `docs/05-runbooks/troubleshooting/` |
| Topic guide | `docs/05-runbooks/troubleshooting/<topic>.md` |
| Worklog | `docs/05-runbooks/troubleshooting/worklogs/` |

> Paths above are recommended convention (Diátaxis "How-to" quadrant).
> If your project's docs structure differs, follow your own SSOT path.

### File naming

| Type | Pattern | Example |
|------|---------|---------|
| Guide | `<topic>.md` | `backend-auth-issues.md` |
| Worklog | `YYYY-MM-DD-<topic>.md` | `2026-01-03-auth-fix.md` |

### Severity → priority mapping

| Severity | Priority | Action |
|----------|----------|--------|
| Critical | P0 | Resolve immediately |
| High | P1 | Resolve same day |
| Medium | P2 | This work unit |
| Low | P3 | Backlog |

## Plugin issue reporting

When you find a hostler plugin bug while working on another project, use the Issue Inbox:

```bash
hstl-oss-oss mailbox submit --target-plugin <name> --type bug \
  --title "..." --symptom "..."
```

Issues are stored under `~/.hostler/inbox/<target_plugin>/`. In the target project,
process them via `hstl-oss mailbox scan` → `hstl-oss mailbox review`.

## backlog sync error response

When `hstl-oss backlog sync` raises errors like NOT NULL:
1. Find Task files missing required frontmatter fields (`status`, `type`, etc.)
2. Add the missing fields and re-run
3. Run `hstl-oss backlog sync --dry-run` first to see the issue list

## Reference files

Detailed templates / checklists / plugin-adoption recovery procedures live under `references/`:

| File | Contents |
|------|----------|
| `references/workflow-guide.md` | Detailed workflow guide |
| `references/troubleshooting-templates.md` | Guide / worklog templates (required sections + index.md hub structure) |
| `references/checklist.md` | Per-situation checklists (incident / investigation / resolution) |
| `references/adoption-recovery.md` | Standard recovery procedure when an existing project first adopts the hostler plugin (works/ + .hstl-oss/ skeleton, file → DB restore, BACKLOG.md regeneration) |

Completed examples:

| File | Description |
|------|-------------|
| `examples/sample-guide.md` | Topic-guide example |
| `examples/sample-worklog.md` | Worklog example |

## Related skills / docs

- `task-management` — Sprint Task / hotfix Task lifecycle (cross-link with this skill's hotfix flow)
- `report-issue` — External inbox issue reporting (separate from this skill: "I'll fix it" vs "delegate to others")
- `doc-templates` — General document templates

## References (external best practices)

- [ITIL Incident Management Best Practices](https://www.inoc.com/blog/itil-incident-management)
- [Atlassian Incident Management Template](https://www.atlassian.com/incident-management/template)
- [SRE Incident Management Checklist](https://rootly.com/sre/2025-sre-incident-management-best-practices-checklist)
