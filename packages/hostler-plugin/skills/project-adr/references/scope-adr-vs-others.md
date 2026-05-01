# ADR vs Other Document Types — Decision Tree

The **scope boundary** between ADRs and RFCs / Design Docs / KB cards / Task bodies / runbooks.

## Decision Tree

```
Q: Where should this content go?
│
├─ "Gathering opinion / discussion" is the main goal → RFC
│   └─ The audience to consult is clear → Architecture Advice Process
│
├─ "Recording an already-made decision" is the main goal
│   │
│   ├─ Architectural level (module boundaries · core tech · quality attributes) → ADR
│   │   ├─ Multi-alternative comparison matters → ADR with MADR 4.0
│   │   ├─ Just record the decision concisely → ADR with Nygard
│   │   └─ 1-sentence draft → ADR with Y-statement
│   │
│   ├─ Implementation choice scoped to 1 Task → Task / Ticket body
│   └─ Iteration-internal policy → iteration goal / approach
│
├─ "Implementation plan / how-to" is the main goal → Design Doc / TDR
│   └─ Storage: docs/design/ (convention)
│
├─ "Lessons / gotchas / insights" → Knowledge base / learning card
│   └─ Use the project's KB tooling
│
├─ "Repeated execution procedure" → Runbook
│   └─ docs/runbooks/ (convention)
│
└─ "Time-limited state · ad-hoc" → commit message / branch description
```

## Comparison Matrix

| Document type | Purpose | Tense | Scope | Immutability | Storage location (convention) |
|----------|------|-----|------|--------|----------------|
| **ADR** | **Record** of a decision | past / present | architectural | ★★★ immutable after accepted | `docs/architecture/` or `docs/adr/` |
| **RFC** | **Solicit** opinions | future / conditional | feature / improvement proposal | ★ may change during discussion | `docs/rfcs/` |
| **Design Doc / TDR** | Implementation **plan** | future | implementation detail | ★★ updated during implementation | `docs/design/` |
| **KB card** | Lessons · gotchas | past | personal / team learning | ★★ updated when context changes | `docs/knowledge/` or team wiki |
| **Task body** | Single task | present | scope of 1 Task | ★★ immutable after completed | issue tracker / `tasks/` |
| **Runbook** | Operational procedure | imperative | repeated execution | ★ continuously updated | `docs/runbooks/` |
| **Standard doc** | Team agreement convention | present | company / project | ★★ updated only after formal review | `docs/standards/` |

## Workflow Relationships

```
 RFC (proposal · discussion · feedback collection)
        │
        ▼
 Decision Meeting / Advice Process
        │
        ▼
   ADR (decision record, immutable)
        │
        ▼
 Design Doc / TDR (implementation plan)
        │
        ▼
   Task (1-commit execution)
        │
        ▼
   Result section + commit log
        │
        ▼
   KB card (lesson · gotcha)
```

## Diagnostic Questions (quick self-check)

### ADR vs RFC

| Question | ADR | RFC |
|------|-----|-----|
| Already decided? | ✅ | ❌ |
| Need to gather opinions? | ❌ | ✅ |
| Tense? | past / present | future / conditional |
| Editable? | immutable after accepted | editable during discussion |

**RFC → ADR transition**: After an RFC passes the Advice Process and the **Decision** is finalized, write the ADR.
The ADR references the RFC via `relates_to: ["RFC-NN"]`.

### ADR vs Design Doc

| Question | ADR | Design Doc |
|------|-----|-----------|
| What does it record? | WHY (decision · rationale) | HOW (implementation method) |
| Level? | architectural boundary | implementation detail (files / functions / APIs) |
| Change? | supersede with a new ADR | edit as implementation progresses |

**Examples**:
- ADR: "Adopt monolith → modular monolith + gradual decomposition" (ADR-002)
- Design Doc: "TypeScript signatures for the Module A ↔ Module B interface"

### ADR vs KB Card

| Question | ADR | KB card |
|------|-----|--------|
| What does it record? | decision | facts · lessons · gotchas |
| Audience? | project | individual + team-shared |
| Format? | Context/Decision/Consequences | category-based, free form |
| Tooling? | git + manual | project KB tool / team wiki |

**Examples**:
- ADR: "Adopt UUIDv7 as the machine identifier" (ADR-019)
- KB: "Watch out for the time_high offset when parsing UUIDv7 — Go uuid library vs JavaScript compatibility gotcha"

### ADR vs Task Body

| Question | ADR | Task body |
|------|-----|---------|
| Level? | architectural | 1-commit work |
| Immutable? | after accepted | after completed |
| Impact? | long-term | ends after Task completes |

**Note**: A **broad-impact decision** found mid-Task is outside the Task scope. In this case, pause the Task, propose an ADR, and resume the Task after approval.

**Examples**:
- Task: "Logger refactor ticket" → Task body records "introduce structured logging"
- ADR: "Logging strategy change discovered mid-Task (standardize on structured logging)" → a separate ADR-NNN

### ADR vs Standard Document

| Question | ADR | Standard doc |
|------|-----|---------|
| What does it record? | trade-offs + rationale at decision time | team-wide convention |
| Change? | supersede with a new ADR | revise via formal review |
| Example | "Adopt UUIDv7" | "All IDs are UUIDv7. Exception policy: ..." |

**Integration scenario**: A convention finalized by an ADR is connected to the standard doc via a **summary + ADR number link**. Detailed trade-offs go in the ADR; day-to-day reference goes in the standard doc.

## Anti-patterns — Wrong Location Choice

### A. ADR-grade decision buried in a Task body

**Symptom**: A 1-line decision like "change the project-wide logging strategy" is buried in the Task body.
**Response**: Remove the decision from the Task → propose a separate ADR → resume the Task after the ADR is accepted.

### B. The "ADR" is actually a runbook

**Symptom**: ADR-NNN contains "1. git clone 2. npm install 3. ..." procedure.
**Response**: Move it to a runbook (e.g. `docs/runbooks/`). Keep only "why we adopted this deployment style" in the ADR.

### C. A KB card written as an ADR

**Symptom**: "How to debug a goroutine leak" is filed as an ADR.
**Response**: Move to a KB / learning card. ADRs are for decisions.

### D. Authored as an ADR while still at the RFC stage

**Symptom**: An idea still under discussion is published as ADR-NNN proposed → sits in proposed for a month.
**Response**: Move to RFC. Promote to ADR after Advice gathering completes.

## Composite Scenarios (generic examples)

### Scenario 1: Designing a New Feature

```
1. Issue a Task → "Design feature X"
2. RFC draft → docs/rfcs/RFC-NN-feature-X.md
3. Advice Process → collect team opinions
4. Decision finalized → ADR-NNN proposed
5. Decider approves → ADR-NNN accepted
6. Design Doc → docs/design/feature-X-design.md
7. Register in feature catalog (when present)
8. Plan an implementation iteration → task breakdown
```

### Scenario 2: Legacy Refactor

```
1. Refactor PoC Task → record results as KB cards / benchmark report
2. ADR-NNN proposed based on the PoC (supersedes ADR-MMM)
3. Approved → ADR-MMM status → superseded
4. Plan a refactor iteration → task breakdown
5. Iteration retrospective after completion
```
