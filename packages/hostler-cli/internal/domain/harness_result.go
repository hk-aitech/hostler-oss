// Package domain — Harness Gate DTO.
//
// The pkg/harness DTOs were promoted into internal/domain.
package domain

// HarnessItem is consolidated as the domain entity in
// domain/harness.go, JSON tags included.

// HarnessResult — harness verification result.
type HarnessResult struct {
	UncheckedRequired []HarnessItem `json:"unchecked_required"`
	AllItems          []HarnessItem `json:"all_items"`
	Blocked           bool          `json:"blocked"`
	RequiredTotal     int           `json:"required_total"`
	RequiredDone      int           `json:"required_done"`
}

// HarnessGetResult — return type of HarnessGet.
type HarnessGetResult struct {
	EntityType    string            `json:"entity_type"`
	EntityID      string            `json:"entity_id"`
	TemplateKey   string            `json:"template_key"`
	Items         []HarnessItemView `json:"items"`
	RequiredTotal int               `json:"required_total"`
	RequiredDone  int               `json:"required_done"`
	Blocked       bool              `json:"blocked"`
}

// HarnessItemView — item view embedded in HarnessGet responses.
type HarnessItemView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Done      bool   `json:"done"`
	Evidence  string `json:"evidence,omitempty"`
	CheckedAt string `json:"checked_at,omitempty"`
	CheckedBy string `json:"checked_by,omitempty"`
}

// HarnessCheckResult — return type of HarnessCheck.
type HarnessCheckResult struct {
	Checked             bool              `json:"checked"`
	ItemID              string            `json:"item_id"`
	EntityType          string            `json:"entity_type"`
	EntityID            string            `json:"entity_id"`
	Remaining           []HarnessItemView `json:"remaining"`
	Blocked             bool              `json:"blocked"`
	SuggestedNextAction string            `json:"suggested_next_action,omitempty"`
}

// CriteriaResult — return type of AutoCheckCriteria / parseCompletionCriteria.
type CriteriaResult struct {
	Total      int      `json:"total"`
	Checked    int      `json:"checked"`
	AllChecked bool     `json:"all_checked"`
	Unchecked  []string `json:"unchecked"`
}
