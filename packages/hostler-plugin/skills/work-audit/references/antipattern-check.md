# work-audit — Antipattern Checklist (Guide Appendix B)

> Source: Appendix B (the original `agent-mcp-development-guide.md` is archived)
> This file is the antipattern audit checklist now folded into work-audit.
> **Read-only**: work-audit only detects; fixes are handled by other skills or after user confirmation.

## How to Use

work-audit applies this checklist automatically during Sprint/Repo audits. Detected items appear under the
"Q (Quality)" axis in the report; each item is reported with (relevant file, guide reference, recommended fix).

Execution flow:
```
work-audit --sprint sprint-N
  → existing D/W/U axis checks
  → Q axis (Appendix B.1 + B.2) checks in parallel
  → unified report output
```

---

## Q Axis — Skill Antipatterns (Guide Appendix B.1, 10 items)

| Q# | Antipattern | Detection | Recommended fix |
|----|-------------|-----------|-----------------|
| Q1 | Vague description | `grep -L "Performs\|Creates\|Manages\|Validates\|Generates\|Provides\|Guides\|Analyzes" skills/*/SKILL.md` returns no match | Use an action verb: "performs X" |
| Q2 | No negative condition | `description` lacks "Do NOT\|do not\|do not use" | Add "Do NOT use when..." |
| Q3 | Description over 250 chars | Frontmatter description > 250 chars | Move the gist to the front, push secondary detail to the body |
| Q4 | SKILL.md over 500 lines | `wc -l skills/*/SKILL.md` > 500 | Split into `references/` |
| Q5 | Hardcoded lookup table | A 20+-row table with static data in SKILL.md body | Move to a CLI response or `references/` |
| Q6 | Inspection skill performs auto-fix | SKILL.md mentions a `--fix` option + uses Write/Edit tools | Enforce read-only |
| Q7 | 10+ ALWAYS/NEVER | "ALWAYS\|NEVER" appears 10+ times | Reserve for data-loss/security risks only |
| Q8 | Missing trigger-eval.json | `skills/{name}/evals/trigger-eval.json` does not exist | Add at least 8 cases |
| Q9 | Project-specific language | Hardcoded project-specific proper nouns | Use technology-neutral wording |
| Q10 | Model-recommended verbs | "may\|might\|can perform" | Use the imperative ("performs") |

### Example check script

```bash
# Q1/Q2: bulk description-quality check
python3 scripts/measure-skill-quality.py  # 5-axis scoring + report
```

---

## Q Axis — CLI Antipatterns (legacy MCP-Tool baseline, archive review needed)

| Q# | Antipattern | Detection | Recommended fix |
|----|-------------|-----------|-----------------|
| Q11 | One-sentence description | registry.Register description length < 100 chars | 5-part structure |
| Q12 | Choices listed as strings | description includes "a/b/c" + the field has no enum | `enum: [...]` |
| Q13 | No Annotations | ToolAnnotations not set | readOnly/destructive required |
| Q14 | Inconsistent error format | Error responses have no error_category | Use a single `ok/blocked/error` helper |
| Q15 | Missing error_category | Error response map has no error_category key | Use the SERF six-category taxonomy |
| Q16 | Missing recovery_hint | Error response has no recovery_hint key | Specify a concrete tool call |
| Q17 | No L1 Server Instructions | (archive — `cmd/mcp/` removed) | — |
| Q18 | Hardcoded validation section heading | Validator hardcodes `## Artifacts` strings | Externalize via env var / YAML |
| Q19 | Cross-cutting concern injected per handler | Same function call repeated across handlers | Use dispatch middleware |
| Q20 | Direct file editing allowed | Docs say "edit files in works/" | Provide a partial-update tool |
| Q21 | advertise = dispatch identical list | registry.AddToServer without LowUse filter | Add LowUse flag + advertiseMode |
| Q22 | Policy weakened on validation failure | strict → warn change in history | Address by correcting the result |
| Q23 | SDK-upgrade regression test skipped | No evidence of pytest/go test in the SDK-upgrade sprint | Run the full Phase 0 |
| Q24 | ID counter not advanced | INSERT without `id_counters` UPDATE | Update both inside the same transaction |
| Q25 | Worktree absolute path stored in DB | Path containing `/worktree/` is in the DB | Normalize to a relative path |

### How to check

Q11-Q16: CLI subcommand description quality and schema are reviewed manually.
(measurement script archived)

---

## Combined Report Format

The final work-audit report adds:

```markdown
## Q Axis — Antipattern Audit

| Axis | Item | Detected | Severity |
|------|------|:---:|:---:|
| Q2 | Skill missing negative condition | 2 / 31 | WARN |
| Q11 | Tool one-sentence description | 0 / 38 | — |
| Q15 | Missing error_category | 1 / 38 | INFO |
...

### Details

- **Q2 Skill missing negative condition**:
  - skills/foo/SKILL.md — recommend adding "Do NOT use for..."
- **Q15 Missing error_category**:
  - cli/cmd/hostler/cmd/bar.go:123 — recommend using the _error() helper
```

---

## Operating Principles

1. **Read-only**: never auto-fix with `--fix` (lessons M02, M04)
2. **Report only**: fixes are handled by another Task or skill after user confirmation
3. **Reuse scripts**: invoke measurement scripts to avoid duplicate implementation
4. **Single source of truth**: antipattern definitions follow Appendix B (the original guide is archived)

---

## Related Scripts

- `scripts/measure-skill-quality.py` — Skill 5-axis scoring (partially automates Q1-Q10)
- `scripts/measure-plugin-components.py` — Command/Hook/Agent baseline
