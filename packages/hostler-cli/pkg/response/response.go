// Package response provides the shared MCP tool response structures.
// Direct port of utils/response.py from the Python implementation.
package response

// Ok returns a success response.
func Ok(data map[string]any) *OkResponse {
	return &OkResponse{
		Status: "ok",
		Data:   data,
	}
}

// Blocked returns a Harness Gate blocked response.
func Blocked(uncheckedItems []string, recoveryHint string) *BlockedResponse {
	if recoveryHint == "" {
		recoveryHint = "run harness_check"
	}
	return &BlockedResponse{
		Status:         "BLOCKED",
		UncheckedItems: uncheckedItems,
		RecoveryHint:   recoveryHint,
	}
}

// Rejected returns an input-validation failure response.
func Rejected(reason string, suggestion string, extra map[string]any) *RejectedResponse {
	return &RejectedResponse{
		Status:     "REJECTED",
		Reason:     reason,
		Suggestion: suggestion,
		Extra:      extra,
	}
}

// Error returns a generic error response.
func Error(message string, errorCategory string, recoveryHint string, extra map[string]any) *ErrorResponse {
	if errorCategory == "" {
		errorCategory = "INTERNAL_ERROR"
	}
	return &ErrorResponse{
		Status:        "error",
		Message:       message,
		ErrorCategory: errorCategory,
		RecoveryHint:  recoveryHint,
		Extra:         extra,
	}
}
