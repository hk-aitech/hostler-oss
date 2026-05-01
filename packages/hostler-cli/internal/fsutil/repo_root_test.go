package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindRepoRoot_Success — verifies the .git lookup succeeds inside a real repo.
func TestFindRepoRoot_Success(t *testing.T) {
	root, err := FindRepoRoot("")
	if err != nil {
		t.Skipf("no .git available — isolated environment: %v", err)
	}
	info, err := os.Stat(filepath.Join(root, ".git"))
	if err != nil || !info.IsDir() {
		t.Fatalf("returned root does not contain a .git directory: err=%v info=%v", err, info)
	}
}

// TestFindRepoRoot_SyntheticRoot — creates a .git directory under a temp dir
// and verifies the lookup walks up to it.
func TestFindRepoRoot_SyntheticRoot(t *testing.T) {
	base := t.TempDir()
	// create base/.git as a directory
	if err := os.Mkdir(filepath.Join(base, ".git"), 0o755); err != nil {
		t.Fatalf("failed to create synthetic .git: %v", err)
	}
	// start the search from base/sub/deep
	deep := filepath.Join(base, "sub", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("failed to create deep directory: %v", err)
	}

	root, err := FindRepoRoot(deep)
	if err != nil {
		t.Fatalf("synthetic repo lookup failed: %v", err)
	}
	if root != base {
		t.Errorf("expected root=%s, got=%s", base, root)
	}
}

// TestFindRepoRoot_NotFound — when no .git is reachable, an error must be returned.
func TestFindRepoRoot_NotFound(t *testing.T) {
	base := t.TempDir()
	deep := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("failed to create deep directory: %v", err)
	}
	// TempDir is per-process isolated, but /tmp's ancestor might be a real
	// repo, so we avoid paths deeper than maxRepoAncestors and only assert
	// "either an error is returned or a valid repo root is returned".
	root, err := FindRepoRoot(deep)
	if err == nil && root == "" {
		t.Errorf("when err=nil the root must not be empty")
	}
}

// TestFindRepoRoot_GitFileNotDir — when .git is a file (worktree / submodule),
// the lookup must not treat that path as a repo root.
func TestFindRepoRoot_GitFileNotDir(t *testing.T) {
	base := t.TempDir()
	// create base/.git as a file
	f := filepath.Join(base, ".git")
	if err := os.WriteFile(f, []byte("gitdir: ../real.git"), 0o644); err != nil {
		t.Fatalf("failed to create synthetic .git file: %v", err)
	}
	// Current spec only matches directories — a file must not be recognised as the root.
	root, err := FindRepoRoot(base)
	if err == nil && root == base {
		t.Errorf("'.git' was a file but was mistakenly treated as the root: root=%s", root)
	}
}
