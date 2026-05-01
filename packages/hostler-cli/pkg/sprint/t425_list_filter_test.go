package sprint_test

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
)

// T425 (Sprint-36) — sprint list filter regression.
//
// Contract:
//   - ""       (default): exclude discarded
//   - "all"    : everything (including discarded) — T423 regression fix
//   - "discarded" : discarded only
//   - "backlog|active|completed" : that status only (existing behaviour)

// seedSprintsMixed inserts 4 statuses of Sprint directly into DB.
func seedSprintsMixed(t *testing.T) {
	t.Helper()
	type seed struct {
		id     string
		title  string
		status string
	}
	for _, sp := range []seed{
		{"sprint-t425-b", "backlog sample", "backlog"},
		{"sprint-t425-a", "active sample", "active"},
		{"sprint-t425-c", "completed sample", "completed"},
		{"sprint-t425-d", "discarded sample", "discarded"},
	} {
		if err := sprint.SaveToDB(sp.id, sp.title, "goal", sp.status, "works/sprints/"+sp.status+"/"+sp.id, nil, nil); err != nil {
			t.Fatalf("SaveToDB %s: %v", sp.id, err)
		}
	}
}

func TestT425_ListFromDB_Default_ExcludesDiscarded(t *testing.T) {
	_ = setupDB(t)
	seedSprintsMixed(t)

	recs, err := sprint.ListFromDB("")
	if err != nil {
		t.Fatalf("ListFromDB: %v", err)
	}
	for _, r := range recs {
		if r.Status == "discarded" {
			t.Errorf("default lookup includes discarded: %s (%s)", r.SprintID, r.Status)
		}
	}
	if len(recs) != 3 {
		t.Errorf("default expects 3 (backlog/active/completed), got %d", len(recs))
	}
}

func TestT425_ListFromDB_All_IncludesDiscarded(t *testing.T) {
	_ = setupDB(t)
	seedSprintsMixed(t)

	recs, err := sprint.ListFromDB("all")
	if err != nil {
		t.Fatalf("ListFromDB all: %v", err)
	}
	if len(recs) != 4 {
		t.Errorf("expected 4 records on 'all' (T423 regression fix), got %d", len(recs))
	}
	// at least one discarded must be included.
	var hasDiscarded bool
	for _, r := range recs {
		if r.Status == "discarded" {
			hasDiscarded = true
		}
	}
	if !hasDiscarded {
		t.Error("'all' filter is missing discarded")
	}
}

func TestT425_ListFromDB_Discarded_OnlyDiscarded(t *testing.T) {
	_ = setupDB(t)
	seedSprintsMixed(t)

	recs, err := sprint.ListFromDB("discarded")
	if err != nil {
		t.Fatalf("ListFromDB discarded: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("expected 1 'discarded' only, got %d", len(recs))
	}
	for _, r := range recs {
		if r.Status != "discarded" {
			t.Errorf("'discarded' filter includes %s status: %s", r.Status, r.SprintID)
		}
	}
}

func TestT425_ListFromDB_Active_Unchanged(t *testing.T) {
	_ = setupDB(t)
	seedSprintsMixed(t)

	recs, err := sprint.ListFromDB("active")
	if err != nil {
		t.Fatalf("ListFromDB active: %v", err)
	}
	if len(recs) != 1 || recs[0].Status != "active" {
		t.Errorf("expected 1 'active' only, got %d records", len(recs))
	}
}
