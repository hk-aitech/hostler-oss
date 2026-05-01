// Package config — T306 DetectConfigProblems / ApplyConfigFixes tests.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestT307_Detect_StaleSchemaVersion verifies that when version is
// already present yet schema_version still lingers (found via T307 plugin
// dogfooding), a deprecation-category problem is detected with
// AutoFixable=true.
func TestT307_Detect_StaleSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	// File where the legacy schema_version remains stale after the v2 conversion.
	content := `version: 2.0.0
project:
  key: testproj
platform:
  kind: go
schema_version: 1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	var stale *ConfigProblem
	for i, p := range report.Problems {
		if p.Code == "stale_schema_version" {
			stale = &report.Problems[i]
			break
		}
	}
	if stale == nil {
		t.Fatalf("stale_schema_version should be detected: %+v", report.Problems)
	}
	if stale.Category != CategoryDeprecation {
		t.Errorf("stale_schema_version must be CategoryDeprecation: got=%s", stale.Category)
	}
	if !stale.AutoFixable {
		t.Errorf("stale_schema_version must be AutoFixable=true (fix target)")
	}

	// Must not be misclassified as v1_schema.
	for _, p := range report.Problems {
		if p.Code == "v1_schema" {
			t.Errorf("file with version must not be classified as v1_schema: %+v", p)
		}
	}

	// Verify that fix actually removes stale_schema_version.
	result, err := ApplyConfigFixes(report, false)
	if err != nil {
		t.Fatalf("ApplyConfigFixes failed: %v", err)
	}
	after, _ := os.ReadFile(path)
	if hasTopLevelKey(after, "schema_version") {
		t.Errorf("schema_version still present after fix:\n%s", string(after))
	}
	if !hasTopLevelKey(after, "version") {
		t.Errorf("fix must not remove version:\n%s", string(after))
	}
	if !result.Applied {
		t.Errorf("expected Applied=true")
	}
}

// TestT306_Detect_V1Schema verifies that when schema_version is present,
// a migration-category problem is detected with AutoFixable=false.
func TestT306_Detect_V1Schema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	content := `schema_version: 1
project:
  key: testproj
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if !report.Exists {
		t.Fatalf("expected Exists=true")
	}

	var v1 *ConfigProblem
	for i, p := range report.Problems {
		if p.Code == "v1_schema" {
			v1 = &report.Problems[i]
			break
		}
	}
	if v1 == nil {
		t.Fatalf("v1_schema should be detected: %+v", report.Problems)
	}
	if v1.Category != CategoryMigration {
		t.Errorf("v1_schema must be CategoryMigration: got=%s", v1.Category)
	}
	if v1.AutoFixable {
		t.Errorf("v1_schema must be AutoFixable=false (no migration auto-fix)")
	}
	if len(v1.Commands) == 0 {
		t.Errorf("v1_schema must include recommended commands")
	}
	// Recommended commands must include 'hstl config migrate'.
	found := false
	for _, c := range v1.Commands {
		if strings.Contains(c, "hstl config migrate") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("v1_schema commands must include 'hstl config migrate': %v", v1.Commands)
	}
}

// TestT306_Detect_V2Clean verifies that a fully v2 file does not produce a
// v1_schema problem.
func TestT306_Detect_V2Clean(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	// Canonical (alphabetical) order — guarantees no canonical_drift.
	content := `version: 2.0.0
platform:
  kind: go
project:
  key: testproj
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	for _, p := range report.Problems {
		if p.Code == "v1_schema" {
			t.Errorf("v2 file must not detect v1_schema: %+v", p)
		}
	}
}

// TestT306_Detect_CanonicalDrift verifies that a file with unsorted keys
// produces a normalize-category problem.
func TestT306_Detect_CanonicalDrift(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	// project comes before platform (alphabetical: version -> platform -> project).
	// Current order: version -> project -> platform -> drift detected.
	content := `version: 2.0.0
project:
  key: testproj
platform:
  kind: go
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	var drift *ConfigProblem
	for i, p := range report.Problems {
		if p.Code == "canonical_drift" {
			drift = &report.Problems[i]
			break
		}
	}
	if drift == nil {
		t.Fatalf("canonical_drift should be detected: %+v", report.Problems)
	}
	if drift.Category != CategoryNormalize {
		t.Errorf("canonical_drift must be CategoryNormalize: got=%s", drift.Category)
	}
	if !drift.AutoFixable {
		t.Errorf("canonical_drift must be AutoFixable=true")
	}
}

