// Package apperr — hostler-specific error types.
//
// The base error types (NotFoundError / BlockedError / RejectedError /
// InvalidStateError) live in errors.go. The five hostler extension types
// (InvalidInputError / HMACVerifyError / GPGVerifyError / ConfigInvalidError /
// DBError) are kept in this file.
package apperr

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON methods — base 4 error types (NotFoundError / BlockedError / RejectedError / InvalidStateError)

// MarshalJSON — standard four fields (status/error_category/message/recovery_hint).
func (e *NotFoundError) MarshalJSON() ([]byte, error) {
	return marshalStandardError(e.Category(), e.Error(), e.RecoveryHint, nil)
}

func (e *BlockedError) MarshalJSON() ([]byte, error) {
	hint := ""
	if len(e.SuggestedActions) > 0 {
		hint = e.SuggestedActions[0]
	}
	extra := map[string]any{
		"unchecked_items": e.UncheckedItems,
		"suggested_actions": e.SuggestedActions,
	}
	if e.CriteriaStatus != nil {
		extra["criteria_status"] = e.CriteriaStatus
	}
	for k, v := range e.Extra {
		extra[k] = v
	}
	return marshalStandardError(e.Category(), e.Error(), hint, extra)
}

// MarshalJSON — standard.
func (e *RejectedError) MarshalJSON() ([]byte, error) {
	extra := map[string]any{}
	if e.EntityID != "" {
		extra["entity_id"] = e.EntityID
	}
	if e.CurrentState != "" {
		extra["current_state"] = e.CurrentState
	}
	if e.ProvidedLength != 0 {
		extra["provided_length"] = e.ProvidedLength
	}
	for k, v := range e.Extra {
		extra[k] = v
	}
	if len(extra) == 0 {
		extra = nil
	}
	return marshalStandardError(e.Category(), e.Error(), e.RecoveryHint, extra)
}

// MarshalJSON — standard.
func (e *InvalidStateError) MarshalJSON() ([]byte, error) {
	extra := map[string]any{}
	if e.EntityType != "" {
		extra["entity_type"] = e.EntityType
	}
	if e.EntityID != "" {
		extra["entity_id"] = e.EntityID
	}
	if e.CurrentState != "" {
		extra["current_state"] = e.CurrentState
	}
	if e.ExpectedState != "" {
		extra["expected_state"] = e.ExpectedState
	}
	if len(extra) == 0 {
		extra = nil
	}
	return marshalStandardError(e.Category(), e.Error(), e.RecoveryHint, extra)
}

// ---------------------------------------------------------------------------
// InvalidInputError 
// ---------------------------------------------------------------------------

// InvalidInputError — invalid input argument (missing required, malformed).
type InvalidInputError struct {
	Field string // offending argument name (e.g. "title", "summary")
	Reason string // reason (e.g. "required", "too short", "invalid format")
	Provided string // value supplied (debug)
	RecoveryHint string
}

func (e *InvalidInputError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Reason)
	}
	return e.Reason
}

func (e *InvalidInputError) Category() string { return CategoryInvalidInput.String() }

func (e *InvalidInputError) MarshalJSON() ([]byte, error) {
	extra := map[string]any{}
	if e.Field != "" {
		extra["field"] = e.Field
	}
	if e.Provided != "" {
		extra["provided"] = e.Provided
	}
	if len(extra) == 0 {
		extra = nil
	}
	return marshalStandardError(e.Category(), e.Error(), e.RecoveryHint, extra)
}

// ---------------------------------------------------------------------------
// HMACVerifyError 
// ---------------------------------------------------------------------------

// HMACVerifyError — HMAC signature verification failure.
type HMACVerifyError struct {
	EntityType string // "task" | "sprint"
	EntityID string
	FilePath string
	Expected string // expected hash (debug, prefix only)
	Computed string // computed hash (debug, prefix only)
	RecoveryHint string
}

func (e *HMACVerifyError) Error() string {
	return fmt.Sprintf("HMAC verification failed: %s/%s (file=%s)", e.EntityType, e.EntityID, e.FilePath)
}

func (e *HMACVerifyError) Category() string { return CategoryHMACVerifyFailed.String() }

func (e *HMACVerifyError) MarshalJSON() ([]byte, error) {
	extra := map[string]any{}
	if e.EntityType != "" {
		extra["entity_type"] = e.EntityType
	}
	if e.EntityID != "" {
		extra["entity_id"] = e.EntityID
	}
	if e.FilePath != "" {
		extra["file_path"] = e.FilePath
	}
	if e.Expected != "" {
		extra["expected_prefix"] = e.Expected
	}
	if e.Computed != "" {
		extra["computed_prefix"] = e.Computed
	}
	if len(extra) == 0 {
		extra = nil
	}
	return marshalStandardError(e.Category(), e.Error(), e.RecoveryHint, extra)
}

// ---------------------------------------------------------------------------
// ConfigInvalidError 
// ---------------------------------------------------------------------------

// ConfigInvalidError — config file or field is invalid.
type ConfigInvalidError struct {
	FilePath string // offending config file path
	Field string // offending field (jq path, e.g. ".harness_required[]")
	Reason string
	RecoveryHint string
}

func (e *ConfigInvalidError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config %s.%s: %s", e.FilePath, e.Field, e.Reason)
	}
	return fmt.Sprintf("config %s: %s", e.FilePath, e.Reason)
}

func (e *ConfigInvalidError) Category() string { return CategoryConfigInvalid.String() }

func (e *ConfigInvalidError) MarshalJSON() ([]byte, error) {
	extra := map[string]any{}
	if e.FilePath != "" {
		extra["file_path"] = e.FilePath
	}
	if e.Field != "" {
		extra["field"] = e.Field
	}
	if len(extra) == 0 {
		extra = nil
	}
	return marshalStandardError(e.Category(), e.Error(), e.RecoveryHint, extra)
}

// ---------------------------------------------------------------------------
// Shared helper
// ---------------------------------------------------------------------------

// marshalStandardError — emits the standard four fields plus extras.
//
// Standard fields: status="error" / error_category / message / recovery_hint (when present)
// extra: per-error context (entity_id, file_path, etc.)
func marshalStandardError(category, message, hint string, extra map[string]any) ([]byte, error) {
	out := map[string]any{
		"status": "error",
		"error_category": category,
		"message": message,
	}
	if hint != "" {
		out["recovery_hint"] = hint
	}
	for k, v := range extra {
		// Avoid clobbering standard fields.
		if _, exists := out[k]; exists {
			continue
		}
		out[k] = v
	}
	return json.Marshal(out)
}
