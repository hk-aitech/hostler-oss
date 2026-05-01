---
description: AI/sub-agent project context query — compact output under 200 tokens for prompt injection. Use when an agent needs lightweight project context, or when the user asks for "context for an agent" or wants the 15-Facet Model with `--facets/--layer/--all`.
argument-hint: "[--ticket <work-ticket>] [--facets <list>] [--layer <1|2|3>] [--all]"
allowed-tools: Bash(hstl-oss:*)
---

# Project Context

Returns compact project context suitable for injecting into an AI/sub-agent prompt.

## CLI usage

```
Bash("hstl-oss context")
```

Text mode:
```bash
hstl-oss context
```

JSON parsing:
```bash
hstl-oss context -o json | jq '.data.sprint'
```

## Modes (15-Facet Model)

The command operates in two modes:

### (a) Default sprint/task context mode (no arguments)

```bash
hstl-oss context           # compact text (~200 tokens)
hstl-oss context -o json   # structured data
```

Output:
```
[project] branch:main uncommitted:0 ahead:0
[sprint] "title" 3/6 done (50%)
[task] "title" (in-progress) | next: ...
[rules] commit-style, const required, jq, Command First, 1 Task = 1 Commit
[issues] none
```

### (b) Facet mode (any of `--facets` / `--layer` / `--all`)

Project Context 15-Facet Model. Queries the project context itself across the
core 12 facets × 3 Layer grid.

```bash
# Layer 1 Stable Core 7 (objective/domain/environment/persona/policy/roadmap/knowledge_index)
hstl-oss context --layer 1

# Layer 1+2 (default. Layer 2 Dynamic 5 = milestone/track/sprint/task/market_state)
hstl-oss context --layer 2

# All — for debugging
hstl-oss context --all

# Explicit facets (shorthand: "objective" -> "core.objective")
hstl-oss context --facets objective,track
hstl-oss context --facets objective,track --layer 1
```

JSON output format:
```json
{
  "data": {
    "mode": "facet",
    "layer": 1,
    "values": [
      { "name": "core.objective", "layer": 1, "content": "...", "source": "file:docs/00-project/pdd.md" },
      ...
    ]
  }
}
```

Text output (one line per facet, content compressed to 200 chars):
```
[core.objective] L1 ---uuid: ...title: ...
[core.track] L2 ---track-id: track-context-15-facet...
```

## Sub-agent injection

```
Agent({
  prompt: "Project context:\n" + contextOutput + "\n\nIn this context, ..."
})
```

Facet mode use:
```
Agent({
  prompt: "Layer 1 context:\n" + run("hstl-oss context --layer 1") + "\n\n..."
})
```

## References

- User briefing: `/hstl-oss:project:brief`
- Skill guide: `skills/project-management/SKILL.md`
