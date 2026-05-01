// frontmatter_sanity.go — generalised frontmatter ID sanity check.
//
// Background: extracted from the assignSanityCheck pattern so the same
// logic can be reused on sprint create / task create / sprint complete and
// other write paths. The original detection logic
// (archive fast-fail + frontmatter id extraction + ID comparison) lived in
// pkg/task/assign_sanity.go; moved into fileutil for shared use without
// creating an import cycle.
package fileutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FrontmatterIDSanityResult is the check result.
//
// Reason empty == pass. Non-empty == violation reason.
type FrontmatterIDSanityResult struct {
	Reason string // violation reason (empty string = pass)
	FmID   string // id extracted from frontmatter (debugging)
}

// CheckFrontmatterIDSanity verifies that the frontmatter id of file
// matches expectedID. The archive path fails fast.
//
// Args:
//   - expectedID: ID the caller expects (taskID or sprintID).
//   - resolvedPath: absolute or repo-relative file path.
//   - entityType: "task" or "sprint" (used in messages).
//
// Returns:
//   - Reason empty: pass.
//   - "archive_path_block:..." = archive path blocked.
//   - "frontmatter_id_mismatch:..." = ID mismatch.
//   - empty + empty FmID = frontmatter absent (normal, e.g. freshly created file).
func CheckFrontmatterIDSanity(expectedID, resolvedPath, entityType string) FrontmatterIDSanityResult {
	rel := filepath.ToSlash(resolvedPath)
	if strings.Contains(rel, "/archive/") || strings.HasPrefix(rel, "archive/") {
		return FrontmatterIDSanityResult{
			Reason: fmt.Sprintf(
				"archive_path_block:%s — archive residual paths are not valid mv targets. Recovery: run backlog sync --dry-run to check consistency",
				resolvedPath,
			),
		}
	}

	fmID, err := ExtractFrontmatterID(resolvedPath)
	if err != nil {
		// Read failure — let the caller handle it via subsequent os.Rename etc. Treat as pass.
		return FrontmatterIDSanityResult{}
	}
	if fmID == "" {
		// frontmatter absent — normal (e.g. just created). Pass.
		return FrontmatterIDSanityResult{}
	}
	if fmID != expectedID {
		return FrontmatterIDSanityResult{
			Reason: fmt.Sprintf(
				"frontmatter_id_mismatch:fm_id=%s,%s_id=%s — possible write-path consistency defect. Recovery: update with the correct file_path or run backlog sync --dry-run",
				fmID, entityType, expectedID,
			),
			FmID: fmID,
		}
	}
	return FrontmatterIDSanityResult{FmID: fmID}
}

// ExtractFrontmatterID extracts the id field from the yaml frontmatter of a markdown file.
//
// Parses the `id: ...` line between `---` markers. Returns the first match.
// Returns an empty string + nil error when frontmatter or id is missing.
func ExtractFrontmatterID(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if lineNum == 1 {
			if strings.TrimSpace(line) != "---" {
				return "", nil
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			break
		}
		if !inFrontmatter {
			break
		}
		if rest, ok := strings.CutPrefix(line, "id:"); ok {
			val := strings.TrimSpace(rest)
			val = strings.Trim(val, `"'`)
			return val, nil
		}
	}
	return "", nil
}
