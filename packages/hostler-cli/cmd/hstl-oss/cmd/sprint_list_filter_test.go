package cmd

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"testing"
)

// Regression tests for applyT237Filters.
// Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.

func seedT237Sprints() []domain.SprintRecord {
	return []domain.SprintRecord{
		{SprintID: "sprint-01", Title: "A", Status: "completed", StartedAt: "2026-03-01", CompletedAt: "2026-03-05"},
		{SprintID: "sprint-02", Title: "B", Status: "completed", StartedAt: "2026-03-10", CompletedAt: "2026-03-15"},
		{SprintID: "sprint-03", Title: "C", Status: "completed", StartedAt: "2026-04-01", CompletedAt: "2026-04-05"},
		{SprintID: "sprint-04", Title: "D", Status: "active", StartedAt: "2026-04-10"},
		{SprintID: "sprint-05", Title: "E", Status: "backlog"},
	}
}

func TestT237_ApplyFilters_Since(t *testing.T) {
	got := applyT237Filters(seedT237Sprints(), "2026-04-01", "", 0, 0)
	// completed_at >= 2026-04-01: sprint-03 (04-05), sprint-04 (started 04-10, completed_at "" -> key=04-10) = 2 entries
	if len(got) != 2 {
		t.Errorf("since=2026-04-01 -> %d entries (want 2)", len(got))
	}
}

func TestT237_ApplyFilters_Until(t *testing.T) {
	got := applyT237Filters(seedT237Sprints(), "", "2026-03-31", 0, 0)
	// <= 2026-03-31: sprint-01 (03-05), sprint-02 (03-15) = 2 entries
	if len(got) != 2 {
		t.Errorf("until=2026-03-31 -> %d entries (want 2)", len(got))
	}
}

func TestT237_ApplyFilters_Last(t *testing.T) {
	got := applyT237Filters(seedT237Sprints(), "", "", 2, 0)
	if len(got) != 2 {
		t.Fatalf("--last 2 -> %d entries (want 2)", len(got))
	}
	// time-desc: between sprint-04 (started 04-10, no completed -> key=04-10) and
	// sprint-03 (completed 04-05) the later one comes first. sprint-04=04-10 > sprint-03=04-05.
	if got[0].SprintID != "sprint-04" {
		t.Errorf("--last priority -> sprint-04 (04-10) should come first (got %s)", got[0].SprintID)
	}
}

func TestT237_ApplyFilters_Limit(t *testing.T) {
	got := applyT237Filters(seedT237Sprints(), "", "", 0, 3)
	if len(got) != 3 {
		t.Errorf("--limit 3 -> %d entries (want 3)", len(got))
	}
}

func TestT237_ApplyFilters_LastWinsOverLimit(t *testing.T) {
	got := applyT237Filters(seedT237Sprints(), "", "", 2, 4)
	if len(got) != 2 {
		t.Errorf("--last 2 --limit 4 -> %d entries (want 2, last wins)", len(got))
	}
}
