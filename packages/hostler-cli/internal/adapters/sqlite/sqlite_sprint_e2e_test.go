// SQLite adapter sprint lifecycle e2e test.
// Verifies the backlog → active → completed transitions and state
// transition integrity.
package sqlite_test

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// TestT242_SprintLifecycle_E2E exercises the full Sprint lifecycle
// (backlog → active → completed) through the SQLite adapter and
// verifies that state plus metadata transitions are intact.
// Acts as a regression guard.
func TestT242_SprintLifecycle_E2E(t *testing.T) {
	store := sqlite.New(openTestDB(t))

	// Step 1: create in backlog.
	sprint := &ports.SprintRecord{
		SprintID:   "sprint-e2e-001",
		Title:      "E2E test sprint",
		Status:     "backlog",
		FolderPath: "works/sprints/backlog/sprint-e2e-001",
		Goal:       "Sprint lifecycle regression guard",
		CreatedAt:  "2026-04-21T00:00:00Z",
		UpdatedAt:  "2026-04-21T00:00:00Z",
	}
	if err := store.SaveSprint(sprint); err != nil {
		t.Fatalf("Step 1 SaveSprint(backlog): %v", err)
	}

	got, err := store.GetSprint("sprint-e2e-001")
	if err != nil || got.Status != "backlog" {
		t.Fatalf("Step 1 verify: status=%q, err=%v", got.Status, err)
	}

	// Step 2: backlog → active.
	startedAt := "2026-04-22T09:00:00Z"
	opts := &ports.SprintUpdateOpts{
		FolderPath: "works/sprints/active/sprint-e2e-001",
		StartedAt:  startedAt,
	}
	if err := store.UpdateSprintStatus("sprint-e2e-001", "active", opts); err != nil {
		t.Fatalf("Step 2 UpdateSprintStatus(active): %v", err)
	}

	got, _ = store.GetSprint("sprint-e2e-001")
	if got.Status != "active" {
		t.Errorf("Step 2 active: want active, got %s", got.Status)
	}
	if got.FolderPath != "works/sprints/active/sprint-e2e-001" {
		t.Errorf("Step 2 folder_path: got %s", got.FolderPath)
	}
	if got.StartedAt != startedAt {
		t.Errorf("Step 2 started_at: got %s", got.StartedAt)
	}

	// Step 3: active → completed.
	completedAt := "2026-04-25T18:00:00Z"
	opts2 := &ports.SprintUpdateOpts{
		FolderPath:  "works/sprints/completed/sprint-e2e-001",
		CompletedAt: completedAt,
	}
	if err := store.UpdateSprintStatus("sprint-e2e-001", "completed", opts2); err != nil {
		t.Fatalf("Step 3 UpdateSprintStatus(completed): %v", err)
	}

	got, _ = store.GetSprint("sprint-e2e-001")
	if got.Status != "completed" {
		t.Errorf("Step 3 completed: want completed, got %s", got.Status)
	}
	if got.CompletedAt != completedAt {
		t.Errorf("Step 3 completed_at: got %s", got.CompletedAt)
	}
	// started_at must be preserved from Step 2.
	if got.StartedAt != startedAt {
		t.Errorf("Step 3 started_at lost: got %s, want %s", got.StartedAt, startedAt)
	}
}
