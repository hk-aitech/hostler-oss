# Security policy

## Supported versions

`hostler-oss` is currently in the `v0.x` line. While the project is pre-1.0, security fixes land on the latest minor release only. Once v1.0 ships, this table will be updated to reflect formal LTS commitments.

| Version | Supported |
|---|---|
| latest `v0.x` | ✅ |
| older `v0.x` | ❌ |

## Reporting a vulnerability

**Do not open a public GitHub issue for security reports.**

Please use one of the two private channels:

1. **GitHub Security Advisories** — go to the repository's Security tab → "Report a vulnerability". This opens a private discussion thread with the maintainers and is the preferred path.
2. **Email** — send to the maintainer email listed in `packages/hostler-plugin/.claude-plugin/plugin.json`. Include a clear reproduction, the affected version, and the impact you observed.

Please do not send unencrypted reproduction artefacts that exfiltrate data from third-party systems. A textual reproduction is enough for triage.

## What to expect

- **Acknowledgement**: within 5 business days.
- **Triage + plan**: a coordinated-disclosure timeline (usually 30–90 days depending on severity) will be agreed within 14 days.
- **Fix + release**: a patched version is published with a corresponding GitHub Security Advisory. The advisory credits the reporter unless they request anonymity.
- **CVE**: requested for any vulnerability rated High or above (CVSS 7.0+).

## Scope

In scope:

- The `hstl-oss` CLI (`packages/hostler-cli/`) — supply-chain, command injection, path traversal, audit-log integrity, SQLite use, file-state state-machine bypass.
- The Claude Code plugin (`packages/hostler-plugin/`) — hook scripts, slash-command shell-out boundaries, agent prompt construction, session-context exposure.
- The `install.sh` and the GitHub Actions release workflow.

Out of scope:

- Issues that require a malicious project-config.yaml or rules.yaml that the user has authored themselves (we treat the project root as a trusted boundary).
- Issues in third-party tools the CLI shells out to (`git`, `go`, the user's shell).
- Findings derived from the user running `go install` or `make install` from an untrusted fork.

## Coordinated disclosure

We follow a 90-day coordinated-disclosure default. The window can be shortened (severe + actively-exploited issues) or extended (low-severity issues with a complex fix) by mutual agreement.

Thank you for helping keep the project safe.
