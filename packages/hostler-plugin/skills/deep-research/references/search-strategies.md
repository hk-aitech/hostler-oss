# Search strategies reference

> Reference for shaping per-topic search queries in Step 2 (search strategy) of the deep-research skill.
> Provides keyword construction patterns and query examples per research type.

---

## Search strategies by research type

### 1. Technology Comparison

**Goal**: compare two or more technologies / libraries / frameworks to back a selection decision.

**Query patterns:**

| Stage | Pattern | Example |
|-------|---------|---------|
| Direct comparison | `{A} vs {B} comparison 2025` | `kafka vs redis streams comparison 2025` |
| Benchmark | `{A} {B} benchmark performance` | `kafka redis benchmark throughput latency` |
| Use cases | `{A} use cases production` | `kafka use cases real-time streaming` |
| Limitations / cons | `{A} limitations drawbacks` | `kafka limitations small team` |
| Migration | `migrate from {A} to {B}` | `migrate from rabbitmq to kafka` |
| Locale supplement | `{A} vs {B} comparison pros cons` | `kafka redis comparison local-language` |

**Recommended searches**: 4–6.

**Trusted source types:**
- Official benchmark pages
- ThoughtWorks Technology Radar
- InfoQ architecture articles
- GitHub README and issues (real user experience)

---

### 2. Competitive Analysis

**Goal**: analyze similar systems / products to surface internal strengths and weaknesses and derive improvements.

**Query patterns:**

| Stage | Pattern | Example |
|-------|---------|---------|
| Market overview | `{domain} open source tools 2025` | `algorithmic trading open source tools 2025` |
| Top players | `top {domain} platforms` | `top stock screening platforms` |
| Feature comparison | `{domain} software comparison features` | `trading system comparison features` |
| GitHub mining | `{keyword} site:github.com stars:>1000` | `stock screener site:github.com` |
| Papers / research | `{domain} architecture research paper` | `real-time stock detection architecture` |
| Locale-specific | `{domain} {locale} {keyword}` | `anomaly-detection system local case` |

**Recommended searches**: 5–8.

**Subject selection criteria:**
- 1,000+ GitHub stars
- Commit activity within the last 6 months
- Similar use-case documentation exists
- Minimum 3, recommended 5–7

---

### 3. Trend Research

**Goal**: capture the latest movement in a tech area and forecast direction.

**Query patterns:**

| Stage | Pattern | Example |
|-------|---------|---------|
| Latest trends | `{domain} trends 2025 2026` | `real-time analytics trends 2026` |
| Major conferences | `{conference} {topic} 2025` | `KubeCon observability 2025` |
| Industry report | `{domain} industry report 2025` | `fintech AI report 2025` |
| Emerging tech | `emerging {domain} technology` | `emerging stream processing technology` |
| Adoption | `{technology} adoption survey 2025` | `CQRS adoption survey enterprise` |
| Locale trends | `{domain} trends {year}` | `fintech AI trends 2026` |

**Recommended searches**: 4–6.

**Trusted sources:**
- Gartner Hype Cycle
- ThoughtWorks Technology Radar
- Stack Overflow Developer Survey
- CNCF Annual Survey

---

### 4. Academic Research

**Goal**: confirm the academic basis of a specific algorithm / model / methodology and capture latest research direction.

**Query patterns:**

| Stage | Pattern | Example |
|-------|---------|---------|
| Paper search | `{topic} paper arxiv 2024 2025` | `anomaly detection financial time series arxiv` |
| Benchmark paper | `{method} benchmark dataset performance` | `lightgbm stock prediction benchmark` |
| Survey paper | `survey {topic} machine learning` | `survey stock price prediction deep learning` |
| Implementation | `{algorithm} implementation github` | `LSTM stock prediction pytorch github` |
| Specific journal | `site:arxiv.org {keyword}` | `site:arxiv.org stock surge detection` |
| Locale academic | `{keyword} paper local-repository` | `stock surge prediction machine learning paper` |

**Recommended searches**: 3–5.

**Major academic sources:**
- arXiv (cs.LG, q-fin)
- Papers With Code (includes implementations)
- Google Scholar
- IEEE Xplore, ACM Digital Library

---

## Search quality checklist

After every search round, confirm:

```
□ Recency: are sources from 2024–2026? (older ones need a follow-up search)
□ Diversity: 3+ independent sources, no single-source bias?
□ Trustworthiness: official docs, academic papers, reputable tech blogs?
□ Relevance: directly comparable with the internal-system context?
□ Bilingual: searched in multiple languages where applicable?
```

---

## Structuring search results

Capture collected information in this format:

```markdown
### {System / technology name}

- **Source**: {URL}
- **Latest version**: {version}, {date}
- **GitHub Stars**: {number} (when applicable)
- **Key features**: {3–5 bullets}
- **Architecture**: {language}, {pattern}, {core design}
- **Results / benchmarks**: {numbers} (when available)
- **Vs. internal system**: {strengths / weaknesses}
```

---

## Related references

- Report structure template: `references/report-template.md`
- deep-research skill workflow: SKILL.md