// TestT306_Detect_FileMissing verifies that, when the file is missing,
// Detect returns an empty report with Exists=false and no error.
func TestT306_Detect_FileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.yaml")

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if report.Exists {
		t.Errorf("expected Exists=false: got=%v", report.Exists)
	}
	if len(report.Problems) != 0 {
		t.Errorf("missing file: expected zero problems, got=%+v", report.Problems)
	}
}

// TestT306_Fix_ExcludesMigration verifies that ApplyConfigFixes never
// auto-handles migration-category problems and instead returns them in
// SkippedMigrations. This is the core design guarantee.
func TestT306_Fix_ExcludesMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	// v1 file (migration problem)
	content := `schema_version: 1
project:
  key: testproj
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// There must be a migration problem.
	if len(report.MigrationProblems()) == 0 {
		t.Fatalf("migration problem should be detected")
	}

	// Apply (dryRun=false)
	result, err := ApplyConfigFixes(report, false)
	if err != nil {
		t.Fatalf("ApplyConfigFixes failed: %v", err)
	}

	// migration must land in SkippedMigrations.
	if len(result.SkippedMigrations) == 0 {
		t.Errorf("migration problem should appear in SkippedMigrations")
	}

	// FixedProblems must not include any migration entries.
	for _, p := range result.FixedProblems {
		if p.Category == CategoryMigration {
			t.Errorf("FixedProblems must not contain migration category: %+v", p)
		}
	}

	// Even if the file changed, schema_version must still be present
	// (Bug guard: canonicalize must not turn v1 into v2).
	after, _ := os.ReadFile(path)
	if !strings.Contains(string(after), "schema_version") {
		t.Errorf("fix must not remove schema_version (structural conversion is migrate-only):\n%s", string(after))
	}
}

// TestT306_Fix_CanonicalDriftApplied verifies that canonical_drift is
// resolved by fix and the file is rewritten in canonical form.
func TestT306_Fix_CanonicalDriftApplied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	// v2 file with drift (unsorted keys)
	original := `version: 2.0.0
project:
  key: testproj
platform:
  kind: go
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if len(report.FixableProblems()) == 0 {
		t.Fatalf("there should be fixable problems")
	}

	result, err := ApplyConfigFixes(report, false)
	if err != nil {
		t.Fatalf("ApplyConfigFixes failed: %v", err)
	}
	if !result.Applied {
		t.Errorf("expected Applied=true")
	}

	// Re-detect: canonical_drift must be gone.
	report2, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("re-Detect failed: %v", err)
	}
	for _, p := range report2.Problems {
		if p.Code == "canonical_drift" {
			t.Errorf("canonical_drift still present after fix: %+v", p)
		}
	}
}

// TestT306_Fix_DryRunNoWrite verifies that, in dry-run mode, the file is
// not modified and only FixedBytes is returned.
func TestT306_Fix_DryRunNoWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	original := `version: 2.0.0
project:
  key: testproj
platform:
  kind: go
`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	result, err := ApplyConfigFixes(report, true)
	if err != nil {
		t.Fatalf("ApplyConfigFixes failed: %v", err)
	}
	if result.Applied {
		t.Errorf("dry-run must not set Applied=true")
	}
	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}
	// File contents must remain identical.
	after, _ := os.ReadFile(path)
	if string(after) != original {
		t.Errorf("dry-run modified the file:\nbefore:\n%s\nafter:\n%s", original, string(after))
	}
}

// TestT407_Detect_UnknownTopLevelKey verifies that a YAML using undefined
// top-level keys raises an unknown_top_level_key warning (Sprint-48).
func TestT407_Detect_UnknownTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	// Canonical order (alphabetical) — intentionally includes two unknown fields.
	content := `briefing: null
deployment_groups: []
extends: []
version: 2.0.0
mysterious_field: 42
platform:
  kind: go
project:
  key: testproj
rogue_section:
  foo: bar
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	var found *ConfigProblem
	for i := range report.Problems {
		if report.Problems[i].Code == "unknown_top_level_key" {
			found = &report.Problems[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("unknown_top_level_key not detected: %+v", report.Problems)
	}
	if !strings.Contains(found.Message, "mysterious_field") ||
		!strings.Contains(found.Message, "rogue_section") {
		t.Errorf("warning message must include both unknown keys: %s", found.Message)
	}
	if found.AutoFixable {
		t.Error("unknown_top_level_key must be manual-only (AutoFixable=false)")
	}
}

// TestT407_Detect_AllKnownKeys verifies that a clean file using only known
// top-level keys produces no unknown_top_level_key warning.
func TestT407_Detect_AllKnownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "project-config.yaml")
	content := `briefing: null
deployment_groups: []
extends: []
version: 2.0.0
platform:
  kind: go
