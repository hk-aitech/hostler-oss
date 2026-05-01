package backlog

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

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// completedSprintsDisplayLimit — Number of recent completed Sprints displayed in BACKLOG.md.
const completedSprintsDisplayLimit = 5

// Constant used to prevent SSOT inversion.
// When the file is under `works/sprints/completed/<sprint-id>/tasks/` and DB status=done but
// the frontmatter is not done, treat the frontmatter as stale and update the file.
const (
	issueDBStatusMismatch       = "db_status_mismatch"
	issueDBTaskStatusRegression = "db_task_status_regression" // blocks DB done -> file todo regression
	issueFileStatusStale        = "file_status_stale_after_complete"
	issueCompletedAnomaly       = "completed_location_status_anomaly"
	issueDBOrphanTask           = "db_orphan_task" // exists in DB but missing file and not listed in BACKLOG.md
	fixFileStatusFixed          = "file_status_fixed"
	fixDBOrphanTaskDeleted      = "db_orphan_task_deleted" // 
	taskStatusDone              = "done"
	sprintLocationCompleted     = "completed"
)

var reTaskFile = regexp.MustCompile(`^(T\d+)-.*\.md$`)
var reBacklogTaskID = regexp.MustCompile(`^\|\s*(T\d+)\s*\|`)

// --------------------------------------------------------------------------
// Sync — orchestrator
// --------------------------------------------------------------------------

// Sync verifies consistency across file/DB/BACKLOG.md and optionally auto-fixes.
// dryRun=true performs detection only (no side effects); false repairs auto_fixable issues.
func Sync(dryRun bool) (*SyncResult, error) {
	root := fileutil.GetProjectRoot()
	taskFiles, err := globTaskFiles(root)
	if err != nil {
		return nil, fmt.Errorf("task file collection failed: %w", err)
	}

	// Phase 1: Check
	var allIssues []Issue

	dupIssues := checkDuplicateIDs(taskFiles)
	allIssues = append(allIssues, dupIssues...)

	locIssues := checkFileLocationVsFrontmatter(taskFiles, root)
	allIssues = append(allIssues, locIssues...)

	dbIssues, err := checkDBVsFrontmatter(taskFiles, root)
	if err != nil {
		return nil, fmt.Errorf("DB vs frontmatter check failed: %w", err)
	}
	allIssues = append(allIssues, dbIssues...)

	backlogIssues := checkBacklogMDVsFiles(taskFiles, root)
	allIssues = append(allIssues, backlogIssues...)

	fmIssues := checkFrontmatterParseErrors(taskFiles)
	allIssues = append(allIssues, fmIssues...)

	summaryIssues, err := checkBacklogSummaryMetadata(root)
	if err != nil {
		return nil, fmt.Errorf("BACKLOG.md summary-metadata check failed: %w", err)
	}
	allIssues = append(allIssues, summaryIssues...)

	// (ISS-20260413-001): Cross-check Sprint files with the DB and recover any missing entries.
	sprintIssues, err := checkDBVsSprintFiles(root)
	if err != nil {
		return nil, fmt.Errorf("Sprint file vs DB check failed: %w", err)
	}
	allIssues = append(allIssues, sprintIssues...)

	// (, ISS-20260422-005): DB orphan — neither the file nor BACKLOG.md mentions it.
	orphanIssues, err := checkDBOrphanTasks(taskFiles, root)
	if err != nil {
		return nil, fmt.Errorf("DB orphan Task check failed: %w", err)
	}
	allIssues = append(allIssues, orphanIssues...)

	// Phase 2: Apply (only when dry_run=false)
	var allFixes []Fix
	if !dryRun {
		allFixes, err = applyFixes(allIssues)
		if err != nil {
			return nil, fmt.Errorf("auto-repair failed: %w", err)
		}
	}

	// Strip internal repair data before response serialisation.
	cleanedIssues := make([]Issue, len(allIssues))
	for i, iss := range allIssues {
		iss.fm = nil
		iss.dbSprints = nil
		iss.sprintInfo = nil
		cleanedIssues[i] = iss
	}

	autoFixable := 0
	manual := 0
	for _, iss := range cleanedIssues {
		if iss.AutoFixable {
			autoFixable++
		} else {
			manual++
		}
	}

	if len(allFixes) > 0 {
		_ = audit.LogEvent(
			"backlog_sync.completed",
			"project", "global",
			"claude",
			map[string]any{"fixes": len(allFixes), "issues": len(cleanedIssues)},
			"",
		)
	}

	return &SyncResult{
		Issues: cleanedIssues,
		Fixes:  allFixes,
		Summary: Summary{
			TotalIssues:      len(cleanedIssues),
			AutoFixable:      autoFixable,
			AutoFixed:        len(allFixes),
			ManualRequired:   manual,
			DryRun:           dryRun,
			TaskFilesScanned: len(taskFiles),
		},
	}, nil
}

// --------------------------------------------------------------------------
// Utility functions
// --------------------------------------------------------------------------

