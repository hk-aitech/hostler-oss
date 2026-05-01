package rules

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Phase 1's "function injection" pattern (delegating the check function
// externally) was replaced with dedicated Check implementations. pkg/rules
// still does not import pkg/task; it remains self-contained, performing
// validation purely through file reads and `git` invocations.
//
// The two Rules in this file are referenced by all three presets
// (strict/recommended/relaxed). They previously stayed Skipped because the
// checker function was nil; this migration makes them functional.

// ── constants ────────────────────────────────────────────────────────

const (
	// taskBodyMinChars is the minimum body length that the Placeholder
	// check trusts.
	taskBodyMinChars = 200
	// taskBodyPlaceholderMaxRatio is the maximum fraction of the body that
	// may consist of placeholder markers. 0.10 = 10 %.
	taskBodyPlaceholderMaxRatio = 0.10
)

// taskResultSectionHeaders lists the `### Artifacts` style sub-heading
// keywords (parser convention).
var taskResultSectionHeaders = []string{
	"### Artifacts",
	"### Changed files",
	"### Files",
}

// taskBodyPlaceholderMarkers lists the placeholder marker candidates.
// Matching is case-insensitive.
var taskBodyPlaceholderMarkers = []*regexp.Regexp{
	regexp.MustCompile(`\{[^}]{2,}\}`), // {requirement 1}
	regexp.MustCompile(`(?i)\bTODO\b`),
	regexp.MustCompile(`(?i)\bTBD\b`),
	regexp.MustCompile(`(?i)\bplaceholder\b`),
	regexp.MustCompile(`(?i)\bFIXME\b`),
}

// taskResultBacktickPath is the backtick-path pattern extracted from the
// result section. Captures the first backtick token of a phrase like
// `path/to/file.go` or `path/to/file.go (modified)`.
var taskResultBacktickPath = regexp.MustCompile("`([^`]+)`")

// ── Rule 1: task.result_section.files_exist ───────────────────────────

type taskResultFilesExistRule struct{}

func (taskResultFilesExistRule) ID() string       { return "task.result_section.files_exist" }
func (taskResultFilesExistRule) Category() string { return "task" }
func (taskResultFilesExistRule) Description() string {
	return "Validate that file paths in the Task result section exist in the git diff (dedicated implementation)"
}
func (taskResultFilesExistRule) DefaultSeverity() Severity { return SeverityBlock }

func (r taskResultFilesExistRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	if ctx.SelfFilePath == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}
	body, err := os.ReadFile(ctx.SelfFilePath)
	if err != nil {
		res.Status = StatusSkipped
		res.Err = err
		res.Duration = time.Since(start)
		return res
	}

	paths := extractResultSectionPaths(string(body))
	if len(paths) == 0 {
		// Result section is empty or no sub-heading is used — skip
		// (another policy handles this).
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	diffSet, err := gitChangedFilesSinceTaskStart()
	if err != nil {
		res.Status = StatusSkipped
		res.Err = err
		res.Duration = time.Since(start)
		return res
	}

	var missing []string
	for _, p := range paths {
		if _, ok := diffSet[p]; !ok {
			missing = append(missing, p)
		}
	}
	missing = FilterEvidence(missing, ctx.SelfFilePath)

	if len(missing) > 0 {
		res.Status = StatusViolated
		res.Message = "Files listed in the result section are missing from the git diff"
		res.Evidence = missing
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// ── Rule 2: task.body.not_placeholder ─────────────────────────────────

type taskBodyNotPlaceholderRule struct{}

func (taskBodyNotPlaceholderRule) ID() string       { return "task.body.not_placeholder" }
func (taskBodyNotPlaceholderRule) Category() string { return "task" }
func (taskBodyNotPlaceholderRule) Description() string {
	return "Validate that the Task body is not filled solely with placeholders (TODO/TBD/{...}) (dedicated)"
}
func (taskBodyNotPlaceholderRule) DefaultSeverity() Severity { return SeverityWarn }

func (r taskBodyNotPlaceholderRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}

	if ctx.SelfFilePath == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}
	data, err := os.ReadFile(ctx.SelfFilePath)
	if err != nil {
		res.Status = StatusSkipped
		res.Err = err
		res.Duration = time.Since(start)
		return res
	}
	body := string(data)
	stripped := stripFrontmatter(body)
	contentLen := len(strings.TrimSpace(stripped))

	if contentLen < taskBodyMinChars {
		res.Status = StatusViolated
		res.Message = "Body too short (suspected placeholder)"
		res.Evidence = []string{"len=" + itoa(contentLen)}
		res.Duration = time.Since(start)
		return res
	}

	markerHits := 0
	for _, re := range taskBodyPlaceholderMarkers {
		markerHits += len(re.FindAllStringIndex(stripped, -1))
	}
	ratio := float64(markerHits) / float64(contentLen) * 1000.0 // markers per 1k chars
	// Violation when there are more than 100 placeholder markers per 1k
	// chars (= 10 %).
	if ratio > taskBodyPlaceholderMaxRatio*1000 {
		res.Status = StatusViolated
		res.Message = "Placeholder marker ratio exceeds the threshold"
		res.Evidence = []string{"markers=" + itoa(markerHits) + " content_len=" + itoa(contentLen)}
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// ── utilities ────────────────────────────────────────────────────────

// extractResultSectionPaths extracts every backtick-delimited path token
// found under an `### Artifacts`-style sub-heading in a Task markdown file.
// Extraction stops when the next H2 (`## `) heading appears.
func extractResultSectionPaths(body string) []string {
	lines := strings.Split(body, "\n")
	inResult := false
	var paths []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inResult = false
		}
		if isResultSectionHeader(trimmed) {
			inResult = true
			continue
		}
		if !inResult {
			continue
		}
		for _, m := range taskResultBacktickPath.FindAllStringSubmatch(line, -1) {
			if len(m) >= 2 {
				p := strings.TrimSpace(m[1])
				// Strip trailing description: "path (modified)" → "path".
				if idx := strings.Index(p, " "); idx > 0 {
					p = p[:idx]
				}
				if p != "" {
					paths = append(paths, p)
				}
			}
		}
	}
	return paths
}

// isResultSectionHeader reports whether a line is an artifacts sub-heading.
func isResultSectionHeader(line string) bool {
	for _, h := range taskResultSectionHeaders {
		if line == h {
			return true
		}
	}
	return false
}

// stripFrontmatter removes a YAML frontmatter block from a markdown body.
func stripFrontmatter(body string) string {
	if !strings.HasPrefix(body, "---\n") {
		return body
	}
	rest := body[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return body
	}
	return rest[end+5:]
}

// gitChangedFilesSinceTaskStart returns the set of files changed in the
// current staged set plus the most recent commit. The implementation is
// intentionally simple and only looks at staged + last commit.
var gitChangedFilesSinceTaskStart = func() (map[string]struct{}, error) {
	out := map[string]struct{}{}
	stagedCmd := exec.Command("git", "diff", "--name-only", "--cached")
	if data, err := stagedCmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line != "" {
				out[line] = struct{}{}
			}
		}
	}
	lastCmd := exec.Command("git", "diff", "--name-only", "HEAD~1..HEAD")
	if data, err := lastCmd.Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line != "" {
				out[line] = struct{}{}
			}
		}
	}
	return out, nil
}

func init() {
	Register(taskResultFilesExistRule{})
	Register(taskBodyNotPlaceholderRule{})
}
