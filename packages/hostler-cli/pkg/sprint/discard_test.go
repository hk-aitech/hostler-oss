package sprint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// T424 (Sprint-36) — exhaustive sprint.Discard tests.
//
// Target transition rules:
//   - backlog  → discarded : allowed
//   - active   → discarded : allowed + return CURRENT-FOCUS to idle
//   - completed→ discarded : rejected (KB O003)
//   - discarded→ discarded : rejected (idempotency not supported)
//   - reason < 10 chars     : rejected
//   - --return-tasks        : auto-unassigns unfinished Tasks

// helper: create a test Sprint record + folder in DB.
func seedSprint(t *testing.T, id, title, status string) string {
	t.Helper()
	folderPath, err := sprint.CreateFolder(id, status)
	if err != nil {
		t.Fatalf("seedSprint CreateFolder: %v", err)
	}
	// SPRINT.md frontmatter (used to verify status updates)
	sprintMD := "---\n" +
		"id: " + id + "\n" +
		"title: \"" + title + "\"\n" +
		"status: " + status + "\n" +
		"---\n"
	if err := os.WriteFile(filepath.Join(folderPath, "SPRINT.md"), []byte(sprintMD), 0o644); err != nil {
		t.Fatalf("seedSprint write SPRINT.md: %v", err)
	}
	rel := filepath.ToSlash(filepath.Join("works", "sprints", status, id))
	if err := sprint.SaveToDB(id, title, "goal", status, rel, nil, nil); err != nil {
		t.Fatalf("seedSprint SaveToDB: %v", err)
	}
	return folderPath
}

func TestT424_Discard_backlog_Success(t *testing.T) {
	tmpRoot := setupDB(t)
	id := "sprint-discard-b1"
	seedSprint(t, id, "backlog discard target", "backlog")

	res, err := sprint.Discard(id, "absorbed into another Sprint due to scope overlap", false, false)
	if err != nil {
		t.Fatalf("Discard failed: %v", err)
	}
	if res.Status != "discarded" || res.PreviousStatus != "backlog" {
		t.Errorf("status transition: got %s→%s, want backlog→discarded", res.PreviousStatus, res.Status)
	}
	assertPathExists(t, "discarded folder",
		filepath.Join(tmpRoot, "works", "sprints", "discarded", id))
	assertPathAbsent(t, "no backlog leftover",
		filepath.Join(tmpRoot, "works", "sprints", "backlog", id))
}

func TestT424_Discard_active_Success(t *testing.T) {
	tmpRoot := setupDB(t)
	id := "sprint-discard-a1"
	seedSprint(t, id, "active discard target", "active")

	res, err := sprint.Discard(id, "halt due to low progress and redesign", false, false)
	if err != nil {
		t.Fatalf("Discard active failed: %v", err)
	}
	if res.PreviousStatus != "active" || res.Status != "discarded" {
		t.Errorf("transition expected active→discarded, got %s→%s", res.PreviousStatus, res.Status)
	}
	assertPathExists(t, "discarded folder",
		filepath.Join(tmpRoot, "works", "sprints", "discarded", id))
	assertPathAbsent(t, "no active leftover",
		filepath.Join(tmpRoot, "works", "sprints", "active", id))
}

func TestT424_Discard_completed_Reject(t *testing.T) {
	_ = setupDB(t)
	id := "sprint-discard-c1"
	seedSprint(t, id, "completed protect target", "completed")

	if _, err := sprint.Discard(id, "accidentally completed Sprint reorganisation", false, false); err == nil {
		t.Fatal("must not allow discarding a completed Sprint — should be rejected (KB O003)")
	} else if !strings.Contains(err.Error(), "cannot be discarded") {
		t.Errorf("expected rejection reason 'cannot be discarded', got: %v", err)
	}
}

func TestT424_Discard_reason_TooShort_Reject(t *testing.T) {
	_ = setupDB(t)
	id := "sprint-discard-r1"
	seedSprint(t, id, "reason validation", "backlog")

	if _, err := sprint.Discard(id, "short", false, false); err == nil {
		t.Fatal("reason <10 chars must be rejected")
	}
	if _, err := sprint.Discard(id, strings.Repeat(" ", 12), false, false); err == nil {
		t.Fatal("whitespace-only reason must be rejected (TrimSpace then <10 chars)")
	}
}

