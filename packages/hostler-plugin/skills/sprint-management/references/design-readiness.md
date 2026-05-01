# Design-readiness check guide

Detailed criteria the AI applies to validate design readiness when a Sprint starts.

## Required documents per Task type

| Task type | Required docs | Validation points |
|-----------|---------------|-------------------|
| New feature implementation | Design doc (`docs/03-design/`) | Whether the Sprint-scope feature's design is documented |
| | FRS requirements (`docs/02-requirements/FRS.md`) | Whether the FR-xxx item for the feature is defined |
| API endpoint addition | API design doc | Endpoint spec, request/response format |
| | FRS | API-related FR item |
| Architecture change | ADR (`docs/02-architecture/`) | Whether the decision is recorded as an ADR |
| | Design doc | Whether the architecture change is reflected |
| Dashboard UI change | UX design or screen spec | Screen layout, data presentation format |
| Operational infra change | SLO definition (`slo-definitions.md`) | SLO definition for the change target |
| | BDM scenarios (`bdm-scenarios/`) | Given-When-Then monitoring scenario |
| | FMA (`failure-mode-analysis.md`) | Failure-mode analysis |
| Monitoring / alert addition | SLO definition | Alert thresholds, Error Budget |
| | BDM scenario | Alert-firing condition spec |
| Performance optimization | Performance-criteria SLO | Optimization target numbers |

## Document-content sufficiency criteria

### FRS (functional requirements)
- The FR-xxx items for the Sprint's features exist in **PLANNED** state
- Each FR has a priority (P0–P3) and acceptance criteria
- **When missing**: recommend a "Define FRS requirements" Task (S, tech-writer)

### Design doc
- The Sprint-scope feature's design is documented under `docs/03-design/`
- Includes data flow, component structure, API spec
- **When missing**: recommend a "Write design doc" Task (M, architect)

### ADR (architecture decision)
- When the Sprint includes an architecture change, an ADR Task exists
- No decision conflicts with existing ADRs
- **When missing**: recommend an "ADR-xxx architecture decision" Task (S, architect)

### SLO definitions
- The SLOs for added/changed features are defined in `slo-definitions.md`
- Include SLI, target, measurement, alerting policy
- **When missing**: add the SLO definition early in the Sprint via `/operability slo`

### BDM scenarios
- The changed components have BDM scenarios under `bdm-scenarios/`
- Given-When-Then-Alert-Recovery is complete
- **When missing**: run `/operability bdm {component}` or add a Task

### FMA (failure-mode analysis)
- The added components are analyzed in the FMA
- **When missing**: run `/operability fma {subsystem}` or add a Task

## Implementation-vs-document consistency criteria (Sprint completion)

Items the AI verifies on Sprint completion:

| Document | What to verify |
|----------|----------------|
| **Roadmap** | Whether Sprint completion is reflected in the Phase goal |
| **FRS** | Whether implemented FRs moved from PLANNED → VERIFIED |
| **Design docs** | Whether the changed feature designs are current |
| **ADR** | Whether decisions made in the Sprint are captured as ADRs |
| **SLO** | Whether SLOs for added / changed features are defined |
| **BDM** | Whether BDM scenarios for changed components are updated |
| **FMA** | Whether new failure modes are added to the FMA |
| **CLAUDE.md** | Whether Sprint status, test counts, and CLI commands are updated |
| **SPRINT.md** | Whether the Task-list status matches the actual Task frontmatter |
| **PRR** | If this was an operability Sprint, whether the PRR report was written |
