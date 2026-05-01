package cmd

import "testing"

// TestT614_ContainsGoCodeKeyword - unit tests for Go code keyword detection.
func TestT614_ContainsGoCodeKeyword(t *testing.T) {
	cases := []struct {
		title   string
		wantOK  bool
		wantKey string
	}{
		{"edit task.go file", true, ".go"},
		{"refactor func ParseTitle", true, "func "},
		{"add struct Config field", true, "struct "},
		{"clean up cli/pkg/task package", true, "cli/pkg/"},
		{"split cli/cmd/foo subcommands", true, "cli/cmd/"},
		{"add interface Writer", true, "interface "},
		{"case insensitive: .GO file", true, ".go"},
		{"docs update only", false, ""},
		{"write Sprint retro", false, ""},
		{"clean up BACKLOG.md", false, ""},
	}

	for _, tc := range cases {
		kw, ok := containsGoCodeKeyword(tc.title)
		if ok != tc.wantOK {
			t.Errorf("[%s] ok=%v, want %v", tc.title, ok, tc.wantOK)
			continue
		}
		if ok && kw != tc.wantKey {
			t.Errorf("[%s] keyword=%q, want %q", tc.title, kw, tc.wantKey)
		}
	}
}

// TestT614_GoCodeWarnTypes - check the goCodeWarnTypes set.
func TestT614_GoCodeWarnTypes(t *testing.T) {
	warnExpected := []string{"docs", "chore"}
	warnNotExpected := []string{"feature", "refactor", "bugfix", "hotfix", "infra", "test", "spike"}

	for _, ty := range warnExpected {
		if !goCodeWarnTypes[ty] {
			t.Errorf("goCodeWarnTypes[%q] should be true", ty)
		}
	}
	for _, ty := range warnNotExpected {
		if goCodeWarnTypes[ty] {
			t.Errorf("goCodeWarnTypes[%q] should be false", ty)
		}
	}
}
