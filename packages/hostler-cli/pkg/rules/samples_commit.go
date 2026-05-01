package rules

import (
	"regexp"
	"strings"
	"time"
	"unicode"
)

// commit-convention 3-rule implementation.
//
// The Rules in this file read the commit-msg body and the staged diff from
// RuleContext.Extra and inspect them. Actual git I/O is performed by the
// CLI runner (`hstl rules run`); the Rules behave as pure functions
// consuming Extra.
//
// Design choices:
//   - Avoid circular imports: pkg/rules does not import git/task/sprint.
//     External data is provided exclusively via the Extra map.
//   - Missing data: when Extra has nothing relevant, the result is
//     StatusSkipped (defends against test environments and accidental
//     invocation in unrelated phases).
//   - Evidence: on violation, return concrete evidence (matched line,
//     score, missing keyword, etc.) so the user can fix it immediately.

// ── constants: Korean ratio threshold and regexes ────────────────────

const (
	// koreanRatioThreshold is the minimum proportion of letters required
	// for a commit message to be considered Korean. 0.30 = 30 % of the
	// letters must be Hangul. This must match the value in the KB
	// feedback_korean_commits card.
	koreanRatioThreshold = 0.30
	// commitMsgMinLength is the minimum letter count for which the
	// Korean-ratio computation is trusted.
	commitMsgMinLength = 8
)

// taskIDPattern matches task IDs in the form T###. Case-insensitive.
var taskIDPattern = regexp.MustCompile(`(?i)\bT\d{1,5}\b`)

// secretPatterns lists the secret candidates we forbid. Static matching
// via grep -E. Only common accidental-commit cases are included; prefixes
// and structures are required to limit false positives. This is not a
// substitute for a full SAST — it is a "first line of defence".
var secretPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"aws_access_key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"private_key_begin", regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`)},
	{"slack_token", regexp.MustCompile(`xox[baprs]-[0-9A-Za-z-]{10,}`)},
	{"github_token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`)},
	{"generic_api_key_assignment", regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token)\s*[:=]\s*["'][A-Za-z0-9_\-]{20,}["']`)},
}

// File-path exceptions for the secret scanner.
//
// Test files, test data, and fixtures may legitimately contain literal
// secrets, so they are excluded from scanning. The path is extracted from
// the `diff --git b/<path>` header of the git diff.
var secretPathSkipPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(^|/)[^/]*_test\.go$`),
	regexp.MustCompile(`(^|/)[^/]*\.test\.[a-zA-Z0-9]+$`),
	regexp.MustCompile(`(^|/)testdata/`),
	regexp.MustCompile(`(^|/)fixtures?/`),
}

// `// nosecret` / `# nosecret` line marker.
// A line containing this marker is excluded individually (no effect on
// adjacent lines).
var nosecretMarker = regexp.MustCompile(`(?i)(//|#)\s*nosecret\b`)

// diffFileHeader is the header that begins each file block in
// `git diff --cached` output. Capture group 1 is the b-side path
// (post-modification path).
var diffFileHeader = regexp.MustCompile(`(?m)^diff --git a/(?:[^\s]+) b/([^\s]+)`)

// ── Rule 1: commit.message.korean ──────────────────────────────────────

type commitMessageKoreanRule struct{}

func (commitMessageKoreanRule) ID() string       { return "commit.message.korean" }
func (commitMessageKoreanRule) Category() string { return "commit" }
func (commitMessageKoreanRule) Description() string {
	return "Validate that the commit message body is written in Korean (Co-Authored-By line is excluded)"
}
func (commitMessageKoreanRule) DefaultSeverity() Severity { return SeverityWarn }

func (r commitMessageKoreanRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}
	raw, ok := ctx.Extra[ExtraKeyCommitMsg].(string)
	if !ok || raw == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}
	body := stripCoAuthored(raw)
	body = strings.TrimSpace(body)
	letters := 0
	hangul := 0
	for _, r := range body {
		if unicode.IsLetter(r) {
			letters++
			if unicode.Is(unicode.Hangul, r) {
				hangul++
			}
		}
	}
	if letters < commitMsgMinLength {
		// Message is too short to evaluate — skip.
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}
	ratio := float64(hangul) / float64(letters)
	if ratio < koreanRatioThreshold {
		res.Status = StatusViolated
		res.Message = "Commit message body has a low Korean ratio. Please write the body (other than Co-Authored-By) in Korean."
		res.Evidence = []string{
			firstLine(body),
		}
	} else {
		res.Status = StatusOK
	}
	res.Duration = time.Since(start)
	return res
}

// ── Rule 2: commit.task_id.present ────────────────────────────────────

type commitTaskIDPresentRule struct{}

