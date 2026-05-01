package sprint

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
)

func TestRunTierCheck_Solo_Pass(t *testing.T) {
	// solo (max=3, capacity≤8) — 1 task XS passes normally.
	// LoadSprintEstimates depends on the filesystem, so this test exercises
	// only the CheckTier flow directly.
	r := CheckTier(TierSolo, []domain.Estimate{domain.EstimateXS}, TierDefaults{})
	if r.IsBlocked() {
		t.Errorf("solo 1 task XS should pass, got violations=%v", r.Violations)
	}
}

func TestRunTierCheck_Standard_CapacityOver(t *testing.T) {
	// standard (max=7, capacity≤20) — XL×3=24pt exceeds, BLOCK
	r := CheckTier(TierStandard, []domain.Estimate{domain.EstimateXL, domain.EstimateXL, domain.EstimateXL}, TierDefaults{})
	if !r.IsBlocked() {
		t.Errorf("standard XL×3=24pt should BLOCK, got pass")
	}
}

func TestFormatTierViolationHint(t *testing.T) {
	r := CheckTier(TierSolo, []domain.Estimate{
		domain.EstimateXS, domain.EstimateXS, domain.EstimateXS, domain.EstimateXS,
	}, TierDefaults{})
	hint := FormatTierViolationHint(r)
	if hint == "" {
		t.Errorf("FormatTierViolationHint empty for blocked result")
	}
	// hint must include the core keywords
	for _, kw := range []string{"Sprint Tier violation", "solo", "--force", "size_mode"} {
		if !contains(hint, kw) {
			t.Errorf("hint missing keyword %q: %s", kw, hint)
		}
	}
}

// contains — alias of strings.Contains (improves test readability).
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) == 0) ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}

func TestResolveCompleteMode(t *testing.T) {
	cases := []struct {
		name    string
		cliLite bool
		envVal  string
		cfg     string
		want    string
	}{
		{"CLI flag wins", true, "", "", "lite"},
		{"CLI flag wins — other values ignored", true, "confirm", "confirm", "lite"},
		{"env wins (CLI unset)", false, "lite", "", "lite"},
		{"config fallback", false, "", "lite", "lite"},
		{"default confirm", false, "", "", "confirm"},
		{"unknown value → confirm", false, "unknown", "", "confirm"},
		{"env confirm explicit", false, "confirm", "lite", "confirm"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveCompleteMode(c.cliLite, c.envVal, c.cfg)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
