---
name: task-management
description: |
  Manages the full Task lifecycle — creation, state transitions, completion verification, commit conventions, and the Harness Gate.
  Use this skill whenever the user mentions a Task / ticket / unit of work / TODO item / backlog item, a T-ID (T###),
  task creation / starting / completion / reopening / deletion / assignment / unassignment
  (task create / start / complete / reopen / delete / assign / unassign),
  batch / bulk task creation, acceptance / completion criteria,
  the Harness Gate / harness check / harness auto-check, BACKLOG.md / CURRENT-FOCUS.md updates,
  result sections / artifacts, T-shirt estimates (XS/S/M/L/XL),
  preflight scope measurement, refactor / interface signature change grep,
  the 1 Task = 1 commit convention, or expressions like
  "write the task body / draft a task / placeholder body / change task status / verify task completion".
  Be a little pushy — invoke this even if the user only loosely hints at task work.
  Do NOT use for Sprint lifecycle (use sprint-management) or Track operations (use track-management).
allowed-tools: [Read, Write, Edit, Bash, Glob, Grep]
paths: ["works/**/*"]
trigger_commands: ["task.*", "harness.*", "backlog.*"]
---

# Task Management

Task lifecycle management runs through the **hostler CLI**. Ceremony-bearing
commands (`create` / `start` / `complete` / `reopen` / `assign` / `unassign`)
must always go through a slash command (`/hstl-oss:task:*`) — never call them
directly via Bash.

CLI command and parameter SSOT: `hstl-oss --manifest --skill task-management`.

## Task State Decision Guide

Decision tree for choosing the right `hstl-oss task` action:

| Situation | Recommended command | Scope of effect |
|------|----------|----------|
| You thought a Task was done but it needs more work | `hstl-oss task reopen <id> --reason "..."` | DB status `done→todo` + folder restoration + Harness Gate reset + audit log |
| Task was created by mistake (todo state) | `hstl-oss task delete <id> --reason "..."` | DB row deletion + file deletion + BACKLOG.md row removal. **done/in-progress cannot be deleted** |
| Change Task metadata (title / type / priority / estimate / depends_on) | `hstl-oss task update <id> --<field> <value>` | DB + frontmatter atomic update + BACKLOG.md sync |
| Move a Backlog Task into a Sprint | `hstl-oss task assign <id> --sprint <sprint-id>` | DB + file move + BACKLOG.md update |
| Return a Sprint Task to the Backlog | `hstl-oss task unassign <id>` | Reverse move + DB sprint column cleared |
| Permanently delete a done Task | 1) `task reopen <id>` → 2) `task delete <id>` | reopen to make it todo, then delete. No single command (intentional, to prevent mistakes) |

**Principles**:
- **reopen / delete require a reason** — reasons are persisted to the audit log
- **Never delete a completed Task directly** — `task delete` is rejected for done Tasks. Reopen is the explicit human signal of intent
- **assign/unassign moves the file** — `git mv` first, falling back to `os.Rename`
- **Tasks with placeholder bodies cannot be assigned** — write the body first and retry. Emergency bypass is via env var (do not use day-to-day)
- **`task create --sprint` flag is deprecated** — Tasks are always created in BACKLOG. Sprint inclusion is a separate step after the body is written ("create → write body → assign" — a deliberate three-step ceremony)
- **update cannot change status** — status transitions are reserved for the `start/complete/reopen` ceremony commands

## Automatic BACKLOG.md / CURRENT-FOCUS.md Updates

`works/tasks/BACKLOG.md` is generated and maintained automatically by the CLI.

**Automatic triggers**:
- `task create / delete / update / assign / unassign` — instant sync
- `sprint start / complete` — bulk update for the Sprint's Tasks
- `backlog sync` — full recompute + automatic drift correction

**`task complete / reopen` does NOT update BACKLOG.md** (intentional design).
Those commands only update DB status, frontmatter, SPRINT.md (when assigned to
a Sprint), and move the file to `completed/` for unassigned Tasks. Bulk
BACKLOG.md updates converge at sprint complete time, or via
`hstl-oss backlog sync` — the separation reflects that task complete is a tight
loop while bulk BACKLOG.md updates are a Sprint-boundary responsibility.

`works/sprints/active/CURRENT-FOCUS.md` is updated only at **sprint start /
complete / delete** events. Task transitions never touch CURRENT-FOCUS.md.

**Strict rule**: Never edit `BACKLOG.md` / `CURRENT-FOCUS.md` directly via
Write/Edit — that creates drift between the DB cache and the frontmatter.
When in doubt, run `hstl-oss backlog sync --dry-run` to inspect the diff,
then `hstl-oss backlog sync`.

## Automatic Folder Move for Sprint-unassigned Tasks

