// Package fileutil — FindRepoRoot .
//
// A layout-agnostic common utility for detecting the repo root.
//
// Background: discovered that cmd/gen-schema/main.go's
// findRepoRoot only recognized the legacy layout (`cli/go.mod`) and failed
// (a sleeping bug) under the new monorepo layout
// (`packages/hostler-cli/go.mod`). Centralizing the logic prevents other
// tools from reintroducing similar hard-coded paths.
package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// cliCandidatePaths is the list of candidate CLI-binary package paths
// under the repo root.
// Supports both the monorepo and legacy layouts. To add a new layout,
// register it here (SSOT).
var cliCandidatePaths = []string{
	filepath.Join("packages", "hostler-cli"), // monorepo 
	"cli", // legacy layout
}

// maxAncestorsDefault is the default maximum search depth for
// FindRepoRoot. Generous upper bound for typical project structures.
const maxAncestorsDefault = 16

// FindRepoRoot walks upward from start and returns the directory
// containing a CLI-binary package (packages/hostler-cli/go.mod or
// cli/go.mod) as the repo root.
//
// Returns an error if the limit is exceeded. Use FindRepoRootMax to
// adjust the limit.
func FindRepoRoot(start string) (string, error) {
	return FindRepoRootMax(start, maxAncestorsDefault)
}

// FindRepoRootMax runs FindRepoRoot with a custom maxAncestors limit.
// Use this for test isolation or deeply nested environments.
func FindRepoRootMax(start string, maxAncestors int) (string, error) {
	if start == "" {
		return "", fmt.Errorf("start path is empty")
	}
	dir := start
	for range maxAncestors {
		// Candidate 1: each cliCandidatePaths under dir.
		for _, candidate := range cliCandidatePaths {
			goMod := filepath.Join(dir, candidate, "go.mod")
			if _, err := os.Stat(goMod); err == nil {
				return dir, nil
			}
		}
		// Candidate 2: dir itself is the CLI directory — has a go.mod and
		// the basename matches.
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			base := filepath.Base(dir)
			parent := filepath.Dir(dir)
			// packages/hostler-cli layout: basename=hostler-cli, parent
			// basename=packages.
			if base == "hostler-cli" && filepath.Base(parent) == "packages" {
				return filepath.Dir(parent), nil
			}
			// legacy cli/ layout
			if base == "cli" {
				return parent, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found (no packages/hostler-cli/go.mod or cli/go.mod, start=%s)", start)
}
