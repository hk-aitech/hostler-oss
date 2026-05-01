# skills/_shared — Shared Python utilities

> Introduced when a duplicated pattern shows up in 2+ skills, the pattern is
> extracted into this module. Preemptive generalization is avoided per YAGNI.

## Modules

| Module | Purpose | Introduced via |
|---|---|---|
| `config_loader` | Dual-search loader for skill config JSON files (script directory → project_root) | (pattern generalization) |
| `analysis-methodology.md` | Analysis methodology SSOT — Perspective Mining, Knowledge Gap loop, Chain-of-Verification, Source-agnostic Grounding Pass | (extracted from deep-research) |
| `report-quality-rubric.md` | Report quality rubric — RACE 4 axes (Depth/Breadth/Following/Readability) + FACT 4 axes (Sources/Accuracy/Recency/Cross-Verification) | (extracted from deep-research) |
| `citation-quality.md` | Citation quality criteria — evidence grades (A/B/C/D), recency, cross-verification obligation, citation format | (extracted from deep-research) |

## config_loader

A dual-search loader that reliably finds a skill's config file even when the
skill operates against an external path (e.g. `/tmp/foo.md`, another
project root).

### Example

```python
from pathlib import Path
from skills._shared.config_loader import load_json_config

cfg = load_json_config(
    config_name="my-config.json",
    script_file=__file__,  # __file__ of the calling script
    project_root=Path.cwd(),
    legacy_relative_path="skills/my-skill/config/my-config.json",
)
if cfg is None:
    # File not found or parse failure
    ...
```

### Search Order

1. **Script directory first** (priority 1): `<script_dir>/../config/<config_name>`
   — most stable since the config ships with the skill itself.
2. **project_root** (optional fallback): `project_root /
   legacy_relative_path` — for legacy compatibility or user override.

## Skills That Use This

- `claude-md-audit/scripts/audit.py` (generalized — `_load_deprecated_config`,
  `_load_conflict_config` delegate to this util)

## Design Rationale

- KB `docs/07-knowledge/architecture/config-loading.md` A01
- Issue: claude-md-audit could not find its config when auditing external files

## Extension Principles

- **YAGNI**: apply when the same pattern actually recurs in another skill. No preemptive batch application.
- **Self-contained**: this module uses only the standard library (json, pathlib).
- **Backward compat**: signature changes must keep existing callers working.
