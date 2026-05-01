// Package ports — HarnessStore port.
//
// Port dedicated to managing Harness Gate state.
// **hstl identity port** — a core asset that must be carried over to the
// Hostler side. Isolates item CRUD, evidence handling, and Sprint Production
// verification under a single responsibility.
//
// Reuses the TaskStore / SprintStore extraction pattern — extracted from
// GraphStore and compile-checked across three adapters in parallel.
package ports

// HarnessStore — port for the Harness Gate domain (items + template + check).
type HarnessStore interface {
	// ── Template ────────────────────────────────────────────────────────
	GetHarnessTemplate(templateKey string) ([]map[string]any, error)
	EnsureHarnessItems(entityType, entityID string, items []HarnessItemTemplate) error

	// ── Read / Query ─────────────────────────────────────────────────────
	CountHarnessItems(entityType, entityID string) (int, error)
	GetHarnessItems(entityType, entityID string) ([]HarnessItem, error)
	GetHarnessItemID(entityType, entityID, itemID string) (int64, error)

	// ── Check (state transition) ─────────────────────────────────────────
	CheckHarnessItem(entityType, entityID, itemID, evidence, actor string) error

	// ── Sprint-specific verification ────────────────────────────────────
	CheckSprintProduction(sprintID string) (bool, error)
}
