# Analysis Methodology — SSOT

> This methodology applies identically across data sources — WebSearch, Grep,
> or anything else. Reuse it for internet research, codebase exploration,
> document analysis, and any other investigative work.

Source: extracted from the `deep-research` skill.

---

## 1. Perspective Mining

Before starting research, **derive 4–6 perspectives** from which the topic can be viewed.

### Purpose

Information collected from a single perspective develops structural bias.
Listing perspectives explicitly guarantees coverage structurally and lets you
distribute follow-up queries / search ranges across each perspective.

### Procedure

1. List 4–6 perspectives that view the topic from different roles / interests.
2. Allocate 1–2 research items (queries, exploration paths, etc.) to each perspective.
3. Ensure every perspective has at least one research item.

### Example (IoT data ingestion architecture)

| Perspective | Core questions |
|------|----------|
| System architect | Scalability, fault tolerance, latency |
| Operator | Deployment complexity, monitoring, recovery |
| Data engineer | Schema evolution, data quality, backfill |
| Cost manager | Infra cost, licensing, operations headcount |
| Security officer | Access control, network isolation, audit logs |

### When to Use

- Comparing or analyzing complex technical topics
- When the scope is wide and it is unclear where to start
- Supporting decisions involving multiple stakeholders

---

## 2. Knowledge Gap Loop (iterative collection cycle)

After the first collection round, **draft the report TOC, identify gaps, and run a second round** of collection.

### Purpose

A single collection round cannot fill every section. The Knowledge Gap loop
makes incomplete sections explicit and enables targeted follow-up collection.

### Procedure

```
1. Draft a TOC from the first round of collected information.
2. Evaluate the strength of evidence for each section:
   ✅ Sufficient — backed by 2+ independent sources (or pieces of evidence)
   ⚠️ Weak     — only 1 source, or no numbers / rationale
   ❌ Lacking  — no source, depends on speculation
3. Run a second round of targeted collection for the ⚠️/❌ sections (gap-specific search).
4. Re-evaluate after the second round.
   → If still ❌, mark "evidence insufficient" or "needs further verification" in the report.
```

### When to Use

- After the first round, when some sections feel evidence-thin
- When a comprehensive report is required (comparative analysis, trend research, audit reports, etc.)
- When the collection scope is clearly defined

---

## 3. Chain-of-Verification

After the report or analysis draft is done, **self-verify that each citation / claim matches the actual evidence**.

### Purpose

It is easy to distort or exaggerate information while writing. Chain-of-Verification
is the last line of defense for catching mismatches between claims and evidence
before final submission.

### Procedure

```
1. Complete the draft report.
2. For each factual claim:
   a. Identify the cited source (or evidence).
   b. Confirm the claim matches the actual content.
   c. On mismatch:
      - Adjust the claim to match the source.
      - For numerical errors, mark "[number requires verification: source says X, report says Y]".
3. Mark unsourced claims as "needs further verification" or remove them.
```

### Handling Unsupported Claims

| Status | Treatment |
|------|----------|
| Has evidence (cross-checked) | Keep |
| Has evidence (single source) | Note "single source, further verification recommended" |
| No evidence (speculation) | Mark "estimate" / "needs further verification" or remove |

### When to Use

- When writing a report citing external data (internet, documents, code)
- Before submitting comparative analysis with numerical / performance data
- Writing audit or review reports where accuracy matters

---

## 4. Source-agnostic Grounding Pass

**Verify that the sources in the references list actually exist and that their content matches the claims**.

### Purpose

Physically verify that references are reachable and that the content lines up
with the claim. Use WebFetch for internet URLs, Read for codebase paths, and
direct lookup for documents.

### Procedure

```
1. Extract sources from the references list.
2. Attempt to access each source (use the tool appropriate to the data source).
3. Classify the result:
   ✅ Confirmed   — access succeeded + content matches
   ⚠️ Uncertain   — access succeeded + content is unclear → mark "[needs verification]"
   ❌ Failed      — file missing, broken link, etc. → mark "[source unknown]"
4. If there are 15+ sources, sample-verify the top 5 (full verification is inefficient).
```

### When to Use

- Final-pass verification of report references
- Confirming the basis for claims that include numbers / performance data
- When summarizing codebase analysis results in a document

---

## Recommended Application Order

| Step | Methodology | Timing |
|------|--------|------|
| Pre-research | Perspective Mining | While planning collection |
| During research | Knowledge Gap loop | After the first round |
| Post-writing | Chain-of-Verification | After the draft is done |
| Pre-submission | Source-agnostic Grounding Pass | Before final submission |

Each methodology can be used independently and you may apply only a subset as needed.
