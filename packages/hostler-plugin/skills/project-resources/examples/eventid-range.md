# EventId Range Allocation scenarios

Reserve per-BC ranges to avoid LoggerMessage / EventId static-number collisions. All commands invoke the `hstl-oss registry` CLI.

---

## Background

.NET's LoggerMessage source generator does not catch EventId(int) collisions at compile time, so a clash leads to runtime misbehavior (wrong message mapping). Reserving a 1000-unit static range per BC eliminates collisions at issuance time.

The same pattern applies to other languages/stacks that use static EventIds (e.g. a custom logging catalog).

---

## Scenario 1: Allocate a 1000-unit EventId range to the new `Reporting` BC

### Step 1. Inspect the current catalog

```bash
hstl-oss-oss registry list resource_type="logger_event_ranges")
```

Response (summarized):
```
| owner       | range_start | range_end | note                |
| Collection  | 2000        | 2999      | Data collection BC  |
| Strategy    | 3000        | 3999      | Strategy BC         |
| Trading     | 4000        | 4999      | Trading execution BC|
| Operations  | 5000        | 5999      | Operations BC       |
| Infra       | 9000        | 9999      | Cross-cutting infra |
```

→ 6000~8999 is free.

### Step 2. Check the candidate range for conflicts

```bash
hstl-oss-oss registry check 
    resource_type="logger_event_ranges",
    range_start=6000,
    range_end=6999
)
```

Response: `{"available": true}` or, on conflict, a `conflicts` payload.

### Step 3. Allocate

```bash
hstl-oss-oss registry allocate 
    resource_type="logger_event_ranges",
    owner="Reporting",
    range_start=6000,
    range_end=6999,
    note="Reporting BC — report generation/delivery EventIds"
)
```

Response: `{"status": "OK", "allocation": {...}, "sync": {...}}`

### Step 4. Use it in code

C# example:
```csharp
[LoggerMessage(EventId = 6001, Level = LogLevel.Information,
               Message = "Daily report scheduled at {Time}")]
public partial void LogReportScheduled(DateTimeOffset time);

[LoggerMessage(EventId = 6002, Level = LogLevel.Error,
               Message = "Report generation failed: {Reason}")]
public partial void LogReportFailed(string reason);
```

→ 6001, 6002 ... 6999 are free to use. No other BC can intrude.

---

## Scenario 2: Conflict

The existing `Strategy` BC owns 3000~3999, and a new BC requests 3500~4500 (overlapping 3500~3999).

```bash
hstl-oss-oss registry allocate 
    resource_type="logger_event_ranges",
    owner="StrategyV2",
    range_start=3500,
    range_end=4500,
    note="Strategy v2"
)
```

Response:
```json
{
  "status": "REJECTED",
  "reason": "conflict",
  "conflicts": [
    {"owner": "Strategy", "range_start": 3000, "range_end": 3999},
    {"owner": "Trading", "range_start": 4000, "range_end": 4999}
  ]
}
```

Response:
1. Search again for an empty range
2. Re-request with a larger or relocated range

```bash
hstl-oss-oss registry allocate 
    resource_type="logger_event_ranges",
    owner="StrategyV2",
    range_start=7000,
    range_end=7999,
    note="Strategy v2 — new EventId range"
)
```

---

## Recommended conventions

| Recommendation | Reason |
|----------------|--------|
| **Use 1000-unit ranges** | Each BC gets enough EventIds (0~999) and BC boundaries become visually obvious |
| **BC name = owner** | Standardize per BC: `Collection`, `Strategy`, `Trading`, etc. |
| **Include BC description in `note`** | Helps new developers orient quickly |
| **Sub-categories via trailing digits** | e.g. 6001~6099 = scheduling, 6100~6199 = generation, 6200~6299 = delivery (registry tracks BC ranges only; sub-grouping is a code convention) |

---

## Applying to other stacks

| Stack | EventId pattern | Recommended use |
|-------|-----------------|-----------------|
| .NET LoggerMessage | `[LoggerMessage(EventId=...)]` | 1.0 catalog standard |
| Go zap/zerolog | (structured logging — no EventId concept) | not used |
| Python structlog | (event-name based) | not used |
| Java SLF4J + EventId catalog | when self-defined | usable |

→ Stacks without an EventId pattern simply skip this resource_type. If you need a different standardization pattern (e.g. an error-code catalog), create a new resource_type dynamically.
