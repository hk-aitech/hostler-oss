---
description: Analyze and set the Task estimate as a T-shirt size (alias for `hstl-oss task update --estimate`). Use when the user says "estimate this task", "size this task", or "set the t-shirt size".
allowed-tools: Read, Glob, Bash(hstl-oss:*)
argument-hint: "<task-id> <size>"
---

# Task Estimate

Sets the Task's estimate as a T-shirt size.

## T-shirt size guidance

| Size | Expected duration | Description |
|--------|-----------|------|
| XS | ≤ 30 min | Typo fix, config value change, simple text edit |
| S | 30 min–2 h | Add an API endpoint, implement a single method, simple bug fix |
| M | 2–4 h | New controller, add a test suite, medium-complexity feature |
| L | 4–8 h | Implement a new service, large refactor — **splitting recommended** |
| XL | 1–2 days | Architecture change, multi-module integration — **splitting required** |

> Tasks estimated L/XL should be broken into sub-tasks or split before `sprint:start` into smaller units.

## Instructions

> **CLI alias**: This Command is an alias entry for `hstl-oss task update <id> --estimate <size>`.
> There is no standalone `task estimate` subcommand (not in the manifest); the analysis
> steps below guide the user (or AI) in choosing a size.
>
> Example: `/hstl-oss:task:estimate T01 M`

1. Inspect Task info

```bash
hstl-oss task get <task-id> | jq '{id,title,type,estimate,sprint}'
```

2. Analyze complexity

Evaluate complexity considering:

### Technical complexity
- Number of affected code areas
- Whether new technology/patterns must be learned
- Integration complexity with existing code

### Uncertainty
- Requirement clarity
- Uncertainty of the technical approach
- External dependencies

### Risk factors
- Impact on existing functionality
- Testing complexity
- Rollback ease

3. Update the Task estimate (must go through the CLI — do not edit frontmatter directly)

```bash
hstl-oss task update <task-id> --estimate <XS|S|M|L|XL>
```

The CLI atomically updates frontmatter and the DB. Editing the Task file with Write/Edit
causes DB drift (CLAUDE.md operations rule).

## Examples by size

### XS Tasks
- Config value change
- Text/typo fix
- Simple bug fix (clear cause, 1-line change)
- Copy-paste of an existing pattern

### S Tasks
- Simple CRUD endpoint addition
- Add a field to an existing component
- Add simple validation
- Add N unit tests

### M Tasks
- New API endpoint (medium complexity)
- Add a UI component
- Refactor existing logic
- Bug fix that requires investigation

### L Tasks (splitting recommended)
- New service implementation
- Multi-component integration
- Performance optimization
- Complex business logic
- Large refactor

### XL Tasks (splitting required)
- Architecture change
- Database migration
- New infrastructure setup
- New external service integration

## Output Format

```markdown
## Task Estimation

### Task Information
| Item | Value |
|------|-----|
| ID | TNN |
| Title | {title} |
| Category | {feature/bugfix/refactor/...} |

### Estimation
| Item | Value |
|------|-----|
| Size | {XS|S|M|L|XL} |
| Expected duration | {time range} |

### Complexity Factors
1. {factor_1}
2. {factor_2}
3. {factor_3}

### Split recommendation (when L/XL)
| Sub-task | Size |
|----------|--------|
| {sub_1} | {S|M} |
| {sub_2} | {S|M} |

### Notes
{Additional considerations}
```

## Re-estimation

Re-estimate when:

1. **Requirements change** — scope significantly altered
2. **Technical discovery** — unexpected complexity uncovered
3. **Dependency change** — new dependencies added/removed
4. **Day-plus overrun** — work in progress takes longer than expected

## Integration

- `/task:create` — set estimate at Task creation
- `/task:start` — verify estimate when starting a Task
- `/task:next` — recommend the next Task by size
- `/session:start` — show Sprint progress
