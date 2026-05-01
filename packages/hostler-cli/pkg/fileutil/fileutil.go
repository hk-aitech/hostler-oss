// Package fileutil creates/reads/updates Task markdown files and updates
// BACKLOG.md / SPRINT.md tables.
package fileutil

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Project root detection
// ---------------------------------------------------------------------------

// GetProjectRoot detects the project root directory.
//
// Lookup priority:
//  1. HSTL_PROJECT_ROOT environment variable (test isolation override)
//  2. git rev-parse --show-toplevel
//  3. current working directory (fallback when git is uninitialised)
func GetProjectRoot() string {
	if override := strings.TrimSpace(envalias.Lookup("PROJECT_ROOT")); override != "" {
		return override
	}
	out, err := runGit("rev-parse", "--show-toplevel")
	if err == nil && out != "" {
		return out
	}
	cwd, _ := os.Getwd()
	return cwd
}

// GetMainRepoRoot returns the main repo path of a git worktree.
// When not in a worktree, behaves like GetProjectRoot().
func GetMainRepoRoot() string {
	if override := strings.TrimSpace(envalias.Lookup("PROJECT_ROOT")); override != "" {
		return override
	}
	out, err := runGit("rev-parse", "--git-common-dir")
	if err == nil && out != "" {
		// Absolute path means worktree — the parent directory is the main repo.
		if filepath.IsAbs(out) {
			return filepath.Dir(out)
		}
		// Relative ".git" -> the current toplevel is the main repo.
		return GetProjectRoot()
	}
	return GetProjectRoot()
}

// ToRepoRelative converts an absolute path to a path relative to the main repo.
// Used for storing paths in the DB so that worktree vs main-repo paths stay consistent.
func ToRepoRelative(absPath string) string {
	mainRoot := GetMainRepoRoot()
	if rel, err := filepath.Rel(mainRoot, absPath); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	root := GetProjectRoot()
	if rel, err := filepath.Rel(root, absPath); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return absPath
}

// runGit executes a git command and returns the trimmed stdout.
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ---------------------------------------------------------------------------
// Slug generation
// ---------------------------------------------------------------------------

var (
	reMultiHyphen = regexp.MustCompile(`-+`)
)

// slugMaxLen — maximum slug length in characters.
const slugMaxLen = 50

// MakeSlug generates a URL-safe slug from a title.
// Keeps letters, digits, and hyphens; replaces spaces with hyphens.
func MakeSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Keep only letters, digits, and hyphens.
	slug = removeNonSlugChars(slug)
	// Collapse consecutive hyphens.
	slug = reMultiHyphen.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	runes := []rune(slug)
	if len(runes) > slugMaxLen {
		runes = runes[:slugMaxLen]
	}
	return string(runes)
}

// removeNonSlugChars drops characters that are not letters, digits, hyphens, or underscores.
func removeNonSlugChars(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Task file creation
// ---------------------------------------------------------------------------

// CreateTaskFile creates a Task markdown file and returns its path.
//
// Path policy (fallback ambiguity removed):
//   - sprint == "" or "backlog" (or other unassigned values) -> works/tasks/T009-{slug}.md (backlog default)
//   - sprint == "sprint-NN" + folder exists -> works/sprints/{active|backlog}/sprint-NN/tasks/T009-{slug}.md
//   - sprint == "sprint-NN" + folder missing -> **returns an error** (no silent backlog fallback)
//
// The CLI (`hstl task create`) always passes sprint="" since the policy
// change, but the internal API still accepts the sprint parameter. This
// function enforces the assumption "if a Sprint is specified, the
// corresponding Sprint folder must exist". When a non-existent Sprint
// silently writes to backlog, frontmatter and the actual path drift apart
// and trigger backlog_sync drift (a previously documented mistake).
//
// Doc: docs/04-guides/task-paths.md
func CreateTaskFile(taskID, title, taskType, sprint, priority, estimate string, dependsOn []string, content string) (string, error) {
	root := GetProjectRoot()
	slug := MakeSlug(title)
	filename := fmt.Sprintf("%s-%s.md", taskID, slug)

	var taskDir string
	if sprint != "" && sprint != "backlog" {
		sprintDir, _, err := FindSprintDir(sprint)
		if err != nil {
			// No silent backlog fallback — clear error when the sprint folder is missing.
			return "", fmt.Errorf("Sprint %q folder not found in works/sprints/{active,backlog,completed}/. Create the Sprint first or pass an empty sprint argument to write into backlog", sprint)
		}
		taskDir = filepath.Join(sprintDir, "tasks")
	} else {
		taskDir = filepath.Join(root, "works", "tasks")
	}

	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return "", fmt.Errorf("create task directory: %w", err)
	}
	filePath := filepath.Join(taskDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write task file: %w", err)
	}
	return filePath, nil
}

