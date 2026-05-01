// Package ports — SprintStore port.
//
// The second port to follow the pattern established by TaskStore.
// Isolates Sprint-domain CRUD + state transitions + aggregation +
// synchronization behind a single-responsibility interface.
//
// Reuses the TaskStore extraction pattern — extracted from GraphStore,
// compile-checked across three adapters in parallel. GraphStore's
// Sprint methods are **retained** for backward compatibility.
package ports

// SprintStore — port for the Sprint domain (CRUD + state transitions + aggregation).
type SprintStore interface {
	// ── Create / Upsert ─────────────────────────────────────────────────
	SaveSprint(sprint *SprintRecord) error
	InsertSprintIfMissing(sprint *SprintRecord) error
	UpsertSprintInsert(sprint *SprintRecord) error

	// ── Read ─────────────────────────────────────────────────────────────
	GetSprint(sprintID string) (*SprintRecord, error)
	GetSprintTaskFilePaths(sprintID string) ([]string, error)

	// ── List / Query ─────────────────────────────────────────────────────
	ListSprints(status string) ([]SprintRecord, error)
	GetAllSprintsSync() ([]SprintSyncRecord, error)
	GetAllSprintsForSync() ([]SprintFullSyncRecord, error)
	GetSprintTaskCounts() (map[string]int, error)

	// ── Aggregate (domain computation) ──────────────────────────────────
	AggregateSprintProgress(sprintID string) (*ProgressResult, error)

	// ── Update ───────────────────────────────────────────────────────────
	UpdateSprintStatus(sprintID, status string, opts *SprintUpdateOpts) error
	UpdateSprintMetadata(sprintID, title, folderPath, status string) error
	UpdateTaskPathsForSprint(sprintID, fromLocation, toLocation string) error
}
