---
name: doc-review
description: Validates Task / Sprint / ADR documents against hostler standards — frontmatter completeness, placeholders (TODO/TBD), checkboxes, empty sections — and auto-fixes them with `--fix`. Use this skill whenever the user mentions doc review, document quality, frontmatter validation, sprint:complete Phase 1, after creating a Task, or any phrasing like "check my docs", "validate this Task file", or "fix the placeholders". Be a little pushy — invoke this even if the user just hints at doc cleanup. Do NOT use for code review (use code-review) or cross-document checks (use doc-cross-check).
compatibility:
  tools: [Read, Write, Edit, Glob, Grep, Bash]
argument-hint: "[--session|--sprint|--repo] [--fix] [path]"
paths: ["**/*.md", "works/**/*", "docs/**/*"]
---

# Doc Quality Check — Document Quality Inspection + Auto-fix

Verifies that project documents (Task, Sprint, reports, ADR, requirements)
follow the hostler templates, and auto-fixes them when `--fix` is supplied.

## Table of Contents

1. Scope (what to inspect)
2. Execution modes
3. Per-document-type checkpoint summary
4. Workflow
5. --fix mode behaviour
6. Using the scripts
7. Output format
8. Reference Files
9. Related skills

---

## Scope

Specify which documents to inspect. Default is `--session` if no flag is given.

| Option | Target files | Interpretation |
|------|----------|------|
| `--session` (default) | Documents modified in the current session | `*.md` files changed since session start, via `git diff --name-only HEAD~N` |
| `--sprint` | All documents for the active Sprint | `works/sprints/active/*/tasks/`, `works/sprints/active/*/SPRINT.md`, plus `docs/` and `commands/` modified during the Sprint |
| `--repo` | Every document in the repo | Every `*.md` file under `works/`, `docs/`, `commands/`, `skills/*/SKILL.md` |

> **commands/ included**: A past incident showed that stale references in
> `commands/sprint/*.md` were missed by doc-review. Since then, the
> `--sprint` / `--repo` scopes include `commands/**/*.md`. The dedicated
> `scripts/check-command-files.sh` script handles frontmatter and stale-pattern checks.

### Scope Resolution Rules
- `--session` is the default, so a bare `/doc-review` only inspects design docs / reports / Task files modified this session
- `--sprint` is auto-supplied when work-audit calls doc-review from inside `sprint:complete`
- Use `--repo` explicitly when you need a full repo-wide review
- If you pass an explicit path (`/doc-review <path>`), only that path is inspected (scope is overridden)

---

## Execution Modes

| Mode | Command | Behaviour |
|------|------|------|
| Inspect only | `/doc-review` | Inspect session-modified documents → print report |
| Inspect + fix | `/doc-review --fix` | Find issues → fix immediately |
| Sprint scope | `/doc-review --sprint` | Inspect all active Sprint documents |
| Whole repo | `/doc-review --repo` | Inspect every document in the repo |
| Specific path | `/doc-review works/sprints/active/sprint-NN/` | Inspect only that path |

---

## Per-document-type Checkpoint Summary

### Task files

**Location**: `works/sprints/*/tasks/T*.md`
**Reference**: `references/task-checklist.md`

| # | Check | Severity | Notes |
|---|------|------|------|
| 1 | All 9 frontmatter fields present | BLOCK | id, title, type, sprint, status, priority, estimate, depends_on, created |
| 2 | Frontmatter values valid | BLOCK | type/status/priority/estimate enum check |
| 3 | Filename pattern (`TNN-kebab-case.md`) | BLOCK | |
| 4 | `## Requirements` or `## Purpose` present + 10+ words | BLOCK | placeholder (TBD/TODO) detected → WARN |
| 5 | `## Done Criteria` present + 2+ checkboxes | BLOCK | item with fewer than 10 chars → WARN |
| 6 | `## Implementation` or `## Scope Limits` present | WARN | |
| 7 | `depends_on` references existing Task | WARN | |
| 8 | `## Result` present + sufficient content (done only) | WARN | placeholder detection, 2+ lines |
| 9 | All completion criteria checked (done only) | WARN | |
| 10 | Body-wide placeholder detection | WARN | TBD/TODO/FIXME more than 3 |
| 11 | Backtick paths in `## References` exist on disk | WARN | only paths containing `/`, missing → stale_ref |

#### Backtick-path Check in `## References`

**Target**: backtick-wrapped **paths containing `/`** in the Task body's `## References` section
(e.g. `` `cli/pkg/foo.go` ``, `` user-project docs ``).

**Exceptions**: code snippets like (`` `const Foo = 1` ``, `` `func Bar()` ``)
have no `/` and are out of scope. Tokens ending in `(`, `,` are also not paths.

**Inspection procedure — manual AI execution example**:

