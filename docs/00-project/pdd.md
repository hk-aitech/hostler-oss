---
title: Project Definition Document
status: draft
version: 0.1.0
last_updated: 2026-04-30
---

# Project Definition Document — hostler-oss

## 1. Purpose

`hostler-oss` packages a Sprint / Task workflow toolkit as an open-source release. It pairs a Claude Code plugin (slash commands, skills, agents, session hooks) with a Go CLI that owns the on-disk state. The two are designed to work together: the plugin orchestrates the Claude Code experience, the CLI persists and validates artefacts.

## 2. Origin

This project is extracted from a private internal toolkit. The extraction keeps only the workflow components that are useful as a generic project-management aid for teams using Claude Code, and discards features that depend on private infrastructure or organisation-specific personas.

## 3. In Scope

- File-based **Sprint lifecycle** (create / start / progress / complete) with ceremony artefacts.
- File-based **Task lifecycle** (backlog → todo → in-progress → done), with assignment, estimate, dependency, and harness gating.
- **Knowledge base** capture (lessons / gotchas) tied to Sprint completion ceremonies.
- **Harness gate** — lightweight automated checks that gate Task / Sprint completion.
- **Project structure scaffolding** — initialise the standard `docs/`, `works/`, `archive/` layout.
- **ADR support** — templates and conventions for Architecture Decision Records.
- **Hotfix workflow** — branching off `main` and back-merge guidance.
- **Audit log** — append-only record of CLI-driven state transitions.
- **Plugin namespace `hstl-oss:`** — slash commands, skills, agents.

## 4. Out of Scope (Non-Goals)

The following are intentionally **excluded** from this project:

- Cryptographic signing of work artefacts (HMAC, GPG, signed documents).
- Multi-AI orchestration, daemon-based worker coordination, message-bus integrations.
- Universal-feature catalogues, cross-domain trace registries, multi-track program management.
- Domain-faceted project context layers.
- Persona-specific tooling beyond a single project owner.

These were present in the originating internal toolkit and have been removed deliberately to keep the OSS surface focused, auditable, and easy to adopt.

## 5. Stakeholders

- **Maintainer**: project owner.
- **Users**: developers using Claude Code who want a lightweight, file-based, opinionated Sprint / Task workflow.
- **Contributors**: external contributions accepted under Apache License 2.0.

## 6. Success Criteria

1. A clean `git clone` followed by `go build ./...` produces a working `hstl-oss` binary.
2. Installing the plugin in Claude Code exposes the `hstl-oss:` slash commands without errors.
3. The end-to-end smoke flow (`project init` → `sprint create / start` → `task create / start / complete` → `sprint complete`) runs without manual intervention.
4. Documentation under `docs/` is sufficient for a new user to reach the smoke flow without reading source code.

## 7. Constraints

- License: **Apache License 2.0**.
- Go toolchain: **1.23+**.
- Plugin runtime: **Claude Code** (current stable).
- No runtime dependency on private networks, message buses, or sign-in services.

## 8. Glossary

See [`domain-glossary.md`](domain-glossary.md) (stub).
