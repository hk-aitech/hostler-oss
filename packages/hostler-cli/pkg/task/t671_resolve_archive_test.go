package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestT671_FindTaskFiles_ArchiveExcluded — archive/ directory is skipped
// during walk. When the same Task ID exists in both archive and active,
// only the active one must be returned.
func TestT671_FindTaskFiles_ArchiveExcluded(t *testing.T) {
	root := t.TempDir()

	mustWrite := func(rel string) string {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("---\nid: T671\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return full
	}

	activePath := mustWrite("works/tasks/T671-active.md")
	mustWrite("archive/backups/legacy/tasks/T671-snapshot.md")
	mustWrite("archive/backups/another-snapshot/tasks/T671-other.md")

	got := findTaskFiles(root, "T671")
	if len(got) != 1 {
		t.Fatalf("expected 1 file (archive excluded), got %d: %v", len(got), got)
	}
	if got[0] != activePath {
		t.Errorf("expected %s, got %s", activePath, got[0])
	}
}

// TestT671_FindTaskFiles_NoActiveJustArchive — when the file is only in
// archive (none in active) the result is empty (archive matches are
// rejected per the T671 read-only policy).
func TestT671_FindTaskFiles_NoActiveJustArchive(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "archive/backups/snapshot/T671-only.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("---\nid: T671\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := findTaskFiles(root, "T671")
	if len(got) != 0 {
		t.Fatalf("expected 0 files (archive excluded), got %d: %v", len(got), got)
	}
}

// TestT671_FindTaskFiles_ActiveAndCompleted — active/backlog/completed
// under works/ all match normally. Only archive is excluded.
func TestT671_FindTaskFiles_ActiveAndCompleted(t *testing.T) {
	root := t.TempDir()

	paths := []string{
		"works/sprints/active/sprint-N/tasks/T671-active.md",
		"works/sprints/completed/sprint-M/tasks/T671-completed.md",
		"works/tasks/T671-loose.md",
	}
	for _, rel := range paths {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("---\nid: T671\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// archive-side duplicate — must be excluded
	archiveFull := filepath.Join(root, "archive/backups/legacy/T671-snapshot.md")
	if err := os.MkdirAll(filepath.Dir(archiveFull), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archiveFull, []byte("---\nid: T671\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := findTaskFiles(root, "T671")
	if len(got) != 3 {
		t.Fatalf("expected 3 (under works/ only), got %d: %v", len(got), got)
	}
	for _, p := range got {
		if strings.Contains(p, "archive/") {
			t.Errorf("archive path included: %s", p)
		}
	}
}
