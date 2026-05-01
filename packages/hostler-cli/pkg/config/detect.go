// Package config — config-problem detection engine.
//
// This file is the shared engine for `hstl config check` and
// `hstl config fix`. User feedback (2026-04-12) consolidated the
// "check -> recommend -> apply" three-stage pattern under the single
// `hstl config` namespace. The category split must always exclude
// structural conversion (migration) from the auto-fix path.
//
// Design principles:
// - migration-category problems run only via `hstl config migrate`
// - normalize / deprecation / schema run automatically via `hstl config fix`
// - Detect is read-only — it does not modify files
package config

import (
	"fmt"
	"os"
	"strings"
)

// ProblemCategory is the category of a detected config problem.
//
// Core rule: `CategoryMigration` is never auto-handled by
// `ApplyConfigFixes`. The user must run `hstl config migrate --apply`
// explicitly (structural conversion is large-scale, no implicit execution).
type ProblemCategory string

const (
	// CategoryMigration — structural conversion required (e.g. v1 -> v2). Excluded from fix.
	CategoryMigration ProblemCategory = "migration"
	// CategoryDeprecation — uses a deprecated field. Eligible for fix.
	CategoryDeprecation ProblemCategory = "deprecation"
	// CategorySchema — schema violation (missing field, wrong enum). Fix-eligible when possible.
	CategorySchema ProblemCategory = "schema"
	// CategoryNormalize — normalisation needed (key order, whitespace). Eligible for fix.
	CategoryNormalize ProblemCategory = "normalize"
)

// ProblemSeverity is the severity of a problem.
type ProblemSeverity string

const (
	SeverityError ProblemSeverity = "error"
	SeverityWarn ProblemSeverity = "warn"
	SeverityInfo ProblemSeverity = "info"
)

// ConfigProblem describes a single detected problem.
type ConfigProblem struct {
	// Code is a programmatic identifier (e.g. "v1_schema", "canonical_drift").
	Code string `json:"code"`

	// Category is the problem classification (migration/deprecation/schema/normalize).
	Category ProblemCategory `json:"category"`

	// Severity is the severity level (error/warn/info).
	Severity ProblemSeverity `json:"severity"`

	// Path is the location inside the YAML (dot notation). Empty when the issue is file-wide.
	Path string `json:"path,omitempty"`

	// Message is a one-line summary intended for the user.
	Message string `json:"message"`

	// RecoveryHint explains how to resolve it.
	RecoveryHint string `json:"recovery_hint,omitempty"`

	// Commands lists the commands the user should run (recommendation).
	Commands []string `json:"commands,omitempty"`

	// AutoFixable indicates whether `hstl config fix` can resolve it.
	// CategoryMigration is always false.
	AutoFixable bool `json:"auto_fixable"`
}

// ProblemReport is the result set returned by Detect.
type ProblemReport struct {
	// YAMLPath is the absolute path of the inspected file. Empty when the file is missing.
	YAMLPath string `json:"yaml_path"`

	// Exists indicates whether the file exists.
	Exists bool `json:"exists"`

	// Problems is the list of detected problems.
	Problems []ConfigProblem `json:"problems"`
}

// HasProblems reports whether the report has any problems.
func (r *ProblemReport) HasProblems() bool {
	return len(r.Problems) > 0
}

// FixableProblems filters the problems eligible for `config fix`.
// CategoryMigration is auto-excluded because AutoFixable=false.
func (r *ProblemReport) FixableProblems() []ConfigProblem {
	out := make([]ConfigProblem, 0, len(r.Problems))
	for _, p := range r.Problems {
		if p.AutoFixable && p.Category != CategoryMigration {
			out = append(out, p)
		}
	}
	return out
}

// MigrationProblems filters problems in the migration category.
// They are recommended for `config migrate --apply`, not for `fix`.
func (r *ProblemReport) MigrationProblems() []ConfigProblem {
	out := make([]ConfigProblem, 0, len(r.Problems))
	for _, p := range r.Problems {
		if p.Category == CategoryMigration {
			out = append(out, p)
		}
	}
	return out
}

