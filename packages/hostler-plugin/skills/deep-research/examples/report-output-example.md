# Output examples — Deep Research report structure

This file shows the **completed output format** of a report produced by the deep-research skill.
Final reports are stored under `docs/08-references/research/`.

---

## Step 1.5 output — Plan Clarification

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Research plan — MCP Tool AI-friendliness design patterns
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Perspectives: protocol design / implementation frameworks / academic / real-world cases
  Expected systems to study: 7
  Expected search queries: 10

  Query list:
  1. "MCP tool description best practices 2025" — protocol standards
  2. "MCP server instructions field usage" — server-level guidance
  3. "MCP tool annotations readOnlyHint" — tool metadata
  4. "agent-aware MCP response patterns" — response design
  5. "MCP tool description smell arXiv" — academic research
  ...

  Trusted Domains: modelcontextprotocol.io, arxiv.org, github.com

  "proceed" → start research
  "revise"  → adjust queries / perspectives
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Step 2.5 output — Perspective Mining

```
Perspective 1: Protocol designer — MCP spec's AI self-guidance mechanisms
Perspective 2: Tool developer — practical description/schema authoring
Perspective 3: AI agent — tool selection / error recovery viewpoint
Perspective 4: Academic researcher — Tool Description Smell, benchmarks
Perspective 5: Framework comparison — FastMCP vs official SDK vs custom
```

## Step 3.5 output — Knowledge Gap evaluation

```
Outline draft vs evidence assessment:
  §1 Research purpose     ✅ sufficient
  §2 MCP spec analysis    ✅ sufficient (official docs + 3 blogs)
  §3 Industry patterns    ✅ sufficient (2 academic + 3 real-world)
  §4 Self assessment      ✅ sufficient (direct code analysis)
  §5 Improvement proposals ⚠️ weak — outputSchema real-world examples lacking
  §6 Comparison           ✅ sufficient
  §7 Conclusion           ✅ sufficient

→ 2nd round search: "MCP outputSchema structuredContent implementation example"
```

## Step 7 output — self-quality evaluation

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Report self-evaluation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [RACE] Depth: 4 | Breadth: 5 | Following: 5 | Readability: 4 = 18/20
  [FACT] Sources: 15 (pass) | Accuracy: 4/5 | Recency: 87% (pass) | Cross: pass
  Grade: A (excellent)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Step 2 output example — 10 search queries ("real-time streaming data ingestion architecture")

| # | Type | Query | Purpose |
|---|------|-------|---------|
| 1 | Core (English) | `real-time streaming data pipeline architecture 2025` | Latest architecture trends |
| 2 | Locale variant | `real-time data ingestion system architecture 2025` | Locale-specific cases |
| 3 | Competing system | `Apache Flink streaming architecture` | Stream-processing competitor |
| 4 | Competing system | `Apache Kafka Streams data feed architecture` | Open-source alternative |
| 5 | Open source | `open source real-time data collector GitHub stars:>500` | Active project discovery |
| 6 | Benchmark | `WebSocket vs gRPC streaming data throughput benchmark` | Protocol comparison |
| 7 | Trend | `event-driven streaming system Kafka vs Redis Streams 2026` | Messaging comparison |
| 8 | Academic | `low-latency data processing actor model paper` | Theoretical grounding |
| 9 | Real-world | `production data collection service architecture lessons learned` | Operational experience |
| 10 | Incident | `streaming data system outage postmortem` | Failure-mode learning |

## Step 6 output example — report skeleton ("real-time streaming")

```markdown
# Real-time Streaming Data Ingestion Architecture — Deep Research Report

## 1. Research purpose
Compare our WebSocket + REST hybrid ingestion architecture against competing systems
and recent trends to surface improvement opportunities.

## 2. Subjects (5 systems)
| System | Type | Language | Notes |
|--------|------|----------|-------|
| Apache Flink | OSS | Java | Stateful stream processing, high throughput |
| Apache Kafka Streams | OSS | Java | Message queue + stream processing in one |
| Redpanda + Materialize | Infra | Rust/SQL | Stream processing pipeline |
| Vector (Datadog) | OSS | Rust | Lightweight data pipeline |
| Internal system | Self | (project language) | (summary of internal architecture) |

## 3. Recent trends (2025–2026)
- ...

## 4. Comparative analysis
### Comparison matrix
| Item | Internal | Flink | Kafka Streams | ... |

### Internal strengths / weaknesses

## 5. Improvement proposals
| # | Proposal | Phase | Rationale |
|---|----------|-------|-----------|
| 1 | ... | immediate | ... |

## 6. Conclusion

## 7. References (Sources)
[1] URL ...
```

---

## Reference reports

Completed research reports:
- `docs/08-references/research/mcp-tool-ai-usability-research.md` — MCP AI-friendliness (315 lines, 15 sources)
- `docs/08-references/research/deep-research-skill-competitive-analysis.md` — Deep Research competitive analysis (315 lines, 25 sources)