// ---------------------------------------------------------------------------
// Move unassigned Task folders
// ---------------------------------------------------------------------------

// MoveTaskToCompleted moves a Sprint-unassigned Task file from
// works/tasks/T*.md to works/tasks/completed/T*.md when the Task transitions to done.
//
// Input:
//
//	filePath — absolute or repo-relative path to the current Task file.
//
// Behaviour:
//   - no-op when not at works/tasks/T*.md (e.g. sprint-assigned Task)
//   - no-op when already under works/tasks/completed/ (idempotent)
//   - creates works/tasks/completed/ automatically
//   - tries git mv; falls back to os.Rename
//
// Returns: new absolute path, whether moved, error.
func MoveTaskToCompleted(filePath string) (string, bool, error) {
	root := GetProjectRoot()
	abs := filePath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, filePath)
	}

	// Normalise: only direct children of works/tasks/ are eligible. Files
	// already under completed/ have moved already.
	tasksDir := filepath.Join(root, "works", "tasks")
	completedDir := filepath.Join(tasksDir, "completed")
	parent := filepath.Dir(abs)

	if parent != tasksDir {
		// Not a direct child of works/tasks/ -> no-op (sprint-assigned / already completed / etc.)
		return abs, false, nil
	}

	if err := os.MkdirAll(completedDir, 0o755); err != nil {
		return abs, false, fmt.Errorf("create completed directory: %w", err)
	}

	dst := filepath.Join(completedDir, filepath.Base(abs))
	if _, err := os.Stat(dst); err == nil {
		// Same name at the destination — for idempotency, just remove src (no duplicate retention).
		_ = os.Remove(abs)
		return dst, true, nil
	}

	// Try git mv.
	cmd := exec.Command("git", "mv", abs, dst)
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		// Fallback: os.Rename.
		if rErr := os.Rename(abs, dst); rErr != nil {
			return abs, false, fmt.Errorf("move task file: %w", rErr)
		}
	}
	return dst, true, nil
}

// MoveTaskFromCompleted moves a Task file back from works/tasks/completed/T*.md
// to works/tasks/T*.md when reopening (done -> todo/in-progress).
//
// Behaviour:
//   - no-op when not at works/tasks/completed/T*.md
//   - no-op when works/tasks/T*.md already has the same name + deletes the
//     completed/ file (prevents dual retention)
//   - tries git mv; falls back to os.Rename
func MoveTaskFromCompleted(filePath string) (string, bool, error) {
	root := GetProjectRoot()
	abs := filePath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, filePath)
	}

	tasksDir := filepath.Join(root, "works", "tasks")
	completedDir := filepath.Join(tasksDir, "completed")
	parent := filepath.Dir(abs)

	if parent != completedDir {
		return abs, false, nil
	}

	dst := filepath.Join(tasksDir, filepath.Base(abs))
	if _, err := os.Stat(dst); err == nil {
		_ = os.Remove(abs)
		return dst, true, nil
	}

	cmd := exec.Command("git", "mv", abs, dst)
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		if rErr := os.Rename(abs, dst); rErr != nil {
			return abs, false, fmt.Errorf("restore task file: %w", rErr)
		}
	}
	return dst, true, nil
}

// ---------------------------------------------------------------------------
// frontmatter read / status update
// ---------------------------------------------------------------------------

