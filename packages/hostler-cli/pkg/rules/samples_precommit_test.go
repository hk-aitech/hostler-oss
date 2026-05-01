package rules

import (
	"os/exec"
	"strings"
	"testing"
)

// T590 (Sprint-66) — full regression for the G5 pre-commit Rule.

// TestT590_AllPrecommitRulesRegistered — verify precommit Rules are
// registered in the Registry.
// T551 (Sprint-82): precommit.sprint.hmac added → 12.
// T753 (Sprint-88): precommit.branch.sync added → 13.
// T220 (Sprint-14): precommit.artifacts.untracked added → 14.
// T335 (Sprint-25): precommit.manifest.external.drift added → 15.
// T327 (Sprint-25): precommit.cli.json_stdout added → 16.
// T459 (Sprint-41): precommit.trac.validate added → 17.
// T468 (Sprint-44): precommit.task.frontmatter_only_edit added → 18.
// T578 (Sprint-56): precommit.skill.oss_identity added → 19.
// T663 (Sprint-73): precommit.sprint.cross_track_deps added → 20.
// T525 (Sprint-77): precommit.kb.reference-format added → 21.
// T526 (Sprint-77): precommit.archive.mirror added → 22.
// T588 (Sprint-77): precommit.uuid.frontmatter added → 23.
func TestT590_AllPrecommitRulesRegistered(t *testing.T) {
	want := 24 // T760 sprint-90 (precommit.skill.size)
	got := 0
	for _, r := range List() {
		if r.Category() == "precommit" {
			got++
		}
	}
	if got != want {
		t.Errorf("precommit Rule count = %d, want %d", got, want)
	}
}

// TestT590_PrecommitRuleIDs — each Rule ID is unique and uses the defined
// prefix.
func TestT590_PrecommitRuleIDs(t *testing.T) {
	expected := map[string]bool{
		"precommit.go.vet":                      true,
		"precommit.go.staticcheck":              true,
		"precommit.go.test_short":               true,
		"precommit.manifest.drift":              true,
		"precommit.skill_trigger.drift":         true,
		"precommit.claudemd.audit":              true,
		"precommit.version.sync":                true,
		"precommit.manifest.skill_command_sync": true,
		"precommit.harness.skill_drift":         true,
		"precommit.sprint_phase.drift":          true,
		"precommit.task.hmac":                   true,
		"precommit.sprint.hmac":                 true, // T551 Sprint-82
		"precommit.branch.sync":                 true, // T753 Sprint-88
		"precommit.artifacts.untracked":         true, // T220 Sprint-14
		"precommit.manifest.external.drift":     true, // T335 Sprint-25
		"precommit.cli.json_stdout":             true, // T327 Sprint-25
		"precommit.trac.validate":               true, // T459 Sprint-41
		"precommit.task.frontmatter_only_edit":  true, // T468 Sprint-44
		"precommit.skill.oss_identity":          true, // T578 Sprint-56
		"precommit.sprint.cross_track_deps":     true, // T663 Sprint-73
		"precommit.kb.reference-format":         true, // T525 Sprint-77
		"precommit.archive.mirror":              true, // T526 Sprint-77
		"precommit.uuid.frontmatter":            true, // T588 Sprint-77
		"precommit.skill.size":                  true, // T760 sprint-90
	}
	for id := range expected {
		if _, ok := Get(id); !ok {
			t.Errorf("Rule %q not registered", id)
		}
	}
}

// TestT590_PrecommitRule_ProjectRoot_Missing_skip — when ProjectRoot is empty
// the result is Skipped.
func TestT590_PrecommitRule_ProjectRoot_Missing_skip(t *testing.T) {
	r, ok := Get("precommit.go.vet")
	if !ok {
		t.Skip("Rule not registered")
	}
	res := r.Check(&RuleContext{})
	if res.Status != StatusSkipped {
		t.Errorf("status=%v, want Skipped", res.Status)
	}
}

// TestT590_PrecommitRule_NonGitDir_skip — when the root has no .git, the
// result is Skipped.
func TestT590_PrecommitRule_NonGitDir_skip(t *testing.T) {
	tmp := t.TempDir()
	r, _ := Get("precommit.go.vet")
	res := r.Check(&RuleContext{ProjectRoot: tmp})
	if res.Status != StatusSkipped {
		t.Errorf("status=%v, want Skipped (no .git)", res.Status)
	}
}

