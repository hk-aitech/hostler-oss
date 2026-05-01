---
name: doc-cross-check
description: Cross-validate design documentation for consistency — broken links, duplicate Feature IDs, event publisher/subscriber integrity, user project guide vs actual structure, BACKLOG vs Sprint. A read-only 10-phase audit. Use this skill whenever the user mentions cross-doc validation, doc consistency, broken links, Feature ID duplication, "BACKLOG vs Sprint mismatch", "guide vs reality", or asks to audit how documents agree with each other — even without explicitly naming the skill. Do NOT use for reviewing a single document (use `doc-review` instead).
argument-hint: "[--session|--sprint|--repo] [--phase=N]"
compatibility:
  tools: [Read, Glob, Grep]
paths: ["docs/**/*", "works/**/*", "**/*.md"]
---

# Design Audit — Cross-Validation of Design Documents

Cross-analyze the project's entire design documentation set to detect inconsistencies, omissions, terminology mixing, and numerical mismatches.
Where `work-audit` checks Task/Sprint workflow compliance, `doc-cross-check` checks the **logical integrity of the design**.

> **audit = read-only inspection.** It does not modify files. Findings are emitted as a report.

## Scope

Specify the range of documents to inspect. With no argument, `--repo` is the default.

| Option | Target files | Interpretation |
|--------|-------------|----------------|
| `--repo` (default) | All design docs in the repository | Every design / requirements / event / deployment doc under `docs/` |
| `--sprint` | Only docs changed in the sprint | Extract files under `docs/` from the `git diff` since sprint start → cross-validate those plus the docs they reference |
| `--session` | Only docs changed in the session | Extract files under `docs/` from `git diff HEAD~N` → cross-validate those plus the docs they reference |

### Scope interpretation rules
- Because `--repo` is the default, calling `/doc-cross-check` alone runs the full design-doc cross-validation
- `--sprint` is forwarded automatically by `sprint:complete` when there are doc changes
- `--session` / `--sprint` include both the changed docs and the documents they reference (validating in isolation cannot establish consistency)
- Specifying a path directly inspects only that path instead of the scope

### Auto-invocation conditions inside sprint:complete
- If anything under `docs/` changed during the sprint, run with `--sprint` automatically
- If nothing under `docs/` changed, skip with the log "doc-cross-check skipped (no doc changes)"

## Inspection Targets

| Document type | Search pattern | What is checked |
|---------------|---------------|-----------------|
| Feature catalog | `**/feature*catalog*.md` | Feature ID uniqueness, totals consistency |
| BC / Aggregate docs | `**/bc-spec*.md`, `**/aggregates/*.md` | Feature mapping, event definitions |
| Functional requirements (FR) | `**/functional*requirements*.md`, `**/FRS*.md` | Coverage of Feature mappings |
| Event catalog | `**/event*catalog*.md` | Publisher/subscriber integrity, topic uniqueness |
| Architecture docs | `**/architecture*.md`, `**/target*.md` | Module/BC list consistency |
| SLO definitions | `**/slo*.md` | Feature/BC mapping |
| Data model | `**/data*model*.md` | Table ↔ Feature mapping |
| Dapr / infra | `**/dapr*.md`, `**/deployment*.md` | Topic ↔ event mapping |
| Overview docs | `**/overview*.md`, `**/pdd*.md` | Numerical integrity (Feature counts, etc.) |

## 10-Phase Inspection

For each phase's detailed checklist and severity criteria, see `references/audit-phases.md`.

### Design-doc cross-validation (Phases 1-7)

| Phase | What is checked | Automation |
|-------|-----------------|-----------|
| 1. Feature ID integrity | ID uniqueness, sequence numbers, declared totals match | `scripts/check-design-consistency.sh` |
| 2. Feature ↔ BC mapping | Feature → BC links resolve, detect double membership | Partially automated |
| 3. Feature ↔ FR mapping | Each Feature has at least one FR; detect ghost references | Manual |
| 4. Event integrity | Publishers/subscribers exist; 1:1 with Dapr topics | Manual |
| 5. Terminology consistency | Synonym mixing, enum mismatches, namespace mixing | Grep-assisted |
| 6. Numerical integrity | Numbers in overview docs == measured aggregates in detail docs | Manual |
| 7. Link health | Relative-path links resolve; detect legacy paths | `scripts/check-design-consistency.sh` |

### Repo-wide cross-validation (Phases 8-10)

#### Phase 8: Full repo link sweep

**Automation**: `scripts/check-dead-links.py` scans every Markdown link
`[text](path)` in SKILL.md / commands / docs and flags references to local
files that do not exist. Runs via `make -C cli check-dead-links` or
`bash scripts/integration-test.sh`.