```bash
# 1) Extract the ## References section from the Task file
sed -n '/^## References/,/^## /p' "$TASK_FILE"

# 2) Extract only backtick paths containing / (grep + ripgrep filter)
rg -o '`[^`]+`' "$TASK_FILE" | tr -d '`' | grep '/' | sort -u > /tmp/refs.txt

# 3) Verify each path exists
while read -r ref; do
  [ -e "$ref" ] || echo "STALE: $ref"
done < /tmp/refs.txt
```

**Severity policy**: WARN (not BLOCK). If a Task file has aged and a path
is stale, surface it without blocking Task progress. BLOCK loses force when
false-positives are common, so we stay conservative with WARN.

**Automation follow-up**: a utility function (new `cli/pkg/docreview/` plus a
dedicated doc-review CLI subcommand) will be implemented in a separate Task.
For now this skill guide is sufficient.

### SPRINT.md

**Location**: `works/sprints/*/SPRINT.md` | **Reference**: `references/sprint-checklist.md`

| # | Check | Severity |
|---|------|------|
| 1 | Frontmatter (id, title, status, start, end) | BLOCK |
| 2 | `## Metadata` table | BLOCK |
| 3 | `## Goal` section | BLOCK |
| 4 | `## Task List` table | BLOCK |
| 5 | `## Done Criteria` section | WARN |
| 6 | Task count consistency (table vs files) | WARN |

### ADR

**Location**: user project docs

| # | Check | Severity |
|---|------|------|
| 1 | Title contains the ADR number | BLOCK |
| 2 | Status indicator (Accepted/Deprecated/Superseded/Proposed) | BLOCK |
| 3 | `## Context` section | BLOCK |
| 4 | `## Decision` section | BLOCK |
| 5 | `## Consequences` section | WARN |
| 6 | `## Alternatives` section | INFO |

### Design Documents (BC specs, Aggregates)

**Location**: user project docs (excluding index.md)

| # | Check | Severity |
|---|------|------|
| 1 | `# Title` (h1) present | BLOCK |
| 2 | Document ID + version + date present | WARN |
| 3 | `## Definition` section or intro paragraph (10+ words) | BLOCK |
| 4 | Aggregate Root code sketch (1+ code block) | WARN |
| 5 | `## Dependencies` section (Upstream/Downstream) | WARN |
| 6 | `## Implementation Notes` or KB references | INFO |
| 7 | Substantive content of 50+ lines | WARN |

### Deployment / Operations Documents

**Location**: user project docs

| # | Check | Severity |
|---|------|------|
| 1 | `# Title` (h1) present | BLOCK |
| 2 | Process / service name spelled out | WARN |
| 3 | Docker / infra config present (when applicable) | INFO |
| 4 | Substantive content of 30+ lines | WARN |

### Reports

**Location**: user project docs | **Reference**: `references/report-checklist.md`

| # | Check | Severity |
|---|------|------|
| 1 | `# Title` (h1) present | BLOCK |
| 2 | Authoring date present | BLOCK |
| 3 | Table of contents (recommended for 5+ sections) | WARN |
| 4 | `## Conclusion` section | WARN |
| 5 | Substantive content of 50+ lines | INFO |

---

## Workflow

### Step 1 — Collect Target Files

Use `find` against the given path or whole repo to collect Task (`T*.md`),
SPRINT.md, report (user project docs), and ADR (user project docs) files.

### Step 2 — Per-type Inspection

Read each file and check it against the appropriate checklist
(`references/`). Severity scheme:

- **BLOCK**: must fix immediately. The document is incomplete.
- **WARN**: should fix. Recommended for quality.
- **INFO**: informational. Fixing is optional.

### Step 3 — Print Report

Group output by severity using the format in "Output format" below.

### Step 4 — Auto-fix (--fix mode)

When `--fix` is supplied, fix items that can be safely auto-fixed.

---

## --fix Mode Behaviour

With `--fix`, the items below are auto-fixed. Fixes are delegated to the
**tech-writer agent**.

### Auto-fixable

| Issue | Fix |
|------|----------|
| Done Task missing `## Result` | Insert default stub template |
| Done Task with unchecked checkboxes | Convert `- [ ]` → `- [x]` |
| Frontmatter field missing | Add with default (estimate: S, depends_on: []) |
| Filename has non-ASCII / special chars | `git mv` to ASCII kebab-case slug |
| SPRINT.md missing fields / sections | Add frontmatter defaults, insert `## Done Criteria` stub |
| Report / ADR missing sections | Insert empty `## Conclusion` / `## Consequences` stub |

### Not Auto-fixable (human judgement)

| Issue | Reason |
|------|------|
| Writing the body of `## Requirements` / `## Done Criteria` | Requires business judgement |
| Listing changed files in `## Result` | Requires reviewing the implementation |
| Task count mismatch | Requires either adding/removing files or editing the table |
| ADR status, report authoring date | Cannot be inferred reliably |

---

## Using the Scripts

### Task File Validation

`scripts/check-task-files.sh` performs a quick frontmatter validation on Task files.

```bash
bash ${CLAUDE_SKILL_DIR}/scripts/check-task-files.sh works/sprints/active/
```

Checks: filename pattern, all 9 frontmatter fields, required sections, and the
result section for done tasks. Returns exit 1 if there is at least 1 FAIL.
Suitable as a CI gate.

### Command File Validation

`scripts/check-command-files.sh` validates `commands/**/*.md` slash command definitions.

```bash
bash ${CLAUDE_SKILL_DIR}/scripts/check-command-files.sh commands/
```

Checks:
- Frontmatter present (first line `---`)
- `description:` field present
- **Stale pattern detection** — literal references to deleted files like `CEREMONY.md`, `SESSION-CONTEXT.md`. Historical context such as "CEREMONY.md deprecated" is allowed.
- H1 title present (WARN)

Regression test: `scripts/check-command-files_test.sh` — fixture-based PASS/FAIL/false-positive 4-scenario test.

---

## Output Format

Group output by severity:

```
## Document Quality Check Result

### BLOCK (must fix) — 3
1. [X] T###-feature-name.md: frontmatter 'estimate' missing
2. [X] T###-feature-name.md: ## Requirements section missing

### WARN (recommended fix) — 2
3. [!] T###-feature-name.md: done but ## Result section missing

### INFO — 1   |   PASS — 25

Total 31 inspected: BLOCK 3 | WARN 2 | INFO 1 | PASS 25
```

In `--fix` mode, items that were auto-fixed are tagged `[FIXED]`.

---

## Reference Files

See `references/` for detailed checklists:

- **`references/task-checklist.md`** — Task file checks in detail, frontmatter examples, auto-fixable / non-fixable items
- **`references/sprint-checklist.md`** — SPRINT.md checks in detail, cross-validation rules
- **`references/report-checklist.md`** — Report / ADR / requirements / design document checks, requirements ID system

Quick validation scripts:

- **`scripts/check-task-files.sh`** — bash script to validate Task file frontmatter + required sections

---

## Related Skills

- `task-management` — applies templates at Task creation
- `sprint-management` — applies templates at Sprint creation
- `doc-templates` — document template references
- `archive` — escalation when archive candidates surface during a quality check

## YAML-schema-based Validation

doc-review references two YAMLs from the `doc-templates` skill as its validation source:

| YAML | Path | Validation content |
|------|------|----------|
| `frontmatter-schemas.yaml` | `skills/doc-templates/references/frontmatter-schemas.yaml` | Required / recommended frontmatter fields per doc type |
| `required-sections.yaml` | `skills/doc-templates/references/required-sections.yaml` | Required / recommended sections per doc type |

### frontmatter-schemas.yaml-based Validation

1. Read the `doc_type` frontmatter field of the target document
2. If `doc_type` is missing, infer the type from the file location (e.g. user project docs → `adr`)
3. Pull the `required` field list for that type from `frontmatter-schemas.yaml`
4. Confirm each required field exists → if missing, **BLOCK**
5. Confirm each `recommended` field exists → if missing, **WARN**

```python
# Pseudocode
schema = load_yaml("skills/doc-templates/references/frontmatter-schemas.yaml")
doc_type = frontmatter.get("doc_type") or infer_type_from_path(file_path)
for field in schema["schemas"][doc_type]["required"]:
    if field["field"] not in frontmatter:
        report(BLOCK, f"frontmatter '{field['field']}' missing")
```

### required-sections.yaml-based Section Validation

1. Pull the section list for that type from `required-sections.yaml`
2. Parse the H2/H3 headings of the document and compare against the alias list
3. If a section has a `condition` field, evaluate it (e.g. `status == done`)
4. If the section is absent, report at the appropriate severity (BLOCK / WARN / INFO)

```python
# Pseudocode
sections = load_yaml("skills/doc-templates/references/required-sections.yaml")
headings = parse_headings(doc_content)  # [h2, h3, ...]
for section in sections["schemas"][doc_type]["required"]:
    if not any(h in section["aliases"] for h in headings):
        if evaluate_condition(section.get("condition"), frontmatter):
            report(section["severity"], f"section '{section['heading']}' missing")
```

> **Current state**: SKILL.md guide updates are complete.
> Auto-validation script implementation is deferred to a future Task.
> Today, the AI manually validates by Reading the two YAMLs.

## Cross-validating Structural Documents

For projects that have a `.hostler/project-config.md`, doc-review additionally:
- Compares the BC structure section of project-config.md against the actual `src/` project

## Per-project Extensions (auto-detected)

This skill auto-detects `.hostler/project-config.md` at the project root at runtime.
If that file exists and has a `## Per-skill Extensions` → `### doc-review` section,
project-specific rules are applied **in addition** to the shared rules.

- Detection path: `{project_root}/.hostler/project-config.md`
- If absent, only the shared rules apply (backward compatible)

## References

- Changelog (maintainer-only): `references/changelog.md`