// globTaskFiles collects T*.md files under works/.
func globTaskFiles(root string) ([]string, error) {
	var results []string
	worksDir := filepath.Join(root, "works")

	err := filepath.WalkDir(worksDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // ignore search errors
		}
		if d.IsDir() {
			return nil
		}
		if reTaskFile.MatchString(d.Name()) {
			results = append(results, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return results, nil
}

// extractTaskIDFromFilename extracts the Task ID (Tdddd) from a file name.
func extractTaskIDFromFilename(name string) string {
	m := reTaskFile.FindStringSubmatch(filepath.Base(name))
	if m != nil {
		return m[1]
	}
	return ""
}

// getDirLocation analyses the location context from the file path.
func getDirLocation(filePath, root string) dirLocation {
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		return dirLocation{}
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")

	// Strict path match..
	// Only `works/tasks/T*.md` (root, length 3) is inBacklog.
	// `works/tasks/completed/T*.md` (length 4) is the completed-move target — not inBacklog.
	// Earlier implementations checked only parts[1] == "tasks", so they misclassified `works/tasks/completed/*`
	// and `works/tasks/*/subdir/*` as inBacklog=true.
	inBacklog := len(parts) == 3 && parts[0] == "works" && parts[1] == "tasks"
	// `works/tasks/completed/T*.md` is identified separately.
	inTasksCompleted := len(parts) == 4 && parts[0] == "works" && parts[1] == "tasks" && parts[2] == "completed"
	inSprint := len(parts) == 6 &&
		parts[0] == "works" &&
		parts[1] == "sprints" &&
		parts[4] == "tasks"

	var sprintID, sprintStatus string
	if inSprint {
		sprintStatus = parts[2] // active / backlog / completed
		sprintID = parts[3]     // sprint-XX
	}

	return dirLocation{
		inBacklogTasks:   inBacklog,
		inTasksCompleted: inTasksCompleted,
		inSprintTasks:    inSprint,
		sprintID:         sprintID,
		sprintStatus:     sprintStatus,
	}
}

// parseBacklogMDTaskIDs returns the set of Task IDs listed in BACKLOG.md.
func parseBacklogMDTaskIDs(root string) map[string]bool {
	backlogPath := filepath.Join(root, "works", "tasks", "BACKLOG.md")
	data, err := os.ReadFile(backlogPath)
	if err != nil {
		return map[string]bool{}
	}

	ids := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		m := reBacklogTaskID.FindStringSubmatch(line)
		if m != nil {
			ids[m[1]] = true
		}
	}
	return ids
}

// readFrontmatterSafe reads the frontmatter and returns nil on failure.
func readFrontmatterSafe(filePath string) map[string]any {
	fm, err := fileutil.ReadTaskFrontmatter(filePath)
	if err != nil {
		return nil
	}
	return fm
}

// fmString safely extracts a string value from a frontmatter map.
// When a YAML null value (`~`, `null`) is mapped to Go nil, returns the empty string.
// When YAML parses a literal such as `2026-03-25` into time.Time,
// normalise to `YYYY-MM-DD` instead of the default timezone-bearing format
// (`2026-03-25 00:00:00 +0000 UTC`) so the DB and display layer do not drift.
func fmString(fm map[string]any, key string) string {
	if fm == nil {
		return ""
	}
	v, ok := fm[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case time.Time:
		return s.Format("2006-01-02")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// NormalizeSprintField normalises the frontmatter sprint field to its canonical form.
// The SSOT moved to fileutil.NormalizeSprintField. This function
// is a wrapper kept for backward compatibility with existing external callers. New callers should
// use fileutil.NormalizeSprintField directly.
func NormalizeSprintField(s string) string { return fileutil.NormalizeSprintField(s) }

// normalizeSprintField is the prior internal alias, kept for caller compatibility.
func normalizeSprintField(s string) string { return fileutil.NormalizeSprintField(s) }

// --------------------------------------------------------------------------
// Check functions
// --------------------------------------------------------------------------

// checkDuplicateIDs detects duplicate Task IDs.
func checkDuplicateIDs(taskFiles []string) []Issue {
	var issues []Issue
	idToFiles := map[string][]string{}
	for _, f := range taskFiles {
		tid := extractTaskIDFromFilename(f)
		if tid != "" {
			idToFiles[tid] = append(idToFiles[tid], f)
		}
	}
	for tid, files := range idToFiles {
		if len(files) >= 2 {
			issues = append(issues, Issue{
				Type:        "duplicate_task_id",
				TaskID:      tid,
				Files:       files,
				AutoFixable: false,
				Message:     fmt.Sprintf("%s %d files exist", tid, len(files)),
			})
		}
	}
	return issues
}

// checkFileLocationVsFrontmatter detects mismatches between file location and frontmatter.
func checkFileLocationVsFrontmatter(taskFiles []string, root string) []Issue {
	var issues []Issue
	for _, f := range taskFiles {
		loc := getDirLocation(f, root)
		fm := readFrontmatterSafe(f)
		if fm == nil {
			continue
		}
		tid := extractTaskIDFromFilename(f)
		fmSprint := fmString(fm, "sprint")
		fmStatus := fmString(fm, "status")

		// works/tasks/ but a sprint field is present -> mismatch (normalisation includes nil/null).
		// Treat the file location as the SSOT and overwrite frontmatter.sprint with "backlog";
		// this auto-fix is offered. Moving the file location is excluded since user intent is ambiguous.
		if loc.inBacklogTasks && normalizeSprintField(fmSprint) != "" {
			issues = append(issues, Issue{
				Type:        "location_frontmatter_mismatch",
				TaskID:      tid,
				File:        f,
				Detail:      "located under works/tasks/ but frontmatter has a sprint field",
				FMSprint:    fmSprint,
				AutoFixable: true,
				Message:     fmt.Sprintf("%s: backlog location but sprint=%s specified", filepath.Base(f), fmSprint),
			})
		}

		// Inconsistency: file is under an active/backlog sprint's tasks/ but status is done/absorbed.
		// Use the actual status value in the message/detail; previously absorbed was incorrectly
		// labelled as done — fixed.
		if loc.inSprintTasks &&
			(fmStatus == "done" || fmStatus == "absorbed") &&
			loc.sprintStatus != "completed" {
			issues = append(issues, Issue{
				Type:        "location_frontmatter_mismatch",
				TaskID:      tid,
				File:        f,
				Detail:      fmt.Sprintf("located in sprint tasks/ but status=%s", fmStatus),
				SprintID:    loc.sprintID,
				FileStatus:  fmStatus,
				AutoFixable: false,
				Message:     fmt.Sprintf("%s: located in sprint=%s tasks/ but status=%s", filepath.Base(f), loc.sprintID, fmStatus),
			})
		}
	}
	return issues
}

// checkDBVsFrontmatter detects mismatches between the DB and the file frontmatter.
func checkDBVsFrontmatter(taskFiles []string, root string) ([]Issue, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	allTasks, err := gs.GetAllTasksSync()
	if err != nil {
		return nil, fmt.Errorf("tasks lookup failure: %w", err)
	}

	dbByID := map[string]dbTaskRow{}
	for _, t := range allTasks {
		r := dbTaskRow{
			taskID:   t.TaskID,
			status:   t.Status,
			filePath: t.FilePath,
		}
		if t.Sprint != "" {
			sp := t.Sprint
			r.sprint = &sp
		}
		dbByID[t.TaskID] = r
	}

	fileByID := map[string]string{}
	for _, f := range taskFiles {
		tid := extractTaskIDFromFilename(f)
		if tid != "" {
			fileByID[tid] = f
		}
	}

	var issues []Issue
	for tid, f := range fileByID {
		dbRow, exists := dbByID[tid]
		fm := readFrontmatterSafe(f)

		if !exists {
			// DB record missing — INSERT required.
			issues = append(issues, Issue{
				Type:        "db_missing_task",
				TaskID:      tid,
				File:        f,
				AutoFixable: fm != nil,
				Message: fmt.Sprintf("%s: file present but missing from DB", tid) +
					func() string {
						if fm != nil {
							return " — INSERT from frontmatter"
						}
						return " — frontmatter parse failed; manual review required"
					}(),
				fm: fm,
			})
			continue
		}

		if fm == nil {
			continue
		}

		fmStatus := fmString(fm, "status")
		fmSprint := NormalizeSprintField(fmString(fm, "sprint"))

		// When the file is at the works/tasks/ (BACKLOG) location, the canonical sprint is "".
		// Even when frontmatter retains sprint=sprint-N, that is handled separately by
		// location_frontmatter_mismatch; to avoid duplicate db_sprint_mismatch detection,
		// fmSprint is treated as "" when compared with the DB.
		fileLoc := getDirLocation(f, root)
		if fileLoc.inBacklogTasks {
			fmSprint = ""
		}

		actualRel := fileutil.ToRepoRelative(f)

		// Normalise the DB-side sprint value the same way.
		// When old task_create runs persisted sprint="backlog" literally to the DB,
		// the file's "backlog" becomes "" via normalizeSprintField but the DB still held "backlog",
		// producing a spurious mismatch. Applying the same rule to both sides blocks this.
		dbSprintStr := ""
		if dbRow.sprint != nil {
			dbSprintStr = NormalizeSprintField(*dbRow.sprint)
		}

		if fmStatus != "" && dbRow.status != fmStatus {
			// Prevents SSOT inversion with the file location as the primary signal.
			// When the file is under completed/<sprint-id>/tasks/ + DB=done + FM != done,
			// treat the frontmatter as stale and update the file to done.
			switch {
			case fileLoc.inSprintTasks &&
				fileLoc.sprintStatus == sprintLocationCompleted &&
				dbRow.status == taskStatusDone:
				// Subject to completed-Sprint file HMAC-signature protection.
				// Editing the frontmatter = changing the file = HMAC drift -> pre-commit BLOCK.
				// Change AutoFixable to false; the user must handle re-signing manually.
				issues = append(issues, Issue{
					Type:        issueFileStatusStale,
					TaskID:      tid,
					DBStatus:    dbRow.status,
					FileStatus:  fmStatus,
					File:        f,
					AutoFixable: false,
					Message: fmt.Sprintf(
						"%s: file is in completed/%s/ with DB=done but frontmatter=%s — protected by HMAC signature (manual re-signing required)",
						tid, fileLoc.sprintID, fmStatus,
					),
				})
			case fileLoc.inSprintTasks &&
				fileLoc.sprintStatus == sprintLocationCompleted &&
				dbRow.status != taskStatusDone:
				// completed folder + DB != done or more state.
				// When the file frontmatter status == done and completion_hmac is valid,
				// the file is the SSOT (a genuinely complete Task) and the DB alone is auto-fixed to done.
				// If HMAC is invalid/missing or fmStatus != done, manual review is retained.
				// Without the HMAC harness, treat a completed-folder task as
				// auto-fixable when the frontmatter already reports "done";
				// otherwise leave it for manual review.
				autoFix := fmStatus == taskStatusDone
				fixReason := "manual-review-required"
				if autoFix {
					fixReason = "frontmatter-done"
				}
				issues = append(issues, Issue{
					Type:        issueCompletedAnomaly,
					TaskID:      tid,
					DBStatus:    dbRow.status,
					FileStatus:  fmStatus,
					File:        f,
					AutoFixable: autoFix,
					Message: fmt.Sprintf(
						"%s: file is in completed/%s/ but DB status=%s, frontmatter=%s (%s)",
						tid, fileLoc.sprintID, dbRow.status, fmStatus, fixReason,
					),
				})
			default:
				// When DB status is ahead of the file (regression), auto_fixable=false.
				if isTaskStatusRegression(dbRow.status, fmStatus) {
					issues = append(issues, Issue{
						Type:        issueDBTaskStatusRegression,
						TaskID:      tid,
						DBStatus:    dbRow.status,
						FileStatus:  fmStatus,
						File:        f,
						AutoFixable: false,
						Message: fmt.Sprintf(
							"%s: DB status=%q > file status=%q — Task state regression blocked. if the file was intentionally reverted, use `hstl task reopen %s --reason \"...\"` use.",
							tid, dbRow.status, fmStatus, tid,
						),
					})
				} else {
					issues = append(issues, Issue{
						Type:        issueDBStatusMismatch,
						TaskID:      tid,
						DBStatus:    dbRow.status,
						FileStatus:  fmStatus,
						File:        f,
						AutoFixable: true,
						Message:     fmt.Sprintf("%s: DB status=%s vs file status=%s", tid, dbRow.status, fmStatus),
					})
				}
			}
		}

		if dbSprintStr != fmSprint {
			issues = append(issues, Issue{
				Type:        "db_sprint_mismatch",
				TaskID:      tid,
				DBSprint:    dbSprintStr,
				FileSprint:  fmSprint,
				File:        f,
				AutoFixable: true,
				Message:     fmt.Sprintf("%s: DB sprint=%s vs file sprint=%s", tid, dbSprintStr, fmSprint),
			})
		}

		if dbRow.filePath != actualRel {
			issues = append(issues, Issue{
				Type:           "db_file_path_mismatch",
				TaskID:         tid,
				DBFilePath:     dbRow.filePath,
				ActualFilePath: actualRel,
				AutoFixable:    true,
				Message:        fmt.Sprintf("%s: DB file_path=%s vs actual=%s", tid, dbRow.filePath, actualRel),
			})
		}
	}
	return issues, nil
}

// checkDBOrphanTasks detects fully orphan rows that exist in the DB but are absent
// from both the file system and BACKLOG.md.
// Background: the prior check ran in the file -> DB direction only and missed
// "ghost Tasks left only in the DB". This caused ghost Tasks to surface in
// hstl task list while DB consistency was silently broken.
// Detection criterion (3-axis cross):
// 1. Row exists in the DB tasks table.
// 2. No actual Task file (T*.md) exists anywhere under works/.
// 3. The TaskID is not listed in BACKLOG.md.
// All three conditions true -> orphan. auto_fixable=true — applyFixes deletes
// the DB row. Destructive, so dry-run only proposes; the actual deletion runs
// when the user invokes `hstl backlog sync` (non dry-run).
func checkDBOrphanTasks(taskFiles []string, root string) ([]Issue, error) {
	gs := store.Get()
	if gs == nil {
		return nil, nil
	}
	allTasks, err := gs.GetAllTasksSync()
	if err != nil {
		return nil, fmt.Errorf("tasks lookup failure: %w", err)
	}

	fileIDs := map[string]bool{}
	for _, f := range taskFiles {
		tid := extractTaskIDFromFilename(f)
		if tid != "" {
			fileIDs[tid] = true
		}
	}
	backlogIDs := parseBacklogMDTaskIDs(root)

	// Enumerate every worktree root — in multi-worktree environments,
	// new tasks created in another worktree must not be wrongly deleted as
	// db_orphan when this worktree runs sync.
	worktreeRoots := EnumerateWorktreeRoots()

	var issues []Issue
	for _, t := range allTasks {
		if fileIDs[t.TaskID] {
			continue
		}
		if backlogIDs[t.TaskID] {
			continue
		}
		// If the file exists in another worktree, it is not an orphan (cross-worktree awareness).
		if FileExistsInAnyWorktree(t.FilePath, worktreeRoots, root) {
			continue
		}
		issues = append(issues, Issue{
			Type:        issueDBOrphanTask,
			TaskID:      t.TaskID,
			DBFilePath:  t.FilePath,
			DBStatus:    t.Status,
			AutoFixable: true,
			Detail: fmt.Sprintf(
				"DB file_path=%q (absent in every worktree), not listed in BACKLOG.md",
				t.FilePath,
			),
			Message: fmt.Sprintf(
				"%s: DB orphan — absent from every worktree and BACKLOG.md (status=%s, file_path=%s) — proposes DB row deletion",
				t.TaskID, t.Status, t.FilePath,
			),
		})
	}
	return issues, nil
}

// dbStatusIsDone looks up the Task status in the DB. When the DB is not initialised
// or lookup fails, returns false — conservatively treats it as "not done" so
// auto-fix is not applied.
func dbStatusIsDone(taskID string) bool {
	gs := store.Get()
	if gs == nil {
		return false
	}
	status, err := gs.GetTaskStatus(taskID)
	if err != nil {
		return false
	}
	return status == taskStatusDone
}

// checkBacklogMDVsFiles detects mismatches between BACKLOG.md and the actual files under works/tasks/.
// Tasks moved to `works/tasks/completed/` whose entries linger in
// BACKLOG.md are classified as `backlog_md_stale_completed` with AutoFixable=true,
// so `hstl backlog sync` can remove them automatically. Resolves the drift that
// previously accumulated when the auto-move side effect and the sync-scan range disagreed.
// (ISS-20260422-013).
// When sprint-assigned Tasks move to
// `works/sprints/completed/sprint-NN/tasks/T*.md` after sprint:complete, the same
// 3-condition AND gate (sprint-completed location + DB=done + BACKLOG residual)
// removes them automatically. The shared reason is `backlog_md_stale_completed`,
// but the audit event reason field distinguishes `sprint_completed_folder` vs
// `tasks_completed_folder` for traceability.
func checkBacklogMDVsFiles(taskFiles []string, root string) []Issue {
	var issues []Issue
	backlogIDs := parseBacklogMDTaskIDs(root)

	actualBacklogFiles := map[string]string{}
	tasksCompletedFiles := map[string]string{}
	sprintCompletedFiles := map[string]string{}
	for _, f := range taskFiles {
		loc := getDirLocation(f, root)
		tid := extractTaskIDFromFilename(f)
		if tid == "" {
			continue
		}
		if loc.inBacklogTasks {
			actualBacklogFiles[tid] = f
		}
		if loc.inTasksCompleted {
			tasksCompletedFiles[tid] = f
		}
		// Identify works/sprints/completed/sprint-*/tasks/T*.md.
		if loc.inSprintTasks && loc.sprintStatus == "completed" {
			sprintCompletedFiles[tid] = f
		}
	}

	// listed in BACKLOG.md but the file is missing
	for tid := range backlogIDs {
		if _, exists := actualBacklogFiles[tid]; exists {
			continue
		}

		// First decide whether it is a done Task moved to completed/.
		// When the file exists under completed/ and DB status=done, sync
		// auto-removes the BACKLOG.md row (resolves stale entry). When any
		// condition fails, preserve the existing missing_file + manual entry.
		if completedFile, inCompleted := tasksCompletedFiles[tid]; inCompleted {
			if dbStatusIsDone(tid) {
				issues = append(issues, Issue{
					Type:        "backlog_md_stale_completed",
					TaskID:      tid,
					File:        completedFile,
					AutoFixable: true,
					Detail:      "done Task moved to works/tasks/completed/ but lingering in BACKLOG.md",
					Message:     fmt.Sprintf("%s: exists under completed/ + DB=done — eligible for automatic BACKLOG.md row removal", tid),
				})
				continue
			}
		}

		// When sprint-assigned Tasks move to sprint-completed/
		// after sprint:complete, the same 3-condition AND gate applies.
		if sprintFile, inSprintCompleted := sprintCompletedFiles[tid]; inSprintCompleted {
			if dbStatusIsDone(tid) {
				issues = append(issues, Issue{
					Type:        "backlog_md_stale_completed",
					TaskID:      tid,
					File:        sprintFile,
					AutoFixable: true,
					Detail:      "done Task moved to works/sprints/completed/sprint-*/tasks/ but lingering in BACKLOG.md",
					Message:     fmt.Sprintf("%s: sprint-completed location + DB=done — eligible for automatic BACKLOG.md row removal", tid),
				})
				continue
			}
		}

		issues = append(issues, Issue{
			Type:        "backlog_md_missing_file",
			TaskID:      tid,
			Detail:      "listed in BACKLOG.md but file missing under works/tasks/",
			AutoFixable: false,
			Message:     fmt.Sprintf("%s: listed in BACKLOG.md but the file is missing", tid),
		})
	}

	// file present but missing from BACKLOG.md (only when unassigned + todo)
	for tid, f := range actualBacklogFiles {
		if backlogIDs[tid] {
			continue
		}
		fm := readFrontmatterSafe(f)
		if fm == nil {
			continue
		}
		fmSprint := fmString(fm, "sprint")
		fmStatus := fmString(fm, "status")
		if fmSprint == "" && fmStatus == "todo" {
			issues = append(issues, Issue{
				Type:        "backlog_md_missing_entry",
				TaskID:      tid,
				File:        f,
				Detail:      "file present but missing from BACKLOG.md (unassigned + todo)",
				AutoFixable: true,
				Message:     fmt.Sprintf("%s: located under works/tasks/ but missing from BACKLOG.md", tid),
				fm:          fm,
			})
		}
	}
	return issues
}

// checkFrontmatterParseErrors detects frontmatter parse failures.
func checkFrontmatterParseErrors(taskFiles []string) []Issue {
	var issues []Issue
	for _, f := range taskFiles {
		_, err := fileutil.ReadTaskFrontmatter(f)
		if err != nil {
			issues = append(issues, Issue{
				Type:        "frontmatter_parse_error",
				TaskID:      extractTaskIDFromFilename(f),
				File:        f,
				Error:       err.Error(),
				AutoFixable: false,
				Message:     fmt.Sprintf("%s: frontmatter parse failed — %v", filepath.Base(f), err),
			})
		}
	}
	return issues
}

var reLastUpdatedLine = regexp.MustCompile(`>\s*Last updated:\s*(\S+)`)
var reTodoHeader = regexp.MustCompile(`##\s+Remaining unassigned Tasks\s*—?\s*(\d+) items?`)
var reTodoRow = regexp.MustCompile(`^\|\s*\*?\*?T\d+`)
var reSprintRow = regexp.MustCompile(`^\|\s*\*?\*?(sprint-\d+)`)

// checkBacklogSummaryMetadata verifies the consistency of the BACKLOG.md summary metadata.
func checkBacklogSummaryMetadata(root string) ([]Issue, error) {
	var issues []Issue
	backlogPath := filepath.Join(root, "works", "tasks", "BACKLOG.md")
	data, err := os.ReadFile(backlogPath)
	if os.IsNotExist(err) {
		return issues, nil
	}
	if err != nil {
		return nil, fmt.Errorf("BACKLOG.md read failure: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	today := time.Now().Format("2006-01-02")

	// 7-1. Check the last-updated date.
	for _, line := range lines {
		m := reLastUpdatedLine.FindStringSubmatch(line)
		if m != nil {
			if m[1] != today {
				issues = append(issues, Issue{
					Type:        "backlog_summary_stale_timestamp",
					Detail:      fmt.Sprintf("last-updated date %s != today %s", m[1], today),
					AutoFixable: true,
					Message:     fmt.Sprintf("BACKLOG.md last-updated date is stale: %s", m[1]),
				})
			}
			break
		}
	}

	// 7-2. Check the unassigned-Task count.
	actualCount := 0
	var headerCount *int
	inTodoTable := false
	for _, line := range lines {
		mh := reTodoHeader.FindStringSubmatch(line)
		if mh != nil {
			n := 0
			_, _ = fmt.Sscanf(mh[1], "%d", &n)
			headerCount = &n
			inTodoTable = true
			continue
		}
		if inTodoTable && strings.HasPrefix(line, "## ") {
			break
		}
		if inTodoTable && reTodoRow.MatchString(line) {
			actualCount++
		}
	}
	if headerCount != nil && *headerCount != actualCount {
		issues = append(issues, Issue{
			Type:        "backlog_summary_count_mismatch",
			Detail:      fmt.Sprintf("header %d items vs actual rows %d items", *headerCount, actualCount),
			AutoFixable: true,
			Message:     fmt.Sprintf("BACKLOG.md unassigned-count mismatch: header=%d actual=%d", *headerCount, actualCount),
		})
	}

	// 7-3. Check the Sprint-assignment table.
	gs := store.Get()
	if gs == nil {
		return issues, nil
	}

	allSprints, err := gs.GetAllSprintsForSync()
	if err != nil {
		return issues, nil // Ignore DB errors (Sprint may be absent).
	}

	var dbSprints []dbSprintRow
	for _, s := range allSprints {
		dbSprints = append(dbSprints, dbSprintRow{
			sprintID: s.SprintID,
			title:    s.Title,
			status:   s.Status,
		})
	}

	// When the "## Sprint Assignment Status" section is absent from
	// BACKLOG.md, rebuildSprintSummaryTable becomes a silent no-op so sync
	// never converges. When the project deliberately omits the section, skip the
	// stale check itself. To opt in, the user manually adds a
	// `## Sprint Assignment Status` header once and a subsequent rebuild fills it in.
	hasSprintSection := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## Sprint Assignment Status") {
			hasSprintSection = true
			break
		}
	}

	if hasSprintSection && len(dbSprints) > 0 {
		sprintIDsInMD := map[string]bool{}
		for _, line := range lines {
			m := reSprintRow.FindStringSubmatch(line)
			if m != nil {
				sprintIDsInMD[m[1]] = true
			}
		}

		dbSprintIDs := map[string]bool{}
		for _, s := range dbSprints {
			dbSprintIDs[s.sprintID] = true
		}

		var missingActive []string
		for sid := range dbSprintIDs {
			if !sprintIDsInMD[sid] {
				// Inspect only active/backlog Sprints.
				for _, s := range dbSprints {
					if s.sprintID == sid && (s.status == "active" || s.status == "backlog") {
						missingActive = append(missingActive, sid)
						break
					}
				}
			}
		}

		if len(missingActive) > 0 {
			issues = append(issues, Issue{
				Type:        "backlog_summary_sprint_table_stale",
				Detail:      fmt.Sprintf("missing from the assignment-status table: %v", missingActive),
				AutoFixable: true,
				Message:     fmt.Sprintf("%d items missing from the BACKLOG.md Sprint Assignment Status table", len(missingActive)),
				dbSprints:   dbSprints,
			})
		}
	}

	return issues, nil
}

// --------------------------------------------------------------------------
// Apply function
// --------------------------------------------------------------------------

// applyFixes repairs auto_fixable=true issues by updating the DB and BACKLOG.md.
func applyFixes(issues []Issue) ([]Fix, error) {
	var fixes []Fix
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	summaryTypes := map[string]bool{
		"backlog_summary_stale_timestamp": true,
		"backlog_summary_count_mismatch":  true,
	}
	needSummaryRebuild := false

	for _, iss := range issues {
		if !iss.AutoFixable {
			continue
		}

		switch iss.Type {
		case issueDBMissingSprint:
			if fix, ok := applySprintInsert(iss); ok {
				fixes = append(fixes, fix)
			}

		case issueSprintMetadataStale:
			if fix, ok := applySprintMetadataFix(iss); ok {
				fixes = append(fixes, fix)
			}

		case issueDBSprintTitleMismatch, issueDBSprintFolderPathMismatch, issueDBSprintStatusFileAbsent:
			if fix, ok := applySprintDriftFix(iss); ok {
				fixes = append(fixes, fix)
			}

		case "db_missing_task":
			fm := iss.fm
			if fm == nil {
				continue
			}
			relPath := fileutil.ToRepoRelative(iss.File)
			now := time.Now().Format("2006-01-02")

			createdAt := fmString(fm, "created")
			if createdAt == "" {
				createdAt = now
			}

			title := fmString(fm, "title")
			taskType := fmString(fm, "type")
			if taskType == "" {
				taskType = "chore"
			}
			fmSprint := normalizeSprintField(fmString(fm, "sprint"))
			status := fmString(fm, "status")
			if status == "" {
				status = "todo"
			}
			priority := fmString(fm, "priority")
			if priority == "" {
				priority = "p2"
			}
			estimate := fmString(fm, "estimate")
			if estimate == "" {
				estimate = "S"
			}

			// Counter repair.
			var taskNum int
			if n, err2 := fmt.Sscanf(iss.TaskID, "T%d", &taskNum); err2 != nil || n != 1 {
				taskNum = 0
			}

			// When frontmatter.work_ticket exists, attach it to TaskRecord and
			// record it in the DB at INSERT time (minimises drift).
			workTicket := fmString(fm, "work_ticket")

			taskRec := &ports.TaskRecord{
				TaskID:     iss.TaskID,
				Title:      title,
				Type:       taskType,
				Sprint:     fmSprint,
				Status:     status,
				Priority:   priority,
				Estimate:   estimate,
				FilePath:   relPath,
				CreatedAt:  createdAt,
				WorkTicket: workTicket,
			}

			if err := gs.UpsertTaskInsert(taskRec, taskNum); err != nil {
				continue
			}

			fixes = append(fixes, Fix{Type: "db_task_inserted", TaskID: iss.TaskID})
			_ = audit.LogEvent("backlog_sync.db_task_inserted", "task", iss.TaskID, "claude",
				map[string]any{"file_path": relPath, "status": status}, "")

		case issueDBTaskStatusRegression:
			// Regression issues are auto_fixable=false and should never reach applyFixes.
			// Defensive handling: only record an audit event and skip.
			_ = audit.LogEvent("backlog_sync.task_status_regression_blocked", "task", iss.TaskID, "claude",
				map[string]any{"db_status": iss.DBStatus, "file_status": iss.FileStatus}, "")

		case issueDBStatusMismatch:
			if iss.FileStatus == "" {
				continue
			}
			if err := gs.UpdateTaskDirect(iss.TaskID, ports.TaskUpdateFields{Status: &iss.FileStatus}); err != nil {
				continue
			}
			fixes = append(fixes, Fix{Type: "db_status_fixed", TaskID: iss.TaskID, NewStatus: iss.FileStatus})
			_ = audit.LogEvent("backlog_sync.db_status_fixed", "task", iss.TaskID, "claude",
				map[string]any{"old": iss.DBStatus, "new": iss.FileStatus}, "")

		case issueCompletedAnomaly:
			// Repair DB status to done for completed Tasks with a valid HMAC.
			// Only entries that already passed HMAC verification in the detect step
			// are forwarded with AutoFixable=true, so updating only the DB to done here is safe.
			newStatus := taskStatusDone
			if err := gs.UpdateTaskDirect(iss.TaskID, ports.TaskUpdateFields{Status: &newStatus}); err != nil {
				continue
			}
			fixes = append(fixes, Fix{Type: "completed_anomaly_fixed", TaskID: iss.TaskID, NewStatus: newStatus})
			_ = audit.LogEvent("backlog_sync.completed_anomaly_fixed", "task", iss.TaskID, "claude",
				map[string]any{
					"old_db_status": iss.DBStatus,
					"new_db_status": newStatus,
					"file":          iss.File,
					"reason":        "hmac_valid_done_task",
				}, "")

		case issueFileStatusStale:
			// DB=done is the correct value; overwrite the file frontmatter to done.
			// Recovers stale frontmatter left in completed/ by historical bugs
			// .
			if iss.File == "" {
				continue
			}
			if err := fileutil.UpdateTaskStatus(iss.File, taskStatusDone); err != nil {
				continue
			}
			fixes = append(fixes, Fix{
				Type:      fixFileStatusFixed,
				TaskID:    iss.TaskID,
				NewStatus: taskStatusDone,
			})
			_ = audit.LogEvent("backlog_sync.file_status_fixed", "task", iss.TaskID, "claude",
				map[string]any{
					"old_frontmatter": iss.FileStatus,
					"new_frontmatter": taskStatusDone,
					"db_status":       iss.DBStatus,
					"file":            iss.File,
					"reason":          "file_stale_after_sprint_complete",
				}, "")

		case "db_sprint_mismatch":
			sprintPtr := &iss.FileSprint
			if err := gs.UpdateTaskDirect(iss.TaskID, ports.TaskUpdateFields{Sprint: sprintPtr}); err != nil {
				continue
			}
			fixes = append(fixes, Fix{Type: "db_sprint_fixed", TaskID: iss.TaskID, NewSprint: iss.FileSprint})
			_ = audit.LogEvent("backlog_sync.db_sprint_fixed", "task", iss.TaskID, "claude",
				map[string]any{"old": iss.DBSprint, "new": iss.FileSprint}, "")

		case "location_frontmatter_mismatch":
			// Only the case where the file is at works/tasks/ but frontmatter.sprint is non-empty is auto-fixed.
			// Treat the file location as the SSOT and overwrite frontmatter.sprint with "backlog".
			// Other location_frontmatter_mismatch sub-cases (under sprint tasks/ but status=done)
			// are collected with AutoFixable=false and never enter this loop.
			if iss.File == "" {
				continue
			}
			if err := fileutil.UpdateTaskFrontmatter(iss.File, map[string]any{"sprint": "backlog"}); err != nil {
				continue
			}
			// Synchronise the DB as well — update lingering legacy data to NULL.
			emptyStr := ""
			_ = gs.UpdateTaskDirect(iss.TaskID, ports.TaskUpdateFields{Sprint: &emptyStr})
			fixes = append(fixes, Fix{
				Type:      "location_frontmatter_sprint_cleared",
				TaskID:    iss.TaskID,
				OldSprint: iss.FMSprint,
				NewSprint: "backlog",
			})
			_ = audit.LogEvent("backlog_sync.location_frontmatter_sprint_cleared", "task", iss.TaskID, "claude",
				map[string]any{"old_fm_sprint": iss.FMSprint, "file": iss.File}, "")

		case "db_file_path_mismatch":
			if err := gs.UpdateTaskDirect(iss.TaskID, ports.TaskUpdateFields{FilePath: &iss.ActualFilePath}); err != nil {
				continue
			}
			fixes = append(fixes, Fix{Type: "db_file_path_fixed", TaskID: iss.TaskID, NewFilePath: iss.ActualFilePath})
			_ = audit.LogEvent("backlog_sync.db_file_path_fixed", "task", iss.TaskID, "claude",
				map[string]any{"old": iss.DBFilePath, "new": iss.ActualFilePath}, "")

		case issueDBOrphanTask:
			// Delete the orphan Task that exists only in the DB.
			// Destructive operation, so dry-run only detects (this branch only runs in non-dry-run).
			if err := gs.DeleteTask(iss.TaskID); err != nil {
				continue
			}
			fixes = append(fixes, Fix{Type: fixDBOrphanTaskDeleted, TaskID: iss.TaskID})
			_ = audit.LogEvent("backlog_sync.db_orphan_task_deleted", "task", iss.TaskID, "claude",
				map[string]any{
					"file_path": iss.DBFilePath,
					"status":    iss.DBStatus,
					"reason":    "file_absent_and_backlog_md_absent",
				}, "")

		case "backlog_md_stale_completed":
			// For done Tasks moved into completed/, remove the lingering
			// BACKLOG.md row. On failure continue so other fixes are not blocked.
			// The sprint-completed path is handled in the same branch.
			// The audit reason field distinguishes the two sources.
			removed, rmErr := fileutil.RemoveFromBacklogMD(iss.TaskID)
			if rmErr != nil || !removed {
				continue
			}
			fixes = append(fixes, Fix{Type: "backlog_md_row_removed", TaskID: iss.TaskID})
			reason := "tasks_completed_folder_and_db_done"
			if strings.Contains(filepath.ToSlash(iss.File), "works/sprints/completed/") {
				reason = "sprint_completed_folder_and_db_done"
			}
			_ = audit.LogEvent("backlog_sync.backlog_md_row_removed", "task", iss.TaskID, "claude",
				map[string]any{
					"reason": reason,
					"file":   iss.File,
				}, "")

		case "backlog_md_missing_entry":
			fm := iss.fm
			if fm == nil {
				continue
			}
			err := fileutil.UpdateBacklogMD(
				iss.TaskID,
				fmString(fm, "title"),
				fmString(fm, "type"),
				fmString(fm, "estimate"),
				fmString(fm, "priority"),
			)
			if err != nil {
				continue
			}
			fixes = append(fixes, Fix{Type: "backlog_md_entry_added", TaskID: iss.TaskID})
			_ = audit.LogEvent("backlog_sync.backlog_md_entry_added", "task", iss.TaskID, "claude",
				map[string]any{"title": fmString(fm, "title")}, "")

		case "backlog_summary_sprint_table_stale":
			if len(iss.dbSprints) > 0 {
				if err := rebuildSprintSummaryTable(iss.dbSprints); err == nil {
					fixes = append(fixes, Fix{Type: "backlog_sprint_table_rebuilt"})
				}
			}
		}

		if summaryTypes[iss.Type] {
			needSummaryRebuild = true
		}
	}

	// Bulk update the summary metadata (date + counts).
	if needSummaryRebuild {
		result, err := fileutil.RebuildBacklogSummary()
		if err == nil && result.Updated {
			fixes = append(fixes, Fix{
				Type:      "backlog_summary_updated",
				Timestamp: result.Timestamp,
				TodoCount: result.TodoCount,
			})
		}
	}

	return fixes, nil
}

// rebuildSprintSummaryTable regenerates the Sprint Assignment Status table in BACKLOG.md from the DB.
func rebuildSprintSummaryTable(dbSprints []dbSprintRow) error {
	root := fileutil.GetProjectRoot()
	backlogPath := filepath.Join(root, "works", "tasks", "BACKLOG.md")
	data, err := os.ReadFile(backlogPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Aggregate Task counts per Sprint.
	taskCounts := map[string]int{}
	if gs := store.Get(); gs != nil {
		if counts, err := gs.GetSprintTaskCounts(); err == nil {
			taskCounts = counts
		}
	}

	// Build the table.
	tableLines := []string{
		"| Sprint | title | state | Tasks |",
		"|--------|------|------|---------|",
	}

	var completed, active, backlog []dbSprintRow
	for _, s := range dbSprints {
		switch s.status {
		case "completed":
			completed = append(completed, s)
		case "active":
			active = append(active, s)
		case "backlog":
			backlog = append(backlog, s)
		}
	}

	statusLabel := map[string]string{
		"completed": "completed",
		"active":    "active",
		"backlog":   "backlog",
	}

	for _, s := range append(active, backlog...) {
		cnt := taskCounts[s.sprintID]
		st := statusLabel[s.status]
		tableLines = append(tableLines,
			fmt.Sprintf("| **%s** | **%s** | **%s** | **%d items** |", s.sprintID, s.title, st, cnt),
		)
	}

	// Recent 5 completed items (reverse order).
	completedSorted := make([]dbSprintRow, len(completed))
	copy(completedSorted, completed)
	// Reverse sort by sprint_id.
	for i, j := 0, len(completedSorted)-1; i < j; i, j = i+1, j-1 {
		completedSorted[i], completedSorted[j] = completedSorted[j], completedSorted[i]
	}
	displayLimit := completedSprintsDisplayLimit
	if len(completedSorted) < displayLimit {
		displayLimit = len(completedSorted)
	}
	for _, s := range completedSorted[:displayLimit] {
		cnt := taskCounts[s.sprintID]
		tableLines = append(tableLines,
			fmt.Sprintf("| %s | %s | completed | %d items |", s.sprintID, s.title, cnt),
		)
	}
	if len(completed) > completedSprintsDisplayLimit {
		tableLines = append(tableLines,
			fmt.Sprintf("| ... | %d earlier items omitted | completed | — |", len(completed)-completedSprintsDisplayLimit),
		)
	}

	newTable := strings.Join(tableLines, "\n")

	// Replace the existing table.
	lines := strings.Split(string(data), "\n")
	reSprintSection := regexp.MustCompile(`##\s+Sprint Assignment Status`)
	startIdx := -1
	endIdx := -1
	for i, line := range lines {
		if reSprintSection.MatchString(line) {
			startIdx = i + 1
		} else if startIdx >= 0 && strings.HasPrefix(line, "## ") && i > startIdx {
			endIdx = i
			break
		}
	}

	if startIdx >= 0 {
		if endIdx < 0 {
			endIdx = len(lines)
		}
		newLines := make([]string, 0, len(lines))
		newLines = append(newLines, lines[:startIdx]...)
		newLines = append(newLines, "", newTable, "")
		newLines = append(newLines, lines[endIdx:]...)
		return os.WriteFile(backlogPath, []byte(strings.Join(newLines, "\n")), 0o644)
	}
	return nil
}

// --------------------------------------------------------------------------
// Sprint scan/restore — (ISS-20260413-001)
// --------------------------------------------------------------------------

// sprintStatusActive / sprintStatusBacklog / sprintStatusCompleted — sprints.status value.
const (
	sprintStatusActive       = "active"
	sprintStatusBacklog      = "backlog"
	sprintStatusCompleted    = "completed"
	issueDBMissingSprint     = "db_missing_sprint"
	issueSprintMetadataStale = "db_sprint_metadata_stale" // title=sprint_id or date missing
	fixSprintInserted        = "db_sprint_inserted"
	fixSprintMetadataFixed   = "db_sprint_metadata_fixed" // 

	// Three drift cases caused by parallel-worktree edits between DB and filesystem.
	issueDBSprintTitleMismatch      = "db_sprint_title_mismatch"
	issueDBSprintFolderPathMismatch = "db_sprint_folder_path_mismatch"
	issueDBSprintStatusFileAbsent   = "db_sprint_status_file_absent"
	// (, ISS-20260421-003) — When the DB Sprint status is ahead of the file location
	// (regression). AutoFixable=false is fixed and the user is steered to manual
	// verification. Progression order: backlog < active < completed.
	issueDBSprintStatusRegression = "db_sprint_status_regression"
	fixSprintTitleFixed           = "db_sprint_title_fixed"
	fixSprintFolderPathFixed      = "db_sprint_folder_path_fixed"
	fixSprintStatusFixed          = "db_sprint_status_fixed"
)

// sprintStatusRank — Sprint status progression order. Used for regression decisions.
// backlog (0) -> active (1) -> completed (2). Unknown values return -1.
func sprintStatusRank(s string) int {
	switch s {
	case "backlog":
		return 0
	case "active":
		return 1
	case sprintStatusCompleted: // "completed"
		return 2
	}
	return -1
}

// taskStatusRank — Task status progression order. Used for regression decisions.
// todo (0) -> in-progress (1) -> done (2). Unknown values return -1.
func taskStatusRank(s string) int {
	switch s {
	case "todo":
		return 0
	case "in-progress", "in_progress":
		return 1
	case taskStatusDone:
		return 2
	}
	return -1
}

// isTaskStatusRegression — is the Task DB status ahead of the file status
// (= would using the file as the SSOT regress DB destructively)?
// Example: DB=done + file=todo -> true (would revert a completed Task to incomplete).
// Example: DB=todo + file=done -> false (file is ahead; this is the normal sync direction).
// On issueDBStatusMismatch detection, check whether it is a regression. When it is,
// classify as auto_fixable=false (issueDBTaskStatusRegression).
func isTaskStatusRegression(dbStatus, fileStatus string) bool {
	dRank := taskStatusRank(dbStatus)
	fRank := taskStatusRank(fileStatus)
	if dRank < 0 || fRank < 0 {
		return false
	}
	return dRank > fRank
}

// isSprintStatusRegression — is the DB status ahead of the file status
// (= would using the file as the SSOT regress DB destructively)?
// Example: DB=completed + file=backlog -> true (would revert a Sprint complete).
// Example: DB=backlog + file=active -> false (file is ahead; normal sync direction).
// ISS-20260421-003 Key defence: regression must never be auto-fixed. The user must explicitly
// reopen the Sprint or intentionally revert it through a manual process.
func isSprintStatusRegression(dbStatus, fileStatus string) bool {
	dRank := sprintStatusRank(dbStatus)
	fRank := sprintStatusRank(fileStatus)
	if dRank < 0 || fRank < 0 {
		return false
	}
	return dRank > fRank
}

// sprintFileInfo is the scan result for works/sprints/*/sprint-*/SPRINT.md.
type sprintFileInfo struct {
	sprintID   string
	status     string // active/backlog/completed — decided by folder location
	folderPath string // relative path to the repo root (works/sprints/.../sprint-NN)
	filePath   string // SPRINT.md absolute path
	fm         map[string]any
}

// globSprintFiles collects works/sprints/{active,backlog,completed}/sprint-*/SPRINT.md.
func globSprintFiles(root string) ([]sprintFileInfo, error) {
	var results []sprintFileInfo
	base := filepath.Join(root, "works", "sprints")

	for _, status := range []string{sprintStatusActive, sprintStatusBacklog, sprintStatusCompleted} {
		statusDir := filepath.Join(base, status)
		entries, err := os.ReadDir(statusDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() || !strings.HasPrefix(e.Name(), "sprint-") {
				continue
			}
			sprintDir := filepath.Join(statusDir, e.Name())
			sprintMD := filepath.Join(sprintDir, "SPRINT.md")
			if _, err := os.Stat(sprintMD); err != nil {
				continue
			}
			fm := readFrontmatterSafe(sprintMD)
			rel, _ := filepath.Rel(root, sprintDir)
			results = append(results, sprintFileInfo{
				sprintID:   e.Name(),
				status:     status,
				folderPath: filepath.ToSlash(rel),
				filePath:   sprintMD,
				fm:         fm,
			})
		}
	}
	return results, nil
}

// checkDBVsSprintFiles flags Sprints whose files exist but are missing from
// the DB or whose metadata is stale.
func checkDBVsSprintFiles(root string) ([]Issue, error) {
	sprintFiles, err := globSprintFiles(root)
	if err != nil {
		return nil, fmt.Errorf("sprint file collection failed: %w", err)
	}
	if len(sprintFiles) == 0 {
		return nil, nil
	}

	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	// Load the current metadata for each sprint from the DB.
	// (title/started_at/completed_at + : status/folder_path)
	type dbRow struct {
		title       string
		startedAt   string
		completedAt string
		status      string
		folderPath  string
	}
	allSprints, err := gs.GetAllSprintsForSync()
	if err != nil {
		return nil, fmt.Errorf("sprints lookup failure: %w", err)
	}

	dbSprints := map[string]dbRow{}
	for _, s := range allSprints {
		dbSprints[s.SprintID] = dbRow{
			title:       s.Title,
			startedAt:   s.StartedAt,
			completedAt: s.CompletedAt,
			status:      s.Status,
			folderPath:  s.FolderPath,
		}
	}

	var issues []Issue
	for _, sf := range sprintFiles {
		sf := sf // capture for pointer
		row, exists := dbSprints[sf.sprintID]
		if !exists {
			// DB missing
			autoFixable := sf.fm != nil
			msg := fmt.Sprintf("%s: file present but missing from DB", sf.sprintID)
			if autoFixable {
				msg += " — INSERT from frontmatter"
			} else {
				msg += " — frontmatter missing; attempting fallback repair"
				autoFixable = true // still try the fallback path even when fm is missing
			}
			issues = append(issues, Issue{
				Type:        issueDBMissingSprint,
				SprintID:    sf.sprintID,
				File:        sf.filePath,
				AutoFixable: autoFixable,
				Message:     msg,
				sprintInfo:  &sf,
			})
			continue
		}

		// Detect DB vs file drift (parallel-worktree edit recovery).
		fileTitle := ""
		if sf.fm != nil {
			fileTitle = strings.TrimSpace(fmString(sf.fm, "title"))
		}
		// 1) title mismatch (frontmatter has title that differs from DB)
		if fileTitle != "" && row.title != "" && row.title != sf.sprintID && row.title != fileTitle {
			issues = append(issues, Issue{
				Type:        issueDBSprintTitleMismatch,
				SprintID:    sf.sprintID,
				File:        sf.filePath,
				AutoFixable: true,
				Message: fmt.Sprintf("%s: DB title=%q vs file title=%q — repair DB using the file as the SSOT",
					sf.sprintID, row.title, fileTitle),
				sprintInfo: &sf,
			})
		}
		// 2) folder_path mismatch
		if row.folderPath != "" && row.folderPath != sf.folderPath {
			issues = append(issues, Issue{
				Type:        issueDBSprintFolderPathMismatch,
				SprintID:    sf.sprintID,
				File:        sf.filePath,
				AutoFixable: true,
				Message: fmt.Sprintf("%s: DB folder_path=%q vs actual=%q — repair DB using the file location",
					sf.sprintID, row.folderPath, sf.folderPath),
				sprintInfo: &sf,
			})
		}
		// 3) status vs folder mismatch (file in backlog/ but DB=active, etc.)
		if row.status != "" && row.status != sf.status {
			// (, ISS-20260421-003): Regression detection. When the DB is ahead of the file
			// (e.g. DB=completed / file=backlog), auto-repair is forbidden.
			if isSprintStatusRegression(row.status, sf.status) {
				issues = append(issues, Issue{
					Type:        issueDBSprintStatusRegression,
					SprintID:    sf.sprintID,
					File:        sf.filePath,
					AutoFixable: false,
					Message: fmt.Sprintf("%s: DB status=%q > file status=%q — state regression blocked (potentially destructive). Manual review required: to actually reopen the Sprint use `hostler sprint reopen %s`; if the file is stale, clean it up manually.",
						sf.sprintID, row.status, sf.status, sf.sprintID),
					sprintInfo: &sf,
				})
			} else {
				issues = append(issues, Issue{
					Type:        issueDBSprintStatusFileAbsent,
					SprintID:    sf.sprintID,
					File:        sf.filePath,
					AutoFixable: true,
					Message: fmt.Sprintf("%s: DB status=%q vs file location=%q — repair DB using the file location",
						sf.sprintID, row.status, sf.status),
					sprintInfo: &sf,
				})
			}
		}

		// Already in DB but metadata is stale — either title=sprint_id
		// or status is completed while completed_at is the empty string.
		titleStale := row.title == "" || row.title == sf.sprintID
		completedMissing := sf.status == sprintStatusCompleted && row.completedAt == ""
		startedMissing := row.startedAt == ""
		if !titleStale && !completedMissing && !startedMissing {
			continue
		}
		reasons := []string{}
		if titleStale {
			reasons = append(reasons, "title=sprint_id")
		}
		if startedMissing {
			reasons = append(reasons, "started_at missing")
		}
		if completedMissing {
			reasons = append(reasons, "completed_at missing")
		}
		issues = append(issues, Issue{
			Type:        issueSprintMetadataStale,
			SprintID:    sf.sprintID,
			File:        sf.filePath,
			AutoFixable: true,
			Message: fmt.Sprintf("%s: metadata stale — %s (repaired from file frontmatter / mtime / git log)",
				sf.sprintID, strings.Join(reasons, ", ")),
			sprintInfo: &sf,
		})
	}
	return issues, nil
}

// applySprintDriftFix repairs the three drift issues by using the file as the SSOT.
// Treats the file as the SSOT and overwrites DB title / folder_path / status with the file values.
func applySprintDriftFix(iss Issue) (Fix, bool) {
	if iss.sprintInfo == nil {
		return Fix{}, false
	}
	gs := store.Get()
	if gs == nil {
		return Fix{}, false
	}
	info := iss.sprintInfo
	fileTitle := ""
	if info.fm != nil {
		fileTitle = strings.TrimSpace(fmString(info.fm, "title"))
	}
	var fixType string
	switch iss.Type {
	case issueDBSprintTitleMismatch:
		if fileTitle == "" {
			return Fix{}, false
		}
		if err := gs.UpdateSprintMetadata(iss.SprintID, fileTitle, "", ""); err != nil {
			return Fix{}, false
		}
		fixType = fixSprintTitleFixed
	case issueDBSprintFolderPathMismatch:
		if err := gs.UpdateSprintMetadata(iss.SprintID, "", info.folderPath, ""); err != nil {
			return Fix{}, false
		}
		fixType = fixSprintFolderPathFixed
	case issueDBSprintStatusFileAbsent:
		if err := gs.UpdateSprintMetadata(iss.SprintID, "", info.folderPath, info.status); err != nil {
			return Fix{}, false
		}
		fixType = fixSprintStatusFixed
	default:
		return Fix{}, false
	}
	_ = audit.LogEvent("backlog_sync."+fixType, "sprint", iss.SprintID, "claude",
		map[string]any{"file_title": fileTitle, "file_status": info.status, "file_folder": info.folderPath}, "")
	return Fix{Type: fixType, SprintID: iss.SprintID}, true
}

// applySprintMetadataFix repairs db_sprint_metadata_stale issues with an UPDATE.
func applySprintMetadataFix(iss Issue) (Fix, bool) {
	if iss.sprintInfo == nil {
		return Fix{}, false
	}
	gs := store.Get()
	if gs == nil {
		return Fix{}, false
	}
	info := iss.sprintInfo
	fm := info.fm

	title := ""
	if fm != nil {
		title = fmString(fm, "title")
	}
	if title == "" {
		if t := extractTitleFromMarkdown(info.filePath, iss.SprintID); t != "" {
			title = t
		}
	}

	var mtimeStr string
	if stat, err := os.Stat(info.filePath); err == nil {
		mtimeStr = stat.ModTime().Format("2006-01-02")
	}

	startedAt := ""
	if fm != nil {
		startedAt = fmStringFirst(fm, "started", "start", "started_at")
	}
	if startedAt == "" {
		startedAt = mtimeStr
	}

	completedAt := ""
	if fm != nil {
		completedAt = fmStringFirst(fm, "completed", "end", "completed_at")
	}
	if completedAt == "" && info.status == sprintStatusCompleted {
		if ts := gitFileTimestamp(info.filePath); ts != "" {
			completedAt = ts
		} else if mtimeStr != "" {
			completedAt = mtimeStr
		}
	}

	// goal fallback — frontmatter -> SPRINT.md `## Goal`
	goal := ""
	if fm != nil {
		goal = fmString(fm, "goal")
	}
	if goal == "" {
		goal = extractGoalFromMarkdown(info.filePath)
	}

	// UpdateSprintMetadata uses COALESCE / NULLIF to preserve existing values and skip empty strings.
	if err := gs.UpdateSprintMetadata(iss.SprintID, title, "", ""); err != nil {
		return Fix{}, false
	}
	// started_at / completed_at / goal are not in UpdateSprintMetadata, so route through SaveSprint.
	// InsertSprintIfMissing is INSERT OR IGNORE and cannot overwrite existing values.
	// UpdateSprintStatus is for status transitions and is not suitable.
	// Minimum invasion: update only the title via UpdateSprintMetadata and the rest via SaveSprint.
	sprint, err := gs.GetSprint(iss.SprintID)
	if err != nil || sprint == nil {
		_ = audit.LogEvent("backlog_sync.db_sprint_metadata_fixed", "sprint", iss.SprintID, "claude",
			map[string]any{"title": title}, "")
		return Fix{Type: fixSprintMetadataFixed, SprintID: iss.SprintID}, true
	}
	if sprint.StartedAt == "" && startedAt != "" {
		sprint.StartedAt = startedAt
	}
	if sprint.CompletedAt == "" && completedAt != "" {
		sprint.CompletedAt = completedAt
	}
	if sprint.Goal == "" && goal != "" {
		sprint.Goal = goal
	}
	_ = gs.SaveSprint(sprint)

	_ = audit.LogEvent("backlog_sync.db_sprint_metadata_fixed", "sprint", iss.SprintID, "claude",
		map[string]any{"title": title, "started_at": startedAt, "completed_at": completedAt, "goal": goal}, "")
	return Fix{Type: fixSprintMetadataFixed, SprintID: iss.SprintID}, true
}

// fmStringFirst returns the first non-empty value among multiple keys in fm.
// Handles legacy / modern schema variants (start vs started, end vs completed).
func fmStringFirst(fm map[string]any, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(fmString(fm, k)); v != "" && v != "~" {
			return v
		}
	}
	return ""
}

// extractTitleFromMarkdown extracts the title from the first `# sprint-NN: Title`
// or `# Title` header in the SPRINT.md body (fallback).
func extractTitleFromMarkdown(filePath, sprintID string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inFrontmatter := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}
		if inFrontmatter {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			t := strings.TrimSpace(strings.TrimPrefix(line, "# "))
			// ": Title" → "Title"
			if idx := strings.Index(t, ": "); idx >= 0 && strings.HasPrefix(t, sprintID+":") {
				return strings.TrimSpace(t[idx+2:])
			}
			return t
		}
	}
	return ""
}

// extractGoalFromMarkdown extracts the goal from the `## Goal` section in the
// SPRINT.md body — the first non-empty paragraph.
// Lines up to the next `##` header are joined by whitespace; reads up to 10 lines.
func extractGoalFromMarkdown(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	inFrontmatter := false
	inGoal := false
	var collected []string
	const maxGoalLines = 10
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "---" {
			inFrontmatter = !inFrontmatter
			continue
		}
		if inFrontmatter {
			continue
		}
		if !inGoal {
			if strings.HasPrefix(trim, "## Goal") || strings.HasPrefix(trim, "## Goal") {
				inGoal = true
			}
			continue
		}
		// inGoal = true
		if strings.HasPrefix(trim, "## ") {
			break
		}
		if trim != "" {
			collected = append(collected, trim)
			if len(collected) >= maxGoalLines {
				break
			}
		}
	}
	return strings.Join(collected, " ")
}

