---
name: retro
description: |
  Run a sprint retrospective: extract KPT (Keep / Problem / Try) lessons from
  a completed sprint and hand them off to the `learned` skill for KB card
  creation. Analyzes Task files, git log, and worklogs for patterns. Use this
  skill whenever the user mentions a sprint retrospective, KPT analysis,
  lesson extraction, post-mortem, sprint review, or asks to "look back on a
  sprint" — even when they don't explicitly say "retro". Mandatory in
  `sprint:complete` Phase 5, and also appropriate any time after a sprint
  finishes. Do NOT use for capturing single-task lessons (use the `learned`
  skill instead).
allowed-tools: [Read, Write, Edit, Bash, Glob, Grep]
argument-hint: "[sprint-id]"
paths: ["works/sprints/**/*", "**/*.md"]
---

# Retro -- Sprint Retrospective Knowledge Extraction

Extract lessons from a completed sprint and record them as cards in the project's knowledge-base directory.

## Table of Contents

1. Context
2. Knowledge Base Path Discovery
3. Sprint Directory Discovery
4. Retrospective Process Overview
5. Workflow
6. Lesson Extraction Method -- KPT Framework
7. Sprint Scope Guidelines
8. Differences from the `learned` skill
9. Integration Points

---

## Context

Read the retrospective checklist. The file is located in the same skill directory under `references/retro-checklist.md`. Read it for detailed extraction patterns. Use the project's knowledge-base directory `INDEX.md` as the index, and the card-format guide as the reference for card structure.

## Knowledge Base Path Discovery

When the skill runs, first locate the project's knowledge-base directory. Search in the following order:

1. The user's project docs (recommended -- doc-centric projects)
2. `works/knowledge-base/` (projects using the hostler `works/` layout)
3. `knowledge-base/` (project root)

Discovery criterion: presence of an `INDEX.md` file. If none is found, ask the user to confirm. The category structure (api, architecture, domain, operations, mistakes) is identical to the `learned` skill.

## Sprint Directory Discovery

Search for sprint files in the following order:

1. `works/sprints/active/` (sprint currently in progress)
2. `works/sprints/completed/` (completed sprints)
3. `works/sprints/backlog/` (future sprints)
4. The path defined in the project's user-facing project guide

Discovery criterion: the presence of a `SPRINT.md` file.

## Retrospective Process Overview

The overall flow of the `retro` skill is a five-stage ceremony. The `Workflow` section below breaks these five stages into seven sub-steps (detect / collect / extract / dedup / confirm with user / write cards / sprint link):

1. **Confirm sprint completion** (Workflow Step 1): identify the target sprint and verify task statuses are DONE
2. **Gather changes** (Workflow Step 2): collect task files, git log, worklogs, and existing docs from the sprint period
3. **Extract lessons** (Workflow Steps 3-5): mine candidates with the KPT framework (Keep / Problem / Try) plus pattern matching, dedup, then confirm with the user
4. **Write cards** (Workflow Step 6): delegate to the `learned` skill -- it owns the standard card format and file recording
5. **Update INDEX** (Workflow Steps 6-7): reflect new cards in KB `INDEX.md` and add cross-references to the sprint document

## Workflow

### Step 1 -- Sprint Detection

Determine the target sprint:

- If invoked as `/retro sprint-NN` -> use the specified sprint
- If invoked as `/retro` with no argument -> identify the most recently completed sprint from the project's task/sprint tracking files
- If ambiguous, ask the user which sprint to review

Locate the sprint directory or task files for the target sprint.

Verify the sprint has completed tasks (status: DONE). If all tasks are still IN_PROGRESS or TODO, warn the user that retrospective is typically done after sprint completion.

### Step 2 -- Data Gathering

Collect data from multiple sources for the target sprint:

1. **Sprint file**: goals, scope, completion criteria
2. **Task files**: difficulty notes on DONE tasks, BLOCKED transition history, ACs that required several attempts
3. **Git log**: from `git log --oneline --since={sprint-start} --until={sprint-end}`, look for fix/revert commits, large diffs, and mentions of "workaround"
4. **Worklogs**: issue sections, plan changes
5. **Existing lessons**: any pre-existing lessons-learned documents related to the sprint

### Step 3 -- Candidate Extraction

From the gathered data, extract lesson candidates using these patterns:

