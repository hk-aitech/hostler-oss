// Package sprint owns Sprint folder creation, SPRINT.md rendering,
// folder moves, and progress aggregation. The filesystem is the SSOT;
// the DB is a cache for fast lookup.
// CEREMONY.md is deprecated. The Sprint ceremony state lives in the
// harness_items DB as the single source of truth; for realtime lookup
// use `hstl harness get sprint <id>` or `hstl sprint progress <id>`.
package sprint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// sprintLogger routes sprint.go's internal diagnostic output through
// the unified logger. The pkg/log globalProxy swaps to io.Discard in
// JSON mode to prevent contamination of stdout JSON.
var sprintLogger = pkglog.NewDefault()

// --------------------------------------------------------------------------
// High-level orchestrators
// --------------------------------------------------------------------------

// Create creates a Sprint (idempotency check + folder + file + DB).
// tasks is the SPRINT.md Task table render input and may be nil.
// trac: HAR-CM001
func Create(id, title, goal string, tasks []map[string]any) (*domain.SprintCreateResult, error) {
	// Idempotency gate — verify that no Sprint already exists at any location.
	root := fileutil.GetProjectRoot()
	for _, loc := range []string{"backlog", "active", "completed", "discarded"} {
		checkPath := filepath.Join(root, "works", "sprints", loc, id, "SPRINT.md")
		if _, statErr := os.Stat(checkPath); statErr == nil {
			return nil, &apperr.RejectedError{
				EntityID:     id,
				Reason:       fmt.Sprintf("Sprint %s already exists at %s.", id, loc),
				RecoveryHint: "delete the Sprint folder first if you want to overwrite.",
			}
		}
	}

	folderPath, err := CreateFolder(id, "backlog")
	if err != nil {
		return nil, fmt.Errorf("Create: folder creation failed (%s): %w", id, err)
	}

	sprintMD := RenderSprintMD(id, title, goal, tasks)
	if writeErr := os.WriteFile(filepath.Join(folderPath, "SPRINT.md"), []byte(sprintMD), 0o644); writeErr != nil {
		return nil, fmt.Errorf("SPRINT.md creation failed: %w", writeErr)
	}

	// CEREMONY.md file rendering is deprecated. Sprint ceremony state is
	// managed via the harness_items DB (SSOT); for realtime lookup use
	// `hstl harness get sprint <id>`.

	relPath := fileutil.ToRepoRelative(folderPath)
	if saveErr := SaveToDB(id, title, goal, "backlog", relPath, nil, nil); saveErr != nil {
		return nil, saveErr
	}

	return &domain.SprintCreateResult{
		SprintID:   id,
		Title:      title,
		Goal:       goal,
		Status:     "backlog",
		FolderPath: folderPath,
	}, nil
}

// Start starts a Sprint (state check + folder move + DB update + Task
// path update).
// trac: HAR-CM002
func Start(id string) (*domain.SprintStartResult, error) {
	rec, err := GetFromDB(id)
	if err != nil {
		return nil, fmt.Errorf("Start: Sprint lookup failed (%s): %w", id, err)
	}

	if rec.Status == "active" {
		return nil, &apperr.RejectedError{
			EntityID:     id,
			Reason:       fmt.Sprintf("Sprint %s is already active.", id),
			RecoveryHint: "Sprint has already been started.",
		}
	}
	if rec.Status == "completed" {
		return nil, &apperr.RejectedError{
			EntityID:     id,
			Reason:       fmt.Sprintf("Sprint %s is already completed.", id),
			RecoveryHint: "a completed Sprint cannot be started.",
		}
	}

	fromLocation := rec.Status
	if fromLocation == "" {
		fromLocation = "backlog"
	}

	newPath, err := MoveSprint(id, fromLocation, "active")
	if err != nil {
		return nil, fmt.Errorf("Start: Sprint folder move failed (%s): %w", id, err)
	}

	today := time.Now().UTC().Format("2006-01-02")

	// SPRINT.md frontmatter update.
	sprintMDPath := filepath.Join(newPath, "SPRINT.md")
	if _, statErr := os.Stat(sprintMDPath); statErr == nil {
		_ = UpdateSprintMDField(sprintMDPath, "status", "active")
		_ = UpdateSprintMDField(sprintMDPath, "started", today)
	}

	// DB update.
	newRelPath := fileutil.ToRepoRelative(newPath)
	if updateErr := UpdateStatusInDB(id, "active", &newRelPath, &today, nil); updateErr != nil {
		return nil, updateErr
	}

	// Batch-update file_path for every Task in this Sprint.
	updateTaskPaths(id, fromLocation, "active", today)

	// CURRENT-FOCUS.md auto-update (prevents stale data accumulation).
	_ = fileutil.UpdateCurrentFocus(id, rec.Title, "sprint-active")

	// Inject the Sprint's implementation Task list into the Dev
	// verification Task body. At sprint-create time the Task list was
	// empty, so the body was a generic template; now regenerate it from
	// the actual Task list. Failure is a warn — sprint start itself
	// remains successful.
	_ = refreshDevVerifyTaskBody(id)

	return &domain.SprintStartResult{
		SprintID:   id,
		Status:     "active",
		StartedAt:  today,
		FolderPath: newPath,
	}, nil
}

