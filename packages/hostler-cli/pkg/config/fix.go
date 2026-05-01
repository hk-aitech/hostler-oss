// Package config — automatic config-repair engine.
//
// Auto-handles results from DetectConfigProblems where `AutoFixable=true`
// and `Category != migration`. The CLI caller may pre-filter the migration
// category, but this function blocks it again as a defensive double check.
package config

import (
	"fmt"
	"os"
	"strings"
)

// hasProblemCode reports whether the problem list contains the given code.
func hasProblemCode(problems []ConfigProblem, code string) bool {
	for _, p := range problems {
		if p.Code == code {
			return true
		}
	}
	return false
}

// stripStaleSchemaVersion removes the top-level `schema_version: N` line
// from the yaml bytes. Head comments and other fields are preserved.
//
// Discovered while dogfooding the plugin itself. When legacy
// schema_version lingers after a v2 conversion, canonicalize alone cannot
// remove it, so a separate pre-pass is required.
//
// Implementation: drops only top-level (unindented) lines that start with
// `schema_version:`. Comment lines and nested keys are untouched.
func stripStaleSchemaVersion(data []byte) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		// Only unindented lines count as top-level keys.
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			out = append(out, line)
			continue
		}
		// Drop top-level lines beginning with "schema_version:".
		if strings.HasPrefix(line, "schema_version:") {
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n")), nil
}

// FixResult is the result returned by ApplyConfigFixes.
type FixResult struct {
	// YAMLPath is the absolute path of the file being fixed.
	YAMLPath string `json:"yaml_path"`

	// Applied indicates whether the file was actually modified. Always false when DryRun=true.
	Applied bool `json:"applied"`

	// DryRun indicates whether the call used dryRun=true.
	DryRun bool `json:"dry_run"`

	// FixedProblems lists the problems that were processed (also in dry-run as "would fix").
	FixedProblems []ConfigProblem `json:"fixed_problems"`

	// SkippedMigrations lists problems skipped because of the migration
	// category. The CLI caller uses this to advise the user to handle
	// migrations with a separate command.
	SkippedMigrations []ConfigProblem `json:"skipped_migrations"`

	// OriginalBytes is the file content before the fix (for diff output).
	OriginalBytes []byte `json:"-"`

	// FixedBytes is the file content after the fix (also returned in dry-run as the "would-be" bytes).
	FixedBytes []byte `json:"-"`
}

// ApplyConfigFixes auto-repairs the issues found by DetectConfigProblems.
//
// When dryRun=true, the file is not modified and only the "what would be
// written" bytes are returned in FixedBytes. When dryRun=false, the file is
// overwritten in place.
//
// Core design rules:
//   - CategoryMigration problems are never processed (defensive double check).
//     Even when the caller already filtered them, they are returned via
//     SkippedMigrations.
//   - Currently supported fixes: CategoryNormalize / canonical_drift only.
//     The fix rewrites the file with the canonicalize output.
//   - Multiple problems can coexist; canonicalize resolves them all in a
//     single write, so no multi-pass is needed.
func ApplyConfigFixes(report *ProblemReport, dryRun bool) (*FixResult, error) {
	if report == nil {
		return nil, fmt.Errorf("report is nil")
	}
	if !report.Exists {
		return &FixResult{YAMLPath: report.YAMLPath, DryRun: dryRun}, nil
	}

	result := &FixResult{
		YAMLPath: report.YAMLPath,
		DryRun:   dryRun,
	}

	// migration is double-blocked defensively — safe even when the caller did not filter.
	for _, p := range report.Problems {
		if p.Category == CategoryMigration {
			result.SkippedMigrations = append(result.SkippedMigrations, p)
		}
	}

	// Actual fix targets — AutoFixable && not migration.
	fixable := report.FixableProblems()
	if len(fixable) == 0 {
		return result, nil
	}

	data, err := os.ReadFile(report.YAMLPath)
	if err != nil {
		return nil, fmt.Errorf("read file (%s): %w", report.YAMLPath, err)
	}
	result.OriginalBytes = data

	// Fix pipeline (ordering matters):
	//   1. remove stale_schema_version (deprecation) — only when present
	//   2. canonicalize (normalize) — always (key order)
	//
	// Why multiple stages: stale_schema_version is field deletion, so it
	// must run before canonicalize. canonicalize itself only sorts keys
	// without changing structure.
	fixed := data
	if hasProblemCode(fixable, "stale_schema_version") {
		stripped, strErr := stripStaleSchemaVersion(fixed)
		if strErr != nil {
			return nil, fmt.Errorf("strip stale_schema_version: %w", strErr)
		}
		fixed = stripped
	}

	// canonicalizeYAMLChecked enforces a key-preservation sanity check.
	// Guards against silent key removal — when a leaf key is missing, the
	// fix itself fails.
	canonical, canonErr := canonicalizeYAMLChecked(fixed)
	if canonErr != nil {
		return nil, fmt.Errorf("canonicalize: %w", canonErr)
	}
	result.FixedBytes = canonical

	// Record the fixed_problems list.
	result.FixedProblems = append(result.FixedProblems, fixable...)

	// In dry-run, return without modifying the file.
	if dryRun {
		return result, nil
	}

	// Actually apply the fix.
	if err := os.WriteFile(report.YAMLPath, canonical, 0o644); err != nil {
		return nil, fmt.Errorf("write file (%s): %w", report.YAMLPath, err)
	}
	result.Applied = true
	return result, nil
}
