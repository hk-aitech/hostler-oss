# Sprint Retrospective Checklist

Detailed guide to the retrospective check items referenced by the `/retro` skill.

## Data Collection Checklist

### Task file review
- [ ] Review notes/issues sections of every DONE task
- [ ] Review BLOCKED -> unblocked transition history
- [ ] Compare estimated vs actual time spent
- [ ] Identify ACs that required multiple attempts to satisfy

### Git history review
- [ ] Identify fix/revert/hotfix commits
- [ ] Look for repeated edits to the same file
- [ ] Spot large refactor commits (100+ lines changed)
- [ ] Find commit messages containing "fix", "revert", or "workaround"

### Worklog review
- [ ] Items in issue sections
- [ ] Plan changes
- [ ] External-dependency problems (API outages, library issues)

### Existing-doc review
- [ ] New entries in lessons-learned documents
- [ ] New entries in the decision log
- [ ] Change history of "current focus" / status documents

## Lesson Extraction Patterns

### Pattern 1: Bug -> fix cycle
**Signals**: fix commits, error-log mentions, added tests
**Question**: Why did this bug occur? How could it have been prevented?
**Category**: usually `api/` or `mistakes/`

### Pattern 2: API docs vs measured behavior mismatch
**Signals**: phrases like "the docs say...", "measured behavior", "in reality"
**Question**: Which part of the official docs is inaccurate?
**Category**: `api/`

### Pattern 3: Design decision and trade-off
**Signals**: "chose X over Y", "trade-off", entries in the decision log
**Question**: Why was this choice made? What alternatives were considered?
**Category**: `architecture/`

### Pattern 4: Performance / optimization discovery
**Signals**: benchmark numbers, "N per second", batch-size tuning
**Question**: What was the bottleneck? What is the optimal setting?
**Category**: `domain/` or `operations/`

### Pattern 5: Repeated mistakes
**Signals**: two or more similar fix commits on the same file
**Question**: What is the root cause and how can it be prevented systemically?
**Category**: `mistakes/`

### Pattern 6: Domain-rule discovery
**Signals**: business rules, domain constraints, traits of external systems
**Question**: Is this a domain rule that should be reflected in code?
**Category**: `domain/`

## Card Quality Criteria

A good retrospective card meets these criteria:
1. **Concrete**: includes specific API names / component names / numbers instead of vague phrasing
2. **Actionable**: clear about what to do next time the same situation arises
3. **Verified**: based on measurement / experiment, not guesswork
4. **Self-contained**: understandable without reading other cards
5. **Time-tagged**: traceable via date and task ID
