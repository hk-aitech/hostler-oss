# Detailed problem-solving workflow guide

> Workflow grounded in ITIL and SRE best practices

---

## Table of contents

1. [Workflow overview](#workflow-overview)
2. [Step 1: Identify](#step-1-identify)
3. [Step 2: Classify](#step-2-classify)
4. [Step 3: Diagnose](#step-3-diagnose)
5. [Step 4: Resolve](#step-4-resolve)
6. [Step 5: Close](#step-5-close)
7. [Escalation guide](#escalation-guide)
8. [Communication templates](#communication-templates)

---

## Workflow overview

### End-to-end flow

```
┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐    ┌────────────┐
│  Identify  │───▶│  Classify  │───▶│  Diagnose  │───▶│   Resolve  │───▶│   Close    │
└────────────┘    └────────────┘    └────────────┘    └────────────┘    └────────────┘
      │                 │                 │                 │                 │
      ▼                 ▼                 ▼                 ▼                 ▼
   Symptom           Severity        Cause analysis    Apply fix          Document
   Impact            Priority        Validate           Verify            Retro
   Reproduction      Escalation      Log analysis      Rollback prep      Share knowledge
```

### Goals per step

| Step | Goal | Output |
|------|------|--------|
| Identify | Capture the problem precisely | Problem write-up |
| Classify | Decide severity / priority | Triaged issue |
| Diagnose | Identify root cause | Cause-analysis report |
| Resolve | Solve and verify | Resolved issue |
| Close | Document and share | Docs, knowledge base |

---

## Step 1: Identify

### Goal

Capture the problem precisely and objectively.

### Information to collect

#### Mandatory

| Item | Question | Example |
|------|----------|---------|
| **Symptom** | What went wrong? | "API call returns 401 Unauthorized" |
| **Expected behavior** | What should happen? | "API returns 200 OK with data" |
| **Actual behavior** | What actually happens? | "401 with empty response" |
| **Impact** | How wide is the impact? | "All /cicd endpoints" |
| **Reproduction steps** | How do you reproduce? | "1. Sign in 2. Visit /cicd" |

#### Environment info

| Item | How to collect |
|------|----------------|
| OS/version | `uname -a`, `cat /etc/os-release` |
| Runtime version | `node --version`, `python --version` |
| Dependency versions | `package.json`, `requirements.txt` |
| Config | Relevant config files |

### Reproduction test

```markdown
## Reproduction steps

1. Environment prep
   - Install Node.js 22
   - Run yarn install

2. Start the services
   - Run yarn dev
   - Confirm ports 3000, 7007

3. Reproduce the problem
   - Open http://localhost:3000/cicd in a browser
   - Confirm the error message

## Reproduction conditions
- [ ] Always reproduces
- [ ] Intermittent
- [ ] Only under specific conditions
```

---

## Step 2: Classify

### Severity criteria

#### Severity Level

| Level | Name | Definition | Example |
|-------|------|------------|---------|
| **S1** | Critical | Whole-service outage, data loss risk | Server down, DB corrupted |
| **S2** | High | Major feature unusable, no workaround | Login broken, payment failing |
| **S3** | Medium | Some features affected, workaround exists | Specific page loads slowly |
| **S4** | Low | Minor inconvenience, no functional impact | UI typo, small style issue |

#### Priority mapping

| Severity | Wide impact | Narrow impact |
|----------|-------------|---------------|
| Critical | P0 (immediate) | P0 (immediate) |
| High | P0 (immediate) | P1 (same day) |
| Medium | P1 (same day) | P2 (this Sprint) |
| Low | P2 (this Sprint) | P3 (backlog) |

### Response-time criteria

| Priority | Response start | Resolution target |
|----------|----------------|-------------------|
| P0 | Immediate | Within 4 hours |
| P1 | Within 1 hour | Within 24 hours |
| P2 | Within 24 hours | This Sprint |
| P3 | Planned time | Next Sprint |

---

## Step 3: Diagnose

### Diagnosis process

```
Gather info ──▶ Form hypotheses ──▶ Validate ──▶ Confirm cause
    │                  │                │                │
    ▼                  ▼                ▼                ▼
 Logs              Likelihood     Repro / isolate     Root cause
 Change history    Evidence-based A/B test            Document
 Monitoring        Elimination     Debugging
```

### Information-gathering checklist

**Log analysis:**
- [ ] Application logs
- [ ] System logs
- [ ] Access logs
- [ ] Error logs

**Change history:**
- [ ] Recent commits (`git log --oneline -10`)
- [ ] Recent deployments
- [ ] Configuration changes
- [ ] Dependency updates

**System state:**
- [ ] Resource usage (CPU, memory, disk)
- [ ] Network state
- [ ] Process state

### Hypothesis-formation techniques

#### 5 Whys

Ask "why?" repeatedly to surface the root cause:

```
Problem: page loads take more than 10 seconds

Why 1: Why slow?
→ API responses are slow

Why 2: Why slow API?
→ DB queries take a long time

Why 3: Why slow queries?
→ Index missing

Why 4: Why no index?
→ Migration script missed it

Why 5: Why missed?
→ Code review didn't have a DB-change checklist

Root cause: missing review process for DB migrations
```

#### Binary search

Halve the problem space at every step:

```
Whole system
    │
    ├── Frontend issue? ──▶ No
    │
    └── Backend issue? ──▶ Yes
            │
            ├── API layer? ──▶ No
            │
            └── Service layer? ──▶ Yes
                    │
                    └── Found CatalogClient auth issue
```

### Debugging tools

| Situation | Tool / command | Use |
|-----------|----------------|-----|
| Live logs | `tail -f log.txt` | Log monitoring |
| Process | `ps aux \| grep node` | Process check |
| Port | `lsof -i :7007` | Port usage |
| Network | `curl -v http://...` | API test |
| DB | `psql -c "SELECT ..."` | Run a query |

---

## Step 4: Resolve

### Resolution types

| Type | Description | When |
|------|-------------|------|
| **Workaround** | Temporary bypass | When you need fast recovery |
| **Fix** | Root-cause solution | When the cause is confirmed |
| **Rollback** | Restore previous state | When the change itself is the cause |

### Change-application process

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Test      │────▶│  Prepare    │────▶│   Apply     │
│   environment│     │             │     │             │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │                   │
       ▼                   ▼                   ▼
   Verify fix         Rollback plan       Monitor
   Side effects       Document change     Verify
```

### Rollback-plan template

```markdown
## Rollback plan

### Rollback triggers
- Same error recurs within 15 minutes of applying the fix
- A new Critical error appears
- Performance metrics degrade by 50% or more

### Rollback procedure
1. Confirm the current change commit: `git log -1`
2. Revert to previous version: `git revert {commit}`
3. Restart the service: `yarn dev`
4. Confirm normal operation: `./scripts/dev-check.sh --status`

### Rollback owner
- Primary: {name}
- Backup: {name}
```

### Verification checklist

- [ ] Original problem resolved?
- [ ] No new problems introduced?
- [ ] Related features still work?
- [ ] No performance impact?
- [ ] No new errors in logs?

---

## Step 5: Close

### Documentation decision flow

```
Resolution complete
      │
      ├─ Could it recur? ──▶ Yes ──▶ Write a topic guide
      │                              {topic}.md
      │
      └─ Was investigation complex? ──▶ Yes ──▶ Write a worklog
                                         worklogs/YYYY-MM-DD-{topic}.md
```

### Retro questions

| Question | Purpose |
|----------|---------|
| What did we learn? | Knowledge accrual |
| How do we prevent recurrence? | Recurrence prevention |
| Does the process need to change? | Process improvement |
| Were tools insufficient? | Tooling improvement |
| Was documentation insufficient? | Documentation improvement |

### Knowledge-base updates

Based on the resolution, update:

- [ ] Troubleshooting index.md
- [ ] Related guide documents
- [ ] Runbook (when applicable)
- [ ] Checklist (when applicable)

---

## Escalation guide

### Escalation criteria

| Condition | Action |
|-----------|--------|
| Critical severity | Escalate immediately |
| No progress within 30 minutes | Consider escalation |
| Beyond your expertise | Escalate to a specialist |
| Impact widening | Escalate immediately |

### Escalation information

What to include when escalating:

```markdown
## Escalation request

### Problem summary
{One-sentence description}

### Current state
- Severity: {S1-S4}
- Impact: {description}
- Started at: {YYYY-MM-DD HH:MM}

### Investigation
- Confirmed: {list}
- Tried: {list}
- Ruled out: {list}

### Current hypothesis
{Most likely cause}

### Request
{Specific help request}
```

---

## Communication templates

### Incident notification

```markdown
**[{severity}] Incident: {title}**

Impact: {scope}
Started: {time}
Status: investigating

Current action: {what you're doing}
Next update: {when}
```

### Progress update

```markdown
**[Update] {title}**

Status: {investigating / fixing / resolved}
Progress: {what's new}
Next step: {what's next}
ETA: {estimated time}
```

### Resolution notification

```markdown
**[Resolved] {title}**

Cause: {root cause}
Fix: {applied solution}
Impact: {scope}
Duration: {start} ~ {end} ({total})

Prevention: {actions}
Docs: {link}
```

---

## Checklist summary

### Immediately after the incident

- [ ] Capture symptoms (screenshots, error messages)
- [ ] Capture reproduction steps
- [ ] Classify severity / priority
- [ ] Escalate when needed

### During investigation

- [ ] Inspect logs
- [ ] Inspect recent changes
- [ ] Maintain a hypothesis list
- [ ] Record progress

### When resolving

- [ ] Verify in test environment
- [ ] Prepare rollback plan
- [ ] Apply the change
- [ ] Verify and monitor

### When closing

- [ ] Document (guide or worklog)
- [ ] Retro
- [ ] Update knowledge base