// Complete completes a Sprint (state check + folder move + DB update +
// Task path update). Harness Gate verification is the caller's
// responsibility.
// trac: HAR-CM003
func Complete(id string) (*domain.SprintCompleteResult, error) {
	rec, err := GetFromDB(id)
	if err != nil {
		return nil, fmt.Errorf("Complete: Sprint lookup failed (%s): %w", id, err)
	}

	if rec.Status == "completed" {
		return nil, &apperr.RejectedError{
			EntityID:     id,
			Reason:       fmt.Sprintf("Sprint %s is already in the completed state.", id),
			RecoveryHint: "Sprint has already been completed.",
		}
	}

	fromLocation := rec.Status
	if fromLocation == "" {
		fromLocation = "active"
	}

	// Before MoveSprint, repair the frontmatter consistency of every
	// Task in this Sprint. If a Task has DB status=done but frontmatter
	// status=todo/in-progress (because of an old bug or partial path),
	// MoveSprint would leave a stale frontmatter under completed/.
	// Batch-overwrite the file frontmatter to done for every Task whose
	// DB status is done before moving.
	_ = reconcileTaskFrontmatterBeforeMove(id)

	// Run the SPRINT.md ceremony-template injection + heuristic check
	// **before** MoveSprint and the status transition.
	// Old flow (defect):
	// MoveSprint -> status=completed -> EnsureCeremonySections ->
	// StampWithValidation -> heuristic rejects -> only the stamp
	// fails while Complete returns success -> Sprint ends up in
	// completed state with Phase 5/6/9 placeholders intact and the
	// HMAC signature missing.
	// Fixed flow:
	// inject the template at the active location -> run the
	// heuristic in a fail-closed mode -> if placeholders remain,
	// abort Complete (BLOCKED) and leave the Sprint state unchanged
	// > the AI/user fills in the real Phase 5/6/9 content and
	// re-invokes -> when the heuristic passes, MoveSprint + DB
	// transition + stamp run.
	// Exception: when HSTL_SPRINT_STAMP_POLICY=off, the heuristic
	// itself is skipped, so the legacy transition is allowed (the
	// explicit override path).
	activeSprintMDPath := filepath.Join(fileutil.GetProjectRoot(), "works", "sprints", fromLocation, id, "SPRINT.md")
	if _, statErr := os.Stat(activeSprintMDPath); statErr == nil {
		if tplErr := EnsureCeremonySections(activeSprintMDPath); tplErr != nil {
			sprintLogger.Warn("ceremony template injection failed", "sprint", id, "err", tplErr.Error())
		}
	}

	// side_effects tracking. Accumulates the files modified during
	// sprint.Complete so the caller (CLI / adapter) can surface "which
	// files are subject to re-commit" via the JSON response.
	sideEffects := []domain.SprintSideEffect{}

	// Right before MoveSprint, run a frontmatter id sanity check.
	// BLOCK when SPRINT.md's frontmatter id does not match the sprint
	// id (i.e. another sprint folder ended up here). Extends the
	// assignSanityCheck pattern up into the sprint lifecycle.
	{
		fromPath := filepath.Join(fileutil.GetProjectRoot(), "works", "sprints", fromLocation, id)
		sprintMDPath := filepath.Join(fromPath, "SPRINT.md")
		if _, statErr := os.Stat(sprintMDPath); statErr == nil {
			sanity := fileutil.CheckFrontmatterIDSanity(id, sprintMDPath, "sprint")
			if sanity.Reason != "" {
				return nil, &apperr.RejectedError{
					EntityID:     id,
					Reason:       fmt.Sprintf("Sprint %s complete aborted — sanity check failed: %s", id, sanity.Reason),
					RecoveryHint: "verify that SPRINT.md frontmatter id matches the sprint ID. Run backlog sync --dry-run to inspect consistency.",
				}
			}
		}
	}

	newPath, err := MoveSprint(id, fromLocation, "completed")
	if err != nil {
		return nil, fmt.Errorf("Complete: Sprint folder move failed (%s): %w", id, err)
	}
	sideEffects = append(sideEffects, domain.SprintSideEffect{
		File:   fileutil.ToRepoRelative(newPath),
		Change: "moved",
	})

	today := time.Now().UTC().Format("2006-01-02")

	// SPRINT.md frontmatter update.
	sprintMDPath := filepath.Join(newPath, "SPRINT.md")
	if _, statErr := os.Stat(sprintMDPath); statErr == nil {
		_ = UpdateSprintMDField(sprintMDPath, "status", "completed")
		_ = UpdateSprintMDField(sprintMDPath, "completed", today)
		sideEffects = append(sideEffects, domain.SprintSideEffect{
			File:   fileutil.ToRepoRelative(sprintMDPath),
			Change: "updated",
		})
	}

	// DB update.
	newRelPath := fileutil.ToRepoRelative(newPath)
	if updateErr := UpdateStatusInDB(id, "completed", &newRelPath, nil, &today); updateErr != nil {
		return nil, updateErr
	}

	// Batch-update file_path for every Task in this Sprint.
	updateTaskPaths(id, fromLocation, "completed", today)

	// Force-sync Task DB status (file is SSOT). Blocks the drift where
	// the file frontmatter is status=done but DB status differs. Solves
	// the issue at the sprint-complete ceremony itself rather than
	// after the fact via `backlog sync --apply`.
	if forced := reconcileTaskDBStatusAfterMove(id); forced > 0 {
		sprintLogger.Info("sprint complete DB status sync",
			"sprint", id, "force_updated", forced)
	}

	// CURRENT-FOCUS.md auto-update — reflect the idle state after
	// Sprint completion.
	if err := fileutil.UpdateCurrentFocus("", "", "idle"); err == nil {
		sideEffects = append(sideEffects, domain.SprintSideEffect{
			File:   "works/CURRENT-FOCUS.md",
			Change: "updated",
		})
	}

	return &domain.SprintCompleteResult{
		SprintID:    id,
		Status:      "completed",
		CompletedAt: today,
		FolderPath:  newPath,
		SideEffects: sideEffects,
	}, nil
}

