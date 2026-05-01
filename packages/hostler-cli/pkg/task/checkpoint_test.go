package task

import (
	"errors"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/inmemory"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// T549 (Sprint-82) — task checkpoint regression tests.

func setupT549(t *testing.T) *inmemory.InMemoryStore {
	t.Helper()
	mem := inmemory.NewForTest()
	store.Init(mem, mem)
	t.Cleanup(store.Reset)
	return mem
}

func insertTestTask(t *testing.T, mem *inmemory.InMemoryStore, id, status string) {
	t.Helper()
	if err := mem.CreateTask(&ports.TaskRecord{
		TaskID: id, Title: "test", Type: "feature", Status: "todo",
		Priority: "p2", Estimate: "S",
	}); err != nil {
		t.Fatal(err)
	}
	if status != "todo" {
		if err := mem.UpdateTaskStatus(id, status); err != nil {
			t.Fatal(err)
		}
	}
}

func TestT549_Checkpoint_RotatesTicket(t *testing.T) {
	mem := setupT549(t)
	const tid = "T549TEST"
	insertTestTask(t, mem, tid, "in-progress")
	_ = mem.SetTaskWorkTicket(tid, "WT-T549TEST-original")
	_ = mem.InsertContextAck("WT-T549TEST-original", "old-hash", 100)

	result, err := Checkpoint(tid, "intermediate context re-confirmation")
	if err != nil {
		t.Fatal(err)
	}
	if result.OldTicket != "WT-T549TEST-original" {
		t.Errorf("old=%q, expected original", result.OldTicket)
	}
	if result.NewTicket == "" || result.NewTicket == result.OldTicket {
		t.Errorf("new ticket not rotated: new=%q, old=%q", result.NewTicket, result.OldTicket)
	}

	stored, _ := mem.GetTaskWorkTicket(tid)
	if stored != result.NewTicket {
		t.Errorf("DB work_ticket=%q, expected %q", stored, result.NewTicket)
	}

	okOld, _ := mem.HasContextAck(result.NewTicket, "old-hash")
	if okOld {
		t.Error("new ticket matches the old hash — checkpoint invalidation failed")
	}
}

func TestT549_Checkpoint_RejectsNonInProgress(t *testing.T) {
	mem := setupT549(t)
	const tid = "T549TODO"
	insertTestTask(t, mem, tid, "todo")

	_, err := Checkpoint(tid, "")
	if err == nil {
		t.Fatal("checkpoint succeeded from todo state — expected reject")
	}
	var rje *apperr.RejectedError
	if !errors.As(err, &rje) {
		t.Errorf("expected RejectedError, got %T: %v", err, err)
	}
}

func TestT549_Checkpoint_NotFoundReturnsError(t *testing.T) {
	setupT549(t)

	_, err := Checkpoint("T999NOTEXIST", "")
	if err == nil {
		t.Fatal("checkpoint succeeded on non-existent Task — expected NotFound")
	}
	var nfe *apperr.NotFoundError
	if !errors.As(err, &nfe) {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}
