# Anti-Patterns & Fallacies — 11 + 7

Detailed appendix to SKILL.md §Gate 3. Use as a full check on a drafted ADR before submission.

## 11 ADR Content Anti-Patterns (Zimmermann, 2023)

### Subjectivity family (4)

#### A1. Fairy Tale

**Symptom**: "We adopted a load balancer because it distributes load" — only upsides, no downsides.

**Detection**: Consequences section lacks keywords like "risk / drawback / trade-off / negative".

**Fix**: Add at least 1 negative consequence. Force MADR's "Bad, because ..." item.

**Common occurrence**: Easy to slip into "strategic vision" / "signature feature" ADRs. The more visionary the ADR,
the more important it is to spell out negatives like "increased operational overhead / debugging complexity /
ecosystem maturity dependency".

#### A2. Sales Pitch

**Symptom**: "This pattern is **innovative**, **massively scalable**, and **used by industry leaders**..." — marketing exaggeration.

**Detection**: grep for words like "innovative / world-class / game-changer / overwhelming / strongest".

**Fix**: Replace with quantitative benchmarks. "Netflix uses it" alone is insufficient evidence — use specifics like "Netflix tech blog 2024, 50% reduction in request handling" with sources.

#### A3. Free Lunch Coupon

**Symptom**: Lists only short-term gains; ignores long-term maintenance / operations / technical debt.

**Detection**: Consequences lacks "long-term / operations / maintenance / operational" perspectives.

**Fix**: Make the time dimension explicit — separate consequences at 3 months / 1 year / 3 years.

#### A4. Dummy Alternative

**Symptom**: The compared alternatives are obviously inferior — e.g. "A vs do-nothing (zero option)".

**Detection**: Considered Options has 1 actually-considered alternative + intentionally weakened ones.

**Fix**: At least 2 real competing candidates — versions / configurations / what competitors use.

### Time & Context family (3)

#### A5. Sprint (myopia)

**Symptom**: Considers only the effect after 2–3 Sprints. Ignores 1+ year long-term impact.

**Detection**: Consequences contains only short-term phrasing like "after 3 sprints / after Q2 / short-term".

**Fix**: Describe expected outcomes at "+3 months / +1 year / +3 years" milestones.

#### A6. Tunnel Vision

**Symptom**: Developer perspective only. Ops / maintenance / security / QA perspectives are absent.

**Detection**: Decision Drivers / Consequences mention none of these roles:
- Operations engineer (SRE / devops)
- QA / tester
- Security
- Newcomer (onboarding cost)

**Fix**: Specify at least 3 role perspectives. Recommended: use Perspective Mining.

#### A7. Maze

**Symptom**: Title says "Decide session storage strategy" but the body drifts into a cache-strategy discussion.

**Detection**: Compare the title's core terms against the keyword distribution in the body. Suspect when title keywords are <10% of the body.

**Fix**: Rename the title or re-scope the body. If the scope splits into two, split the ADR into two.

### Size & Nature family (3)

#### A8. Blueprint or Policy in Disguise

**Symptom**: Cookbook style like "Implement in N steps: 1. Install X 2. Run Y 3. ...".

**Detection**: Excessive numbered lists + imperative tone ("must / shall").

**Fix**: Move it to a runbook (e.g. `docs/runbooks/`). Keep only "what was decided + why" in the ADR.

#### A9. Mega-ADR

**Symptom**: A single ADR contains component design + 5 diagrams + 200 lines of code + a test plan, exceeding 400 lines.

