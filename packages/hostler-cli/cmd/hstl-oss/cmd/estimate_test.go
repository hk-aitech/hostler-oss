// Regression tests for the detectMigrationEstimateRisk heuristic.
package cmd

import "testing"

func TestT442_DetectMigrationEstimateRisk(t *testing.T) {
	cases := []struct {
		name         string
		title        string
		summary      string
		estimate     string
		wantNonEmpty bool
	}{
		{name: "migrate+XS WARN", title: "Go module path migrate", summary: "migrate every import", estimate: "XS", wantNonEmpty: true},
		{name: "cutover+XS WARN", title: "config dir cutover Phase 2", summary: "switch writer code", estimate: "XS", wantNonEmpty: true},
		{name: "migrate+S OK", title: "Go module migrate", summary: "tidy up paths", estimate: "S", wantNonEmpty: false},
		{name: "keyword none XS OK", title: "fix task list sort bug", summary: "apply sort rules", estimate: "XS", wantNonEmpty: false},
		{name: "case-insensitive MIGRATE xs", title: "MIGRATE LEGACY DB", summary: "move database", estimate: "xs", wantNonEmpty: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectMigrationEstimateRisk(c.title, c.summary, c.estimate)
			if c.wantNonEmpty && got == "" {
				t.Errorf("want non-empty WARN, got=empty")
			}
			if !c.wantNonEmpty && got != "" {
				t.Errorf("want empty, got=%q", got)
			}
		})
	}
}