// ReadTaskFrontmatter returns the Task file frontmatter as a map.
func ReadTaskFrontmatter(filePath string) (map[string]any, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	fm, _, err := parseFrontmatter(data)
	if err != nil {
		return nil, err
	}
	return fm, nil
}

// UpdateTaskFrontmatter partially updates frontmatter fields of a Task file.
// Replaces only the specified keys; the remaining frontmatter and body stay intact.
func UpdateTaskFrontmatter(filePath string, fields map[string]any) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	fm, body, err := parseFrontmatter(data)
	if err != nil {
		return err
	}

	for k, v := range fields {
		fm[k] = v
	}

	out, err := dumpFrontmatter(fm, body)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(out), 0o644)
}

// UpdateTaskStatus updates only the frontmatter status field (compat wrapper).
func UpdateTaskStatus(filePath, newStatus string) error {
	return UpdateTaskFrontmatter(filePath, map[string]any{"status": newStatus})
}

// parseFrontmatter splits a markdown file into YAML frontmatter and body.
func parseFrontmatter(data []byte) (map[string]any, string, error) {
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return map[string]any{}, content, nil
	}

	// Locate the second "---" after the first one.
	rest := content[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return map[string]any{}, content, nil
	}

	fmRaw := rest[:idx]
	body := strings.TrimPrefix(rest[idx+4:], "\n") // after "\n---"

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return nil, "", fmt.Errorf("parse YAML frontmatter: %w", err)
	}
	if fm == nil {
		fm = map[string]any{}
	}
	return fm, body, nil
}

// CoerceDateFields converts time.Time values in date fields to "2006-01-02"
// strings so that yaml.Encode does not emit a "T00:00:00Z" time portion.
// Resolves the round-trip issue caused by yaml.v3 auto-parsing bare dates as time.Time.
//
// Target fields (SPRINT/Task frontmatter convention): created, started, completed.
// When time precision is required, record it separately in the audit log or history section.
//
// Exported so CanonicalTaskBytes / CanonicalSprintBytes can call it before
// canonical computation, guaranteeing stamp <-> verify round-trip
// invariance. (Equalises the canonical of files containing bare YAML dates
// vs. quoted strings, preventing recurring HMAC mismatches.)
func CoerceDateFields(fm map[string]any) {
	dateFields := []string{"created", "started", "completed"}
	for _, k := range dateFields {
		v, ok := fm[k]
		if !ok {
			continue
		}
		if t, ok := v.(time.Time); ok {
			fm[k] = t.Format("2006-01-02")
		}
	}
}

// dumpFrontmatter combines the frontmatter map and body into a markdown string.
func dumpFrontmatter(fm map[string]any, body string) (string, error) {
	// Convert date-typed time.Time fields to "YYYY-MM-DD" strings.
	CoerceDateFields(fm)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(fm); err != nil {
		return "", fmt.Errorf("YAML serialize: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("close YAML encoder: %w", err)
	}

	fmStr := strings.TrimRight(buf.String(), "\n")
	return fmt.Sprintf("---\n%s\n---\n%s", fmStr, body), nil
}

// ---------------------------------------------------------------------------
// Sprint directory lookup (shared helper)
// ---------------------------------------------------------------------------

// FindSprintDir locates the Sprint directory for sprint_id, searching
// active/backlog/completed in that order.
// Returns: (sprintDir, location, error). location is "active"|"backlog"|"completed".
func FindSprintDir(sprintID string) (string, string, error) {
	root := GetProjectRoot()
	for _, loc := range []string{"active", "backlog", "completed"} {
		candidate := filepath.Join(root, "works", "sprints", loc, sprintID)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, loc, nil
		}
	}
	return "", "", fmt.Errorf("sprint directory not found: %s", sprintID)
}

// ---------------------------------------------------------------------------
// BACKLOG.md row removal
// ---------------------------------------------------------------------------

