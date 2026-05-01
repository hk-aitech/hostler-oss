// Task body claim re-validation.
// Extracts whitelisted commands (grep / wc / ls / rg / find) from
// ```bash code fences in the Task body, re-runs them, and compares the
// resulting line count against numbers like "N matches" / "N items" /
// "N files" recorded in the body. Returns drift WARNs on mismatch.
// Motivation: a downstream incident — a Task was authored claiming
// "BootstrapEventIds grep 0 hits" but between planning and execution the
// scope spread to 7 Hosts × 9~10 calls. If the AI/user reads the body
// literally, the same range-misinterpretation can recur. This module
// detects it at task-start time and emits drift evidence.
// Security policy:
// Executable whitelist: only grep / rg / wc / ls / find allowed.
// Forbid shell metacharacters: |, >, <, &, ;, $(, `, \n inside the
// command string.
// Run via exec.Command with separate executable + args (bypasses the
// shell).
// Per-command timeout: 5 s (context.WithTimeout).
// Total timeout: 30 s (managed by the caller — current implementation
// enforces only the per-command timeout).

package task

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// allowedExecutables — commands that may be run during body-claim
// re-validation.
var allowedExecutables = map[string]bool{
	"grep": true,
	"rg":   true,
	"wc":   true,
	"ls":   true,
	"find": true,
}

// dangerousMetaRe — pattern that detects shell metacharacters. A match
// rejects execution.
var dangerousMetaRe = regexp.MustCompile("[|><&;$`\n]|\\$\\(")

// claimNumberRe — extracts the count from body lines that read like
// "N matches" / "N items" / "N files" / "N lines". The first match
// within the bullet line is used.
var claimNumberRe = regexp.MustCompile(`(\d+)\s*(?:lines?|files?|occurrences?|matches?|items?|hits?)`)

// BodyClaim — one re-validatable unit inside the body.
type BodyClaim struct {
	Command       []string // ["grep", "-rn", "pattern", "path/"]
	ExpectedCount int      // expected value extracted from the claim line
	ClaimLine     string   // claim text (used in the report)
	CodeFenceLine int      // 1-based; line within the bash code fence
}

// ParseBodyClaims extracts re-validatable claims from a Task body.
// Rules:
// 1. Only ```bash ... ``` code-fence blocks are considered.
// 2. Extract whitelisted commands without metacharacters from inside the
// block.
// 3. Within a 5-line window before/after each block, link the first
// claimNumberRe match as the expected value.
// 4. If no expected value is linked, the block is not validated (silent
// skip).
func ParseBodyClaims(body string) []BodyClaim {
	lines := strings.Split(body, "\n")
	claims := []BodyClaim{}
	inBash := false
	bashStart := -1
	var bashCmds []int // 0-based line indices of commands inside the bash fence
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "```bash") {
			inBash = true
			bashStart = i
			bashCmds = bashCmds[:0]
			continue
		}
		if inBash && strings.HasPrefix(trimmed, "```") {
			// Code fence ended — match collected commands with the
			// surrounding claim.
			if len(bashCmds) > 0 {
				expected := findNearbyClaim(lines, bashStart, i)
				if expected >= 0 {
					for _, ci := range bashCmds {
						args := splitCmd(strings.TrimSpace(lines[ci]))
						if args == nil {
							continue
						}
						claims = append(claims, BodyClaim{
							Command:       args,
							ExpectedCount: expected,
							ClaimLine:     strings.TrimSpace(lines[ci]),
							CodeFenceLine: ci + 1,
						})
					}
				}
			}
			inBash = false
			bashStart = -1
			bashCmds = bashCmds[:0]
			continue
		}
		if inBash {
			clean := strings.TrimSpace(line)
			// Skip comments / blank lines.
			if clean == "" || strings.HasPrefix(clean, "#") {
				continue
			}
			bashCmds = append(bashCmds, i)
		}
	}
	return claims
}