Sprint-unassigned Tasks (`works/tasks/T*.md`) automatically follow folder
location on state transitions:

| Action | Transition | File move |
|------|------|----------|
| `task complete` | todo/in-progress → done | `works/tasks/T*.md` → `works/tasks/completed/T*.md` |
| `task reopen` | done → todo/in-progress | `works/tasks/completed/T*.md` → `works/tasks/T*.md` |

- **Sprint-assigned Tasks** move with the sprint folder, so they are unaffected — never moved individually
- Idempotent: no-op if already at the destination. `git mv` first, falling back to `os.Rename`. `tasks.file_path` in the DB is updated automatically

See the "Task move flow" section of the `project-structure` skill for the canonical definition.

## Frontmatter Canonical Notation

The 4 frontmatter fields on Task files are normalized to canonical form at the
CLI entrypoint. Even if you write non-canonical values via Write/Edit, the
`task create / update` paths will auto-correct them — but the manual edit path
can introduce drift.

| Field | Canonical | Allowed aliases (auto-normalized) |
|------|----------|-----------------------|
| `status` | `todo` / `in-progress` / `done` / `blocked` / `reopened` | `in_progress`, `in progress`, case variations |
| `priority` | `p0` / `p1` / `p2` / `p3` | uppercase variants |
| `type` | `feature` / `bugfix` / `docs` / `refactor` / `infra` / `test` / `chore` / `spike` / `hotfix` | `fix`→`bugfix`, `doc`→`docs`, `feat`→`feature`, case variations |
| `estimate` | `XS` / `S` / `M` / `L` / `XL` | `xs`→`XS`, `small`→`S`, `extra-large`→`XL`, case variations |

**Manual editing rule**: Use canonical notation, then run `hstl-oss backlog sync --dry-run` to confirm no drift.

## Harness Gate

`task complete` validates the Harness checklist for the Task type. Incomplete items result in BLOCKED.

### `harness get` is mandatory before `harness check`

Because Harness items vary by Task type, you must run `harness get` before
`harness check` to confirm the available `item_id` values:

```bash
hstl-oss-oss harness get --entity-type task --entity-id <ID>          # list available item_id values
hstl-oss-oss harness check --entity-type task --entity-id <ID> --item-id <id> --evidence '...'
hstl-oss-oss task complete <ID> --with-ceremony                       # after all are checked
```

### Harness Items by Task Type

| Task Type | Required Items | Optional / Conditional |
|-----------|----------------|-------------------------|
| `feature` | `criteria_checked`, `build_passed`, `tests_passed`, `code_review`, `context_acknowledged` | `lint_passed` |
| `bugfix` | `criteria_checked`, `reproduction`, `root_cause`, `build_passed`, `tests_passed`, `context_acknowledged` | `lint_passed` |
| `refactor` | `criteria_checked`, `code_review`, `context_acknowledged` | `build_passed`, `tests_passed`, `lint_passed` (required when Go is changed) |
| `hotfix` | `criteria_checked`, `reproduction`, `root_cause`, `build_passed`, `tests_passed`, `rollback_plan` | `lint_passed` |
| `infra` | `criteria_checked`, `deploy_verified`, `rollback_plan`, `change_record`, `context_acknowledged` | — |
| `docs` | `criteria_checked`, `doc_review`, `context_acknowledged` | — |
| `test` | `criteria_checked`, `tests_passed`, `context_acknowledged` | `coverage_checked` |
| `chore` | `criteria_checked` | `build_passed`, `tests_passed`, `lint_passed` (conditional) |
| `spike` | `criteria_checked`, `evaluation_scope`, `findings_documented`, `context_acknowledged` | — |

> The table above reflects the hostler harness defaults schema. The actual
> items always come from the `harness_get` response — that is the absolute
> source of truth. `lint_passed` is initialized as required for Go projects
> (when `go.mod` is present).

### Automatic Verification of Deterministic Items

`hstl-oss harness auto-check` batches the deterministic items
(`build_passed` / `tests_passed` / `lint_passed` / `context_acknowledged`)
through automated processing. Items that require human judgement
(`criteria_checked`, `code_review`, `reproduction`, `root_cause`,
`deploy_verified`) are left untouched, preserving the safety of the Harness Gate.

Recommended flow:

```bash
hstl-oss-oss harness auto-check task <ID>                                        # 1. deterministic items
hstl-oss-oss harness check task <ID> criteria_checked --evidence "..."           # 2. human judgement only
hstl-oss-oss harness check task <ID> code_review --evidence "..."
hstl-oss-oss task complete <ID> --with-ceremony                                  # 3. complete
```

### --auto-check-criteria Flag

`hstl-oss task complete --auto-check-criteria` parses every top-level
`- [ ]` / `- [x]` checkbox directly under the Task file's `## Done Criteria`
(Completion Criteria) section:

