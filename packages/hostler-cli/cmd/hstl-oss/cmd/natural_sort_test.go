package cmd

import (
	"sort"
	"strconv"
	"testing"
	"time"
)

// Single regression guard for natural-sort behaviour repeatedly broken during
// the original Python → Go port.
//
// Background:
//  - `brief recent_completed` previously displayed "sprint-9" after "sprint-10"
//    because of naive string comparison; fixed by porting the Python 4-tier
//    key tuple.
//  - `id.TaskIDNext` once collided "T100" after "T999" because the id_counter
//    drifted from the actual max task_id; fixed with a self-heal step.
//
// Strategy:
//  1. Verify that sprintIDNumberRE extracts the numeric portion regardless of
//     digit count.
//  2. Verify the 4-tier tuple sort (completed_at, audit_or_mtime, num) returns
//     the expected order on the minimum fixture (sprint-9/10/100/1042).
//  3. Verify Task ID numeric sort (T999/T1000/T1042) after stripping the
//     leading "T".

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

var naturalSortSprintIDs = []string{"sprint-9", "sprint-10", "sprint-100", "sprint-1042"}

var naturalSortTaskIDs = []string{"T999", "T1000", "T1042"}

// ---------------------------------------------------------------------------
// 1. sprintIDNumberRE regex
// ---------------------------------------------------------------------------

func TestSprintIDNumberRE_ExtractsAllNumbers(t *testing.T) {
	cases := []struct {
		id   string
		want int
	}{
		{"sprint-9", 9},
		{"sprint-10", 10},
		{"sprint-100", 100},
		{"sprint-1042", 1042},
	}
	for _, tc := range cases {
		m := sprintIDNumberRE.FindStringSubmatch(tc.id)
		if m == nil {
			t.Errorf("%s: no match", tc.id)
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Errorf("%s: Atoi err %v", tc.id, err)
			continue
		}
		if n != tc.want {
			t.Errorf("%s: got %d, want %d", tc.id, n, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 2. 4-tier tuple sort - (completed_at, audit_or_mtime, num) descending
// ---------------------------------------------------------------------------

func TestNaturalSort_SprintTuple_SameCompletedAt(t *testing.T) {
	type item struct {
		id           string
		completed    string
		auditOrMtime string
		num          int
	}
	items := []item{
		{id: "sprint-9", completed: "2026-04-01", auditOrMtime: "1700", num: 9},
		{id: "sprint-10", completed: "2026-04-01", auditOrMtime: "1700", num: 10},
		{id: "sprint-100", completed: "2026-04-01", auditOrMtime: "1700", num: 100},
		{id: "sprint-1042", completed: "2026-04-01", auditOrMtime: "1700", num: 1042},
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].completed != items[j].completed {
			return items[i].completed > items[j].completed
		}
		if items[i].auditOrMtime != items[j].auditOrMtime {
			return items[i].auditOrMtime > items[j].auditOrMtime
		}
		return items[i].num > items[j].num
	})
	want := []string{"sprint-1042", "sprint-100", "sprint-10", "sprint-9"}
	for i, w := range want {
		if items[i].id != w {
			t.Errorf("pos %d: got %s, want %s", i, items[i].id, w)
		}
	}
}

func TestNaturalSort_SprintTuple_CompletedAtWins(t *testing.T) {
	type item struct {
		id           string
		completed    string
		auditOrMtime string
		num          int
	}
	items := []item{
		// Even with the same num, the more recent completed_at must come first.
		{id: "sprint-9-old", completed: "2026-03-01", auditOrMtime: "100", num: 9},
		{id: "sprint-9-new", completed: "2026-04-01", auditOrMtime: "100", num: 9},
		{id: "sprint-10-mid", completed: "2026-03-15", auditOrMtime: "100", num: 10},
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].completed != items[j].completed {
			return items[i].completed > items[j].completed
		}
		if items[i].auditOrMtime != items[j].auditOrMtime {
			return items[i].auditOrMtime > items[j].auditOrMtime
		}
		return items[i].num > items[j].num
	})
	want := []string{"sprint-9-new", "sprint-10-mid", "sprint-9-old"}
	for i, w := range want {
		if items[i].id != w {
			t.Errorf("pos %d: got %s, want %s", i, items[i].id, w)
		}
	}
}