// TestT598_PrecommitRule_AllBlock — T598 (Sprint-67) promoted severity from
// advisory → block. The Rule Engine is the enforcement entrypoint.
func TestT598_PrecommitRule_AllBlock(t *testing.T) {
	for _, r := range List() {
		if r.Category() != "precommit" {
			continue
		}
		if r.DefaultSeverity() != SeverityBlock {
			t.Errorf("Rule %s severity = %v, want block (T598 enforcement promotion)", r.ID(), r.DefaultSeverity())
		}
	}
}

// TestT598_PrecommitRule_ShouldRun_Conditional — verify each Rule runs
// conditionally based on staged files. Uses stub staged files.
func TestT598_PrecommitRule_ShouldRun_Conditional(t *testing.T) {
	cases := map[string]struct {
		staged  []string
		wantRun bool
	}{
		"go file changed":     {[]string{"cli/pkg/foo/bar.go"}, true},
		"only skill files":    {[]string{"skills/foo/SKILL.md"}, false},
		"only docs changed":   {[]string{"docs/readme.md"}, false},
	}
	r, _ := Get("precommit.go.vet")
	// Test go vet Rule's shouldRun directly — precommitWrapperRule's
	// internal precommitCheck field is not externally accessible, so check
	// hasGoFiles directly.
	_ = r
	for name, tc := range cases {
		got := hasGoFiles(tc.staged)
		if got != tc.wantRun {
			t.Errorf("%s: hasGoFiles=%v, want %v", name, got, tc.wantRun)
		}
	}
}

// TestT616_ManifestDrift_CliPkg_Trigger — regression test.
// When Go files under cli/pkg/ change, the precommit.manifest.drift rule
// must run. hasGoFiles watches the entire cli/ prefix, which includes
// cli/pkg/.
func TestT616_ManifestDrift_CliPkg_Trigger(t *testing.T) {
	cases := []struct {
		name    string
		staged  []string
		wantRun bool
	}{
		{"cli/pkg Go file changed", []string{"cli/pkg/task/task.go"}, true},
		{"cli/pkg nested package", []string{"cli/pkg/db/db.go"}, true},
		{"cli/cmd Go file changed", []string{"cli/cmd/hstl-oss/cmd/task.go"}, true},
		{"cli/pkg JSON file only (not Go)", []string{"cli/pkg/db/schemas/harness_defaults.json"}, false},
		{"only skills files", []string{"skills/task-management/SKILL.md"}, false},
	}
	for _, tc := range cases {
		got := hasGoFiles(tc.staged)
		if got != tc.wantRun {
			t.Errorf("[%s] hasGoFiles=%v, want %v (manifest.drift shouldRun)", tc.name, got, tc.wantRun)
		}
	}
}

// TestT788_ManifestDrift_AutoregenEnvVar — T788 (Sprint-92): auto-regen
// option. precommit.manifest.drift's command must include the
// HSTL_PRECOMMIT_MANIFEST_AUTOREGEN env-var check, and when it is unset
// the existing BLOCK behavior must be preserved.
func TestT788_ManifestDrift_AutoregenEnvVar(t *testing.T) {
	// The Rule's command lives inline as a raw string — verify structurally.
	var driftRule *precommitCheck
	for i := range precommitChecks {
		if precommitChecks[i].id == "precommit.manifest.drift" {
			driftRule = &precommitChecks[i]
			break
		}
	}
	if driftRule == nil {
		t.Fatal("precommit.manifest.drift rule not registered")
		return // nolint (SA5011 guard)
	}
	cmdStr := strings.Join(driftRule.command, " ")

	// env var check must be present.
	if !strings.Contains(cmdStr, "HSTL_PRECOMMIT_MANIFEST_AUTOREGEN") {
		t.Errorf("autoregen env var check missing: %s", cmdStr)
	}
	// auto-regen path: make generate-manifest + git add.
	if !strings.Contains(cmdStr, "generate-manifest") {
		t.Errorf("regenerate command missing: %s", cmdStr)
	}
	if !strings.Contains(cmdStr, "git add docs/generated/manifest.json") {
		t.Errorf("git add missing — staged file not added after regen: %s", cmdStr)
	}
	// Default off: compares against the unset value "0".
	if !strings.Contains(cmdStr, `:-0`) {
		t.Errorf("default-off (:-0) pattern missing: %s", cmdStr)
	}
	// On no drift, succeed immediately — keeps check-manifest as the leading step.
	if !strings.Contains(cmdStr, "check-manifest") {
		t.Errorf("leading check-manifest missing: %s", cmdStr)
	}
	// On regen failure or env off, signal BLOCK.
	if !strings.Contains(cmdStr, "exit 1") {
		t.Errorf("BLOCK signal (exit 1) missing: %s", cmdStr)
	}
}