// RemoveFromBacklogMD removes the row for a given Task from works/tasks/BACKLOG.md.
// Returns true on removal, false when the file is missing or the row is absent.
func RemoveFromBacklogMD(taskID string) (bool, error) {
	root := GetProjectRoot()
	backlogPath := filepath.Join(root, "works", "tasks", "BACKLOG.md")
	if _, err := os.Stat(backlogPath); os.IsNotExist(err) {
		return false, nil
	}

	data, err := os.ReadFile(backlogPath)
	if err != nil {
		return false, fmt.Errorf("read BACKLOG.md: %w", err)
	}

	lines := splitLines(string(data))
	marker := fmt.Sprintf("| %s |", taskID)
	newLines := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if strings.Contains(line, marker) {
			removed = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !removed {
		return false, nil
	}

	if err := os.WriteFile(backlogPath, []byte(joinLines(newLines)), 0o644); err != nil {
		return false, fmt.Errorf("write BACKLOG.md: %w", err)
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// BACKLOG.md update
// ---------------------------------------------------------------------------

// UpdateBacklogMD adds a row to the unassigned-Tasks section of works/tasks/BACKLOG.md.
// Parses the existing table header and emits a row in matching column order.
// Auto-creates BACKLOG.md when missing.
func UpdateBacklogMD(taskID, title, taskType, estimate, priority string) error {
	root := GetProjectRoot()
	backlogDir := filepath.Join(root, "works", "tasks")
	backlogPath := filepath.Join(backlogDir, "BACKLOG.md")
	if _, err := os.Stat(backlogPath); os.IsNotExist(err) {
		// Create the parent directory and write a skeleton BACKLOG.md.
		if mkErr := os.MkdirAll(backlogDir, 0o755); mkErr != nil {
			return fmt.Errorf("create BACKLOG.md directory: %w", mkErr)
		}
		skeleton := `# Backlog

Project list of unassigned Tasks (no Sprint). Auto-synchronised on
` + "`hstl task create`" + ` / ` + "`hstl task assign`" + `.

## Unassigned Tasks

| ID | Title | Size | Type | Priority | Depends |
|----|-------|------|------|----------|---------|
`
		if wErr := os.WriteFile(backlogPath, []byte(skeleton), 0o644); wErr != nil {
			return fmt.Errorf("auto-create BACKLOG.md: %w", wErr)
		}
	}

	data, err := os.ReadFile(backlogPath)
	if err != nil {
		return fmt.Errorf("read BACKLOG.md: %w", err)
	}
	content := string(data)
	lines := splitLines(content)

	// Find the table header to build the column mapping.
	headerIdx := -1
	var headerCols []string
	for i, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(line, "| ID |") || strings.Contains(lower, "| id |") {
			headerIdx = i
			parts := strings.Split(line, "|")
			for _, p := range parts[1 : len(parts)-1] {
				col := strings.ToLower(strings.TrimSpace(strings.Trim(strings.TrimSpace(p), "*")))
				headerCols = append(headerCols, col)
			}
			break
		}
	}

	if headerIdx < 0 {
		// No table — append a default-format row at end of file.
		newRow := fmt.Sprintf("| %s | %s | %s | %s | %s | — |", taskID, title, estimate, taskType, priority)
		content = content + "\n" + newRow + "\n"
		return os.WriteFile(backlogPath, []byte(content), 0o644)
	}

	// Column-name -> value mapping.
	colValues := map[string]string{
		"id":         taskID,
		"title":      title,
		"estimate":   estimate,
		"size":       estimate,
		"type":       taskType,
		"priority":   priority,
		"depends":    "—",
		"depends_on": "—",
		"note":       "—",
		"notes":      "—",
	}

	// Emit a row in the header column order.
	var cells []string
	for _, col := range headerCols {
		val, ok := colValues[col]
		if !ok {
			val = "—"
		}
		cells = append(cells, val)
	}
	newRow := "| " + strings.Join(cells, " | ") + " |"

	// Insert just before the end of the table (next ## header or EOF).
	insertIdx := -1
	inTable := false
	for i, line := range lines {
		if i == headerIdx {
			inTable = true
		}
		if inTable && i > headerIdx+1 && strings.HasPrefix(line, "## ") {
			insertIdx = i
			break
		}
	}

	if insertIdx >= 0 {
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:insertIdx]...)
		newLines = append(newLines, newRow)
		newLines = append(newLines, lines[insertIdx:]...)
		lines = newLines
	} else {
		lines = append(lines, newRow)
	}

	return os.WriteFile(backlogPath, []byte(joinLines(lines)), 0o644)
}

