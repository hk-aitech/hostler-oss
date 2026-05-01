# Resource Types Catalog (project-resource 1.0)

The standard catalog of resource types recognized by the project-resource skill.
The hostler CLI's `registry` subcommand can add new types dynamically, but the
four standard types in this catalog are interpreted with the same meaning across
every project.

---

## Standard 1.0 catalog (4 types)

### 1. `ports` (kind: unique)

**Purpose**: TCP/UDP port number that a service / process listens on.

**Required fields**:
- `value` (int): port number
- `owner` (string): service identifier (e.g. `Collection.Host`, `web-frontend`)

**Recommended fields**:
- `env` (string): `prod` | `dev` | `localdev` (independently allocated per environment — same owner with different env counts as a separate allocation)
- `note` (string): operational note (e.g. "Group 3 periodic collection — docker-core-01:5100")

**Conflict rule**: only one entry per (resource_type, value, env) tuple is allowed.

**Example**:
```python
hstl-oss-oss registry allocate --type ports --owner "OrderService.Host" \
    --value 5701 --env prod --note "orders domain API"
```

---

### 2. `logger_event_ranges` (kind: range)

**Purpose**: reserve a per-BC range of LoggerMessage / EventId static numbers to prevent collisions.

**Required fields**:
- `range_start` (int)
- `range_end` (int)
- `owner` (string): typically the BC name (e.g. `Collection`, `Strategy`)

**Recommended fields**:
- `note` (string): "Data collection BC — uses 2000~2999 EventIds"

**Conflict rule**: REJECTED if the range **overlaps** any other allocation. 1000-unit ranges recommended.

**Example**:
```python
hstl-oss-oss registry allocate --type logger_event_ranges --owner "Reporting" \
    --range-start 7000 --range-end 7999 --note "Reporting BC"
```

---

### 3. `dns_records` (kind: unique)

**Purpose**: hostname → IP / service endpoint mapping. Used for internal service discovery and external access-point management.

**Required fields**:
- `value` (string): hostname (e.g. `my-service.example.com`)
- `owner` (string): service identifier

**Recommended fields**:
- `note` (string): A record / CNAME / specific IP / TTL etc.

**Conflict rule**: only one entry per hostname.

**Example**:
```bash
hstl-oss-oss registry allocate --type dns_records --owner dashboard \
    --value "dashboard.example.com" --note "A → 192.168.1.50, TTL 300"
```

---

### 4. `external_apis` (kind: unique)

**Purpose**: external API endpoint URL / provider identifier. API keys live in a separate secrets store (vault); only the identifier lives here.

**Required fields**:
- `value` (string): endpoint URL or identifying string
- `owner` (string): consuming component (e.g. `PriceCollector`)

**Recommended fields**:
- `note` (string): auth scheme, rate limit, response format, etc.

**Conflict rule**: only one entry per (owner, value) tuple. If multiple components use the same endpoint, register them with different owners.

**Example**:
```bash
hstl-oss-oss registry allocate --type external_apis --owner MarketDataIngest \
    --value "api.example.com/v1/quotes" --note "Bearer token, 100 req/sec rate limit"
```

---

---

## Extension candidates (created dynamically when needed)

The following are seed types that frequently come up. They aren't in the 1.0 catalog, but pass `resource_kind` and `resource_description` on the first allocation and they register as a new type immediately.

### `dapr_app_ids` (kind: unique)

Dapr Application ID management (environments using a Dapr sidecar).

```bash
hstl-oss-oss registry allocate --type dapr_app_ids --owner OrderService \
    --value "order-svc-prod" --note "Dapr Application ID"
```

### `pg_databases` (kind: unique)

PostgreSQL database name management (multi-DB environments).

```bash
hstl-oss-oss registry allocate --type pg_databases --owner OrderService \
    --value "orders_prod" --note "PostgreSQL database name"
```

### `kafka_topics` (kind: unique)

Kafka topic name management (message broker environments). Enforces topic-naming convention consistency.

```bash
hstl-oss-oss registry allocate --type kafka_topics --owner OrderService \
    --value "orders.events.v1" --note "Kafka topic name"
```

### `vip_addresses` (kind: unique)

Network VIPs (HA environments).

```bash
hstl-oss-oss registry allocate --type vip_addresses --owner traefik-cluster \
    --value "192.168.100.10" --note "HA Virtual IP — Traefik front VIP"
```

---

## Procedure for adding a new type

1. Call `hstl-oss registry allocate` for the first allocation
2. Subsequent allocations of the same type can omit description/kind
3. **Recommended: add it to this catalog doc** so other users can discover it via search

This catalog is a guideline rather than the SSOT; the actual SSOT is the file `~/.hostler/data/{project_key}/registry.json`. Catalog docs and actual data may evolve independently.

## Related documents

- This skill's design: `docs/02-design/project-resource-requirements.md`
- CLI implementation: `cli/cmd/hostler/cmd/resource.go`
- Example scenarios: `../examples/port-allocation.md`, `../examples/eventid-range.md`
