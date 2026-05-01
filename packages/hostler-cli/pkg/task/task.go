// Package task owns Task CRUD and status-transition logic.
// It provides an atomic workflow: ID allocation -> file creation ->
// BACKLOG/SPRINT update -> DB insert -> audit log.
// trac: WRK-CM003,WRK-EV001,WRK-EV002,WRK-EV003,WRK-EV004,WRK-FT001
// trac: WRK-FT002,WRK-FT003,WRK-FT004,WRK-FT005,WRK-FT006,WRK-FT007,WRK-QR009
package task

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/events"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/gatejudgement"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/harness"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/id"
	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/migration"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/template"
)

// taskLogger — package-level logger via pkg/log.
var taskLogger = pkglog.NewDefault()

// --------------------------------------------------------------------------
// Constants / shared mappings
// --------------------------------------------------------------------------

// typePrefix maps Task type to commit-message prefix.
var typePrefix = map[string]string{
	"feature":  "feat",
	"bugfix":   "fix",
	"hotfix":   "hotfix",
	"docs":     "docs",
	"refactor": "refactor",
	"infra":    "infra",
	"test":     "test",
	"chore":    "chore",
	"spike":    "spike",
}

// pseudoSprints lists values that represent the "no sprint assigned" state.
// escape hatch env var that disables the guard preventing placeholder
// Tasks from being assigned to a Sprint. Only used in emergencies (marker
// false-positives). Disables the guard only when set to 1/true.
const envAllowPlaceholderKey = "HSTL_ASSIGN_ALLOW_PLACEHOLDER"

func allowPlaceholderAssign() bool {
	v := envalias.Lookup("ASSIGN_ALLOW_PLACEHOLDER")
	return v == "1" || strings.EqualFold(v, "true")
}

var pseudoSprints = map[string]bool{
	"":        true,
	"backlog": true,
	"~":       true,
}

// --------------------------------------------------------------------------
// ResolveTaskPath helper
// --------------------------------------------------------------------------

// ResolveTaskPath resolves the Task file via DB file_path, falling back to
// glob lookup when missing. When exactly one candidate is found, the DB
// file_path is updated automatically.
// when DB file_path points into archive/ (read-only), the
// transition cannot succeed there, so the active area fallback is used
// immediately. When the same ID exists in both archive/ and active, active
// is the SSOT.
// trac: WRK-QR001
func ResolveTaskPath(taskID, dbFilePath string) (string, error) {
	root := fileutil.GetProjectRoot()
	absPath := filepath.Join(root, dbFilePath)
	archivePrefix := filepath.Join(root, "archive") + string(filepath.Separator)
	dbPathInArchive := strings.HasPrefix(absPath+string(filepath.Separator), archivePrefix) || absPath == filepath.Join(root, "archive")

	// 1. Direct lookup via the DB path — skip when it points into archive/ .
	if !dbPathInArchive {
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}
	}

	// 2. glob fallback: works/**/T{id}-*.md
	pattern := filepath.Join(root, "works", "**", taskID+"-*.md")
	candidates, err := filepath.Glob(pattern)
	if err != nil || len(candidates) == 0 {
		// Recursive scan of subdirectories (: excludes archive/).
		candidates = findTaskFiles(root, taskID)
	}

	if len(candidates) == 1 {
		resolved := candidates[0]
		newRelPath := fileutil.ToRepoRelative(resolved)

		// Auto-update DB.
		if gs := store.Get(); gs != nil {
			_ = gs.UpdateTaskFilePath(taskID, newRelPath)
			_ = audit.LogEvent("task.path_auto_resolved", "task", taskID, "claude", map[string]any{
				"stale_path":    dbFilePath,
				"resolved_path": newRelPath,
			}, "")
		}
		return resolved, nil
	}

	return "", fmt.Errorf("task file not found: %s", taskID)
}

// formatDependsOnSlice formats a dependsOn slice for the SPRINT.md table
// column.
// Empty slice -> "—" (em dash)
// Single item -> ""
// Many items -> ", , "
// Caller: used when dependsOn already exists as []string in memory, e.g.
// immediately after Task creation.
func formatDependsOnSlice(deps []string) string {
	if len(deps) == 0 {
		return "—"
	}
	return strings.Join(deps, ", ")
}

// formatDependsOnForTable converts the DB cache's depends_on JSON string into
// the SPRINT.md table column representation.
// Input: JSON string of the form `["",""]` (or empty array `[]`,
// NULL -> "").
// Output: same rules as formatDependsOnSlice ("—" / "" / ", ").
// On parse failure, falls back safely to empty result ("—") so table rendering
// never breaks.
func formatDependsOnForTable(jsonStr string) string {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" || jsonStr == "[]" || jsonStr == "null" {
		return "—"
	}
	var deps []string
	if err := json.Unmarshal([]byte(jsonStr), &deps); err != nil {
		return "—"
	}
	return formatDependsOnSlice(deps)
}

// findTaskFiles recursively scans for markdown files whose name starts with
// taskID under root.
// the archive/ directory (external-project snapshots /
// legacy preservation) is read-only and is excluded from the walk. This
// prevents the defect (reported in dog food §2.4) where, when an
// active Task and an archive snapshot share an ID, transition commands
// matched the archive copy and corrupted file_path.
// works/ first-matching is safe — works/ is the only active work area, and
// archive/backups/ holds external-project snapshots where duplicate Task
// IDs are common.
func findTaskFiles(root, taskID string) []string {
	var found []string
	prefix := taskID + "-"
	archivePrefix := filepath.Join(root, "archive") + string(filepath.Separator)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// skip archive/ at the directory level (more efficient).
		if d.IsDir() {
			if path == filepath.Join(root, "archive") || strings.HasPrefix(path+string(filepath.Separator), archivePrefix) {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".md") {
			found = append(found, path)
		}
		return nil
	})
	return found
}

// --------------------------------------------------------------------------
// Create
// --------------------------------------------------------------------------

// Create creates a Task atomically.
// ID allocation -> file creation -> BACKLOG/SPRINT update -> DB insert ->
// audit log.
func Create(title, taskType, sprint, priority, estimate, summary string, dependsOn []string) (result any, returnErr error) {
	if dependsOn == nil {
		dependsOn = []string{}
	}
	if priority == "" {
		priority = "p2"
	}
	if estimate == "" {
		estimate = "M"
	}

	// normalize the sprint parameter.
	// "backlog" / "~" / "null" / "<nil>" all mean unassigned (=BACKLOG).
	// Without normalisation, (1) the file frontmatter records the literal
	// "backlog", (2) the DB stores either NULL or the literal "backlog", and
	// (3) backlog_sync then reports a spurious "DB sprint=... vs file
	// sprint=..." mismatch. After normalisation everything routes through
	// the single canonical form sprint=="".
	// the SSOT is fileutil.NormalizeSprintField. It
	// replaces the inline switch from , removing duplication while
	// keeping the package import cycle in check.
	sprint = fileutil.NormalizeSprintField(sprint)

	root := fileutil.GetProjectRoot()
	today := time.Now().Format("2006-01-02")

	// 0. Bootstrap Gate
	bootstrap, err := migration.CheckBootstrapNeeded(root)
	if err != nil {
		return nil, fmt.Errorf("bootstrap check failed: %w", err)
	}
	if bootstrap != nil {
		return migration.BuildBootstrapResponse(bootstrap, root), nil
	}

	// title-duplicate detection guard — prevents recurrence
	// of the downstream duplicate-create pattern. Run before ID issuance to
	// avoid burning the counter. Bypass via env var
	// HSTL_TASK_CREATE_ALLOW_DUPLICATE_TITLE=1 or the CLI
	// -allow-duplicate-title flag (the CLI maps the flag to that env var).
	if dupErr := CheckDuplicateTitleOrReject(title); dupErr != nil {
		return nil, dupErr
	}

	// issue a PENDING work ticket — even before a Task ID
	// is allocated, this lets every failure path (file/DB/rollback) be traced
	// in audit_log under a single ticket. On success the ticket is promoted
	// to GenerateWorkTicket(taskID) and the final value is stored in
	// tasks.work_ticket. The PENDING ticket itself is never written to the
	// tasks table; only audit_log retains a record of it.
	pendingTicket, ptErr := db.GeneratePendingTicket()
	if ptErr != nil {
		// crypto/rand failure is an extremely rare system-level error. Since
		// even audit logging is unavailable, return early.
		return nil, fmt.Errorf("pending work ticket generation failed: %w", ptErr)
	}

	// 1. Atomic ID allocation.
	taskID, err := id.TaskIDNext()
	if err != nil {
		_ = audit.LogEvent("task.create_failed", "task", "", "claude", map[string]any{
			"pending_ticket": pendingTicket,
			"failure_reason": "id_allocation_failed",
			"title":          title,
			"type":           taskType,
			"error":          err.Error(),
		}, "")
		return nil, fmt.Errorf("task ID allocation failed: %w", err)
	}

	// failure-path rollback — when any step after step 2
	// fails, remove the orphan artefacts (file / DB row) created earlier. ID
	// counter gaps are allowed (only monotonicity is guaranteed). Best-effort
	// cleanup aligned with the 2026-04-17 orphan-recurrence prevention
	// and the user principle "do not consume an ID when create fails".
	// after rollback, log PENDING ticket + failure_reason
	// to audit_log so the cause of the failure is traceable.
	var (
		filePathCreated string
		dbInserted      bool
		failureReason   string // step-by-step failure cause — used by defer audit.
	)
	defer func() {
		if returnErr == nil {
			return
		}
		// Rollback order: DB -> file (DB is SSOT, so deleting it first keeps
		// the same taskID reusable on a subsequent call).
		if dbInserted {
			if gs := store.Get(); gs != nil {
				_ = gs.DeleteTask(taskID)
			}
		}
		if filePathCreated != "" {
			_ = os.Remove(filePathCreated)
		}
		// audit the create failure (after rollback completes).
		if failureReason == "" {
			failureReason = "unknown"
		}
		_ = audit.LogEvent("task.create_failed", "task", taskID, "claude", map[string]any{
			"pending_ticket": pendingTicket,
			"failure_reason": failureReason,
			"title":          title,
			"type":           taskType,
			"rollback_file":  filePathCreated != "",
			"rollback_db":    dbInserted,
			"error":          returnErr.Error(),
		}, "")
	}()

	// 2. Render the markdown content.
	content := template.RenderTaskTemplate(taskID, title, taskType, sprint, priority, estimate, dependsOn, summary)

	// 3. Create the file.
	filePath, err := fileutil.CreateTaskFile(taskID, title, taskType, sprint, priority, estimate, dependsOn, content)
	if err != nil {
		failureReason = "file_create_failed"
		return nil, fmt.Errorf("task file creation failed: %w", err)
	}
	filePathCreated = filePath
	relPath := fileutil.ToRepoRelative(filePath)

	// 4. Update BACKLOG.md or SPRINT.md (non-fatal — silent).
	if sprint == "" {
		_ = fileutil.UpdateBacklogMD(taskID, title, taskType, estimate, priority)
		_, _ = fileutil.RebuildBacklogSummary()
	} else {
		dependsOnDisplay := formatDependsOnSlice(dependsOn)
		_ = fileutil.UpdateSprintMD(sprint, taskID, title, taskType, estimate, priority, "todo", dependsOnDisplay)
	}

	// 5. DB cache insert.
	gs, gsErr := store.MustGet()
	if gsErr != nil {
		failureReason = "store_not_initialized"
		return nil, fmt.Errorf("DB is not initialised")
	}

	dependsOnJSON, _ := json.Marshal(dependsOn)

	if err = gs.CreateTask(&ports.TaskRecord{
		TaskID:    taskID,
		Title:     title,
		Type:      taskType,
		Status:    "todo",
		Priority:  priority,
		Estimate:  estimate,
		Sprint:    sprint,
		FilePath:  relPath,
		DependsOn: string(dependsOnJSON),
		CreatedAt: today,
	}); err != nil {
		failureReason = "db_insert_failed"
		return nil, fmt.Errorf("task DB insert failed: %w", err)
	}
	dbInserted = true

	// promote PENDING -> WT-T{id}-{hex}.
	// Start's existing logic only issues a ticket when "existing == ''",
	// so promoting in Create makes Start naturally recognise it as
	// pre-existing and skip issuance. Promotion failure (e.g. crypto/rand
	// outage) is non-fatal — the Task is already in the DB and Start will
	// retry. We just record the event in audit.
	promotedTicket, promErr := db.GenerateWorkTicket(taskID)
	if promErr == nil {
		if err := gs.SetTaskWorkTicket(taskID, promotedTicket); err != nil {
			// DB write failure is non-fatal — also skip the frontmatter update.
			_ = audit.LogEvent("task.work_ticket_promotion_failed", "task", taskID, "claude", map[string]any{
				"pending_ticket": pendingTicket,
				"reason":         "db_set_failed",
				"error":          err.Error(),
			}, "")
			promotedTicket = ""
		} else {
			_ = writeWorkTicketToFrontmatter(taskID, promotedTicket)
		}
	} else {
		_ = audit.LogEvent("task.work_ticket_promotion_failed", "task", taskID, "claude", map[string]any{
			"pending_ticket": pendingTicket,
			"reason":         "generate_failed",
			"error":          promErr.Error(),
		}, "")
		promotedTicket = ""
	}

	// 6. Audit log — : record both PENDING and the promoted final ticket
	// so the mapping is traceable.
	_ = audit.LogEvent("task.created", "task", taskID, "claude", map[string]any{
		"title":          title,
		"type":           taskType,
		"sprint":         sprint,
		"priority":       priority,
		"estimate":       estimate,
		"depends_on":     dependsOn,
		"file_path":      relPath,
		"pending_ticket": pendingTicket,
		"work_ticket":    promotedTicket,
	}, "")

	return &domain.CreateResult{
		TaskID:    taskID,
		FilePath:  relPath,
		CreatedAt: today,
		Status:    "created",
		// ( session feedback): show a body-authoring reminder
		// right after creation. Prevents the common mistake where an AI
		// creates a Task and skips writing the body.
		Warnings: []string{
			fmt.Sprintf("Task %s created. Before starting implementation, fill in the Purpose / Requirements / Done Criteria / Type sections of the Task file (%s). Calling task_start while the body is a placeholder triggers a design-readiness warning.", taskID, relPath),
		},
	}, nil
}

// --------------------------------------------------------------------------
// Start
// --------------------------------------------------------------------------

