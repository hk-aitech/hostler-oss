---
title: Roadmap
status: draft
version: 0.1.0
last_updated: 2026-04-30
---

# Roadmap

The initial extraction is broken into eight sprints (S0–S7). Each sprint produces a verifiable artefact.

| Sprint | Goal | Exit criteria |
|---|---|---|
| **S0** | Repository skeleton | This document, `LICENSE`, `NOTICE`, `README.md`, `docs/` tree, `works/` tree, `go.work`, `.gitlab-ci.yml`, ADR-001 committed and pushed. |
| **S1** | CLI source extraction | `packages/hostler-cli/` populated; out-of-scope packages removed; `go build ./...` passes. |
| **S2** | CLI test cleanup | Tests for excluded domains deleted; `go test ./...` passes. |
| **S3** | Plugin extraction & namespace rename | `packages/hostler-plugin/` populated; out-of-scope commands / skills / agents removed; namespace migrated to `hstl-oss:`; binary references updated to `hstl-oss`. |
| **S4** | Sanitisation pass | Skills / commands / docs cleansed of references to the originating private toolkit; persona-specific skills rewritten for a single-owner project. |
| **S5** | Frontmatter schema strip | Out-of-scope frontmatter fields removed across templates, validators, and existing documents. |
| **S6** | E2E smoke test | Full lifecycle (`project init → sprint → task → kb`) runs end-to-end with the OSS build. |
| **S7** | OSS polish | `README.md` finalised, `CONTRIBUTING.md` added, CI green, first tagged release prepared. |

## Beyond S7

- Tagged release `v0.1.0`.
- Public mirror (optional).
- Issue / MR templates.
- Community guidelines.