| Pattern | Source | Signal |
|---------|--------|--------|
| Bug fix cycle | fix/revert commits | Error -> diagnosis -> fix sequence |
| API gotcha | Task notes, worklogs | Undocumented behavior discovery |
| Design decision | Sprint docs, large refactors | "why X instead of Y" reasoning |
| Performance finding | Commit messages, task notes | Benchmark numbers, bottleneck fixes |
| Domain insight | Feature tuning, config changes | Business rule or domain edge case |
| Time sink | Multiple commits for same issue | Repeated fixes on the same component |
| Process improvement | Retrospective sections | Workflow or tooling lessons |

For each candidate, draft a knowledge card in the standard format:

```
### {Title} ({date}, {task-id})
**Problem**: ...
**Cause**: ...
**Resolution**: ...
**How discovered**: Sprint {XX} {task description}
**Related code**: ...
```

### Step 4 -- Deduplication Check

Read existing knowledge base files to check for duplicates:

1. Read the knowledge-base `INDEX.md` for an overview
2. For each candidate, read the target category file
3. Compare titles and problem descriptions
4. Mark duplicates as "SKIP (existing card present: {existing title})"
5. If an existing card needs updating with new information, mark as "UPDATE" instead of creating a new card

### Step 5 -- Present and Confirm

Present candidates grouped by category with dedup status:

```
## Sprint {XX} Retrospective -- Lesson Candidates ({N} found, {M} duplicates excluded)

### api/ (3)
1. Y **Token refresh timing issue** -- must refresh just before expiry
2. Y **Rate limit measured vs documented mismatch** -- real limit differs from docs
3. SKIP -- existing card: "Token Content-Type"

### operations/ (2)
4. Y **WS connection ordering guarantee** -- must wait for create_task to finish
5. UPDATE -- append new case to existing "DSN configuration" card

Tell me which items to confirm / edit / drop.
```

Wait for user confirmation before writing.

### Step 6 -- Write Cards (delegate to the `learned` skill)

Hand the confirmed cards over to **Steps 5-6 of the `learned` skill** for recording.

1. Use the `learned` skill's KB path discovery logic to identify the target directory
2. Use the `learned` skill's card-writing logic to add or update files
3. Use the `learned` skill's INDEX update logic to update `INDEX.md`

> Do not Write/Edit cards directly inside `retro`. The card format and INDEX management owned by `learned` are the SSOT.

### Step 7 -- Sprint Link

Add cross-references to the sprint document. If a retrospective section already exists, append there; otherwise create a new section at the end of the file and list the knowledge-base files and cards produced by this retrospective using relative links.

## Lesson Extraction Method -- KPT Framework

When running a sprint retrospective, apply the KPT (Keep / Problem / Try) framework to classify lessons systematically.

### Keep -- What went well

Identify methods, tools, and processes that worked well during the sprint (e.g., upfront design review reduced cycle time, automated tests caught regressions early). Record Keep items under the `architecture/` or `domain/` category, prefixing the title with "[Keep]".

### Problem -- What went wrong