// Start performs the todo -> in-progress status transition.
// trac: HAR-CM009
func Start(taskID string) (any, error) {
	result, err := transitionTask(taskID, "todo", "in-progress", "task.transitioned")
	if err != nil {
		return nil, fmt.Errorf("Start: task status transition failed (%s): %w", taskID, err)
	}
	tr, ok := result.(*domain.TransitionResult)
	if !ok {
		return result, nil
	}
	reminders := config.GetReminders(string(events.EventTaskStart))
	if len(reminders) > 0 {
		tr.Reminders = reminders
	}
	// Placeholder body warning.
	if HasPlaceholderBody(taskID) {
		tr.Warning = fmt.Sprintf("Task %s body is in a placeholder state. Fill in Purpose / Requirements / Done Criteria before implementing.", taskID)
	}
	// issue work ticket + record in DB/frontmatter.
	if gs := store.Get(); gs != nil {
		existing, _ := gs.GetTaskWorkTicket(taskID)
		if existing == "" {
			ticket, err := db.GenerateWorkTicket(taskID)
			if err == nil {
				if err := gs.SetTaskWorkTicket(taskID, ticket); err == nil {
					tr.WorkTicket = ticket
					_ = writeWorkTicketToFrontmatter(taskID, ticket)
				}
			}
		} else {
			tr.WorkTicket = existing
		}
	}
	tr.SuggestedNextAction = fmt.Sprintf("after implementation completes call task_complete(task_id='%s')", taskID)

	// re-validate body claims. Re-run whitelisted commands
	// (grep/wc/ls/rg/find) inside ```bash code fences in the Task body and
	// emit a WARN if there is drift between the body's recorded "N matches"
	// and the live result. Opt out via env HSTL_TASK_START_BODY_REVALIDATE=off.
	if gs := store.Get(); gs != nil {
		if fp, err := gs.GetTaskFilePath(taskID); err == nil && fp != "" {
			if abs, err := ResolveTaskPath(taskID, fp); err == nil {
				if warns := RevalidateTaskBody(abs); len(warns) > 0 {
					tr.ClaimDriftWarnings = warns
				}
			}
		}
	}

	// (): auto-reconcile file_path drift at task start.
	// () only covered task complete — this fills the gap.
	// Graceful — failure does not affect the task-start flow.
	if gs := store.Get(); gs != nil {
		if details, derr := gs.GetTaskFull(taskID); derr == nil && details != nil && details.Sprint != "" {
			reconcileTaskPathsForSprint(details.Sprint)
		}
	}

	return tr, nil
}

// writeWorkTicketToFrontmatter records work_ticket in the Task file's
// frontmatter . File read/write failure is non-fatal — DB is the SSOT,
// so the failure is logged silently.
func writeWorkTicketToFrontmatter(taskID, ticket string) error {
	gs := store.Get()
	if gs == nil {
		return fmt.Errorf("store nil")
	}
	fp, err := gs.GetTaskFilePath(taskID)
	if err != nil || fp == "" {
		return err
	}
	abs, err := ResolveTaskPath(taskID, fp)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("frontmatter missing")
	}
	end := strings.Index(content[4:], "\n---\n")
	if end < 0 {
		return fmt.Errorf("frontmatter end marker missing")
	}
	fm := content[4 : 4+end]
	rest := content[4+end+5:]
	// Strip any existing work_ticket line, then append.
	var newLines []string
	for _, line := range strings.Split(fm, "\n") {
		if strings.HasPrefix(line, "work_ticket:") {
			continue
		}
		newLines = append(newLines, line)
	}
	newLines = append(newLines, fmt.Sprintf("work_ticket: %s", ticket))
	newContent := "---\n" + strings.Join(newLines, "\n") + "\n---\n" + rest
	return os.WriteFile(abs, []byte(newContent), 0o644)
}

// --------------------------------------------------------------------------
// Checkpoint
// --------------------------------------------------------------------------

// CheckpointResult alias removed. Callers reference
// domain.CheckpointResult directly.

// Checkpoint issues a fresh work ticket for an in-progress Task (
// ). Acknowledgements against the previous ticket are invalidated
// automatically because tasks.work_ticket is rotated to the new ticket and
// HasContextAck lookups no longer match (no separate supersede table is
// needed). Usage scenario: in the middle of a long-running Task, re-confirm
// "what has been done / what comes next" and require a new ack to keep the
// context fresh.
// reason is optional (used in the audit log). The call is rejected when the
// Task is not in-progress (checkpoint is only meaningful for running Tasks).
// trac: HAR-CM012
func Checkpoint(taskID, reason string) (*domain.CheckpointResult, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("store not initialised")
	}

	// Task existence + status check.
	details, err := gs.GetTaskDetails(taskID)
	if err != nil || details == nil {
		return nil, &apperr.NotFoundError{EntityType: "Task", EntityID: taskID}
	}
	if details.Status != "in-progress" {
		return nil, &apperr.RejectedError{
			Reason: fmt.Sprintf(
				"checkpoint is only valid on in-progress Tasks. Current status=%s. Run 'hstl task start %s' first or reopen the Task",
				details.Status, taskID,
			),
		}
	}

	oldTicket, _ := gs.GetTaskWorkTicket(taskID)
	newTicket, err := db.GenerateWorkTicket(taskID)
	if err != nil {
		return nil, fmt.Errorf("new ticket generation failed: %w", err)
	}
	if err := gs.SetTaskWorkTicket(taskID, newTicket); err != nil {
		return nil, fmt.Errorf("ticket swap failed: %w", err)
	}
	_ = writeWorkTicketToFrontmatter(taskID, newTicket)

	if reason == "" {
		reason = "checkpoint"
	}

	return &domain.CheckpointResult{
		TaskID:    taskID,
		OldTicket: oldTicket,
		NewTicket: newTicket,
		Reason:    reason,
	}, nil
}

// --------------------------------------------------------------------------
// Complete
// --------------------------------------------------------------------------

// Complete performs the in-progress -> done status transition, including
// the Harness Gate.
// _taskIDRefRe matches Task ID references.
var _taskIDRefRe = regexp.MustCompile(`T\d{3,}`)

// _followupSectionKeywords — keywords that identify a "follow-up" section.
var _followupSectionKeywords = []string{"unimplemented", "follow-up", "Followup", "TODO", "Follow-up Tasks", "incomplete"}

// _backtickPathRe extracts file paths wrapped in backticks. Examples:
// `mcp-server/pkg/foo.go` or `path/to/file.md`.
var _backtickPathRe = regexp.MustCompile("`([^`]+)`")

// _fileExtRe validates a file path extension (.ext form, 1–7 lowercase
// alphanumerics). The draft was case-insensitive; tightens it to
// lowercase only because file extensions are conventionally lowercase, while
// upper-case-leading tokens are usually class/method names (e.g. .DoWork in
// MyClass.DoWork) and must be distinguished.
var _fileExtRe = regexp.MustCompile(`\.[a-z0-9]{1,7}$`)

// _knownFileExts is the whitelist-based extension set .
// Lists extensions for major development languages, document formats, and
// configuration files. Extensions not in the whitelist are excluded from
// path candidates.
var _knownFileExts = map[string]bool{
	// Go / Rust / C / C++
	".go": true, ".rs": true, ".c": true, ".h": true, ".cc": true, ".cpp": true, ".hpp": true,
	// .NET / JVM
	".cs": true, ".fs": true, ".vb": true, ".java": true, ".kt": true, ".scala": true, ".groovy": true,
	// JS / TS / Web
	".js": true, ".ts": true, ".jsx": true, ".tsx": true, ".mjs": true, ".cjs": true,
	".vue": true, ".svelte": true, ".html": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
	// Python / Ruby / PHP / Lua / Dart
	".py": true, ".pyi": true, ".rb": true, ".php": true, ".lua": true, ".dart": true,
	// Docs / markup
	".md": true, ".mdx": true, ".rst": true, ".adoc": true, ".txt": true, ".tex": true,
	// Config / data
	".yaml": true, ".yml": true, ".json": true, ".toml": true, ".xml": true, ".ini": true, ".conf": true, ".cfg": true, ".env": true, ".properties": true,
	// Scripts / build
	".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".ps1": true, ".bat": true, ".cmd": true, ".mk": true,
	// DB / query / schema
	".sql": true, ".graphql": true, ".proto": true, ".csv": true, ".tsv": true,
	// Images / other resources (rarely, but referenced)
	".svg": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
	// Misc
	".lock": true, ".sum": true, ".mod": true, ".gitignore": true, ".dockerignore": true, ".editorconfig": true,
}

// isLikelyFilePath returns true when the token is likely a file path
// ( / / ).
// Conditions:
// 1. Has an extension — 1–7 *lowercase* alphanumerics after the last dot
// (.go/.py/.md, etc.).
// 2. Not a slash command or absolute path — does not start with "/".
// 3. Not a relative-path indicator or test-package argument — does not
// start with "./".
// 4. Not a template placeholder — no "{" or "}".
// 5. Not a slash-command namespace — no ":" (Windows paths do not appear
// in this project).
// 6. Passes the extension whitelist — present in _knownFileExts .
// 7. No glob wildcards — no "*" / "?" (; multi-file references such
// as `Strategy/SplitBuy*.cs` are excluded).
// Removes false positives observed in ~25:
// "list/single" / "delete/irreversible" -> excluded (no extension)
// "./internal/..." -> excluded (starts with ./)
// "/hstl:kb:init" -> excluded (leading / + colon)
// "commands/{namespace}/{action}.md" -> excluded (braces)
// "MyClass.DoWork" -> excluded (uppercase-leading extension)
// "obj.someMethod" -> excluded (.someMethod not in whitelist)
// "JSON.stringify" -> excluded (.stringify not in whitelist)
// "Strategy/SplitBuy*.cs" -> excluded (glob wildcard)
// "src/foo?.cs" -> excluded (single-char glob)
func isLikelyFilePath(s string) bool {
	if s == "" {
		return false
	}
	// allow absolute paths `/` through the parser and let
	// downstream (verifyTaskResultsForType) decide whether they sit outside
	// projectRoot. Excluding them at the parser level previously meant that
	// listing `/tmp/fake.txt` as an artefact bypassed verification.
	if strings.HasPrefix(s, "./") {
		return false
	}
	// exclude paths outside the repo.
	// In spike Tasks etc., references to files outside the repo (home
	// directory artefacts, parent directories, remote URLs) caused false
	// positive BLOCKs in the git-diff verification. Paths not tracked by the
	// repo are excluded from verification.
	if strings.HasPrefix(s, "~/") || strings.HasPrefix(s, "../") {
		return false
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return false
	}
	if strings.ContainsAny(s, "{}:") {
		return false
	}
	// exclude when glob wildcards are present (multi-file reference).
	if strings.ContainsAny(s, "*?") {
		return false
	}
	// Extension regex match + whitelist verification .
	match := _fileExtRe.FindString(s)
	if match == "" {
		return false
	}
	return _knownFileExts[match]
}

// _placeholderPaths are common placeholders (empty template residue) seen in
// the Result section. Excluded from verification.
var _placeholderPaths = map[string]bool{
	"path/to/file":  true,
	"path/to/file1": true,
	"path/to/file2": true,
	"(review result document or modified source file)": true,
	"…":   true,
	"...": true,
}

// verifyTaskResults verifies that the paths recorded in the Task file's
// Result section exist in git diff.
// The section-title list and policy are resolved through
// config.GetTaskResultSectionTitles / GetTaskResultCheckPolicy in the order:
// environment variable -> project-config.yaml -> defaults
// (["Result","Artifacts"], strict).
// Returns:
// missing: files absent from git diff or with mismatched paths.
// checked: every file path extracted from the Result section.
// sectionTitles: the recognised section titles (used in the response).
// Policy enforcement is handled by the caller (Complete, etc.).
// Introduced in . Replaces the earlier verifyArtifacts/parseArtifactSection.
// backward-compat wrapper. For taskType-aware behaviour use
// verifyTaskResultsForType.
func verifyTaskResults(taskID, filePath string) (missing []domain.MissingFile, checked []string, sectionTitles []string) {
	return verifyTaskResultsForType(taskID, filePath, "")
}

// verifyTaskResultsForType is the task-type-aware version of
// verifyTaskResults .
// When taskType is "infra", the Result-section parser operates in narrative
// default mode and only scans for file paths under explicit artifact
// sub-headings such as `### Artifacts`. Other types keep the prior
// backward-compat behaviour (full scan).
// Background: for infra Tasks (e.g. Dev-environment deployment verification)
// the artefact is essentially the verification record itself, so narrative
// backtick paths (e.g. the Task file's own path) caused false-positive
// BLOCKs in measurements.
func verifyTaskResultsForType(taskID, filePath, taskType string) (missing []domain.MissingFile, checked []string, sectionTitles []string) {
	sectionTitles = config.GetTaskResultSectionTitles()

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, sectionTitles
	}

	checked = parseTaskResultFilesForType(string(content), sectionTitles, taskType)
	if len(checked) == 0 {
		return nil, nil, sectionTitles
	}

	// exclude the Task file's own path from artefact scan.
	// If the body self-references (e.g. "Follow-up: see this Task document"),
	// the parser extracts that path and it usually appears in git diff so it
	// passes; but in special paths such as hotfix/retry it produces a
	// false-positive BLOCK. Explicit exclusion improves reliability and
	// removes the temptation to bypass.
	checked = excludeSelfReference(checked, filePath)
	if len(checked) == 0 {
		return nil, nil, sectionTitles
	}

	changedFiles := getGitChangedFiles()
	if len(changedFiles) == 0 {
		// When git diff is empty, verification is not possible — return an
		// empty result rather than treating everything as missing (avoids
		// false positives in special situations such as immediately after
		// commit).
		return nil, checked, sectionTitles
	}

	// Match using normalised paths — exact or suffix ( — absolute-path
	// support).
	projectRoot := fileutil.GetProjectRoot()
	normalizedChanged := make([]string, len(changedFiles))
	for i, c := range changedFiles {
		normalizedChanged[i] = filepath.Clean(c)
	}

	for _, artifact := range checked {
		normalizedArtifact := filepath.Clean(artifact)
		// if absolute, strip project root to get a repo-relative path.
		// absolute paths outside projectRoot (/tmp/...,
		// /var/...) could be matched accidentally by the basename fallback
		// (below). Classify them as missing explicitly to block the
		// strict-verification bypass.
		if filepath.IsAbs(normalizedArtifact) && projectRoot != "" {
			rel, err := filepath.Rel(projectRoot, normalizedArtifact)
			if err != nil || strings.HasPrefix(rel, "..") {
				// Absolute path outside projectRoot -> immediately missing
				// (blocks bypass).
				missing = append(missing, domain.MissingFile{
					Path:   artifact,
					Reason: "absolute path outside projectRoot — not allowed as artefact ",
				})
				continue
			}
			normalizedArtifact = filepath.Clean(rel)
		}
		found := false
		for _, changed := range normalizedChanged {
			if changed == normalizedArtifact {
				found = true
				break
			}
			// suffix match — e.g. "pkg/task/task.go" vs "mcp-server/pkg/task/task.go".
			if strings.HasSuffix(changed, "/"+normalizedArtifact) || strings.HasSuffix(normalizedArtifact, "/"+changed) {
				found = true
				break
			}
			// basename final fallback — match on filename alone
			// (lenient handling of sub-bullet / multi-line misreads).
			if filepath.Base(changed) == filepath.Base(normalizedArtifact) {
				found = true
				break
			}
		}
		if !found {
			// drift-less regenerated-file exception —
			// excluded from missing when it matches the config list by
			// suffix and the file actually exists.
			if isDriftLessRegenerated(normalizedArtifact, projectRoot) {
				continue
			}
			missing = append(missing, domain.MissingFile{
				Path:   artifact,
				Reason: "not in git diff (typo or file never created/modified)",
			})
		}
	}
	return missing, checked, sectionTitles
}