// extractFrontmatterField extracts the value for key from a yaml frontmatter
// in a markdown body. Simple line-based parsing (key: value or key: "value").
// Returns "" when not found.
//
// Lightweight helper used by UpdateCurrentFocus to read the
// track_id/milestone fields from SPRINT.md. A simple use case that does
// not justify yaml.Unmarshal.
func extractFrontmatterField(content, key string) string {
	// Frontmatter region only (--- ... ---).
	idx := strings.Index(content, "---\n")
	if idx != 0 {
		return ""
	}
	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return ""
	}
	fm := content[4 : 4+end]
	prefix := key + ":"
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		val := strings.TrimSpace(trimmed[len(prefix):])
		val = strings.Trim(val, `"'`)
		return val
	}
	return ""
}

// ---------------------------------------------------------------------------
// CURRENT-FOCUS.md auto-management
// ---------------------------------------------------------------------------

// UpdateCurrentFocus auto-generates/updates works/CURRENT-FOCUS.md on
// Sprint state transitions. Skips when works/sprints/ does not exist
// (projects that do not use sprints). Manages the file by rewriting both
// frontmatter and body in full so ceremony files are not damaged.
//
// Parameters:
//
//	activeSprintID: the active Sprint ID (empty when there is none)
//	activeSprintTitle: title (empty when unused)
//	status: "sprint-active" | "sprint-completed" | "idle"
//
// Overwrites the existing file. When the user already curated rich body
// content, the caller is responsible for merging; this helper simply
// guarantees that the SessionStart hook never sees a stale state.
func UpdateCurrentFocus(activeSprintID, activeSprintTitle, status string) error {
	root := GetProjectRoot()
	// Treat absence of works/sprints/ as a no-sprints project — no-op.
	if _, err := os.Stat(filepath.Join(root, "works", "sprints")); os.IsNotExist(err) {
		return nil
	}
	worksDir := filepath.Join(root, "works")
	if err := os.MkdirAll(worksDir, 0o755); err != nil {
		return fmt.Errorf("create works directory: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	sprintLine := "none"
	statusLine := status
	if statusLine == "" {
		statusLine = "idle"
	}
	if activeSprintID != "" {
		sprintLine = activeSprintID
	}

	// Layer 2 frontmatter auto-fill:
	// (1) track: track_id field from the active SPRINT.md frontmatter
	// (2) environment: HSTL_ENVIRONMENT env (default "LocalDev")
	// (3) milestone: milestone field from SPRINT.md frontmatter (omitted when absent)
	trackLine := ""
	milestoneLine := ""
	environmentLine := strings.TrimSpace(envalias.Lookup("ENVIRONMENT"))
	if environmentLine == "" {
		environmentLine = "LocalDev"
	}
	if activeSprintID != "" {
		sprintMD := filepath.Join(root, "works", "sprints", "active", activeSprintID, "SPRINT.md")
		if data, err := os.ReadFile(sprintMD); err == nil {
			trackLine = extractFrontmatterField(string(data), "track_id")
			milestoneLine = extractFrontmatterField(string(data), "milestone")
		}
	}

	var body strings.Builder
	body.WriteString("---\n")
	body.WriteString("task: none\n")
	body.WriteString(fmt.Sprintf("status: %s\n", statusLine))
	body.WriteString(fmt.Sprintf("sprint: %s\n", sprintLine))
	if trackLine != "" {
		body.WriteString(fmt.Sprintf("track: %s\n", trackLine))
	}
	if milestoneLine != "" {
		body.WriteString(fmt.Sprintf("milestone: %s\n", milestoneLine))
	}
	body.WriteString(fmt.Sprintf("environment: %s\n", environmentLine))
	body.WriteString(fmt.Sprintf("updated: %s\n", today))
	body.WriteString("---\n\n")
	body.WriteString("# Current Focus\n\n")
	body.WriteString("## Current state\n\n")
	if activeSprintID != "" {
		if activeSprintTitle != "" {
			body.WriteString(fmt.Sprintf("**Active Sprint**: %s — %s\n", activeSprintID, activeSprintTitle))
		} else {
			body.WriteString(fmt.Sprintf("**Active Sprint**: %s\n", activeSprintID))
		}
		body.WriteString(fmt.Sprintf("**Status**: %s\n", statusLine))
		body.WriteString(fmt.Sprintf("**Updated**: %s\n\n", today))
	} else {
		body.WriteString("No active Sprint.\n\n")
		body.WriteString(fmt.Sprintf("**Updated**: %s\n\n", today))
	}
	body.WriteString("> This file is updated automatically on `hstl sprint start/complete`.\n")
	body.WriteString("> Manual edits may be overwritten — keep permanent records in\n")
	body.WriteString("> SPRINT.md / Task files under `works/sprints/`.\n")

	focusPath := filepath.Join(worksDir, "CURRENT-FOCUS.md")
	if err := os.WriteFile(focusPath, []byte(body.String()), 0o644); err != nil {
		return fmt.Errorf("write CURRENT-FOCUS.md: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// BACKLOG.md summary metadata refresh
// ---------------------------------------------------------------------------

var (
	reLastUpdated = regexp.MustCompile(`(>\s*Last updated:\s*).*`)
	reTodoHeader  = regexp.MustCompile(`(##\s+Outstanding Unassigned Tasks)\s*—?\s*\d*`)
	reTodoRow     = regexp.MustCompile(`^\|\s*\*?\*?T\d+`)
	reInHistory   = regexp.MustCompile(`##\s+Outstanding Unassigned Tasks`)
)

// RebuildBacklogSummary recomputes BACKLOG.md summary metadata from the
// actual data. Updates: the "> Last updated:" line and the
// "## Outstanding Unassigned Tasks — N" header.
func RebuildBacklogSummary() (*BacklogSummaryResult, error) {
	root := GetProjectRoot()
	backlogPath := filepath.Join(root, "works", "tasks", "BACKLOG.md")
	if _, err := os.Stat(backlogPath); os.IsNotExist(err) {
		return &BacklogSummaryResult{Updated: false, Reason: "BACKLOG.md not found"}, nil
	}

	data, err := os.ReadFile(backlogPath)
	if err != nil {
		return nil, fmt.Errorf("read BACKLOG.md: %w", err)
	}
	content := string(data)
	today := time.Now().Format("2006-01-02")
	updated := false

	// 1. Refresh the "Last updated" line.
	newContent := reLastUpdated.ReplaceAllString(content, "${1}"+today)
	if newContent != content {
		updated = true
		content = newContent
	}

	// 2. Count actual unassigned-Task table rows.
	lines := strings.Split(content, "\n")
	todoCount := 0
	inTodoTable := false
	for _, line := range lines {
		if reInHistory.MatchString(line) {
			inTodoTable = true
			continue
		}
		if inTodoTable && strings.HasPrefix(line, "## ") {
			break
		}
		if inTodoTable && reTodoRow.MatchString(line) {
			todoCount++
		}
	}

	// Refresh the "## Outstanding Unassigned Tasks — N" header.
	newContent = reTodoHeader.ReplaceAllString(content, fmt.Sprintf("${1} — %d", todoCount))
	if newContent != content {
		updated = true
		content = newContent
	}

	if updated {
		if err := os.WriteFile(backlogPath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write BACKLOG.md: %w", err)
		}
	}

	return &BacklogSummaryResult{
		Updated:   updated,
		Timestamp: today,
		TodoCount: todoCount,
	}, nil
}

// ---------------------------------------------------------------------------
// SPRINT.md update
// ---------------------------------------------------------------------------

var (
	reStatusTodo       = regexp.MustCompile(`\|\s*todo\s*\|`)
	reStatusInProgress = regexp.MustCompile(`\|\s*in-progress\s*\|`)
	reStatusDone       = regexp.MustCompile(`\|\s*done\s*\|`)
)

// UpdateSprintMD adds a row to the SPRINT.md Task list table or updates
// the existing status. Silently skips when SPRINT.md is missing.
//
// New row format (matches RenderSprintMD, 7 columns):
//
//	| {taskID} | {title} | {taskType} | {estimate} | {priority} | {status} | {dependsOn} |
//
// dependsOn is provided by the caller as a serialised string from a slice
// like ["T001","T002"] -> "T001, T002" (or "—" when empty). Empty values
// render as "—".
//
// Existing-row update: only the status column is replaced via regex
// (priority/depends_on changes are not this function's responsibility and
// belong to task_update etc.).
//
// Resolves the 5/7-column omission bug.
func UpdateSprintMD(sprintID, taskID, title, taskType, estimate, priority, status, dependsOn string) error {
	sprintDir, _, err := FindSprintDir(sprintID)
	if err != nil {
		return nil
	}
	sprintPath := filepath.Join(sprintDir, "SPRINT.md")
	if _, err := os.Stat(sprintPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(sprintPath)
	if err != nil {
		return fmt.Errorf("read SPRINT.md: %w", err)
	}
	lines := splitLines(string(data))

	// Update an existing row when task_id is already present.
	// Only the status column is replaced — the status label pattern is the
	// same whether the header has 5 or 7 columns, so it is safe.
	//
	// Cascade-drift guard: match rows strictly.
	// Using strings.Contains("| TID |") alone would also match other
	// rows whose depends_on cell ends with "| ... | TID |", overwriting
	// the wrong row's status (cascade evidence reproduced in tests).
	// Solution: confirm the first cell is exactly taskID.
	marker := fmt.Sprintf("| %s |", taskID)
	firstCell := fmt.Sprintf("| %s ", taskID) // first-cell prefix pattern (| space taskID space)
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		// Confirm the first cell is taskID (starts with "| TNNN" followed by ' ' + '|' or '|').
		if !strings.HasPrefix(trimmed, firstCell) && !strings.HasPrefix(trimmed, marker) {
			continue
		}
		updated := reStatusTodo.ReplaceAllString(line, fmt.Sprintf("| %s |", status))
		updated = reStatusInProgress.ReplaceAllString(updated, fmt.Sprintf("| %s |", status))
		updated = reStatusDone.ReplaceAllString(updated, fmt.Sprintf("| %s |", status))
		lines[i] = updated
		return os.WriteFile(sprintPath, []byte(joinLines(lines)), 0o644)
	}

	// Add a new row — insert after the Task table header.
	// Uses the same 7-column format as RenderSprintMD.
	depsField := dependsOn
	if strings.TrimSpace(depsField) == "" {
		depsField = "—"
	}
	priorityField := priority
	if strings.TrimSpace(priorityField) == "" {
		priorityField = "p2"
	}
	newRow := fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |",
		taskID, title, taskType, estimate, priorityField, status, depsField)
	tableHeaderIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "| ID |") {
			if tableHeaderIdx < 0 {
				tableHeaderIdx = i
			}
		}
		if tableHeaderIdx >= 0 && strings.Contains(line, "|-") {
			tableHeaderIdx = i + 1
			break
		}
	}

	if tableHeaderIdx >= 0 {
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:tableHeaderIdx]...)
		newLines = append(newLines, newRow)
		newLines = append(newLines, lines[tableHeaderIdx:]...)
		lines = newLines
	} else {
		lines = append(lines, newRow)
	}

	return os.WriteFile(sprintPath, []byte(joinLines(lines)), 0o644)
}