// taskStatusDone — used by reconcileTaskFrontmatterBeforeMove to repair
// Task frontmatter just before sprint_complete moves the folder.
const taskStatusDone = "done"

// reconcileTaskFrontmatterBeforeMove batch-fixes Task file frontmatter
// to "done" for every Task whose DB status is done, immediately before
// the Sprint folder is moved into completed/. Prevents the recurrence
// of the defect where Tasks with stale frontmatter end up in the
// completed folder after MoveSprint due to a partial task_complete
// path or an old bug. Returns the number of Tasks repaired.
// Safety:
// Tasks whose DB status is not "done" are not touched (the cascade
// check would already have BLOCKED sprint_complete in that case).
// Missing files or read failures are silently skipped (best-effort).
// No DB writes. Only file frontmatter is updated.
func reconcileTaskFrontmatterBeforeMove(sprintID string) int {
	gs := store.Get()
	if gs == nil {
		return 0
	}
	filePaths, err := gs.GetSprintTaskFilePaths(sprintID)
	if err != nil {
		return 0
	}

	root := fileutil.GetProjectRoot()
	reconciled := 0
	for _, filePath := range filePaths {
		if filePath == "" {
			continue
		}
		absPath := filepath.Join(root, filePath)
		if _, statErr := os.Stat(absPath); statErr != nil {
			continue
		}
		// Overwrite frontmatter to done (no-op if already done).
		if updErr := fileutil.UpdateTaskStatus(absPath, taskStatusDone); updErr == nil {
			reconciled++
		}
	}
	return reconciled
}