- All `- [x]` → PASS, Harness `criteria_checked` is checked automatically
- Any `- [ ]` remaining → FAIL, completion is blocked

```bash
hstl-oss-oss task complete <ID> --auto-check-criteria --with-ceremony
hstl-oss-oss task complete <ID> --auto-check-criteria --dry-run    # validate only
```

Other sections (e.g. `## Requirements`, `## References`) are not in scope. Nested checkbox
indentation is also not parsed. See the "Completion Criteria vs Goals" section
below for guidance on writing completion criteria.

## Completion Criteria vs Goals

**Completion criteria (`## Done Criteria`)** = the measurable minimum
(acceptance threshold). Every item must be `[x]` before a Task can transition to done.

**Goals (aspirational)** = numbers or states that are good to hit but should
not block completion if missed. Put them in a separate body section or under
`## References`. **Do not include them under completion criteria**.

For each completion criterion, specify in parentheses how the achievement is
verified — what command or observation:

- `- [ ] go build ./... PASS` — the build command itself is the verification
- `- [ ] grep -c "<pattern>" SKILL.md >= 1` — grep match count
- `- [ ] audit verify result tampered=0` — CLI response value
- `- [ ] At least 3 new tests (TestXxx through TestYyy)` — verifiable by file/test names

**Forbidden**: vague phrasing like "generally improved" / "works well" /
"users are satisfied", or any subjective judgement without a grep/command/state check.

### Defer sprint:complete Responsibilities

The completion criteria of a dev verification Task must NOT include checks for
**events that have not yet happened at Task complete time**. Such items
conflict with the invariant that Task complete precedes sprint:complete,
making the Task impossible to truly complete and leaving hypocritical checkboxes.

```markdown
# Bad example (dev verification Task)
## Done Criteria
- [ ] SPRINT.md Phase 5/6/9 entries written at sprint complete  ← order paradox
- [ ] N KB cards registered                                     ← sprint:complete responsibility
- [ ] Sprint HMAC signature complete                            ← sprint:complete responsibility

# Good example (dev verification Task)
## Done Criteria
- [ ] Build/tests PASS (observable at Task time)
- [ ] Regression checks complete
- [ ] make install complete
```

**Underlying rule**: completion criteria = only events observable within the
Task's own scope. Sprint or external ceremony verification belongs in
`SPRINT.md ## Done Criteria` or the sprint harness.

## Estimate Guidelines

The `estimate` field is a T-shirt size (XS / S / M / L / XL). Use it
together with line count, file count, and complexity:

| estimate | Rough scale (heuristic) |
|----------|----------------------|
| XS | < 5 lines / 1 file |
| S | < 50 lines / 1–3 files |
| M | 50–200 lines or 4–10 files |
| L | 200–500 lines or 11–20 files |
| XL | > 500 lines or 21+ files → **require Phase split review** |

Complexity uplift (+1 size): Contract / concurrency tests, multi-layer changes, learning a new domain.

**Recommend splitting L/XL** if any of these apply: 4-axis bundle (design + impl + tests + docs) /
2+ layer crossing / 1+ week effort. Split into 2–3 M-sized Tasks instead.

The combination of `migrate` / `migration` / `cutover` / `rename` keywords with
`--estimate XS` triggers a stderr WARN (to guard against structural
underestimation bias). For recommended minimum estimates and detailed cases,
see `references/detailed-guide.md`.

## preflight-scope Measurement Required

The Task body must be written based on **measurement, not estimation**. There
is an accumulated track record of estimation-based planning showing 4x+
deviations from actual measurements.

### Required Convention (mandatory)

1. **Task body must include a `## Requirements → ### Measurement` section** (every Task, regardless of size):
   ```markdown
   ### Measurement (YYYY-MM-DD)
   - Target grep command: `grep -rn "pattern" src/`
   - Result: N hits (with actual file paths)
   - Scope of impact: X packages / Y files
   ```
2. **Measure first when scoping a Sprint** — before `task create`, the planner
   should run each Task's regex/grep to capture hit counts, then record them
   in the Task summary as a "measurement-based estimate".
3. **L/XL Tasks must attach the measurement grep**

For brand new Tasks (file does not yet exist), there is nothing to measure, so explicitly note "No measurement target (new)".

## Scope Limits / Nature Sections

New Task creation auto-includes the following two sections in the template:

```markdown
## Nature
- [ ] Brand new structure
- [ ] Update to existing structure
- [ ] Measurement / verification / deployment
- [ ] Mixed (some new + some updated)

## Scope Limits
### Included in this Sprint
- {confirmed items}
### Deferred to a future Sprint
- {deferred items — write "none" if there are no deferrals}
```

