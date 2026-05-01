// Package task — T701 (Sprint-81) static heuristics regression tests.
package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestT701_ResolvePreset_UnknownFallsBackToStrict(t *testing.T) {
	cfg := ResolvePreset("unknown-preset")
	if cfg.Preset != PresetStrict {
		t.Errorf("unknown preset fallback = %q, want strict", cfg.Preset)
	}
}

func TestT701_ResolvePreset_Off_BypassesAllChecks(t *testing.T) {
	cfg := ResolvePreset("off")
	issues, ok := HeuristicCheck("x", cfg) // extremely short summary
	if !ok || len(issues) > 0 {
		t.Errorf("off preset triggered validation: issues=%v", issues)
	}
}

func TestT701_Strict_RejectsSoloKeyword(t *testing.T) {
	cfg := ResolvePreset("strict")
	for _, bad := range []string{
		"fixed a bug",
		"update",
		"fix",
		"refactor cleanup",
	} {
		issues, ok := HeuristicCheck(bad, cfg)
		if ok {
			t.Errorf("strict must not accept: %q (issues=%v)", bad, issues)
		}
	}
}

func TestT701_Strict_AcceptsProperSummary(t *testing.T) {
	cfg := ResolvePreset("strict")
	good := "Wrap transitionTask in a SQLite transaction in order to prevent DB status drift; the done criterion is the T677 reproduction test going green"
	issues, ok := HeuristicCheck(good, cfg)
	if !ok {
		t.Errorf("strict rejected a good summary: %v", issues)
	}
}

func TestT701_Strict_RejectsMissingPurposeConnector(t *testing.T) {
	cfg := ResolvePreset("strict")
	bad := "file move stage transactional atomicity hardening SQLite adapter store switchover" // no connector
	issues, ok := HeuristicCheck(bad, cfg)
	if ok {
		t.Errorf("a summary without a purpose connector passed: %v", issues)
	}
	joined := strings.Join(issues, "|")
	if !strings.Contains(joined, "purpose") && !strings.Contains(joined, "connector") {
		t.Errorf("expected an issue mentioning purpose/connector, got: %s", joined)
	}
}

func TestT701_Moderate_OnlyHeuristicsNoLLM(t *testing.T) {
	cfg := ResolvePreset("moderate")
	if cfg.EnableLLMJudge {
		t.Error("moderate has LLM judge enabled")
	}
	if len(cfg.RequiredPurposeConnector) != 0 {
		t.Error("moderate should not require a connector")
	}
}

func TestT701_Lenient_AcceptsShortButNotSoloFix(t *testing.T) {
	cfg := ResolvePreset("lenient")
	// solo "fix" — expected reject
	if _, ok := HeuristicCheck("fix", cfg); ok {
		t.Error("lenient accepted solo 'fix'")
	}
	// generic short description — expected pass
	if _, ok := HeuristicCheck("Add a TLS certificate renewal script", cfg); !ok {
		t.Error("lenient rejected an ordinary short summary")
	}
}

func TestT701_HeuristicCheck_CountUniqueWords(t *testing.T) {
	if n := countUniqueWords("one two one three"); n != 3 {
		t.Errorf("unique-word count error: %d, want 3", n)
	}
	if n := countUniqueWords("a b, c; a!"); n != 3 {
		t.Errorf("unique-word count with punctuation error: %d, want 3", n)
	}
}

func TestT701_PresetEnvOverrideViaCaller(t *testing.T) {
	// Simulate the path where a caller selects preset via env.
	orig := os.Getenv("HSTL_SUMMARY_POLICY")
	defer os.Setenv("HSTL_SUMMARY_POLICY", orig)

	os.Setenv("HSTL_SUMMARY_POLICY", "off")
	cfg := ResolvePreset(os.Getenv("HSTL_SUMMARY_POLICY"))
	if cfg.Preset != PresetOff {
		t.Errorf("env=off but preset = %q", cfg.Preset)
	}
}

// TestT705_LoadProjectPreset_Overrides_Embed — regression-tests that a
// project-local override is loaded with priority over the embed YAML.
// New in T705 Sprint-82.
//
// Scenario:
//  1. Create .hostler/summary-validation/strict.yaml under a tmp project root
//  2. Set MinUniqueWords = 99 (sentinel) in that file
//  3. Point HSTL_PROJECT_ROOT at the tmp dir
//  4. Confirm ResolvePreset("strict") returns 99 (not the embed default)
func TestT705_LoadProjectPreset_Overrides_Embed(t *testing.T) {
	tmpRoot := t.TempDir()
	presetDir := filepath.Join(tmpRoot, ".hostler", "summary-validation")
	if err := os.MkdirAll(presetDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	// Sentinel value to clearly differ from the embed default.
	const sentinelMinWords = 99
	yaml := `preset: strict
min_unique_words: 99
required_purpose_connectors: []
`
	path := filepath.Join(presetDir, "strict.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	orig := os.Getenv("HSTL_PROJECT_ROOT")
	defer os.Setenv("HSTL_PROJECT_ROOT", orig)
	os.Setenv("HSTL_PROJECT_ROOT", tmpRoot)

	cfg := ResolvePreset("strict")
	if cfg.MinUniqueWords != sentinelMinWords {
		t.Errorf("loadProjectPreset priority failed: MinUniqueWords=%d, want %d (project YAML override not applied)",
			cfg.MinUniqueWords, sentinelMinWords)
	}
	if len(cfg.RequiredPurposeConnector) != 0 {
		t.Errorf("override field not honoured: RequiredPurposeConnector=%v, want empty slice", cfg.RequiredPurposeConnector)
	}
}

// TestT705_LoadProjectPreset_Missing_FallsBackToEmbed — confirms the embed
// YAML fallback works when no project-local override exists.
func TestT705_LoadProjectPreset_Missing_FallsBackToEmbed(t *testing.T) {
	tmpRoot := t.TempDir() // empty dir (no override file)

	orig := os.Getenv("HSTL_PROJECT_ROOT")
	defer os.Setenv("HSTL_PROJECT_ROOT", orig)
	os.Setenv("HSTL_PROJECT_ROOT", tmpRoot)

	cfg := ResolvePreset("strict")
	if cfg.Preset != PresetStrict {
		t.Errorf("fallback failed: preset=%q, want strict", cfg.Preset)
	}
	// embed strict.yaml default should be a practical value (not 99).
	if cfg.MinUniqueWords >= 99 {
		t.Errorf("sentinel applied without override: MinUniqueWords=%d", cfg.MinUniqueWords)
	}
}
