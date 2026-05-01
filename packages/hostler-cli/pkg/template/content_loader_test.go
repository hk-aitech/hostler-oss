package template

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeTemplate writes a template file at the given path. Test-only helper.
func writeTemplate(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

// setupTempProjectRoot points the project root at a fresh temp directory.
// fileutil.GetProjectRoot honours HSTL_PROJECT_ROOT (envalias handles the legacy alias).
func setupTempProjectRoot(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HSTL_PROJECT_ROOT", tmp)
	return tmp
}

func TestLoadReminderContent_NormalFile(t *testing.T) {
	root := setupTempProjectRoot(t)

	const content = `---
event: task.start
description: test reminder
severity: info
---

- First reminder
- Second reminder
- Third reminder
`
	path := filepath.Join(root, templatesRootDirName, reminderSubDir, "task.start.md")
	writeTemplate(t, path, content)

	items, meta, err := LoadReminderContent("task.start")
	if err != nil {
		t.Fatalf("LoadReminderContent: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("item count=%d (want 3)", len(items))
	}
	if items[0] != "First reminder" {
		t.Errorf("first item=%q", items[0])
	}
	if meta == nil || meta.Event != "task.start" {
		t.Errorf("meta.Event=%q (want task.start)", meta.Event)
	}
	if meta.Severity != "info" {
		t.Errorf("meta.Severity=%q", meta.Severity)
	}
}

func TestLoadReminderContent_FileMissing(t *testing.T) {
	setupTempProjectRoot(t)
	_, _, err := LoadReminderContent("task.start")
	if err == nil {
		t.Fatal("expected error when file is missing")
	}
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestLoadReminderContent_NoFrontmatter(t *testing.T) {
	root := setupTempProjectRoot(t)
	// Plain markdown without frontmatter must still extract the bullets.
	const content = `- Item 1
- Item 2
`
	path := filepath.Join(root, templatesRootDirName, reminderSubDir, "task.complete.md")
	writeTemplate(t, path, content)

	items, meta, err := LoadReminderContent("task.complete")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("item count=%d", len(items))
	}
	if meta == nil {
		t.Error("meta is nil")
	}
}

func TestLoadReminderContent_EmptyEventName(t *testing.T) {
	_, _, err := LoadReminderContent("")
	if err == nil {
		t.Fatal("expected error on empty event name")
	}
}

func TestLoadCeremonyContent_StartComplete(t *testing.T) {
	root := setupTempProjectRoot(t)

	startContent := `---
section: start
---

- Sprint kickoff step 1
- Sprint kickoff step 2
`
	completeContent := `---
section: complete
---

- Phase 1
- Phase 2
- Phase 3
`
	writeTemplate(t, filepath.Join(root, templatesRootDirName, ceremonySubDir, "start.md"), startContent)
	writeTemplate(t, filepath.Join(root, templatesRootDirName, ceremonySubDir, "complete.md"), completeContent)

	startItems, _, err := LoadCeremonyContent("start")
	if err != nil {
		t.Fatalf("load start: %v", err)
	}
	if len(startItems) != 2 {
		t.Errorf("start item count=%d", len(startItems))
	}

	completeItems, _, err := LoadCeremonyContent("complete")
	if err != nil {
		t.Fatalf("load complete: %v", err)
	}
	if len(completeItems) != 3 {
		t.Errorf("complete item count=%d", len(completeItems))
	}
}

func TestExtractBulletItems(t *testing.T) {
	body := []byte(`# heading is ignored

- First item
  - Indented item (also captured)
- Second

- Third

ordinary prose is ignored
`)
	items := extractBulletItems(body)
	if len(items) != 4 {
		t.Errorf("item count=%d, items=%v", len(items), items)
	}
}

func TestSplitFrontmatter_NoClosingDelim(t *testing.T) {
	data := []byte(`---
event: task.start
body begins
- Item 1
`)
	meta, body, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if meta == nil {
		t.Error("meta is nil")
	}
	// Without a closing delim the entire payload is treated as body.
	if len(body) == 0 {
		t.Error("body is empty")
	}
}
