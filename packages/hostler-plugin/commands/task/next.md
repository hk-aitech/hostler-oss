---
description: Recommend the next Task by priority. Use when the user asks "what's next?", "recommend a task", or wants the next item to work on.
argument-hint: ""
allowed-tools: Bash(hstl-oss:*), Read, Glob
---

# Next Task Recommendation

## Instructions

1. Scan `works/sprints/active/` (Tasks with status todo or in-progress)
2. Sort by priority
3. Check dependencies
4. Provide a recommendation list

## Task Discovery

### Sprint folder structure

```
works/sprints/
├── active/          # Currently in-progress Sprint
│   └── sprint-NN/
│       └── tasks/
│           ├── T01-title.md   # status: in-progress
│           └── T02-title.md   # status: todo
├── backlog/         # Future Sprints
│   └── sprint-NN/
│       └── tasks/
└── completed/       # Completed Sprints
```

### Search command
```bash
# Search todo Tasks in the active Sprint (frontmatter status: todo)
find works/sprints/active -name "*.md" -path "*/tasks/*" -type f
```

## Priority Rules

| Priority | Description | Selection criterion |
|----------|------|-----------|
| **P0** | Critical | Handle immediately, blocker |
| **P1** | High | Current Sprint goal |
| **P2** | Medium | Candidate for the next Sprint |
| **P3** | Low | Keep in backlog |

## Selection Criteria

1. **Active Sprint first**: Tasks under `works/sprints/active/` first
2. **Priority**: P0 > P1 > P2 > P3
3. **Dependencies**: predecessor Task done (frontmatter `depends_on`)
4. **Complexity**: size that fits the remaining time

## Agent Mapping Preview

| Task type keywords | Recommended agent |
|-----------------|---------------|
| implementation, development, bug | developer |
| test, verification | qa-engineer |
| docs, README | tech-writer |
| Docker, CI/CD | devops |
| design, architecture | architect |

## Output Format

```markdown
## Next Task Recommendations

### Current Sprint: sprint-NN
**Sprint goal**: {sprint_goal}
**Remaining Tasks**: {remaining_count}

### Top Recommendations

#### 1. (recommended)
| Item | Value |
|------|-----|
| Title | {title} |
| Priority | P{N} |
| Location | works/sprints/active/sprint-NN/tasks/ |
| Recommended agent | {agent} |
| Expected complexity | {complexity} |

**Reason**: {reason}

#### 2.
| Item | Value |
|------|-----|
| Title | {title} |
| Priority | P{N} |
| Location | works/sprints/active/sprint-NN/tasks/ |
| Recommended agent | {agent} |

### Blocked Tasks
| Task | Blocker | Status |
|------|--------|------|
| T03 | T01 must complete | waiting |

### Sprint Progress
- Done: {completed}/{total} ({percent}%)
- Remaining estimate: {remaining_estimate}
```

## Quick Start

To start a Task:
```
/task:start
```

## Notes

- Tasks with dependencies are recommended only after predecessors are done (check `depends_on`)
- Tasks in the active Sprint are recommended first
- Backlog Sprints are recommended only after the active Sprint
