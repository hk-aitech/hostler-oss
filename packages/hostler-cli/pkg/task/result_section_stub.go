// Package task — auto-insert the Result section stub.
package task

import (
	"os"
	"strings"
)

// resultSectionHeading is the heading text of the `## Result` section.
const resultSectionHeading = "## Result"

// historyHeading is the heading text of the `## Status change history`
// section. It is the anchor used when inserting the stub.
const historyHeading = "## Status change history"

// resultSectionStub is the auto-inserted stub used at task-complete time
// when no `## Result` section exists. It follows the sub-heading
// convention.
// Brace placeholders are not recognised as paths by isLikelyFilePath, so
// they do not produce false positives in verifyTaskResultsForType.
var resultSectionStub = "## Result\n\n" +
	"> Write this section at task-complete time. The three required\n" +
	"> sub-headings (Design Decisions / Artifacts / Verification) must be\n" +
	"> present. The parser only scans for file paths under `### Artifacts`;\n" +
	"> backticked paths inside `### Design Decisions` are treated as\n" +
	"> narrative and excluded.\n\n" +
	"### Design Decisions\n\n" +
	"{Describe the main design choices and trade-offs. Backticked file\n" +
	"names are allowed — the parser treats them as narrative and excludes\n" +
	"them.}\n\n" +
	"### Artifacts\n\n" +
	"- {full file path} (created/modified/deleted)\n\n" +
	"### Verification\n\n" +
	"- Build: {result}\n" +
	"- Tests: {result}\n" +
	"- Other checks: {command + result}\n\n" +
	"### Commits\n\n" +
	"- {commit SHA}: {message}\n"

// StripT578Blockquote removes the legacy guidance blockquote from the
// Result section once the user has filled in their own content. The
// blockquote was auto-added at stub-insertion time but becomes
// unnecessary as soon as actual content is written. Removes the
// contiguous blockquote that begins with "> Write this section". Other
// user-authored blockquotes are preserved.
// Returns: (stripped, err) — stripped indicates whether anything was
// removed.
func StripT578Blockquote(absPath string) (stripped bool, err error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}
	content := string(raw)
	lines := strings.Split(content, "\n")

	out := make([]string, 0, len(lines))
	inGuidanceBlock := false
	found := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inGuidanceBlock {
			// Detect the guidance blockquote.
			if strings.HasPrefix(trimmed, "> Write this section at task-complete time") {
				inGuidanceBlock = true
				found = true
				continue
			}
			out = append(out, line)
		} else {
			// Skip subsequent blockquote lines.
			if strings.HasPrefix(trimmed, ">") {
				continue
			}
			// Also skip the blank line directly after the blockquote
			// (cleans up nested formatting).
			if trimmed == "" {
				inGuidanceBlock = false
				continue
			}
			// On the first non-blockquote line, end the block and keep
			// that line.
			inGuidanceBlock = false
			out = append(out, line)
		}
	}
	if !found {
		return false, nil
	}
	if werr := os.WriteFile(absPath, []byte(strings.Join(out, "\n")), 0o644); werr != nil {
		return false, werr
	}
	return true, nil
}

// hasResultSection reports whether the file contains a `## Result`
// section. Returns true if a line matches "## Result" or starts with
// "## Result " (with a trailing space).
func hasResultSection(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == resultSectionHeading ||
			strings.HasPrefix(trimmed, resultSectionHeading+" ") {
			return true
		}
	}
	return false
}

// EnsureResultSectionStub auto-inserts the Result-section stub when
// absPath has no `## Result` section. No-op when the section is already
// present.
// When `## Status change history` exists, the stub is inserted just
// before it; otherwise the stub is appended to the end of the file.
// Returns:
// stubbed: true when a stub was actually inserted.
// err: file read/write error.
func EnsureResultSectionStub(absPath string) (stubbed bool, err error) {
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}
	content := string(raw)
	if hasResultSection(content) {
		return false, nil
	}

	lines := strings.Split(content, "\n")

	// Locate the `## Status change history` line.
	insertIdx := len(lines)
	for i, line := range lines {
		if strings.TrimSpace(line) == historyHeading {
			insertIdx = i
			break
		}
	}

	// Build the stub line list (with a leading blank line guaranteed).
	stubLines := strings.Split(resultSectionStub, "\n")

	// Ensure a blank line precedes the insertion point.
	prefix := lines[:insertIdx]
	if len(prefix) > 0 && strings.TrimSpace(prefix[len(prefix)-1]) != "" {
		prefix = append(prefix, "")
	}

	result := make([]string, 0, len(lines)+len(stubLines)+2)
	result = append(result, prefix...)
	result = append(result, stubLines...)
	result = append(result, lines[insertIdx:]...)

	newContent := strings.Join(result, "\n")
	if werr := os.WriteFile(absPath, []byte(newContent), 0o644); werr != nil {
		return false, werr
	}
	return true, nil
}
