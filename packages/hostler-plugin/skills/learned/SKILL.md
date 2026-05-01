---
name: learned
description: Captures lessons, gotchas, and domain insights as KB cards via the `hstl-oss kb create` CLI (auto ID, INDEX recount, audit). Use this skill any time the user mentions "learned", "lesson", "gotcha", "knowledge card", "lesson learned", or just had a debugging revelation, hit an undocumented API behavior, or made a non-obvious design decision worth remembering — even when they don't explicitly ask. Mandatory in sprint:complete Phase 6. Do NOT use for full Sprint retrospectives (use `retro`).
compatibility:
  tools: [Read, Grep, Bash]
argument-hint: "[lesson summary]"
paths: ["docs/07-knowledge/**/*", "docs/knowledge/**/*", "**/kb/**/*.md", "works/sprints/**/*"]
trigger_commands: ["kb.*"]
---

# Learned — Knowledge Capture Skill

Records lessons discovered during a conversation through the `hstl-oss kb create` CLI. The CLI handles file edits and INDEX updates atomically.

## CLI usage

Inspect commands and parameters via the manifest:

```bash
hstl-oss-oss --manifest --group kb
```

## CLI invocation is the rule

This skill performs only three steps: **gather arguments + confirm with the user + invoke the CLI**. It never writes or edits files directly.

| Step | Action | Tool |
|------|--------|------|
| 1. Extract candidates | Identify lesson candidates from the conversation | read-only analysis |
| 2. Decide category / fields | Settle category / title / problem / solution / cause / prevention / refs | classification logic |
| 3. Confirm with user | Present the extraction + apply edits | dialogue |
| 4. Invoke CLI | `hstl-oss kb create` per card | CLI |

Run a duplicate check with `hstl-oss kb list --keyword ...` before invoking.

## Workflow

### Step 1 — Extract candidates

Scan the conversation for these patterns:
- **Error → Fix cycles** — cases that closed with both root-cause analysis and a fix
- **API gotchas** — gaps between official docs and observed behavior
- **Design decisions** — trade-offs, "why X instead of Y"
- **Domain insights** — project-specific rules / tuning
- **"Never again" moments** — repeated mistakes

### Step 2 — Decide category

| Category | Subcategory examples | Prefix (auto) |
|----------|----------------------|---------------|
| `mistakes` | code-review-findings, doc-consistency | M, D |
| `operations` | infrastructure | O |
| `architecture` | decisions, (per-file) | A |
| `api` | broker-auth, backfill-api, (per-file) | API |
| `domain` | dotnet, (per-file) | DOM |

The prefix is decided automatically by `hstl-oss kb create`'s internal mapping — don't specify it manually.

### Step 3 — Build fields

For each card, assemble the following arguments:

| Field | Required | Description |
|-------|----------|-------------|
| `category` | Y | See table above |
| `subcategory` | Y | filename (no extension) |
| `title` | Y | concise, searchable title |
| `problem` | Y | 1–2 sentences |
| `solution` | Y | 1–2 sentences |
| `cause` | recommended | root cause |
| `prevention` | recommended | prevention measures (numbered list OK) |
| `refs` | recommended | list of files / commits / Task IDs |
| `context` | optional | discovery context (Sprint/Task) |
| `occurred_at` | optional | defaults to today |
| `card_id` | optional | auto-issued by default |

### Step 4 — Duplicate check

```bash
hstl-oss-oss kb list --keyword "<key keyword>"
```

If a similar card exists, recommend updating the existing one or cancel the creation.

### Step 5 — Confirm with the user

Present extracted cards grouped by category:

```
## Extracted lessons (N total)

### mistakes/code-review-findings (2)
1. **Token refresh Content-Type mismatch** — REST uses form-urlencoded, WS uses something else
2. **Bearer prefix terminates WS connection** — LS sends only the token

### architecture/decisions (1)
3. **Don't mix UTC and local timezones** — inject TimeProvider

If anything needs to be edited or removed, point it out; otherwise reply "confirm".
```

Proceed to Step 6 once the user confirms.

### Step 6 — Invoke the CLI

For each card, **call `hstl-oss kb create` once**:

```bash
hstl-oss-oss kb create \
    --category mistakes \
    --subcategory code-review-findings \
    --title "Token refresh Content-Type mismatch" \
    --problem "..." \
    --solution "..." \
    --cause "..." \
    --prevention "..." \
    --refs "src/...,T###" \
    --context "Sprint NN T###"
```

Validate the return value:
- Was a `card_id` issued (e.g. "M28")?
- Does `file_path` point to the correct file?
- Did `total_cards` increase?

On failure, surface the error message verbatim and ask whether to retry.

### Quick Capture Mode (`/learned "<message>"`)

When invoked with a quoted argument, treat the argument as the `problem` field and pull `solution`/`cause` from the conversation context. If extraction is hard, ask the user one clarifying question, then invoke the tool.

## Relationship with the retro skill

`/learned` captures **1–3 lessons in real time**, while `/retro` does **Sprint-wide bulk extraction**. Both internally call `hstl-oss kb create`. retro additionally runs `hstl-oss kb list` first for duplicate checks.

## Why manual editing is forbidden

- Prevents ID collisions (concurrent sessions)
- INDEX.md auto-update
- audit_events log entries (`kb.card.created`)
- Standard format compliance

Editing files under the user project's docs directory directly via Write/Edit drops all four. For bulk cleanup or restructuring, use `hstl-oss kb rebuild` to regenerate the index instead.

## Initial setup

If the user project has no docs directory yet, the first `hstl-oss kb create` call creates the directory and files automatically. No separate init command is required.

## Integration Points

- **After debugging (30+ min)**: suggest `/learned`
- **After API integration**: suggest after discovering undocumented behavior
- **sprint:complete Phase 6**: required (run alongside retro)
- **Right after task:complete**: when there's a lesson, recording one card is recommended

## Index synchronization

After creating a KB card, if BACKLOG.md is suspected to be stale (e.g. when
introducing this into an externally adopted project), check the impact range
first with `hstl-oss backlog rebuild-md --dry-run`, then clean up:

```bash
hstl-oss-oss backlog rebuild-md --dry-run | jq '{count, would_backup}'
hstl-oss-oss backlog rebuild-md
```

Details: `skills/troubleshooting/references/hostler-adoption-recovery.md`
