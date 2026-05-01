# Design Audit — Detailed Per-Phase Inspection Items

> Detailed inspection items for each phase of the doc-cross-check skill.
> SKILL.md has only a one-line summary per phase; consult this file when you need the detailed checklist.

---

## Phase 1: Feature ID Integrity

**Goal**: guarantee Feature ID uniqueness and the mathematical accuracy of catalog totals.

### Checks

1. **ID uniqueness**: Feature IDs are not duplicated across Contexts.
   - How: collect every `{CTX}-F\d{3}` pattern and detect duplicates
   - Example error: `DCL-F001` exists in two Contexts

2. **Declared count match**: each Context's "Feature count: N" declaration == actual rows listed
   - How: parse the header declaration and compare against the row count

3. **Overall total integrity**: the overview document's "N Features total" == sum across Contexts
   - How: sum per-Context Feature counts and compare against the declared total

4. **Sequence continuity**: check for gaps in the Feature ID sequence
   - Example error: F001, F002, F004 → F003 is missing
   - However, deleted Features should not be reused, so a gap is a warning (MEDIUM)

5. **ID format compliance**: passes the regex `^[A-Z]{3}-F\d{3}$`
   - Example errors: `DC-F01`, `DCL-FTask`

### Severity

| Finding | Severity |
|---------|----------|
| ID duplicate | CRITICAL |
| 3+ total mismatches | HIGH |
| Sequence gap | MEDIUM |
| ID format mismatch | HIGH |

---

## Phase 2: Feature ↔ BC Mapping

**Goal**: each Feature belongs to exactly one BC, and the BC spec and catalog reference each other consistently.

### Checks

1. **Feature → BC link**: each Feature in the catalog has a "design-doc" link, and the linked file actually exists.
   - How: verify path existence with Glob

2. **BC spec → catalog back-reference**: Feature IDs listed in BC spec docs exist in the feature catalog.
   - Example error: BC spec lists `DCL-F099` but the catalog does not contain it

3. **Feature membership uniqueness**: the same Feature is not mapped to two or more BCs.
   - How: collect Feature IDs across all BC specs and detect duplicates

4. **Aggregate → Feature mapping**: the Feature mapping table in Aggregate docs matches the catalog.

### Severity

| Finding | Severity |
|---------|----------|
| Broken link (file missing) | HIGH |
| Feature ID not in BC spec | HIGH |
| Feature double membership | CRITICAL |

---

## Phase 3: Feature ↔ FR Mapping

**Goal**: every Feature has at least one functional requirement, and conversely no requirement references a ghost Feature.

### Checks

1. **Feature → FR existence**: each Feature has at least one corresponding FR.
   - A Feature without an FR cannot be traced back to requirements -> design has no rationale

2. **FR → Feature back-reference validity**: Feature IDs referenced from FR docs exist in the catalog.
   - Example error: FR references `DCL-F099` but it is not in the catalog

3. **Traceability coverage calculation**: (Features with an FR) / (total Features) × 100
   - Target: ≥ 70%
   - < 60%: HIGH

4. **FR ID format**: complies with the regex `FR-{CTX}-\d{3}`

### Severity

| Finding | Severity |
|---------|----------|
| Ghost FR reference | HIGH |
| Traceability < 50% | HIGH |
| Traceability 50-70% | MEDIUM |
| FR format mismatch | LOW |

---

## Phase 4: Event Integrity

**Goal**: every event in the catalog has valid publishers/subscribers and corresponds 1:1 with the Dapr topics table.

### Checks

1. **Event ID uniqueness**: no duplicate IDs in the event catalog

2. **Publisher BC exists**: the BC named in an event's "publishing BC" field is an actual BC

3. **Subscriber BC exists**: the BC named in an event's "subscribing BC" field is an actual BC

4. **Detect events with no publisher**: an event with no publishing BC is a ghost event (CRITICAL)

