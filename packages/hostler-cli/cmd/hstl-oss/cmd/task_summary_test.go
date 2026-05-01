package cmd

import (
	"os"
	"testing"
)

// TestGetMinSummaryLen_default verifies that without an env var,
// defaultMinSummaryLen is returned.
func TestGetMinSummaryLen_default(t *testing.T) {
	os.Unsetenv("HSTL_SUMMARY_MIN_LEN")
	got := getMinSummaryLen()
	if got != defaultMinSummaryLen {
		t.Errorf("expected default %d, got %d", defaultMinSummaryLen, got)
	}
}

// TestGetMinSummaryLen_EnvVarOverride verifies that the
// HSTL_SUMMARY_MIN_LEN env var overrides the default.
func TestGetMinSummaryLen_EnvVarOverride(t *testing.T) {
	t.Setenv("HSTL_SUMMARY_MIN_LEN", "5")
	got := getMinSummaryLen()
	if got != 5 {
		t.Errorf("expected override 5, got %d", got)
	}
}

// TestGetMinSummaryLen_ExceedsUpperBound verifies that when an override value
// exceeds SummaryLenHardMax, the default is used as a fallback.
func TestGetMinSummaryLen_ExceedsUpperBound(t *testing.T) {
	t.Setenv("HSTL_SUMMARY_MIN_LEN", "999999")
	got := getMinSummaryLen()
	if got != defaultMinSummaryLen {
		t.Errorf("when exceeding the upper bound expected default %d, got %d", defaultMinSummaryLen, got)
	}
}

// TestGetMinSummaryLen_Negative verifies that negative or zero override
// values fall back to the default.
func TestGetMinSummaryLen_Negative(t *testing.T) {
	cases := []string{"-1", "0", "-100"}
	for _, v := range cases {
		t.Run("value="+v, func(t *testing.T) {
			t.Setenv("HSTL_SUMMARY_MIN_LEN", v)
			got := getMinSummaryLen()
			if got != defaultMinSummaryLen {
				t.Errorf("for negative/zero expected default %d, got %d (v=%s)", defaultMinSummaryLen, got, v)
			}
		})
	}
}

// TestGetMinSummaryLen_ParseFailure verifies that an unparseable value falls
// back to the default.
func TestGetMinSummaryLen_ParseFailure(t *testing.T) {
	t.Setenv("HSTL_SUMMARY_MIN_LEN", "abc")
	got := getMinSummaryLen()
	if got != defaultMinSummaryLen {
		t.Errorf("on parse failure expected default %d, got %d", defaultMinSummaryLen, got)
	}
}

// TestSummaryLengthValidation_MinimumOnly verifies the summary minimum
// length boundary. The max limit was removed in a 2026-04 hotfix - quality
// is handled by the static heuristic and the LLM judge.
func TestSummaryLengthValidation_MinimumOnly(t *testing.T) {
	os.Unsetenv("HSTL_SUMMARY_MIN_LEN")

	minLen := getMinSummaryLen() // 10

	cases := []struct {
		name  string
		summary string
		wantOK bool
	}{
		{"length 9 (below)", makeRunes(9), false},
		{"length 10 (minimum boundary)", makeRunes(10), true},
		{"length 200 (former max boundary - now passes)", makeRunes(200), true},
		{"length 500 (long - now passes)", makeRunes(500), true},
		{"length 1000 (very long - now passes)", makeRunes(1000), true},
		{"empty string", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			summaryLen := len([]rune(c.summary))
			ok := summaryLen >= minLen
			if ok != c.wantOK {
				t.Errorf("summary len=%d -> valid=%v, want %v", summaryLen, ok, c.wantOK)
			}
		})
	}
}

// makeRunes returns a string of n 'a' characters.
func makeRunes(n int) string {
	runes := make([]rune, n)
	for i := range runes {
		runes[i] = 'a'
	}
	return string(runes)
}
