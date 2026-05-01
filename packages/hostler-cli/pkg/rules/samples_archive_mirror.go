package rules

import (
	"path/filepath"
	"strings"
	"time"
)

// archive mirror structure auto-validation Rule.
//
// Purpose:
//
//	Enforce the archive skill SSOT — the monorepo `<path>/file` mirrors
//	to `archive/<path>/file` (e.g. docs/02-architecture/adrs/ADR-002.md →
//	archive/docs/02-architecture/adrs/ADR-002.md). External-project
//	snapshots live under archive/backups/<snapshot>/.
//
//	Block creation of non-standard sub-archive directories such as
//	`docs/archive/`.
//
// Subjects (BLOCK):
//
//	staged file paths that contain an `archive/` directory anywhere
//	other than the root.
//	e.g. docs/archive/foo.md, packages/cli/archive/old.go,
//	     works/sprints/archive/sprint-99/SPRINT.md
//
// Allowed:
//
//	- top-level archive/ (root) — the canonical archive location
//	- archive/backups/<snapshot>/ — external project snapshot bucket
//	- any path unrelated to archive/
//
// Scope:
//
//	all staged new/modified files. .md / .go / .yaml / .json / any file.
//	archive is a directory policy, so the file extension does not matter.

const (
	archiveMirrorRuleID      = "precommit.archive.mirror"
	archiveMirrorCategory    = "precommit"
	archiveMirrorEvidenceMax = 10
)

// archiveMirrorRule detects sub-archive directory violations in staged
// file paths.
type archiveMirrorRule struct{}

func (archiveMirrorRule) ID() string       { return archiveMirrorRuleID }
func (archiveMirrorRule) Category() string { return archiveMirrorCategory }
func (archiveMirrorRule) Description() string {
	return "Validate archive mirror structure — block non-root archive/ directories (archive skill SSOT)"
}
func (archiveMirrorRule) DefaultSeverity() Severity { return SeverityBlock }

func (r archiveMirrorRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	root := ctx.ProjectRoot
	if root == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	staged := precommitStagedFiles(ctx)
	if len(staged) == 0 {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	var hits []string
	for _, rel := range staged {
		rel = filepath.ToSlash(rel)
		if isArchiveMirrorViolation(rel) {
			hits = append(hits, rel+": sub-archive directory — use the root archive/ mirror instead")
			if len(hits) >= archiveMirrorEvidenceMax {
				break
			}
		}
	}

	if len(hits) > 0 {
		res.Status = StatusViolated
		res.Message = "archive mirror structure violation — do not create archive/ directories outside the root archive/. Use archive/<original-path>/ mirror. External snapshots go under archive/backups/<snapshot>/."
		res.Evidence = hits
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// isArchiveMirrorViolation decides whether a path is a sub-archive
// violation.
//
// Violation: `archive/` appears mid-path (not at the root).
// Examples:
//   - docs/archive/foo.md          → segments[0]="docs", "archive" found
//   - packages/cli/archive/old.go  → segments[2]="archive"
//   - works/archive/sprint-99/...  → segments[1]="archive"
//
// Allowed:
//   - archive/docs/foo.md          → segments[0]="archive" (root)
//   - archive/backups/snap/foo.md  → ditto
//   - other/path/no-archive/foo.md → no "archive/"
func isArchiveMirrorViolation(rel string) bool {
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return false
	}
	segments := strings.Split(rel, "/")
	if len(segments) < 2 {
		return false
	}
	// segments[0] == "archive" → root archive — OK
	if segments[0] == "archive" {
		return false
	}
	// Avoid a false positive when a skill is itself named "archive"
	// inside a skills/ directory. The skills/<name>/ pattern with
	// <name> == "archive" — the SKILL.md body of the archive skill
	// legitimately discusses archive concerns.
	if strings.Contains(rel, "/skills/archive/") || strings.HasPrefix(rel, "skills/archive/") {
		return false
	}
	// Otherwise, an "archive" segment elsewhere in the path is a violation.
	for i := 1; i < len(segments); i++ {
		if segments[i] == "archive" {
			return true
		}
	}
	return false
}

func init() {
	Register(archiveMirrorRule{})
}
