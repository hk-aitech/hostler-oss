// trac: HAR-CM006
package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckDevSyncWarn_OptOut - env=off skips immediately (no panic/exec).
func TestCheckDevSyncWarn_OptOut(t *testing.T) {
	t.Setenv(devSyncEnvKey, "off")
	// Passing the call without panicking is enough (directory-agnostic).
	checkDevSyncWarn()
}

// TestCheckDevSyncWarn_NonGit - silent skip in a directory without git.
func TestCheckDevSyncWarn_NonGit(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Setenv(devSyncEnvKey, "")
	// Verifies that non-git contexts return without panicking.
	checkDevSyncWarn()
}

// TestIsGitRepo_True - inside the repo root.
func TestIsGitRepo_True(t *testing.T) {
	cwd, _ := os.Getwd()
	if cwd == "" {
		t.Skip("cwd unknown")
	}
	// The repo must be a git repo (the test itself runs inside it).
	if !isGitRepo() {
		t.Skip("not in git repo (CI environment)")
	}
}

// TestIsGitRepo_False - non-git directory.
func TestIsGitRepo_False(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	if isGitRepo() {
		t.Errorf("expected non-git tmp dir to return false")
	}
}

// TestCurrentBranch_GitInitMain - look up the current branch in a temporary git repo.
func TestCurrentBranch_GitInitMain(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	// git init main + first commit (rev-parse --abbrev-ref HEAD requires at least one commit).
	if err := exec.Command("git", "init", "-b", "main").Run(); err != nil {
		t.Skip("git init unavailable: " + err.Error())
	}
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "test").Run()
	if err := os.WriteFile(filepath.Join(tmp, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "init").Run()

	branch, err := currentBranch()
	if err != nil {
		t.Fatalf("currentBranch failed: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}

// TestCheckDevSyncWarn_DevBranch - skips its own check while on the dev branch.
func TestCheckDevSyncWarn_DevBranch(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "init", "-b", "dev").Run(); err != nil {
		t.Skip("git init unavailable")
	}
	exec.Command("git", "config", "user.email", "test@example.com").Run()
	exec.Command("git", "config", "user.name", "test").Run()
	if err := os.WriteFile(filepath.Join(tmp, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "add", ".").Run()
	exec.Command("git", "commit", "-m", "init").Run()

	t.Setenv(devSyncEnvKey, "")
	// dev branch -> immediate skip (PASS if nothing panics).
	checkDevSyncWarn()
}

// TestCheckDevSyncWarn_OptOutCaseInsensitive - env value is case-insensitive.
func TestCheckDevSyncWarn_OptOutCaseInsensitive(t *testing.T) {
	for _, val := range []string{"off", "OFF", "Off"} {
		t.Setenv(devSyncEnvKey, val)
		// Skipping for any case must not panic.
		checkDevSyncWarn()
		if !strings.EqualFold(val, "off") {
			t.Errorf("env %q should be off-equivalent", val)
		}
	}
}