**Detection**:
```bash
[ $(wc -l < <ADR.md>) -gt 400 ] && echo "WARN: Mega-ADR suspected"
[ $(grep -c '```' <ADR.md>) -gt 6 ] && echo "WARN: more than 3 code-block pairs"
```

**Fix**: Split into architecture view documents (e.g. `docs/architecture/<layer>-view.md`). Keep **what + why** in the ADR.

**Common occurrence**: Language / tech stack selection ADRs and governance / identity policy ADRs often balloon past 500 lines. Split the body into view documents and shrink the ADR to a summary.

#### A10. Novel / Epic

**Symptom**: Tries to compress the entire SAD (Software Architecture Document) into a single ADR.

**Detection**: More than 15 sections. Multiple architecture decisions packed into one document.

**Fix**: Split each decision into its own ADR. Link via `relates_to`.

### Deception family (1)

#### A11. Magic Tricks

**Symptom**:
- Fake urgency: "If we don't decide now, production will fall over"
- Pseudo-quantitative scoring: "Option A: 8.7 / Option B: 6.2" with no rationale
- Problem–solution mismatch: the solution does not actually address the stated problem

**Detection**: Numeric scores present but no scoring rubric. Excessive "urgent / immediate / now" keywords.

**Fix**: When using scores, specify the **rubric** (e.g. weighted 1–5 score per driver).

## 7 Decision-Making Fallacies (Zimmermann, 2025)

### F1. Blind Flight

**Symptom**: Picks a solution without analyzing requirements / NFRs.

**Countermeasure**: Agree on an "Operating range". Define measurable NFRs first.

### F2. Following the Crowd

**Symptom**: "Netflix/Google/Meta use it, so we will too" (argumentum ad populum).

**Countermeasure**: Validate compatibility with our own requirements first. Record the validation results in Decision Drivers.

### F3. Anecdotal Evidence

**Symptom**: "We succeeded with A at the previous company, so A" — single-case rationale.

**Countermeasure**: Use SMART NFRs as justification criteria. State the confidence level.

### F4. Blending Whole and Part

**Symptom**: "I dislike a part of the microservice pattern, so I'll throw out microservices entirely" — whole/part confusion.

**Countermeasure**: Divide and conquer. Allow selective adoption of style elements.

### F5. Abstraction Aversion

**Symptom**: Bundles a conceptual decision with a tech choice into one decision. Example: "We use a DB = we use Postgres" (level confusion).

**Countermeasure**: Two-step ADRs — a high-level concept ADR + a low-level tech-choice ADR. Connect via `relates_to`.

### F6. Golden Hammer

**Symptom**: "Microservices for everything", "Event-driven for everything" — one-size-fits-all.

**Countermeasure**: Expand the toolbox. Learn new practices. Always have 3+ alternatives in scope.

### F7. Time Dimension Ignored

**Symptom**: A 5-year-old decision is reused as-is. Tech evolution is unaddressed.

**Countermeasure**: Set a **review due date** on the ADR. A field like "Re-review 2027-Q2".

### F8 (Bonus). AI Over-confidence

**Symptom**: An AI-generated ADR is adopted as-is with no quality assurance.

**Countermeasure**: **Pass all of Gates 1–3 in this skill**. The AI does not finalize the Decision sentence (user review is mandatory).

## Generic Violation Examples and Lessons

### Case 1 — Duplicate ADR for the same topic (related to A7 Maze + A4)

**Phenomenon**: Two ADRs share the same kebab-case title (e.g. ADR-XX / ADR-YY both `cli-output-philosophy`). Two ADRs exist for the same topic.

**Common causes**:
- During work, "expanding an existing ADR's content significantly while assigning a new number"
- The correct flow (mark the old ADR superseded + write a new one) is skipped

**Fix**: Update the old ADR with `superseded by ADR-YY`, and add `supersedes: [ADR-XX]` to the new ADR. If the titles are identical, rewrite the title to differentiate them.

**Recurrence prevention**:
```bash
# Run before drafting a new ADR
ADR_DIR=${ADR_DIR:-docs/architecture}
NEW_TITLE="my-new-topic-kebab-case"
grep -l "$NEW_TITLE" "$ADR_DIR"/ADR-*.md && echo "WARN: duplicate topic — review supersede"
```

### Case 2 — Meta-ADR (When-to-ADR Q4 violation)

**Phenomenon**: An ADR like `ADR-XXX-acceptance-of-ADR-YYY.md` whose only purpose is to accept another ADR.

**Cause**: Splitting "fill in the original ADR's accepted_date" into a separate ADR.

**Fix**: Add `accepted_date` to the original ADR's frontmatter. Mark the meta-ADR as deprecated with the reason recorded. **Never delete** (preserve history).

### Case 3 — Number gap (Lifecycle convention violation)

**Phenomenon**: Gaps in ADR numbering (e.g. ADR-N then ADR-N+2). Unclear whether retired or renumbered.

**Cause**: On retirement, only the number is deleted without leaving a rejected file.

**Fix**: Gaps are **kept permanently** (no reuse). Preserve retired ADRs as files like `ADR-NNN-<reason>-rejected.md` in the rejected state.

**Recurrence prevention**: Update the Index (README) table immediately on number issuance, and create a rejected file on retirement.

## Automated Validation Script (bash)

```bash
#!/bin/bash
# Gate 3 Anti-Pattern Check for <ADR.md>
FILE=$1
[ -z "$FILE" ] && { echo "Usage: $0 <ADR.md>"; exit 1; }

echo "=== $FILE Anti-Pattern Check ==="

# A1 Fairy Tale
grep -A 30 "^## Consequences\|^### Consequences" "$FILE" 2>/dev/null | \
  grep -iqE "risk|drawback|trade-off|negative|bad, because" || \
  echo "  [WARN] A1 Fairy Tale — missing negative consequences"

# A8 Blueprint
grep -cE "^[0-9]+\." "$FILE" | awk '{ if($1>10) print "  [WARN] A8 Blueprint — "$1" numbered list items" }'

# A9 Mega-ADR — 400 lines / 3 code-block pairs
LC=$(wc -l < "$FILE")
CB=$(grep -c '```' "$FILE")
[ "$LC" -gt 400 ] && echo "  [WARN] A9 Mega-ADR — $LC lines (>400)"
[ "$CB" -gt 6 ] && echo "  [WARN] A9 Mega-ADR — $((CB/2)) code-block pairs (>3)"

# A2 Sales Pitch
grep -ciE "innovative|game-changer|world-class|industry-leading|overwhelming|perfect" "$FILE" | \
  awk '{ if($1>0) print "  [WARN] A2 Sales Pitch — "$1" marketing terms" }'

# Frontmatter required fields
for field in id title status date deciders; do
  grep -q "^$field:" "$FILE" || echo "  [BLOCK] frontmatter '$field' field missing"
done

echo "=== done ==="
```