// reconcileTaskDBStatusAfterMove force-syncs the DB status of every
// Task in a Sprint to the file SSOT immediately after sprint complete.
// Background: after Sprint complete, some Tasks' DB status remained
// stuck at todo/in-progress while the file had already been written to
// done with a completion HMAC signature. The DB simply lagged. The
// `backlog sync --apply` path detects and repairs the drift, but it is
// a cure-after-occurrence pattern that leaves a drift window.
// Design — under the "file is SSOT, DB is supplementary" principle,
// when the file is done the DB is brought into line. This is the
// inverse direction of reconcileTaskFrontmatterBeforeMove (DB->file)
// and runs as the final step of the sprint-complete ceremony. When an
// individual Task already completed normally it is a no-op; in the
// rare drift case it force-syncs and records the audit event
// `task.status.force_updated`.
// Returns: the number of Tasks force-updated (for diagnostics /
// logging).
func reconcileTaskDBStatusAfterMove(sprintID string) int {
	gs := store.Get()
	if gs == nil {
		return 0
	}
	filePaths, err := gs.GetSprintTaskFilePaths(sprintID)
	if err != nil || len(filePaths) == 0 {
		return 0
	}

	root := fileutil.GetProjectRoot()
	forceUpdated := 0
	for _, filePath := range filePaths {
		if filePath == "" {
			continue
		}
		absPath := filepath.Join(root, filePath)
		fm, readErr := fileutil.ReadTaskFrontmatter(absPath)
		if readErr != nil || fm == nil {
			continue
		}
		fmStatus, _ := fm["status"].(string)
		if fmStatus != taskStatusDone {
			continue
		}
		taskID, _ := fm["id"].(string)
		if taskID == "" {
			continue
		}
		dbStatus, statErr := gs.GetTaskStatus(taskID)
		if statErr != nil || dbStatus == taskStatusDone {
			continue
		}
		if err := gs.UpdateTaskStatusDirect(taskID, taskStatusDone); err != nil {
			sprintLogger.Warn("sprint complete DB status sync failed",
				"sprint", sprintID, "task", taskID, "err", err.Error())
			continue
		}
		forceUpdated++
		_ = audit.LogEvent("task.status.force_updated", "task", taskID, "sprint-complete",
			map[string]any{
				"old_db_status": dbStatus,
				"new_db_status": taskStatusDone,
				"reason":        "sprint-complete-auto-sync",
				"sprint":        sprintID,
				"file":          filePath,
			}, "")
	}
	return forceUpdated
}

// updateTaskPaths batch-updates the file_path for every Task in this
// Sprint when the Sprint folder moves.
// Two-phase behaviour:
//	Phase 1 — prefix REPLACE: only updates rows where
//	tasks.sprint=sprintID.
//	Phase 2 — filesystem SSOT self-heal: walks the toLocation's
//	sprint tasks/ folder and, by frontmatter id, force-fixes
//	sprint + file_path. Reconciles tasks where the DB UPDATE
//	at task-assign time was missed or where a multi-worktree
//	race left the sprint column NULL or stale, using the
//	file location as the SSOT.
// Recurrence background: ceremony after ceremony, the team had to run
// `backlog sync` for manual self-heal. The root cause was that Phase
// 1's `WHERE sprint=?` filter updated zero rows when the sprint column
// was empty, so file_path drift remained stale.
func updateTaskPaths(sprintID, fromLocation, toLocation, _ string) {
	gs := store.Get()
	if gs == nil {
		return
	}
	// Phase 1: prefix REPLACE — only updates file_path for tasks where
	// sprint=sprintID.
	_ = gs.UpdateTaskPathsForSprint(sprintID, fromLocation+"/"+sprintID+"/", toLocation+"/"+sprintID+"/")

	// Phase 2: filesystem SSOT self-heal.
	reconcileTaskPathsFromFS(sprintID, toLocation)
}