// TestT139_PythonScriptMissing_Skip — T139 (Sprint-07 cleanup) regression.
// precommit.claudemd.audit / precommit.skill_trigger.drift had a bug where
// missing scripts caused `test -f X && Y` to exit 1 (block); fixed to use
// `[ ! -f X ] || Y`. This test confirms the new guard pattern is in the
// command string. The plugin's home repo still runs the script while
// downstream consumers without the script (e.g. external monorepos) skip
// safely.
func TestT139_PythonScriptMissing_Skip(t *testing.T) {
	cases := []struct {
		id         string
		scriptPath string
	}{
		{"precommit.claudemd.audit", "skills/claude-md-audit/scripts/audit.py"},
		{"precommit.skill_trigger.drift", "scripts/generate-skill-triggers.py"},
	}
	for _, tc := range cases {
		var check *precommitCheck
		for i := range precommitChecks {
			if precommitChecks[i].id == tc.id {
				check = &precommitChecks[i]
				break
			}
		}
		if check == nil {
			t.Errorf("rule %s not registered", tc.id)
			continue
		}
		cmdStr := strings.Join(check.command, " ")
		// Old block-causing pattern must not be present.
		if strings.Contains(cmdStr, "test -f "+tc.scriptPath+" && python3") {
			t.Errorf("[%s] old pattern still present (test -f && python3) — would block on missing: %s", tc.id, cmdStr)
		}
		// New pattern must be present.
		if !strings.Contains(cmdStr, "[ ! -f "+tc.scriptPath+" ]") {
			t.Errorf("[%s] new guard pattern ([ ! -f X ]) missing: %s", tc.id, cmdStr)
		}
		// python3 invocation must be retained (still runs in the home repo).
		if !strings.Contains(cmdStr, "python3 "+tc.scriptPath) {
			t.Errorf("[%s] python3 call missing — must not skip in home repo: %s", tc.id, cmdStr)
		}
	}
}

// TestT598_StagedFilesFromDiff — extract paths from unified diff text.
func TestT598_StagedFilesFromDiff(t *testing.T) {
	diff := "diff --git a/cli/foo.go b/cli/foo.go\n--- a/cli/foo.go\n+++ b/cli/foo.go\n@@ -1 +1 @@\n-old\n+new\n" +
		"diff --git a/skills/x/SKILL.md b/skills/x/SKILL.md\n--- a/skills/x/SKILL.md\n+++ b/skills/x/SKILL.md\n@@ -1 +1 @@\n-a\n+b\n"
	got := stagedFilesFromDiff(diff)
	if len(got) != 2 {
		t.Errorf("file count = %d, want 2: %v", len(got), got)
	}
	want := []string{"cli/foo.go", "skills/x/SKILL.md"}
	for i, w := range want {
		if i >= len(got) || got[i] != w {
			t.Errorf("got[%d]=%q, want %q", i, got[i], w)
		}
	}
}

// TestISS20260419_006_claudemd_audit_ExternalProject_skip — hotfix.
// Confirms the rule skips (exit 0) when audit.py is missing.
func TestISS20260419_006_claudemd_audit_ExternalProject_skip(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp) // simulate environment without audit.py

	var cmdArgs []string
	for _, c := range precommitChecks {
		if c.id == "precommit.claudemd.audit" {
			cmdArgs = c.command
			break
		}
	}
	if len(cmdArgs) == 0 {
		t.Fatal("precommit.claudemd.audit not registered")
	}

	out, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
	if err != nil {
		t.Errorf("external project (no audit.py) was BLOCKED: %v, out=%s", err, out)
	}
}

// TestISS20260419_006_version_sync_ExternalProject_skip — skip when
// scripts/check-version-sync.sh is missing.
func TestISS20260419_006_version_sync_ExternalProject_skip(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	var cmdArgs []string
	for _, c := range precommitChecks {
		if c.id == "precommit.version.sync" {
			cmdArgs = c.command
			break
		}
	}
	if len(cmdArgs) == 0 {
		t.Fatal("precommit.version.sync not registered")
	}
	out, err := exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
	if err != nil {
		t.Errorf("external project (no check-version-sync.sh) was BLOCKED: %v, out=%s", err, out)
	}
}
