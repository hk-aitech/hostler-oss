// Package task — git log fallback for strict result_check.
// Background: the strict result_check policy unions
// `git diff (staged + unstaged) + the most recent N commit logs` to build
// the "changed files" list. When a Task is completed post-hoc (for
// example, calling task complete on a Task whose work was already merged
// in a separate commit), commits outside the recent-N window cause the
// files to be misclassified as missing and produce a BLOCKED outcome.
// `applyGitLogFallback` performs a second pass for each file flagged as
// missing in the first pass, scanning the entire git log since the Task's
// creation date. If any commit touched the path, it is removed from the
// missing list.
// Policy:
// The fallback runs only under the strict policy (warn/off keep their
// previous behaviour).
// When createdAt is missing or unparseable, the fallback is skipped
// (safe mode).
// Fallback results are recorded in the audit log (visibility of
// changes).
// Rejected alternatives:
// Increasing N in `getGitChangedFiles` to infinity: degrades
// performance on large repos (thousands of commits × changed files ×
// Tasks cross-product). Per-path lookup is more efficient.
// Narrowing to "since merge base": misses cases when Sprints merge
// across each other. Task created_at is a safer lower bound.
package task

import (
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// applyGitLogFallback filters the missing list using "git log since
// Task.createdAt" results.
// Input:
// createdAt: the Task frontmatter `created` value (YYYY-MM-DD or
// RFC3339).
// missing: the list of files flagged missing by the first pass.
// Output:
// stillMissing: files that remain missing after the fallback.
// recovered: files recovered by the fallback (recorded in audit).
// No-op when createdAt is empty or missing is empty.
func applyGitLogFallback(createdAt string, missing []domain.MissingFile) (stillMissing []domain.MissingFile, recovered []string) {
	if createdAt == "" || len(missing) == 0 {
		return missing, nil
	}
	since := normalizeSince(createdAt)
	if since == "" {
		// Invalid date — skip the fallback. Strict behaviour is preserved.
		return missing, nil
	}
	root := fileutil.GetProjectRoot()
	if root == "" {
		return missing, nil
	}

	stillMissing = make([]domain.MissingFile, 0, len(missing))
	for _, m := range missing {
		if pathInGitLogSince(root, since, m.Path) {
			recovered = append(recovered, m.Path)
			continue
		}
		stillMissing = append(stillMissing, m)
	}
	return stillMissing, recovered
}

// normalizeSince converts a `created:` value (YYYY-MM-DD or an RFC3339
// prefix) into the ISO-8601-ish form accepted by `git log --since`. git
// already parses YYYY-MM-DD, so the leading 10-character date is enough.
// Invalid input (empty string, non-ISO format) returns "" so the caller
// can skip the fallback.
func normalizeSince(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	// Strip quotes left in by the YAML parser.
	trimmed = strings.Trim(trimmed, `"'`)
	// Minimum length for YYYY-MM-DD is 10.
	if len(trimmed) < 10 {
		return ""
	}
	// Validate the YYYY-MM-DD prefix — digits and hyphens only.
	prefix := trimmed[:10]
	if prefix[4] != '-' || prefix[7] != '-' {
		return ""
	}
	for i, ch := range prefix {
		if i == 4 || i == 7 {
			continue
		}
		if ch < '0' || ch > '9' {
			return ""
		}
	}
	return prefix
}

// pathInGitLogSince reports whether any commit since `since` has touched
// `path`.
// `git log --since=<since> --name-only --pretty=format: -- <path>` emits
// a list of changed files when matching commits exist and an empty
// output otherwise; a simple trim suffices to detect that.
// exec failures are treated as "no match" (safe mode) so that strict
// verification is not relaxed in edge cases such as missing git or
// permission errors.
func pathInGitLogSince(root, since, path string) bool {
	if path == "" {
		return false
	}
	out, err := runGit(root,
		"log",
		"--since="+since,
		"--name-only",
		"--pretty=format:",
		"--",
		path,
	)
	if err != nil {
		return false
	}
	trimmed := strings.TrimSpace(string(out))
	return trimmed != ""
}