// gitFileTimestamp returns the ISO 8601 timestamp of the most recent commit
// that added or modified the file (used to recover sprints with no frontmatter).
func gitFileTimestamp(filePath string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%cI", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// applySprintInsert repairs db_missing_sprint issues by INSERTing into the sprints table.
// Recovery-metadata loss fix —
// 1. title: frontmatter -> markdown `#` header -> sprintID fallback
// 2. started_at: `started` → `start` → file mtime
// 3. completed_at: `completed` -> `end` -> git log timestamp (when in completed/ folder)
// 4. created_at: `created` → `start` → file mtime → now
func applySprintInsert(iss Issue) (Fix, bool) {
	if iss.sprintInfo == nil {
		return Fix{}, false
	}
	gs := store.Get()
	if gs == nil {
		return Fix{}, false
	}
	info := iss.sprintInfo
	fm := info.fm // may be nil if frontmatter absent
	now := time.Now().Format("2006-01-02")

	// --- title ----
	title := ""
	if fm != nil {
		title = fmString(fm, "title")
	}
	if title == "" {
		if t := extractTitleFromMarkdown(info.filePath, iss.SprintID); t != "" {
			title = t
		}
	}
	if title == "" {
		title = iss.SprintID
	}

	// --- goal ---- (: frontmatter → SPRINT.md `## Goal` section fallback)
	var goal string
	if fm != nil {
		goal = fmString(fm, "goal")
	}
	if goal == "" {
		goal = extractGoalFromMarkdown(info.filePath)
	}

	// --- file mtime (fallback) ----
	var mtimeStr string
	if stat, err := os.Stat(info.filePath); err == nil {
		mtimeStr = stat.ModTime().Format("2006-01-02")
	}

	// --- started_at ----
	var startedAt string
	if fm != nil {
		startedAt = fmStringFirst(fm, "started", "start", "started_at")
	}
	if startedAt == "" {
		startedAt = mtimeStr
	}

	// --- completed_at ----
	var completedAt string
	if fm != nil {
		completedAt = fmStringFirst(fm, "completed", "end", "completed_at")
	}
	if completedAt == "" && info.status == sprintStatusCompleted {
		// Extract the file commit time from git log — the time it moved into completed/.
		if ts := gitFileTimestamp(info.filePath); ts != "" {
			completedAt = ts
		} else if mtimeStr != "" {
			completedAt = mtimeStr
		}
	}

	// --- created_at ----
	var createdAt string
	if fm != nil {
		createdAt = fmStringFirst(fm, "created", "created_at", "start", "started")
	}
	if createdAt == "" {
		if mtimeStr != "" {
			createdAt = mtimeStr
		} else {
			createdAt = now
		}
	}

	sprintRec := &ports.SprintRecord{
		SprintID:    iss.SprintID,
		Title:       title,
		Status:      info.status,
		FolderPath:  info.folderPath,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Goal:        goal,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	if err := gs.InsertSprintIfMissing(sprintRec); err != nil {
		return Fix{}, false
	}
	_ = audit.LogEvent("backlog_sync.db_sprint_inserted", "sprint", iss.SprintID, "claude",
		map[string]any{"folder_path": info.folderPath, "status": info.status, "title": title}, "")
	return Fix{Type: fixSprintInserted, SprintID: iss.SprintID}, true
}
