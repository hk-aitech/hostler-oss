package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureResultSectionStub_InsertsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	taskFile := filepath.Join(dir, "T676-test.md")

	content := "# T676 test Task\n\n## Background\n\nbackground details.\n\n## Status change history\n\n- todo → in-progress\n"
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("file create failed: %v", err)
	}

	stubbed, err := EnsureResultSectionStub(taskFile)
	if err != nil {
		t.Fatalf("EnsureResultSectionStub error: %v", err)
	}
	if !stubbed {
		t.Error("expected stubbed=true because the ## Result section is missing")
	}

	raw, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("file read failed: %v", err)
	}
	got := string(raw)

	if !strings.Contains(got, "## Result") {
		t.Error("file is missing the '## Result' text after insertion")
	}

	// the stub must come before ## Status change history
	resultIdx := strings.Index(got, "## Result")
	historyIdx := strings.Index(got, "## Status change history")
	if resultIdx == -1 || historyIdx == -1 {
		t.Fatal("missing ## Result or ## Status change history")
	}
	if resultIdx >= historyIdx {
		t.Errorf("expected ## Result section (%d) to precede ## Status change history (%d)", resultIdx, historyIdx)
	}
}

func TestEnsureResultSectionStub_NoOpWhenExists(t *testing.T) {
	dir := t.TempDir()
	taskFile := filepath.Join(dir, "T676-existing.md")

	content := "# T676 existing-result Task\n\n## Result\n\nalready written result.\n\n## Status change history\n\n- todo → done\n"
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("file create failed: %v", err)
	}

	stubbed, err := EnsureResultSectionStub(taskFile)
	if err != nil {
		t.Fatalf("EnsureResultSectionStub error: %v", err)
	}
	if stubbed {
		t.Error("expected stubbed=false because ## Result already exists")
	}

	raw, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("file read failed: %v", err)
	}
	// content must not change
	if string(raw) != content {
		t.Error("no-op but file content changed")
	}
}

func TestEnsureResultSectionStub_InsertsAtEndWhenNoHistory(t *testing.T) {
	dir := t.TempDir()
	taskFile := filepath.Join(dir, "T676-no-history.md")

	content := "# T676 Task without history\n\n## Background\n\nbackground details.\n"
	if err := os.WriteFile(taskFile, []byte(content), 0o644); err != nil {
		t.Fatalf("file create failed: %v", err)
	}

	stubbed, err := EnsureResultSectionStub(taskFile)
	if err != nil {
		t.Fatalf("EnsureResultSectionStub error: %v", err)
	}
	if !stubbed {
		t.Error("expected stubbed=true because the ## Result section is missing")
	}

	raw, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("file read failed: %v", err)
	}
	got := string(raw)

	if !strings.Contains(got, "## Result") {
		t.Error("file is missing the '## Result' text after insertion")
	}
	// without history, ## Result must be at the end of the file
	resultIdx := strings.Index(got, "## Result")
	if resultIdx == -1 {
		t.Fatal("## Result section is missing")
	}
	// must come after ## Background
	bgIdx := strings.Index(got, "## Background")
	if resultIdx <= bgIdx {
		t.Errorf("expected ## Result section (%d) to come after ## Background (%d)", resultIdx, bgIdx)
	}
}
