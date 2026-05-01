# Sprint estimation

## Task size definitions

| Size | Code | Time | Suitable for |
|------|------|------|--------------|
| XS | Extra Small | 30 min | Config tweak, typo fix |
| S | Small | 1–2 h | Single function, simple API |
| M | Medium | 2–4 h | Component, service class |
| L | Large | 4–8 h | Complex feature, integration work |
| XL | Extra Large | 8 h+ | **Must be decomposed** |

---

## Estimating Sprint workload

```
Sprint workload = Σ (estimated time per Task)

Example:
- XS × 3 = 1.5 h
- S × 5 = 7.5 h
- M × 4 = 14 h
- L × 1 = 6 h
────────────────────
Total: 29 h

✅ Within healthy range (20–40 h)
```

---

## Sprint size recommendations

| Size | Tasks | Workload | Suitable for |
|------|-------|----------|--------------|
| Small | 3–5 | 10–20 h | Short cycle, maintenance |
| **Standard** | **5–15** | **20–40 h** | **Recommended** |
| Large | 15–20 | 40–60 h | Focused development period |

---

## Priority matrix

```
              Importance
           High      Low
         ┌─────────┬─────────┐
 High    │   P0    │   P1    │
Urgency  │ Now     │ Plan    │
         ├─────────┼─────────┤
 Low     │   P2    │   P3    │
         │ Sched.  │ Backlog │
         └─────────┴─────────┘
```

| Priority | Description | Action |
|----------|-------------|--------|
| P0 | Blocker, must resolve immediately | Top priority in the current Sprint |
| P1 | Important, within planned schedule | Include in current Sprint |
| P2 | Schedule-flexible | Could land in the next Sprint |
| P3 | Backlog | Pick up when capacity allows |

---

## Inclusion criteria for documentation Tasks

If you only ship code and defer the docs, doc/code drift accumulates and forces a separate "doc-fix Sprint" later.

| Sprint type | Recommended Tasks | Candidates |
|-------------|-------------------|------------|
| Feature implementation (code change) | 1–2 required | Refresh design docs, sync RTM, write ADR |
| Bug fix / hotfix | 0–1 optional | Document error codes, refresh ops guide |
| Docs / design only | N/A | The Sprint itself is doc work |

Path reference for documentation Tasks:

| Document type | Path |
|---------------|------|
| Refresh design docs | `docs/03-design/` (domain/, features/, operations/) |
| Reflect into FRS / SRS | `docs/02-requirements/FRS.md`, `SRS.md` |
| Sync RTM | `docs/02-requirements/RTM.md` |
| Write ADR | `docs/02-architecture/` |
| CLAUDE.md implementation state | `CLAUDE.md` at the project root |

---

## Sprint-planning best practices

- **80–85% capacity**: plan for 80–85% of total available time. Leave the rest as an uncertainty buffer.
- **Dependency mapping**: declare Task dependencies in `depends_on`; identify bottlenecks up front.
- **Balanced mix**: distribute P0–P2 evenly (don't fill the Sprint with P0 only).
- **Include spikes**: assign XS–S spike (research/experiment) Tasks for items with high technical uncertainty.
- **Clear goal**: define the Sprint goal in 1–2 sentences (prevents scope creep).
- **Check prior documents**: review `docs/00-project/pdd.md`, `docs/00-project/roadmap.md`, `docs/03-design/`.

---

## Improving estimation accuracy

### Use historical data

```markdown
## Estimate vs actual

| Sprint | Estimate | Actual | Error |
|--------|----------|--------|-------|
| p1-s1 | 30h | 35h | +17% |
| p1-s2 | 25h | 28h | +12% |
| p1-s3 | 32h | 30h | -6% |

Average error: +7.7%
→ Add a 10% buffer to the next Sprint's estimate
```

### Uncertainty buffer

| Uncertainty | Buffer | When to apply |
|-------------|--------|---------------|
| Low | +10% | Familiar tech |
| Medium | +25% | New technology adoption |
| High | +50% | Uncharted territory |
