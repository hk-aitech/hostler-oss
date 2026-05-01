package registry

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// Registry path cascade verification.
// Priority: HSTL_REGISTRY_PATH env → yaml (registryConfigReader) → default.

func TestRuntimePath_EnvOverride(t *testing.T) {
	t.Setenv("HSTL_REGISTRY_PATH", "/custom/env/registry.json")
	t.Setenv("HSTL_PROJECT", "test-project")

	got, err := GetRuntimeRegistryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/custom/env/registry.json" {
		t.Errorf("env override priority broken: got %q", got)
	}
}

func TestRuntimePath_YamlOverride(t *testing.T) {
	t.Setenv("HSTL_REGISTRY_PATH", "")
	t.Setenv("HSTL_PROJECT", "test-project")

	// YAML reader stub
	old := registryConfigReader
	t.Cleanup(func() { registryConfigReader = old })
	registryConfigReader = func(field string) string {
		if field == "runtime" {
			return "/yaml/override/registry.json"
		}
		return ""
	}

	got, err := GetRuntimeRegistryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/yaml/override/registry.json" {
		t.Errorf("yaml override priority broken: got %q", got)
	}
}

func TestRuntimePath_Default_UsesBrand(t *testing.T) {
	t.Setenv("HSTL_REGISTRY_PATH", "")
	t.Setenv("HSTL_PROJECT", "test-project")

	old := registryConfigReader
	t.Cleanup(func() { registryConfigReader = old })
	registryConfigReader = func(field string) string { return "" }

	got, err := GetRuntimeRegistryPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify the default uses brand.ProjectDirName (no hard-coded
	// directory name).
	if !strings.Contains(got, brand.ProjectDirName) {
		t.Errorf("default does not contain brand.ProjectDirName: got %q, want contains %q",
			got, brand.ProjectDirName)
	}
	if !strings.Contains(got, "test-project") {
		t.Errorf("default does not contain the project key: got %q", got)
	}
	if !strings.HasSuffix(got, "registry.json") {
		t.Errorf("default does not end with registry.json: got %q", got)
	}
}

func TestDocsPath_EnvOverride(t *testing.T) {
	t.Setenv("HSTL_REGISTRY_DOCS_PATH", "/custom/docs/registry.json")
	got := GetDocsRegistryPath("/any/root")
	if got != "/custom/docs/registry.json" {
		t.Errorf("docs env override failed: got %q", got)
	}
}

func TestDocsPath_Default_UsesBrand(t *testing.T) {
	t.Setenv("HSTL_REGISTRY_DOCS_PATH", "")

	old := registryConfigReader
	t.Cleanup(func() { registryConfigReader = old })
	registryConfigReader = func(field string) string { return "" }

	root := "/project/root"
	got := GetDocsRegistryPath(root)

	expected := filepath.Join(root, brand.ProjectDirName, "data", "registry.json")
	if got != expected {
		t.Errorf("docs default mismatch: got %q, want %q", got, expected)
	}
}
