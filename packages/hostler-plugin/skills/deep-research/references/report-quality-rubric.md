# Report quality rubric — relocation notice

> **This file was moved during a refactor.**
> RACE/FACT rubric SSOT: **`skills/_shared/report-quality-rubric.md`**
> Reference the path above instead of this file.

---

<!-- The content below is legacy. Use _shared/ as the live reference. -->

Used in Step 7 self-evaluation after the report is complete.
Based on the DeepResearch Bench's two axes: RACE (report quality) + FACT (citation trustworthiness).

## RACE axis (report quality, 1–5 each)

### Depth

| Score | Criterion |
|-------|-----------|
| 5 | Every subject covered in depth (architecture / results / pros & cons) with numbers |
| 4 | Mostly in depth, a few items shallow |
| 3 | Mid-level, half detailed and half outline |
| 2 | Mostly outline-level, lacks numeric evidence |
| 1 | Mere enumeration, no analysis |

### Breadth

| Score | Criterion |
|-------|-----------|
| 5 | 7+ systems, 4+ perspectives, academic + real-world both |
| 4 | 5–6 systems, 3 perspectives |
| 3 | 3–4 systems, 2 perspectives |
| 2 | 2 systems, 1 perspective |
| 1 | 1 system or no perspective |

### Instruction-Following

| Score | Criterion |
|-------|-----------|
| 5 | Covers every aspect of the user's request |
| 4 | Covers all core points but misses 1 secondary request |
| 3 | Covers half the core |
| 2 | Drifts away from the core into other topics |
| 1 | Irrelevant content |

### Readability

| Score | Criterion |
|-------|-----------|
| 5 | 7-section structure with tables and code blocks; smooth flow |
| 4 | Good structure but some verbosity |
| 3 | Structure exists but disorganized |
| 2 | No structure, just enumeration |
| 1 | Hard to read |

## FACT axis (citation trustworthiness)

| Item | Criterion | Verdict |
|------|-----------|---------|
| Source Count | 10+ sources | pass (10+) / marginal (5–9) / fail (<5) |
| Citation Accuracy | [source#] references match the actual URL contents | 1–5 (verify 3 samples) |
| Recency | Share of sources within 2 years | % (70%+ pass) |
| Cross-Verification | Core claims backed by 2+ independent sources | pass / fail |

## Total scoring

```
RACE: Depth + Breadth + Instruction + Readability = ?/20
FACT: Source(pass/fail) + Accuracy(?/5) + Recency(?%) + Cross(pass/fail)
Combined: RACE ?/20 + FACT 4 items → report grade
```

| Grade | RACE | FACT |
|-------|------|------|
| A (excellent) | 16–20 | All items pass + Accuracy 4+ |
| B (good) | 12–15 | Source pass + 2 or more items pass |
| C (fair) | 8–11 | Source marginal |
| D (poor) | <8 | Source fail or Accuracy <3 |

## Usage

Run automatically in report Step 7:

```markdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Report self-evaluation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [RACE] Depth: 4 | Breadth: 5 | Following: 4 | Readability: 5 = 18/20
  [FACT] Sources: 12 (pass) | Accuracy: 4/5 | Recency: 83% (pass) | Cross: pass
  Grade: A (excellent)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
