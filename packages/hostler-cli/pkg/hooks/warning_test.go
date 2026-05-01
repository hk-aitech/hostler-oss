// Package hooks — pre-commit hook warning unit tests.
package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// captureStderr intercepts stderr output during fn execution and returns it.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = buf.ReadFrom(r)
		close(done)
	}()

	fn()
	_ = w.Close()
	<-done
	return buf.String()
}

// makeGitProject creates a git project with or without the pre-commit hook based on hookInstalled.
func makeGitProject(t *testing.T, hookInstalled bool) string {
	t.Helper()
	tmp := t.TempDir()
	hooksDir := filepath.Join(tmp, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	if hookInstalled {
		if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write hook: %v", err)
		}
	}
	return tmp
}

func TestT741_CheckAndWarnOnce_HookInstalled_NoWarning(t *testing.T) {
	resetWarnOnceForTest()
	t.Setenv(HookWarningEnvVar, "")
	root := makeGitProject(t, true)

	out := captureStderr(t, func() {
		CheckAndWarnOnce(root)
	})
	if out != "" {
		t.Errorf("hook installed but warning printed: %q", out)
	}
}

func TestT741_CheckAndWarnOnce_HookMissing_WarningOnce(t *testing.T) {
	resetWarnOnceForTest()
	t.Setenv(HookWarningEnvVar, "")
	root := makeGitProject(t, false)

	out := captureStderr(t, func() {
		CheckAndWarnOnce(root)
		CheckAndWarnOnce(root) // second call must be skipped by sync.Once.
	})

	// Warning identifier must appear exactly once.
	count := bytes.Count([]byte(out), []byte("pre-commit hook not installed"))
	if count != 1 {
		t.Errorf("warning emit count: got=%d want=1 (per-session guarantee failed)\noutput: %q", count, out)
	}
}

func TestT741_CheckAndWarnOnce_EnvOff_Skip(t *testing.T) {
	resetWarnOnceForTest()
	t.Setenv(HookWarningEnvVar, "off")
	root := makeGitProject(t, false)

	out := captureStderr(t, func() {
		CheckAndWarnOnce(root)
	})
	if out != "" {
		t.Errorf("env=off but warning printed: %q", out)
	}
}

func TestT741_CheckAndWarnOnce_NotGitProject_Skip(t *testing.T) {
	resetWarnOnceForTest()
	t.Setenv(HookWarningEnvVar, "")
	tmp := t.TempDir() // ordinary directory without .git

	out := captureStderr(t, func() {
		CheckAndWarnOnce(tmp)
	})
	if out != "" {
		t.Errorf("non-git project but warning printed: %q", out)
	}
}

func TestT741_CheckAndWarnOnce_EmptyProjectRoot_CwdFallback(t *testing.T) {
	resetWarnOnceForTest()
	t.Setenv(HookWarningEnvVar, "")
	root := makeGitProject(t, false)

	// Switch cwd to the git project.
	t.Chdir(root)

	out := captureStderr(t, func() {
		CheckAndWarnOnce("") // empty projectRoot -> cwd fallback
	})
	if !bytes.Contains([]byte(out), []byte("pre-commit hook not installed")) {
		t.Errorf("cwd fallback failed: %q", out)
	}
}
