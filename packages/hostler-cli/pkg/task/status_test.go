// Package task — T391 (Sprint-32) unit tests.
//
// Verifies that Canonicalize normalises various user-input variants into the
// canonical form ("in-progress", etc.). Regression guard for
// ISS-20260422-011.
package task

import "testing"

// TestT391_Canonicalize_InProgressAliases — exhaustively covers the four
// in-progress aliases. The actually-observed problem case was "in_progress"
// (snake_case).
func TestT391_Canonicalize_InProgressAliases(t *testing.T) {
	// CamelCase (InProgress) is deliberately omitted — it never appeared in
	// real reports and lower-casing + underscore/space handling already
	// covers the user/AI usage patterns.
	cases := []string{
		"in-progress",
		"in_progress",
		"  in-progress  ",
		"IN-PROGRESS",
		"IN_PROGRESS",
		"in progress",
	}
	for _, raw := range cases {
		got, err := Canonicalize(raw)
		if err != nil {
			t.Errorf("Canonicalize(%q) error: %v", raw, err)
			continue
		}
		if got != StatusInProgress {
			t.Errorf("Canonicalize(%q) = %q, want %q", raw, got, StatusInProgress)
		}
	}
}

// TestT391_Canonicalize_ValidStatuses — every valid canonical value must
// pass through unchanged (idempotence).
func TestT391_Canonicalize_ValidStatuses(t *testing.T) {
	valid := []string{StatusTodo, StatusInProgress, StatusDone, StatusBlocked, StatusReopened}
	for _, v := range valid {
		got, err := Canonicalize(v)
		if err != nil {
			t.Errorf("Canonicalize(%q) error: %v", v, err)
		}
		if got != v {
			t.Errorf("idempotence violation: Canonicalize(%q) = %q", v, got)
		}
	}
}

// TestT391_Canonicalize_EmptyInput — empty input means "no filter" and
// returns ("", nil).
func TestT391_Canonicalize_EmptyInput(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t"} {
		got, err := Canonicalize(raw)
		if err != nil {
			t.Errorf("empty input %q error: %v", raw, err)
		}
		if got != "" {
			t.Errorf("empty input %q = %q, want %q", raw, got, "")
		}
	}
}

// TestT391_Canonicalize_Unknown — unrecognised values return an error.
func TestT391_Canonicalize_Unknown(t *testing.T) {
	unknown := []string{"pending", "waiting", "cancelled", "wip", "in_progresss"}
	for _, raw := range unknown {
		_, err := Canonicalize(raw)
		if err == nil {
			t.Errorf("Canonicalize(%q) expected error, got nil", raw)
		}
	}
}

// TestT391_CanonicalizeOrDefault — on error return the original value
// (backwards-compat path).
func TestT391_CanonicalizeOrDefault(t *testing.T) {
	if got := CanonicalizeOrDefault("in_progress"); got != "in-progress" {
		t.Errorf("CanonicalizeOrDefault in_progress = %q, want in-progress", got)
	}
	if got := CanonicalizeOrDefault("bogus"); got != "bogus" {
		t.Errorf("CanonicalizeOrDefault bogus = %q, want bogus (fallback)", got)
	}
}
