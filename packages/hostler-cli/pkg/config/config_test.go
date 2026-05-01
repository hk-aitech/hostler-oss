package config

import (
	"strings"
	"testing"
)

// T424 (Sprint-50) — exercises the core public API of pkg/config against the v2 schema.
//
// Rewrites the previous config_test.go (25 tests, deleted in T402 (Sprint-48)
// because of v1 fixture dependence) so it targets the v2 canonical schema
// only. Migration Runner tests are excluded because the Migration Runner
// itself was removed.
//
// helpers: see precedence_test.go for setupTempRoot / writeTestYAML.

const v2FixtureMinimal = `version: "2.0.0"
project:
  key: test-min
  name: Test Minimal
`

const v2FixtureFull = `version: "2.0.0"
project:
  key: test-full
  name: Test Full Config
platform:
  kind: go
  go:
    module: example.com/test-full
tasks:
  result_check:
    policy: warn
    section_titles:
      - Result
      - Deliverables
      - Outcome
sprints:
  ceremony:
    start:
      - "git checkout -b {sprint_id}"
    complete:
      - "Phase 1: doc-review"
    design_checks:
      - .hstl/scripts/check-custom.sh
skills:
  code-review:
    rules:
      - "go vet must pass"
      - "detect magic constants"
  doc-cross-check:
    rules:
      - "validate CLAUDE.md vs manifest drift"
`

// ---------------------------------------------------------------------------
// LoadProjectConfig
// ---------------------------------------------------------------------------

func TestLoadProjectConfig_MinimalV2(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureMinimal)

	cfg, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig err = %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg nil")
		return // nolint (SA5011 guard)
	}
	if cfg.Version != "2.0.0" {
		t.Errorf("Version = %q, want 2.0.0", cfg.Version)
	}
	if cfg.Project.Key != "test-min" {
		t.Errorf("Project.Key = %q", cfg.Project.Key)
	}
}

func TestLoadProjectConfig_FullV2(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	cfg, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig err = %v", err)
	}
	if cfg.Platform == nil || cfg.Platform.Kind != "go" {
		t.Errorf("Platform.Kind missing")
	}
	if cfg.Sprints == nil || cfg.Sprints.Ceremony == nil {
		t.Fatal("Sprints.Ceremony nil")
	}
	if len(cfg.Sprints.Ceremony.DesignChecks) != 1 {
		t.Errorf("DesignChecks len = %d", len(cfg.Sprints.Ceremony.DesignChecks))
	}
}

func TestLoadProjectConfig_Missing_NilNoError(t *testing.T) {
	setupTempRoot(t)
	cfg, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("missing yaml must not be an error: %v", err)
	}
	// Missing yaml returns a nil cfg (consumers must nil-check).
	if cfg != nil {
		t.Errorf("missing yaml: want nil cfg, got %+v", cfg)
	}
}

// ---------------------------------------------------------------------------
// GetProjectKey
// ---------------------------------------------------------------------------

func TestGetProjectKey_FromYAML(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureMinimal)

	if got := GetProjectKey(); got != "test-min" {
		t.Errorf("GetProjectKey = %q, want test-min", got)
	}
}

