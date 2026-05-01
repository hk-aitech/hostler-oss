---
id: ADR-001
title: Open-source extraction of Sprint / Task workflow toolkit
status: Accepted
date: 2026-04-30
deciders: [hyunkim]
---

# ADR-001 — Open-source extraction of Sprint / Task workflow toolkit

## Context

A Sprint / Task workflow toolkit (Claude Code plugin + Go CLI) was originally developed inside a private monorepo, where it co-existed with components specific to that environment: cryptographic signing, multi-agent orchestration, persona-aware tooling, cross-domain registries, and faceted project-context layers.

The workflow components themselves — Sprint and Task lifecycle, knowledge-base capture, ADR conventions, harness gating, and the standard `docs/` / `works/` layout — are general-purpose and useful outside that environment. The remaining components are not: they assume infrastructure, conventions, and personas that do not transfer.

## Decision

Extract the general-purpose workflow toolkit into a standalone open-source repository, **`hostler-oss`**, under the **Apache License 2.0**, with the following shape:

- A monorepo containing two coupled packages: a Claude Code plugin (`packages/hostler-plugin`) and a Go CLI (`packages/hostler-cli`).
- The plugin namespace is `hstl-oss:` and the CLI binary is `hstl-oss`.
- Only the file-based workflow surface is migrated. Cryptographic signing, multi-AI / daemon orchestration, universal feature catalogues, cross-domain trace registries, multi-track program management, and faceted context layers are **deliberately excluded**.
- Documentation, skills, and commands are sanitised so the OSS release does not reference the originating private environment beyond an abstract acknowledgement of origin.

## Consequences

### Positive

- A focused OSS surface that is easy to adopt without private infrastructure.
- Clear scope boundary: the project does one thing (file-based Sprint / Task workflow) and does it without leaking organisation-specific assumptions.
- Apache 2.0 provides explicit patent grants and notice obligations, which improves enterprise adoptability.

### Negative

- Features that some users may want (signing, multi-agent orchestration, cross-track planning) are unavailable here and must be provided externally if needed.
- Two parallel implementations exist for some time: the originating private toolkit and the OSS extraction. They will diverge.

### Neutral

- The two packages are versioned together inside this monorepo. Splitting into separate repositories may be reconsidered later if their lifecycles diverge significantly.

## Alternatives Considered

1. **Mirror the entire originating toolkit verbatim.** Rejected — exposes private-environment assumptions and creates a maintenance burden for users who do not need them.
2. **Publish only the CLI, omit the plugin.** Rejected — the plugin is a primary differentiator and the value of the workflow comes from the pairing.
3. **Two separate repositories from day one.** Rejected for now — the packages are tightly coupled in their first releases; the monorepo simplifies coordinated changes.
