package config

import (
	"testing"
)

// -- T209 / I07 Profile Composition tests --

func TestLoadPreset_HostlerBase(t *testing.T) {
	m, err := LoadPreset("builtin:hostler-base")
	if err != nil {
		t.Fatalf("LoadPreset error: %v", err)
	}
	if m == nil {
		t.Fatal("LoadPreset nil map")
	}
	tasks, ok := m["tasks"].(map[string]any)
	if !ok {
		t.Fatal("hostler-base has no tasks section")
	}
	if _, ok := tasks["result_check"]; !ok {
		t.Errorf("hostler-base.tasks.result_check missing")
	}
}

func TestLoadPreset_GoPlugin(t *testing.T) {
	m, err := LoadPreset("builtin:go-plugin")
	if err != nil {
		t.Fatalf("LoadPreset error: %v", err)
	}
	platform, ok := m["platform"].(map[string]any)
	if !ok {
		t.Fatal("go-plugin has no platform section")
	}
	if kind := platform["kind"]; kind != "go" {
		t.Errorf("platform.kind = %v (expected: go)", kind)
	}
}

func TestLoadPreset_Unknown(t *testing.T) {
	_, err := LoadPreset("builtin:nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown preset")
	}
}

func TestLoadPreset_InvalidPrefix(t *testing.T) {
	_, err := LoadPreset("nonbuiltin:foo")
	if err == nil {
		t.Fatal("expected error for non-builtin prefix")
	}
}

func TestResolveExtends_SinglePreset(t *testing.T) {
	merged, err := ResolveExtends([]string{"builtin:go-plugin"}, 0, nil)
	if err != nil {
		t.Fatalf("ResolveExtends error: %v", err)
	}
	if merged == nil {
		t.Fatal("merged nil")
	}
	platform, ok := merged["platform"].(map[string]any)
	if !ok || platform["kind"] != "go" {
		t.Errorf("platform.kind merge failed")
	}
}

func TestResolveExtends_Chain_LaterWins(t *testing.T) {
	// hostler-base + go-plugin: where the two overlap, go-plugin wins.
	merged, err := ResolveExtends(
		[]string{"builtin:hostler-base", "builtin:go-plugin"}, 0, nil,
	)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// tasks.result_check.policy from hostler-base must be preserved.
	tasks, ok := merged["tasks"].(map[string]any)
	if !ok {
		t.Fatal("merged.tasks missing")
	}
	rc, _ := tasks["result_check"].(map[string]any)
	if rc["policy"] != "strict" {
		t.Errorf("tasks.result_check.policy = %v (expected: strict)", rc["policy"])
	}
	// platform.kind from go-plugin must be go.
	platform, _ := merged["platform"].(map[string]any)
	if platform["kind"] != "go" {
		t.Errorf("platform.kind = %v (expected: go)", platform["kind"])
	}
	// skills.code-review from go-plugin must be merged.
	skills, _ := merged["skills"].(map[string]any)
	if _, ok := skills["code-review"]; !ok {
		t.Errorf("skills.code-review missing")
	}
}

func TestMergeMaps_DeepMerge(t *testing.T) {
	dst := map[string]any{
		"a": 1,
		"nested": map[string]any{
			"x": 10,
			"y": 20,
		},
	}
	src := map[string]any{
		"b": 2,
		"nested": map[string]any{
			"y": 99, // overwrite
			"z": 30, // add
		},
	}
	merged := mergeMaps(dst, src)
	if merged["a"] != 1 || merged["b"] != 2 {
		t.Errorf("top-level merge failed: %+v", merged)
	}
	n, _ := merged["nested"].(map[string]any)
	if n["x"] != 10 || n["y"] != 99 || n["z"] != 30 {
		t.Errorf("nested merge failed: %+v", n)
	}
}

