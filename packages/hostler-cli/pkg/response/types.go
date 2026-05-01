// Package response provides the shared MCP tool response structures.
package response

// OkResponse is the success response.
type OkResponse struct {
	Status string         `json:"status"`
	Data   map[string]any `json:"data,omitempty"`
}

// BlockedResponse is the Harness Gate blocked response.
type BlockedResponse struct {
	Status         string   `json:"status"`
	UncheckedItems []string `json:"unchecked_items"`
	RecoveryHint   string   `json:"recovery_hint"`
}

// RejectedResponse is the input-validation failure response.
type RejectedResponse struct {
	Status     string         `json:"status"`
	Reason     string         `json:"reason"`
	Suggestion string         `json:"suggestion,omitempty"`
	Extra      map[string]any `json:"extra,omitempty"`
}

// ErrorResponse is the generic error response.
type ErrorResponse struct {
	Status        string         `json:"status"`
	Message       string         `json:"message"`
	ErrorCategory string         `json:"error_category"`
	RecoveryHint  string         `json:"recovery_hint,omitempty"`
	Extra         map[string]any `json:"extra,omitempty"`
}
