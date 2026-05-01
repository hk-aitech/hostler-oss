// T430 (Sprint-36) — SprintStart.DivergenceWarnCommits config-fallback
// verification.
//
// Priority: env var > project-config.yaml > default (25).
package ceremony

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
)

func seedConfigForDivergence(t *testing.T, yaml string) {
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
	t.Setenv(DivergenceWarnEnv, "")
	os.Unsetenv(DivergenceWarnEnv)
}

func TestT430_DivergenceWarn_ConfigFallback(t *testing.T) {
	seedConfigForDivergence(t, `
project:
  key: test
sprint_start:
  divergence_warn_commits: 40
`)
	got := resolveDivergenceWarn()
	if got != 40 {
		t.Errorf("config should provide 40, got %d", got)
	}
}

func TestT430_DivergenceWarn_EnvOverConfig(t *testing.T) {
	seedConfigForDivergence(t, `
project:
  key: test
sprint_start:
  divergence_warn_commits: 40
`)
	t.Setenv(DivergenceWarnEnv, "15")
	got := resolveDivergenceWarn()
	if got != 15 {
		t.Errorf("env should win, expected 15, got %d", got)
	}
}

func TestT430_DivergenceWarn_NoConfigNoEnv(t *testing.T) {
	seedConfigForDivergence(t, "")
	got := resolveDivergenceWarn()
	if got != divergenceWarnDefault {
		t.Errorf("default %d expected when neither config nor env is set, got %d", divergenceWarnDefault, got)
	}
}

func TestT430_DivergenceWarn_InvalidConfigFallback(t *testing.T) {
	// when config value is 0 / negative the default applies.
	seedConfigForDivergence(t, `
project:
  key: test
sprint_start:
  divergence_warn_commits: 0
`)
	got := resolveDivergenceWarn()
	if got != divergenceWarnDefault {
		t.Errorf("0 is invalid — default fallback %d, got %d", divergenceWarnDefault, got)
	}
}