// reconcileTaskPathsFromFS walks the sprint's toLocation tasks/ folder
// and, treating each task file's frontmatter id as the SSOT,
// force-updates DB tasks.sprint + tasks.file_path.
// Safety:
// Missing folder -> silent skip (sprint with no tasks).
// Files without a frontmatter id -> skip (invalid file).
// UpdateTaskDirect failure -> silent skip (the next sync catches it).
// Call cost: O(N) frontmatter parses, with N typically < 10.
func reconcileTaskPathsFromFS(sprintID, toLocation string) {
	gs := store.Get()
	if gs == nil {
		return
	}
	root := fileutil.GetProjectRoot()
	tasksDir := filepath.Join(root, "works", "sprints", toLocation, sprintID, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		filePath := filepath.Join(tasksDir, e.Name())
		fm, ferr := fileutil.ReadTaskFrontmatter(filePath)
		if ferr != nil || fm == nil {
			continue
		}
		taskIDRaw, ok := fm["id"]
		if !ok {
			continue
		}
		taskID := strings.TrimSpace(fmt.Sprintf("%v", taskIDRaw))
		if taskID == "" || !strings.HasPrefix(taskID, "T") {
			continue
		}
		relPath := fileutil.ToRepoRelative(filePath)
		sp := sprintID
		_ = gs.UpdateTaskDirect(taskID, ports.TaskUpdateFields{
			Sprint:   &sp,
			FilePath: &relPath,
		})
	}
}

// --------------------------------------------------------------------------
// folder management
// --------------------------------------------------------------------------

// CreateFolder creates the Sprint folder and its tasks/ directory.
// location: "backlog" | "active" | "completed" | "discarded".
// Returns: the created sprint folder path.
// trac: HAR-CM001
func CreateFolder(sprintID, location string) (string, error) {
	if location == "" {
		location = "backlog"
	}
	root := fileutil.GetProjectRoot()
	sprintDir := filepath.Join(root, "works", "sprints", location, sprintID)
	tasksDir := filepath.Join(sprintDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		return "", &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Sprint folder creation failed: %v", err),
			RecoveryHint: "verify directory permissions.",
		}
	}
	return sprintDir, nil
}

// MoveSprint moves the Sprint folder from from_location to
// to_location. Tries `git mv` first and falls back to os.Rename on
// failure.
// trac: HAR-CM002
func MoveSprint(sprintID, fromLocation, toLocation string) (string, error) {
	root := fileutil.GetProjectRoot()
	src := filepath.Join(root, "works", "sprints", fromLocation, sprintID)
	dstParent := filepath.Join(root, "works", "sprints", toLocation)
	dst := filepath.Join(dstParent, sprintID)

	if _, err := os.Stat(src); os.IsNotExist(err) {
		// Idempotent handling: if src is missing but dst already
		// exists, the user moved it manually or a previous run failed
		// partway through. When the folder is already at the
		// destination, treat the filesystem step as a successful
		// no-op and return dst so the caller only updates the DB
		// status.
		if _, dstErr := os.Stat(dst); dstErr == nil {
			return dst, nil
		}
		return "", &apperr.NotFoundError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Sprint folder not found: %s", src),
			RecoveryHint: "use sprint_list to verify the Sprint list.",
		}
	}

	if err := os.MkdirAll(dstParent, 0o755); err != nil {
		return "", &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("destination directory creation failed: %v", err),
			RecoveryHint: "verify directory permissions.",
		}
	}

	// Try git mv.
	cmd := exec.Command("git", "mv", src, dst)
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		// On git mv failure, fall back to os.Rename.
		if renameErr := os.Rename(src, dst); renameErr != nil {
			return "", &apperr.InvalidStateError{
				EntityType:   "Sprint",
				EntityID:     sprintID,
				Message:      fmt.Sprintf("Sprint folder move failed: %v", renameErr),
				RecoveryHint: "verify directory permissions.",
			}
		}
	}

	// Remove the source folder if it remains (when git mv moved only
	// part of the contents).
	if src != dst {
		if _, err := os.Stat(src); err == nil {
			_ = os.RemoveAll(src)
		}
	}

	return dst, nil
}

// --------------------------------------------------------------------------
// template rendering
// --------------------------------------------------------------------------