func TestGetProjectKey_Missing(t *testing.T) {
	setupTempRoot(t)
	if got := GetProjectKey(); got != "" {
		t.Errorf("GetProjectKey on missing yaml = %q, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// GetTaskResultCheckPolicy (T182)
// ---------------------------------------------------------------------------

func TestGetTaskResultCheckPolicy_DefaultStrict(t *testing.T) {
	setupTempRoot(t)
	if got := GetTaskResultCheckPolicy(); got != "strict" {
		t.Errorf("default = %q, want strict", got)
	}
}

func TestGetTaskResultCheckPolicy_FromYAML(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	if got := GetTaskResultCheckPolicy(); got != "warn" {
		t.Errorf("yaml policy = %q, want warn", got)
	}
}

func TestGetTaskResultCheckPolicy_EnvWins(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull) // yaml = warn
	t.Setenv("HSTL_TASK_RESULT_CHECK_POLICY", "off")

	if got := GetTaskResultCheckPolicy(); got != "off" {
		t.Errorf("env override = %q, want off", got)
	}
}

func TestGetTaskResultCheckPolicy_InvalidFallsBack(t *testing.T) {
	setupTempRoot(t)
	t.Setenv("HSTL_TASK_RESULT_CHECK_POLICY", "bogus")
	if got := GetTaskResultCheckPolicy(); got != "strict" {
		t.Errorf("invalid env should fall back to default, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// GetTaskResultSectionTitles
// ---------------------------------------------------------------------------

func TestGetTaskResultSectionTitles_Default(t *testing.T) {
	setupTempRoot(t)
	titles := GetTaskResultSectionTitles()
	if len(titles) != 2 || titles[0] != "Result" || titles[1] != "Deliverables" {
		t.Errorf("default titles = %v", titles)
	}
}

func TestGetTaskResultSectionTitles_FromYAML(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	titles := GetTaskResultSectionTitles()
	if len(titles) != 3 {
		t.Fatalf("want 3 titles, got %d: %v", len(titles), titles)
	}
	joined := strings.Join(titles, ",")
	if !strings.Contains(joined, "Outcome") {
		t.Errorf("yaml titles not reflected: %v", titles)
	}
}

func TestGetTaskResultSectionTitles_EnvOverride(t *testing.T) {
	setupTempRoot(t)
	t.Setenv("HSTL_TASK_RESULT_SECTION_TITLES", "Outputs, Deliverables")
	titles := GetTaskResultSectionTitles()
	if len(titles) != 2 || titles[0] != "Outputs" || titles[1] != "Deliverables" {
		t.Errorf("env override = %v", titles)
	}
}

func TestGetTaskResultSectionTitles_DedupPreservesOrder(t *testing.T) {
	setupTempRoot(t)
	t.Setenv("HSTL_TASK_RESULT_SECTION_TITLES", "A,B,A,C,B")
	titles := GetTaskResultSectionTitles()
	want := []string{"A", "B", "C"}
	if len(titles) != len(want) {
		t.Fatalf("dedup want %v, got %v", want, titles)
	}
	for i, w := range want {
		if titles[i] != w {
			t.Errorf("pos %d: want %q, got %q", i, w, titles[i])
		}
	}
}

// ---------------------------------------------------------------------------
// GetSprintCeremony
// ---------------------------------------------------------------------------

func TestGetSprintCeremony_FromYAML(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	cer := GetSprintCeremony()
	if cer == nil {
		t.Fatal("SprintCeremony nil")
		return // nolint (SA5011 guard)
	}
	if len(cer.Start) == 0 || !strings.Contains(cer.Start[0], "{sprint_id}") {
		t.Errorf("Start = %v", cer.Start)
	}
	if len(cer.DesignChecks) != 1 {
		t.Errorf("DesignChecks = %v", cer.DesignChecks)
	}
}

func TestGetSprintCeremony_MissingYAML(t *testing.T) {
	setupTempRoot(t)
	cer := GetSprintCeremony()
	if cer != nil && (len(cer.Start) > 0 || len(cer.Complete) > 0) {
		t.Errorf("missing yaml produced non-empty ceremony: %+v", cer)
	}
}

// ---------------------------------------------------------------------------
// GetSkillExtensions / GetSkillRules
// ---------------------------------------------------------------------------

func TestGetSkillExtensions_FromYAML(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	rules := GetSkillExtensions("code-review")
	if len(rules) != 2 {
		t.Fatalf("want 2 rules, got %v", rules)
	}
	if !strings.Contains(rules[0], "go vet") {
		t.Errorf("rules[0] = %q", rules[0])
	}
}

func TestGetSkillExtensions_Unknown(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	if got := GetSkillExtensions("unknown-skill"); len(got) != 0 {
		t.Errorf("unknown skill = %v", got)
	}
}

func TestGetSkillExtensions_EmptyName(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	if got := GetSkillExtensions(""); len(got) != 0 {
		t.Errorf("empty name = %v", got)
	}
}

// ---------------------------------------------------------------------------
// GetReminders (template-first fallback, so empty even without yaml)
// ---------------------------------------------------------------------------

func TestGetReminders_EmptyEventName(t *testing.T) {
	setupTempRoot(t)
	if got := GetReminders(""); len(got) != 0 {
		t.Errorf("empty event = %v", got)
	}
}

func TestGetReminders_UnknownEvent(t *testing.T) {
	setupTempRoot(t)
	if got := GetReminders("bogus.event"); len(got) != 0 {
		t.Errorf("unknown event = %v", got)
	}
}

// ---------------------------------------------------------------------------
// GetProjectConfigSummary
// ---------------------------------------------------------------------------

func TestGetProjectConfigSummary_MissingReturnsNil(t *testing.T) {
	setupTempRoot(t)
	if got := GetProjectConfigSummary(); got != nil {
		t.Errorf("missing yaml: want nil, got %v", got)
	}
}

func TestGetProjectConfigSummary_Minimal(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureMinimal)

	summary := GetProjectConfigSummary()
	if summary == nil {
		t.Fatal("summary nil")
	}
	if summary["exists"] != true {
		t.Errorf("exists = %v", summary["exists"])
	}
	if summary["format"] != "yaml" {
		t.Errorf("format = %v", summary["format"])
	}
}

func TestGetProjectConfigSummary_FullV2(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, v2FixtureFull)

	summary := GetProjectConfigSummary()
	if summary == nil {
		t.Fatal("summary nil")
	}
	if summary["platform_kind"] != "go" {
		t.Errorf("platform_kind = %v", summary["platform_kind"])
	}
}