// isDriftLessRegenerated checks whether the artifact path matches an entry
// in config.GetTaskResultDriftLessPaths by suffix and the file actually
// exists (, ). When true, strict verification does not treat
// it as missing.
func isDriftLessRegenerated(artifactRel, projectRoot string) bool {
	patterns := config.GetTaskResultDriftLessPaths()
	if len(patterns) == 0 {
		return false
	}
	clean := filepath.Clean(artifactRel)
	matched := false
	for _, p := range patterns {
		pClean := filepath.Clean(p)
		if clean == pClean ||
			strings.HasSuffix(clean, "/"+pClean) ||
			strings.HasSuffix(pClean, "/"+clean) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	// File-existence check.
	abs := clean
	if !filepath.IsAbs(abs) && projectRoot != "" {
		abs = filepath.Join(projectRoot, clean)
	}
	if _, err := os.Stat(abs); err == nil {
		return true
	}
	return false
}

// excludeSelfReference removes the Task body file's own path from the
// artefact list when present . Comparison runs after
// filepath.Clean on both absolute and relative inputs; matches by identical
// basename + suffix are also treated as self-references.
// Call context:
// filePath: path to the Task body .md file (usually absolute or
// project-relative).
// paths: artefact-path list extracted by parseTaskResultFilesForType.
// Returns: a new slice containing only paths that do not match filePath.
// The matching rules mirror those of verifyTaskResultsForType's
// missing-detection (exact / suffix / basename) for symmetry.
func excludeSelfReference(paths []string, filePath string) []string {
	if len(paths) == 0 || filePath == "" {
		return paths
	}
	selfClean := filepath.Clean(filePath)
	// Also normalise to a project-root-relative path.
	if projectRoot := fileutil.GetProjectRoot(); projectRoot != "" && filepath.IsAbs(selfClean) {
		if rel, err := filepath.Rel(projectRoot, selfClean); err == nil && !strings.HasPrefix(rel, "..") {
			selfClean = filepath.Clean(rel)
		}
	}
	selfBase := filepath.Base(selfClean)

	out := make([]string, 0, len(paths))
	for _, p := range paths {
		pc := filepath.Clean(p)
		if pc == selfClean {
			continue
		}
		if strings.HasSuffix(selfClean, "/"+pc) || strings.HasSuffix(pc, "/"+selfClean) {
			continue
		}
		if filepath.Base(pc) == selfBase {
			continue
		}
		out = append(out, p)
	}
	return out
}

// _artifactSubHeadings is the set of sub-heading (### ) names inside the
// "## Result" section that are treated as actual artefact lists (
// ).
// When users structure the Result section with sub-headings, only sub-trees
// whose heading name is in this set are scanned for file paths. Any other
// sub-heading (e.g. "### Design Decisions", "### Verification") is treated
// as narrative and excluded from parsing.
// When a sub-heading not in this set is encountered, the parser switches to
// "narrative mode" and the body under it is excluded from path scanning.
// backward compat: if no sub-headings are used at all (everything is
// directly under "## Result"), the original behaviour scans the whole
// section.
var _artifactSubHeadings = map[string]bool{
	"Artifacts":     true,
	"Artifact":      true,
	"Changed files": true,
	"Changed Files": true,
	"Files":         true,
}

// parseTaskResultFiles extracts the list of file paths from the markdown
// section that matches one of sectionTitles.
// Introduced in . Since it is a wrapper around
// parseTaskResultFilesForType (the task-type-aware version). When taskType
// is unspecified, the full-scan behaviour is preserved (backward compat).
func parseTaskResultFiles(content string, sectionTitles []string) []string {
	return parseTaskResultFilesForType(content, sectionTitles, "")
}

// parseTaskResultFilesForType is the task-type-aware version of
// parseTaskResultFiles.
// Supported formats:
// Section entry: "## Result", "## Artifacts", "## Result", "## Artifact",
// etc. (the sectionTitles argument).
// Sub-sections: only sub-trees under a sub-heading
// present in `_artifactSubHeadings` (### Artifacts, ### Changed files,
// etc.) are scanned for file paths. Other sub-headings (### Design
// Decisions, ### Verification, etc.) are treated as narrative.
// Hyphen lists: "- `path/to/file` (description)",
// "- path/to/file — description", "- path/to/file".
// Markdown tables: "| `path` | change description |",
// "| path/to/file | ... |".
// Auto-extract backtick paths: paths inside `…` are preferred.
// Section end: the next "## ..." or "---" or EOF.
// Empty items, placeholders (`path/to/file`, etc.) and empty values are
// excluded from the result.
// **Per-type behaviour **:
// taskType == "infra": narrative default mode. Scanning is only enabled
// when an explicit artefact sub-heading (### Artifacts, etc.) is
// entered. Prevents false positives from narrative backtick paths such
// as the Task file itself.
// Otherwise (feature/bugfix/refactor/docs/test/chore/hotfix/empty
// string): the original "full scan" mode (backward compat). When
// sub-headings are used, the rule is applied.
// backward compat : Tasks that use no sub-headings still scan the
// whole section directly under "## Result" (infra is the exception).
func parseTaskResultFilesForType(content string, sectionTitles []string, taskType string) []string {
	if len(sectionTitles) == 0 {
		return nil
	}

	// Section-title matcher — normalised to the "## {title}" form.
	titleMap := make(map[string]bool, len(sectionTitles))
	for _, t := range sectionTitles {
		titleMap[strings.TrimSpace(t)] = true
	}

	var paths []string
	seen := make(map[string]bool)

	// scanner buffer sizes identified by the magic-number
	// audit. Initial 64 KiB, max 1 MiB — covers a realistic upper bound on
	// Task body line length.
	const (
		taskLineScannerInitKB int = 64
		taskLineScannerMaxKB  int = 1024
		kibiByte              int = 1024
	)
	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Buffer(make([]byte, 0, taskLineScannerInitKB*kibiByte), taskLineScannerMaxKB*kibiByte)
	inSection := false
	// sub-heading mode tracking:
	// "all" — initial value, before any sub-heading (backward
	// compat full scan).
	// "artifact" — under ### Artifacts/Changed files etc. (scan target).
	// "narrative" — under any other ### (scan excluded).
	const (
		modeAll       = "all"
		modeArtifact  = "artifact"
		modeNarrative = "narrative"
	)
	// infra type defaults to narrative mode.
	// Scanning is only enabled when an explicit artefact sub-heading is
	// entered.
	// spike type uses the same narrative default — its
	// outputs target docs/08-references/research/ and code artefacts are
	// not the primary deliverable.
	initialMode := modeAll
	if taskType == "infra" || taskType == "spike" {
		initialMode = modeNarrative
	}
	scanMode := initialMode

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Section-entry detection: "## title" pattern.
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			// If extra text follows the title, match by the first word
			// (e.g. "## Result summary" -> "Result").
			firstWord := heading
			if idx := strings.IndexAny(heading, " \t"); idx > 0 {
				firstWord = heading[:idx]
			}
			if titleMap[heading] || titleMap[firstWord] {
				inSection = true
				scanMode = initialMode // Reset to per-type initial mode on entering a new Result section .
				continue
			}
			// Encountering another ## section ends the scan.
			if inSection {
				break
			}
		}

		// Section-end detection: "---" rule (in body, not frontmatter).
		if inSection && trimmed == "---" {
			break
		}

		if !inSection {
			continue
		}

		// Sub-section (### ...) — : switch mode.
		if strings.HasPrefix(trimmed, "### ") {
			subHeading := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			// Match by first word (e.g. "### Artifacts (8 items)" -> "Artifacts").
			subFirstWord := subHeading
			if idx := strings.IndexAny(subHeading, " \t"); idx > 0 {
				subFirstWord = subHeading[:idx]
			}
			if _artifactSubHeadings[subHeading] || _artifactSubHeadings[subFirstWord] {
				scanMode = modeArtifact
			} else {
				scanMode = modeNarrative
			}
			continue
		}

		// Skip path scanning in narrative mode.
		if scanMode == modeNarrative {
			continue
		}

		// a nested bullet (>=2 leading whitespaces in the original
		// line) is treated as narrative. Prevents false positives where text
		// inside a sub-bullet description is misread as a path.
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent >= 2 && (strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")) {
			continue
		}

		// Attempt item extraction (modeAll or modeArtifact).
		extracted := extractPathsFromLine(trimmed)
		for _, p := range extracted {
			if p == "" || _placeholderPaths[p] {
				continue
			}
			if seen[p] {
				continue
			}
			seen[p] = true
			paths = append(paths, p)
		}
	}
	return paths
}

// _gitMvArrow detects "A → B" or "A -> B" move notation .
// When this pattern appears in a line, the left-hand path is treated as the
// source (deleted, absent from git diff) and only the right-hand path is
// kept as the final artefact.
var _gitMvArrow = regexp.MustCompile(`\s(?:→|->)\s`)

// stripMovedSource detects the "A.cs → B.cs" or "A.cs -> B.cs" pattern and
// returns only the right-hand side. If the pattern is absent, returns the
// input unchanged .
func stripMovedSource(line string) string {
	if m := _gitMvArrow.FindStringIndex(line); m != nil {
		return strings.TrimSpace(line[m[1]:])
	}
	return line
}

// extractPathsFromLine extracts file-path candidates from a single line.
// Order:
// 1. If a backtick-wrapped path (`path`) is present, prefer it.
// 2. For a markdown table row ("| ... | ...") extract a backtick path or
// plain path from the first column (between pipes).
// 3. For a hyphen list ("- ...") strip the hyphen, then take a backtick
// path or the first word/phrase as the path.
// 4. Otherwise, return empty.
// Backticks take precedence so "- `path` (description)" yields just `path`.
// for "A → B" / "A -> B" move notation, only the right-hand
// (destination) is used.
func extractPathsFromLine(line string) []string {
	if line == "" {
		return nil
	}
	// handle git mv arrow notation — strip the left-hand source.
	line = stripMovedSource(line)

	// Markdown table row: split on pipes, extract from the first data
	// column. The "|---|---|" separator row contains no data and is
	// excluded automatically.
	if strings.HasPrefix(line, "|") && strings.Contains(line[1:], "|") {
		// Exclude separator rows.
		if strings.Contains(line, "---") || strings.Contains(line, "===") {
			return nil
		}
		cells := strings.Split(line, "|")
		if len(cells) < 2 {
			return nil
		}
		// cells[0] is the empty string before the leading pipe; cells[1]
		// is the first column.
		first := strings.TrimSpace(cells[1])
		if backticks := _backtickPathRe.FindStringSubmatch(first); len(backticks) >= 2 {
			if isLikelyFilePath(backticks[1]) {
				return []string{backticks[1]}
			}
			return nil
		}
		// Without backticks, treat the entire first cell as a path
		// candidate (split on whitespace / parentheses).
		return splitPathCandidate(first)
	}

	// Hyphen list.
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		entry := strings.TrimSpace(line[2:])
		if backticks := _backtickPathRe.FindStringSubmatch(entry); len(backticks) >= 2 {
			if isLikelyFilePath(backticks[1]) {
				return []string{backticks[1]}
			}
			return nil
		}
		return splitPathCandidate(entry)
	}

	// Other lines: extract only backtick paths (prose mentions of paths).
	if matches := _backtickPathRe.FindAllStringSubmatch(line, -1); len(matches) > 0 {
		var out []string
		for _, m := range matches {
			if len(m) >= 2 && isLikelyFilePath(m[1]) {
				out = append(out, m[1])
			}
		}
		return out
	}

	return nil
}

// splitPathCandidate extracts the first path candidate from a string.
// Steps:
// If a separator " (", " — " or " - " is present, use only the prefix.
// Take only the first whitespace-delimited token as the path.
// Accept the result only when isLikelyFilePath passes .
func splitPathCandidate(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	// Drop everything after a separator.
	separators := []string{" (", " — ", " – ", " - "}
	for _, sep := range separators {
		if idx := strings.Index(s, sep); idx > 0 {
			s = s[:idx]
			break
		}
	}

	// First token only.
	if idx := strings.IndexAny(s, " \t"); idx > 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)

	// Three-condition gate: extension + exclusion of command /
	// placeholder forms .
	if !isLikelyFilePath(s) {
		return nil
	}
	return []string{s}
}

// defaultTaskResultDiffDepth is the default number of recent commits to
// inspect (walking back from HEAD) during Result-section verification
// . The principle is "1 Task = 1 Commit" but follow-up commits are
// allowed, so a margin is given. Override with env var
// HSTL_TASK_RESULT_DIFF_DEPTH.
const defaultTaskResultDiffDepth = 10

// taskResultDiffDepth returns the env-var override or the default.
func taskResultDiffDepth() int {
	if v := strings.TrimSpace(envalias.Lookup("TASK_RESULT_DIFF_DEPTH")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultTaskResultDiffDepth
}

// GetGitChangedFiles is the public wrapper for getGitChangedFiles .
// Provides a shared entry point so external packages (ceremony etc.) can
// reuse the same git invocation as Result-section verification, preventing
// drift between preview output and strict checks.
func GetGitChangedFiles() []string { return getGitChangedFiles() }

// getGitChangedFiles returns the list of changed files from `git diff`
// plus the most recent N commits' logs.
// the prior `git diff HEAD~1` only included the single
// directly-previous commit, so when a Task had several follow-up commits
// files from earlier commits were omitted. The function now unions
// (a) the working tree's staged + unstaged diff and (b) the most recent
// N commits' log --name-only. N = HSTL_TASK_RESULT_DIFF_DEPTH (default
// 10).
func getGitChangedFiles() []string {
	root := fileutil.GetProjectRoot()
	seen := make(map[string]bool)
	var files []string

	add := func(out []byte) {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || seen[line] {
				continue
			}
			seen[line] = true
			files = append(files, line)
		}
	}

	// (a) working tree: unstaged + staged
	if out, err := runGit(root, "diff", "--name-only"); err == nil {
		add(out)
	}
	if out, err := runGit(root, "diff", "--name-only", "--cached"); err == nil {
		add(out)
	}

	// (b) Most recent N commits — `git log --name-only --pretty=format: -N`.
	depth := taskResultDiffDepth()
	if out, err := runGit(root, "log", "--name-only", "--pretty=format:", fmt.Sprintf("-%d", depth)); err == nil {
		add(out)
	}

	return files
}

// runGit executes a git subcommand from the project root and returns the
// stdout bytes.
// force `-c core.quotepath=false` — git's default emits
// non-ASCII paths as octal escapes (e.g. "\354\234\274...") which produce
// false mismatches in Go string comparison. Root-fix for the
// defect in which task-complete strict verification of artefacts
// with non-ASCII paths was BLOCKED.
func runGit(root string, args ...string) ([]byte, error) {
	full := append([]string{"-c", "core.quotepath=false"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = root
	return cmd.Output()
}

// verifyFollowups inspects "Unimplemented" / "Follow-up" sections in the
// Task file and returns a warning if no Task ID reference (T\d+) is found.
func verifyFollowups(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	var warnings []string
	inFollowupSection := false
	sectionName := ""
	hasTaskRef := false
	sectionContent := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			// Validate when leaving a previous follow-up section.
			if inFollowupSection && !hasTaskRef && sectionContent != "" {
				warnings = append(warnings, fmt.Sprintf(
					"section '%s' has no follow-up Task ID reference (TID)", sectionName))
			}
			// Detect a new section — : word-boundary match avoids substring misfires.
			inFollowupSection = false
			for _, kw := range _followupSectionKeywords {
				if headingTitleContainsKeyword(trimmed, kw) {
					inFollowupSection = true
					sectionName = trimmed
					hasTaskRef = false
					sectionContent = ""
					break
				}
			}
			continue
		}

		if inFollowupSection {
			sectionContent += trimmed
			if _taskIDRefRe.MatchString(trimmed) {
				hasTaskRef = true
			}
		}
	}

	// Validate the final section.
	if inFollowupSection && !hasTaskRef && sectionContent != "" {
		warnings = append(warnings, fmt.Sprintf(
			"section '%s' has no follow-up Task ID reference (TID)", sectionName))
	}

	return warnings
}

