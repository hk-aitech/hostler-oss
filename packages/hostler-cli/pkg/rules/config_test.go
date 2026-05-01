package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseSeverity_Defaults — string → Severity mapping works correctly.
func TestParseSeverity_Defaults(t *testing.T) {
	cases := map[string]Severity{
		"off":        SeverityOff,
		"advisory":   SeverityAdvisory,
		"warn":       SeverityWarn,
		"block":      SeverityBlock,
		"hard-block": SeverityHardBlock,
		"HARD-BLOCK": SeverityHardBlock,
	}
	for in, want := range cases {
		got, err := ParseSeverity(in)
		if err != nil {
			t.Errorf("ParseSeverity(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("ParseSeverity(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := ParseSeverity("invalid"); err == nil {
		t.Error("invalid severity should return an error")
	}
}

// TestLoadConfig_NoFile_Fallback — when .hstl/rules.yaml is absent, the
// recommended preset is used.
func TestLoadConfig_NoFile_Fallback(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Extends) != 1 || cfg.Extends[0] != "preset://hostler/recommended" {
		t.Errorf("fallback extends = %v, want [preset://hostler/recommended]", cfg.Extends)
	}
}

// TestLoadConfig_MinimalYAML — load a minimal rules.yaml.
func TestLoadConfig_MinimalYAML(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".hstl"), 0o755)
	content := []byte(`version: 1
extends:
  - preset://hostler/recommended
rules:
  commit.message.korean: block
`)
	_ = os.WriteFile(filepath.Join(dir, ".hstl", "rules.yaml"), content, 0o644)

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("version = %d, want 1", cfg.Version)
	}
	if cfg.Rules["commit.message.korean"] != "block" {
		t.Errorf("rules.commit.message.korean = %q", cfg.Rules["commit.message.korean"])
	}
}

// TestResolveEffective_PresetChain — resolve the extends chain.
func TestResolveEffective_PresetChain(t *testing.T) {
	cfg := &EngineConfig{
		Version: 1,
		Extends: []string{"preset://hostler/recommended"},
	}
	eff, err := ResolveEffective(cfg, "")
	if err != nil {
		t.Fatalf("ResolveEffective: %v", err)
	}
	// recommended has task.result_section.files_exist as a top-level rule.
	r, ok := eff["task.result_section.files_exist"]
	if !ok {
		t.Fatal("task.result_section.files_exist missing from cascade result")
	}
	if r.Severity != SeverityBlock {
		t.Errorf("severity = %v, want block", r.Severity)
	}
	if r.Origin.Layer != "preset" {
		t.Errorf("origin.layer = %q, want preset", r.Origin.Layer)
	}
}

// TestResolveEffective_RulesOverrideTakesPrecedence — top-level rules
// override preset values.
func TestResolveEffective_RulesOverrideTakesPrecedence(t *testing.T) {
	cfg := &EngineConfig{
		Version: 1,
		Extends: []string{"preset://hostler/recommended"},
		Rules: map[string]string{
			"task.result_section.files_exist": "hard-block",
		},
	}
	eff, err := ResolveEffective(cfg, "")
	if err != nil {
		t.Fatalf("ResolveEffective: %v", err)
	}
	r := eff["task.result_section.files_exist"]
	if r.Severity != SeverityHardBlock {
		t.Errorf("severity = %v, want hard-block", r.Severity)
	}
	if r.Origin.Layer != "rules" {
		t.Errorf("origin.layer = %q, want rules", r.Origin.Layer)
	}
	if len(r.PreviousChain) == 0 {
		t.Error("PreviousChain empty — preset stage should be recorded")
	}
}

// TestResolveEffective_PoliciesSeverityOverrides — policy layer override.
func TestResolveEffective_PoliciesSeverityOverrides(t *testing.T) {
	cfg := &EngineConfig{
		Version: 1,
		Extends: []string{"preset://hostler/recommended"},
		Policies: []PolicyConfig{
			{
				Name: "team/stricter",
				SeverityOverrides: map[string]Severity{
					"task.result_section.files_exist": SeverityHardBlock,
				},
			},
		},
	}
	eff, _ := ResolveEffective(cfg, "")
	r := eff["task.result_section.files_exist"]
	if r.Severity != SeverityHardBlock {
		t.Errorf("severity = %v, want hard-block", r.Severity)
	}
	if r.Origin.Layer != "policy" || r.Origin.Source != "team/stricter" {
		t.Errorf("origin = %+v, want policy/team/stricter", r.Origin)
	}
}

