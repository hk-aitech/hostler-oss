// Package audit — Restore regression tests.
package audit

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/sqlite"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// restoreTestSchema is the minimal audit_events schema for tests.
// prev_hash + event_hash columns are added per the hash-chain feature.
const restoreTestSchema = `
CREATE TABLE audit_events (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp   TEXT    NOT NULL DEFAULT (datetime('now')),
	event_type  TEXT    NOT NULL,
	entity_type TEXT    NOT NULL,
	entity_id   TEXT    NOT NULL,
	actor       TEXT    NOT NULL,
	details     TEXT,
	session_id  TEXT,
	prev_hash   TEXT,
	event_hash  TEXT
);
`

func setupRestoreEnv(t *testing.T) func() {
	t.Helper()
	tmp := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "test")

	sprintDir := filepath.Join(tmp, "works", "sprints", "completed", "sprint-01")
	if err := os.MkdirAll(filepath.Join(sprintDir, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sprintDir, "SPRINT.md"), []byte("# sprint-01"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sprintDir, "tasks", "T100-test.md"),
		[]byte("---\nid: T100\nstatus: done\n---\n\n# T100"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sprintDir, "tasks", "T101-todo.md"),
		[]byte("---\nid: T101\nstatus: todo\n---\n\n# T101"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "seed fixture")

	dbPath := filepath.Join(tmp, "hstl.db")
	dbh, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dbh.Exec(restoreTestSchema); err != nil {
		t.Fatal(err)
	}
	s := sqlite.New(dbh)
	store.Init(s, s)

	t.Chdir(tmp)

	return func() {
		_ = dbh.Close()
	}
}

// qf is a frequently used QueryAuditEvents filter builder.
func qf(eventType, entityType, entityID string) ports.AuditQueryFilter {
	return ports.AuditQueryFilter{
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Limit:      10,
	}
}

func TestT425_Restore_DryRun_NoWrite(t *testing.T) {
	teardown := setupRestoreEnv(t)
	defer teardown()

	result, err := Restore(true)
	if err != nil {
		t.Fatalf("Restore dry: %v", err)
	}
	if !result.DryRun {
		t.Error("DryRun flag not set")
	}
	if result.ScannedSprints != 1 {
		t.Errorf("ScannedSprints = %d, expected 1", result.ScannedSprints)
	}
	if result.Restored == 0 {
		t.Error("dry-run also produced 0 restoration targets — suspect git log failure")
	}

	gs := store.Get()
	events, err := gs.QueryAuditEvents(qf("sprint.completed", "sprint", "sprint-01"))
	if err != nil {
		t.Fatalf("QueryAuditEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("dry-run actually inserted: %d entries", len(events))
	}
}

func TestT425_Restore_Apply_InsertsEvents(t *testing.T) {
	teardown := setupRestoreEnv(t)
	defer teardown()

	result, err := Restore(false)
	if err != nil {
		t.Fatalf("Restore apply: %v", err)
	}
	if result.Restored < 2 {
		t.Errorf("Restored = %d, expected >= 2 (sprint + task)", result.Restored)
	}

	gs := store.Get()
	sprintEvents, _ := gs.QueryAuditEvents(qf("sprint.completed", "sprint", "sprint-01"))
	if len(sprintEvents) != 1 {
		t.Errorf("sprint.completed events = %d, expected 1", len(sprintEvents))
	}
	taskEvents, _ := gs.QueryAuditEvents(qf("task.completed", "task", "T100"))
	if len(taskEvents) != 1 {
		t.Errorf("task.completed T100 events = %d, expected 1", len(taskEvents))
	}
	noTaskEvents, _ := gs.QueryAuditEvents(qf("task.completed", "task", "T101"))
	if len(noTaskEvents) != 0 {
		t.Errorf("T101 (status=todo) was restored: %d entries", len(noTaskEvents))
	}
}

func TestT425_Restore_Idempotent_SkipsExisting(t *testing.T) {
	teardown := setupRestoreEnv(t)
	defer teardown()

	r1, err := Restore(false)
	if err != nil {
		t.Fatalf("1st Restore: %v", err)
	}
	firstRestored := r1.Restored

	r2, err := Restore(false)
	if err != nil {
		t.Fatalf("2nd Restore: %v", err)
	}
	if r2.Restored != 0 {
		t.Errorf("2nd restore inserted new entries: %d (1st=%d) — idempotent violation", r2.Restored, firstRestored)
	}
	if r2.SkippedExists == 0 {
		t.Error("2nd run produced SkippedExists=0 — natural-key dedupe not working")
	}
}
