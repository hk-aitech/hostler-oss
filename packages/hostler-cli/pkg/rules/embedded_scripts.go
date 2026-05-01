package rules

import (
	_ "embed"
	"fmt"
	"os"
)

// ISS-20260419-001 hotfix — bundles bash scripts invoked from rule actions
// into the hstl binary via go:embed.
//
// Background: precommitCheck.command referenced external scripts by relative
// path (e.g. `["bash", "scripts/check-XXX.sh"]`), but the rule execution
// workdir is the user's project root, so outside the plugin repo (in
// downstream projects) the script was missing and the precommit hook
// failed wholesale with exit 127 (command not found).
//
// Fix: move the scripts into cli/pkg/rules/scripts/ (within the go:embed
// scope) and at runtime extract them via os.CreateTemp before passing to
// bash. The workdir stays at the user's project (cmd.Dir = ProjectRoot)
// so git commands keep working.

//go:embed scripts/check-branch-sync.sh
var embeddedCheckBranchSync string

//go:embed scripts/check-version-sync.sh
var embeddedCheckVersionSync string

//go:embed scripts/check-manifest-sync.sh
var embeddedCheckManifestSync string

//go:embed scripts/check-harness-skill-drift.sh
var embeddedCheckHarnessSkillDrift string

//go:embed scripts/check-sprint-phase-drift.sh
var embeddedCheckSprintPhaseDrift string

// embeddedRuleScripts maps a rule command's relative-path key to the embedded
// script body. precommitWrapperRule.Check() looks up command[1] in this map
// and, on hit, writes the body to a temp file and executes it
// (ISS-20260419-001).
var embeddedRuleScripts = map[string]string{
	"scripts/check-branch-sync.sh":         embeddedCheckBranchSync,
	"scripts/check-version-sync.sh":        embeddedCheckVersionSync,
	"scripts/check-manifest-sync.sh":       embeddedCheckManifestSync,
	"scripts/check-harness-skill-drift.sh": embeddedCheckHarnessSkillDrift,
	"scripts/check-sprint-phase-drift.sh":  embeddedCheckSprintPhaseDrift,
}

// embeddedScriptMode is the temp-script file permission; executable for bash.
const embeddedScriptMode os.FileMode = 0o700

// embeddedScriptTempPattern is the os.CreateTemp pattern; * is replaced with
// a unique identifier.
const embeddedScriptTempPattern = "hstl-rule-script-*.sh"

// writeEmbeddedScript extracts the embedded script body to a temp file and
// returns its path. The caller is responsible for cleaning it up via
// defer os.Remove. If scriptKey is not in the embed map, returns "" + error.
func writeEmbeddedScript(scriptKey string) (string, error) {
	body, ok := embeddedRuleScripts[scriptKey]
	if !ok {
		return "", fmt.Errorf("embedded script not found: %s", scriptKey)
	}
	f, err := os.CreateTemp("", embeddedScriptTempPattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	path := f.Name()
	if _, err := f.WriteString(body); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to write embedded script: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Chmod(path, embeddedScriptMode); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to set temp file permission: %w", err)
	}
	return path, nil
}
