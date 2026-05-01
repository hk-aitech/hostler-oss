# Report Quality Rubric — RACE + FACT

Used for self-evaluation after a report is finished.
Based on the two-axis model from DeepResearch Bench: RACE (report quality) + FACT (citation trust).

Source: extracted from `deep-research/references/report-quality-rubric.md`.

---

## RACE Axis (report quality, 1–5 points each)

### R — Depth (analytical depth)

| Score | Criterion |
|------|------|
| 5 | Each target gets architecture / performance / pros-and-cons backed by numbers, deeply analyzed |
| 4 | Mostly deep, some superficial |
| 3 | Medium — half detailed, half overview |
| 2 | Mostly overview, lacking numerical rationale |
| 1 | Just enumeration, no analysis |

### A — Breadth (coverage)

| Score | Criterion |
|------|------|
| 5 | 7+ items, 4+ perspectives, multiple sources combined |
| 4 | 5–6 items, 3 perspectives |
| 3 | 3–4 items, 2 perspectives |
| 2 | 2 items, 1 perspective |
| 1 | 1 item or no perspective |

### C — Instruction-Following (faithfulness to the request)

| Score | Criterion |
|------|------|
| 5 | Covers every aspect of the user's request without omission |
| 4 | Covers all the core asks but misses 1 secondary one |
| 3 | Covers half of the core asks |
| 2 | Strays from the core request to other topics |
| 1 | Content unrelated to the request |

### E — Readability

| Score | Criterion |
|------|------|
| 5 | Clear section structure, tables / code blocks used appropriately, natural flow |
| 4 | Structure is good but some sections are long-winded |
| 3 | Has structure but lacks polish |
| 2 | No structure, just enumeration |
| 1 | Hard to read |

---

## FACT Axis (citation trust)

| Item | Criterion | Judgement |
|------|------|----------|
| F — Source Count | Number of sources | pass (10+) / marginal (5–9) / fail (<5) |
| A — Citation Accuracy | Citations match actual source content | 1–5 points (3-sample check) |
| C — Recency | Share of sources within 2 years | % (70%+ pass) |
| T — Cross-Verification | Core claims backed by 2+ independent sources | pass / fail |

---

## Total Score

```
RACE: Depth + Breadth + Following + Readability = ?/20
FACT: Source Count(pass/marginal/fail) + Accuracy(?/5) + Recency(?%) + Cross(pass/fail)
Composite: RACE ?/20 + FACT 4 items → report grade
```

| Grade | RACE | FACT |
|------|------|------|
| A (excellent) | 16–20 | All items pass + Accuracy 4+ |
| B (good) | 12–15 | Source pass + 2+ items pass |
| C (fair) | 8–11 | Source marginal |
| D (insufficient) | <8 | Source fail or Accuracy <3 |

---

## Self-Evaluation Output Format

Append the following block to the end of the report:

```markdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Report Self-Evaluation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [RACE] Depth: {1-5} | Breadth: {1-5} | Following: {1-5} | Readability: {1-5} = {sum}/20
  [FACT] Sources: {N} ({pass/marginal/fail}) | Accuracy: {1-5}/5 | Recency: {%} | Cross: {pass/fail}
  Grade: {A/B/C/D} ({excellent/good/fair/insufficient})
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Example

```markdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Report Self-Evaluation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [RACE] Depth: 4 | Breadth: 5 | Following: 4 | Readability: 5 = 18/20
  [FACT] Sources: 12 (pass) | Accuracy: 4/5 | Recency: 83% | Cross: pass
  Grade: A (excellent)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## When to Use

- After completing a research report (internet research, codebase audit, document analysis, etc.)
- When checking report quality objectively before submission
- If RACE is D or any FACT item is fail, fix that section and re-evaluate
