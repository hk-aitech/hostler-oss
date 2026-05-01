// Package fsutil — shared filesystem utilities.
//
// Keeps filesystem helpers (such as repo-root lookup) that recur across
// tests and production code in a single source. Prevents regressions
// caused by duplicate implementations or hard-coded relative paths.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// maxRepoAncestors caps the number of parent directories traversed during
// FindRepoRoot lookups. Typical monorepos rarely exceed 15 levels.
const maxRepoAncestors = 15

// FindRepoRoot walks upward from start and returns the first directory that
// contains a .git directory.
//
// Semantics: ".git exists as a directory" — .git files used by submodules and
// worktrees are intentionally out of scope. Most Hostler tests run against a
// single repo with a .git directory, which is sufficient.
//
// Returns err != nil when no repo is found — the caller chooses between
// t.Skip and graceful degradation.
func FindRepoRoot(start string) (string, error) {
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get cwd: %w", err)
		}
		start = cwd
	}
	dir := start
	for range maxRepoAncestors {
		info, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .git directory found (start=%s, max=%d)", start, maxRepoAncestors)
}