Identify areas where behavior diverged from expectations or that ate excessive time (e.g., long debugging session caused by a mismatch between an external API's docs and its actual behavior; service outage from incorrect deploy ordering). After root-cause analysis, file Problem items under whichever of `api/`, `operations/`, or `mistakes/` fits best.

### Try -- What to attempt next

Capture improvement ideas derived from Problems, or ideas to reinforce a Keep (e.g., run a minimum round-trip test before integrating with an external API; create a deploy checklist). Record Try items as knowledge-base cards, and confirm with the user whether they should also be registered as tasks in the next sprint.

## KPT Analysis Output Example

The following is a domain-neutral example of a KPT analysis after a hypothetical sprint (for output-format reference). Assign categories among `architecture`, `domain`, `api`, `operations`, and `mistakes` as appropriate.

```markdown
## Sprint <ID> KPT Analysis — <Sprint Theme>

### Keep — What went well
1. **[Keep] Writing the ADR up front shortened design discussion**
   - Locking key design decisions (e.g., test classification strategy / data model
     choice) into ADRs early in the sprint meant later implementation skipped the
     "which approach?" debate. Estimated savings: 15-20 minutes per Task.
   - → record under architecture/

2. **[Keep] Abstracting time-dependent code stabilized tests**
   - Hiding non-deterministic dependencies (time, randomness, external IO) behind
     interfaces and injecting fakes dropped flaky-rate on timer/reconnect tests
     to 0%.
   - → record under domain/

### Problem — What went wrong
3. **N hours of debugging due to differences in external API auth specs**
   - Trial and error caused by the official docs not specifying header
     differences across protocols (REST vs WebSocket / gRPC, etc.)
   - Root cause: missing per-protocol spec table in the official API docs
   - → record under api/

4. **Missing DB migration left production without an index**
   - The migration SQL omitted `CREATE INDEX`; query latency surfaced after
     production deploy
   - Root cause: no index item on the migration review checklist
   - → record under mistakes/

5. **Race condition — duplicate INSERT on simultaneous requests**
   - Found a case where rapid double-clicks in the UI produced two records of
     the same item
   - Root cause: the API endpoint had no idempotency key applied
   - → record under operations/

6. **Background-process accumulation caused OOM**
   - In long sessions, background workers were not terminated, causing OOM
   - → record under operations/

### Try — What to attempt next
7. **Run minimum round-trip test before integrating an external API** (→ candidate Task next sprint)
   - Catch differences between official docs and actual behavior up front by
     running a minimum request/response round-trip before integration coding

8. **Add index/constraint items to the migration review checklist** (→ candidate Task next sprint)
   - Prevent the omission pattern from recurring

9. **Check background-process count every 30 minutes per session** (→ add to ops checklist)
   - Detect zombie processes early via `ps aux | grep <worker> | wc -l`

---
Summary: 9 lesson candidates (6 new, 2 candidate Tasks, 1 ops checklist item, 0 duplicates)
```

> **Note on category diversity**: the example above happens to span all five
> categories (api / operations / mistakes / architecture / domain), but a real
> sprint's KPT distribution depends on its domain. Add `ui/` or `frontend/`
> for a UI-heavy sprint, `security/` for a security-heavy sprint,
> `analytics/` for a data-analysis sprint, and so on -- adapt to your
> project's KB structure.

## Sprint Scope Guidelines

- **Small sprint (3-5 tasks)**: Expect 3-8 lesson candidates
- **Medium sprint (6-10 tasks)**: Expect 8-15 candidates
- **Large sprint**: Expect 15-25 candidates
- If fewer than 3 candidates found, expand search to adjacent worklogs and git history

## Coupling with the `learned` skill (mandatory)

`retro` performs lesson extraction (through Step 3) and **delegates card
writing (Steps 5-6) to the `learned` skill**. This keeps card format,
deduplication checks, and INDEX update logic in a single place (`learned`).

### Integration flow

```
retro Steps 1-4 (Sprint analysis + lesson extraction + dedup + user confirmation)
  ↓ confirmed lesson list
learned Steps 3-6 (card writing + file recording + INDEX update)
```

### Concrete integration

1. After the user confirms lessons in retro Step 4,
2. Pass the confirmed list in the format expected by **Step 5 (Write Cards)** of the `learned` skill
3. Reuse `learned`'s KB path discovery, card writing, and INDEX update logic verbatim
4. retro Step 7 (Sprint Link) is performed by retro itself

> **Core principle**: the Single Source of Truth for card writing and INDEX
> updates is the `learned` skill. Do not write cards directly from retro.

### Role split with the `learned` skill

| Aspect | retro | learned |
|--------|-------|---------|
| When it runs | After sprint completion | Immediately during conversation |
| Input scope | Entire sprint (tasks, git, worklogs) | Current conversation context |
| Extraction style | Bulk collection then KPT classification | Single-card capture |
| Card writing | **Delegated to learned** | Performed directly |
| Trigger | Explicit `/retro` invocation | `/learned` or auto-suggested |

The recommended pattern is to use `/learned` for immediate capture during the sprint, and `/retro` after sprint completion to extract any lessons that slipped through.

## KPT Try → registering follow-up work

Confirmed Try items should be registered as backlog items during the next
sprint planning. Enter them into whichever sprint tracker is in use (Jira /
Linear / GitHub Issues / Markdown-based), and cross-link the `learned` card
and the work item ID in both directions.

> **Hostler-integrated environment**: `hstl-oss backlog rebuild-md`
> regenerates the BACKLOG.md index. Preview impact with a dry run first:
>
> ```bash
> hstl-oss backlog rebuild-md --dry-run | jq '{count, would_backup}'
> hstl-oss backlog rebuild-md
> ```

## Integration Points

- **After sprint completion**: Primary trigger -- run `/retro` when the last task in a sprint moves to DONE
- **Individual lessons during sprint**: Use `/learned` for immediate capture (see `skills/learned/SKILL.md`)
- **Cross-sprint patterns**: If the same type of issue appears across sprints, flag it in `mistakes/never-again.md`
- **Sprint review**: Wire `/retro` results into the Sprint review-management skill so the review report reflects them (see `skills/review-management/SKILL.md`)
- **References**: `references/retro-checklist.md` -- detailed guide to retrospective check items
