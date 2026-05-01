package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// Unit tests for version bump.

func TestBumpSemver(t *testing.T) {
	cases := []struct {
		old, bumpType, want string
	}{
		{"3.3.0", "patch", "3.3.1"},
		{"3.3.0", "minor", "3.4.0"},
		{"3.3.0", "major", "4.0.0"},
		{"0.0.9", "patch", "0.0.10"},
		{"1.2.3", "minor", "1.3.0"},
	}
	for _, c := range cases {
		got, err := bumpSemver(c.old, c.bumpType)
		if err != nil {
			t.Fatalf("bumpSemver(%q,%q) err=%v", c.old, c.bumpType, err)
		}
		if got != c.want {
			t.Errorf("bumpSemver(%q,%q)=%q want %q", c.old, c.bumpType, got, c.want)
		}
	}
}

func TestBumpSemver_Invalid(t *testing.T) {
	if _, err := bumpSemver("not.a.version", "patch"); err == nil {
		t.Error("expected error on non-semver")
	}
	if _, err := bumpSemver("1.2", "patch"); err == nil {
		t.Error("expected error on incomplete semver")
	}
}

func TestWritePluginJSONVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.json")
	original := fmt.Sprintf(`{
  "name": "%s",
  "version": "3.3.0",
  "description": "..."
}
`, brand.ProductName)
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writePluginJSONVersion(path, "3.4.0"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"version": "3.4.0"`) {
		t.Errorf("version not updated: %s", data)
	}
	// formatting preserved
	if !strings.Contains(string(data), fmt.Sprintf(`"name": "%s"`, brand.ProductName)) {
		t.Errorf("formatting lost: %s", data)
	}
}

func TestWriteMarketplaceJSONVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "marketplace.json")
	obj := map[string]any{
		"plugins": []any{
			map[string]any{"name": brand.ProductName, "version": "3.3.0"},
			map[string]any{"name": "other", "version": "1.0.0"},
		},
	}
	data, _ := json.MarshalIndent(obj, "", "  ")
	os.WriteFile(path, data, 0o644)

	if err := writeMarketplaceJSONVersion(path, "3.4.0"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var got map[string]any
	json.Unmarshal(raw, &got)
	plugins := got["plugins"].([]any)
	primary := plugins[0].(map[string]any)
	if primary["version"] != "3.4.0" {
		t.Errorf("primary plugin version=%v want 3.4.0", primary["version"])
	}
	other := plugins[1].(map[string]any)
	if other["version"] != "1.0.0" {
		t.Errorf("other plugin should not change")
	}
}

func TestWriteClaudeMDVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	original := "# Title\n\n- **Version**: 3.3.0 (2026-04-14) - notes\n\nbody...\n"
	os.WriteFile(path, []byte(original), 0o644)
	if err := writeClaudeMDVersion(path, "3.4.0"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "**Version**: 3.4.0") {
		t.Errorf("version not updated: %s", data)
	}
}

func TestPrependChangelogEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")
	original := "# Changelog\n\n## v3.3.0 (2026-04-14)\n\nexisting content\n"
	os.WriteFile(path, []byte(original), 0o644)
	if err := prependChangelogEntry(path, "3.4.0"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	s := string(data)
	if !strings.HasPrefix(s, "# Changelog\n") {
		t.Errorf("header lost")
	}
	if !strings.Contains(s, "## v3.4.0") {
		t.Errorf("new entry missing")
	}
	if !strings.Contains(s, "## v3.3.0") {
		t.Errorf("old entry lost")
	}
	// new entry before old
	newIdx := strings.Index(s, "## v3.4.0")
	oldIdx := strings.Index(s, "## v3.3.0")
	if newIdx >= oldIdx {
		t.Errorf("new entry should be prepended above old")
	}
}
