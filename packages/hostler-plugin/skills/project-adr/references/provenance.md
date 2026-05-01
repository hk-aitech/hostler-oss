# Provenance — Authoritative Sources for This Skill

This skill's conventions, checklists, and templates rest on the standards,
papers, and practitioner guides below. Grounding Pass passed on 2026-04-23.

## Tier 1 — Original / Academic

### Michael Nygard (2011)

*Documenting Architecture Decisions*.
- URL: https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions
- Core: First formalization of the 3-section Context / Decision / Consequences structure
- Adopted: Full Nygard template + 1–2 page length recommendation

### Zdun, Capilla, Tran, Zimmermann (2013)

*Sustainable Architectural Decisions*. IEEE Software 30(6).
- Core: Y-statement format + decision-sustainability concept
- Adopted: Y-statement template + the "accepting" field as A1 Fairy Tale defense

### Olaf Zimmermann (2023)

*How to create Architectural Decision Records (ADRs) — and how not to*.
- URL: https://www.ozimmer.ch/practices/2023/04/03/ADRCreation.html
- Core: Codifies 11 anti-patterns (Fairy Tale, Sales Pitch, Free Lunch Coupon, etc.)
- Adopted: 11-anti-pattern checklist (references/anti-patterns.md §A1–A11)

### Olaf Zimmermann (2025)

*Seven Architectural Decision Making Fallacies (and Ways Around Them)*.
- URL: https://ozimmer.ch/practices/2025/09/01/ADMFallacies.html
- Core: 7 fallacies (Blind Flight, Following the Crowd, etc.) + bonus AI over-confidence
- Adopted: 7-fallacy checklist + F7 Time Dimension mitigation (`review_due` field)

## Tier 2 — Community Standards

### MADR (Markdown Any Decision Records)

- Official site: https://adr.github.io/madr/
- Repo: https://github.com/adr/madr
- Latest: MADR 4.0.0 (released 2024-09-17)
- Adopted: Full + minimal MADR 4.0 templates (references/templates/madr.md)
- Previous name: Markdown **Architectural** Decision Records → **Any** (renamed in 2023)

### ADR.github.io (community hub)

- URL: https://adr.github.io/
- Content: Hub for ADR template comparisons, tooling catalogs, paper links
- Used: Basis for the comparison among the three templates (Nygard / MADR / Y-statement)

### joelparkerhenderson/architecture-decision-record

- URL: https://github.com/joelparkerhenderson/architecture-decision-record
- Content: Large repo of multi-language templates + practical examples
- Used: Reference for the Korean-language Nygard template

### adr-tools (Nat Pryce)

- URL: https://github.com/npryce/adr-tools
- Latest: v3.0.0 (2017-07-25, with later minor updates)
- Features: CLI-based ADR management, Graphviz visualization
- Used: Inspiration for the design of the `/project-adr supersede` mode

## Tier 3 — Enterprise / Cloud Guides

### AWS Prescriptive Guidance

*Using architectural decision records to streamline technical decision-making*.
- URL: https://docs.aws.amazon.com/prescriptive-guidance/latest/architectural-decision-records/welcome.html
- Core: ADR process + best practices + governance structure
- Adopted: Status lifecycle transition convention (proposed → accepted → superseded / deprecated)

### Microsoft Azure Well-Architected Framework

*Maintain an architecture decision record (ADR)*.
- URL: https://learn.microsoft.com/en-us/azure/well-architected/architect-role/architecture-decision-record
- Core: Architecturally Significant Requirements (ASR) concept
- Adopted: Gate 1 Q1 "Architecturally Significant?" criterion

### ThoughtWorks

*Lightweight technology governance*.
- URL: https://www.thoughtworks.com/en-de/insights/articles/lightweight-technology-governance
- Core: Folding ADRs into the Technology Radar Adopt category
- Related: Architecture Advice Process (decentralized decision-making)
- Adopted: Advice Process bypass route when Gate 1 Q1–Q5 do not all pass

