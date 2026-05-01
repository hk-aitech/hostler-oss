// Task type input alias normalization plus enum validation.
//
// Background: when alias values such as `fix`, `bug fix`, or `doc` flow into
// the Task frontmatter `type:` field, they drift away from the canonical
// values (`bugfix`, `docs`). We replicate the status canonical pattern from
// pkg/task/status.go and normalize the value at the CLI entry point.
//
// Measurement (full grep over works/, 2026-04-23): one Task body used
// `type: fix`. Adding this normalizer prevents the same drift from recurring.
package cmd

import (
	"fmt"
	"strings"
)

// validTaskTypes holds the allowed canonical type values. SSOT - 1:1 with the
// template_key suffixes (`task:feature`, `task:bugfix`, ...) in
// cli/pkg/db/schemas/harness_defaults.json.
var validTaskTypes = map[string]struct{}{
	"feature":  {},
	"bugfix":   {},
	"docs":     {},
	"refactor": {},
	"infra":    {},
	"test":     {},
	"chore":    {},
	"spike":    {},
	"hotfix":   {},
}

// typeAliases maps non-canonical inputs to canonical values. Captures observed
// drift plus common confusable variants. Missing entries fall through to the
// canonical set as-is.
var typeAliases = map[string]string{
	"fix":           "bugfix", // observed drift
	"bug":           "bugfix",
	"bug-fix":       "bugfix",
	"doc":           "docs",
	"documentation": "docs",
	"feat":          "feature",
	"test":          "test",
	"tests":         "test",
	"testing":       "test",
}

// normalizeType converts a type input string to the canonical form.
// Trims whitespace, lowercases, collapses underscores/spaces to hyphens (type
// has no internal hyphens), applies alias mapping, and validates the enum.
// Empty input is allowed (the flag is optional).
//
// Accepted: "bugfix", "Bugfix", "fix", "bug fix", "doc" -> "bugfix" / "docs" etc.
// Rejected: "support", "p0" -> error
func normalizeType(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	lower := strings.ToLower(trimmed)
	// Remove whitespace and underscores (type is a single hyphen-free token).
	collapsed := strings.NewReplacer(" ", "-", "_", "-").Replace(lower)
	// Apply alias mapping first.
	if mapped, ok := typeAliases[collapsed]; ok {
		return mapped, nil
	}
	if _, ok := validTaskTypes[collapsed]; !ok {
		return "", fmt.Errorf(
			"type value %q is not allowed - must be one of feature|bugfix|docs|refactor|infra|test|chore|spike|hotfix",
			raw,
		)
	}
	return collapsed, nil
}
