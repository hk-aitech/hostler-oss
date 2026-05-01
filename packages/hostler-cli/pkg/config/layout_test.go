// T137 (Sprint-07): LayoutConfig parser + default fallback tests.
package config

import (
	"testing"
)

// TestT137_Layout_DefaultFallback — when Layout is unset, returns the default (works/).
func TestT137_Layout_DefaultFallback(t *testing.T) {
	c := &ProjectConfig{} // Layout unset
	if got := c.ResolvedSprintsRoot(); got != "works/sprints" {
		t.Errorf("ResolvedSprintsRoot default: got=%q, want=%q", got, "works/sprints")
	}
	if got := c.ResolvedTasksRoot(); got != "works/tasks" {
		t.Errorf("ResolvedTasksRoot default: got=%q, want=%q", got, "works/tasks")
	}
}

// TestT137_Layout_NilSafe — nil ProjectConfig still returns the default (no crash).
func TestT137_Layout_NilSafe(t *testing.T) {
	var c *ProjectConfig
	if got := c.ResolvedSprintsRoot(); got != "works/sprints" {
		t.Errorf("nil ResolvedSprintsRoot: got=%q", got)
	}
	if got := c.ResolvedTasksRoot(); got != "works/tasks" {
		t.Errorf("nil ResolvedTasksRoot: got=%q", got)
	}
}

// TestT137_Layout_ExplicitOverride — when Layout is set explicitly, returns the configured value.
func TestT137_Layout_ExplicitOverride(t *testing.T) {
	c := &ProjectConfig{
		Layout: &LayoutConfig{
			SprintsRoot: "custom/sprints",
			TasksRoot:   "custom/tasks",
		},
	}
	if got := c.ResolvedSprintsRoot(); got != "custom/sprints" {
		t.Errorf("override SprintsRoot: got=%q, want=%q", got, "custom/sprints")
	}
	if got := c.ResolvedTasksRoot(); got != "custom/tasks" {
		t.Errorf("override TasksRoot: got=%q, want=%q", got, "custom/tasks")
	}
}

// TestT137_Layout_YAMLParse — confirms the layout section parses from YAML.
func TestT137_Layout_YAMLParse(t *testing.T) {
	tmp := setupTempRoot(t)
	yaml := `
version: "2.0.0"
project:
  key: test-layout
layout:
  sprints_root: hostler/sprints
  tasks_root: hostler/tasks
`
	writeTestYAML(t, tmp, yaml)

	cfg, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg nil")
	}
	if cfg.Layout == nil {
		t.Fatal("Layout nil — YAML parsing failed")
	}
	if cfg.Layout.SprintsRoot != "hostler/sprints" {
		t.Errorf("YAML SprintsRoot: got=%q", cfg.Layout.SprintsRoot)
	}
	if cfg.ResolvedSprintsRoot() != "hostler/sprints" {
		t.Errorf("Resolved: got=%q", cfg.ResolvedSprintsRoot())
	}
}