const (
	// rollbackSectionMinChars — minimum body length for the Rollback
	// section. Anything below this is treated as "empty". Overridable via
	// the HSTL_ROLLBACK_MIN_CHARS env var.
	rollbackSectionMinChars = 20
)

// _rollbackSectionHeaders — headers recognised as a Rollback section
// (case-insensitive; trailing text is allowed).
var _rollbackSectionHeaders = []string{"rollback"}

// verifyRollbackSection checks that the Task file has a `## Rollback`
// section with sufficient content. Returns false + a reason when missing or
// too short.
func verifyRollbackSection(filePath string) (ok bool, reason string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Sprintf("could not read task file: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	inSection := false
	var body strings.Builder
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## ") {
			// End of section.
			if inSection {
				break
			}
			// Detect a new section (## Rollback, ## Rollback Plan, etc.).
			headerLower := strings.ToLower(strings.TrimPrefix(trimmed, "## "))
			for _, h := range _rollbackSectionHeaders {
				if strings.HasPrefix(headerLower, h) {
					inSection = true
					found = true
					break
				}
			}
			continue
		}

		if inSection {
			body.WriteString(trimmed)
		}
	}

	if !found {
		return false, "## Rollback section missing"
	}
	bodyContent := strings.TrimSpace(body.String())
	if len(bodyContent) < rollbackSectionMinChars {
		return false, fmt.Sprintf(
			"## Rollback section body is too short (%d chars, %d required)",
			len(bodyContent), rollbackSectionMinChars)
	}
	return true, ""
}

// task_complete may be invoked from either todo or in-progress.
// When the current status is todo, the function performs an internal
// cascade todo -> in-progress -> done. Previously only in-progress was
// accepted, which caused either an InvalidStateError when callers skipped
// task_start, or partial completes left in an intermediate state
// (ISS-20260412-002 Bug A).
const (
	taskStatusTodo       = "todo"
	taskStatusInProgress = "in-progress"
	taskStatusDone       = "done"

	eventTaskCascadeStarted    = "task.cascade_started"
	eventTaskCascadeRolledBack = "task.cascade_rolled_back"
)

// Complete performs the todo -> in-progress -> done cascade transition
// (including Harness Gate verification and Result-section verification).
// trac: HAR-CM010
func Complete(taskID string, skipHarness bool) (any, error) {
	// Result-section verification result (filled before
	// transitionTask, attached after a successful transitionTask).
	var completeVerificationResult *domain.VerificationResult

	// query the current status first to decide whether to cascade.
	// Harness / hotfix / Result-section checks are status-agnostic, so they
	// run before the transition. The cascade todo -> in-progress -> done
	// only proceeds after every check has passed.
	currentDBStatus := ""
	if gs := store.Get(); gs != nil {
		currentDBStatus, _ = gs.GetTaskStatus(taskID)
	}
	if currentDBStatus != taskStatusTodo && currentDBStatus != taskStatusInProgress {
		return nil, &apperr.InvalidStateError{
			EntityType:    "task",
			EntityID:      taskID,
			CurrentState:  currentDBStatus,
			ExpectedState: taskStatusTodo + "|" + taskStatusInProgress,
			Message: fmt.Sprintf(
				"status transition not allowed: current=%s, expected=todo or in-progress",
				currentDBStatus,
			),
			RecoveryHint: "task_complete only accepts Tasks in todo or in-progress. Use task_reopen if it is already done.",
		}
	}

	if !skipHarness {
		// Auto-verify done-criteria checkboxes.
		criteria, _ := harness.AutoCheckCriteria(taskID)

		// Harness Gate verification.
		harnessResult, err := harness.CheckHarness("task", taskID)
		if err != nil {
			return nil, fmt.Errorf("harness verification failed: %w", err)
		}

		if harnessResult.Blocked {
			uncheckedItems := make([]apperr.UncheckedItem, 0, len(harnessResult.UncheckedRequired))
			for _, item := range harnessResult.UncheckedRequired {
				uncheckedItems = append(uncheckedItems, apperr.UncheckedItem{
					ID:       item.ID,
					Name:     item.Name,
					Required: item.Required,
				})
			}

			auditItems := make([]map[string]any, 0, len(uncheckedItems))
			for _, item := range uncheckedItems {
				auditItems = append(auditItems, map[string]any{
					"id":       item.ID,
					"name":     item.Name,
					"required": item.Required,
				})
			}
			_ = audit.LogEvent("harness.blocked", "task", taskID, "claude", map[string]any{
				"unchecked_required": auditItems,
				"criteria_status":    criteria,
			}, "")

			// include task-type info + harness_get pre-call guidance
			// in the BLOCKED response. Reduces cases where AIs use the
			// wrong feature-pattern item_id on non-feature types
			// (infra / hotfix / docs).
			templateKey, _ := harness.ResolveTemplateKey("task", taskID)
			var availableItemIDs []string
			if allItems, ierr := harness.EnsureHarnessItems("task", taskID); ierr == nil {
				availableItemIDs = make([]string, 0, len(allItems))
				for _, it := range allItems {
					availableItemIDs = append(availableItemIDs, it.ID)
				}
			}
			suggestedActions := []string{
				fmt.Sprintf("call harness_get(entity_type='task', entity_id='%s') to list the available item_ids", taskID),
			}
			suggestedActions = append(suggestedActions, buildSuggestedActions(harnessResult.UncheckedRequired)...)

			return nil, &apperr.BlockedError{
				EntityType:       "task",
				EntityID:         taskID,
				UncheckedItems:   uncheckedItems,
				CriteriaStatus:   criteria,
				Message:          fmt.Sprintf("%d unchecked item(s)", len(uncheckedItems)),
				SuggestedActions: suggestedActions,
				Extra: map[string]any{
					"template_key":       templateKey,
					"available_item_ids": availableItemIDs,
					"recovery_hint": fmt.Sprintf(
						"Harness items differ per Task type. First call harness_get(entity_type='task', entity_id='%s') to list the item_ids available under the %s template. Conventional ids such as build_passed / tests_passed cause NOT_FOUND on infra/docs/chore Tasks.",
						taskID, templateKey,
					),
				},
			}
		}

		// hotfix-type-specific verification + Result-section verification —
		// both must run before transitionTask so that a BLOCKED response
		// prevents any status transition.
		if gs := store.Get(); gs != nil {
			var taskType, filePath string
			taskType, filePath, _ = gs.GetTaskTypeAndPath(taskID)

			// fetch createdAt for the git log fallback.
			// On failure, leave it empty and skip the fallback — strict
			// verification still operates safely.
			var taskCreatedAt string
			if details, derr := gs.GetTaskDetails(taskID); derr == nil && details != nil {
				taskCreatedAt = details.CreatedAt
			}

			// hotfix Rollback-section verification (existing).
			if taskType == "hotfix" && filePath != "" {
				absPath := filepath.Join(fileutil.GetProjectRoot(), filePath)
				if ok, reason := verifyRollbackSection(absPath); !ok {
					_ = audit.LogEvent("hotfix.rollback_missing", "task", taskID, "claude", map[string]any{
						"reason": reason,
					}, "")
					return nil, &apperr.BlockedError{
						EntityType: "task",
						EntityID:   taskID,
						Message:    fmt.Sprintf("hotfix rollback-section verification failed: %s", reason),
						SuggestedActions: []string{
							"add a `## Rollback` section to the Task file and document the rollback procedure",
							"the body must be at least 20 characters long",
						},
					}
				}
			}

			// Result section vs git diff verification (policy
			// strict -> BLOCKED, warn -> warning only).
			if filePath != "" {
				absPath := filepath.Join(fileutil.GetProjectRoot(), filePath)
				policy := config.GetTaskResultCheckPolicy()
				// switched to JSON SSOT (harness_defaults.json) lookup.
				shouldVerify := policy != "off" && (policy == "strict" || harness.RequiresGitDiff(taskType))

				if shouldVerify {
					// use the task-type-aware version —
					// infra type defaults to narrative mode.
					missing, checked, sectionTitles := verifyTaskResultsForType(taskID, absPath, taskType)

					// strict-only git log fallback —
					// rescues post-hoc task_complete cases (already merged
					// commits) from missing misclassification. Runs only
					// when the first pass produced misses; recovered
					// entries are moved to checked and surfaced in the
					// success response.
					var recoveredViaHistory []string
					if len(missing) > 0 && policy == "strict" {
						var stillMissing []domain.MissingFile
						stillMissing, recoveredViaHistory = applyGitLogFallback(taskCreatedAt, missing)
						if len(recoveredViaHistory) > 0 {
							_ = audit.LogEvent("task.result_fallback_recovered", "task", taskID, "claude", map[string]any{
								"policy":       policy,
								"recovered":    recoveredViaHistory,
								"created_at":   taskCreatedAt,
								"fallback":     "git_log_since",
								"before_count": len(missing),
								"after_count":  len(stillMissing),
							}, "")
							checked = append(checked, recoveredViaHistory...)
						}
						missing = stillMissing
					}

					if len(missing) > 0 && policy == "strict" {
						missingPaths := make([]string, len(missing))
						for i, m := range missing {
							missingPaths[i] = m.Path
						}
						_ = audit.LogEvent("task.result_unverified", "task", taskID, "claude", map[string]any{
							"policy":   policy,
							"missing":  missing,
							"checked":  checked,
							"sections": sectionTitles,
						}, "")
						return nil, &apperr.BlockedError{
							EntityType: "task",
							EntityID:   taskID,
							Message: fmt.Sprintf(
								"%d file(s) recorded in the Task Result section are not present in git diff/log. The work output is unverified — fix the underlying issue and call task_complete again with verifiable results (actual modified files + correct paths). Missing paths: %s",
								len(missing), strings.Join(missingPaths, ", "),
							),
							SuggestedActions: []string{
								"check the Task Result section for typos or fabricated paths",
								"actually create/modify the missing files and stage them in git (include git add)",
								fmt.Sprintf("verify whether already committed: `git log --since=%s -- <path>`", taskCreatedAt),
								"after fixing, retry task_complete",
							},
							Extra: map[string]any{
								"reason":              "task_result_unverified",
								"policy":              policy,
								"section_titles_used": sectionTitles,
								"checked_files":       checked,
								"missing_files":       missing,
								"recovery_hint":       "1) check for typos in the Result-section paths and fix them, or 2) actually create/modify the missing files and stage them in git, then retry. 3) If already merged, verify history with `git log --since=<task.created> -- <path>` (the fallback inspects the range from Task created_at onwards automatically). To temporarily relax the policy, set HSTL_TASK_RESULT_CHECK_POLICY=warn — bypassing verification is discouraged.",
							},
						}
					}
					// Save the verification result for downstream processing
					// (populate ArtifactWarnings under warn policy and
					// attach to the success response).
					completeVerificationResult = &domain.VerificationResult{
						Policy:            policy,
						SectionTitlesUsed: sectionTitles,
						CheckedFiles:      checked,
						MissingFiles:      missing,
					}
					// gate-log entry — task result verification verdict.
					recordTaskResultJudgement(taskID, policy, missing, checked)
				}
			}
		}
	}

	// auto-insert Result-section stub (before HMAC compute).
	// When the file lacks a `## Result` section, insert a stub so that the
	// "cannot retroactively add" problem does not occur.
	var resultSectionStubbed bool
	if gs := store.Get(); gs != nil {
		stubFilePath, _ := gs.GetTaskFilePath(taskID)
		if stubFilePath != "" {
			absStubPath := filepath.Join(fileutil.GetProjectRoot(), stubFilePath)
			if inserted, ierr := EnsureResultSectionStub(absStubPath); ierr == nil {
				resultSectionStubbed = inserted
			}
			// if the user wrote a Result section, remove
			// the guidance blockquote (the user has already produced
			// meaningful content, so guidance is no longer needed). Skip
			// removal when the stub was just inserted (still a placeholder).
			if !resultSectionStubbed {
				_, _ = StripT578Blockquote(absStubPath)
			}
		}
	}

	// cascade: when currentDBStatus is todo, perform the
	// todo -> in-progress transition first. If the second transition
	// (in-progress -> done) fails, rollback restores the original todo
	// state. When called from in-progress, behaviour is unchanged.
	cascadeFromTodo := currentDBStatus == taskStatusTodo
	if cascadeFromTodo {
		if _, cerr := transitionTask(
			taskID, taskStatusTodo, taskStatusInProgress, eventTaskCascadeStarted,
		); cerr != nil {
			return nil, cerr
		}
	}

	result, err := transitionTask(taskID, taskStatusInProgress, taskStatusDone, "task.completed")
	if err != nil {
		if cascadeFromTodo {
			// Rollback: in-progress -> todo. Even if rollback fails, return
			// the original error.
			_, _ = transitionTask(
				taskID, taskStatusInProgress, taskStatusTodo, eventTaskCascadeRolledBack,
			)
		}
		return nil, fmt.Errorf("Complete: task status transition failed (%s): %w", taskID, err)
	}

	// (): task lifecycle file SSOT self-heal.
	// Right after transitionTask, if a sprint is bound, fs-scan that sprint's
	// tasks/ folder to reconcile file_path drift. Blocks / regression
	// patterns. Graceful — failure does not affect the task-complete flow.
	if gs := store.Get(); gs != nil {
		if details, derr := gs.GetTaskFull(taskID); derr == nil && details != nil && details.Sprint != "" {
			reconcileTaskPathsForSprint(details.Sprint)
		}
	}

	tr, ok := result.(*domain.TransitionResult)
	if !ok {
		return result, nil
	}
	tr.ResultSectionStubbed = resultSectionStubbed

	// Attach the verification result to the response (after transitionTask succeeds).
	if completeVerificationResult != nil {
		tr.VerificationResult = completeVerificationResult
		// Under warn policy, also fill ArtifactWarnings when missing entries exist.
		if completeVerificationResult.Policy == "warn" && len(completeVerificationResult.MissingFiles) > 0 {
			var warns []string
			for _, m := range completeVerificationResult.MissingFiles {
				warns = append(warns, fmt.Sprintf("%s: %s", m.Path, m.Reason))
			}
			tr.ArtifactWarnings = warns
		}
	}

	// Follow-up verification (policy-independent, informational WARN).
	if gs := store.Get(); gs != nil {
		filePath, _ := gs.GetTaskFilePath(taskID)
		if filePath != "" {
			absPath := filepath.Join(fileutil.GetProjectRoot(), filePath)
			if followupWarns := verifyFollowups(absPath); len(followupWarns) > 0 {
				tr.FollowupWarnings = followupWarns
			}
		}
	}

	// post_actions: hotfix / infra type only.
	tr.PostActions = buildPostActions(tr.Type, taskID, tr.Title)

	reminders := config.GetReminders(string(events.EventTaskComplete))
	if len(reminders) > 0 {
		tr.Reminders = reminders
	}
	sprintID, _ := tr.Sprint.(string)
	if sprintID != "" {
		tr.SuggestedNextAction = fmt.Sprintf("next Task: call task_next() or check sprint_progress('%s')", sprintID)
	} else {
		tr.SuggestedNextAction = "next Task: call task_next()"
	}
	return tr, nil
}

