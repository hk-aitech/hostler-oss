# Report format samples — best practices by type

Per-type structure and format samples for reports produced by the deep-research skill.
Pick the type that fits the user's request and follow the corresponding structure.

---

## Type 1: Competitive Analysis report

The most common type. Compares an internal system to competitors and surfaces improvements.

```markdown
# {Topic} — Competitive Analysis Report

> Date: {date}
> Topic: {topic}
> Subjects: {N} systems

---

## 1. Research purpose

### 1.1 Background
{2–3 sentences on why this research is needed}

### 1.2 Subjects

| Subject | Type | Perspective |
|---------|------|-------------|
| {System A} | Commercial | {perspective} |
| {System B} | OSS | {perspective} |
| {Internal system} | Self | Current AI-friendliness assessment |

---

## 2. Subjects in detail

### 2.1 {System A} (Commercial)

| Item | Content |
|------|---------|
| Source | {URL} |
| Tech base | {tech stack} |
| Key features | {3–5 items} |

**Key features**:
- According to [1], {feature 1 detail}
- According to [2], {feature 2 detail}

### 2.2 {System B} (OSS)
{Repeat the same structure}

---

## 3. Recent trends (2025–2026)

### Trend 1 — {name}
{description}. According to [3], {evidence}.
**Relevance to internal system**: {high/medium/low} — {reason}

### Trend 2 — {name}
{Same structure}

---

## 4. Comparative analysis

### 4.1 Comparison matrix

| Item | Internal | System A | System B | System C |
|------|----------|----------|----------|----------|
| {axis 1} | {value} | {value} | {value} | {value} |
| {axis 2} | {value} | {value} | {value} | {value} |

### 4.2 Internal strengths
1. **{strength 1}** — {description}. {Why no competitor offers it}

### 4.3 Internal weaknesses
1. **{weakness 1}** — {description}. {Which competitor handles it better}

---

## 5. Improvement proposals

### Immediate adoption (Phase 1)

| # | Proposal | Rationale |
|---|----------|-----------|
| 1 | **{proposal}** | {validated by which competitor / paper} |

### Mid-term adoption (Phase 2)
{Same table}

---

## 6. Conclusion

### Key findings
1. {finding 1}
2. {finding 2}

### Recommended priority
1. {single highest-impact item}

---

## 7. References (Sources)

| # | Title | URL | Category |
|---|-------|-----|----------|
| 1 | {title} | {URL} | {academic / official / blog} |
```

---

## Type 2: Technology Trend report

Investigate the latest movement in a specific technology area to decide on adoption.

```markdown
# {Technology area} Trend Report (2025–2026)

> Date: {date}
> Scope: {technology area}

---

## 1. Executive Summary

{3–5 sentence summary of key findings. A decision-maker should be able to read this alone and grasp the conclusion.}

## 2. Background

### 2.1 Why now
{Market shift, technology maturity, cost change, etc.}

### 2.2 Evaluation criteria

| Criterion | Weight | Measurement |
|-----------|--------|-------------|
| Maturity | 30% | GitHub stars, adopting companies |
| Performance | 25% | Benchmark numbers |
| Ecosystem | 20% | Plugins, integrations, docs |
| Cost | 15% | Licensing, infra |
| Learning curve | 10% | Onboarding time |

## 3. Technology landscape

### 3.1 {Technology A}
- **Current state**: According to [1], {state}
- **Adoption**: {companies / projects}
- **Pros**: {3 items}
- **Cons**: {3 items}

### 3.2 {Technology B}
{Same structure}

## 4. Comparative evaluation

| Criterion | Tech A | Tech B | Tech C |
|-----------|--------|--------|--------|
| Maturity (30%) | ★★★★☆ | ★★★☆☆ | ★★★★★ |
| Performance (25%) | ★★★☆☆ | ★★★★★ | ★★★★☆ |
| Ecosystem (20%) | ★★★★★ | ★★★☆☆ | ★★★★☆ |
| Cost (15%) | ★★★★☆ | ★★★★★ | ★★☆☆☆ |
| Learning curve (10%) | ★★★☆☆ | ★★★★☆ | ★★★☆☆ |
| **Weighted total** | **3.85** | **3.80** | **3.95** |

## 5. Recommendation

### Recommended: {Tech C}
- **Reason**: {top weighted score + key strengths}
- **Risk**: {primary weakness + mitigation}
- **Roadmap**: Phase 1 (PoC) → Phase 2 (pilot) → Phase 3 (full rollout)

### Alternative: {Tech A}
- **When**: {if a specific risk of Tech C materializes}

## 6. References
{Same structure}
```

---

## Type 3: Resource report

Collect and organize key resources (libraries, tools, papers) for a topic.

