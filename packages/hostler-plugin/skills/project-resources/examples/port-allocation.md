# Port Allocation Scenarios / Examples

Hands-on examples for registering a port for a new service. All commands invoke the `hstl-oss registry` CLI.

---

## Scenario 1: Register a new service `OrderService`

A new BC, `OrderService.Host`, needs a port to listen on across all three environments — prod / dev / localdev.

### Step 1. Inspect the current port catalog

```bash
hstl-oss-oss registry list resource_type="ports")
```

Response (summarized):
```
| owner                  | value | env       | note                  |
| Collection.Host        | 5100  | prod      | Group 3 periodic collection |
| Strategy.Host          | 5200  | prod      | ...                   |
| Trading.Host           | 5300  | prod      | ...                   |
...
```

→ Within the 5xxx band, the 5700s are free.

### Step 2. Check the candidate port for conflicts

```bash
hstl-oss-oss registry check resource_type="ports", value=5700)
```

Response: `{"available": true, ...}` or, on conflict, a `conflicts` array.

### Step 3. Allocate per environment

Call once per env (prod, dev, localdev):

```bash
hstl-oss-oss registry allocate 
    resource_type="ports",
    owner="OrderService.Host",
    value=5700,
    env="prod",
    note="orders domain — docker-svc-01:5700"
)
hstl-oss-oss registry allocate 
    resource_type="ports",
    owner="OrderService.Host",
    value=5700,
    env="dev",
    note="orders dev environment"
)
hstl-oss-oss registry allocate 
    resource_type="ports",
    owner="OrderService.Host",
    value=5700,
    env="localdev",
    note="developer local"
)
```

→ Same `value` (5700), but different `env`, so there's no conflict.

### Step 4. Verify

```bash
hstl-oss-oss registry list owner="OrderService.Host")
```

Confirms all three entries are registered.

---

## Scenario 2: Conflict and resolution

5500 is already allocated to Lite.Host (prod), and a new service requests 5500.

```bash
hstl-oss-oss registry allocate 
    resource_type="ports",
    owner="NewService.Host",
    value=5500,
    env="prod",
    note="new service"
)
```

Response:
```json
{
  "status": "REJECTED",
  "reason": "conflict",
  "conflicts": [
    {"owner": "Lite.Host", "value": 5500, "env": "prod", "note": "Edge UI"}
  ]
}
```

Response:
1. Re-list current state → find an available range
2. Re-request with a different value (e.g. 5710)

```bash
hstl-oss-oss registry check resource_type="ports", value=5710)
# available: true
hstl-oss-oss registry allocate resource_type="ports", owner="NewService.Host",
                  value=5710, env="prod", note="new service")
```

---

## Scenario 3: Reclaiming a port (removing an allocation)

The current `hstl-oss registry` CLI is add-only by design. Explicit reclamation
(remove) is done by directly editing `~/.hostler/data/{project}/registry.json`
to delete the entry. Recommended procedure:

1. Record the reason in an audit log before removal (manual)
2. Edit registry.json → remove the allocation
3. Call `hstl-oss registry sync` and verify docs are regenerated (auto-sync runs only on add)

The intentional design decision: reclamations are rare, and the add-only pattern keeps conflict detection safe.

---

## Recommendations

- **Specify HOSTLER_PROJECT explicitly**: set it in `.mcp.json` env so the project that owns the registry is visible
- **owner naming convention**: keep `{Component}[.Subcomponent]` or `{service-name}` consistent
- **Use `note` actively**: capture deployment target, group, etc. so origins surface quickly during ops/debugging
- **Port band convention**: define per-project bands to reduce conflicts (e.g. 5100~5400 = prod backend, 5500~5599 = edge UI)
