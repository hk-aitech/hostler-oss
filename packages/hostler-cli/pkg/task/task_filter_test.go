package task

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/adapters/inmemory"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// T236 (Sprint-16) — ListTasksFiltered regression tests.
//
// Verifies each ports.TaskListFilter field (priority/type/since/until/last/
// limit) operates independently and that combinations are AND-ed.
// InMemoryStore is used to confirm semantic equivalence with SQLiteStore.
// Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.

func seedT236Tasks(t *testing.T, mem *inmemory.InMemoryStore) {
	t.Helper()
	seed := []struct {
		rec        ports.TaskRecord
		updated_at string
	}{
		{ports.TaskRecord{TaskID: "T001", Title: "A", Type: "feature", Status: "todo", Priority: "p0", Estimate: "S", CreatedAt: "2026-03-01"}, "2026-03-01"},
		{ports.TaskRecord{TaskID: "T002", Title: "B", Type: "bugfix", Status: "todo", Priority: "p1", Estimate: "M", CreatedAt: "2026-04-01"}, "2026-04-01"},
		{ports.TaskRecord{TaskID: "T003", Title: "C", Type: "feature", Status: "done", Priority: "p1", Estimate: "L", CreatedAt: "2026-04-05"}, "2026-04-05"},
		{ports.TaskRecord{TaskID: "T004", Title: "D", Type: "bugfix", Status: "todo", Priority: "p2", Estimate: "XS", CreatedAt: "2026-04-15"}, "2026-04-15"},
		{ports.TaskRecord{TaskID: "T005", Title: "E", Type: "docs", Status: "done", Priority: "p3", Estimate: "XS", CreatedAt: "2026-04-20"}, "2026-04-20"},
	}
	for i := range seed {
		if err := mem.CreateTask(&seed[i].rec); err != nil {
			t.Fatal(err)
		}
		// T236: CreateTask pins UpdatedAt to now → inject fixed time for tests
		if err := mem.SetTaskTimestampsForTest(seed[i].rec.TaskID, seed[i].rec.CreatedAt, seed[i].updated_at); err != nil {
			t.Fatal(err)
		}
	}
}

func TestT236_ListFiltered_Priority(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	p := "p1"
	result, err := mem.ListTasksFiltered(ports.TaskListFilter{Priority: &p})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tasks) != 2 {
		t.Errorf("priority=p1 → %d items (want 2)", len(result.Tasks))
	}
}

func TestT236_ListFiltered_Type(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	ty := "bugfix"
	result, _ := mem.ListTasksFiltered(ports.TaskListFilter{Type: &ty})
	if len(result.Tasks) != 2 {
		t.Errorf("type=bugfix → %d items (want 2: T002/T004)", len(result.Tasks))
	}
}

func TestT236_ListFiltered_Since(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	since := "2026-04-01"
	result, _ := mem.ListTasksFiltered(ports.TaskListFilter{Since: &since})
	// blank UpdatedAt → fallback to CreatedAt. created ≥ 2026-04-01:
	// T002/T003/T004/T005 = 4
	if len(result.Tasks) != 4 {
		t.Errorf("since=2026-04-01 → %d items (want 4)", len(result.Tasks))
	}
}

func TestT236_ListFiltered_Until(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	until := "2026-03-31"
	result, _ := mem.ListTasksFiltered(ports.TaskListFilter{Until: &until})
	// created ≤ 2026-03-31: T001 only
	if len(result.Tasks) != 1 {
		t.Errorf("until=2026-03-31 → %d items (want 1: T001)", len(result.Tasks))
	}
}

func TestT236_ListFiltered_Last(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	result, _ := mem.ListTasksFiltered(ports.TaskListFilter{Last: 2})
	if len(result.Tasks) != 2 {
		t.Fatalf("--last 2 → %d items (want 2)", len(result.Tasks))
	}
	// time-desc → T005 (04-20), T004 (04-15)
	if result.Tasks[0].TaskID != "T005" || result.Tasks[1].TaskID != "T004" {
		ids := []string{result.Tasks[0].TaskID, result.Tasks[1].TaskID}
		t.Errorf("--last 2 → %v (want [T005, T004])", ids)
	}
}

func TestT236_ListFiltered_Limit(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	result, _ := mem.ListTasksFiltered(ports.TaskListFilter{Limit: 3})
	if len(result.Tasks) != 3 {
		t.Errorf("--limit 3 → %d items (want 3)", len(result.Tasks))
	}
}

func TestT236_ListFiltered_LastWinsOverLimit(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	// when both --last and --limit are given → --last wins (time-desc + truncate)
	result, _ := mem.ListTasksFiltered(ports.TaskListFilter{Last: 2, Limit: 4})
	if len(result.Tasks) != 2 {
		t.Errorf("--last 2 --limit 4 → %d items (want 2, last takes precedence)", len(result.Tasks))
	}
	if len(result.Tasks) > 0 && result.Tasks[0].TaskID != "T005" {
		t.Errorf("--last takes precedence → newest T005 first (got %s)", result.Tasks[0].TaskID)
	}
}

func TestT236_ListFiltered_PriorityAndTypeCombo(t *testing.T) {
	mem := inmemory.NewForTest()
	seedT236Tasks(t, mem)

	p := "p1"
	ty := "bugfix"
	result, _ := mem.ListTasksFiltered(ports.TaskListFilter{Priority: &p, Type: &ty})
	// p1 AND bugfix: T002 only
	if len(result.Tasks) != 1 || result.Tasks[0].TaskID != "T002" {
		ids := make([]string, len(result.Tasks))
		for i, tk := range result.Tasks {
			ids[i] = tk.TaskID
		}
		t.Errorf("--priority p1 --type bugfix → %v (want [T002])", ids)
	}
}
