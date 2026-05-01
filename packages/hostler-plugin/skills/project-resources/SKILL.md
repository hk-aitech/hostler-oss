---
name: project-resources
description: |
  Allocates project-wide resources (ports, EventId ranges, DNS records,
  external API endpoints) through hostler CLI registry commands so they
  don't conflict. Use this skill when the user mentions "assign a port",
  "reserve an EventId range", "register a DNS record", "register an
  external API endpoint", "prevent resource conflicts", or any equivalent
  phrasing — even when they don't explicitly ask for the registry.
  Trigger any time someone is about to pick a port number, EventId,
  hostname, or external API endpoint that other components might also
  use. Do NOT use for in-source variables or constants — that's regular
  code work.
compatibility:
  tools: [Read, Bash]
trigger_commands: ["registry.*"]
---

# Project Resources

Centrally manages project-wide resources (ports, EventId ranges, DNS records,
external API endpoints) to prevent conflicts.

**SSOT**: `~/{brand.ProjectDirName}/data/{HOSTLER_PROJECT}/registry.json`
(auto-synced, isolated by the HOSTLER_PROJECT environment variable; on a fork
`.hostler` automatically switches to `.<new brand>`)

## CLI usage

Inspect commands and parameters via the manifest:

```bash
hstl-oss-oss --manifest --group registry
```

## Path configuration

The registry.json location is resolved through a **3-level priority** chain:

1. **Environment variables** `HSTL_REGISTRY_PATH` (runtime) / `HSTL_REGISTRY_DOCS_PATH` (docs snapshot) — official override
2. **`registry` field in `.hostler/project-config.yaml`**:
   ```yaml
   registry:
     path: /custom/runtime/registry.json        # runtime path override
     docs_path: /custom/docs/registry.json      # git snapshot path override
   ```
3. **Default** — when nothing is configured, derived from brand:
   - Runtime: `~/{brand.ProjectDirName}/data/{HOSTLER_PROJECT}/registry.json`
   - Docs: `{project_root}/{brand.ProjectDirName}/data/registry.json`

**Note**: `HSTL_REGISTRY_PATH` had its "test-only" restriction lifted and is
promoted to an official override.

---

## When This Skill Triggers

| Request | Action |
|---------|--------|
| Show allocations | `hstl-oss registry list` |
| Check conflict | `hstl-oss registry check` |
| Allocate new | `hstl-oss registry allocate` |
| Sync file | `hstl-oss registry sync` |

Triggers: "assign a port", "what port should I use", "check port conflict",
"show all ports", "reserve EventId range", "what EventId range to use",
"pick a LoggerMessage number", "add DNS record", "register external API",
"list resource allocations", "look up the registry", "allocate a port for a new service"

---

## Standard 1.0 catalog (4 types)

| ID | kind | Purpose | Applies to |
|----|------|---------|------------|
| `ports` | unique | Service / process TCP/UDP ports | Any server-style project |
| `logger_event_ranges` | range | Static number ranges for LoggerMessage / EventId | .NET-centric, projects using EventId pattern |
| `dns_records` | unique | hostname ↔ IP / endpoint | Any project |
| `external_apis` | unique | External API endpoint URL / auth identifier | Any project |

> Extension candidates (`dapr_app_ids`, `pg_databases`, `kafka_topics`, `vip_addresses`, etc.) can be created dynamically. See `references/resource-types-catalog.md` for details.

---

## Port Selection Guide

When adding a port for a new service:

1. **Check current state**: `hstl-oss registry list --type ports`
2. **Check candidate port for conflicts**: `hstl-oss registry check --type ports --value XXXX`
3. **Follow existing per-environment patterns** (prod / dev / localdev)
4. **Register each env separately**: run `hstl-oss registry allocate` once per prod / dev / localdev

Recommended owner naming: `{Component}[.Subcomponent]` (e.g. `Collection.Host`, `web-frontend`, `worker-1`).

---

## EventId Range Selection Guide

When reserving an EventId range for a new module:

1. **Check current state**: `hstl-oss registry list --type logger_event_ranges`
2. **Find unallocated gaps**: look for empty stretches between existing BCs' range_start/range_end (1000-unit gaps recommended)
3. **Conflict check**: `hstl-oss registry check --type logger_event_ranges --range-start X --range-end Y`
4. **Allocate**: `hstl-oss registry allocate --type logger_event_ranges --owner NewBC --range-start X --range-end Y`

---

## Scope Boundaries

| Action | Owning skill |
|--------|--------------|
| Allocate ports / DNS / API / EventId | project-resources |
| Define in-source variables / constants | (out of scope — regular code work) |

---

## Detailed references

- 1.0 catalog (5 types) + 4 extension candidates: `references/resource-types-catalog.md`
- Real-world port allocation scenarios: `examples/port-allocation.md`
- Real-world EventId range scenarios: `examples/eventid-range.md`
- Trigger validation: `evals/trigger-eval.json`
- Path-config implementation rationale: `cli/pkg/config/types.go` RegistryConfig
- Change history (maintainers only): `references/changelog.md`

## References

- Registry SSOT: `~/{brand.ProjectDirName}/data/{HOSTLER_PROJECT}/registry.json` (auto-synced)
- CLI manifest: `hstl-oss --manifest --group registry`
- Design doc: user project docs
- Upstream ADR §4.2 ConfigProvider port — Registry path methods
