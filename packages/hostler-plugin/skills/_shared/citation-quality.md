# Citation Quality Criteria — Evidence Grades and Verification Obligations

The evidence grading standard applied when citing information in a report or
analysis document. Provides the basis for the trust judgement in the FACT rubric.

Source: extracted from `deep-research/references/citation-quality.md`.

---

## 1. Source Hierarchy

| Grade | Type | Examples |
|------|------|------|
| A (top) | Academic papers (peer-reviewed) | arxiv.org, IEEE, ACM |
| A | Official documentation / vendor announcements | docs.*, official blog, official site |
| A | Primary sources (code, git log, config files) | source code read directly, commit history |
| B (good) | Technical blogs (personal / corporate) | medium.com, dev.to, company blog |
| B | Open-source repos + README | github.com |
| C (reference) | Community / forums | stackoverflow, reddit, HN |
| D (discouraged) | SEO spam, auto-generated content | unclear-origin sites |

**Rules**:
- Do not cite D-grade sources.
- C-grade only with cross-verification by A / B sources.
- For codebase analysis, source code read directly is treated as A-grade.

---

## 2. Recency

| Criterion | Verdict |
|------|------|
| Within 2 years | ✅ Recommended |
| 2–5 years | ⚠️ Mark "as of {year}" |
| 5+ years | ❌ Cite only in historical context |

**Rule**: At least **70%** of all sources in the report must be within 2 years.

For codebase sources, judge by commit date or file modification time.

---

## 3. Cross-Verification

| Criterion | Verdict |
|------|------|
| 2+ independent sources confirm the same fact | ✅ pass |
| Only 1 source confirms | ⚠️ Mark "single source, further verification recommended" |
| Claim without source | ❌ Remove or mark "estimate" |

---

## 4. Authority

| Criterion | Verdict |
|------|------|
| Specialist organization / author in the field | ✅ High |
| Practitioner in a related field | ⚠️ Medium |
| Unclear author / organization | ❌ Cross-verification required |

---

## 5. Citation Format

```markdown
According to [1] (arxiv, 2025, peer-reviewed), ...
← Grade A, within 2 years, cross-verified

According to [5] (blog, 2024, cross-verified by [7]), ...
← Grade B, cross-verified

Note: [9] is a single community source; further verification recommended.
← Grade C, single source
```

### Codebase Source Citation

```markdown
According to [src:path/to/file.go:L42], ...
← Grade A (primary source read directly)

As of commit a1b2c3d (2025-03-15), ...
← Grade A (git log confirmed)
```

---

## 6. Verification Obligation by Grade

| Grade | Verification obligation |
|------|----------|
| A | Source URL or file path required when citing |
| B | URL required + A/B cross-verification recommended |
| C | A or B cross-verification required |
| D | Citation forbidden |

---

## When to Use

- When citing external information in a report or analysis document
- When evaluating the Citation Accuracy of the FACT rubric
- Common application across codebase audits, internet research, document analysis, and any other investigative work
