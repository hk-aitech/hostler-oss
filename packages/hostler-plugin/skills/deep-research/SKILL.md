---
name: deep-research
description: |
  Conduct deep web research with 5–10 multi-angle searches, Perspective Mining, a
  Knowledge Gap loop, and RACE/FACT self-evaluation, then deliver a structured
  research report. Use this skill when the user mentions "deep research",
  "competitive analysis", "trend research", "prior art", "tech selection",
  "compare with other systems", "research this thoroughly", or asks for current
  external data — even when they don't explicitly say "deep-research". Do NOT
  use for codebase-only questions.
compatibility:
  tools: [Read, WebSearch, WebFetch, Bash]
paths: ["**/*"]
---

# Deep Research — Deep-Dive Analytical Report

Survey a topic broadly across the web, analyze the findings, and produce a structured deep-research report. Coverage includes competitive system comparison, technology trend analysis, and improvement proposals.

## Trigger keywords

"deep analysis", "competitive analysis", "research report", "trend analysis",
"technology comparison", "benchmark study", "find the latest info on", "deep research",
"compare us with similar systems", "/deep-research"

## When you must use this skill

- A systematic comparison against competing systems or similar projects is needed
- Current trend or benchmark data is needed before a technology choice
- The question requires multi-angle cross-analysis rather than a single search

## Absolute rules

- Run 5–10 multi-angle searches, not a single search (returning a single search result directly is forbidden)
- Cite sources for every claim ("According to [source#]" format is mandatory)
- Validate the report through RACE/FACT self-evaluation before delivery

## Shared methodology reference

> This skill follows the shared analytical methodology defined in `skills/_shared/analysis-methodology.md`
> (Perspective Mining, Knowledge Gap loop, Chain-of-Verification, Grounding Pass).
> This document layers **specifics for external web research** on top of that.
> (Parallel WebSearch invocation patterns, Trusted Domain matrix, URL Grounding implementation)

## Workflow

### Step 1 — Define the research topic

Extract the following from the user's request:
- Research topic (what to investigate)
- Comparison targets (when an internal system exists)
- Perspective of interest (architecture? technology? cost? performance?)
- Output format (report? comparison table? improvement list?)
- **Research mode**: Standard mode (default) or Outline-First mode (paper-style / long-form)

**Mode selection criteria**:

| Condition | Mode |
|-----------|------|
| Comparison analysis, trend research, resource compilation | **Standard mode** (Steps 1–7.5 sequentially) |
| "Paper-style", "detailed report", expected 20+ pages, academic analysis | **Outline-First mode** (outline approval → section-by-section deep writing) |
| User explicitly requests "outline first" or "structure first" | **Outline-First mode** |

For the detailed Outline-First workflow, see `references/report-format-samples.md` Type 4.

### Step 1.5 — Plan Clarification (research plan review)

After Step 1, **present the research plan to the user** and obtain approval or revision.
Output format: see `examples/report-output-example.md` §Step 1.5.

This gate prevents misalignment in research direction up front.

### Step 2 — Build search strategy

Derive 5–10 search queries from the topic.

Search types: core keywords (English + Korean), competing systems / open source, academic papers / benchmarks, recent trends (2025–2026), real-world cases.
Run searches in **both English and Korean** and include the latest years (2025–2026).

For detailed query patterns and examples by research type (technology comparison, competitive analysis, trend research, academic research) see `references/search-strategies.md`.

### Step 2.5 — Perspective Mining

Before generating queries, **automatically generate 4–6 perspectives** that the topic can be viewed from and assign 1–2 queries per perspective.
Methodology details: see `skills/_shared/analysis-methodology.md` §1.

**Example for deep-research** ("real-time IoT data collection architecture"):
- System architect: scalability, fault tolerance, latency
- Operator: deployment complexity, monitoring, recovery
- Data engineer: schema evolution, data quality, backfill
- Cost manager: infra cost, licensing, headcount
- Security engineer: API key management, network isolation, audit logs