// buildPostActions returns a list of follow-up actions to perform after
// completion, depending on the Task type.
func buildPostActions(taskType, taskID, taskTitle string) []string {
	var actions []string
	switch taskType {
	case "hotfix":
		actions = append(actions,
			fmt.Sprintf("create_followup_test: create a test Task for %s via task_create", taskID),
			fmt.Sprintf("notify_review: request a post-merge code review for %s (hotfixes are urgent fixes — post-review is mandatory)", taskID),
		)
	case "infra":
		actions = append(actions,
			fmt.Sprintf("schedule_verification: %s requires staggered checks after deployment/config change (health check at 10 min / 30 min / 2 h)", taskID),
			fmt.Sprintf("require_change_record: record the %s change in the KB or the operations log", taskID),
		)
	}
	return actions
}

// buildSuggestedActions builds a list of recommended actions from the
// unchecked harness items.
func buildSuggestedActions(uncheckedItems []domain.HarnessItem) []string {
	actionMap := map[string]string{
		"build_passed":      "dotnet build (or run the build script)",
		"tests_passed":      "pytest tests/ (or dotnet test)",
		"code_review":       "run /hstl:code-review",
		"criteria_checked":  "check every done-criteria checkbox in the Task file",
		"reproduction":      "verify the fix in the bug-reproduction environment",
		"root_cause":        "record the root cause in the Task file",
		"deploy_verified":   "deploy to Dev and check /healthz",
		"rollback_plan":     "document the rollback procedure in the Task file",
		"doc_review":        "run /hstl:doc-review",
		"coverage_checked":  "run pytest --cov or dotnet test --collect",
		"git_diff_verified": "verify code changes via git diff",
		"change_record":     "record the change (what was done, scope of impact)",
	}
	var actions []string
	for _, item := range uncheckedItems {
		if action, ok := actionMap[item.ID]; ok {
			actions = append(actions, action)
		}
	}
	return actions
}

// --------------------------------------------------------------------------
// Reopen
// --------------------------------------------------------------------------

// Reopen performs the done/in-progress -> todo/in-progress reverse
// transition. A reason is required.
// trac: HAR-CM011
func Reopen(taskID, reason, newStatus string) (any, error) {
	// 1. Validate reason length.
	if len(strings.TrimSpace(reason)) < 10 {
		return nil, &apperr.RejectedError{
			Reason:         "reason must be at least 10 characters",
			ProvidedLength: len(strings.TrimSpace(reason)),
		}
	}

	// 2. Validate target_status.
	if newStatus != "todo" && newStatus != "in-progress" {
		return nil, &apperr.RejectedError{
			Reason: fmt.Sprintf("target_status must be 'todo' or 'in-progress', got '%s'", newStatus),
		}
	}

	// 3. Look up the current status from the DB.
	gs, gsErr := store.MustGet()
	if gsErr != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	info, err := gs.GetTaskReopenInfo(taskID)
	if err != nil || info == nil {
		return nil, &apperr.NotFoundError{
			EntityType: "task",
			EntityID:   taskID,
			Message:    fmt.Sprintf("Task %s not found.", taskID),
		}
	}
	currentStatus := info.Status
	filePath := info.FilePath
	sprintID := info.SprintID
	title := info.Title
	taskType := info.Type
	estimate := info.Estimate
	priority := info.Priority
	dependsOnJSON := info.DependsOnJSON
	dependsOnDisplay := formatDependsOnForTable(dependsOnJSON)

	// 4. Check whether the transition is allowed.
	if currentStatus != "done" && currentStatus != "in-progress" {
		return nil, &apperr.RejectedError{
			Reason:       fmt.Sprintf("reverse transition not allowed: current=%s. Only done or in-progress can be reopened.", currentStatus),
			EntityID:     taskID,
			CurrentState: currentStatus,
		}
	}

	if currentStatus == "in-progress" && newStatus != "todo" {
		return nil, &apperr.RejectedError{
			Reason:   "from in-progress, the only valid reverse transition is to todo.",
			EntityID: taskID,
		}
	}

	// 5. Update the file frontmatter.
	resolvedPath, _ := ResolveTaskPath(taskID, filePath)
	historyAppended := false
	if resolvedPath != "" {
		_ = fileutil.UpdateTaskStatus(resolvedPath, newStatus)
		_ = fileutil.AppendStatusHistory(resolvedPath, currentStatus, newStatus, reason)
		historyAppended = true
	}

	// when a sprint-unassigned Task transitions
	// done -> todo/in-progress via reopen, move the file back from
	// works/tasks/completed/T*.md to works/tasks/T*.md.
	if resolvedPath != "" && sprintID == "" && currentStatus == "done" {
		if newPath, moved, mErr := fileutil.MoveTaskFromCompleted(resolvedPath); mErr == nil && moved {
			newRel := fileutil.ToRepoRelative(newPath)
			_ = gs.UpdateTaskFilePath(taskID, newRel)
		}
	}

	// 6. Update the DB.
	_ = gs.UpdateTaskStatusDirect(taskID, newStatus)

	// 7. Update the SPRINT.md status column.
	if sprintID != "" {
		_ = fileutil.UpdateSprintMD(sprintID, taskID, title, taskType, estimate, priority, newStatus, dependsOnDisplay)
	}

	// 8. Reset the Harness Gate (only on done -> reopen).
	harnessReset := false
	if currentStatus == "done" {
		_ = gs.DeleteHarnessItemsForEntity("task", taskID)
		harnessReset = true
	}

	// 9. Audit log.
	_ = audit.LogEvent("task.reopened", "task", taskID, "claude", map[string]any{
		"from":   currentStatus,
		"to":     newStatus,
		"reason": reason,
	}, "")

	// (): auto-reconcile file_path drift on task reopen.
	// () only covered task complete — this fills the gap.
	// sprintID was already looked up above. Graceful — failure does not
	// affect the reopen flow.
	if sprintID != "" {
		reconcileTaskPathsForSprint(sprintID)
	}

	return &domain.ReopenResult{
		TaskID:          taskID,
		PreviousStatus:  currentStatus,
		NewStatus:       newStatus,
		Reason:          reason,
		HistoryAppended: historyAppended,
		HarnessReset:    harnessReset,
	}, nil
}

// --------------------------------------------------------------------------
// List
// --------------------------------------------------------------------------

// List performs a DB-cache-based listing.
// requiredTemplateSections lists the section headers that
// RenderTaskTemplate inserts into a new Task body . When a
// header in this slice is missing from the Task file body, outdated is
// reported as true.
// Extension: when a new template section is introduced in the future, add
// it here and task_list / project_status will pick it up in the outdated
// determination automatically.
// Policy: existing Task bodies are never retroactively edited (
// decision). The outdated flag is purely informational ("noted at
// sprint planning time") and does not trigger automatic migration.
var requiredTemplateSections = []string{
	"## Scope Limits", // canonical (English)
	"## Type Tags",
}

// isTaskTemplateOutdated returns true when the Task file is missing any of
// requiredTemplateSections. Returns false if the file cannot be read
// (conservative — undecided when not seen).
// the previous strings.Contains check produced false
// negatives when narrative body text included "## Scope Limits" as a
// substring (found while writing tests). Now uses
// line-based exact match so the string only matches when it forms a
// standalone heading line.
func isTaskTemplateOutdated(filePath string) bool {
	if filePath == "" {
		return false
	}
	absPath := filePath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(fileutil.GetProjectRoot(), filePath)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false
	}
	body := string(data)
	for _, section := range requiredTemplateSections {
		if !containsExactHeading(body, section) {
			return true
		}
	}
	return false
}

// containsExactHeading checks line-by-line whether body has a standalone
// line that equals heading . Avoids substring misfires from
// `strings.Contains`.
// Matching rule: `strings.TrimSpace(line) == heading`. Therefore:
// "## Scope Limits" matches.
// "## Scope Limits notes" does not (different suffix).
// body text mentioning "## Scope Limits" inline does not (only headings
// match).
func containsExactHeading(body, heading string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == heading {
			return true
		}
	}
	return false
}

// headingTitleContainsKeyword checks whether the title portion of a "## XXX"
// heading line includes kw as a standalone whitespace-bounded token.
//
// Token boundary characters: start/end of string, whitespace.
// Hyphenated compounds (e.g. "Followup-style") count as a single
// alphanumeric token and DO NOT match a bare keyword "Followup".
// This avoids substring misfires from `strings.Contains` while still
// matching natural English token boundaries. For example with kw="Followup":
//
//	"## Followup"                matches (exact).
//	"## Followup tasks"          matches (space boundary).
//	"## Major Followup handling" matches (space boundary).
//	"## Followup-style analysis" does NOT match (kw is embedded in the
//	                             compound token "Followup-style").
//	"## METHODOLOGY summary"     does NOT match for kw="TODO" (kw is
//	                             embedded inside an alphanumeric word).
func headingTitleContainsKeyword(headingLine, kw string) bool {
	if kw == "" {
		return false
	}
	// Strip the "## " prefix to extract the bare title.
	title := strings.TrimSpace(strings.TrimPrefix(headingLine, "## "))
	if title == kw {
		return true
	}
	// Word-boundary check via whitespace padding.
	padded := " " + title + " "
	return strings.Contains(padded, " "+kw+" ")
}

// List returns a Task list filtered by sprint/status (legacy entry).
// trac: WRK-QR002
func List(sprint, status *string) (any, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	listResult, err := gs.ListTasks(sprint, status)
	if err != nil {
		return nil, fmt.Errorf("task list lookup failed: %w", err)
	}

	var tasks []domain.TaskSummary
	outdatedCount := 0
	for _, d := range listResult.Tasks {
		var dependsOn []string
		_ = json.Unmarshal([]byte(d.DependsOn), &dependsOn)
		if dependsOn == nil {
			dependsOn = []string{}
		}
		var sprintVal interface{}
		if d.Sprint != "" {
			sprintVal = d.Sprint
		}
		outdated := isTaskTemplateOutdated(d.FilePath)
		if outdated {
			outdatedCount++
		}
		tasks = append(tasks, domain.TaskSummary{
			TaskID:           d.TaskID,
			Title:            d.Title,
			Type:             d.Type,
			Sprint:           sprintVal,
			Status:           d.Status,
			Priority:         d.Priority,
			Estimate:         d.Estimate,
			DependsOn:        dependsOn,
			CreatedAt:        d.CreatedAt,
			TemplateOutdated: outdated,
		})
	}
	if tasks == nil {
		tasks = []domain.TaskSummary{}
	}

	return &domain.ListResult{
		Tasks:                 tasks,
		Count:                 len(tasks),
		OutdatedTemplateCount: outdatedCount,
	}, nil
}

// ListFiltered is the extended filter path.
// Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.
// trac: WRK-QR002
func ListFiltered(filter ports.TaskListFilter) (any, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	listResult, err := gs.ListTasksFiltered(filter)
	if err != nil {
		return nil, fmt.Errorf("task list lookup failed: %w", err)
	}

	var tasks []domain.TaskSummary
	outdatedCount := 0
	for _, d := range listResult.Tasks {
		var dependsOn []string
		_ = json.Unmarshal([]byte(d.DependsOn), &dependsOn)
		if dependsOn == nil {
			dependsOn = []string{}
		}
		var sprintVal interface{}
		if d.Sprint != "" {
			sprintVal = d.Sprint
		}
		outdated := isTaskTemplateOutdated(d.FilePath)
		if outdated {
			outdatedCount++
		}
		tasks = append(tasks, domain.TaskSummary{
			TaskID:           d.TaskID,
			Title:            d.Title,
			Type:             d.Type,
			Sprint:           sprintVal,
			Status:           d.Status,
			Priority:         d.Priority,
			Estimate:         d.Estimate,
			DependsOn:        dependsOn,
			CreatedAt:        d.CreatedAt,
			TemplateOutdated: outdated,
		})
	}
	if tasks == nil {
		tasks = []domain.TaskSummary{}
	}

	return &domain.ListResult{
		Tasks:                 tasks,
		Count:                 len(tasks),
		OutdatedTemplateCount: outdatedCount,
	}, nil
}

// --------------------------------------------------------------------------
// Get
// --------------------------------------------------------------------------

// Get fetches a single Task from the DB plus the file.
// trac: WRK-QR001
func Get(taskID string) (any, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	rec, err := gs.GetTaskFull(taskID)
	if err != nil || rec == nil {
		return nil, &apperr.NotFoundError{
			EntityType:   "task",
			EntityID:     taskID,
			Message:      fmt.Sprintf("Task %s not found.", taskID),
			RecoveryHint: "use task_list to look up an existing Task ID.",
		}
	}

	var dependsOn []string
	_ = json.Unmarshal([]byte(rec.DependsOn), &dependsOn)
	if dependsOn == nil {
		dependsOn = []string{}
	}

	var sprintVal interface{}
	if rec.Sprint != "" {
		sprintVal = rec.Sprint
	}

	filePath := rec.FilePath
	getResult := &domain.GetResult{
		TaskID:    taskID,
		Title:     rec.Title,
		Type:      rec.Type,
		Sprint:    sprintVal,
		Status:    rec.Status,
		Priority:  rec.Priority,
		Estimate:  rec.Estimate,
		FilePath:  filePath,
		DependsOn: dependsOn,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
	}

	// If the file exists, merge in the frontmatter and the ## Summary section.
	resolvedPath, err := ResolveTaskPath(taskID, filePath)
	if err == nil && resolvedPath != "" {
		fm, fmErr := fileutil.ReadTaskFrontmatter(resolvedPath)
		if fmErr == nil {
			getResult.Frontmatter = fm
		}
		// assign the first line of the ## Summary section to Summary.
		if body, readErr := os.ReadFile(resolvedPath); readErr == nil {
			getResult.Summary = extractSummarySection(string(body))
		}
	} else {
		falseVal := false
		getResult.FileExists = &falseVal
	}

	return getResult, nil
}

// extractSummarySection returns the first non-empty line of the `## Summary`
// section in the Markdown body. Returns an empty string if the section is
// absent or empty.
func extractSummarySection(body string) string {
	const summaryHeader = "## Summary"
	idx := strings.Index(body, summaryHeader)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(summaryHeader):]
	for _, line := range strings.Split(rest, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			break // Next section starts.
		}
		return trimmed
	}
	return ""
}

// --------------------------------------------------------------------------
// Next
// --------------------------------------------------------------------------

