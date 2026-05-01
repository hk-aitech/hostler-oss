package domain

// GateScope — the scope to which a Harness Gate applies.
type GateScope string

const (
	GateScopeTask   GateScope = "task"
	GateScopeSprint GateScope = "sprint"
)

// HarnessItem — Harness checklist item (unifies the domain entity with the
// external DTO).
//
// Merged with the same-named DTO in pkg/harness. JSON tags drive CLI
// response serialization, and domain logic (HarnessGate.IsSatisfied,
// etc.) shares the same type.
type HarnessItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Done      bool   `json:"done"`
	Evidence  string `json:"evidence,omitempty"`
	CheckedAt string `json:"checked_at,omitempty"`
	CheckedBy string `json:"checked_by,omitempty"`
}

// HarnessGate — Harness verification gate.
type HarnessGate struct {
	Scope GateScope
	Items []HarnessItem
}

// IsSatisfied reports whether every required item is complete.
func (g HarnessGate) IsSatisfied() bool {
	return len(g.BlockingItems()) == 0
}

// AllRequiredDone is a synonym for IsSatisfied. The ADR text lists
// AllRequiredDone as an example signature, so we expose an explicit
// alias for API compatibility.
func (g HarnessGate) AllRequiredDone() bool {
	return g.IsSatisfied()
}

// BlockingItems returns the list of incomplete required items.
func (g HarnessGate) BlockingItems() []HarnessItem {
	var blocked []HarnessItem
	for _, item := range g.Items {
		if item.Required && !item.Done {
			blocked = append(blocked, item)
		}
	}
	return blocked
}