// RemoveRowFromSprintMD removes the row for a given Task from a Sprint's
// SPRINT.md (compensating-transaction usage).
// Returns true on removal, false when not found.
func RemoveRowFromSprintMD(sprintID, taskID string) (bool, error) {
	sprintDir, _, err := FindSprintDir(sprintID)
	if err != nil {
		return false, nil
	}
	sprintPath := filepath.Join(sprintDir, "SPRINT.md")
	if _, err := os.Stat(sprintPath); os.IsNotExist(err) {
		return false, nil
	}

	data, err := os.ReadFile(sprintPath)
	if err != nil {
		return false, fmt.Errorf("read SPRINT.md: %w", err)
	}
	lines := splitLines(string(data))
	marker := fmt.Sprintf("| %s |", taskID)
	newLines := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		if strings.Contains(line, marker) {
			removed = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !removed {
		return false, nil
	}
	if err := os.WriteFile(sprintPath, []byte(joinLines(newLines)), 0o644); err != nil {
		return false, fmt.Errorf("write SPRINT.md: %w", err)
	}
	return true, nil
}

// UpdateCeremonyCheckbox + harnessItemToCeremonyPhase removed. CEREMONY.md
// retired — the harness_items DB is the SSOT for ceremony state. The
// caller in harness.go was removed at the same time.

// ---------------------------------------------------------------------------
// Append Task status-change history
// ---------------------------------------------------------------------------

// statusHistoryHeading is the heading of the Task file's status-change history section.
// Avoiding substring false positives requires line-based exact matching.
const statusHistoryHeading = "## Status change history"

// hasStatusHistorySection reports whether content has a status-change-history section.
// strings.Contains may match `### Status change history` (level-3) as a
// substring and produce false positives, so split into lines, TrimSpace
// each, and compare exactly.
func hasStatusHistorySection(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == statusHistoryHeading {
			return true
		}
	}
	return false
}