// DetectConfigProblems inspects the target YAML file and returns a ProblemReport.
//
// Detections supported in the current initial version:
// - v1_schema (migration) — `schema_version` field present in the file
// - canonical_drift (normalize) — file bytes differ from canonicalize result
// - schema_invalid (schema) — ValidateProjectConfigYAML returned an error
//
// Future extensions:
// - deprecated fields (those scheduled for deprecation after v2)
// - missing required fields
// - enum violations
func DetectConfigProblems(yamlPath string) (*ProblemReport, error) {
	report := &ProblemReport{
		YAMLPath: yamlPath,
	}
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return nil, fmt.Errorf("read file (%s): %w", yamlPath, err)
	}
	report.Exists = true

	// 1. schema_version detection — split categories based on whether the
	// top-level version field exists.
	//
	// (a) schema_version present + version absent -> pure v1 file
	// -> migration/v1_schema (no autofix, migrate-only)
	//
	// (b) schema_version present + version present -> v2 conversion
	// finished but legacy field lingered
	// -> deprecation/stale_schema_version (config fix target)
	//
	// Key point: once the structural conversion is complete, schema_version
	// is just a deprecated field and fix can remove it. By contrast, v1
	// files without version still need structural conversion and are
	// migrate-only.
	hasSchemaVersion := hasTopLevelKey(data, "schema_version")
	hasVersion := hasTopLevelKey(data, "version")

	switch {
	case hasSchemaVersion && !hasVersion:
		report.Problems = append(report.Problems, ConfigProblem{
			Code: "v1_schema",
			Category: CategoryMigration,
			Severity: SeverityWarn,
			Path: "schema_version",
			Message: "schema_version field in use — v2 structural conversion required",
			RecoveryHint: "Structural conversion only runs via `hstl config migrate`. " +
				"`hstl config fix` does not handle this issue.",
			Commands: []string{
				"hstl config migrate # preview (dry-run)",
				"hstl config migrate --apply # apply with .pre-v2.bak backup",
			},
			AutoFixable: false, // migration is never auto-fixed
		})
	case hasSchemaVersion && hasVersion:
		report.Problems = append(report.Problems, ConfigProblem{
			Code: "stale_schema_version",
			Category: CategoryDeprecation,
			Severity: SeverityInfo,
			Path: "schema_version",
			Message: "schema_version field still present after v2 conversion (deprecated)",
			RecoveryHint: "version is already set, so schema_version is an " +
				"unnecessary legacy field. `hstl config fix` can remove it.",
			Commands: []string{
				"hstl config fix --dry-run # preview the changes",
				"hstl config fix # apply (with confirmation prompt)",
			},
			AutoFixable: true,
		})
	}

	// 2. canonical_drift (normalize) — the file differs from the canonical
	// result, so it needs normalising. When canonicalize fails outright
	// (parse error) it is treated as a separate schema problem.
	canonical, canonErr := canonicalizeYAML(data)
	switch {
	case canonErr != nil:
		report.Problems = append(report.Problems, ConfigProblem{
			Code: "yaml_parse_error",
			Category: CategorySchema,
			Severity: SeverityError,
			Message: fmt.Sprintf("YAML parse failed: %s", canonErr.Error()),
			RecoveryHint: "The file is corrupt YAML. Restore from backup or fix it manually.",
			AutoFixable: false,
		})
	case string(canonical) != string(data):
		report.Problems = append(report.Problems, ConfigProblem{
			Code: "canonical_drift",
			Category: CategoryNormalize,
			Severity: SeverityInfo,
			Message: "file key order differs from the canonical form (recommend reordering)",
			RecoveryHint: "Use `hstl config fix` to restore the canonical form. " +
				"There are no structural changes; only the key order becomes alphabetical.",
			Commands: []string{
				"hstl config fix --dry-run # preview the changes",
				"hstl config fix # apply (with confirmation prompt)",
			},
			AutoFixable: true,
		})
	}

	// Unknown top-level key detection — prevents the
	// Migration Runner's additive-only limitation from recurring. When a user
	// accidentally introduces a top-level key not defined in the
	// ProjectConfig schema, emit an "unknown_top_level_key" warning.
	if unknownKeys := findUnknownTopLevelKeys(data); len(unknownKeys) > 0 {
		report.Problems = append(report.Problems, ConfigProblem{
			Code: "unknown_top_level_key",
			Category: CategorySchema,
			Severity: SeverityWarn,
			Message: fmt.Sprintf("undefined top-level keys (%d): %s",
				len(unknownKeys), strings.Join(unknownKeys, ", ")),
			RecoveryHint: "Field is not defined in the ProjectConfig schema. " +
				"Could be a leftover deprecated field or a typo — verify and remove it manually. " +
				"See docs/04-guides/configuration.md for the allowed key list.",
			AutoFixable: false, // auto-deletion risks data loss; require manual handling
		})
	}

	// Unknown nested key detection — extends the top-level guard.
	// Top-level checks miss typos like `platform.dotnet.bounded_context`, so
	// we complement them with a reflect-based recursive path comparison.
	if unknownNested, nestedErr := FindUnknownNestedKeys(data); nestedErr == nil && len(unknownNested) > 0 {
		report.Problems = append(report.Problems, ConfigProblem{
			Code: "unknown_nested_key",
			Category: CategorySchema,
			Severity: SeverityWarn,
			Message: fmt.Sprintf("undefined nested keys (%d): %s",
				len(unknownNested), strings.Join(unknownNested, ", ")),
			RecoveryHint: "Does not match a nested path in the ProjectConfig schema. " +
				"Could be a singular/plural typo (e.g. bounded_context vs. bounded_contexts) — " +
				"verify and fix it manually. Allowed paths follow the yaml tags in types.go.",
			AutoFixable: false,
		})
	}

	// 3. schema_invalid (schema) — JSON Schema validation failed.
	// canonicalize can succeed while the schema is still wrong, so we run
	// this check separately.
	if _, verr, vErr := ValidateProjectConfigYAML(data); vErr == nil && verr != nil {
		report.Problems = append(report.Problems, ConfigProblem{
			Code: "schema_invalid",
			Category: CategorySchema,
			Severity: SeverityError,
			Path: verr.Path,
			Message: fmt.Sprintf("schema validation failed: %s", verr.Message),
			RecoveryHint: verr.RecoveryHint + " Auto-fix is not possible — fix the file manually.",
			AutoFixable: false,
		})
	}

	return report, nil
}