// RenderSprintMD renders the SPRINT.md content.
// trac: HAR-CM001
func RenderSprintMD(sprintID, title, goal string, tasks []map[string]any) string {
	today := time.Now().UTC().Format("2006-01-02")

	// Build the Task table rows.
	var taskRows strings.Builder
	for _, t := range tasks {
		taskID := strVal(t, "id", strVal(t, "task_id", ""))
		taskTitle := strVal(t, "title", "")
		if taskID == "" && taskTitle == "" {
			continue
		}
		taskType := strValDefault(t, "type", "feature")
		taskEst := strValDefault(t, "estimate", "M")
		taskPriority := strValDefault(t, "priority", "p2")
		taskStatus := strValDefault(t, "status", "todo")
		taskDeps := parseDepsField(t)
		taskRows.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
			taskID, taskTitle, taskType, taskEst, taskPriority, taskStatus, taskDeps))
	}

	return fmt.Sprintf(`---
id: %s
title: "%s"
status: backlog
goal: "%s"
started: ~
completed: ~
created: %s
---

# %s: %s

## Goal

%s

## Task list

| ID | title | type | estimate | priority | status | depends_on |
|----|------|------|----------|----------|--------|------------|
%s
## Done Criteria

- [ ] every Task transitions to `+"`done`"+`
- [ ] sprint:complete Phases all executed
`, sprintID, title, goal, today, sprintID, title, goal, taskRows.String())
}

// CEREMONY.md rendering helpers (RenderCeremonyMD / loadCeremonySection /
// renderCeremonyItem / sprintIDPrefix / defaultCeremonyStart /
// sprintCeremonyPostPhaseItems / defaultCeremonyComplete /
// ceremonyTemplateSection*) were removed. The CEREMONY.md file is no
// longer rendered; harness_items is the only ceremony state SSOT.

// --------------------------------------------------------------------------
// SPRINT.md frontmatter update
// --------------------------------------------------------------------------

// UpdateSprintMDField updates a specific field in the SPRINT.md
// frontmatter.
// trac: HAR-CM004
func UpdateSprintMDField(sprintMDPath, field, value string) error {
	data, err := os.ReadFile(sprintMDPath)
	if err != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Sprint",
			Message:      fmt.Sprintf("SPRINT.md read failed: %v", err),
			RecoveryHint: "verify the file path and permissions.",
		}
	}

	lines := strings.Split(string(data), "\n")
	updated := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, field+":") {
			updated = append(updated, field+": "+value)
		} else {
			updated = append(updated, line)
		}
	}
	return os.WriteFile(sprintMDPath, []byte(strings.Join(updated, "\n")), 0o644)
}

// ReadSprintMDField returns the single top-level field value from
// SPRINT.md's frontmatter. Returns "" if the field is missing or the
// file cannot be read.
// Use case: enriching a sprint list response with a field such as
// track_id that lives only in the frontmatter, not the DB. Simple
// prefix match — nested YAML structures are not supported.
// trac: WRK-QR007
func ReadSprintMDField(sprintMDPath, field string) string {
	data, err := os.ReadFile(sprintMDPath)
	if err != nil {
		return ""
	}
	prefix := field + ":"
	fmDelim := 0
	for line := range strings.SplitSeq(string(data), "\n") {
		trim := strings.TrimSpace(line)
		if trim == "---" {
			fmDelim++
			if fmDelim >= 2 {
				return ""
			}
			continue
		}
		if fmDelim == 1 && strings.HasPrefix(line, prefix) {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, prefix)), `"'`)
		}
	}
	return ""
}

// AddSprintMDField adds a new top-level field to the SPRINT.md
// frontmatter.
// Unlike UpdateSprintMDField, this inserts `<field>: <value>` on the
// line immediately after the first `---` only when the field does not
// already exist. When it does exist, the call is delegated to
// UpdateSprintMDField (in-place replacement).
// trac: HAR-CM004
func AddSprintMDField(sprintMDPath, field, value string) error {
	if existing := ReadSprintMDField(sprintMDPath, field); existing != "" {
		return UpdateSprintMDField(sprintMDPath, field, value)
	}
	data, err := os.ReadFile(sprintMDPath)
	if err != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Sprint",
			Message:      fmt.Sprintf("SPRINT.md read failed: %v", err),
			RecoveryHint: "verify the file path and permissions.",
		}
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines)+1)
	inserted := false
	for i, line := range lines {
		out = append(out, line)
		if !inserted && i == 0 && strings.TrimSpace(line) == "---" {
			out = append(out, field+": "+value)
			inserted = true
		}
	}
	if !inserted {
		// No frontmatter start marker — return unchanged.
		return nil
	}
	return os.WriteFile(sprintMDPath, []byte(strings.Join(out, "\n")), 0o644)
}

