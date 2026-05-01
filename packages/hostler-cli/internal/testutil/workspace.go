// Package testutil provides shared test helpers.
//
// Many tests need an isolated workspace: a temp directory containing the
// project's hidden config directory (default brand.ProjectDirName) plus an
// optional project-config.yaml. NewIsolatedWorkspace wraps that boilerplate
// with an Options pattern.
//
// Usage:
//
//	ws := testutil.NewIsolatedWorkspace(t)
//	ws := testutil.NewIsolatedWorkspace(t, testutil.WithProjectConfig("project:\n  key: test\n"))
//	ws := testutil.NewIsolatedWorkspace(t, testutil.WithHostlerSubdirs("audit", "logs"))
//	ws.WriteFile("foo.txt", "bar")
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// Workspace represents an isolated test workspace.
// Root is a t.TempDir()-based temporary directory; ConfigDir is the
// absolute path of the brand-specific configuration directory.
type Workspace struct {
	t         *testing.T
	Root      string
	ConfigDir string
}

// Option is an optional setting for NewIsolatedWorkspace.
type Option func(*workspaceOptions)

type workspaceOptions struct {
	configDirName  string   // defaults to brand.ProjectDirName
	projectConfig  string   // project-config.yaml content; "" skips file creation
	hostlerSubdirs []string // additional subdirectories to create under ConfigDir
}

// WithConfigDirName overrides the hidden config-directory name (default
// brand.ProjectDirName). Use sparingly — most tests should rely on the
// brand-derived default.
func WithConfigDirName(name string) Option {
	return func(o *workspaceOptions) {
		o.configDirName = name
	}
}

// WithProjectConfig specifies the contents of project-config.yaml.
// An empty string skips file creation (the default).
func WithProjectConfig(content string) Option {
	return func(o *workspaceOptions) {
		o.projectConfig = content
	}
}

// WithHostlerSubdirs names extra subdirectories to create under ConfigDir.
// Example: WithHostlerSubdirs("audit", "mailbox/inbox", "templates/reminders").
func WithHostlerSubdirs(names ...string) Option {
	return func(o *workspaceOptions) {
		o.hostlerSubdirs = append(o.hostlerSubdirs, names...)
	}
}

// NewIsolatedWorkspace constructs an isolated test workspace.
//   - root created via t.TempDir().
//   - ConfigDir (.hostler by default) created with MkdirAll.
//   - project-config.yaml and extra subdirs created based on options.
//
// On failure it stops the test immediately via t.Fatalf. The returned
// Workspace exposes Root / ConfigDir paths and helper methods to the
// caller.
func NewIsolatedWorkspace(t *testing.T, opts ...Option) *Workspace {
	t.Helper()

	o := &workspaceOptions{
		configDirName: brand.ProjectDirName,
	}
	for _, opt := range opts {
		opt(o)
	}

	root := t.TempDir()
	configDir := filepath.Join(root, o.configDirName)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	for _, sub := range o.hostlerSubdirs {
		full := filepath.Join(configDir, sub)
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatalf("failed to create subdir %s: %v", sub, err)
		}
	}

	if o.projectConfig != "" {
		path := filepath.Join(configDir, "project-config.yaml")
		if err := os.WriteFile(path, []byte(o.projectConfig), 0o644); err != nil {
			t.Fatalf("failed to write project-config.yaml: %v", err)
		}
	}

	return &Workspace{
		t:         t,
		Root:      root,
		ConfigDir: configDir,
	}
}

// WriteFile writes a file at a workspace-root-relative path.
// Parent directories are created automatically.
// Calls t.Fatalf on failure.
func (w *Workspace) WriteFile(relPath, content string) string {
	w.t.Helper()
	full := filepath.Join(w.Root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		w.t.Fatalf("failed to create parent directory (%s): %v", full, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		w.t.Fatalf("failed to write file (%s): %v", full, err)
	}
	return full
}

// Path returns the absolute path of a workspace-root-relative path.
func (w *Workspace) Path(relPath string) string {
	return filepath.Join(w.Root, relPath)
}