### GDS Way (UK Government Digital Service)

*Documenting architecture decisions*.
- URL: https://gds-way.digital.cabinet-office.gov.uk/standards/architecture-decisions.html
- Core: Government-grade ADR standard (extension of Nygard)
- Used: Reference for public-sector operations

### TechTarget (2020, 2025 updated)

*8 best practices for creating architecture decision records*.
- URL: https://www.techtarget.com/searchapparchitecture/tip/4-best-practices-for-creating-architecture-decision-records
- Used: The "accepted is immutable" rule mentioned in the 2025 update

## Tier 4 — Anthropic Claude Code Skills v2.0

### Official Documentation

- URL: https://code.claude.com/docs/en/skills
- Captured: 2026-04-23
- Adopted:
  - Frontmatter v2 fields: `name` / `description` / `when_to_use` / `allowed-tools`
    / `user-invocable` / `argument-hint` / `paths`
  - Progressive disclosure (SKILL.md < 500 lines, the rest in references/)
  - Single-line description rule (avoid multi-line YAML wrapping)
  - 1536-char description cap

### Agent Skills Open Standard

- URL: https://agentskills.io
- Compatibility: Supports Claude Code extensions (invocation control, subagent execution)

### skill-creator SKILL.md Pattern

- URL: https://github.com/anthropics/skills/blob/main/skills/skill-creator/SKILL.md
- Used: Adopted the directory structure (SKILL.md + references/ + examples/) and the imperative + WHY style of writing

## Tier 5 — Other References

### Relationship-clarification Material

- *Documenting Design Decisions using RFCs and ADRs* (Bruno Scheufler, 2020)
  - https://brunoscheufler.com/blog/2020-07-04-documenting-design-decisions-using-rfcs-and-adrs
- *Engineering Planning with RFCs, Design Documents and ADRs* (Pragmatic Engineer)
  - https://newsletter.pragmaticengineer.com/p/rfcs-and-design-docs
- *ADRs and RFCs: Their Differences and Templates* (Candost's Blog)
  - https://candost.blog/adrs-rfcs-differences-when-which/

### Industry Practice

- Red Hat Developer Blog — *Why you should be using ADRs*
  - https://www.redhat.com/en/blog/architecture-decision-records
- Martin Fowler — *Scaling the Practice of Architecture, Conversationally*
  - https://martinfowler.com/articles/scaling-architecture-conversationally.html
- IcePanel Blog — *Architecture Decision Records (ADRs)*
  - https://icepanel.io/blog/2023-03-29-architecture-decision-records-adrs
- InfoQ — *Has Your Architectural Decision Record Lost Its Purpose?*
  - https://www.infoq.com/articles/architectural-decision-record-purpose/

## Grounding Pass Summary

| Source | URL verification | Numerical accuracy | Trust |
|------|---------|-----------|--------|
| Nygard 2011 | ✅ | ✅ | original |
| Zimmermann 2023 (how not to) | ✅ | ✅ 11 anti-patterns | original |
| Zimmermann 2025 (fallacies) | ✅ | ✅ 7 + 1 bonus fallacy | original |
| MADR 4.0 | ✅ | ✅ released 2024-09-17 | official |
| AWS Prescriptive | ✅ | ✅ | official |
| Azure WAF | ✅ | ✅ | official |
| Claude Code Skills v2 | ✅ (code.claude.com) | ✅ as of 2026-04 | official |

**Grounding verdict**: All Tier 1–2 original URLs were physically verified. Numbers (11 anti-patterns / 7 fallacies / MADR 4.0 / 2024-09-17) re-verified.

**General-purpose skill policy**: This skill is intended to ship as a **shared
package** — examples / terminology / workflows tied to a specific tool / CLI /
project directory layout are deliberately omitted. The ADR directory path is
designed to be controlled by the `$ADR_DIR` environment variable or project
convention, and only Conventional Commits examples are shown for commit
message conventions.
