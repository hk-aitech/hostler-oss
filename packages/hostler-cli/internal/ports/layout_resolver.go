// Package ports — LayoutResolver port.
//
// Isolates works/ layout path computation behind a single interface so
// the CLI migration can route every file access through ports.
//
// Today the global functions in `pkg/fileutil/fileutil.go` provide the
// implementation. This port is an extracted declaration that restates
// the existing function signatures as an interface; the Service layer
// can accept a LayoutResolver in its constructor and invoke it from
// there.
//
// Verification pattern (a wrapper struct is added in
// `cli/pkg/fileutil/layout_adapter.go`):
//
//	var _ ports.LayoutResolver = (*fsLayout)(nil)
//
// A successful compile-time check proves that the fs implementation satisfies
// the LayoutResolver contract.
package ports

// LayoutResolver computes paths in the works/ directory layout.
// The foundation of every file access — a prerequisite for the CLI
// migration through ports.
//
// Implementation contract:
//   - ProjectRoot: prefer the HSTL_PROJECT_ROOT env var, then git toplevel,
//     finally fall back to cwd.
//   - RepoRelative: returns paths relative to the main repo even from worktrees
//     (matching the ToRepoRelative behaviour).
//   - SprintDir: searches active / backlog / completed in that order and
//     returns the first match. location is one of "active" | "backlog" |
//     "completed".
type LayoutResolver interface {
	// ProjectRoot returns the absolute path of the project root directory.
	ProjectRoot() string

	// MainRepoRoot returns the main repo path of the git worktree.
	// If not in a worktree, equals ProjectRoot().
	MainRepoRoot() string

	// RepoRelative converts an absolute path to a path relative to the main
	// repo. Used for DB storage so paths stay consistent regardless of
	// worktree / main repo location.
	RepoRelative(absPath string) string

	// SprintDir locates the Sprint directory for the given sprintID.
	// Returns: (absolute sprintDir path, location, error).
	SprintDir(sprintID string) (string, string, error)

	// UpdateCurrentFocus refreshes works/CURRENT-FOCUS.md whenever a Sprint
	// transitions state. Exposes the operation via LayoutResolver so cmd
	// no longer calls pkg/fileutil directly.
	//
	// Parameters:
	//
	//	activeSprintID: ID of the currently active Sprint (empty string if none)
	//	activeSprintTitle: title (empty string if unused)
	//	status: "sprint-active" | "sprint-completed" | "idle"
	//
	// No-op if the works/sprints/ directory does not exist (supports
	// projects that do not use sprints).
	UpdateCurrentFocus(activeSprintID, activeSprintTitle, status string) error
}