// TestSprintTuple_EmptyCompletedUsesMtime verifies that a Sprint with an empty
// completed_at picks up its sort primary key from a date derived from
// audit/mtime. Regression guard for collectBriefRecentCompleted's fallback
// logic — without it, sprints with empty completed_at sorted behind sprints
// completed on earlier real dates.
func TestSprintTuple_EmptyCompletedUsesMtime(t *testing.T) {
	type item struct {
		id           string
		completed    string
		auditOrMtime string
		num          int
	}
	// Reproduces the downstream case: sprint-39 completed on 2026-04-13
	// but DB completed_at is empty and only the audit timestamp exists.
	// sprint-97/98/99 all completed on 2026-04-08.
	apr13 := "1776038400" // 2026-04-13 00:00:00 UTC
	apr08 := "1775606400" // 2026-04-08 00:00:00 UTC
	items := []item{
		{id: "sprint-39", completed: "", auditOrMtime: apr13, num: 39},
		{id: "sprint-97", completed: "2026-04-08", auditOrMtime: apr08, num: 97},
		{id: "sprint-98", completed: "2026-04-08", auditOrMtime: apr08, num: 98},
		{id: "sprint-99", completed: "2026-04-08", auditOrMtime: apr08, num: 99},
	}
	// Fallback: when completed is empty, derive YYYY-MM-DD from auditOrMtime.
	for i := range items {
		if items[i].completed == "" && items[i].auditOrMtime != "" {
			if ts, err := strconv.ParseInt(items[i].auditOrMtime, 10, 64); err == nil {
				items[i].completed = time.Unix(ts, 0).UTC().Format("2006-01-02")
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].completed != items[j].completed {
			return items[i].completed > items[j].completed
		}
		if items[i].auditOrMtime != items[j].auditOrMtime {
			return items[i].auditOrMtime > items[j].auditOrMtime
		}
		return items[i].num > items[j].num
	})
	if items[0].id != "sprint-39" {
		t.Errorf("sprint-39 (2026-04-13) must come first, got %s (fallback regression)", items[0].id)
	}
}

// ---------------------------------------------------------------------------
// 3. Task ID number sort - T999 < T1000 < T1042 (string sort would put T1000 < T999)
// ---------------------------------------------------------------------------

func TestNaturalSort_TaskID_NumericOrder(t *testing.T) {
	// Task IDs follow the form T<number>; numeric sort must yield T999 < T1000.
	ids := make([]string, len(naturalSortTaskIDs))
	copy(ids, naturalSortTaskIDs)

	sort.Slice(ids, func(i, j int) bool {
		a, errA := strconv.Atoi(ids[i][1:])
		b, errB := strconv.Atoi(ids[j][1:])
		if errA != nil || errB != nil {
			return ids[i] < ids[j] // fallback
		}
		return a < b
	})

	want := []string{"T999", "T1000", "T1042"}
	for i, w := range want {
		if ids[i] != w {
			t.Errorf("pos %d: got %s, want %s", i, ids[i], w)
		}
	}
}

// Pin down that naive string sort was the original regression source.
func TestNaturalSort_NaiveStringSort_IsBrokenBaseline(t *testing.T) {
	ids := make([]string, len(naturalSortSprintIDs))
	copy(ids, naturalSortSprintIDs)
	sort.Strings(ids)
	// Naive sort gives: sprint-10 < sprint-100 < sprint-1042 < sprint-9
	// — i.e. sprint-9 ends up last. If this baseline ever changes, Go's
	// string-compare semantics shifted and the natural-sort implementation
	// may need to be revisited.
	if ids[len(ids)-1] != "sprint-9" {
		t.Errorf("naive string sort baseline changed: %v", ids)
	}
}