func (commitTaskIDPresentRule) ID() string       { return "commit.task_id.present" }
func (commitTaskIDPresentRule) Category() string { return "commit" }
func (commitTaskIDPresentRule) Description() string {
	return "Validate that the commit message contains a Task ID (T###)"
}
func (commitTaskIDPresentRule) DefaultSeverity() Severity { return SeverityAdvisory }

func (r commitTaskIDPresentRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}
	raw, ok := ctx.Extra[ExtraKeyCommitMsg].(string)
	if !ok || raw == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}
	body := stripCoAuthored(raw)
	if taskIDPattern.MatchString(body) {
		res.Status = StatusOK
	} else {
		res.Status = StatusViolated
		res.Message = "Commit message has no Task ID (T###). Linking the Task to the commit is recommended."
		res.Evidence = []string{firstLine(body)}
	}
	res.Duration = time.Since(start)
	return res
}

// ── Rule 3: commit.files.no_secrets ───────────────────────────────────

type commitFilesNoSecretsRule struct{}

func (commitFilesNoSecretsRule) ID() string       { return "commit.files.no_secrets" }
func (commitFilesNoSecretsRule) Category() string { return "commit" }
func (commitFilesNoSecretsRule) Description() string {
	return "Validate that the staged diff contains no AWS/Slack/GitHub tokens, private keys, or api_key assignment literals"
}
func (commitFilesNoSecretsRule) DefaultSeverity() Severity { return SeverityBlock }

func (r commitFilesNoSecretsRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}
	diff, ok := ctx.Extra[ExtraKeyStagedDiff].(string)
	if !ok || diff == "" {
		res.Status = StatusSkipped
		res.Duration = time.Since(start)
		return res
	}

	var hits []string
	skippedPaths := 0
	markerSkips := 0

	for _, block := range splitDiffByFile(diff) {
		if isSecretSkipPath(block.path) {
			skippedPaths++
			continue
		}
		for _, line := range strings.Split(block.body, "\n") {
			// Only inspect added lines — exclude the +++ header.
			if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
				continue
			}
			if nosecretMarker.MatchString(line) {
				markerSkips++
				continue
			}
			for _, p := range secretPatterns {
				if loc := p.pattern.FindStringIndex(line); loc != nil {
					snippet := line[loc[0]:loc[1]]
					hits = append(hits, p.name+" ("+block.path+"): "+snippet)
				}
			}
		}
	}

	if len(hits) > 0 {
		res.Status = StatusViolated
		res.Message = "A secret pattern was detected in the staged diff. Please cancel the commit and remove the secret."
		res.Evidence = hits
	} else {
		res.Status = StatusOK
		if skippedPaths > 0 || markerSkips > 0 {
			res.Evidence = []string{
				"path_skipped=" + itoa(skippedPaths) + " marker_skipped=" + itoa(markerSkips),
			}
		}
	}
	res.Duration = time.Since(start)
	return res
}

// diffFileBlock is the unit produced by splitDiffByFile.
type diffFileBlock struct {
	path string
	body string
}

// splitDiffByFile splits `git diff --cached` output into per-file blocks.
// Blocks are separated at every `diff --git a/<old> b/<new>` header and
// the b-side path is captured. A diff without a header (e.g. a hand-crafted
// fragment) returns a single block with path="".
func splitDiffByFile(diff string) []diffFileBlock {
	matches := diffFileHeader.FindAllStringSubmatchIndex(diff, -1)
	if len(matches) == 0 {
		return []diffFileBlock{{path: "", body: diff}}
	}
	blocks := make([]diffFileBlock, 0, len(matches))
	for i, m := range matches {
		pathStart, pathEnd := m[2], m[3]
		bodyStart := m[1]
		bodyEnd := len(diff)
		if i+1 < len(matches) {
			bodyEnd = matches[i+1][0]
		}
		blocks = append(blocks, diffFileBlock{
			path: diff[pathStart:pathEnd],
			body: diff[bodyStart:bodyEnd],
		})
	}
	return blocks
}

// isSecretSkipPath reports whether the path matches one of the
// test/fixture exception patterns.
func isSecretSkipPath(path string) bool {
	if path == "" {
		return false
	}
	for _, re := range secretPathSkipPatterns {
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

// itoa is a very small int-to-string helper (avoids importing strconv;
// used for evidence strings).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ── utilities ─────────────────────────────────────────────────────────

// stripCoAuthored removes lines beginning with "Co-Authored-By:" from the
// commit message body. This excludes trailers from the Korean-ratio
// computation.
func stripCoAuthored(msg string) string {
	lines := strings.Split(msg, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Co-Authored-By:") || strings.HasPrefix(trimmed, "Signed-off-by:") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// firstLine returns the first non-empty line of a message (used as
// evidence).
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func init() {
	Register(commitMessageKoreanRule{})
	Register(commitTaskIDPresentRule{})
	Register(commitFilesNoSecretsRule{})
}
