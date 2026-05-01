// multiworktree.go — multi-worktree sprint ID conflict check.
// Background: when multiple git worktrees of the same project exist
// (e.g. several feature branches running in parallel), each worktree
// may independently call `hstl sprint create --id sprint-NN`, which can
// race on the same ID. This module fs-scans every worktree's sprint
// folder at sprint-creation time, computes the occupied IDs, and
// suggests the next available ID on conflict.
// Behaviour:
// opt-out: when env HSTL_SPRINT_ID_CONFLICT_CHECK=off, the check is
// skipped.
// graceful degradation: if `git worktree list` fails, emit a
// stderr WARN and continue normally.
// --force flag handling is the caller's responsibility (cmd/sprint.go).
package sprint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
)

// IDConflictCheckEnv — opt-out env var name. envalias auto-prepends the
// HSTL_ prefix, so the actual variable is HSTL_SPRINT_ID_CONFLICT_CHECK.
// "off" disables the check.
const IDConflictCheckEnv = "SPRINT_ID_CONFLICT_CHECK"

// sprintIDPattern matches sprint-N (N is one or more digits).
var sprintIDPattern = regexp.MustCompile(`^sprint-(\d+)$`)

// CheckMultiWorktreeIDConflict scans every worktree's sprint folder
// and returns a ConflictError if sprintID is already occupied. When
// occupied, RecoveryHint includes the next available ID.
// opt-out: env HSTL_SPRINT_ID_CONFLICT_CHECK=off -> returns nil (skip).
// Graceful: if `git worktree list` fails, emits one stderr WARN line
// and returns nil.
func CheckMultiWorktreeIDConflict(sprintID string) error {
	if strings.EqualFold(strings.TrimSpace(envalias.Lookup(IDConflictCheckEnv)), "off") {
		return nil
	}

	worktrees, err := listWorktreePaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN multi-worktree ID conflict check skipped: %v\n", err)
		return nil
	}

	occupied := scanOccupiedSprintIDs(worktrees)
	if owner, ok := occupied[sprintID]; ok {
		next := nextAvailableSprintID(occupied, sprintID)
		return &apperr.RejectedError{
			EntityID: sprintID,
			Reason: fmt.Sprintf(
				"%s is already occupied in worktree %s.",
				sprintID, owner,
			),
			RecoveryHint: fmt.Sprintf(
				"next available ID: %s. Or use --force to override, or set HSTL_SPRINT_ID_CONFLICT_CHECK=off.",
				next,
			),
		}
	}
	return nil
}

// listWorktreePaths extracts the worktree paths from
// `git worktree list --porcelain` output.
func listWorktreePaths() ([]string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w", err)
	}

	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, strings.TrimPrefix(line, "worktree "))
		}
	}
	return paths, nil
}

// scanOccupiedSprintIDs walks each worktree's
// works/sprints/{backlog,active,completed,discarded}/sprint-*
// directories and builds the set of occupied sprint IDs. The value is
// the owning worktree path (used for debugging / messages).
func scanOccupiedSprintIDs(worktreePaths []string) map[string]string {
	occupied := map[string]string{}
	for _, wt := range worktreePaths {
		for _, loc := range []string{"backlog", "active", "completed", "discarded"} {
			locDir := filepath.Join(wt, "works", "sprints", loc)
			entries, err := os.ReadDir(locDir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				name := e.Name()
				if sprintIDPattern.MatchString(name) {
					if _, exists := occupied[name]; !exists {
						occupied[name] = wt
					}
				}
			}
		}
	}
	return occupied
}

// nextAvailableSprintID returns the first unoccupied sprint ID starting
// from baseID's number + 1. If baseID is not in sprint-N form, uses
// max(occupied) + 1 as the starting point.
func nextAvailableSprintID(occupied map[string]string, baseID string) string {
	start := 1
	if m := sprintIDPattern.FindStringSubmatch(baseID); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			start = n + 1
		}
	}

	maxN := start
	for id := range occupied {
		m := sprintIDPattern.FindStringSubmatch(id)
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n >= maxN {
			maxN = n + 1
		}
	}

	candidates := make([]int, 0, maxN-start+1)
	for n := start; n <= maxN; n++ {
		candidates = append(candidates, n)
	}
	sort.Ints(candidates)
	for _, n := range candidates {
		id := fmt.Sprintf("sprint-%d", n)
		if _, taken := occupied[id]; !taken {
			return id
		}
	}
	return fmt.Sprintf("sprint-%d", maxN)
}