// TestResolveEffective_EnvOverride — environment variable has the final
// say.
func TestResolveEffective_EnvOverride(t *testing.T) {
	t.Setenv("HSTL_RULE_TASK_RESULT_SECTION_FILES_EXIST", "advisory")
	cfg := &EngineConfig{
		Version: 1,
		Extends: []string{"preset://hostler/recommended"},
		Rules: map[string]string{
			"task.result_section.files_exist": "block",
		},
	}
	eff, _ := ResolveEffective(cfg, "")
	r := eff["task.result_section.files_exist"]
	if r.Severity != SeverityAdvisory {
		t.Errorf("severity = %v, want advisory", r.Severity)
	}
	if r.Origin.Layer != "env" {
		t.Errorf("origin.layer = %q, want env", r.Origin.Layer)
	}
}

// TestResolveEffective_HardBlock_EnvForbidden — hard-block cannot be env-overridden.
func TestResolveEffective_HardBlock_EnvForbidden(t *testing.T) {
	t.Setenv("HSTL_RULE_CRIT_RULE", "off")
	cfg := &EngineConfig{
		Version: 1,
		Rules: map[string]string{
			"crit.rule": "hard-block",
		},
	}
	eff, _ := ResolveEffective(cfg, "")
	r := eff["crit.rule"]
	if r.Severity != SeverityHardBlock {
		t.Errorf("hard-block must not be env-overridable, got %v", r.Severity)
	}
	if r.Origin.Layer == "env" {
		t.Error("hard-block origin was changed to env (protection failed)")
	}
}

// TestResolveEffective_OffDisables — off disables a rule.
func TestResolveEffective_OffDisables(t *testing.T) {
	cfg := &EngineConfig{
		Version: 1,
		Extends: []string{"preset://hostler/recommended"},
		Rules: map[string]string{
			"task.result_section.files_exist": "off",
		},
	}
	eff, _ := ResolveEffective(cfg, "")
	r := eff["task.result_section.files_exist"]
	if r.Severity != SeverityOff {
		t.Errorf("severity = %v, want off", r.Severity)
	}
}

// TestResolveEffective_PresetChainExtension — strict inherits recommended.
func TestResolveEffective_PresetChainExtension(t *testing.T) {
	cfg := &EngineConfig{
		Version: 1,
		Extends: []string{"preset://hostler/strict"},
	}
	eff, _ := ResolveEffective(cfg, "")
	r := eff["task.result_section.files_exist"]
	if r == nil {
		t.Fatal("task.result_section.files_exist missing")
		return // nolint (SA5011 guard)
	}
	if r.Severity != SeverityHardBlock {
		t.Errorf("strict severity = %v, want hard-block", r.Severity)
	}
	// PreviousChain must record the recommended stage.
	if len(r.PreviousChain) == 0 {
		t.Error("PreviousChain empty — recommended stage should be recorded")
	}
}

// TestResolveEffective_CycleDetection — extending oneself returns an error.
func TestResolveEffective_CycleDetection(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.yaml")
	bPath := filepath.Join(dir, "b.yaml")
	_ = os.WriteFile(aPath, []byte("version: 1\nextends: ['file://b.yaml']\n"), 0o644)
	_ = os.WriteFile(bPath, []byte("version: 1\nextends: ['file://a.yaml']\n"), 0o644)

	cfg := &EngineConfig{Extends: []string{"file://a.yaml"}}
	_, err := ResolveEffective(cfg, dir)
	if err == nil {
		t.Error("expected error on cyclic extends, got nil")
	}
}

// TestPoliciesForPath_GlobMatch — Target glob resolution.
func TestPoliciesForPath_GlobMatch(t *testing.T) {
	cfg := &EngineConfig{
		Targets: []TargetConfig{
			{Paths: []string{"*.go"}, Policies: []string{"go-strict"}},
			{Paths: []string{"*.md"}, Policies: []string{"doc-light"}},
		},
	}
	if got := cfg.PoliciesForPath("main.go"); len(got) != 1 || got[0] != "go-strict" {
		t.Errorf("main.go match = %v, want [go-strict]", got)
	}
	if got := cfg.PoliciesForPath("README.md"); len(got) != 1 || got[0] != "doc-light" {
		t.Errorf("README.md match = %v", got)
	}
	if got := cfg.PoliciesForPath("x.txt"); len(got) != 0 {
		t.Errorf("x.txt match = %v, want []", got)
	}
}

// TestLoadConfigBytes_ParseError — invalid yaml returns an error.
func TestLoadConfigBytes_ParseError(t *testing.T) {
	_, err := LoadConfigBytes([]byte("invalid: [unclosed"))
	if err == nil {
		t.Error("expected error on invalid yaml")
	}
}

// TestLoadConfig_EmptyFile — empty yaml yields an empty EngineConfig.
func TestLoadConfig_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".hstl"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".hstl", "rules.yaml"), []byte(""), 0o644)
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Version != 0 {
		t.Errorf("empty file version = %d, want 0", cfg.Version)
	}
}
