# Knowledge Card writing guide

The detailed card-writing guide referenced by the `/learned` skill.

## Card structure

```
### {Title} ({YYYY-MM-DD}, {task-id})
**Problem**: the issue that occurred or the fact discovered
**Cause**: the root cause
**Solution**: the fix that was applied
**Discovery context**: which task / situation surfaced this
**Related code**: path/to/file.py
```

### Field-writing guide

| Field | Required | Guidance |
|-------|----------|----------|
| Title | Y | Include searchable keywords. State concrete API/component names |
| Date | Y | YYYY-MM-DD format. The date the lesson was discovered |
| Task ID | Y | Related task ID. "ad-hoc" if unrelated to a task |
| Problem | Y | 1–2 sentences. "When ~ happened, ~ resulted" or "Discovered that ~" |
| Cause | Y | Root cause. If unknown, "Cause unknown, presumed ~" |
| Solution | Y | Applied fix or conclusion reached. If unresolved: "Unresolved -- {current state}" |
| Discovery context | optional | Can be omitted, but include when context matters |
| Related code | optional | Strongly recommended for API/mistakes categories |

### Category classification details

#### api/ — external API integration lessons

Targets: gotchas discovered while integrating with external APIs, undocumented behavior, rate limits, auth issues.

File partitioning: separate by API/service (e.g. `api/service-a.md`, `api/service-b.md`)

Keywords: token, auth, rate limit, subscription, session, response format, error code

#### architecture/ — design decisions

Targets: answers to "why was it designed this way?".

File: `decisions.md` (numbered D01~D99) or split by topic

Keywords: protocol, queue, trade-off, abstraction, pattern choice

#### domain/ — domain knowledge

Targets: domain knowledge you can't infer from code alone, business rules, tuning rationale.

File partitioning: by topic (e.g. `domain/business-rules.md`, `domain/tuning.md`)

Keywords: threshold, business rule, domain constraint

#### operations/ — operations lessons

Targets: lessons from deployment, monitoring, incident response, infrastructure.

File: by topic (e.g. `operations/deploy.md`, `operations/monitoring.md`)

Keywords: deploy, incident, restart, log, monitoring, Docker

#### mistakes/ — prevention

Targets: mistakes that took 30+ minutes to debug, repeated mistakes, situations risking data loss.

File: `never-again.md`

Keywords: mistake, repetition, time waste, data loss, rollback

## Card-writing principles

1. **Fact-based**: record measurements/experiment results, not guesses
2. **Reproducible**: written so others (or future-you) can apply it in the same situation
3. **Concise**: one core insight per card. If there are several lessons, split into multiple cards
4. **Date required**: behavior may change over time
5. **Code links**: always include the relevant source file path

## Card quality checklist

- [ ] Is the title concrete and searchable?
- [ ] Are the date and task ID present?
- [ ] Are problem / cause / solution each 1–2 sentences?
- [ ] Can someone else read it and understand the context?
- [ ] Do the related-code paths match the current codebase?
- [ ] Does it not duplicate an existing card?
