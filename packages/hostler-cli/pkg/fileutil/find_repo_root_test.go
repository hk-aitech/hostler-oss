// T354 (Sprint-29) — FindRepoRoot unit tests. Validates monorepo +
// legacy cli/ layouts and the not-found case.
package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// TestFindRepoRoot_MonorepoLayout verifies that the monorepo layout
// (packages/hostler-cli/go.mod) returns the correct root.
func TestFindRepoRoot_MonorepoLayout(t *testing.T) {
	tmp := t.TempDir()
	cliDir := filepath.Join(tmp, "packages", "hostler-cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// start = tmp — packages/hostler-cli is found under tmp.
	root, err := fileutil.FindRepoRoot(tmp)
	if err != nil {
		t.Fatalf("FindRepoRoot(tmp) failed: %v", err)
	}
	if root != tmp {
		t.Errorf("tmp layout root mismatch: got=%s, want=%s", root, tmp)
	}

	// start = cliDir itself — recognized as basename=hostler-cli and
	// parent=packages.
	root2, err := fileutil.FindRepoRoot(cliDir)
	if err != nil {
		t.Fatalf("FindRepoRoot(cliDir) failed: %v", err)
	}
	if root2 != tmp {
		t.Errorf("cliDir layout root mismatch: got=%s, want=%s", root2, tmp)
	}

	// start in a deeper subdir (cmd/gen-schema) — walks up.
	deep := filepath.Join(cliDir, "cmd", "gen-schema")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	root3, err := fileutil.FindRepoRoot(deep)
	if err != nil {
		t.Fatalf("FindRepoRoot(deep) failed: %v", err)
	}
	if root3 != tmp {
		t.Errorf("deep layout root mismatch: got=%s, want=%s", root3, tmp)
	}
}

// TestFindRepoRoot_LegacyCliLayout verifies the legacy layout
// (cli/go.mod).
func TestFindRepoRoot_LegacyCliLayout(t *testing.T) {
	tmp := t.TempDir()
	cliDir := filepath.Join(tmp, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, err := fileutil.FindRepoRoot(tmp)
	if err != nil {
		t.Fatalf("FindRepoRoot(tmp) failed: %v", err)
	}
	if root != tmp {
		t.Errorf("tmp layout root mismatch: got=%s, want=%s", root, tmp)
	}

	// Called from cliDir itself.
	root2, err := fileutil.FindRepoRoot(cliDir)
	if err != nil {
		t.Fatalf("FindRepoRoot(cliDir) failed: %v", err)
	}
	if root2 != tmp {
		t.Errorf("cliDir (legacy) root mismatch: got=%s, want=%s", root2, tmp)
	}
}

// TestFindRepoRoot_NotFound verifies an error is returned when no
// candidate path exists.
func TestFindRepoRoot_NotFound(t *testing.T) {
	tmp := t.TempDir()
	// No candidate path created.
	_, err := fileutil.FindRepoRootMax(tmp, 3)
	if err == nil {
		t.Error("expected error when no candidate path, got nil")
	}
}

// TestFindRepoRoot_EmptyStart — guards against an empty start argument.
func TestFindRepoRoot_EmptyStart(t *testing.T) {
	_, err := fileutil.FindRepoRoot("")
	if err == nil {
		t.Error("expected error for empty start, got nil")
	}
}
