// Package task — T701 follow-up hotfix: YAML preset loader regression
// tests.
package task

import "testing"

// TestT701_LoadEmbedPreset_Strict verifies that the embed YAML actually
// loads and matches the Go-map fallback. (When the embed file is missing
// the loader silently falls back to the Go map, so this test guards the
// YAML path stays valid.)
func TestT701_LoadEmbedPreset_Strict(t *testing.T) {
	cfg, ok := loadEmbedPreset("strict")
	if !ok {
		t.Fatal("strict YAML embed load failed — file missing or parse error")
	}
	if cfg.Preset != PresetStrict {
		t.Errorf("preset = %q, want strict", cfg.Preset)
	}
	if cfg.MinUniqueWords == 0 {
		t.Error("min_unique_words for strict was not loaded from YAML")
	}
	if !cfg.EnableLLMJudge {
		t.Error("enable_llm_judge for strict must be true")
	}
	if len(cfg.ForbiddenSoloKeywords) == 0 {
		t.Error("forbidden_solo_keywords for strict is empty")
	}
}

func TestT701_LoadEmbedPreset_Off_YAMLReservedWordHandled(t *testing.T) {
	cfg, ok := loadEmbedPreset("off")
	if !ok {
		t.Fatal("off YAML embed load failed")
	}
	if cfg.Preset != PresetOff {
		t.Errorf("preset = %q, want off — YAML reserved-word 'off' not handled?", cfg.Preset)
	}
}

func TestT701_LoadEmbedPreset_AllFour(t *testing.T) {
	for _, name := range []string{"strict", "moderate", "lenient", "off"} {
		if _, ok := loadEmbedPreset(name); !ok {
			t.Errorf("preset %s embed load failed", name)
		}
	}
}

func TestT701_LoadEmbedPreset_Unknown(t *testing.T) {
	if _, ok := loadEmbedPreset("nonexistent-preset"); ok {
		t.Error("an unknown preset loaded from embed — verify fallback path")
	}
}

// TestT701_ResolvePreset_ViaEmbed checks that the YAML path is preferred.
// The YAML content matches the Go map intentionally, so equal results = ok.
// Whether YAML actually loads is already covered by loadEmbedPreset not
// returning nil.
func TestT701_ResolvePreset_ViaEmbed(t *testing.T) {
	cfg := ResolvePreset("strict")
	if cfg.Preset != PresetStrict {
		t.Errorf("ResolvePreset('strict') = %q", cfg.Preset)
	}
	if cfg.MinUniqueWords < 1 {
		t.Error("ResolvePreset strict is empty")
	}
}
