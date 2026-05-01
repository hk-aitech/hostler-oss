# CLAUDE.md migration case study — hostler-plugin

A real Sprint case you can use as a reference when interpreting the migration
matrix output of the `claude-md-audit` skill. The step-by-step walk-through of
how hostler-plugin's CLAUDE.md was reduced from **414 lines → 103 lines (-75%)**.

## Initial state (before)

```
CLAUDE.md measurement — 2026-04-11 before
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total:    414 lines — Bloated (207% of Anthropic 200-line target)
  Sections: 12
  Largest:  "Skill development standards" 138 lines (33.3 %)
  @import:  0
  .claude/rules: 0 (none yet)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Top-bloat sections:
1. Skill development standards (138 lines, 33.3 %)
2. MCP Tool development standards (71 lines)
3. Command development standards (46 lines)
4. Skill trigger summary (33 lines)
5. Roadmap (26 lines)
6. Environment variables (18 lines)

## Migration step-by-step

| Task | Section | Principle | Migration target | paths pattern | Reduction | Cumulative |
|------|---------|-----------|------------------|---------------|-----------|------------|
| | Skill development standards | P4 | `.claude/rules/skill-development.md` + `docs/04-guides/skill-development-standard.md` | `skills/**/*`, `**/SKILL.md` | -132 | 414 → 282 |
| | MCP Tool development standards | P4 | `.claude/rules/mcp-tool-development.md` + `docs/04-guides/mcp-tool-development-standard.md` | `cli/**/*.go` | -65 | 282 → 217 |
| | Command development standards | P4 | `.claude/rules/command-development.md` + `docs/04-guides/command-development-standard.md` | `commands/**/*.md` | -40 | 217 → 177 |
| | Skill trigger summary | P5 | `docs/03-design/skill-trigger-index.md` (auto-generated) + `scripts/generate-skill-triggers.py` | — | -27 | 177 → 150 |
| | Environment variables | P3 | `docs/04-guides/configuration.md §4~§5` (pointer) | — | -12 | 150 → 138 |
| | Roadmap | P6 | `docs/00-project/roadmap.md` | — | -22 | 138 → 116 |
| | Final slim + consolidation | P7 | unified `## Development standards` section + section reordering | — | -13 | 116 → 103 |

## Final state

```
CLAUDE.md audit — hostler-plugin 2026-04-11 after
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total:    103 lines — ✅ Good (51.5% of the 200-line target)
  Sections: 8 L2 sections
  Largest:  "MCP Tool first-use rule" 22 lines
  @import:  0
  .claude/rules: 3
              - skill-development.md (paths: skills/**/*, **/SKILL.md)
              - mcp-tool-development.md (paths: cli/**/*.go)
              - command-development.md (paths: commands/**/*.md)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Migration pattern types

### Type A — Path-scoping with dual storage (//)

The biggest win. Author both `.claude/rules/*.md` (for AI auto-load) and
`docs/04-guides/*-development-standard.md` (for human discovery)
simultaneously.

**Pros**: only loads when paths match (token saving) + humans can find it via git grep
**Cons**: drift risk (same content in two files)
**Mitigation**: claude-md-audit watches for drift (this skill)

### Type B — Auto-generated Index

Replace a manually maintained list with a script. We empirically caught
**a missing skill in the manual table** — proof that manual maintenance
fails.

**Pros**: zero drift guaranteed, CI integration possible (`git diff --exit-code`)
**Cons**: static output format (a separate index is needed when queries are required)

### Type C — Pointer Reduction

Replace a CLAUDE.md copy of content that already lives in another doc with a
summary + pointer.

**Pros**: simple work, quick win
**Cons**: small reduction (tens of lines)

### Type D — Structure Consolidation

Merge multiple small sections into one + reorder sections. 3 individual
pointers → unified "Development standards" section, 4 meta sections → 1
"Status / Project / Roadmap" section.

**Pros**: less cognitive load (easier comparison), fewer section overheads
**Cons**: a unified section can occasionally grow long (balance required)

## Lessons

1. **Path-scoping has the largest impact** — accounted for ~60% of the 74% total reduction
2. **Auto-generation solves drift** — surfaced an actual missing entry
3. **SSOT beats duplication** — env variables already lived in configuration.md (duplication removed)
4. **Order matters too** — section reordering trimmed only 13 lines but improved AI attention

## Applying this to other projects

1. `python3 skills/claude-md-audit/scripts/audit.py /path/to/other/CLAUDE.md`
2. Plan a Sprint based on the Task templates in the migration-matrix output
3. Execute each migration as 1 Task = 1 Commit (minimize drift)
4. After completion, verify paths-scoping behavior with `/memory` + the `InstructionsLoaded` hook

Sprint sizing: hostler-plugin was 7 Tasks (~7 commits), roughly half a day of focused work.