```markdown
# {Topic} resource guide

> Date: {date}
> Resources collected: {N}

## Resources by category

### Official documentation
| # | Resource | URL | Notes |
|---|----------|-----|-------|
| 1 | {official spec} | {URL} | Latest version {X.Y} |

### Open source tools
| # | Name | GitHub | Stars | Language | Notes |
|---|------|--------|-------|----------|-------|
| 1 | {tool A} | {URL} | {N}k | {language} | {one-liner} |

### Academic papers
| # | Title | Authors | Year | Key contribution |
|---|-------|---------|------|------------------|
| 1 | {paper title} | {authors} | {year} | {one-line summary} |

### Tutorials / blogs
| # | Title | URL | Difficulty | Why recommended |
|---|-------|-----|------------|-----------------|
| 1 | {title} | {URL} | beginner/intermediate/advanced | {reason} |

## Recommended learning path

```
1. Intro: {resource A} → {resource B}
2. Practice: {resource C} → {resource D}
3. Advanced: {resource E} → {paper F}
```

## References
{Search queries and source list used during collection}
```

---

## Type 4: Outline report

A two-stage report that fixes the structure first, gets user approval, and then fills in the details.
The Outline-First pattern in the STORM style.

```markdown
# {Topic} — Research Outline

> Status: outline stage — approval required before drafting
> Expected length: {N} lines

## Proposed table of contents

### 1. Introduction
- Motivation: {one line}
- Research questions: {1–3}
- Scope: {included / excluded}

### 2. Background
- 2.1 {sub-topic A}: {one-line coverage}
- 2.2 {sub-topic B}: {one-line coverage}

### 3. Findings
- 3.1 {area A}: {expected source count} sources
- 3.2 {area B}: {expected source count} sources
- 3.3 {area C}: {expected source count} sources

### 4. Analysis
- 4.1 Comparison matrix: {axes}
- 4.2 SWOT or strengths/weaknesses

### 5. Recommendations
- Roughly {N} expected

### 6. Conclusion

### 7. References

## Review request

Please review this table of contents:
- "approve" → start drafting
- "revise: drop 3.3 and add a security section" → re-present after revision
```

---

## Type 5: Incident / Postmortem report

Structures a system incident's cause, impact, response, and prevention.

```markdown
# Incident Report — {incident title}

> Started: {date time}
> Recovered: {date time}
> Impact duration: {N}h {M}m
> Severity: {Critical / High / Medium}

## 1. Summary

| Item | Content |
|------|---------|
| Symptom | {symptom users experienced} |
| Root cause | {one sentence} |
| Impact scope | {affected services / users} |
| Resolution | {one sentence} |

## 2. Timeline

| Time (KST) | Event |
|------------|-------|
| {HH:MM} | {incident started — first detection} |
| {HH:MM} | {triage started — owner notified} |
| {HH:MM} | {root cause identified} |
| {HH:MM} | {fix applied} |
| {HH:MM} | {recovery confirmed} |

## 3. Root cause analysis (5 Whys)

1. **Why** did {symptom} happen? → {direct cause}
2. **Why** did {direct cause} happen? → {secondary cause}
3. **Why** did {secondary cause} happen? → {third cause}
4. **Why** did {third cause} happen? → {fourth cause}
5. **Why** did {fourth cause} happen? → **{root cause}**

## 4. Impact analysis

| Impact | Number |
|--------|--------|
| Downtime | {N} min |
| Data loss | {count or 0} |
| Financial impact | {amount or none} |

## 5. Prevention actions

| # | Action | Owner | Due | Task |
|---|--------|-------|-----|------|
| 1 | {immediate action} | {owner} | done | — |
| 2 | {short-term action} | {owner} | {date} | T{NNN} |
| 3 | {long-term action} | {owner} | {date} | T{NNN} |

## 6. Lessons learned

- **Keep**: {what went well — quick detection, effective triage, etc.}
- **Problem**: {what hurt — missing monitoring, missing docs, etc.}
- **Try**: {improvements — add alerts, auto-recovery, etc.}

## 7. References

- Incident log: {path}
- Related KB: {KB card ID}
- Related Task: {Task ID}
```

---

## Format selection guide

| User request | Recommended type |
|--------------|------------------|
| "compare with X" / "competitive analysis" | Type 1 Competitive analysis |
| "latest trends" / "tech comparison" / "what should we use" | Type 2 Technology trend |
| "compile related resources" / "resource list" | Type 3 Resource |
| "big topic, structure first" / "paper-style" | Type 4 Outline (Outline-First) |
| "incident analysis" / "why did this happen" / "postmortem" | Type 5 Incident analysis |
| Unspecified (default) | Type 1 Competitive analysis |
