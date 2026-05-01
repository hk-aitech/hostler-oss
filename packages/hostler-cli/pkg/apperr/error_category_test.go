// Package apperr — T867 (Sprint-102) regression tests for error category subdivision.
package apperr

import (
	"strings"
	"testing"
)

// TestT867_ErrorCategory_AtLeast15Constants — catalog agreement (T867 Task body)
// guarantees a minimum of 10 categories plus subdivision constants for a total of 15.
// When any are missing, the catalog document also flags drift.
func TestT867_ErrorCategory_AtLeast15Constants(t *testing.T) {
	cats := AllCategories()
	if len(cats) < 15 {
		t.Errorf("expected category count >=15, got %d", len(cats))
	}
	// Duplicate check
	seen := make(map[string]bool)
	for _, c := range cats {
		s := c.String()
		if seen[s] {
			t.Errorf("duplicate category: %s", s)
		}
		seen[s] = true
		if !isScreamingSnake(s) {
			t.Errorf("category naming convention violated (SCREAMING_SNAKE_CASE): %s", s)
		}
	}
}

// TestT867_RejectedError_CategoryOverride — when a subdivided category is
// supplied, Category() must return that value.
func TestT867_RejectedError_CategoryOverride(t *testing.T) {
	e := &RejectedError{
		Reason:           "already complete",
		CategoryOverride: CategoryAlreadyInState,
	}
	if got := e.Category(); got != CategoryAlreadyInState.String() {
		t.Errorf("override should take precedence: got %s, want %s", got, CategoryAlreadyInState)
	}
}

// TestT867_RejectedError_Category_Default — when override is unset, default to REJECTED.
func TestT867_RejectedError_Category_Default(t *testing.T) {
	e := &RejectedError{Reason: "generic validation failure"}
	if got := e.Category(); got != CategoryRejected.String() {
		t.Errorf("default should be REJECTED: got %s", got)
	}
}

// TestT867_BlockedError_CategoryOverride — HARNESS_BLOCKED subdivision.
func TestT867_BlockedError_CategoryOverride(t *testing.T) {
	e := &BlockedError{
		EntityID:         "T001",
		Message:          "harness incomplete",
		CategoryOverride: CategoryHarnessBlocked,
	}
	if got := e.Category(); got != CategoryHarnessBlocked.String() {
		t.Errorf("override should take precedence: got %s, want %s", got, CategoryHarnessBlocked)
	}
}

// TestT867_BlockedError_Category_Default — when override is unset, default to BLOCKED.
func TestT867_BlockedError_Category_Default(t *testing.T) {
	e := &BlockedError{EntityID: "T001"}
	if got := e.Category(); got != CategoryBlocked.String() {
		t.Errorf("default should be BLOCKED: got %s", got)
	}
}

// isScreamingSnake — [A-Z][A-Z0-9_]* heuristic.
func isScreamingSnake(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && (r < 'A' || r > 'Z') {
			return false
		}
		isUpper := r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		isUnder := r == '_'
		if !isUpper && !isDigit && !isUnder {
			return false
		}
	}
	// No lowercase allowed (the loop above already guarantees this; double-check).
	if strings.ToUpper(s) != s {
		return false
	}
	return true
}