// AppendStatusHistory appends/updates the '## Status change history' table
// at the bottom of the Task file. Creates the section when missing,
// otherwise appends a row.
func AppendStatusHistory(filePath, fromStatus, toStatus string, reason string) error {
	now := time.Now().Format("2006-01-02")
	newRow := fmt.Sprintf("| %s | %s -> %s | %s |", now, fromStatus, toStatus, reason)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	content := string(data)

	if hasStatusHistorySection(content) {
		// Append after the last row of the existing table.
		lines := strings.Split(content, "\n")
		insertIdx := -1
		inHistory := false
		for i, line := range lines {
			if strings.TrimSpace(line) == statusHistoryHeading {
				inHistory = true
			} else if inHistory && strings.HasPrefix(line, "|") {
				insertIdx = i + 1
			} else if inHistory && !strings.HasPrefix(line, "|") && strings.TrimSpace(line) != "" {
				break
			}
		}
		if insertIdx >= 0 {
			newLines := make([]string, 0, len(lines)+1)
			newLines = append(newLines, lines[:insertIdx]...)
			newLines = append(newLines, newRow)
			newLines = append(newLines, lines[insertIdx:]...)
			lines = newLines
		} else {
			lines = append(lines, newRow)
		}
		return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o644)
	}

	// Create a new section.
	section := "\n\n" + statusHistoryHeading + "\n\n| When | Change | Reason |\n|------|--------|--------|\n" + newRow + "\n"
	return os.WriteFile(filePath, []byte(strings.TrimRight(content, "\n")+section), 0o644)
}

// ---------------------------------------------------------------------------
// Internal utilities
// ---------------------------------------------------------------------------

// splitLines splits a string into lines, dropping trailing newline characters.
func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	// Empty entry not appended when the source ends with a newline; joinLines restores it.
	return lines
}

// joinLines is the inverse of splitLines.
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// NormalizeSprintField normalises various "unassigned" representations of
// the frontmatter sprint field into the empty string. "backlog", "~",
// "null", "<nil>", "" -> "". Other values are returned unchanged and used
// as the actual sprint ID.
//
// To avoid the pkg/backlog <-> pkg/task import cycle, lifted into fileutil
// for shared use. pkg/backlog.NormalizeSprintField and the inline switches
// in pkg/task become thin wrappers or direct calls into this function.
func NormalizeSprintField(s string) string {
	switch s {
	case "", "backlog", "~", "null", "<nil>":
		return ""
	}
	return s
}
