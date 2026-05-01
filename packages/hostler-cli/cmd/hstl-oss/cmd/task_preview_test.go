package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// printChangedFilesPreview previews the current git-changed files on stderr.
// Reuses pkg/task.GetGitChangedFiles, so the unit test runs against an
// actual git repo.
//
// When DefaultFormat is FormatJSON, output.Stderr() returns io.Discard in
// JSON mode. This test verifies the preview text on stderr, so we force
// console mode - in production, suppressing the preview in JSON mode is
// the intended behavior.

// forceConsoleForPreview forces console mode in a JSON-default environment
// so the preview test can capture stderr. Restores via t.Cleanup.
func forceConsoleForPreview(t *testing.T) {
	t.Helper()
	prev := output.CurrentFormat()
	output.SetCurrentFormat(output.FormatConsole)
	t.Cleanup(func() { output.SetCurrentFormat(prev) })
}

func runGitInDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func withTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitInDir(t, dir, "init", "-q")
	runGitInDir(t, dir, "config", "user.email", "test@example.com")
	runGitInDir(t, dir, "config", "user.name", "test")
	runGitInDir(t, dir, "commit", "--allow-empty", "-q", "-m", "init")
	// Register t.Cleanup so the original chdir is restored.
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Setenv("HSTL_PROJECT_ROOT", dir)
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	return dir
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = origStderr
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestT488_PrintChangedFilesPreview_Empty(t *testing.T) {
	forceConsoleForPreview(t)
	withTempGitRepo(t)
	// Sanity: task.GetGitChangedFiles must also return empty.
	if files := task.GetGitChangedFiles(); len(files) != 0 {
		t.Fatalf("initial repo had %d changed files (expected 0)", len(files))
	}
	out := captureStderr(t, func() { printChangedFilesPreview("T999") })
	if !strings.Contains(out, "(0)") {
		t.Errorf("missing 0 count notice:\n%s", out)
	}
	if !strings.Contains(out, "T999") {
		t.Errorf("missing Task ID:\n%s", out)
	}
}

func TestT488_PrintChangedFilesPreview_UnstagedOnly(t *testing.T) {
	forceConsoleForPreview(t)
	dir := withTempGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInDir(t, dir, "add", "tracked.txt")
	runGitInDir(t, dir, "commit", "-q", "-m", "add")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := captureStderr(t, func() { printChangedFilesPreview("T999") })
	if !strings.Contains(out, "tracked.txt") {
		t.Errorf("missing unstaged file:\n%s", out)
	}
}

func TestT488_PrintChangedFilesPreview_StagedOnly(t *testing.T) {
	forceConsoleForPreview(t)
	dir := withTempGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInDir(t, dir, "add", "staged.txt")
	out := captureStderr(t, func() { printChangedFilesPreview("T999") })
	if !strings.Contains(out, "staged.txt") {
		t.Errorf("missing staged file:\n%s", out)
	}
}

func TestT488_PrintChangedFilesPreview_MixedAndRecentCommit(t *testing.T) {
	forceConsoleForPreview(t)
	dir := withTempGitRepo(t)
	// Recent commit file.
	if err := os.WriteFile(filepath.Join(dir, "recent.txt"), []byte("r"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInDir(t, dir, "add", "recent.txt")
	runGitInDir(t, dir, "commit", "-q", "-m", "recent")
	// staged
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInDir(t, dir, "add", "staged.txt")
	// unstaged
	if err := os.WriteFile(filepath.Join(dir, "unstaged.txt"), []byte("u"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitInDir(t, dir, "add", "unstaged.txt")
	runGitInDir(t, dir, "commit", "-q", "-m", "u")
	if err := os.WriteFile(filepath.Join(dir, "unstaged.txt"), []byte("u2"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := captureStderr(t, func() { printChangedFilesPreview("T999") })
	for _, want := range []string{"recent.txt", "staged.txt", "unstaged.txt"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}
}
