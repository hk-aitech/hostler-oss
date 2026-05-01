# Code Quality Metrics

## Complexity Thresholds

| Metric | Good | Acceptable | Needs Attention |
|--------|------|------------|-----------------|
| Cyclomatic Complexity | < 5 | 5-10 | > 10 |
| Function Length | < 25 lines | 25-50 lines | > 50 lines |
| File Length | < 300 lines | 300-500 lines | > 500 lines |
| Nesting Depth | < 3 | 3-4 | > 4 |
| Parameters | < 4 | 4-5 | > 5 |

## Test Coverage Targets

| Code Type | Minimum | Recommended |
|-----------|---------|-------------|
| Core Business Logic | 90% | 100% |
| Domain Models | 80% | 90% |
| Application Services | 70% | 80% |
| Infrastructure | 50% | 60% |
| New Code | 80% | 90% |

## Duplication Limits

| Metric | Threshold |
|--------|-----------|
| Duplicate Lines | < 3% |
| Similar Blocks | < 5% |
| Copy-Paste Detection | 0 exact copies |

## Naming Conventions

### C# / .NET
```
Classes:        PascalCase        (OrderService)
Interfaces:     IPascalCase       (IOrderService)
Methods:        PascalCase        (ProcessOrder)
Properties:     PascalCase        (TotalAmount)
Private Fields: _camelCase        (_orderRepository)
Parameters:     camelCase         (orderId)
Constants:      PascalCase        (MaxRetryCount)
```

### TypeScript / JavaScript
```
Classes:        PascalCase        (OrderService)
Functions:      camelCase         (processOrder)
Variables:      camelCase         (totalAmount)
Constants:      SCREAMING_SNAKE   (MAX_RETRY_COUNT)
Interfaces:     PascalCase        (OrderRequest)
```

### Python
```
Classes:        PascalCase        (OrderService)
Functions:      snake_case        (process_order)
Variables:      snake_case        (total_amount)
Constants:      SCREAMING_SNAKE   (MAX_RETRY_COUNT)
Modules:        snake_case        (order_service.py)
```

### Go
```
Exported:       PascalCase        (ProcessOrder)
Unexported:     camelCase         (processOrder)
Packages:       lowercase         (orderservice)
Constants:      PascalCase        (MaxRetryCount)
```

## Issue Severity Classification

| Severity | Criteria | Action Required |
|----------|----------|-----------------|
| **Critical** | Security vulnerability, data loss risk | Block merge, fix immediately |
| **Major** | Functional bug, significant performance issue | Fix before release |
| **Minor** | Style violation, minor improvement | Add to backlog |
| **Info** | Suggestion, best practice | Optional |
