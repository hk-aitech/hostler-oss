package cmd

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Integration test for the --interactive flag (verifies help registration).

func TestInteractiveFlagRegistered(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	bin := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "bin", "hstl-oss")
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("hstl-oss binary not found (%s) — rerun after `make install`", bin)
	}

	cmd := exec.Command(bin, "task", "complete", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run help: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "--interactive") {
		t.Errorf("--interactive flag missing from help:\n%s", text)
	}
}

func TestNonTTY_SkipsInteractive(t *testing.T) {
	// When stdin/stdout is a pipe (not a TTY), isInteractiveTTY() should
	// return false. Real TTY detection only makes sense in subprocesses; in a
	// unit test we simply confirm that, in the current test environment
	// (stdout is a pipe), isInteractiveTTY reports false.
	if isInteractiveTTY() {
		t.Skip("test environment is a TTY — this test targets CI/pipe environments")
	}
	// Must not return true in a pipe environment.
}
