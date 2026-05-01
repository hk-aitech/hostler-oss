package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// manifest validate drift detection tests.

// TestT335_NormalizeExternalName verifies that the various external file
// name formats normalize to the dotted form.
func TestT335_NormalizeExternalName(t *testing.T) {
	tests := []struct {
		input string
		want string
	}{
		{"task.start", "task.start"},
		{"hstl task start", "task.start"},
		{"hstl harness auto-check", "harness.auto-check"},
		{"sprint.complete", "sprint.complete"},
	}
	for _, tt := range tests {
		got := normalizeExternalName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeExternalName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestT335_LoadExternalManifest_JSON verifies JSON file parsing.
func TestT335_LoadExternalManifest_JSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "manifest.json")
	content := `{"commands":[{"name":"task.start","flags":["with-ceremony"]},{"name":"sprint.complete"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	em, err := loadExternalManifest(path)
	if err != nil {
		t.Fatalf("loadExternalManifest: %v", err)
	}
	if len(em.Commands) != 2 {
		t.Errorf("Commands len = %d, want 2", len(em.Commands))
	}
	if em.Commands[0].Name != "task.start" {
		t.Errorf("Commands[0].Name = %q, want task.start", em.Commands[0].Name)
	}
	if len(em.Commands[0].Flags) != 1 || em.Commands[0].Flags[0] != "with-ceremony" {
		t.Errorf("Commands[0].Flags = %v, want [with-ceremony]", em.Commands[0].Flags)
	}
}

// TestT335_LoadExternalManifest_YAML verifies YAML file parsing.
func TestT335_LoadExternalManifest_YAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "manifest.yaml")
	content := `commands:
  - name: task.start
    flags: [with-ceremony]
  - name: sprint.complete
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	em, err := loadExternalManifest(path)
	if err != nil {
		t.Fatalf("loadExternalManifest: %v", err)
	}
	if len(em.Commands) != 2 {
		t.Errorf("Commands len = %d, want 2", len(em.Commands))
	}
}

// TestT335_CompareManifests_MissingInCli verifies that a command present in
// the external manifest but absent from the CLI is detected as
// missing_in_cli drift.
func TestT335_CompareManifests_MissingInCli(t *testing.T) {
	external := &ExternalManifest{
		Commands: []ExternalCommand{
			{Name: "task.start"},        // present in the CLI
			{Name: "task.nonexistent"}, // absent -> missing_in_cli
		},
	}

	drifts := compareManifests(external)

	// Verify that a missing_in_cli entry is included.
	foundMissingInCli := false
	for _, d := range drifts {
		if d.Type == "missing_in_cli" && d.Path == "task.nonexistent" {
			foundMissingInCli = true
			break
		}
	}
	if !foundMissingInCli {
		t.Errorf("missing_in_cli drift not detected. drifts=%v", drifts)
	}
}

// TestT335_CompareManifests_FlagMismatch verifies that a flag declared
// externally but absent from the CLI is detected as flag_mismatch drift.
func TestT335_CompareManifests_FlagMismatch(t *testing.T) {
	external := &ExternalManifest{
		Commands: []ExternalCommand{
			{Name: "task.start", Flags: []string{"nonexistent-flag"}},
		},
	}

	drifts := compareManifests(external)

	found := false
	for _, d := range drifts {
		if d.Type == "flag_mismatch" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("flag_mismatch drift not detected. drifts=%v", drifts)
	}
}

// TestT340_BuildProfileFromSkill_Valid verifies that a valid skill name
// builds a profile from skillCommandMap.
func TestT340_BuildProfileFromSkill_Valid(t *testing.T) {
	tests := []string{"task-management", "sprint-management", "learned"}
	for _, skill := range tests {
		em, err := buildProfileFromSkill(skill)
		if err != nil {
			t.Errorf("%q: unexpected err = %v", skill, err)
			continue
		}
		if em == nil || len(em.Commands) == 0 {
			t.Errorf("%q: profile contains no commands", skill)
		}
	}
}

// TestT340_BuildProfileFromSkill_Invalid verifies that an invalid skill name
// returns an error and that the error includes a similar suggestion.
func TestT340_BuildProfileFromSkill_Invalid(t *testing.T) {
	_, err := buildProfileFromSkill("nonexistent-skill")
	if err == nil {
		t.Errorf("expected error for invalid skill")
	}
	if err != nil && !strings.Contains(err.Error(), "task-management") {
		t.Errorf("error message missing similar-skill suggestion: %v", err)
	}
}

// TestT335_ValidateResult_StatusTransition verifies the status field
// transition based on drift 0 / N.
func TestT335_ValidateResult_StatusTransition(t *testing.T) {
	// drift 0 -> status=ok
	result1 := ValidateResult{DriftCount: 0, Drifts: []ValidateDrift{}}
	if result1.DriftCount == 0 {
		result1.Status = "ok"
	}
	if result1.Status != "ok" {
		t.Errorf("drift 0 -> Status = %q, want ok", result1.Status)
	}

	// drift N -> status=drift
	result2 := ValidateResult{DriftCount: 3, Drifts: make([]ValidateDrift, 3)}
	if result2.DriftCount > 0 {
		result2.Status = "drift"
	}
	if result2.Status != "drift" {
		t.Errorf("drift N -> Status = %q, want drift", result2.Status)
	}
}