// findNearbyClaim — find the first claim-number match within ±5 lines of
// the bash code-fence block (start..end). Returns the extracted integer
// or -1 (no match).
func findNearbyClaim(lines []string, start, end int) int {
	windowBefore := start - 5
	if windowBefore < 0 {
		windowBefore = 0
	}
	windowAfter := end + 5
	if windowAfter > len(lines) {
		windowAfter = len(lines)
	}
	// Inspect the "before" window first — narrative such as "measured
	// result" is usually written above the fence.
	for i := windowBefore; i < start; i++ {
		if m := claimNumberRe.FindStringSubmatch(lines[i]); len(m) > 1 {
			v, err := strconv.Atoi(m[1])
			if err == nil {
				return v
			}
		}
	}
	// "After" window.
	for i := end + 1; i < windowAfter; i++ {
		if m := claimNumberRe.FindStringSubmatch(lines[i]); len(m) > 1 {
			v, err := strconv.Atoi(m[1])
			if err == nil {
				return v
			}
		}
	}
	return -1
}

// splitCmd — split a single command line into executable + args.
// Also runs whitelist + metachar checks; returns nil if the command is
// unsafe. Uses simple whitespace splitting rather than shell parsing
// (quoted arguments are unsupported and skipped when present).
func splitCmd(line string) []string {
	// Strip a leading "$ " prompt ("$ grep ...").
	if strings.HasPrefix(line, "$ ") {
		line = strings.TrimPrefix(line, "$ ")
	}
	if dangerousMetaRe.MatchString(line) {
		return nil
	}
	// Quoted segments make the split unreliable — skip.
	if strings.ContainsAny(line, `"'`) {
		return nil
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}
	if !allowedExecutables[parts[0]] {
		return nil
	}
	return parts
}

// RevalidateClaims runs each claim and compares the resulting line count
// against the expected value. The execution directory is workDir (the
// project root that contains the Task file). Failed commands are skipped.
// Returns warning messages (empty when no drift).
func RevalidateClaims(claims []BodyClaim, workDir string) []string {
	warnings := []string{}
	for _, c := range claims {
		count, ok := runAndCountLines(c.Command, workDir)
		if !ok {
			continue
		}
		if count != c.ExpectedCount {
			warnings = append(warnings,
				fmt.Sprintf("body claim drift (line %d): body recorded %d != actual %d. Command: `%s`",
					c.CodeFenceLine, c.ExpectedCount, count, c.ClaimLine))
		}
	}
	return warnings
}

// runAndCountLines — runs the command and counts stdout lines. Returns
// (false) on failure. Timeout: 5 s. exit code 1 (grep no match) is
// treated as a line count of 0.
func runAndCountLines(args []string, workDir string) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	// Minimal env.
	cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/local/bin", "LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		// grep exit 1 = no match. Other errors are skipped.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && args[0] == "grep" {
			return 0, true
		}
		return 0, false
	}
	// `wc -l` and friends print the number itself; other commands need a
	// line count.
	if len(args) > 0 && args[0] == "wc" {
		// Extract the first numeric token from wc output.
		s := strings.TrimSpace(string(out))
		parts := strings.Fields(s)
		if len(parts) > 0 {
			if n, err := strconv.Atoi(parts[0]); err == nil {
				return n, true
			}
		}
		return 0, false
	}
	return countLinesInBytes(out), true
}

// countLinesInBytes — counts non-empty lines in the output.
func countLinesInBytes(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			n++
		}
	}
	return n
}

// RevalidateTaskBody reads the Task file and returns body-claim
// re-validation results. Opt out via env
// HSTL_TASK_START_BODY_REVALIDATE=off (returns an empty slice).
func RevalidateTaskBody(taskFile string) []string {
	if envalias.Lookup("TASK_START_BODY_REVALIDATE") == "off" {
		return nil
	}
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return nil
	}
	claims := ParseBodyClaims(string(data))
	if len(claims) == 0 {
		return nil
	}
	// workDir: project root for the Task file (the parent of works/).
	workDir := findProjectRoot(taskFile)
	return RevalidateClaims(claims, workDir)
}

// findProjectRoot — locates the directory above `works/` in the
// taskFile path. Returns an empty string on failure (the cmd's cwd is
// used in that case).
func findProjectRoot(taskFile string) string {
	idx := strings.LastIndex(taskFile, "/works/")
	if idx < 0 {
		return ""
	}
	return taskFile[:idx]
}