project:
  key: testproj
skills: {}
sprints: {}
tasks: {}
ufc:
  policy: warn
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	report, err := DetectConfigProblems(path)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	for _, p := range report.Problems {
		if p.Code == "unknown_top_level_key" {
			t.Errorf("clean yaml produced false-positive unknown_top_level_key: %+v", p)
		}
	}
}

// TestT306_HasTopLevelKey directly exercises the top-level key scan helper.
func TestT306_HasTopLevelKey(t *testing.T) {
	cases := []struct {
		name string
		data string
		key  string
		want bool
	}{
		{"straight", "schema_version: 1\nfoo: bar\n", "schema_version", true},
		{"absent", "version: 2.0.0\n", "schema_version", false},
		{"indented ignored", "parent:\n  schema_version: 1\n", "schema_version", false},
		{"comment ignored", "# schema_version: 1\nfoo: bar\n", "schema_version", false},
		{"head comments then key", "# comment\n\nschema_version: 1\n", "schema_version", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasTopLevelKey([]byte(tc.data), tc.key)
			if got != tc.want {
				t.Errorf("hasTopLevelKey(%q, %q) = %v, want %v", tc.data, tc.key, got, tc.want)
			}
		})
	}
}

// =============================================================================
// T426 (Sprint-53) — FindUnknownNestedKeys
// =============================================================================

// invalidYAMLNestedTypo includes platform.dotnet.bounded_context (singular typo).
const invalidYAMLNestedTypo = `version: "2.0.0"
project:
  key: test
platform:
  kind: dotnet
  dotnet:
    solution_file: Test.slnx
    bounded_context:  # singular typo (correct: bounded_contexts)
      - id: COL
        name: Collection
`

// validYAMLNestedOk is a regression guard — proper nested paths must not be flagged unknown.
const validYAMLNestedOk = `version: "2.0.0"
project:
  key: test
platform:
  kind: dotnet
  dotnet:
    solution_file: Test.slnx
    bounded_contexts:
      - id: COL
        name: Collection
`

func TestT426_FindUnknownNestedKeys_DetectsSingularTypo(t *testing.T) {
	unknown, err := FindUnknownNestedKeys([]byte(invalidYAMLNestedTypo))
	if err != nil {
		t.Fatalf("FindUnknownNestedKeys error: %v", err)
	}
	found := false
	for _, p := range unknown {
		if p == "platform.dotnet.bounded_context" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("singular typo platform.dotnet.bounded_context not detected: %v", unknown)
	}
}

func TestT426_FindUnknownNestedKeys_CleanFile(t *testing.T) {
	unknown, err := FindUnknownNestedKeys([]byte(validYAMLNestedOk))
	if err != nil {
		t.Fatalf("FindUnknownNestedKeys error: %v", err)
	}
	// platform.dotnet.bounded_contexts must always pass.
	for _, p := range unknown {
		if p == "platform.dotnet.bounded_contexts" {
			t.Errorf("valid path was misclassified as unknown: %s", p)
		}
	}
}

// T445 — reflect-based automatic free-form detection.
// The free-form event keys under tasks.events / sprints.events
// (map[string][]string) must not be flagged as unknown. The previous
// hard-coded list omitted these paths; map-typed fields are auto-included
// from T445 onward.
func TestT445_FreeFormParent_AutoDetect_MapFields(t *testing.T) {
	yamlWithFreeFormEvents := `version: "2.0.0"
project:
  key: test
tasks:
  events:
    start:
      - "Reminder A"
    complete:
      - "Reminder B"
    custom_xyz:
      - "user-defined event"
sprints:
  events:
    start:
      - "Reminder C"
`
	unknown, err := FindUnknownNestedKeys([]byte(yamlWithFreeFormEvents))
	if err != nil {
		t.Fatalf("FindUnknownNestedKeys error: %v", err)
	}
	for _, p := range unknown {
		if strings.HasPrefix(p, "tasks.events.") || strings.HasPrefix(p, "sprints.events.") {
			t.Errorf("free-form map child misclassified as unknown: %s", p)
		}
	}
	// Confirm the internal collected set was populated.
	ensureAllowedNestedKeyPaths()
	if !freeFormNestedPaths["tasks.events"] {
		t.Errorf("tasks.events was not auto-registered in freeFormNestedPaths")
	}
	if !freeFormNestedPaths["sprints.events"] {
		t.Errorf("sprints.events was not auto-registered in freeFormNestedPaths")
	}
	if !freeFormNestedPaths["skills"] {
		t.Errorf("skills (map[string]*SkillConfig) was not auto-registered")
	}
}