// Next recommends the next Task by priority + dependencies.
// trac: HAR-CM017
func Next() (any, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	// Use ListTasks to fetch all todo Tasks (store.NextTask does not check
	// dependency resolution).
	todoStatus := "todo"
	listResult, err := gs.ListTasks(nil, &todoStatus)
	if err != nil {
		return nil, fmt.Errorf("todo Task lookup failed: %w", err)
	}

	// done Task ID set — used to check dependency resolution.
	doneTasks, _ := gs.GetDoneTasksWithPaths()
	doneIDs := make(map[string]bool, len(doneTasks))
	for _, t := range doneTasks {
		doneIDs[t.TaskID] = true
	}

	// Fetch the active Sprint list.
	activeSprints, _ := gs.ListSprints("active")
	activeSprintSet := make(map[string]bool, len(activeSprints))
	for _, sp := range activeSprints {
		activeSprintSet[sp.SprintID] = true
	}

	// Pick the first Task whose dependencies are resolved — active Sprints first.
	findFirst := func(sprintFilter string) *domain.TaskSummary {
		for _, d := range listResult.Tasks {
			if sprintFilter != "" && d.Sprint != sprintFilter {
				continue
			}
			var deps []string
			_ = json.Unmarshal([]byte(d.DependsOn), &deps)
			if deps == nil {
				deps = []string{}
			}
			allResolved := true
			for _, dep := range deps {
				if !doneIDs[dep] {
					allResolved = false
					break
				}
			}
			if !allResolved {
				continue
			}
			var sprintVal interface{}
			if d.Sprint != "" {
				sprintVal = d.Sprint
			}
			return &domain.TaskSummary{
				TaskID:    d.TaskID,
				Title:     d.Title,
				Type:      d.Type,
				Sprint:    sprintVal,
				Priority:  d.Priority,
				Estimate:  d.Estimate,
				DependsOn: deps,
				CreatedAt: d.CreatedAt,
			}
		}
		return nil
	}

	// First pass: prefer todo Tasks in an active Sprint.
	for sprintID := range activeSprintSet {
		if result := findFirst(sprintID); result != nil {
			return result, nil
		}
	}

	// Second pass: global todo fallback.
	if result := findFirst(""); result != nil {
		return result, nil
	}

	return &domain.NextMessageResult{
		Message: "no Task to recommend — every todo Task still has unresolved dependencies.",
	}, nil
}

// --------------------------------------------------------------------------
// Placeholder detection
// --------------------------------------------------------------------------

// _backtickContentRe matches inline code / paths / strings wrapped in
// backticks . extractBodyForPlaceholderScan uses it to strip
// backtick-quoted spans (false-positive prevention).
var _backtickContentRe = regexp.MustCompile("`[^`]*`")

// _placeholderScanResultTitles — set of headings recognised as the "## Result"
// section entry . Must stay in sync with ceremony.resultSectionTitles.
var _placeholderScanResultTitles = map[string]bool{
	"Result": true, "Artifact": true, "Artifacts": true,
}

// _placeholderScanNarrativeSubs — set of narrative sub-headings inside
// "## Result" . Must stay in sync with ceremony.narrativeSubHeadings.
var _placeholderScanNarrativeSubs = map[string]bool{
	"Design": true, "Design Decisions": true,
	"Validation": true, "Commit": true, "Commits": true,
	"Trade-offs": true, "Tradeoffs": true, "Follow-up": true, "Follow-Ups": true,
}

// extractBodyForPlaceholderScan is the body pre-processor used by
// HasPlaceholderBody .
// Behaviour:
// 1. Strip narrative lines under "## Result -> ### Design Decisions /
// Validation / Commit / Follow-up" — prevents false positives when a
// marker is quoted descriptively.
// 2. Strip backtick-quoted spans (integrating the original
// _backtickContentRe logic, ).
// Note: applies the same rules as ceremony.ExtractPlaceholderScanContent.
// A local implementation is kept rather than calling that function directly
// because of the ceremony -> task import cycle.
func extractBodyForPlaceholderScan(body string) string {
	var out strings.Builder
	inResult := false
	inNarrativeSub := false

	firstWord := func(s string) string {
		if idx := strings.IndexAny(s, " \t"); idx > 0 {
			return s[:idx]
		}
		return s
	}

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			fw := firstWord(heading)
			inResult = _placeholderScanResultTitles[heading] || _placeholderScanResultTitles[fw]
			inNarrativeSub = false
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			sub := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
			fw := firstWord(sub)
			if inResult {
				inNarrativeSub = _placeholderScanNarrativeSubs[sub] || _placeholderScanNarrativeSubs[fw]
			} else {
				inNarrativeSub = false
			}
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		if inNarrativeSub {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return _backtickContentRe.ReplaceAllString(out.String(), "")
}

// _placeholderMarkers lists the markers used to detect whether placeholder body
// content still remains . Backtick-quoted examples are stripped before the
// scan so only genuine markers in the body trigger a match.
// ( hotfix): the markers are restricted to **structural patterns
// that exist only at template-generation time** (brace placeholders). Natural
// language substrings that match the detection message are not used as markers
// because Task bodies that merely mention the term descriptively would yield
// false positives. See KB M001.
var _placeholderMarkers = []string{
	"{The problem this Task",
	"{Requirement ",
	"{Criterion ",
	"{TODO",
	"{One-line summary",
	"{Work items confirmed at sprint",
	"{Items deferred outside",
	"{Related documents",
}

// strengthens the sprint_start design-readiness gate.
// A supplementary criterion that prevents "empty shell" Tasks (where only
// template markers were removed but Purpose/Requirements were left blank)
// from passing without a WARN.
// minTaskBodyChars: minimum character count for the body after stripping
// frontmatter and backtick blocks. Anything below this is treated as a
// placeholder. The threshold is generous so short chore Tasks are not
// flagged, yet a populated default template clears it comfortably.
// minRequirementItems: minimum number of `- [ ]` / `- [x]` checkboxes
// under the `## Requirements` section. Zero means Requirements is empty.
const (
	minTaskBodyChars    = 300
	minRequirementItems = 1
)

// _requirementSectionHeading is the exact heading used to identify the start
// of the Requirements section. Substring matching such as
// strings.HasPrefix("##") is avoided because it would also match
// `### Requirements details`.
const _requirementSectionHeading = "## Requirements"

// _structuredSectionHeadings lists the level-2 sections in which Task content
// may legitimately be recorded . The detector treats the body as
// non-placeholder when at least one item is found in any of these sections.
// The original implementation counted only `- [ ]` checkboxes under
// `## Requirements`. In real projects, however, authors often record
// requirements under `## Done Criteria` or `## Scope`, or use numbered lists
// or bullets instead of checkboxes. In that case the body is sufficiently
// written yet still triggers a false positive — dozens of such cases were
// observed downstream. hotfix relaxes this.
var _structuredSectionHeadings = []string{
	"## Requirements",
	"## Done Criteria",
	"## Scope",
	"## Scope Limits",
}

// _structuredItemRe matches structural items.
// Checkboxes: `- [ ]` / `- [x]`.
// Numbered list: `1.` / `2.` etc.
// Bullet: `- text` / `* text` (excluding overlap with `- [` checkbox start).
var _structuredItemRe = regexp.MustCompile(`(?m)^\s*(?:-\s*\[[ xX]\]|\d+\.\s|[-*]\s+[^\[\s])`)

// _requirementCheckboxRe matches checkboxes only (kept for backward
// compatibility with the original requirement count helper).
var _requirementCheckboxRe = regexp.MustCompile(`(?m)^\s*-\s*\[[ xX]\]`)

// HasPlaceholderBody checks whether the Task file's body still contains
// placeholder patterns. After frontmatter, it inspects the body for
// `{The problem this Task` markers and structural emptiness in the
// `## Purpose` / `## Requirements` sections.
// Backtick-wrapped code/quote examples are stripped before the check so
// that bodies that self-reference (e.g. "remove placeholders such as
// `{The problem this Task ...}`") do not produce false positives.
func HasPlaceholderBody(taskID string) bool {
	gs := store.Get()
	if gs == nil {
		return false
	}

	filePath, err := gs.GetTaskFilePath(taskID)
	if err != nil || filePath == "" {
		return false
	}

	root := fileutil.GetProjectRoot()
	absPath := filepath.Join(root, filePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false
	}

	content := string(data)
	// Body extraction (after frontmatter).
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return false
	}
	body := parts[2]

	// Narrative-section exclusion + backtick strip (combined preprocessing).
	stripped := extractBodyForPlaceholderScan(body)

	for _, marker := range _placeholderMarkers {
		if strings.Contains(stripped, marker) {
			return true
		}
	}

	// Body-length criterion — when every marker has been removed but the
	// body is too short, treat as an "empty shell" Task. Threshold: after
	// frontmatter exclusion + backtick stripping, the whitespace-stripped
	// character count must be at least minTaskBodyChars.
	compact := strings.Join(strings.Fields(stripped), "")
	if len(compact) < minTaskBodyChars {
		return true
	}

	// Structural-item count criterion.
	// If at least minRequirementItems checkbox / numbered-list / bullet
	// items are present in any of the candidate sections (## Requirements
	// / ## Done Criteria / ## Scope), the body passes. The original
	// implementation that checked only one section + one pattern produced
	// false positives that depended on stylistic choice.
	if countStructuredItemsAcrossSections(body) < minRequirementItems {
		return true
	}

	return false
}

// countStructuredItemsAcrossSections sums structural items across every
// candidate section. Each section contributes the count of checkboxes,
// numbered-list entries, and bullets found beneath it; absent sections
// contribute zero.
// The earlier implementation looked only at `## Requirements` plus
// checkboxes, but in practice authors place requirements under
// `## Done Criteria` / `## Scope` and use numbered lists / bullets, which
// produced many false positives. Relaxed policy: a body is non-placeholder
// when any structural section has at least one item.
func countStructuredItemsAcrossSections(body string) int {
	total := 0
	for _, heading := range _structuredSectionHeadings {
		total += countItemsInSection(body, heading, _structuredItemRe)
	}
	return total
}

// countItemsInSection counts lines under the given level-2 section that
// match the regular expression. Termination happens at the next level-2
// `## ` heading or EOF.
// Section-heading matching is prefix-based: `## Requirements` matches
// `## Requirements`, `## Requirements (deferred)`,
// `## Requirements — after M2`, and similar suffix-annotated variants.
// Level-3 `### ...` is naturally excluded since the target is `## `.
func countItemsInSection(body, sectionHeading string, itemRe *regexp.Regexp) int {
	lines := strings.Split(body, "\n")
	inSection := false
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inSection {
			if trimmed == sectionHeading || strings.HasPrefix(trimmed, sectionHeading+" ") {
				inSection = true
			}
			continue
		}
		// Stop at the next level-2 section (when not the same heading).
		if strings.HasPrefix(trimmed, "## ") &&
			!(trimmed == sectionHeading || strings.HasPrefix(trimmed, sectionHeading+" ")) {
			break
		}
		if itemRe.MatchString(line) {
			count++
		}
	}
	return count
}

// countRequirementItems counts the `- [ ]` / `- [x]` checkboxes under the
// `## Requirements` section. The section runs up to the next `## `
// level-2 heading (or EOF).
func countRequirementItems(body string) int {
	lines := strings.Split(body, "\n")
	inSection := false
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inSection {
			if trimmed == _requirementSectionHeading {
				inSection = true
			}
			continue
		}
		// Stop on the next `## ` level-2 heading.
		if strings.HasPrefix(trimmed, "## ") && trimmed != _requirementSectionHeading {
			break
		}
		if _requirementCheckboxRe.MatchString(line) {
			count++
		}
	}
	return count
}

// FindPlaceholderTasks returns the IDs of Tasks whose body still has
// placeholder content within a specific Sprint (or all Sprints).
func FindPlaceholderTasks(sprintFilter string) []string {
	gs := store.Get()
	if gs == nil {
		return nil
	}

	taskIDs, err := gs.FindPlaceholderTasks(sprintFilter)
	if err != nil {
		return nil
	}

	var placeholders []string
	for _, tid := range taskIDs {
		if HasPlaceholderBody(tid) {
			placeholders = append(placeholders, tid)
		}
	}
	return placeholders
}

// --------------------------------------------------------------------------
// Delete
// --------------------------------------------------------------------------

// DeleteOptions — domain.DeleteOptions alias.
type DeleteOptions = domain.DeleteOptions

// Delete removes a Task in the todo state. A reason of at least 10
// characters is required.
// trac: HAR-CM016
// Backward-compat wrapper — equivalent to
// DeleteWithOptions(..., DeleteOptions{}).
func Delete(taskID, reason string) (any, error) {
	return DeleteWithOptions(taskID, reason, DeleteOptions{})
}

// DeleteWithOptions is the option-aware Delete entry point. When
// OrphanFile=true, DB drift (file exists without a DB row) is recorded as
// the `task.orphan_deleted` audit event before the file and BACKLOG entry
// are cleaned up.
// trac: HAR-CM016
func DeleteWithOptions(taskID, reason string, opts DeleteOptions) (any, error) {
	root := fileutil.GetProjectRoot()

	// Bootstrap Gate
	bootstrap, err := migration.CheckBootstrapNeeded(root)
	if err != nil {
		return nil, fmt.Errorf("bootstrap check failed: %w", err)
	}
	if bootstrap != nil {
		return migration.BuildBootstrapResponse(bootstrap, root), nil
	}

	if len(reason) < 10 {
		return nil, &apperr.RejectedError{
			Reason:   "deletion reason must be at least 10 characters.",
			EntityID: taskID,
		}
	}

	gs, gsErr := store.MustGet()
	if gsErr != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	details, err := gs.GetTaskDetails(taskID)
	if err != nil || details == nil {
		// DB-drift path — only the file may exist.
		if opts.OrphanFile {
			return deleteOrphanFile(taskID, reason, root)
		}
		return nil, &apperr.NotFoundError{
			EntityType: "task",
			EntityID:   taskID,
			Message:    fmt.Sprintf("Task %s not found. (Use --orphan-file when only the file exists due to drift.)", taskID),
		}
	}
	taskStatus := details.Status
	sprintID := details.Sprint
	filePath := details.FilePath

	if taskStatus != "todo" {
		return nil, &apperr.RejectedError{
			Reason:       fmt.Sprintf("only todo Tasks may be deleted (current: %s)", taskStatus),
			EntityID:     taskID,
			CurrentState: taskStatus,
		}
	}

	// Delete the file.
	resolvedPath, err := ResolveTaskPath(taskID, filePath)
	if err == nil && resolvedPath != "" {
		_ = os.Remove(resolvedPath)
	}

	// Remove the row from SPRINT.md or BACKLOG.md.
	if sprintID != "" {
		_, _ = fileutil.RemoveRowFromSprintMD(sprintID, taskID)
	}
	_, _ = fileutil.RemoveFromBacklogMD(taskID)
	_, _ = fileutil.RebuildBacklogSummary()

	// DB delete.
	_ = gs.DeleteTask(taskID)

	// Audit log.
	_ = audit.LogEvent("task.deleted", "task", taskID, "claude", map[string]any{
		"reason": reason,
		"sprint": sprintID,
	}, "")

	return &domain.DeleteResult{
		TaskID: taskID,
		Status: "deleted",
		Reason: reason,
	}, nil
}

