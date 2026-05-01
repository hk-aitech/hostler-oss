// Package task — T392 (Sprint-32) unit tests.
//
// Edge-case checks for applyGitLogFallback and normalizeSince.
// Scenarios that need real git calls are integration-level, so this file
// covers only the pure-function parts (normalizeSince, fallback-skip
// paths). End-to-end regression of the real git-log fallback is reproduced
// against a real repo in the Sprint-32 T396 dev-verification Task.
package task

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
)

// TestT392_NormalizeSince_Valid — YYYY-MM-DD prefix parses successfully.
func TestT392_NormalizeSince_Valid(t *testing.T) {
	cases := map[string]string{
		"2026-04-23":                      "2026-04-23",
		"2026-04-23T10:00:00Z":            "2026-04-23",
		`"2026-04-23"`:                    "2026-04-23",
		"  2026-04-23  ":                  "2026-04-23",
		"2020-01-01T00:00:00+09:00extras": "2020-01-01",
	}
	for in, want := range cases {
		got := normalizeSince(in)
		if got != want {
			t.Errorf("normalizeSince(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestT392_NormalizeSince_Invalid — invalid input yields empty string.
func TestT392_NormalizeSince_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"   ",
		"2026",
		"not-a-date",
		"2026/04/23",
		"2026-4-23",  // single-digit month → format is strict
		"20260423",   // no hyphens
		"2026-04-2a", // non-digit
	}
	for _, in := range invalid {
		if got := normalizeSince(in); got != "" {
			t.Errorf("normalizeSince(%q) = %q, want empty", in, got)
		}
	}
}

// TestT392_ApplyGitLogFallback_EmptyInputs — no-op on empty missing or
// empty createdAt.
func TestT392_ApplyGitLogFallback_EmptyInputs(t *testing.T) {
	// empty createdAt
	missing := []domain.MissingFile{{Path: "foo.go", Reason: "no git diff"}}
	still, recovered := applyGitLogFallback("", missing)
	if len(still) != 1 || still[0].Path != "foo.go" {
		t.Errorf("missing not preserved on empty createdAt: %v", still)
	}
	if len(recovered) != 0 {
		t.Errorf("recovered should be empty on empty createdAt: %v", recovered)
	}

	// empty missing
	still2, rec2 := applyGitLogFallback("2026-04-23", nil)
	if len(still2) != 0 || len(rec2) != 0 {
		t.Errorf("non-empty result on empty missing: still=%v rec=%v", still2, rec2)
	}
}

// TestT392_ApplyGitLogFallback_InvalidCreatedAt — skip when createdAt is
// unparseable.
func TestT392_ApplyGitLogFallback_InvalidCreatedAt(t *testing.T) {
	missing := []domain.MissingFile{{Path: "foo.go", Reason: "no git diff"}}
	still, recovered := applyGitLogFallback("not-a-date", missing)
	if len(still) != 1 {
		t.Errorf("missing not preserved on invalid createdAt: %v", still)
	}
	if len(recovered) != 0 {
		t.Errorf("recovered should be empty on invalid createdAt: %v", recovered)
	}
}
