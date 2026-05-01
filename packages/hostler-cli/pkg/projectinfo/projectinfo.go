// Package projectinfo provides shared helpers used by hstl brief / context
// to collect project state.
//
// The two commands have different roles (brief is a rich user briefing
// while context produces an AI-compressed context), but the base data
// (git state, placeholder warnings, etc.) must come from a single source
// to avoid drift. This package is the single source of truth.
package projectinfo

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// Issue is the shared warning structure used by brief / context.
type Issue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// GitState is the git working-tree state summary.
type GitState struct {
	Branch      string `json:"branch"`
	Uncommitted int    `json:"uncommitted"`
	Ahead       int    `json:"ahead"`
	Behind      int    `json:"behind"`
}

// CollectGitState collects the git state for the current directory.
// On failure, Branch falls back to "detached".
func CollectGitState() GitState {
	g := GitState{}
	g.Branch = runGit("branch", "--show-current")
	if g.Branch == "" {
		g.Branch = "detached"
	}

	if porcelain := runGit("status", "--porcelain"); porcelain != "" {
		for _, line := range strings.Split(strings.TrimSpace(porcelain), "\n") {
			if strings.TrimSpace(line) != "" {
				g.Uncommitted++
			}
		}
	}

	if aheadStr := runGit("rev-list", "--count", "@{u}..HEAD"); aheadStr != "" {
		_, _ = fmt.Sscanf(aheadStr, "%d", &g.Ahead)
	}

	if behindStr := runGit("rev-list", "--count", "HEAD..@{u}"); behindStr != "" {
		_, _ = fmt.Sscanf(behindStr, "%d", &g.Behind)
	}

	return g
}

// runGit executes a git command and returns the trimmed stdout.
// Returns an empty string on failure.
func runGit(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CollectCoreIssues collects the core warnings shared by brief / context
// (Sprint completion readiness, residual placeholder bodies).
// activeSprintIDs is the already-known list of active sprint IDs to avoid
// duplicate queries.
func CollectCoreIssues(activeSprintIDs []string) []Issue {
	var issues []Issue

	// Sprint completion readiness — Total>0 and Done==Total.
	var sprintReady []string
	for _, sid := range activeSprintIDs {
		prog, err := sprint.AggregateProgress(sid)
		if err == nil && prog != nil && prog.Total > 0 && prog.Done == prog.Total {
			sprintReady = append(sprintReady, sid)
		}
	}
	if len(sprintReady) > 0 {
		issues = append(issues, Issue{
			Level:   "WARN",
			Message: fmt.Sprintf("Sprint %s has all Tasks done — run sprint:complete", strings.Join(sprintReady, ", ")),
		})
	}

	// Residual placeholder bodies.
	if placeholders := task.FindPlaceholderTasks(""); len(placeholders) > 0 {
		issues = append(issues, Issue{
			Level:   "WARN",
			Message: fmt.Sprintf("%d Tasks with residual placeholder bodies: %s", len(placeholders), strings.Join(placeholders, ", ")),
		})
	}

	return issues
}
