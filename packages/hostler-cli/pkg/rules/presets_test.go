package rules

import "testing"

// TestPresetManifest_Load — manifest.json parsing + required fields.
func TestPresetManifest_Load(t *testing.T) {
	m, err := LoadPresetManifest()
	if err != nil {
		t.Fatalf("LoadPresetManifest: %v", err)
	}
	if m.Revision == "" {
		t.Error("revision is empty")
	}
	if m.PresetCount != 3 {
		t.Errorf("preset_count = %d, want 3", m.PresetCount)
	}
	if len(m.Presets) != 3 {
		t.Errorf("presets = %v, want 3 entries", m.Presets)
	}
}

// TestRecommended_RuleCount — at least 14 rules after the recommended
// preset cascade.
func TestRecommended_RuleCount(t *testing.T) {
	cfg := &EngineConfig{Extends: []string{"preset://hostler/recommended"}}
	eff, err := ResolveEffective(cfg, "")
	if err != nil {
		t.Fatalf("ResolveEffective: %v", err)
	}
	if len(eff) < 14 {
		t.Errorf("recommended rule count = %d, want >= 14", len(eff))
	}
	// Confirm core rules are present.
	mustHave := []string{
		"task.feature.criteria_checked",
		"task.bugfix.reproduction",
		"task.infra.deploy_verified",
		"sprint.phase.doc_review",
		"commit.message.korean",
	}
	for _, id := range mustHave {
		if _, ok := eff[id]; !ok {
			t.Errorf("required rule missing: %s", id)
		}
	}
}

// TestStrict_SeverityElevated — strict overrides recommended.
func TestStrict_SeverityElevated(t *testing.T) {
	cfg := &EngineConfig{Extends: []string{"preset://hostler/strict"}}
	eff, err := ResolveEffective(cfg, "")
	if err != nil {
		t.Fatalf("ResolveEffective: %v", err)
	}
	// strict elevates lint_passed to block.
	r, ok := eff["task.feature.lint_passed"]
	if !ok {
		t.Fatal("task.feature.lint_passed missing")
	}
	if r.Severity != SeverityBlock {
		t.Errorf("strict task.feature.lint_passed = %v, want block", r.Severity)
	}
	// Result-section validation is hard-block.
	if eff["task.result_section.files_exist"].Severity != SeverityHardBlock {
		t.Errorf("strict result_section = %v, want hard-block", eff["task.result_section.files_exist"].Severity)
	}
}

// TestRelaxed_SeverityRelaxed — relaxed loosens recommended.
func TestRelaxed_SeverityRelaxed(t *testing.T) {
	cfg := &EngineConfig{Extends: []string{"preset://hostler/relaxed"}}
	eff, _ := ResolveEffective(cfg, "")
	if eff["commit.message.korean"].Severity != SeverityAdvisory {
		t.Errorf("relaxed commit.message.korean = %v, want advisory", eff["commit.message.korean"].Severity)
	}
	if eff["sprint.phase.retro"].Severity != SeverityWarn {
		t.Errorf("relaxed sprint.phase.retro = %v, want warn", eff["sprint.phase.retro"].Severity)
	}
}

// TestPreset_NoCycle — all 3 presets load without cycles.
func TestPreset_NoCycle(t *testing.T) {
	for _, name := range []string{"recommended", "strict", "relaxed"} {
		cfg := &EngineConfig{Extends: []string{"preset://hostler/" + name}}
		if _, err := ResolveEffective(cfg, ""); err != nil {
			t.Errorf("%s load failed: %v", name, err)
		}
	}
}