// ReplaceFrontmatterField replaces the value of a frontmatter field
// inside SPRINT.md content.
// In-memory variant of UpdateSprintMDField. Used in flows that perform
// several field replacements + body substitutions on a single content
// string and then save the whole thing with one WriteFile. The caller
// owns the content because this function does not perform I/O.
// Behaviour:
// The value is written using the YAML double-quoted string format
// (`%q`) so special characters (`"`, backslash, etc.) are escaped
// safely.
// Replaces only the first match inside the frontmatter (the first
// `---` ... `---` block).
// If no field matches, returns the original content unchanged
// (no error).
// If the content has no frontmatter, returns it unchanged.
// Note: this function is line-prefix based, so when the field lives
// inside a nested YAML structure (e.g. `items.title`) only the
// top-level field is found. Use only for flat frontmatter structures.
// trac: HAR-CM004
func ReplaceFrontmatterField(content, field, value string) string {
	lines := strings.Split(content, "\n")
	fmDelim := 0
	prefix := field + ":"
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			fmDelim++
			if fmDelim >= 2 {
				break
			}
			continue
		}
		if fmDelim == 1 && strings.HasPrefix(line, prefix) {
			lines[i] = fmt.Sprintf("%s: %q", field, value)
			return strings.Join(lines, "\n")
		}
	}
	return content
}

// --------------------------------------------------------------------------
// DB CRUD
// --------------------------------------------------------------------------

// SaveToDB issues an INSERT OR REPLACE for the Sprint into the DB.
// trac: HAR-CM001
func SaveToDB(sprintID, title, goal, status, folderPath string, startedAt, completedAt *string) error {
	gs, err := store.MustGet()
	if err != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      "store is not initialised",
			RecoveryHint: "run project_migrate to initialise the DB.",
		}
	}
	sp := &ports.SprintRecord{
		SprintID:   sprintID,
		Title:      title,
		Goal:       goal,
		Status:     status,
		FolderPath: folderPath,
	}
	if startedAt != nil {
		sp.StartedAt = *startedAt
	}
	if completedAt != nil {
		sp.CompletedAt = *completedAt
	}
	if saveErr := gs.SaveSprint(sp); saveErr != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Sprint DB save failed: %v", saveErr),
			RecoveryHint: "verify the DB state and retry.",
		}
	}
	return nil
}

// UpdateStatusInDB updates Sprint status / folder_path / started_at /
// completed_at.
// trac: HAR-CM004
func UpdateStatusInDB(sprintID, status string, folderPath, startedAt, completedAt *string) error {
	gs, err := store.MustGet()
	if err != nil {
		return &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      "store is not initialised",
			RecoveryHint: "run project_migrate to initialise the DB.",
		}
	}
	opts := &ports.SprintUpdateOpts{}
	if folderPath != nil {
		opts.FolderPath = *folderPath
	}
	if startedAt != nil {
		opts.StartedAt = *startedAt
	}
	if completedAt != nil {
		opts.CompletedAt = *completedAt
	}
	return gs.UpdateSprintStatus(sprintID, status, opts)
}

// GetFromDB looks up the Sprint in the DB.
// trac: WRK-QR007
func GetFromDB(sprintID string) (*domain.SprintRecord, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      "store is not initialised",
			RecoveryHint: "run project_migrate to initialise the DB.",
		}
	}
	rec, err := gs.GetSprint(sprintID)
	if err != nil {
		return nil, &apperr.NotFoundError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Sprint %s not found.", sprintID),
			RecoveryHint: "use sprint_list to verify the Sprint list.",
		}
	}
	if rec == nil {
		return nil, &apperr.NotFoundError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Sprint %s not found.", sprintID),
			RecoveryHint: "use sprint_list to verify the Sprint list.",
		}
	}
	return &domain.SprintRecord{
		SprintID:    rec.SprintID,
		Title:       rec.Title,
		Status:      rec.Status,
		FolderPath:  rec.FolderPath,
		StartedAt:   rec.StartedAt,
		CompletedAt: rec.CompletedAt,
		Goal:        rec.Goal,
		CreatedAt:   rec.CreatedAt,
		UpdatedAt:   rec.UpdatedAt,
	}, nil
}

