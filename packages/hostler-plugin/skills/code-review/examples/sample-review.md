# Code Review Report Example

## Summary

| Item | Value |
|------|-------|
| Files Reviewed | 5 |
| Lines Changed | +234 / -45 |
| Issues Found | 7 |
| Critical | 1 |
| Major | 2 |
| Minor | 3 |
| Info | 1 |

---

## Critical Issues

### [CRITICAL] Hardcoded API Key

- **File**: `src/services/payment.ts:45`
- **Code**:
  ```typescript
  const API_KEY = "sk-live-abc123xyz789...";
  ```
- **Description**: Production API key exposed in source code
- **Suggestion**: Move to environment variable
  ```typescript
  const API_KEY = process.env.PAYMENT_API_KEY;
  ```
- **Reference**: OWASP A3 - Sensitive Data Exposure

---

## Major Issues

### [MAJOR] N+1 Query Pattern

- **File**: `src/repositories/order.py:78`
- **Code**:
  ```python
  for order in orders:
      items = db.query(OrderItem).filter_by(order_id=order.id).all()
  ```
- **Description**: Loop executes N additional queries
- **Suggestion**: Use eager loading
  ```python
  orders = db.query(Order).options(joinedload(Order.items)).all()
  ```

### [MAJOR] Missing Input Validation

- **File**: `src/controllers/UserController.cs:32`
- **Code**:
  ```csharp
  public async Task<IActionResult> CreateUser(UserDto dto)
  {
      var user = new User { Name = dto.Name, Email = dto.Email };
  ```
- **Description**: No validation before creating user
- **Suggestion**: Add FluentValidation or data annotations

---

## Minor Issues

### [MINOR] Function Name Improvement

- **File**: `src/utils/helper.go:23`
- **Current**: `func do(data string)`
- **Suggestion**: `func processOrderData(data string)`

### [MINOR] Missing XML Documentation

- **File**: `src/services/OrderService.cs:15`
- **Description**: Public method lacks XML documentation
- **Suggestion**: Add `<summary>`, `<param>`, `<returns>` tags

### [MINOR] Magic Number

- **File**: `src/validators/AgeValidator.ts:8`
- **Code**: `if (age < 18)`
- **Suggestion**: Extract constant `const MINIMUM_AGE = 18;`

---

## Info

### [INFO] Consider Using Record Type

- **File**: `src/models/OrderSummary.cs`
- **Description**: Immutable DTO could use C# record syntax
- **Suggestion**: `public record OrderSummary(int Id, decimal Total);`

---

## Statistics

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Cyclomatic Complexity (max) | 8 | < 10 | PASS |
| Function Length (max lines) | 42 | < 50 | PASS |
| Test Coverage | 72% | > 80% | FAIL |
| Duplicated Lines | 1.2% | < 3% | PASS |

---

## Recommendations

1. **Security**: Move all secrets to environment variables
2. **Performance**: Review database queries for N+1 patterns
3. **Testing**: Increase test coverage to meet 80% target
4. **Documentation**: Add XML docs to public API methods

---

## Approval Status

- [ ] All Critical issues resolved
- [ ] All Major issues resolved or acknowledged
- [x] Tests passing
- [ ] Ready for merge