### Step 3 — Information gathering (WebSearch × N)

Run a WebSearch for each query and structure the results.

**Parallel search recommended**: invoke **parallel WebSearch within a single message** for independent queries. Claude Code supports parallel tool calls, so wall-clock time is reduced N-fold.

**Prefer trusted domains**: prioritize trusted sources by research type.

| Research type | Trusted Domains |
|---------------|----------------|
| Academic | arxiv.org, scholar.google.com, ieee.org, acm.org |
| Technical | github.com, docs.*, stackoverflow.com, dev.to |
| Korean market | dart.fss.or.kr, krx.co.kr, kind.krx.co.kr |
| Official docs | *.io/docs, docs.*, developer.* |

| Field | Content |
|-------|---------|
| System / framework name | Name, GitHub stars, latest version |
| Architecture | Language, pattern, core design |
| Core features | 3–5 differentiating features |
| Performance / benchmarks | Quantitative results (when available) |
| Pros and cons | Based on public reviews / issues |
| URLs | Official site, GitHub, papers |

**Citation rule**: tag every collected piece of information with `[source#]` immediately. Cite as "According to [source#]" when writing the report.

### Step 3.5 — Knowledge Gap loop

After the first round of searches, **draft a report outline**, identify gaps, and run a second round of searches.
Methodology details: see `skills/_shared/analysis-methodology.md` §2.

**Application to deep-research**: for each section, evaluate WebSearch source sufficiency (✅ sufficient / ⚠️ weak / ❌ insufficient) and run gap-specific queries for ⚠️/❌ sections.
If still ❌, mark the section as "evidence insufficient" in the report.

### Step 4 — Comparative analysis

Compare the gathered information against the internal system. For comparison matrix structure details, see `references/report-template.md` §4.

Core output: comparison matrix + internal-system strengths and weaknesses (relative to competitors).

### Step 5 — Derive improvement proposals

Derive concrete improvements from the comparative analysis:

```
Improvement-derivation criteria:
1. Features competitors have but you don't → consider adoption
2. Patterns validated by current trends → consider applying
3. Techniques proven in academic benchmarks → consider experimenting
4. Assign a Phase to each proposal (immediate / mid-term / long-term)
```

### Step 6 — Write the report

For the standard report structure (7 sections: Research Purpose, Subjects, Recent Trends, Comparative Analysis, Improvements, Conclusion, References) and per-section guidance and length targets, see `references/report-template.md`.

**Hallucination guardrails** (mandatory while writing Step 6):
1. **According-to**: every factual claim must carry an `According to [source#]` citation
2. **Chain-of-Verification (CoVe)**: run the self-verification loop after the draft is complete. Details: `skills/_shared/analysis-methodology.md` §3
3. **No unsourced claims**: any claim without a source must be marked "needs further verification" or removed

Citation quality criteria: see `skills/_shared/citation-quality.md`.

### Step 7 — Self-quality evaluation

After the report is complete, run a self-evaluation against the RACE/FACT criteria.
Rubric and output format: see `skills/_shared/report-quality-rubric.md`.

### Step 7.5 — Grounding Pass (real URL verification)

**WebFetch every reference URL** in the report to verify it exists.
Methodology details: see `skills/_shared/analysis-methodology.md` §4.

**deep-research implementation** (WebFetch specifics):
- Run **WebFetch on each URL** (parallelism recommended)
- For numeric citations, re-confirm the numbers in the WebFetch result
- If there are 15+ URLs, **sample-verify only the 5 most important**
- A bot block (403) should be marked as [access restricted] and not penalize trust score
- Append the Grounding result as an appendix at the end of the report

```markdown
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Grounding Pass results
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Sampled: {M}/{N}
  ✅ Confirmed: {N} | ⚠️ Uncertain: {N} | ❌ Failed: {N}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Outline-First mode

A two-stage workflow used for **paper-style / long-form reports**.
Steps 1–3.5 stay identical to the standard mode; the workflow branches at Step 4.

### Outline-First workflow

```
Steps 1–3.5: same as standard mode (define topic → search → Knowledge Gap)
  ↓
