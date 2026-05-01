package rules

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
)

// branch-sync config → env injection logic verification.
//
// Priority: env > project-config.yaml > defaults.
// When env is already set, no config value is injected.

// setupT430Config writes a YAML file under a temporary root and clears all
// related env vars.
func setupT430Config(t *testing.T, yaml string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)
	if yaml != "" {
		dir := filepath.Join(tmp, brand.ProjectDirName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "project-config.yaml"), []byte(yaml), 0o644); err != nil {
			t.Fatalf("write yaml: %v", err)
		}
	}
	config.ResetConfigCache()
	// Clear related env vars.
	for _, k := range []string{
		"HSTL_BRANCH_SYNC_THRESHOLD",
		"HSTL_BRANCH_SYNC_WARN_RATIO",
		"HSTL_BRANCH_SYNC",
		"HSTL_BRANCH_SYNC_STRICT",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

// envContains reports whether a "KEY=VAL" slice contains the given key=val.
func envContains(env []string, kv string) bool {
	return slices.Contains(env, kv)
}

// envHasKey reports whether any element starts with "KEY=".
func envHasKey(env []string, key string) bool {
	prefix := key + "="
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func TestT430_BranchSyncEnv_ConfigOnly(t *testing.T) {
	setupT430Config(t, `
project:
  key: test
precommit:
  branch_sync:
    threshold: 60
    warn_ratio: 70
    enabled: true
    strict_mode: true
`)
	env := branchSyncEnv()
	if !envContains(env, "HSTL_BRANCH_SYNC_THRESHOLD=60") {
		t.Errorf("threshold injection missing: %v", env)
	}
	if !envContains(env, "HSTL_BRANCH_SYNC_WARN_RATIO=70") {
		t.Errorf("warn_ratio injection missing: %v", env)
	}
	if !envContains(env, "HSTL_BRANCH_SYNC_STRICT=1") {
		t.Errorf("strict_mode=true → HSTL_BRANCH_SYNC_STRICT=1 missing: %v", env)
	}
	// Enabled=true should not be injected (matches default).
	if envHasKey(env, "HSTL_BRANCH_SYNC") {
		t.Errorf("Enabled=true must not inject HSTL_BRANCH_SYNC: %v", env)
	}
}

func TestT430_BranchSyncEnv_EnvOverrideConfig(t *testing.T) {
	setupT430Config(t, `
project:
  key: test
precommit:
  branch_sync:
    threshold: 60
    warn_ratio: 70
`)
	// When env is already set, config should not override it.
	t.Setenv("HSTL_BRANCH_SYNC_THRESHOLD", "100")
	env := branchSyncEnv()
	if envHasKey(env, "HSTL_BRANCH_SYNC_THRESHOLD") {
		t.Errorf("env already set must not be overwritten by config: %v", env)
	}
	// WarnRatio is not env-set, so config should be injected.
	if !envContains(env, "HSTL_BRANCH_SYNC_WARN_RATIO=70") {
		t.Errorf("WarnRatio with no env should come from config: %v", env)
	}
}

func TestT430_BranchSyncEnv_NoConfigNoEnv(t *testing.T) {
	setupT430Config(t, "") // no config file
	env := branchSyncEnv()
	if len(env) > 0 {
		t.Errorf("no config and no env should yield an empty slice (defaults handled in shell): %v", env)
	}
}

func TestT430_BranchSyncEnv_Disabled(t *testing.T) {
	setupT430Config(t, `
project:
  key: test
precommit:
  branch_sync:
    enabled: false
`)
	env := branchSyncEnv()
	if !envContains(env, "HSTL_BRANCH_SYNC=off") {
		t.Errorf("Enabled=false → HSTL_BRANCH_SYNC=off injection missing: %v", env)
	}
}

func TestT430_BuildPrecommitEnv_UnknownRule(t *testing.T) {
	setupT430Config(t, `
project:
  key: test
precommit:
  branch_sync:
    threshold: 60
`)
	// Other rule IDs return an empty slice (only branch.sync is supported today).
	env := buildPrecommitEnv("precommit.unknown.rule")
	if len(env) > 0 {
		t.Errorf("unsupported rule ID must return nil/empty slice: %v", env)
	}
}