func TestT424_Discard_returnTasks_UnfinishedReturned(t *testing.T) {
	_ = setupDB(t)
	id := "sprint-discard-rt"
	seedSprint(t, id, "return tasks verification", "active")

	// Compose 3 Tasks (mix of todo / in-progress / done).
	gs := store.Get()
	if gs == nil {
		t.Fatal("store nil")
	}
	addTask := func(taskID, status string) {
		rec := &ports.TaskRecord{
			TaskID:   taskID,
			Title:    taskID + " sample",
			Type:     "feature",
			Status:   status,
			Priority: "p3",
			Estimate: "XS",
			Sprint:   id,
			FilePath: filepath.ToSlash(filepath.Join("works", "sprints", "active", id, "tasks", taskID+".md")),
		}
		if err := gs.CreateTask(rec); err != nil {
			t.Fatalf("CreateTask %s: %v", taskID, err)
		}
	}
	addTask("T9001", "todo")
	addTask("T9002", "in-progress")
	addTask("T9003", "done")

	res, err := sprint.Discard(id, "verifying returnTasks behaviour", true, false)
	if err != nil {
		t.Fatalf("Discard --return-tasks failed: %v", err)
	}
	// exactly 2 should be returned (todo + in-progress).
	if len(res.ReturnedTasks) != 2 {
		t.Errorf("expected ReturnedTasks count 2, got %d (%v)", len(res.ReturnedTasks), res.ReturnedTasks)
	}
	// done must not be included.
	for _, rt := range res.ReturnedTasks {
		if rt == "T9003" {
			t.Error("done Task must not be in ReturnedTasks")
		}
	}
	// confirm the sprint field was cleared on those Tasks in DB.
	for _, tid := range []string{"T9001", "T9002"} {
		details, dErr := gs.GetTaskDetails(tid)
		if dErr != nil {
			t.Fatalf("GetTaskDetails %s: %v", tid, dErr)
		}
		if details.Sprint != "" {
			t.Errorf("Task %s expected empty sprint field, got %q", tid, details.Sprint)
		}
	}
}

func TestT424_Discard_audit_EventRecorded(t *testing.T) {
	tmpRoot := setupDB(t)
	id := "sprint-discard-ae"
	seedSprint(t, id, "audit recording", "backlog")

	if _, err := sprint.Discard(id, "verifying audit-event recording", false, false); err != nil {
		t.Fatalf("Discard: %v", err)
	}

	// Look up sprint.discarded event in audit_events.
	events, err := audit.QueryEvents(audit.QueryFilter{
		EventType:  "sprint.discarded",
		EntityType: "sprint",
		EntityID:   id,
	})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("sprint.discarded event not found (entity=%s). tmpRoot=%s", id, tmpRoot)
	}
	e := events[0]
	if e.Details["from_status"] != "backlog" {
		t.Errorf("from_status expected backlog, got %v", e.Details["from_status"])
	}
	if e.Details["to_status"] != "discarded" {
		t.Errorf("to_status expected discarded, got %v", e.Details["to_status"])
	}
	reason, _ := e.Details["reason"].(string)
	if !strings.Contains(reason, "audit-event") {
		t.Errorf("reason expected to contain 'audit-event', got %v", reason)
	}
}

// double discard (re-discard from discarded state) is rejected.
func TestT424_Discard_DiscardedRetry_Reject(t *testing.T) {
	_ = setupDB(t)
	id := "sprint-discard-double"
	seedSprint(t, id, "prevent double discard", "backlog")

	if _, err := sprint.Discard(id, "first discard — proceeding", false, false); err != nil {
		t.Fatalf("first Discard: %v", err)
	}
	if _, err := sprint.Discard(id, "attempt second discard — already discarded", false, false); err == nil {
		t.Fatal("re-discarding an already discarded Sprint must be rejected")
	}
}

// T436 — --db-only path: clean up drift state where DB has the Sprint but
// the folder is missing.
func TestT436_Discard_DBOnly_Success(t *testing.T) {
	tmpRoot := setupDB(t)
	id := "sprint-discard-dbonly"

	// Register only in DB (no folder) — simulate drift.
	rel := filepath.ToSlash(filepath.Join("works", "sprints", "backlog", id))
	if err := sprint.SaveToDB(id, "DB drift simulation", "goal", "backlog", rel, nil, nil); err != nil {
		t.Fatalf("SaveToDB: %v", err)
	}

	res, err := sprint.Discard(id, "officially clean up folderless DB drift state", false, true)
	if err != nil {
		t.Fatalf("Discard --db-only failed: %v", err)
	}
	if res.PreviousStatus != "backlog" || res.Status != "discarded" {
		t.Errorf("transition expected backlog→discarded, got %s→%s", res.PreviousStatus, res.Status)
	}
	if !res.DBOnly {
		t.Error("DBOnly field expected true")
	}
	// the folder must not be created.
	assertPathAbsent(t, "discarded folder not created",
		filepath.Join(tmpRoot, "works", "sprints", "discarded", id))

	// confirm audit event includes db_only: true.
	events, qErr := audit.QueryEvents(audit.QueryFilter{
		EventType:  "sprint.discarded",
		EntityType: "sprint",
		EntityID:   id,
	})
	if qErr != nil {
		t.Fatalf("QueryEvents: %v", qErr)
	}
	if len(events) == 0 {
		t.Fatalf("sprint.discarded event not found (entity=%s). tmpRoot=%s", id, tmpRoot)
	}
	if events[0].Details["db_only"] != true {
		t.Errorf("audit db_only expected true, got %v", events[0].Details["db_only"])
	}
}