Step OL-1: produce a detailed Outline
  - For each section, state expected length, key arguments, and the source numbers to use
  - Present to the user → "approve" / "revise"
  ↓
Step OL-2: write each section in depth
  - Additional WebSearch is allowed per section (when needed)
  - Emit interim progress as each section completes
  ↓
Step OL-3: integrate + reconcile cross-references
  - Verify cross-section reference consistency
  - Remove duplicates + unify terminology
  ↓
Steps 7–7.5: self-evaluation + Grounding (same as standard mode)
```

### Standard vs Outline-First

| Item | Standard mode | Outline-First |
|------|---------------|---------------|
| Suitable for | Comparison, trend research | Paper-style, 20+ pages |
| Search timing | Bulk in Step 3 | Step 3 + per-section additions |
| User approval | Step 1.5 (once) | Steps 1.5 + OL-1 (twice) |
| Writing style | Single bulk pass | Section-by-section |
| Expected length | 200–500 lines | 500–1,000+ lines |

For the detailed Outline template, see `references/report-format-samples.md` Type 4.

## Subagent delegation mode

For **XL-scale research expecting 15+ sources**, delegate to subagents in parallel.

### Workflow

```
Step 2.5 Perspective Mining → derive 4–6 perspectives
  ↓
Run an Agent(Explore) or general-purpose subagent in parallel per perspective
  ↓
Each subagent: 2–3 queries for its perspective → WebSearch → returns a summary
  ↓
Collect results → Step 3.5 Knowledge Gap evaluation → Step 4 comparative analysis
```

### Subagent prompt template

```
Topic: {research topic}
Assigned perspective: {perspective name} — {perspective description}
Query scope: {2–3 search queries}
Collection format: for each source, capture (name, URL, 3-line key content, numeric data)
Citation rule: tag every piece of information with [source#]
Return: markdown summary, ≤200 lines
```

**Usage criteria**: regular research (5–10 sources) — single agent; XL research (15+ sources) — subagents.

## Quality criteria

| Criterion | Minimum | Recommended |
|-----------|---------|-------------|
| WebSearch invocations | 3 | 5–10 |
| Comparison systems | 3 | 5–7 |
| Improvement proposals | 3 | 5–10 |
| Sources | 5 | 10–15 |
| Report length | 200 lines | 300–500 lines |
| RACE total | 12/20 | 16/20 |
| FACT cross-verification | — | pass |

## Edge Cases

- **Too few search results (<3)**: broaden the queries (drop year filter, retry in English) and expand scope to similar systems in adjacent domains (crypto, FX).
- **Pure research without an internal system**: turn the Step 4 comparison into "comparison among the subjects". Replace the "internal system" column with "requirements".
- **Confidential system comparison request**: collect only public information. Use non-public internal materials only when the user explicitly provides them.

## Reference files

### References
- **`references/search-strategies.md`** — Query patterns by research type
- **`references/report-template.md`** — 7-section report structure + writing guidance
- **`../_shared/citation-quality.md`** — FACT 4-axis citation quality checklist (SSOT)
- **`../_shared/report-quality-rubric.md`** — RACE/FACT report-quality rubric (SSOT)
- **`../_shared/analysis-methodology.md`** — Analytical methodology (Perspective Mining, Knowledge Gap loop, CoVe, Grounding Pass) (SSOT)
- **`references/report-format-samples.md`** — Format samples for 5 report types (competitive analysis / tech trend / resource / outline / incident analysis)

### Examples
- **`examples/report-output-example.md`** — Sample outputs for each Step (10 search queries + report skeleton included)

## Related skills

- `architecture-design` — Use competitive analysis when making architecture decisions
- `implementation-planning` — Convert improvements into Sprints / Tasks