// deleteOrphanFile removes a Task that is in DB-drift state (file exists
// without a DB row). Uses findTaskFiles to recursively glob works/ for
// `T{id}-*.md` and remove it. Also strips the row from BACKLOG.md.
// Returns NotFoundError when no file exists (signals misuse of the drift
// flag). When several files match, returns an error for safety so the
// user can clean up manually.
func deleteOrphanFile(taskID, reason, root string) (any, error) {
	matches := findTaskFiles(root, taskID)
	if len(matches) == 0 {
		return nil, &apperr.NotFoundError{
			EntityType: "task",
			EntityID:   taskID,
			Message:    fmt.Sprintf("Task %s: not present in either DB or filesystem. Before using --orphan-file, verify drift via `hstl backlog sync --dry-run`.", taskID),
		}
	}
	if len(matches) > 1 {
		return nil, &apperr.RejectedError{
			Reason:   fmt.Sprintf("Task %s: %d files share this ID — manual cleanup required. Candidates: %v", taskID, len(matches), matches),
			EntityID: taskID,
		}
	}

	target := matches[0]
	relPath := fileutil.ToRepoRelative(target)
	if err := os.Remove(target); err != nil {
		return nil, fmt.Errorf("orphan file removal failed (%s): %w", relPath, err)
	}

	// Strip the row from BACKLOG.md (silent — the row may not be present).
	_, _ = fileutil.RemoveFromBacklogMD(taskID)
	_, _ = fileutil.RebuildBacklogSummary()

	_ = audit.LogEvent("task.orphan_deleted", "task", taskID, "claude", map[string]any{
		"reason":    reason,
		"file_path": relPath,
	}, "")

	return &domain.DeleteResult{
		TaskID: taskID,
		Status: "orphan_deleted",
		Reason: reason,
	}, nil
}

// --------------------------------------------------------------------------
// Update — task_update tool backend
// --------------------------------------------------------------------------

// _validTaskTypes is the set of valid Task type values.
var _validTaskTypes = map[string]bool{
	"feature":  true,
	"bugfix":   true,
	"hotfix":   true,
	"refactor": true,
	"infra":    true,
	"docs":     true,
	"test":     true,
	"chore":    true,
	"spike":    true,
}

// _validPriorities is the set of valid priority values.
var _validPriorities = map[string]bool{
	"p0": true, "p1": true, "p2": true, "p3": true,
}

// _validEstimates is the set of valid estimate values.
var _validEstimates = map[string]bool{
	"XS": true, "S": true, "M": true, "L": true, "XL": true,
}

// _validUpdateStatuses is the set of status values accepted by
// task_update --status.
var _validUpdateStatuses = map[string]bool{
	"todo":        true,
	"in-progress": true,
	"done":        true,
	"absorbed":    true,
}

// _normalTransitions is the set of "normal transition" adjacent pairs.
// Transitions outside this set are logged as WARN by Update but not
// blocked.
var _normalTransitions = map[string]bool{
	"todo|in-progress":     true,
	"in-progress|done":     true,
	"done|in-progress":     true, // reopen reverse transition
	"in-progress|todo":     true, // reopen reverse transition
	"done|absorbed":        true,
	"todo|absorbed":        true,
	"in-progress|absorbed": true,
}

// Update partially updates the frontmatter fields of an existing Task.
// Order of operations (atomicity not guaranteed but close to idempotent
// without compensating transactions):
// 1. Input validation (HasUpdates, enum values, etc.).
// 2. Load current values from the DB (NotFound check).
// 3. Partial frontmatter update on the file (UpdateTaskFrontmatter).
// 4. UPDATE on the DB tasks table (changed columns only).
// 5. Update the SPRINT.md / BACKLOG.md row (when title/priority and
// similar table columns change).
// 6. Record task.updated in audit_events (changed fields + old/new values).
// Returns:
// domain.UpdateResult: changed fields, old/new value maps.
// InvalidStateError: input validation failed (HasUpdates false, enum
// violation).
// NotFoundError: Task ID does not exist.
// trac: HAR-CM015
func Update(req *domain.UpdateRequest) (*domain.UpdateResult, error) {
	if req == nil || req.TaskID == "" {
		return nil, &apperr.InvalidStateError{
			Message:      "task_id is empty",
			RecoveryHint: "supply task_id.",
		}
	}
	if !req.HasUpdates() {
		return nil, &apperr.InvalidStateError{
			EntityID:     req.TaskID,
			Message:      "no fields to update were specified",
			RecoveryHint: "specify at least one of title/type/priority/estimate/depends_on.",
		}
	}

	// Input enum validation.
	if req.Type != nil && !_validTaskTypes[*req.Type] {
		return nil, &apperr.InvalidStateError{
			EntityID:     req.TaskID,
			Message:      fmt.Sprintf("invalid type value: %q", *req.Type),
			RecoveryHint: "type must be one of feature/bugfix/hotfix/refactor/infra/docs/test/chore/spike.",
		}
	}
	if req.Priority != nil && !_validPriorities[*req.Priority] {
		return nil, &apperr.InvalidStateError{
			EntityID:     req.TaskID,
			Message:      fmt.Sprintf("invalid priority value: %q", *req.Priority),
			RecoveryHint: "priority must be one of p0/p1/p2/p3.",
		}
	}
	if req.Estimate != nil && !_validEstimates[*req.Estimate] {
		return nil, &apperr.InvalidStateError{
			EntityID:     req.TaskID,
			Message:      fmt.Sprintf("invalid estimate value: %q", *req.Estimate),
			RecoveryHint: "estimate must be one of XS/S/M/L/XL.",
		}
	}
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, &apperr.InvalidStateError{
			EntityID:     req.TaskID,
			Message:      "title is an empty string",
			RecoveryHint: "title must not be empty. To leave it unchanged, omit the title argument.",
		}
	}
	if req.Status != nil && !_validUpdateStatuses[*req.Status] {
		return nil, &apperr.InvalidStateError{
			EntityID:     req.TaskID,
			Message:      fmt.Sprintf("invalid status value: %q", *req.Status),
			RecoveryHint: "status must be one of todo / in-progress / done / absorbed.",
		}
	}

	gs, gsErr := store.MustGet()
	if gsErr != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	// Load current values (for NotFound check + building old_values).
	details, err := gs.GetTaskDetails(req.TaskID)
	if err != nil || details == nil {
		return nil, &apperr.NotFoundError{
			EntityType:   "task",
			EntityID:     req.TaskID,
			Message:      fmt.Sprintf("Task %s not found.", req.TaskID),
			RecoveryHint: "use task_list to look up an existing Task ID.",
		}
	}
	currentTitle := details.Title
	currentType := details.Type
	currentPriority := details.Priority
	currentEstimate := details.Estimate
	currentSprint := details.Sprint
	currentStatus := details.Status
	currentFilePath := details.FilePath
	currentDependsOnJSON := details.DependsOn

	// Determine changes (only fields whose values actually differ are added
	// to updated_fields).
	updatedFields := []string{}
	oldValues := map[string]any{}
	newValues := map[string]any{}
	frontmatterUpdates := map[string]any{}

	if req.Title != nil && *req.Title != currentTitle {
		updatedFields = append(updatedFields, "title")
		oldValues["title"] = currentTitle
		newValues["title"] = *req.Title
		frontmatterUpdates["title"] = *req.Title
	}
	if req.Type != nil && *req.Type != currentType {
		updatedFields = append(updatedFields, "type")
		oldValues["type"] = currentType
		newValues["type"] = *req.Type
		frontmatterUpdates["type"] = *req.Type
	}
	if req.Priority != nil && *req.Priority != currentPriority {
		updatedFields = append(updatedFields, "priority")
		oldValues["priority"] = currentPriority
		newValues["priority"] = *req.Priority
		frontmatterUpdates["priority"] = *req.Priority
	}
	if req.Estimate != nil && *req.Estimate != currentEstimate {
		updatedFields = append(updatedFields, "estimate")
		oldValues["estimate"] = currentEstimate
		newValues["estimate"] = *req.Estimate
		frontmatterUpdates["estimate"] = *req.Estimate
	}
	if req.DependsOn != nil {
		oldDeps := decodeJSONStringSlice(currentDependsOnJSON)
		newDeps := *req.DependsOn
		if newDeps == nil {
			newDeps = []string{}
		}
		if !stringSlicesEqual(oldDeps, newDeps) {
			updatedFields = append(updatedFields, "depends_on")
			oldValues["depends_on"] = oldDeps
			newValues["depends_on"] = newDeps
			frontmatterUpdates["depends_on"] = newDeps
		}
	}
	if req.Status != nil && *req.Status != currentStatus {
		updatedFields = append(updatedFields, "status")
		oldValues["status"] = currentStatus
		newValues["status"] = *req.Status
		frontmatterUpdates["status"] = *req.Status
		// Abnormal transitions are logged WARN-only and not blocked.
		transitionKey := currentStatus + "|" + *req.Status
		if !_normalTransitions[transitionKey] {
			taskLogger.Warn(brand.LogPrefix+" task abnormal status transition — escape-hatch path",
				"task_id", req.TaskID, "from", currentStatus, "to", *req.Status)
		}
	}
	if req.Sprint != nil && *req.Sprint != currentSprint {
		updatedFields = append(updatedFields, "sprint")
		oldValues["sprint"] = currentSprint
		newValues["sprint"] = *req.Sprint
		frontmatterUpdates["sprint"] = *req.Sprint
	}
	if len(updatedFields) == 0 {
		// Inputs were provided but each matched the current value — no-op success.
		return &domain.UpdateResult{
			TaskID:        req.TaskID,
			UpdatedFields: updatedFields,
			OldValues:     oldValues,
			NewValues:     newValues,
		}, nil
	}

	// 1. Update the file frontmatter.
	resolvedPath, err := ResolveTaskPath(req.TaskID, currentFilePath)
	if err == nil && resolvedPath != "" {
		if err := fileutil.UpdateTaskFrontmatter(resolvedPath, frontmatterUpdates); err != nil {
			return nil, fmt.Errorf("frontmatter update failed: %w", err)
		}
	}

	// 2. Update the DB — only changed columns.
	dbFields := ports.TaskUpdateFields{}
	for _, f := range updatedFields {
		switch f {
		case "title":
			v := newValues["title"].(string)
			dbFields.Title = &v
		case "type":
			v := newValues["type"].(string)
			dbFields.Type = &v
		case "priority":
			v := newValues["priority"].(string)
			dbFields.Priority = &v
		case "estimate":
			v := newValues["estimate"].(string)
			dbFields.Estimate = &v
		case "status":
			v := newValues["status"].(string)
			dbFields.Status = &v
		case "sprint":
			v := newValues["sprint"].(string)
			dbFields.Sprint = &v
		case "depends_on":
			b, _ := json.Marshal(newValues["depends_on"])
			s := string(b)
			dbFields.DependsOn = &s
		}
	}
	_ = gs.UpdateTaskDirect(req.TaskID, dbFields)

	// 3. Update SPRINT.md / BACKLOG.md rows when title/type/estimate/
	// priority/depends_on changes. When the row itself needs to be
	// rebuilt, combine RemoveRowFromSprintMD + UpdateSprintMD.
	tableAffected := false
	for _, f := range updatedFields {
		if f == "title" || f == "type" || f == "estimate" || f == "priority" || f == "depends_on" || f == "status" {
			tableAffected = true
			break
		}
	}
	if tableAffected {
		// Pull post-update values.
		newTitle := currentTitle
		if v, ok := newValues["title"].(string); ok {
			newTitle = v
		}
		newType := currentType
		if v, ok := newValues["type"].(string); ok {
			newType = v
		}
		newEstimate := currentEstimate
		if v, ok := newValues["estimate"].(string); ok {
			newEstimate = v
		}
		newPriority := currentPriority
		if v, ok := newValues["priority"].(string); ok {
			newPriority = v
		}
		newStatus := currentStatus
		if v, ok := newValues["status"].(string); ok {
			newStatus = v
		}
		newDepsDisplay := formatDependsOnForTable(currentDependsOnJSON)
		if v, ok := newValues["depends_on"].([]string); ok {
			newDepsDisplay = formatDependsOnSlice(v)
		}

		if currentSprint != "" && !pseudoSprints[currentSprint] {
			// SPRINT.md: remove the existing row, then add a new one
			// (status is also part of the row).
			_, _ = fileutil.RemoveRowFromSprintMD(currentSprint, req.TaskID)
			_ = fileutil.UpdateSprintMD(currentSprint, req.TaskID, newTitle, newType, newEstimate, newPriority, newStatus, newDepsDisplay)
		} else {
			// BACKLOG.md: remove the existing row, then add a new one.
			_, _ = fileutil.RemoveFromBacklogMD(req.TaskID)
			_ = fileutil.UpdateBacklogMD(req.TaskID, newTitle, newType, newEstimate, newPriority)
			_, _ = fileutil.RebuildBacklogSummary()
		}
	}

	// 4. Audit log.
	_ = audit.LogEvent("task.updated", "task", req.TaskID, "claude", map[string]any{
		"updated_fields": updatedFields,
		"old_values":     oldValues,
		"new_values":     newValues,
	}, "")

	// Dedicated event for status-escape-hatch monitoring. Tracks direct
	// status changes that bypass the normal path (task start/complete/
	// reopen). Searchable via
	// `hstl audit log --event-type task.status.force_updated`.
	var warnings []string
	for _, f := range updatedFields {
		if f == "sprint" {
			warnings = append(warnings, "the sprint field updates only the DB and frontmatter. To move the file, use task assign/unassign.")
		}
		if f == "status" {
			oldStatus, _ := oldValues["status"].(string)
			newStatus, _ := newValues["status"].(string)
			_ = audit.LogEvent("task.status.force_updated", "task", req.TaskID, "task-update-cli",
				map[string]any{
					"from":   oldStatus,
					"to":     newStatus,
					"source": "hstl task update --status",
				}, "")
			warnings = append(warnings,
				"escape hatch used — audit event task.status.force_updated recorded. Trace via `hstl audit log --event-type task.status.force_updated`. Normal path: task start/complete/reopen.")
		}
	}

	return &domain.UpdateResult{
		TaskID:        req.TaskID,
		UpdatedFields: updatedFields,
		OldValues:     oldValues,
		NewValues:     newValues,
		Warnings:      warnings,
	}, nil
}

// decodeJSONStringSlice decodes a DB-stored JSON string into []string.
// Returns an empty slice on parse failure.
func decodeJSONStringSlice(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "null" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}

// stringSlicesEqual reports whether two string slices contain the same
// items in the same order.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --------------------------------------------------------------------------
// AssignSprint
// --------------------------------------------------------------------------