**Inspection targets**:
- `skills/*/SKILL.md`
- `commands/**/*.md`
- `docs/**/*.md`
- Default exclusions: `archive/**`, the user's project docs (historical record), `works/sprints/completed/**`,
  `skills/*/references/**` (templates for user projects -- many placeholder paths)
- Extend exclusions via `HSTL_DEAD_LINK_SKIP=glob1,glob2,...`

**How to run** (locally / manually):
```bash
python3 scripts/check-dead-links.py               # text output
python3 scripts/check-dead-links.py --format json # JSON pipeline
make -C cli check-dead-links                      # Makefile target
```

**Common findings**:
- References not updated after a file rename (e.g., `bc-spec.md` -> `bc-specification.md`)
- Old paths left over after a directory refactor (e.g., `docs/design/` -> the user's project docs)
- References to files that have been moved into `archive/`

| Finding | Severity |
|---------|----------|
| Broken link between design docs | HIGH |
| Broken link inside `works/` | MEDIUM |
| Reference into `archive/` | LOW |

#### Phase 9: User project guide ↔ project structure

Verify that the figures and structure described in the user's project guide match the actual project.

**Checks**:
1. **Process count**: "6 processes" in the guide == actual number of host projects
2. **Feature count**: Feature counts mentioned in the guide == measured count in the feature catalog
3. **Solution structure**: directory tree in the guide == actual `src/`, `tests/` layout
4. **Tech-stack table**: items in the tech-stack table are actually in use

**Common findings**:
- Guide numbers not refreshed after Features were added (e.g., still says "239 Features" when there are really 250)
- Solution-structure tree not updated after projects were added/removed
- Tech-stack table not updated after a tech change (e.g., Redis -> Valkey migration)

| Finding | Severity |
|---------|----------|
| Process / BC count mismatch | HIGH |
| Feature count mismatch | MEDIUM |
| Solution-structure mismatch | HIGH |
| Tech-stack mismatch | MEDIUM |

#### Phase 10: BACKLOG.md ↔ works/tasks/

Verify the BACKLOG table matches the actual Task files.

**Checks**:
1. **Row count**: number of rows in BACKLOG.md == file count in `works/tasks/`
2. **Task ID**: Task ID in BACKLOG == Task ID in the file name
3. **Status**: status column in BACKLOG == status field in the Task file's YAML frontmatter
4. **Sprint**: sprint column in BACKLOG == sprint field in the Task file

**How to run**:
```bash
# BACKLOG row count vs actual file count
backlog_count=$(grep -cE '^\| T\d{3}' works/BACKLOG.md)
file_count=$(ls works/tasks/T*.md 2>/dev/null | wc -l)
[ "$backlog_count" -ne "$file_count" ] && echo "MISMATCH: BACKLOG=$backlog_count files=$file_count"
```

**Common findings**:
- Task created but not registered in the BACKLOG table
- Task completed but BACKLOG status not updated (file says done, BACKLOG says in-progress)
- Sprint moved but BACKLOG sprint column not updated

| Finding | Severity |
|---------|----------|
| Row count mismatch | HIGH |
| Status mismatch | HIGH |
| Task ID mismatch | CRITICAL |

#### Phase 11: hstl-oss --manifest ↔ Skill/Command consistency

Verify that the Command/Skill metadata declared by `hstl-oss --manifest`
(in particular the `command_ref` for the six ceremonies) matches the actual
files. Catches drift between the manifest and the repo when a Skill is
renamed or a Command is deleted.

**Checks**:

1. **Command file existence**: the md file pointed to by `manifest.commands[].file_path` exists
2. **Skill file existence**: `manifest.skills[].file_path` (when present) matches the actual SKILL.md
3. **ceremony command_ref ↔ Command files**: all six ceremony=true commands (task:create/start/complete,
   sprint:create/start/complete) exist under `commands/task/*.md` /
   `commands/sprint/*.md`
4. **`--with-ceremony` flag**: the CLI help for ceremony=true commands (`hstl-oss task start --help`,
   etc.) declares a `--with-ceremony` flag
5. **description match**: manifest description matches the Command/Skill's actual frontmatter
   description (partial drift tolerated; complete mismatch = WARN)

**How to run**:

```bash
# Phase 11 core checks — pseudocode
hstl-oss-oss --manifest > /tmp/manifest.json

# 1. Command file existence
jq -r '.commands[] | .file_path' /tmp/manifest.json | while read fp; do
  [ ! -f "$fp" ] && echo "MISSING: $fp"
done

# 2. All six ceremony commands exist
for cmd in task:create task:start task:complete sprint:create sprint:start sprint:complete; do
  ns="${cmd%:*}"; action="${cmd#*:}"
  [ ! -f "commands/$ns/$action.md" ] && echo "MISSING ceremony: $cmd"
done

# 3. --with-ceremony flag exists
for c in "task create" "task start" "task complete" "sprint create" "sprint start" "sprint complete"; do
  hstl-oss $c --help 2>&1 | grep -q "with-ceremony" || echo "NO --with-ceremony: $c"
done
```

**Common findings**:

- Command file renamed but manifest schema not updated (CLI source is stale)
- ceremony=true declared but `--with-ceremony` flag not implemented
- Skill renamed but manifest's skill reference drifts

| Finding | Severity |
|---------|----------|
| ceremony command_ref file missing | CRITICAL |
| --with-ceremony flag missing | HIGH |
| Command/Skill file path mismatch | HIGH |
| description drift | LOW |

**Integration point**: this can be automated via a new
`scripts/check-manifest-sync.sh`; doc-cross-check Phase 11 either calls that
script or inlines the same logic. A separate Makefile target / pre-commit
hook is left as follow-up work.

## Common Findings by Phase (summary)

| Phase | Most common problem | Frequency |
|-------|--------------------|----------|
| 1 | Total declarations not refreshed after Features added | High |
| 2 | Feature mapping missing in Aggregate docs | Medium |
| 3 | New Feature without an FR written | High |
| 4 | Events with no subscribers (dead letter) | Medium |
| 5 | BC name mixing ("Collection" vs "DataCollection") | High |
| 6 | PDD numbers vs measured catalog totals | High |
| 7 | Old paths still referenced after file moves | High |
| 8 | References to files moved into archive | Medium |
| 9 | Feature counts in user project guide not refreshed | Medium |
| 10 | BACKLOG not updated after Task completion | High |
| 11 | ceremony command_ref ↔ Command file drift | Low |

## Output Format

```
═══════════════════════════════════════════════
  DESIGN AUDIT REPORT — {date}
  Target: {path} ({N} documents)
═══════════════════════════════════════════════

Phase 1: Feature ID integrity
  ✅ All {N} unique, totals match

Phase 2: Feature ↔ BC mapping
  ⚠️ {Feature ID}: design-doc link broken → {path}

Phase 3: Feature ↔ FR mapping
  ⚠️ Features without an FR: {ID list}

Phase 4: Event integrity
  ❌ {event}: no subscriber (dead letter)

Phase 5: Terminology consistency
  ⚠️ "Processing" vs "Strategy" — {file list}

Phase 6: Numerical integrity
  ❌ PDD "156" vs catalog "207"

Phase 7: Link health
  ⚠️ {N} broken links

Phase 8: Repo-wide links
  ⚠️ {N} broken under docs/, {N} broken under works/

Phase 9: User project guide consistency
  ❌ Feature count: guide "239" vs catalog "250"

Phase 10: BACKLOG consistency
  ⚠️ {N} status mismatches

Phase 11: manifest ↔ Skill/Command consistency
  ✅ All 6 ceremony Command files present
  ✅ All --with-ceremony flags declared
  ⚠️ description drift on {N} entries (LOW)

═══════════════════════════════════════════════
  SUMMARY
  ─────────────────────────────────────────────
  CRITICAL: {N} | HIGH: {N} | MEDIUM: {N} | LOW: {N}
  ─────────────────────────────────────────────
  Phases passed: {N}/10 | Items inspected: {N} | Issues: {N}
═══════════════════════════════════════════════

  Recommended fixes (priority order):
  1. [CRITICAL] {concrete fix}
  2. [HIGH] {concrete fix}
  3. [MEDIUM] {concrete fix}

  Apply fixes? (all / select / no)
```

### Severity criteria

| Severity | Meaning | Recommended handling |
|----------|---------|---------------------|
| CRITICAL | Design integrity broken (ID duplicate, double membership) | Fix immediately and re-run audit |
| HIGH | Major traceability/integrity error | Fix within the current sprint |
| MEDIUM | Warning level, no short-term impact | Address in the next sprint |
| LOW | Quality-improvement suggestion | Add to backlog |

> **Required: create a Task before fixing**: when the user replies "fix", first create a `works/tasks/TNN-doc-cross-check-fix-{date}.md` BACKLOG Task and then proceed. After the fix, run task:complete -> commit. This way audit fixes are also traceable through Git history.

## Run Modes

| Mode | Command | Behavior |
|------|---------|----------|
| Full audit | `/doc-cross-check` | All 10 phases |
| Path-scoped | `/doc-cross-check export/` | Only that path |
| Specific phase | `/doc-cross-check --phase=4` | Event integrity only |
| Design only | `/doc-cross-check --phase=1-7` | Phases 1-7 (design docs) |
| Repo only | `/doc-cross-check --phase=8-10` | Phases 8-10 (repo-wide) |
| Auto-fix | `/doc-cross-check --fix` | Apply fixable findings immediately |

## Related Skills

- `work-audit` -- checks Task/Sprint workflow compliance (work process, not design)
- `doc-review` -- checks document template compliance