// knownTopLevelKeys is the whitelist of top-level YAML keys defined in ProjectConfig.
// Used by the unknown_top_level_key detector.
//
// This list must stay aligned with the yaml tags on the types.go ProjectConfig
// struct. New fields must be added here as well; TestT407_UnknownTopLevelKey
// enforces this empirically.
var knownTopLevelKeys = map[string]bool{
	"version": true,
	"extends": true,
	"project": true,
	"platform": true,
	"deployment_groups": true,
	"tasks": true,
	"sprints": true,
	"ufc": true,
	"skills": true,
	"briefing": true,
	// precommit rule thresholds + sprint start ceremony settings.
	"precommit": true,
	"sprint_start": true,
}

// findUnknownTopLevelKeys collects the top-level keys in the YAML file that
// are absent from knownTopLevelKeys. Duplicates are removed
// while preserving first-seen order.
func findUnknownTopLevelKeys(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	var unknown []string
	seen := make(map[string]bool)
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		// top-level (no indentation)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}
		// extract "key:"
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if !knownTopLevelKeys[key] {
			unknown = append(unknown, key)
		}
	}
	return unknown
}

// hasTopLevelKey reports whether the YAML data contains the given top-level
// key, using a single scan. Matches only lines whose pattern is `key:`
// without comments or indentation, case-sensitive.
//
// Avoids yaml parsing in this minimal implementation (v1 files parse fine,
// but only key presence is needed and a line-based scan is more robust).
func hasTopLevelKey(data []byte, key string) bool {
	prefix := key + ":"
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		// skip comments
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		// only unindented lines count as top-level
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}
