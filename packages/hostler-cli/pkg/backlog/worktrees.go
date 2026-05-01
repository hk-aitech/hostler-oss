// Package backlog — multi-worktree-aware helpers.
// Background: `hstl backlog sync` previously made the db_orphan_task
// verdict by scanning only the current working tree (cwd). With a main
// worktree plus per-sprint work worktrees, new tasks created in another
// worktree were incorrectly deleted as db_orphan during sync.
// Fix: enumerate every worktree root via `git worktree list
// -porcelain`, combine each root with the DB file_path, and stat to
// verify — if at least one location has the file, it is not an orphan.
package backlog

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnumerateWorktreeRoots returns the absolute paths of every working
// tree from `git worktree list --porcelain`.
// Returns an empty slice + nil error when git is missing, the directory
// is not a git repo, or any other error occurs (graceful skip).
// Callers may fall back to a single-root check when the result is empty.
// Porcelain format:
//	worktree /path/to/main
//	HEAD <sha>
//	branch refs/heads/dev
//	(blank line)
//	worktree /path/to/sprint-NN
//	HEAD <sha>
//	branch refs/heads/sprint-NN
// Only the paths on lines beginning with "worktree " are extracted.
func EnumerateWorktreeRoots() []string {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil
	}

	roots := make([]string, 0, 4)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if rest, ok := strings.CutPrefix(line, "worktree "); ok {
			path := strings.TrimSpace(rest)
			if path != "" {
				roots = append(roots, path)
			}
		}
	}
	return roots
}

// FileExistsInAnyWorktree reports whether file_path (a DB-relative
// path) exists as an actual file under any of the enumerated worktree
// roots.
// When roots is empty, falls back to a single-root check (the current
// cwd or the caller's root). If fallbackRoot is empty, cwd is used.
// Absolute file_path values are stat'd directly without root joining
// (worktree-agnostic).
func FileExistsInAnyWorktree(filePath string, roots []string, fallbackRoot string) bool {
	if filePath == "" {
		return false
	}

	// For absolute paths, stat directly without root joining.
	if filepath.IsAbs(filePath) {
		_, err := os.Stat(filePath)
		return err == nil
	}

	// Candidate root list — enumeration result + fallback.
	candidates := roots
	if len(candidates) == 0 {
		if fallbackRoot != "" {
			candidates = []string{fallbackRoot}
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return false
			}
			candidates = []string{cwd}
		}
	}

	for _, root := range candidates {
		full := filepath.Join(root, filePath)
		if _, err := os.Stat(full); err == nil {
			return true
		}
	}
	return false
}
