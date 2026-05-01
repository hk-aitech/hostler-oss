// Package harness return-type declarations.
// The DTOs originally defined here were promoted to internal/domain.
// All type aliases were removed; domain is the single SSOT. Callers
// reference `domain.HarnessGetResult` and friends directly (requires
// importing internal/domain).
package harness
