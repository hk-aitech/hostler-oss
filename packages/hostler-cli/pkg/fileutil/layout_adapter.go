// Package fileutil — LayoutResolver adapter.
//
// ADR-001 §4.1 Phase C. Wraps the existing fileutil globals in a struct so
// they satisfy the ports.LayoutResolver interface. In Phase D, the
// service layer accepts this adapter via its constructor and routes calls
// through it as the fs implementation.
package fileutil

import "github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"

// FSLayout exposes the fileutil globals as a port.
// Stateless — obtain via `NewFSLayout()` or use the zero value `FSLayout{}`.
type FSLayout struct{}

// NewFSLayout constructs the default fs-based LayoutResolver.
func NewFSLayout() *FSLayout { return &FSLayout{} }

// ProjectRoot delegates to the GetProjectRoot global.
func (FSLayout) ProjectRoot() string { return GetProjectRoot() }

// MainRepoRoot delegates to the GetMainRepoRoot global.
func (FSLayout) MainRepoRoot() string { return GetMainRepoRoot() }

// RepoRelative delegates to the ToRepoRelative global.
func (FSLayout) RepoRelative(absPath string) string { return ToRepoRelative(absPath) }

// SprintDir delegates to the FindSprintDir global.
func (FSLayout) SprintDir(sprintID string) (string, string, error) {
	return FindSprintDir(sprintID)
}

// UpdateCurrentFocus delegates to the UpdateCurrentFocus global. After
// removing the direct pkg/fileutil call from cmd, all access funnels
// through the app layer.
func (FSLayout) UpdateCurrentFocus(activeSprintID, activeSprintTitle, status string) error {
	return UpdateCurrentFocus(activeSprintID, activeSprintTitle, status)
}

// Compile-time check — FSLayout satisfies the ports.LayoutResolver contract.
var _ ports.LayoutResolver = (*FSLayout)(nil)
