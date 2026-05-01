# Context Engineering reference

> "Find the **smallest high-signal token set** that maximizes the chance of the desired outcome."
> — Anthropic, Context Engineering Guide

---

## Core techniques

### 1. Context Pruning

Remove unnecessary information to improve token efficiency.

| Target | Action | Reason |
|--------|--------|--------|
| Old error messages | Remove | The problem was resolved |
| Duplicate code blocks | Replace with a reference | Avoid repetition |
| Completed Task detail | Keep a summary only | Preserve only the essential |
| Older session logs | Keep the most recent 5 | Maintain relevance |

**When to apply**:
- At session start
- When context utilization hits 50%
- On switching to a new topic

### 2. Compaction

**Run when context utilization hits 70%**:

```markdown
## Compaction procedure

1. Summarize current progress
   - Completed work
   - In-progress work
   - Remaining work

2. Extract key decisions
   - Architecture decisions
   - Tech choices
   - Changed requirements

3. Clarify next steps
   - Immediate to-dos
   - Required context

4. Run `/compact`
```

**Pre-compact checklist**:
- [ ] CURRENT-FOCUS.md updated
- [ ] Key decisions recorded
- [ ] Next steps clear
- [ ] Important code changes committed

### 3. Just-in-Time Loading

Don't pre-load. Load **only when needed**.

| Tool | Use | Example |
|------|-----|---------|
| `Read` | Specific file only | The file you'll edit |
| `Grep` | Related code only | Pattern search |
| `Glob` | File listing only | Structure exploration |

**Recommended pattern**:

```markdown
## Good (Just-in-Time)
1. Start the task
2. Read the file you need
3. Edit
4. Done

## Avoid (Pre-loading)
1. Read every related file
2. Start the task
3. Edit
4. Done
```

### 4. Structured notes (Scratchpad)

Use CURRENT-FOCUS.md as a working notebook:

```markdown
## Progress

### S#XX (YYYY-MM-DD)
- [x] Requirements analysis
- [x] Design complete
- [ ] Implementation in progress

### Decisions
- Use pattern A
- Pick library B

### Next steps
1. Finish implementation
2. Write tests
```

---

## Context-utilization monitoring

### Recommended thresholds

| Utilization | State | Action |
|-------------|-------|--------|
| 0–50% | Normal | Continue |
| 50–70% | Caution | Consider pruning |
| **70–85%** | **Warning** | **`/compact` recommended** |
| 85%+ | Critical | Compact immediately |

---

## Optimization strategies

### CURRENT-FOCUS.md optimization

**Problem**: as Sprint / Task info grows, the file balloons.

**Fix**:
1. Keep only the current Sprint / Task state
   - Reference `works/sprints/completed/` for completed Sprint detail
   - CURRENT-FOCUS.md should be only a roadmap + current state summary

2. Load on-demand
   - Load the current Sprint only
   - Treat completed Sprints as references

### Worklog optimization (works/worklogs/)

**Recommended**: load only the most recent 5 dated files.

**Additional optimization**:
- At session start, show only today's worklog summary
- Pull detail from per-date files when needed

### Using CURRENT-FOCUS.md

**Keep only the essentials**:
- Current Task metadata
- Checklist
- Progress (current session only)

---

## Long-session guide

### 2+ hour sessions

```markdown
## Time-based checkpoints

### After 1 hour
- [ ] Record progress
- [ ] Check context utilization

### After 2 hours
- [ ] Write an interim summary
- [ ] Consider committing
- [ ] Verify context is below 70%

### After 3 hours
- [ ] Run `/compact` recommended
- [ ] Consider splitting the session
```

### Complex-work guide

**Large refactors, integration tests, etc.**:

1. **Split the work**
   - Split into logical units
   - Commit per unit

2. **Manage context**
   - Prune at the end of each unit
   - Remove unnecessary code blocks

3. **Record progress**
   - Detailed notes in CURRENT-FOCUS.md
   - Notes for the next session

---

## References

- [Anthropic — Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
