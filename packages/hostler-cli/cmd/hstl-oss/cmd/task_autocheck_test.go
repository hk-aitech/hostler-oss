package cmd

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// CLI integration tests for --auto-check-criteria.
// Skipped when the hstl-oss binary is not installed.

func hstlOssBinary(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	// cmd/hstl-oss/cmd/task_autocheck_test.go → bin/hstl-oss (3 levels up).
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(root, "bin", "hstl-oss")
}

func TestAutoCheckCriteria_FlagRegistered(t *testing.T) {
	bin := hstlOssBinary(t)
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("hstl-oss binary not found (%s) — rerun after `make install`", bin)
	}

	cmd := exec.Command(bin, "task", "complete", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run task complete --help: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--auto-check-criteria") {
		t.Errorf("--auto-check-criteria flag missing from help:\n%s", out)
	}
}