The headings `## Nature` and `## Scope Limits` are template-emitted literals that the parser keys on.

- **3 or more deferred items** → consider splitting the Task or registering separate Tasks
- **"None" is the right answer for many Tasks** — common at XS / S sizes

## Result Section Sub-heading Convention

The strict result-section parser used by `task complete` supports sub-heading mode:

| Sub-heading | Scan mode |
|-------------|-----------|
| `### Artifacts` / `### Changed files` / `### Files` | **artifact** — file path scan |
| Any other ### (e.g. `### Design decision`, `### Verification`) | **narrative** — excluded from scan |

- **Backward compat**: writing directly under `## Result` without a sub-heading scans the entire section
- Wrapping design / verification / retrospective prose in explicit sub-headings prevents file-name mentions there from being false positives

**Good example**:

```markdown
## Result
### Design decision
Used the global `<config>` repository.
### Artifacts
- `<repo>/pkg/<pkg>/<file>.go` (new)
### Verification
- `go test ./pkg/<pkg>/...`: PASS
```

**Anti-patterns**:
1. Backward compat mode + backtick paths in the narrative → parser catches them and BLOCKS.
   Fix: always wrap narrative paths in an explicit sub-heading.
2. An ad-hoc heading like `### Changes` listing artifacts → treated as narrative, not scanned.
   Fix: use one of the exact artifact-keyword headings listed above.

## Task Reopen

`hstl-oss task reopen <id> --reason "..."` reverses a done or in-progress Task
back to todo (default) or in-progress.

| Item | Behaviour |
|------|------|
| State transition | `done → todo` (default) / `done → in-progress` (`--status in-progress`) |
| reason | **At least 10 characters required** — shorter is BLOCKED, persisted to the audit log |
| Harness Gate | Auto-reset (`harness_items` rows deleted → re-verification required) |
| Folder restore | Sprint-unassigned Tasks only — `completed/T*.md → T*.md` |
| SPRINT.md status column | Sprint-assigned Tasks only |
| BACKLOG.md sync | **Not updated** — bulk update happens at sprint complete or via `backlog sync` |
| Audit log | `task.reopened` event + reason |

**When to use**: rework needed / completed by mistake / follow-up verification failed / stakeholder rejection.

**Permanent deletion of a done Task**: no single command. The 2-step
`task reopen` → `task delete` is enforced (mistake prevention).

## Principles

- **Go through the CLI** — never edit files under `works/` directly with Write/Edit
- **No bypassing the Harness Gate** — if the response is BLOCKED, fix the missing items and retry
- **1 Task = 1 commit** — commit all changes once at `task:complete` time
- **Commit prefix per type**: feature→`feat:`, bugfix→`fix:`, hotfix→`hotfix:`,
  docs→`docs:`, refactor→`refactor:`, infra→`infra:`, test→`test:`,
  chore→`chore:`, spike→`spike:`
- **Auto DB path recovery** — even if you `git mv` a file, glob fallback locates it and updates the DB
- **Tasks can be deleted only in todo state** — done / in-progress cannot be deleted. Completed Tasks belong in archive
- **HMAC secret rotation ceremony rule** — `hstl-oss task rotate-hmac-secret` requires both
  `--reason "..."` (30+ chars) and `--evidence` (KB card / issue / Task identifier).
  Without these, the CLI rejects with exit code 2

## Pre-claim a TRAC Marker ID

When introducing a new command series or feature series, attaching code marker
IDs guessed as the next sequential number can collide with IDs already taken
in another area. There is a regression precedent where post-hoc bulk `sed`
replacement after a collision wasted a full Task's worth of effort.

**Recommended workflow — register first, then attach the code marker**:

1. **Check what's taken (mandatory pre-step)**:
   ```bash
   hstl-oss trac list <CTX> --type <T> -o json | jq -r '.data.entries[].id'
   ```
2. **Register first** — `hstl-oss trac register` automatically allocates the
   next available Sequence. Use the `id` field from the register response
   verbatim on the code marker.
3. **Attach the code marker** — `// trac: <id from register response>` in the function doc comment.
4. **Verify** — every category of `hstl-oss trac validate` passes.

For the full standard plus per-Context · Type pre-claim command tables, see
the TRAC v1.1.1 standard document, under the "ID issuance procedure" sub-section.

## References

- CLI manifest: `hstl-oss --manifest --skill task-management`
- Detailed rules: `references/detailed-guide.md`
  (backtick path conventions, interface/refactor grep checklist, Edit section
  preservation, task:complete pre-checks, escape hatch rules, E2E temp Task
  patterns, compressed-issue Bulk Migration patterns)
- Checklist collection: `references/checklists.md`
- AI agent invocation guide: `references/agent-guide.md`
- Changelog (maintainer-only): `references/changelog.md`