// ListFromDB looks up Sprints from the DB.
// filter convention:
// "" (default): excludes discarded; surfaces only Sprints in
// active management.
// "all" : everything (including discarded).
// "discarded": only discarded Sprints.
// "backlog|active|completed": only the matching status (preserves
// prior behaviour).
// trac: WRK-QR006
func ListFromDB(status string) ([]domain.SprintRecord, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, &apperr.InvalidStateError{
			EntityType:   "Sprint",
			Message:      "store is not initialised",
			RecoveryHint: "run project_migrate to initialise the DB.",
		}
	}

	// Filter normalisation — the store layer only handles "" /
	// "<status>", so the "all" / default post-filter logic is
	// implemented here.
	var storeFilter string
	excludeDiscarded := false
	switch status {
	case "", "default":
		// Default: fetch all -> filter out discarded below.
		storeFilter = ""
		excludeDiscarded = true
	case "all":
		storeFilter = ""
		// Include everything (return discarded as well).
	default:
		storeFilter = status
	}

	recs, err := gs.ListSprints(storeFilter)
	if err != nil {
		return nil, &apperr.InvalidStateError{
			EntityType:   "Sprint",
			Message:      fmt.Sprintf("Sprint list lookup failed: %v", err),
			RecoveryHint: "verify the DB state.",
		}
	}
	result := make([]domain.SprintRecord, 0, len(recs))
	for _, rec := range recs {
		if excludeDiscarded && rec.Status == "discarded" {
			continue
		}
		result = append(result, domain.SprintRecord{
			SprintID:    rec.SprintID,
			Title:       rec.Title,
			Status:      rec.Status,
			FolderPath:  rec.FolderPath,
			StartedAt:   rec.StartedAt,
			CompletedAt: rec.CompletedAt,
			Goal:        rec.Goal,
			CreatedAt:   rec.CreatedAt,
			UpdatedAt:   rec.UpdatedAt,
		})
	}
	return result, nil
}

// AggregateProgress aggregates the Task state for a Sprint.
// trac: WRK-QR008
func AggregateProgress(sprintID string) (*domain.ProgressResult, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      "store is not initialised",
			RecoveryHint: "run project_migrate to initialise the DB.",
		}
	}
	pr, err := gs.AggregateSprintProgress(sprintID)
	if err != nil {
		return nil, &apperr.InvalidStateError{
			EntityType:   "Sprint",
			EntityID:     sprintID,
			Message:      fmt.Sprintf("Task state aggregation failed: %v", err),
			RecoveryHint: "verify the DB state.",
		}
	}
	burndown := make([]domain.BurndownEntry, 0, len(pr.Burndown))
	for _, b := range pr.Burndown {
		burndown = append(burndown, domain.BurndownEntry{Date: b.Date, Done: b.Done})
	}
	return &domain.ProgressResult{
		SprintID:   pr.SprintID,
		Done:       pr.Done,
		InProgress: pr.InProgress,
		Todo:       pr.Todo,
		Total:      pr.Total,
		Percent:    pr.Percent,
		Burndown:   burndown,
	}, nil
}

// --------------------------------------------------------------------------
// Internal utilities
// --------------------------------------------------------------------------

// strVal safely extracts a string value from a map.
func strVal(m map[string]any, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

// strValDefault behaves like strVal but also substitutes the fallback
// for empty strings.
func strValDefault(m map[string]any, key, fallback string) string {
	s := strVal(m, key, fallback)
	if s == "" {
		return fallback
	}
	return s
}

// parseDepsField formats the depends_on field as a display string,
// handling []string, []any and string inputs.
func parseDepsField(t map[string]any) string {
	if deps, ok := t["depends_on"].([]string); ok {
		if len(deps) == 0 {
			return "—"
		}
		return strings.Join(deps, ", ")
	}
	if deps, ok := t["depends_on"].([]any); ok {
		if len(deps) == 0 {
			return "—"
		}
		var ss []string
		for _, d := range deps {
			if s, ok := d.(string); ok {
				ss = append(ss, s)
			}
		}
		if len(ss) == 0 {
			return "—"
		}
		return strings.Join(ss, ", ")
	}
	if s := strVal(t, "depends_on", ""); s != "" {
		return s
	}
	return "—"
}