// AssignSprint assigns Tasks to a Sprint.
// trac: HAR-CM013
func AssignSprint(taskIDs []string, sprintID string) (any, error) {
	root := fileutil.GetProjectRoot()

	// Bootstrap Gate
	bootstrap, err := migration.CheckBootstrapNeeded(root)
	if err != nil {
		return nil, fmt.Errorf("bootstrap check failed: %w", err)
	}
	if bootstrap != nil {
		return migration.BuildBootstrapResponse(bootstrap, root), nil
	}

	gs, gsErr := store.MustGet()
	if gsErr != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	// Verify Sprint existence.
	sprintRec, err := gs.GetSprint(sprintID)
	if err != nil || sprintRec == nil {
		return nil, &apperr.NotFoundError{
			EntityType:   "sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Sprint %s not found. Call sprint_create first.", sprintID),
			RecoveryHint: "call sprint_create first.",
		}
	}

	// Locate the Sprint directory.
	sprintDir, _, err := fileutil.FindSprintDir(sprintID)
	if err != nil {
		return nil, &apperr.NotFoundError{
			EntityType:   "sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Sprint %s directory missing under works/sprints/.", sprintID),
			RecoveryHint: "check works/sprints/ or rerun sprint_create.",
		}
	}
	sprintTasksDir := filepath.Join(sprintDir, "tasks")

	var assigned []string
	var skipped []domain.SkippedItem
	var failed []domain.FailedItem

	for _, taskID := range taskIDs {
		details, dErr := gs.GetTaskDetails(taskID)
		if dErr != nil || details == nil {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: "not_found"})
			continue
		}
		taskTitle := details.Title
		taskType := details.Type
		currentSprint := details.Sprint
		currentStatus := details.Status
		estimate := details.Estimate
		oldFilePath := details.FilePath
		priority := details.Priority
		dependsOnJSON := details.DependsOn
		dependsOnDisplay := formatDependsOnForTable(dependsOnJSON)

		// Validation.
		if currentSprint == sprintID {
			skipped = append(skipped, domain.SkippedItem{TaskID: taskID, Reason: "already_in_sprint"})
			continue
		}
		if !pseudoSprints[currentSprint] && currentSprint != sprintID {
			failed = append(failed, domain.FailedItem{
				TaskID: taskID,
				Reason: fmt.Sprintf("already_in_other_sprint:%s", currentSprint),
			})
			continue
		}
		if currentStatus == "done" {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: "task_done"})
			continue
		}

		// Hard-block on placeholder body at Sprint-assignment time.
		if !allowPlaceholderAssign() && HasPlaceholderBody(taskID) {
			failed = append(failed, domain.FailedItem{
				TaskID: taskID,
				Reason: "placeholder_body",
			})
			continue
		}

		// Locate the file.
		resolvedPath, err := ResolveTaskPath(taskID, oldFilePath)
		if err != nil {
			failed = append(failed, domain.FailedItem{
				TaskID: taskID,
				Reason: fmt.Sprintf("file_not_found:%s", oldFilePath),
			})
			continue
		}

		// Pre-assign sanity check — guards against DB-file_path-trust defects.
		// (1) Fast-fail on archive residue: archive/ files are not move
		// targets.
		// (2) Frontmatter ID mismatch: BLOCK when the file's frontmatter id
		// differs from task_id.
		if reason := assignSanityCheck(taskID, resolvedPath); reason != "" {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: reason})
			continue
		}

		// Move the file.
		if err := os.MkdirAll(sprintTasksDir, 0o755); err != nil {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: fmt.Sprintf("mkdir_failed:%v", err)})
			continue
		}
		newFilePath := filepath.Join(sprintTasksDir, filepath.Base(resolvedPath))
		if err := os.Rename(resolvedPath, newFilePath); err != nil {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: fmt.Sprintf("move_failed:%v", err)})
			continue
		}

		newRelPath := fileutil.ToRepoRelative(newFilePath)
		ck := map[string]bool{"fm": false, "backlog": false, "sprint_md": false, "db": false}

		// Compensating transaction.
		var txErr error
		if err := fileutil.UpdateTaskFrontmatter(newFilePath, map[string]any{"sprint": sprintID}); err != nil {
			txErr = err
		} else {
			ck["fm"] = true
			if _, err := fileutil.RemoveFromBacklogMD(taskID); err == nil {
				ck["backlog"] = true
			}
			if err := fileutil.UpdateSprintMD(sprintID, taskID, taskTitle, taskType, estimate, priority, currentStatus, dependsOnDisplay); err == nil {
				ck["sprint_md"] = true
			}
			if dbErr := gs.UpdateTaskDirect(taskID, ports.TaskUpdateFields{
				Sprint:   &sprintID,
				FilePath: &newRelPath,
			}); dbErr == nil {
				ck["db"] = true
			} else {
				txErr = dbErr
			}
		}

		if txErr != nil {
			// Rollback.
			if ck["sprint_md"] {
				_, _ = fileutil.RemoveRowFromSprintMD(sprintID, taskID)
			}
			if ck["backlog"] {
				_ = fileutil.UpdateBacklogMD(taskID, taskTitle, taskType, estimate, "p2")
			}
			if ck["fm"] {
				_ = fileutil.UpdateTaskFrontmatter(newFilePath, map[string]any{"sprint": "~"})
			}
			_ = os.Rename(newFilePath, resolvedPath)
			failed = append(failed, domain.FailedItem{
				TaskID:      taskID,
				Reason:      fmt.Sprintf("post_move_failed:%v", txErr),
				Rollback:    "compensated",
				Checkpoints: ck,
			})
			continue
		}

		_ = audit.LogEvent("task.sprint_assigned", "task", taskID, "claude", map[string]any{
			"from_sprint":   currentSprint,
			"to_sprint":     sprintID,
			"old_file_path": oldFilePath,
			"new_file_path": newRelPath,
		}, "")
		assigned = append(assigned, taskID)
	}

	var backlogSummary interface{}
	if len(assigned) > 0 {
		bs, _ := fileutil.RebuildBacklogSummary()
		backlogSummary = bs
	}

	if skipped == nil {
		skipped = []domain.SkippedItem{}
	}
	if failed == nil {
		failed = []domain.FailedItem{}
	}

	return &domain.AssignSprintResult{
		SprintID:       sprintID,
		Assigned:       orEmpty(assigned),
		Skipped:        skipped,
		Failed:         failed,
		BacklogSummary: backlogSummary,
	}, nil
}

// --------------------------------------------------------------------------
// UnassignSprint
// --------------------------------------------------------------------------

// UnassignSprint detaches Tasks from a Sprint and returns them to the
// backlog.
// trac: HAR-CM014
func UnassignSprint(taskIDs []string) (any, error) {
	root := fileutil.GetProjectRoot()

	// Bootstrap Gate
	bootstrap, err := migration.CheckBootstrapNeeded(root)
	if err != nil {
		return nil, fmt.Errorf("bootstrap check failed: %w", err)
	}
	if bootstrap != nil {
		return migration.BuildBootstrapResponse(bootstrap, root), nil
	}

	gs, gsErr := store.MustGet()
	if gsErr != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	backlogTasksDir := filepath.Join(root, "works", "tasks")
	_ = os.MkdirAll(backlogTasksDir, 0o755)

	var unassigned []string
	var skipped []domain.SkippedItem
	var failed []domain.FailedItem
	var warnings []string

	for _, taskID := range taskIDs {
		details, dErr := gs.GetTaskDetails(taskID)
		if dErr != nil || details == nil {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: "not_found"})
			continue
		}
		taskTitle := details.Title
		taskType := details.Type
		currentSprint := details.Sprint
		currentStatus := details.Status
		estimate := details.Estimate
		priority := details.Priority
		oldFilePath := details.FilePath

		// Validation.
		if currentSprint == "" {
			skipped = append(skipped, domain.SkippedItem{TaskID: taskID, Reason: "not_in_sprint"})
			continue
		}
		if currentStatus == "done" {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: "task_done"})
			continue
		}

		var warn string
		if currentStatus == "in-progress" {
			warn = fmt.Sprintf("%s: detached while in-progress", taskID)
		}

		// Locate the file.
		resolvedPath, err := ResolveTaskPath(taskID, oldFilePath)
		if err != nil {
			failed = append(failed, domain.FailedItem{
				TaskID: taskID,
				Reason: fmt.Sprintf("file_not_found:%s", oldFilePath),
			})
			continue
		}

		// Move the file (sprint/tasks/ -> works/tasks/).
		newFilePath := filepath.Join(backlogTasksDir, filepath.Base(resolvedPath))
		if err := os.Rename(resolvedPath, newFilePath); err != nil {
			failed = append(failed, domain.FailedItem{TaskID: taskID, Reason: fmt.Sprintf("move_failed:%v", err)})
			continue
		}

		// Post-move bookkeeping.
		var postErr error
		if err := fileutil.UpdateTaskFrontmatter(newFilePath, map[string]any{"sprint": "backlog"}); err != nil {
			postErr = err
		} else {
			_, _ = fileutil.RemoveRowFromSprintMD(currentSprint, taskID)
			_ = fileutil.UpdateBacklogMD(taskID, taskTitle, taskType, estimate, priority)

			newRelPath := fileutil.ToRepoRelative(newFilePath)
			emptyStr := ""
			if dbErr := gs.UpdateTaskDirect(taskID, ports.TaskUpdateFields{
				Sprint:   &emptyStr,
				FilePath: &newRelPath,
			}); dbErr != nil {
				postErr = dbErr
			}
		}

		if postErr != nil {
			_ = os.Rename(newFilePath, resolvedPath)
			failed = append(failed, domain.FailedItem{
				TaskID:   taskID,
				Reason:   fmt.Sprintf("post_move_failed:%v", postErr),
				Rollback: "file_restored",
			})
			continue
		}

		newRelPath := fileutil.ToRepoRelative(newFilePath)
		_ = audit.LogEvent("task.sprint_unassigned", "task", taskID, "claude", map[string]any{
			"from_sprint":   currentSprint,
			"old_file_path": oldFilePath,
			"new_file_path": newRelPath,
			"warn":          warn,
		}, "")

		if warn != "" {
			warnings = append(warnings, warn)
		}
		unassigned = append(unassigned, taskID)
	}

	if len(unassigned) > 0 {
		_, _ = fileutil.RebuildBacklogSummary()
	}

	if skipped == nil {
		skipped = []domain.SkippedItem{}
	}
	if failed == nil {
		failed = []domain.FailedItem{}
	}

	return &domain.UnassignSprintResult{
		Unassigned: orEmpty(unassigned),
		Skipped:    skipped,
		Failed:     failed,
		Warnings:   orEmpty(warnings),
	}, nil
}

// --------------------------------------------------------------------------
// transitionTask — shared helper for status transitions
// --------------------------------------------------------------------------

func transitionTask(taskID, expectedStatus, newStatus, eventType string) (any, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	details, err := gs.GetTaskFull(taskID)
	if err != nil || details == nil {
		return nil, &apperr.NotFoundError{
			EntityType:   "task",
			EntityID:     taskID,
			Message:      fmt.Sprintf("Task %s not found.", taskID),
			RecoveryHint: "use task_list to look up an existing Task ID.",
		}
	}

	currentStatus := details.Status
	filePath := details.FilePath
	sprintID := details.Sprint
	title := details.Title
	taskType := details.Type
	estimate := details.Estimate
	priority := details.Priority
	dependsOnJSON := details.DependsOn

	if currentStatus != expectedStatus {
		return nil, &apperr.InvalidStateError{
			EntityType:    "task",
			EntityID:      taskID,
			CurrentState:  currentStatus,
			ExpectedState: expectedStatus,
			Message:       fmt.Sprintf("status transition not allowed: current=%s, expected=%s", currentStatus, expectedStatus),
			RecoveryHint:  fmt.Sprintf("only Tasks currently in %s can perform this operation.", expectedStatus),
		}
	}

	// Atomicise DB status, file move, frontmatter, and file_path within a
	// single transaction. Previously, each step continued on failure,
	// causing 3-way drift (DB <-> file location <-> frontmatter). Now
	// gs.TransitionTaskAtomic frames the tx boundary and, if the inner
	// fileOp fails, reverts already-performed file changes before
	// tx.Rollback restores everything.
	resolvedPath, resolveErr := ResolveTaskPath(taskID, filePath)
	if resolveErr != nil || resolvedPath == "" {
		return nil, fmt.Errorf("task file not found: %s (glob fallback also failed)", filePath)
	}

	fileUpdated := false
	finalPath := resolvedPath

	txErr := gs.TransitionTaskAtomic(taskID, expectedStatus, newStatus, func() (string, error) {
		// 1) Update the frontmatter in place.
		if err := fileutil.UpdateTaskStatus(resolvedPath, newStatus); err != nil {
			return "", fmt.Errorf("frontmatter update failed: %s — %w", filepath.Base(resolvedPath), err)
		}
		fmUndo := func() { _ = fileutil.UpdateTaskStatus(resolvedPath, currentStatus) }
		_ = fileutil.AppendStatusHistory(resolvedPath, currentStatus, newStatus, "")

		// 2) Auto-move folders for Sprint-unassigned Tasks.
		// done transition -> works/tasks/T*.md -> works/tasks/completed/T*.md
		// reopen transition -> completed/ -> works/tasks/
		if sprintID != "" {
			fileUpdated = true
			return "", nil // No path change.
		}

		var movedPath string
		var didMove bool
		var moveErr error
		switch newStatus {
		case taskStatusDone:
			movedPath, didMove, moveErr = fileutil.MoveTaskToCompleted(resolvedPath)
		case taskStatusTodo, taskStatusInProgress:
			if currentStatus == taskStatusDone {
				movedPath, didMove, moveErr = fileutil.MoveTaskFromCompleted(resolvedPath)
			}
		}
		if moveErr != nil {
			fmUndo()
			return "", fmt.Errorf("file move failed: %w", moveErr)
		}

		fileUpdated = true
		if didMove {
			finalPath = movedPath
			return fileutil.ToRepoRelative(movedPath), nil
		}
		return "", nil
	})

	if txErr != nil {
		return nil, fmt.Errorf("status transition failed (rollback complete): %w", txErr)
	}
	_ = finalPath // Future extension point.

	// SPRINT.md status-column update — outside atomicity (file-based,
	// reconcilable later).
	if sprintID != "" {
		_ = fileutil.UpdateSprintMD(sprintID, taskID, title, taskType, estimate, priority, newStatus, formatDependsOnForTable(dependsOnJSON))
	}

	// Audit log.
	_ = audit.LogEvent(eventType, "task", taskID, "claude", map[string]any{
		"from": currentStatus,
		"to":   newStatus,
	}, "")

	var dependsOn []string
	_ = json.Unmarshal([]byte(dependsOnJSON), &dependsOn)
	if dependsOn == nil {
		dependsOn = []string{}
	}

	prefix, ok := typePrefix[taskType]
	if !ok {
		prefix = "chore"
	}

	sprintVal := interface{}(nil)
	if sprintID != "" {
		sprintVal = sprintID
	}

	tr := &domain.TransitionResult{
		TaskID:         taskID,
		PreviousStatus: currentStatus,
		NewStatus:      newStatus,
		Title:          title,
		Type:           taskType,
		Estimate:       estimate,
		Priority:       priority,
		Sprint:         sprintVal,
		DependsOn:      dependsOn,
		CommitPrefix:   prefix,
	}

	tr.FileUpdated = fileUpdated
	return tr, nil
}

// --------------------------------------------------------------------------
// Internal utilities
// --------------------------------------------------------------------------

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// recordTaskResultJudgement records the task-result verification verdict.
// policy: "off" (skip) | "warn" (warn) | "strict" (block if missing).
func recordTaskResultJudgement(taskID, policy string, missing []domain.MissingFile, checked []string) {
	v := ports.Verdict(ports.VerdictPass)
	snippet := ""
	if len(missing) > 0 {
		paths := make([]string, 0, len(missing))
		for _, m := range missing {
			paths = append(paths, m.Path)
		}
		snippet = strings.Join(paths, ",")
		switch policy {
		case "warn":
			v = ports.VerdictWarn
		case "strict":
			v = ports.VerdictBlock
		default:
			v = ports.VerdictPass
		}
	}
	gatejudgement.RecordJudgement(ports.JudgementRecord{
		GateType:     "task_result",
		ItemID:       taskID,
		Verdict:      v,
		RuleID:       "task.result.file_check",
		RuleSeverity: policy,
		InputSnippet: snippet,
		Actual:       snippet,
		EvidenceRefs: checked,
		Metadata: map[string]string{
			"policy":      policy,
			"missing_cnt": fmt.Sprintf("%d", len(missing)),
			"checked_cnt": fmt.Sprintf("%d", len(checked)),
		},
	})
}