5. **Detect events with no subscriber**: an event with no subscribing BC is a dead letter (HIGH)
   - Note: events published by external systems may legitimately have no subscriber → flag for review

6. **Dapr topic mapping**: each topic in the Dapr topics table corresponds 1:1 with the event catalog
   - Topic missing from event catalog = missing
   - Event missing from the topics table = unregistered

### Severity

| Finding | Severity |
|---------|----------|
| Event with no publisher | CRITICAL |
| Event with no subscriber | HIGH |
| Dapr topic mismatch | HIGH |
| Event ID duplicate | CRITICAL |

---

## Phase 5: Terminology Consistency

**Goal**: detect cases where the same concept is expressed by different terms, causing communication confusion.

### Checks

1. **Synonym pair detection**: search for predefined synonym pairs to detect mixed usage
   - Examples: `OrderSide` vs `TradeDirection`, `Processing` vs `Strategy`
   - Cross-reference each project's domain glossary (`domain-glossary.md`)

2. **Enum value consistency**: detect the same enum being defined with different values across documents
   - Example: `OrderStatus: PENDING / FILLED` vs `OrderStatus: OPEN / EXECUTED`

3. **Namespace mixing**: consistency across BC names, module names, and event namespaces
   - Example: `Contracts.Events` vs `Contracts.IntegrationEvents`

4. **BC-name mixing**: cases where the same BC is referred to by different names across documents
   - Example: `DataCollection` vs `Collection` vs `Collector`

### Detection method

- Grep suspect term pairs in parallel
- Reference the domain glossary first if one exists
- Otherwise, manually extract frequently recurring terms across the documentation and present them as candidates

### Severity

| Finding | Severity |
|---------|----------|
| BC-name mixing | HIGH |
| Event namespace mixing | HIGH |
| Enum value mismatch | CRITICAL |
| Synonym mixing | MEDIUM |

---

## Phase 6: Numerical Integrity

**Goal**: numbers in overview documents match the actual aggregates measured from detail documents.

### Checks

1. **Feature count integrity**: PDD/Overview "N Features" == actual catalog row count

2. **FR count integrity**: overview "N FRs" == actual item count in FR docs

3. **SLO count integrity**: overview "N SLOs" == actual item count in SLO docs

4. **Event count integrity**: overview "N events" == actual rows in the event catalog

5. **BC count integrity**: overview "N BCs" == BC spec file count / BC list item count

6. **Module count integrity**: architecture doc "N modules" == module count described in the solution structure

7. **Aggregate-table sum verification**: a "total" row in a table equals the sum of the individual rows

### Severity

| Finding | Severity |
|---------|----------|
| 5+ mismatches | HIGH |
| 1-4 mismatches | MEDIUM |
| Total-row error | MEDIUM |

---

## Phase 7: Link Health

**Goal**: relative-path links inside documents resolve to actual files.

### Checks

1. **Markdown link validity**: relative paths in `[text](path)` resolve to real files
   - How: extract link patterns → verify existence relative to the base path

2. **Legacy path detection**: references to old project names or old directory structure
   - Example: old paths still in docs after a migration

3. **Image / diagram links**: images in the form `![](path)` actually exist

4. **Cyclic-reference detection** (optional): A → B → A loops

### Severity

| Finding | Severity |
|---------|----------|
| Broken link in core design doc | HIGH |
| Broken link in supporting doc | MEDIUM |
| Legacy-path reference | LOW |

---

## Severity Aggregation in the Full Report

| Severity | Meaning | Recommended handling |
|----------|---------|---------------------|
| CRITICAL | Design integrity broken | Fix immediately and re-run audit |
| HIGH | Major traceability or integrity error | Fix within the current sprint |
| MEDIUM | Warning level, no short-term impact | Address in the next sprint |
| LOW | Quality-improvement suggestion | Add to backlog |

---

## Related

- audit automation script: `scripts/check-design-consistency.sh`
- doc-cross-check run modes: see the "Run Modes" section in SKILL.md
