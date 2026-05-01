---
name: claude-md-audit
description: Audits the user's project guide (CLAUDE.md) for size and structure against Anthropic's 200/300/60-line targets, then proposes a migration matrix to move sections into `.claude/rules/` or `docs/`. Use this skill whenever the user mentions "CLAUDE.md audit", "memory file optimization", "progressive disclosure", "user project guide audit", "slim down CLAUDE.md", or whenever their CLAUDE.md exceeds 200 lines, even if they haven't explicitly asked. Also invoke before any planned memory-cleanup sprint. Read-only — never edits files. Do NOT use for SKILL.md audits (use `measure-skill-quality.py`).
compatibility:
  tools: [Read, Grep, Glob, Bash]
argument-hint: "[CLAUDE.md path]"
paths: [".claude/**/*", "docs/**/*", "**/*.md"]
user-invocable: false
---

# claude-md-audit — automatic CLAUDE.md optimization diagnosis + migration proposal

**Triggers**: "user project guide audit", "measure CLAUDE.md size",
"memory file optimization", "slim down CLAUDE.md", "apply progressive disclosure",
"claude md audit"

Quantitatively measures the size and structure of the user project guide, compares it
against Anthropic's official recommendations, identifies migration candidates per
section, and proposes a migration matrix. Designed to be project-independent so it can
be reused beyond hostler-plugin.

This skill is **read-only** — it never modifies files (measure-vs-write
separation principle — user project docs). When changes are needed, generate
explicit Tasks from the proposed Task templates and execute them separately.

## 5-step operation

### Step 1 — Measure

- `wc -l <user project guide>` total line count
- Per-section (`^##`) line distribution + each section's share
- Number of `@import` or `@path` references (whether the user project guide loads external files)
- Whether `.claude/rules/*.md` exist + check each file's `paths` field
- Code-block / table / list ratio (density indicators)

### Step 2 — Anthropic baseline comparison

Anthropic's official memory docs + hostler research §3 baseline:

| Target | Level | Criterion |
|--------|-------|-----------|
| Excellent | ≤ 60 lines | Anthropic "minimal user project guide" example level |
| Good | ≤ 200 lines | Official target ("target under 200 lines per user project guide file") |
| Acceptable | ≤ 300 lines | Soft ceiling — exceeding it triggers an adherence-degradation warning |
| Bloated | > 300 lines | Action required immediately |

Output example:
```
User project guide measurement
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total:    414 lines  — Bloated (207% of the 200-line limit)
  Sections: 12
  Largest:  "Skill development standards" 138 lines (33.3 %)
  @import:  0
  .claude/rules: 0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Step 3 — Identify per-section migration candidates

Migration decisions follow design principles P1~P7 from the research §4:

- **P1 Universal Context Only** — keep only context needed in every session in the user project guide
- **P2 Action > Knowledge** — "do X" (action) takes priority over "X exists" (reference)
- **P3 Reference > Duplicate** — don't repeat detail; keep only reference pointers
- **P4 Path-scoped Rules** — file-type-specific rules move to `.claude/rules/` + `paths`
- **P5 Auto-generated Index** — manually maintained lists are replaced with auto-generation scripts
- **P6 Version-aware Pointers** — version/status info goes to a separate file (roadmap.md, etc.)
- **P7 Universal Rules First** — core rules at the top, metadata at the bottom

Migration decision rules:
1. `^## .*development standards` / `^## .*development` → `.claude/rules/` (apply paths scoping)
2. `^## environment variables` → migrate to user project docs (verify it already exists)
3. `^## skill trigger` → replace with auto-generation script
4. `^## roadmap` → user project docs
5. Section ≥ 50 lines AND applies only to specific file types → paths-scoping candidate
6. Many code examples → migrate to user project docs and leave a pointer

### Step 4 — Present the migration matrix

Output format (user project docs §5.4 style):

```
| Section | Current lines | Class | Migration target | paths pattern | Expected reduction |
|---------|---------------|-------|------------------|---------------|--------------------|
| Skill development standards | 138 | P4 | .claude/rules/skill-development.md + user project docs | skills/**/*, **/SKILL.md | -134 |
| MCP Tool development standards | 71 | P4 | .claude/rules/mcp-tool-development.md | cli/**/*.go | -67 |
| Skill trigger summary | 33 | P5 | user project docs (auto-generated) | — | -28 |
```

### Step 5 — Execution guidance

- Generate a Task template for each proposed migration (in a form ready for `task:create`)
- Sprint planning guide: XS / S size estimates
- Live verification plan: use `/memory` + the `InstructionsLoaded` hook
- **Never modify** — this skill only proposes; the user or a separate Task does the modifications

## How to use it

### Basic usage

```
/claude-md-audit
```

- Audits the user project guide in the current working directory automatically
- Runs the 5 steps in order and prints the result

### Specify a path

```
/claude-md-audit /path/to/project/<user project guide>
```

- Audits another project's user project guide (external user projects, etc.)

### Run the script directly

Run the Python script without auto-triggering the skill:

```bash
python3 skills/claude-md-audit/scripts/audit.py
python3 skills/claude-md-audit/scripts/audit.py /path/to/other/<user project guide>
```

## Output conventions

1. **Measurement**: ASCII frame + 5 indicators (total / sections / largest / imports / rules)
2. **Comparison**: Verdict of Excellent / Good / Acceptable / Bloated + rationale
3. **Migration matrix**: per-section table (research §5.4 style)
4. **Task templates**: ready for `hstl-oss task create` per migration candidate
5. **Summary**: expected final line count + whether the goal is met

## References

- Design principles P1~P7: `references/design-principles.md`
- Migration examples (maintainer reference): `references/migration-examples.md`
- Research report: user project docs
- Anthropic official: https://code.claude.com/docs/en/memory
- Change history (maintainers only): `references/changelog.md`

## Output example

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  User project guide audit — <your-project>
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Measurement
  ───────────
  Total:    103 lines
  Sections: 9
  Largest:  "MCP Tool first-use rule" ~18 lines
  @import:  0
  .claude/rules: 3

  Comparison
  ──────────
  Good — within 51.5% of the 200-line target
  73.6% of the 140-line goal (comfortable margin)

  Migration matrix
  ────────────────
  No further migration candidates (goal reached)

  Recommendations
  ───────────────
  - drift watch: python3 scripts/generate-skill-triggers.py
  - dev-standard sync: .claude/rules/ vs the user project guides directory
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Read-only principle

This skill **never performs modifying actions**. Following the
measure-vs-write separation principle (user project docs):

- ❌ no `--fix` option
- ❌ no direct edits to the user project guide
- ❌ no auto-generation of `.claude/rules/`
- ✅ measurement + comparison + suggestion only
- ✅ modifications are executed by the user via separate explicit Tasks

The user verifies actual paths-scoping behavior manually with the `/memory`
command or the `InstructionsLoaded` hook (see Limitations section).
