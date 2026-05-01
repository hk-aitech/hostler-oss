package cmd

import (
	"strings"
	"testing"
)

// T532: verifies that contextIssue renders with the same format as brief.
func TestT532_formatContextText_issuesRendering(t *testing.T) {
	cases := []struct {
		name      string
		issues    []contextIssue
		wantLines []string
	}{
		{
			name:      "no warnings",
			issues:    nil,
			wantLines: []string{"[issues] none"},
		},
		{
			name: "single placeholder warning",
			issues: []contextIssue{
				{Level: "WARN", Message: "placeholder body still present: 3 - T001, T002, T003"},
			},
			wantLines: []string{
				"[issues] [WARN] placeholder body still present: 3 - T001, T002, T003",
			},
		},
		{
			name: "multiple warnings rendered per line",
			issues: []contextIssue{
				{Level: "WARN", Message: "Sprint ready to complete: 1 - sprint-61"},
				{Level: "WARN", Message: "placeholder body still present: 2 - T100, T101"},
			},
			wantLines: []string{
				"[issues] [WARN] Sprint ready to complete: 1 - sprint-61",
				"[issues] [WARN] placeholder body still present: 2 - T100, T101",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &contextData{
				Project: contextProject{Name: "test", Branch: "dev"},
				Rules:   "test-rules",
				Issues:  tc.issues,
			}
			out := formatContextText(ctx)
			for _, line := range tc.wantLines {
				if !strings.Contains(out, line) {
					t.Errorf("missing line %q in output:\n%s", line, out)
				}
			}
		})
	}
}

// T532: verify that contextData.Issues is now a []contextIssue slice
// (regression guard against the previous int usage).
func TestT532_contextData_IssuesIsSlice(t *testing.T) {
	ctx := &contextData{}
	ctx.Issues = []contextIssue{{Level: "WARN", Message: "X"}}
	if len(ctx.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(ctx.Issues))
	}
	if ctx.Issues[0].Level != "WARN" || ctx.Issues[0].Message != "X" {
		t.Errorf("issue fields wrong: %+v", ctx.Issues[0])
	}
}
