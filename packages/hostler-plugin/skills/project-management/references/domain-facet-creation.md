---
uuid: 019dc434-f9b3-7c71-ba67-376ef36de111
title: "Custom domain-facet registration guide — work on the user project side"
type: reference
related_skill: context-engineering
---

# Custom domain-facet registration guide

> The procedure for a user project to register its own domain facets on top of hostler core. Layer 1~3 are all possible, but the standard for domain-specific facets is to add them as Layer 3 plugins.

## Table of contents

1. Where to register — Layer 1~2 (yaml) vs Layer 3 (plugin)
2. Layer 1~2 yaml registration procedure
3. Layer 3 plugin registration procedure (stdio JSON-RPC v2.0)
4. Verification — `hstl-oss context --facets` e2e
5. User project case studies (dogfood)

## 1. Decide where to register

| Domain facet character | Recommended Layer | Registration location |
|------------------------|-------------------|-----------------------|
| Project identity / policy (months–years) | Layer 1 | `project.facets` section in `.hstl-oss/project-config.yaml` |
| Progress state (days–weeks) | Layer 2 | the same yaml + `task-frontmatter` or `subprocess` resolver |
| External state / OS-level snapshot (minutes–days) | **Layer 3 plugin** | `.hstl-oss/plugins/<name>/manifest.yaml + plugin.sh` |

Domain-specific facets are almost always Layer 3. Layer 1/2 are usually fine with hostler core's 12 standard facets.

## 2. Layer 1~2 yaml registration

Add to the `project.facets` section of `.hstl-oss/project-config.yaml`:

```yaml
project:
  key: "your-project-key"
  facets:
    # core facet override — change the default source path
    core.objective:
      source: "file:my-vision/mission.md"
      layer: 1

    # New Layer 2 facet (domain-specific progress state)
    your.deployment_status:
      source: "subprocess:kubectl get deploy -o json"
      layer: 2
      refresh: "5m"
      description: "Current deployment state"
```

source resolver prefix definitions: `12-facets-table.md` §5 (single SSOT). This section covers registration examples only.

## 3. Layer 3 plugin registration (stdio JSON-RPC v2.0)

### 3-1. Directory structure

```
.hstl-oss/plugins/<plugin-name>/
├── manifest.yaml      ← plugin metadata + facets list
└── plugin.sh           ← entry executable (or Python / Go binary)
```

### 3-2. manifest.yaml schema

```yaml
name: your-broker-api
version: 0.1.0
entry: ./plugin.sh
timeout: 10s
facets:
  - plugin.your.broker_api
  - plugin.your.market_hours
```

### 3-3. plugin.sh (bash example)

```bash
#!/bin/bash
set -uo pipefail

# read 1 line from stdin (JSON-RPC request)
read -r REQ || true

# Collect domain state (e.g. OS-level snapshot via pgrep)
state_count=$(pgrep -f "your-server" 2>/dev/null | wc -l | tr -d ' \n')
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# content payload — string embedded in JSON, escape newlines as \n
content="state @ ${timestamp}\nactive: ${state_count}"

# JSON-RPC response (single line, id=1 fixed)
printf '{"jsonrpc":"2.0","id":1,"result":{"content":"%s","source":"your-broker-api v0.1.0"}}\n' "$content"
```

### 3-4. Register the facet in project-config.yaml

```yaml
project:
  facets:
    plugin.your.broker_api:
      source: "plugin:your-broker-api"
      layer: 3
      refresh: "60s"
      description: "broker market state"
```

### 3-5. Add a .gitignore whitelist (don't forget)

If `.hstl-oss/*` is gitignored, the plugin files won't be tracked:

```
!.hstl-oss/plugins/
```

## 4. Verification — e2e

### 4-1. Run the plugin directly (verify JSON response)

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"facet.resolve"}' | .hstl-oss/plugins/your-broker-api/plugin.sh
```

Expected: a single-line JSON `{"jsonrpc":"2.0","id":1,"result":{"content":"...","source":"..."}}`.

### 4-2. Through hstl-oss context

```bash
hstl-oss-oss context --facets plugin.your.broker_api -o json | jq '.data.values[0]'
```

Expected: `{name, layer: 3, content, source: "plugin:your-broker-api", resolved_at}`.

### 4-3. Regression (unit tests)

The hostler-plugin's plugin SDK itself doesn't change — only the user project's plugin is added — so user-project-side e2e is sufficient for regression.

## 5. User project case studies (dogfood)

### 5-1. broker-api facet example

A trading-domain user project exposes broker-call results as a Layer 3
plugin facet — `plugin.<project>.broker_api`. Domain-specific data (price,
balance, market hours, etc.) is converted to standard JSON by plugin.sh so
hostler recognizes it as a facet.

### 5-2. dev-server-state facet example

A web-app user project exposes vite / tsc --watch / go test progress as a
Layer 3 plugin — `plugin.<project>.dev_server_state`. The plugin.sh
example in this guide was extracted from this pattern. User projects can
register their own domain facets with the same pattern.

## 6. Authoring caveats — known traps

### 6-1. Avoid multi-line bash output

The pattern `pgrep -fc "..." || echo 0` produces "0\n0" multi-line output when there are 0 matches, which breaks the single-line JSON-RPC response. Use the **`pgrep -f | wc -l | tr -d ' \n'`** pattern (guarantees a single line).

Details: hostler-plugin output KB W001 (`docs/07-knowledge/workflow/plugin-sdk.md` in the user project — a hostler-plugin-based KB card).

### 6-2. validator rejecting unknown properties

The `project.facets` section of `.hstl-oss/project-config.yaml` was rejected by the validator in a regression (caught during dogfood). hostler core has since corrected this (`pkg/config/types.go` `ProjectMeta.Facets map[string]any`).

### 6-3. Separate the plugin SDK and the first dogfood Sprint

Don't bundle the new plugin SDK itself with the first dogfood (real plugin authoring + e2e) in the same Sprint. Splitting "SDK Sprint → dogfood Sprint" naturally surfaces SDK defects + lets you fix them inside the dogfood Sprint.

Details: hostler-plugin output KB A023 (`docs/07-knowledge/architecture/patterns.md` in the user project).
