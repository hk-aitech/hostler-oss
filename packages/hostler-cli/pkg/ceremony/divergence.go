// Package ceremony — early divergence warning at sprint start.
// During an earlier Sprint, dev was 34 commits ahead of origin/main
// at sprint-start time and the first sprint-branch commit hit
// precommit.branch.sync's 30-commit threshold and was BLOCKED. The
// AI was forced to discover the cause and merge dev -> main first
// because there had been no briefing at the start of the Sprint.
// This module reads the divergence at sprint-start time and, when it
// reaches the WARN threshold (default 25), adds a WARN entry to the
// readiness checks so the situation is surfaced early. The BLOCK
// threshold (30) remains the responsibility of
// precommit.branch.sync.
package ceremony

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
)

// DivergenceWarnEnv is the WARN-threshold env var name (commits).
const DivergenceWarnEnv = "HSTL_SPRINT_START_DIVERGENCE_WARN_COMMITS"

// divergenceWarnDefault is the default WARN threshold — set 5
// commits below precommit.branch.sync's BLOCK threshold (30) to
// leave a buffer.
const divergenceWarnDefault = 25

// resolveDivergenceWarn determines the WARN threshold.
// Priority: env var > project-config.yaml > default.
// 1. HSTL_SPRINT_START_DIVERGENCE_WARN_COMMITS env, if a positive
// integer.
// 2. project-config.yaml `sprint_start.divergence_warn_commits`, if a
// positive integer.
// 3. The built-in default (25).
func resolveDivergenceWarn() int {
	raw := strings.TrimSpace(envalias.Lookup("SPRINT_START_DIVERGENCE_WARN_COMMITS"))
	if raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
		// Invalid env value -> fall back to config / default
		// (silently ignore the invalid value).
	}
	// project-config.yaml lookup.
	if cfg, err := config.LoadProjectConfig(); err == nil && cfg != nil {
		if ssc := cfg.GetSprintStartConfig(); ssc != nil && ssc.DivergenceWarnCommits != nil && *ssc.DivergenceWarnCommits > 0 {
			return *ssc.DivergenceWarnCommits
		}
	}
	return divergenceWarnDefault
}

// countCommitsAheadOfMain returns the commit count for
// origin/main..HEAD. Returns ok=false on git failure so the caller
// can silently skip.
// Required conditions:
// The directory is a local git repo.
// origin/main exists.
// Other cases (non-git, missing remote, etc.) silent-skip.
func countCommitsAheadOfMain() (count int, ok bool) {
	// Hard-coded "origin/main" — fixed against the main branch.
	// Projects using a different main-branch name can add an env-var
	// extension later.
	cmd := exec.Command("git", "rev-list", "--count", "origin/main..HEAD")
	out, err := cmd.Output()
	if err != nil {
		return 0, false
	}
	s := strings.TrimSpace(string(out))
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// collectMainDivergenceCheck performs the WARN check on commit count
// against origin/main. Below the WARN threshold, or when git lookup
// fails, returns pass (or nil). The BLOCK threshold (30) is owned by
// precommit.branch.sync, so this function only handles WARN.
func collectMainDivergenceCheck() *ReadinessCheck {
	count, ok := countCommitsAheadOfMain()
	if !ok {
		return &ReadinessCheck{
			Check:  "T779 divergence vs main",
			Status: "info",
			Detail: "git rev-list unavailable (non-git or origin/main unset) — skip",
		}
	}
	warnThreshold := resolveDivergenceWarn()
	if count < warnThreshold {
		return &ReadinessCheck{
			Check:  "T779 divergence vs main",
			Status: "pass",
			Detail: fmt.Sprintf("%d commits ahead (threshold %d)", count, warnThreshold),
		}
	}
	return &ReadinessCheck{
		Check:  "T779 divergence vs main",
		Status: "warn",
		Detail: fmt.Sprintf("dev is %d commits ahead of origin/main — merge soon (precommit.branch.sync BLOCK threshold = 30)", count),
	}
}
