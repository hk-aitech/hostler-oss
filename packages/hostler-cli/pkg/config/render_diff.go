// Package config — render_diff.go: diff renderer for `config fix --dry-run`.
//
// When the Migration Runner was removed in , the RenderDiff
// helper from migrate_cli.go was split out into this file for `config fix`.
package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// diffContextLines is the number of context lines in the unified diff.
const diffContextLines = 3

// RenderDiff returns a unified diff between two YAML byte buffers.
// Falls back to a line-count summary when the system `diff -u` is missing.
func RenderDiff(original, migrated []byte) string {
	if string(original) == string(migrated) {
		return "(no change)\n"
	}
	result, err := renderUnifiedDiff(original, migrated)
	if err != nil {
		return renderFallbackSummary(original, migrated)
	}
	return result
}

// renderUnifiedDiff produces a unified diff using the system `diff -u`.
func renderUnifiedDiff(original, migrated []byte) (string, error) {
	tmpOrig, err := os.CreateTemp("", "hstl-diff-orig-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpOrig.Name()) }()

	tmpMig, err := os.CreateTemp("", "hstl-diff-mig-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpMig.Name()) }()

	if _, werr := tmpOrig.Write(original); werr != nil {
		return "", werr
	}
	_ = tmpOrig.Close()
	if _, werr := tmpMig.Write(migrated); werr != nil {
		return "", werr
	}
	_ = tmpMig.Close()

	cmd := exec.Command("diff",
		fmt.Sprintf("-U%d", diffContextLines),
		"--label", "original",
		"--label", "migrated",
		tmpOrig.Name(),
		tmpMig.Name(),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return string(out), nil
		}
		return "", fmt.Errorf("run diff: %w", err)
	}
	return "(no change)\n", nil
}

// renderFallbackSummary returns a line-count summary when the diff binary is unavailable.
func renderFallbackSummary(original, migrated []byte) string {
	origLines := strings.Count(string(original), "\n")
	migLines := strings.Count(string(migrated), "\n")
	delta := migLines - origLines
	sign := "+"
	if delta < 0 {
		sign = ""
	}
	return fmt.Sprintf("original %d lines -> converted %d lines (%s%d lines)\n(unified diff requires the 'diff' binary)\n",
		origLines, migLines, sign, delta)
}
