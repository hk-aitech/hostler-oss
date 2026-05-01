// Package sprint — Sprint Tier runtime check.
// Runtime entry point that calls the CheckTier helper at sprint-start
// time. Collects task estimates by scanning files directly to avoid an
// app.TaskList dependency.
// SSOT: docs/08-references/standards/sprint-tier-spec.md §6
package sprint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
)

// LoadSprintEstimates collects estimates for the tasks bound to the
// sprint.
// Parses the estimate field of the frontmatter in
// works/sprints/<state>/<sprintID>/tasks/T*.md. Tasks with unsupported
// estimates or missing frontmatter are ignored (treated as zero
// capacity).
// trac: HAR-CM003-CLI
func LoadSprintEstimates(sprintID string) []domain.Estimate {
	out, err := exec.Command("bash", "-c",
		fmt.Sprintf(`find works/sprints -name 'T*.md' -path '*%s/tasks/*' 2>/dev/null`, sprintID)).Output()
	if err != nil {
		return nil
	}
	paths := strings.Split(strings.TrimSpace(string(out)), "\n")
	estimates := make([]domain.Estimate, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		content, _ := os.ReadFile(filepath.Clean(p))
		// Extract the frontmatter estimate field (simple grep).
		for line := range strings.SplitSeq(string(content), "\n") {
			if rest, ok := strings.CutPrefix(line, "estimate:"); ok {
				val := strings.TrimSpace(rest)
				val = strings.Trim(val, `"'`)
				estimates = append(estimates, domain.Estimate(val))
				break
			}
			if line == "---" && len(estimates) > 0 {
				break // frontmatter end
			}
		}
	}
	return estimates
}

// RunTierCheck is the size_mode verification entry point at sprint
// start time.
// Runs even when sizeMode is empty or "solo" (the default).
// Collects estimates for the sprint's bound tasks -> calls
// CheckTier.
// On violation, returns the TierCheckResult so the caller can
// BLOCK and emit recovery_hint output.
// `override` is converted into TierDefaults from SprintSizeOverride and
// passed in. A nil override is treated as an empty TierDefaults (all
// defaults).
func RunTierCheck(sprintID string, sizeMode string, override TierDefaults) TierCheckResult {
	tier, _ := ParseTier(sizeMode)
	estimates := LoadSprintEstimates(sprintID)
	return CheckTier(tier, estimates, override)
}

// FormatTierViolationHint builds the user-facing message for a
// TierCheckResult violation.
// Standard recovery_hint patterns:
// sprint create --force or hstl sprint start --force.
// Increase .hstl-oss/project-config.yaml's sprints.size_mode.
// Move some tasks to the next sprint.
func FormatTierViolationHint(r TierCheckResult) string {
	return fmt.Sprintf(
		"Sprint Tier violation (%s): %s — fix: (a) --force override / (b) raise sprints.size_mode (current %s) / (c) split tasks",
		r.Tier, r.Reason(), r.Tier)
}

// ResolveCompleteMode resolves the sprint:complete ceremony mode by
// priority.
// Priority (highest to lowest):
// 1. CLI flag --lite (when cliLite=true, returns "lite" immediately).
// 2. env HSTL_SPRINT_COMPLETE_MODE.
// 3. config sprints.complete_mode.
// 4. default "confirm".
// Valid values: "lite" | "confirm". Unknown values fall back to
// "confirm" (safe default).
//
// envValue is supplied by the caller (typically via envalias.Lookup).
func ResolveCompleteMode(cliLite bool, envValue, configValue string) string {
	if cliLite {
		return "lite"
	}
	for _, v := range []string{envValue, configValue} {
		switch v {
		case "lite":
			return "lite"
		case "confirm":
			return "confirm"
		}
	}
	return "confirm"
}
