// Package apperr provides domain error types.
// Functions in pkg/ return errors that satisfy the error interface so the
// adapter layer can dispatch per type via errors.As.
package apperr

import "fmt"

// ---------------------------------------------------------------------------
// NotFoundError
// ---------------------------------------------------------------------------

// NotFoundError signals that an entity lookup failed.
type NotFoundError struct {
	EntityType string
	EntityID string
	Message string
	RecoveryHint string
}

func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%s %s not found.", e.EntityType, e.EntityID)
}

// Category returns the error category string.— uses the constant.
func (e *NotFoundError) Category() string { return CategoryNotFound.String() }

// ---------------------------------------------------------------------------
// BlockedError
// ---------------------------------------------------------------------------

// BlockedError signals a Harness Gate block.
//
// Extra is a free-form field exposed verbatim during response conversion (e.g.
// missing_files, policy, section_titles_used and other validation details).
// errorToResponse merges it as long as keys do not collide.
//
// CategoryOverride may be set to a BLOCKED subdivision.
// When empty the legacy "BLOCKED" value is returned (backward compat). When
// supplied it can branch to HARNESS_BLOCKED / RESULT_STRICT_MISSING_FILES /
// CEREMONY_SECTION_MISSING and so on.
type BlockedError struct {
	EntityType string
	EntityID string
	UncheckedItems []UncheckedItem
	SuggestedActions []string
	CriteriaStatus any
	Message string
	Extra map[string]any
	CategoryOverride ErrorCategory //— when set, returned directly
}

// UncheckedItem is an unfinished Harness item.
type UncheckedItem struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Required bool `json:"required"`
}

func (e *BlockedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%d items unfinished", len(e.UncheckedItems))
}

// Category returns the error category string.— CategoryOverride wins.
func (e *BlockedError) Category() string {
	if e.CategoryOverride != "" {
		return e.CategoryOverride.String()
	}
	return CategoryBlocked.String()
}

// ---------------------------------------------------------------------------
// RejectedError
// ---------------------------------------------------------------------------

// RejectedError signals a validation failure or policy violation.
//
// CategoryOverride may be set to a REJECTED subdivision.
type RejectedError struct {
	Reason string
	RecoveryHint string
	EntityID string
	CurrentState string
	ProvidedLength int
	Extra map[string]string // extra context (optional)
	CategoryOverride ErrorCategory //— when set, returned directly
}

func (e *RejectedError) Error() string { return e.Reason }

// Category returns the error category string.— CategoryOverride wins.
func (e *RejectedError) Category() string {
	if e.CategoryOverride != "" {
		return e.CategoryOverride.String()
	}
	return CategoryRejected.String()
}

// ---------------------------------------------------------------------------
// InvalidStateError
// ---------------------------------------------------------------------------

// InvalidStateError signals an illegal state transition.
type InvalidStateError struct {
	EntityType string
	EntityID string
	CurrentState string
	ExpectedState string
	Message string
	RecoveryHint string
}

func (e *InvalidStateError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("state transition not allowed: current=%s, expected=%s", e.CurrentState, e.ExpectedState)
}

// Category returns the error category string.— uses the constant.
func (e *InvalidStateError) Category() string { return CategoryInvalidState.String() }

// ---------------------------------------------------------------------------
// Categorizer interface
// ---------------------------------------------------------------------------

// Categorizer is the interface for errors that expose a category.
// Adapter code uses it together with errors.As for shared handling.
type Categorizer interface {
	error
	Category() string
}