func TestMergeMaps_ArrayConcat(t *testing.T) {
	dst := map[string]any{"rules": []any{"a", "b"}}
	src := map[string]any{"rules": []any{"c", "d"}}
	merged := mergeMaps(dst, src)
	rules, _ := merged["rules"].([]any)
	if len(rules) != 4 {
		t.Errorf("concat failed: %+v", rules)
	}
	if rules[0] != "a" || rules[3] != "d" {
		t.Errorf("concat order wrong: %+v", rules)
	}
}

func TestMergeMaps_ScalarOverride(t *testing.T) {
	dst := map[string]any{"version": "1.0"}
	src := map[string]any{"version": "2.0"}
	merged := mergeMaps(dst, src)
	if merged["version"] != "2.0" {
		t.Errorf("scalar override failed: %v", merged["version"])
	}
}

func TestApplyExtendsAndMerge_NoExtends(t *testing.T) {
	data := []byte("project:\n  key: foo\n")
	merged, err := applyExtendsAndMerge(data)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if merged["project"] == nil {
		t.Error("project key missing")
	}
	if _, hasExt := merged["extends"]; hasExt {
		t.Error("extends key still present")
	}
}

func TestApplyExtendsAndMerge_WithPreset(t *testing.T) {
	data := []byte(`
extends:
  - builtin:hostler-base
  - builtin:go-plugin
project:
  key: my-plugin
platform:
  go:
    module: example.com/test
`)
	merged, err := applyExtendsAndMerge(data)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Project wins overall: project.key is preserved.
	proj, _ := merged["project"].(map[string]any)
	if proj["key"] != "my-plugin" {
		t.Errorf("project.key lost: %v", proj["key"])
	}
	// platform.kind from the go-plugin preset.
	plat, _ := merged["platform"].(map[string]any)
	if plat["kind"] != "go" {
		t.Errorf("platform.kind preset missing")
	}
	// platform.go.module from the project (must coexist with the preset's lint).
	plGo, _ := plat["go"].(map[string]any)
	if plGo["module"] != "example.com/test" {
		t.Errorf("platform.go.module lost: %v", plGo["module"])
	}
	if _, ok := plGo["lint"].([]any); !ok {
		t.Errorf("platform.go.lint (from preset) missing")
	}
	// tasks.result_check from hostler-base.
	if _, ok := merged["tasks"].(map[string]any); !ok {
		t.Errorf("tasks section missing (hostler-base preset)")
	}
	// extends is preserved for traceability.
	if _, hasExt := merged["extends"]; !hasExt {
		t.Errorf("extends tracking field lost")
	}
}

func TestValidateProjectConfigYAML_WithExtends(t *testing.T) {
	data := []byte(`
extends:
  - builtin:hostler-base
  - builtin:go-plugin
project:
  key: plugin
platform:
  go:
    module: example.com/test
`)
	cfg, verr, err := ValidateProjectConfigYAML(data)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if verr != nil {
		t.Errorf("validation failed: %v", verr)
	}
	if cfg == nil {
		t.Fatal("cfg nil")
	}
	if !cfg.IsGo() {
		t.Errorf("IsGo() = false (expected: true)")
	}
	if cfg.Platform == nil || cfg.Platform.Go == nil || cfg.Platform.Go.Module != "example.com/test" {
		t.Errorf("Platform.Go merge failed: %+v", cfg.Platform)
	}
	// tasks.result_check from hostler-base must unmarshal into the struct.
	if cfg.Tasks == nil || cfg.Tasks.ResultCheck == nil {
		t.Errorf("Tasks.ResultCheck merge failed")
	} else if cfg.Tasks.ResultCheck.Policy != "strict" {
		t.Errorf("ResultCheck.Policy = %q (expected: strict)", cfg.Tasks.ResultCheck.Policy)
	}
	// skills.code-review from go-plugin.
	if rules := cfg.GetSkillRules("code-review"); len(rules) == 0 {
		t.Errorf("code-review rules merge failed")
	}
	// Extends field tracking preserved.
	if len(cfg.Extends) != 2 {
		t.Errorf("Extends field = %+v (expected: 2 entries)", cfg.Extends)
	}
}
