# Deep-research report structure template

> Standard report format used in Step 6 (writing) of the deep-research skill.
> Applies uniformly to competitive analysis, trend research, technology comparison, and any other deep-research output.

---

## Report header

```markdown
# {Topic} — Deep Research Report

> **Document ID**: {project}-RESEARCH-NNN
> **Date**: {YYYY-MM-DD}
> **Topic**: {research topic}
> **Research type**: Competitive analysis | Tech comparison | Trend research | Academic research
> **Searches**: {N}
> **Subjects**: {N} systems / technologies / papers
```

---

## Per-section authoring guide

### Section 1: Research purpose

```markdown
## 1. Research purpose

**Background**: {why this research is needed; current state and the problem}

**Research questions**:
1. {Question 1 — the most important decision question}
2. {Question 2}
3. {Question 3 (optional)}

**Scope**:
- Included: {what is being investigated}
- Excluded: {what was deliberately left out}
```

**Writing guidance:**
- Keep to 1–2 short paragraphs
- Be explicit about what decision will be made after the research

---

### Section 2: Subjects

```markdown
## 2. Subjects ({N} systems / technologies)

### 2.1 {System / technology A}

| Item | Content |
|------|---------|
| Source | {official site URL} |
| Version / date | {latest version}, {date} |
| License | {MIT / Apache 2.0 / commercial / etc.} |
| GitHub Stars | {number} (when applicable) |

**Key features**:
- {feature 1}
- {feature 2}
- {feature 3}

**Architecture summary**: {language, pattern, core design — 1–2 sentences}

**Results / benchmarks**: {published numbers — omit if unavailable}

---
{Repeat for each subject}
```

**Writing guidance:**
- Minimum 3 subjects, recommended 5–7
- Each subject must be self-contained and understandable on its own
- Cite a source for every piece of information

---

### Section 3: Recent trends

```markdown
## 3. Recent trends ({N} trends)

> Reference period: 2024–2026

### Trend 1: {name}

{trend description + supporting source}

**Relevance to internal system**: {high / medium / low} — {reason}

---
{Repeat per trend}
```

**Writing guidance:**
- 2–5 trends
- Include relevance evaluation for the internal system on every trend
- State the period explicitly so the recency is auditable

---

### Section 4: Comparative analysis

```markdown
## 4. Comparative analysis

### 4.1 Comparison matrix

| Item | Internal | {Competitor A} | {Competitor B} | {Competitor C} |
|------|----------|----------------|----------------|----------------|
| Language / platform | ... | ... | ... | ... |
| Architecture pattern | ... | ... | ... | ... |
| Core features | ... | ... | ... | ... |
| AI / ML | ... | ... | ... | ... |
| Real-time processing | ... | ... | ... | ... |
| Scalability | ... | ... | ... | ... |
| Open source | ... | ... | ... | ... |

### 4.2 Internal strengths (vs. competitors)

1. {Strength 1} — {evidence}
2. {Strength 2} — {evidence}
3. {Strength 3} — {evidence}

### 4.3 Internal weaknesses (vs. competitors)

1. {Weakness 1} — {how the competitor solved it}
2. {Weakness 2} — {how the competitor solved it}
3. {Weakness 3} — {how the competitor solved it}
```

**Writing guidance:**
- 5–10 matrix items is a comfortable range
- Always pair strengths/weaknesses with evidence (no bare lists)

---

### Section 5: Improvement proposals

```markdown
## 5. Improvement proposals ({N})

### Immediate adoption (Phase 1 — 1–2 Sprints)

| # | Proposal | Rationale |
|---|----------|-----------|
| 1 | {proposal} | {learned from competitor A} |

### Mid-term adoption (Phase 2 — 1–3 months)

{Same format}

### Long-term consideration (Phase 3 — 3+ months)

{Same format}
```

**Writing guidance:**
- Minimum 3, recommended 5–10
- Tag every proposal with a Phase (immediate / mid-term / long-term)
- Lean into Phase 1: immediately actionable proposals carry the most value

---

### Section 6: Conclusion

```markdown
## 6. Conclusion

**Key insights**:
1. {Insight 1 — the most important finding}
2. {Insight 2}
3. {Insight 3}
4. {Insight 4 (optional)}
5. {Insight 5 (optional)}

**Recommended priority**: {1–2 highest-impact immediate proposals}

**Next steps**:
- {action item 1}
- {action item 2}
```

**Writing guidance:**
- 3–5 key insights
- 1–2 paragraphs (keep it tight)
- "Next steps" must be concrete action items

---

### Section 7: References

```markdown
## References

1. [{title}]({URL}) — {source description}
2. [{title}]({URL}) — {source description}
...
```

**Writing guidance:**
- Include every URL cited in the body (mandatory)
- Minimum 5, recommended 10–15
- Prefer official docs / papers / GitHub links over plain search-result URLs

---

## Length targets

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| Total report length | 200 lines | 300–500 lines |
| Subjects | 3 | 5–7 |
| Comparison matrix items | 5 | 8–12 |
| Improvement proposals | 3 | 5–10 |
| References | 5 | 10–15 |

---

---

## Outline-First mode report structure

A two-stage structure used for paper-style / long-form reports.
Generate the Outline below in Step OL-1, get user approval, and write section by section in OL-2.

### Outline output format

```markdown
# {Topic} — Research Outline

> Status: awaiting Outline approval
> Expected length: {N} lines (~{M} pages)
> Expected sources: {N}

## Proposed table of contents

### 1. Introduction (~100 lines)
- Motivation: {one-line summary}
- Research questions: {1–3}
- Scope: include({scope}) / exclude({scope})
- Sources used: [1], [3], [5]

### 2. Background (~150 lines)
- 2.1 {sub-topic A}: {coverage} — sources: [2], [4]
- 2.2 {sub-topic B}: {coverage} — sources: [6], [8]

### 3. Findings (~200 lines)
- 3.1 {area A}: {N} sources — [1], [7], [9]
- 3.2 {area B}: {N} sources — [3], [10], [11]
- 3.3 {area C}: {N} sources — [5], [12]

### 4. Analysis (~150 lines)
- 4.1 Comparison matrix: {axes}
- 4.2 SWOT / strengths & weaknesses

### 5. Improvement proposals (~100 lines)
- ~{N} expected, Phase (immediate / mid / long)

### 6. Conclusion (~50 lines)

### 7. References
- ~{N} expected

---
Approve: "proceed" → start drafting per section
Revise: "drop 2.2, add a security section" → re-present after Outline revision
```

### Grounding Pass appendix format

Append the Step 7.5 Grounding result at the end of the report:

```markdown
## Appendix: Grounding Pass results

| # | URL | Status | Notes |
|---|-----|--------|-------|
| [1] | {URL} | ✅ confirmed | content matches |
| [3] | {URL} | ⚠️ uncertain | numeric position on page not confirmed |
| [7] | {URL} | ❌ broken | 404 Not Found |
| [9] | {URL} | ⚠️ restricted | 403 Forbidden (bot block) |

Verified: {M}/{N} sampled | ✅ {N} | ⚠️ {N} | ❌ {N}
```

---

## Related references

- Search strategies guide: `references/search-strategies.md`
- Citation quality criteria: `skills/_shared/citation-quality.md`
- Report quality rubric: `skills/_shared/report-quality-rubric.md`
- Report formats by type: `references/report-format-samples.md`
- deep-research skill workflow: SKILL.md
