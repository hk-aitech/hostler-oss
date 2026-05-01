// Package hooks — pre-commit hook missing-warning.
//
// Background: when the hstl plugin is installed in an external project,
// missing `.git/hooks/pre-commit` means HMAC validation rules never run
// and the Sprint ceremony gate ( sprint_hmac signature) cannot block
// commits. In other words, a key quality gate is silently disabled —
// observed in a downstream project on 2026-04-17.
//
// This package checks hook installation **once per session** in CLI
// commands routed through mustInitDB and prints a warning to stderr.
// Installation is performed separately via `hstl hooks install` (we do
// not auto-install — never modify .git/hooks/ without user consent).
//
// Disable: HSTL_HOOK_WARNING=off.
package hooks

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
)

// hooksLogger — : switched the hook warning from direct
// stderr printing to going through the Logger.
var hooksLogger = pkglog.NewDefault()

// Env var / file path constants.
const (
	// HookWarningEnvVar — when unset the default action (warn) runs.
	// "off" disables it.
	HookWarningEnvVar = "HSTL_HOOK_WARNING"
	hookWarningOffValue = "off"
	// gitHooksRelativePath is the pre-commit hook path relative to
	// projectRoot.
	gitHooksRelativePath = ".git/hooks/pre-commit"
	// gitDirName is the git directory name (used to detect non-git
	// projects).
	gitDirName = ".git"
)

// warnOnce gates the warning to one print per session. sync.Once gives
// concurrency safety and dedup.
var warnOnce sync.Once

// CheckAndWarnOnce checks for the projectRoot's pre-commit hook and, when
// it is missing, prints a warning to stderr **once per session**.
//
// Behavior:
// 1. If HSTL_HOOK_WARNING=off, return immediately (skip).
// 2. If projectRoot/.git is missing → not a git project → skip.
// 3. If projectRoot/.git/hooks/pre-commit is missing, warn once.
// 4. Otherwise return silently.
//
// If projectRoot is empty, resolve it from HSTL_PROJECT_ROOT or cwd.
func CheckAndWarnOnce(projectRoot string) {
	if envalias.Lookup("HOOK_WARNING") == hookWarningOffValue {
		return
	}

	root := resolveProjectRoot(projectRoot)
	if root == "" {
		return
	}

	// Skip non-git projects.
	if _, err := os.Stat(filepath.Join(root, gitDirName)); err != nil {
		return
	}

	hookPath := filepath.Join(root, gitHooksRelativePath)
	if _, err := os.Stat(hookPath); err == nil {
		return // Installed — return silently.
	}

	warnOnce.Do(func() {
		hooksLogger.Warn("pre-commit hook not installed — HMAC validation rules disabled (Sprint ceremony gate bypassed)",
			"recover", "run `hstl hooks install`, then retry git commit",
			"disable", "HSTL_HOOK_WARNING=off (not recommended)")
	})
}

// resolveProjectRoot resolves projectRoot when it is empty by looking up
// HSTL_PROJECT_ROOT, then cwd. Returns "" if neither is available.
func resolveProjectRoot(projectRoot string) string {
	if projectRoot != "" {
		return projectRoot
	}
	if v := envalias.Lookup("PROJECT_ROOT"); v != "" {
		return v
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

// resetWarnOnceForTest is a test-only helper that isolates sync.Once
// between tests. Never call from production code.
func resetWarnOnceForTest() {
	warnOnce = sync.Once{}
}
