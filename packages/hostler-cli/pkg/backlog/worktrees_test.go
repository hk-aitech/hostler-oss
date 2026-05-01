// trac: HAR-CM030
package backlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnumerateWorktreeRoots_NonGit(t *testing.T) {
	// when called outside a git directory the result is empty (graceful)
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	roots := EnumerateWorktreeRoots()
	if len(roots) != 0 {
		t.Errorf("non-git dir should return empty, got %v", roots)
	}
}

func TestFileExistsInAnyWorktree_AbsolutePath(t *testing.T) {
	tmp := t.TempDir()
	abs := filepath.Join(tmp, "task.md")
	if err := os.WriteFile(abs, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !FileExistsInAnyWorktree(abs, nil, "") {
		t.Errorf("absolute path should be detected directly, abs=%q", abs)
	}
}

func TestFileExistsInAnyWorktree_RelativePath_FallbackRoot(t *testing.T) {
	tmp := t.TempDir()
	rel := "works/tasks/T999-test.md"
	full := filepath.Join(tmp, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	// when roots is empty, fallbackRoot is used
	if !FileExistsInAnyWorktree(rel, nil, tmp) {
		t.Errorf("relative path with fallback root should be detected, rel=%q tmp=%q", rel, tmp)
	}
}

func TestFileExistsInAnyWorktree_MultiWorktree(t *testing.T) {
	// 2-worktree fixture — file exists only in worktree-B
	worktreeA := t.TempDir()
	worktreeB := t.TempDir()
	rel := "works/tasks/T999-cross-worktree.md"
	fullB := filepath.Join(worktreeB, rel)
	if err := os.MkdirAll(filepath.Dir(fullB), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullB, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// scan both worktreeA root and worktreeB root
	roots := []string{worktreeA, worktreeB}
	if !FileExistsInAnyWorktree(rel, roots, worktreeA) {
		t.Errorf("file in worktree-B should be detected when scanning multi-roots, rel=%q", rel)
	}
}

func TestFileExistsInAnyWorktree_NotFound(t *testing.T) {
	worktreeA := t.TempDir()
	rel := "works/tasks/T999-missing.md"
	if FileExistsInAnyWorktree(rel, []string{worktreeA}, worktreeA) {
		t.Errorf("missing file should return false")
	}
}

func TestFileExistsInAnyWorktree_EmptyPath(t *testing.T) {
	if FileExistsInAnyWorktree("", nil, "") {
		t.Errorf("empty path should return false")
	}
}
