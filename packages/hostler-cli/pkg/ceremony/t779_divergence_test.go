// Package ceremony — T779 (Sprint-91) divergence early-warning unit tests.
//
// countCommitsAheadOfMain calls an external git process so it depends on
// the environment. Core logic is validated through (1) the ReadinessCheck
// response shape, (2) env-variable round-trip, and (3) the fallback path
// when git lookup fails.
package ceremony

import (
	"strings"
	"testing"
)

// TestT779_ResolveDivergenceWarn_Default — when the env var is unset the
// default 25 is used.
func TestT779_ResolveDivergenceWarn_Default(t *testing.T) {
	t.Setenv(DivergenceWarnEnv, "")
	if got := resolveDivergenceWarn(); got != divergenceWarnDefault {
		t.Errorf("expected default %d, got %d", divergenceWarnDefault, got)
	}
}

// TestT779_ResolveDivergenceWarn_Override — env-variable value applies.
func TestT779_ResolveDivergenceWarn_Override(t *testing.T) {
	t.Setenv(DivergenceWarnEnv, "10")
	if got := resolveDivergenceWarn(); got != 10 {
		t.Errorf("expected override 10, got %d", got)
	}
}

// TestT779_ResolveDivergenceWarn_Invalid — invalid values fall back to the
// default.
func TestT779_ResolveDivergenceWarn_Invalid(t *testing.T) {
	for _, raw := range []string{"abc", "-5", "0"} {
		t.Setenv(DivergenceWarnEnv, raw)
		if got := resolveDivergenceWarn(); got != divergenceWarnDefault {
			t.Errorf("invalid=%q expected default fallback %d, got %d", raw, divergenceWarnDefault, got)
		}
	}
}

// TestT779_CollectCheck_Structure — verifies ReadinessCheck return shape.
// Depends on the git environment — whatever the result, status/check
// fields must be populated.
func TestT779_CollectCheck_Structure(t *testing.T) {
	check := collectMainDivergenceCheck()
	if check == nil {
		t.Fatal("collectMainDivergenceCheck returned nil")
	}
	if check.Check == "" {
		t.Error("Check field is empty string")
	}
	if !strings.Contains(check.Check, "T779") {
		t.Errorf("Check has no T779 tag: %q", check.Check)
	}
	switch check.Status {
	case "pass", "warn", "info":
		// OK
	default:
		t.Errorf("Status not in expected set: %q", check.Status)
	}
}

// TestT779_CollectCheck_EnvOverride — lowering the WARN threshold flips
// pass to warn. Assuming the current repo's origin/main..HEAD is 0 ~
// several commits, dropping the threshold to >=1 may flip status. This
// generic test only checks the boundary.
func TestT779_CollectCheck_EnvOverride(t *testing.T) {
	// very large threshold → almost guaranteed pass
	t.Setenv(DivergenceWarnEnv, "100000")
	check := collectMainDivergenceCheck()
	if check == nil {
		t.Fatal("nil")
	}
	if check.Status == "warn" {
		t.Errorf("warn at threshold 100000 — origin/main is abnormally far behind")
	}
}
