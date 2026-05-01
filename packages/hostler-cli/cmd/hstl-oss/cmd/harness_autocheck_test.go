package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestT478_DetectGoModuleDir verifies that go.mod detection works for both
// root/ and root/cli/ relative to the project root.
func TestT478_DetectGoModuleDir(t *testing.T) {
	t.Run("root/go.mod", func(t *testing.T) {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test"), 0o644); err != nil {
			t.Fatal(err)
		}
		got := detectGoModuleDir(tmp)
		if got != tmp {
			t.Errorf("failed to detect root/go.mod: got=%q want=%q", got, tmp)
		}
	})
	t.Run("root/cli/go.mod", func(t *testing.T) {
		tmp := t.TempDir()
		cliDir := filepath.Join(tmp, "cli")
		if err := os.MkdirAll(cliDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cliDir, "go.mod"), []byte("module test"), 0o644); err != nil {
			t.Fatal(err)
		}
		got := detectGoModuleDir(tmp)
		if got != cliDir {
			t.Errorf("failed to detect root/cli/go.mod: got=%q want=%q", got, cliDir)
		}
	})
	t.Run("no go.mod", func(t *testing.T) {
		tmp := t.TempDir()
		got := detectGoModuleDir(tmp)
		if got != "" {
			t.Errorf("expected empty string when go.mod is absent: got=%q", got)
		}
	})
}

// TestT478_TrimOutput verifies that the output trim helper handles short and
// long strings correctly.
func TestT478_TrimOutput(t *testing.T) {
	if got := trimOutput("short", 100); got != "short" {
		t.Errorf("short string: got=%q", got)
	}
	if got := trimOutput("  spaces  ", 100); got != "spaces" {
		t.Errorf("whitespace trim: got=%q", got)
	}
	long := "0123456789"
	if got := trimOutput(long, 5); got != "01234…(truncated)" {
		t.Errorf("long string truncation: got=%q", got)
	}
}

// TestT586_AutoCheckDeterministicWhitelist verifies the whitelist boundary.
// context_acknowledged must be on the deterministic whitelist; human-judgment
// items must be excluded (preserves Harness Gate safety, KB A01).
func TestT586_AutoCheckDeterministicWhitelist(t *testing.T) {
	if _, ok := autoCheckDeterministicItems["context_acknowledged"]; !ok {
		t.Error("context_acknowledged missing from whitelist")
	}
	humanJudgment := []string{
		"criteria_checked",
		"code_review",
		"reproduction",
		"root_cause",
		"deploy_verified",
	}
	for _, id := range humanJudgment {
		if _, ok := autoCheckDeterministicItems[id]; ok {
			t.Errorf("human-judgment item incorrectly included in whitelist: %s", id)
		}
		if _, ok := autoCheckGoItems[id]; ok {
			t.Errorf("human-judgment item incorrectly included in Go toolchain whitelist: %s", id)
		}
	}
}
