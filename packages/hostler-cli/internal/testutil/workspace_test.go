package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// TestNewIsolatedWorkspace_Default verifies that only the .hostler
// directory is created with the default options.
func TestNewIsolatedWorkspace_Default(t *testing.T) {
	ws := NewIsolatedWorkspace(t)
	if ws.Root == "" {
		t.Fatal("Root is empty")
	}
	info, err := os.Stat(ws.ConfigDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("ConfigDir not present: %v", err)
	}
	if filepath.Base(ws.ConfigDir) != brand.ProjectDirName {
		t.Errorf("default ConfigDir expected %s, got %s",
			brand.ProjectDirName, filepath.Base(ws.ConfigDir))
	}
	// project-config.yaml is not created by default.
	if _, err := os.Stat(filepath.Join(ws.ConfigDir, "project-config.yaml")); !os.IsNotExist(err) {
		t.Error("project-config.yaml should not exist by default")
	}
}

// TestNewIsolatedWorkspace_WithProjectConfig verifies that the
// configured contents are reflected in the file.
func TestNewIsolatedWorkspace_WithProjectConfig(t *testing.T) {
	content := "project:\n  key: test-key\n"
	ws := NewIsolatedWorkspace(t, WithProjectConfig(content))
	path := filepath.Join(ws.ConfigDir, "project-config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read project-config.yaml: %v", err)
	}
	if string(data) != content {
		t.Errorf("project-config.yaml content mismatch: got %q, want %q", string(data), content)
	}
}

// TestNewIsolatedWorkspace_WithHostlerSubdirs verifies that extra
// sub-directories are created.
func TestNewIsolatedWorkspace_WithHostlerSubdirs(t *testing.T) {
	ws := NewIsolatedWorkspace(t, WithHostlerSubdirs("audit", "mailbox/inbox", "templates/reminders"))
	for _, sub := range []string{"audit", "mailbox/inbox", "templates/reminders"} {
		full := filepath.Join(ws.ConfigDir, sub)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			t.Errorf("subdir %s does not exist: %v", sub, err)
		}
	}
}

// TestNewIsolatedWorkspace_WithConfigDirName verifies that the override
// option changes the hidden config directory name.
func TestNewIsolatedWorkspace_WithConfigDirName(t *testing.T) {
	ws := NewIsolatedWorkspace(t, WithConfigDirName(".alt"))
	if filepath.Base(ws.ConfigDir) != ".alt" {
		t.Errorf("ConfigDir name expected .alt, got %s", filepath.Base(ws.ConfigDir))
	}
}

// TestWorkspace_WriteFile_PathHelpers verifies WriteFile + Path behaviour.
func TestWorkspace_WriteFile_PathHelpers(t *testing.T) {
	ws := NewIsolatedWorkspace(t)
	full := ws.WriteFile("sub/nested/file.txt", "hello")
	if full == "" {
		t.Fatal("WriteFile expected to return an absolute path")
	}
	if ws.Path("sub/nested/file.txt") != full {
		t.Errorf("Path return value mismatch: %s != %s", ws.Path("sub/nested/file.txt"), full)
	}
	data, err := os.ReadFile(full)
	if err != nil || string(data) != "hello" {
		t.Errorf("file contents check failed: %v / %q", err, data)
	}
}
