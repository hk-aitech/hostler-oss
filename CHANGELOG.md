# Changelog

All notable changes to **hostler-oss** are recorded here. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

(none)

## [0.1.0] — 2026-04-30

Initial extraction. The toolkit was previously developed inside a private monorepo and is now published as a focused, single-persona open-source release.

### Added

- **`packages/hostler-cli`** — the `hstl-oss` Go CLI. Manages the file-based state for sprints, tasks, backlog, registry, audit log, ceremony artefacts, and the harness gate.
- **`packages/hostler-plugin`** — the Claude Code plugin (namespace `hstl-oss:`) shipping 7 command groups, 20 user-invocable skills, 4 agents, and 3 session hooks (session context briefing + Agent-tool subagent context-ack injection + pre-commit check).
- Standard project layout under `docs/` (00–08 numeric prefixes), `works/`, `archive/`, with the [`project-structure`](packages/hostler-plugin/skills/project-structure) skill as SSOT.
- ADR-001 — open-source extraction rationale.
- Smoke E2E lifecycle report at `docs/06-reports/spikes/smoke-e2e-lifecycle.md`.
- Apache License 2.0, NOTICE, README, CONTRIBUTING, CI workflows (`go vet` / `go build` / `go test`).
- GitHub Actions release workflow that cross-compiles for `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64` on `v*` tags and publishes binaries plus `install.sh` to GitHub Releases.
- One-line installer: `curl -fsSL https://github.com/hk-aitech/hostler-oss/releases/latest/download/install.sh | sh`.
- `.claude-plugin/marketplace.json` so Claude Code can register the repo as a plugin marketplace.
- `SECURITY.md`, `CODE_OF_CONDUCT.md`, issue / PR templates, and Dependabot configuration.

### Removed (intentionally — out of OSS scope)

- Cryptographic signing (HMAC, GPG, signed documents).
- Multi-AI orchestration, daemon-based worker coordination, message-bus integrations.
- Universal feature catalogues, cross-domain trace registries, multi-track program management.
- Faceted project-context layers.
- Persona-specific tooling beyond a single project owner.

### Notes

- Public CLI surface and plugin command surface may shift before `v1.0.0`. Breaking changes will be announced in the release notes.
- Plugin skill bodies and references are English-only.

[Unreleased]: https://github.com/hk-aitech/hostler-oss/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/hk-aitech/hostler-oss/releases/tag/v0.1.0
