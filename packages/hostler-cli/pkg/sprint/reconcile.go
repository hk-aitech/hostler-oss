// Package sprint — reconcile.go
// Recovery tool that uses the file as the SSOT and re-syncs the DB
// when worktree parallel work has caused DB records and the
// filesystem to drift. Acts as the shared engine for solutions A
// (extend update) and C (introduce reconcile) among the four
// candidates A/B/C/D, and the helper here is also reused by D
// (create guard).
package sprint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// sprintLocations — the three filesystem locations where a Sprint
// can live. Order is the search priority (active -> backlog ->
// completed).
var sprintLocations = []string{"active", "backlog", "completed"}

// FindSprintOnDisk searches works/sprints/{active,backlog,completed}/<id>/SPRINT.md
// and returns the actual file location. NotFound on miss.
// trac: HAR-CM005
func FindSprintOnDisk(sprintID string) (*domain.LocatedSprint, error) {
	root := fileutil.GetProjectRoot()
	for _, loc := range sprintLocations {
		sprintDir := filepath.Join(root, "works", "sprints", loc, sprintID)
		sprintMD := filepath.Join(sprintDir, "SPRINT.md")
		if _, err := os.Stat(sprintMD); err != nil {
			continue
		}
		fm, _ := fileutil.ReadTaskFrontmatter(sprintMD) // best-effort
		rel, _ := filepath.Rel(root, sprintDir)
		return &domain.LocatedSprint{
			SprintID:    sprintID,
			Location:    loc,
			FolderPath:  filepath.ToSlash(rel),
			SprintMD:    sprintMD,
			Frontmatter: fm,
		}, nil
	}
	return nil, &apperr.NotFoundError{
		EntityType:   "Sprint",
		EntityID:     sprintID,
		RecoveryHint: "verify that SPRINT.md exists under works/sprints/{active,backlog,completed}/.",
	}
}

// fmStr safely extracts a string value from a frontmatter map.
func fmStr(fm map[string]any, key string) string {
	if fm == nil {
		return ""
	}
	if v, ok := fm[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return ""
}

// Reconcile re-syncs the Sprint DB record with the filesystem as the
// SSOT. Recovers the situation where parallel worktree edits
// overwrote the DB record for the same sprint_id. dryRun=true detects
// drift without modifying the DB.
// trac: HAR-CM005
func Reconcile(sprintID string, dryRun bool) (*domain.ReconcileResult, error) {
	located, err := FindSprintOnDisk(sprintID)
	if err != nil {
		return nil, err
	}

	fileTitle := fmStr(located.Frontmatter, "title")
	if fileTitle == "" {
		fileTitle = sprintID
	}
	fileGoal := fmStr(located.Frontmatter, "goal")
	fileStatus := located.Location // folder location is the SSOT

	rec, err := GetFromDB(sprintID)
	dbExists := err == nil && rec != nil

	var diffs []domain.ReconcileDiff
	if dbExists {
		if rec.Title != fileTitle {
			diffs = append(diffs, domain.ReconcileDiff{Field: "title", DBValue: rec.Title, File: fileTitle})
		}
		if rec.Status != fileStatus {
			diffs = append(diffs, domain.ReconcileDiff{Field: "status", DBValue: rec.Status, File: fileStatus})
		}
		if rec.FolderPath != located.FolderPath {
			diffs = append(diffs, domain.ReconcileDiff{Field: "folder_path", DBValue: rec.FolderPath, File: located.FolderPath})
		}
		if fileGoal != "" && rec.Goal != fileGoal {
			diffs = append(diffs, domain.ReconcileDiff{Field: "goal", DBValue: rec.Goal, File: fileGoal})
		}
	} else {
		diffs = append(diffs, domain.ReconcileDiff{Field: "record", DBValue: "(missing)", File: "(new entry from file)"})
	}

	// Detect Task DB drift.
	taskDrifts, _ := reconcileSprintTaskDrift(sprintID, dryRun)

	result := &domain.ReconcileResult{
		SprintID:   sprintID,
		DryRun:     dryRun,
		Applied:    false,
		Diffs:      diffs,
		TaskDrifts: taskDrifts,
		FolderPath: located.FolderPath,
		Status:     fileStatus,
		Title:      fileTitle,
	}

	if dryRun || len(diffs) == 0 {
		return result, nil
	}

	// SaveToDB is an UPSERT, accepting both new and existing records.
	var startedAt, completedAt *string
	if s := fmStr(located.Frontmatter, "started"); s != "" && s != "~" {
		startedAt = &s
	}
	if s := fmStr(located.Frontmatter, "completed"); s != "" && s != "~" {
		completedAt = &s
	}
	if err := SaveToDB(sprintID, fileTitle, fileGoal, fileStatus, located.FolderPath, startedAt, completedAt); err != nil {
		return result, err
	}
	result.Applied = true
	return result, nil
}

// taskSearchDirs lists locations where a Task file may live (relative
// to root). Sprint-bound locations are added dynamically inside
// reconcileSprintTaskDrift.
var taskSearchDirs = []string{
	"works/tasks",
	"works/tasks/completed",
}

// reconcileSprintTaskDrift compares the DB file_path of each Task in
// the Sprint with the actual file location and returns the mismatch
// list. When dryRun=false, mismatching Tasks have their DB file_path
// updated to the actual location.
func reconcileSprintTaskDrift(sprintID string, dryRun bool) ([]domain.TaskDrift, error) {
	gs := store.Get()
	if gs == nil {
		return nil, nil
	}
	sprintFilter := sprintID
	res, err := gs.ListTasks(&sprintFilter, nil)
	if err != nil || res == nil {
		return nil, err
	}

	root := fileutil.GetProjectRoot()

	// Add the Sprint-bound search locations dynamically —
	// active/backlog/completed/discarded x sprintID.
	searchDirs := make([]string, len(taskSearchDirs))
	copy(searchDirs, taskSearchDirs)
	for _, loc := range append(sprintLocations, "discarded") {
		searchDirs = append(searchDirs, filepath.Join("works", "sprints", loc, sprintID, "tasks"))
	}

	var drifts []domain.TaskDrift
	for _, t := range res.Tasks {
		dbPath := t.FilePath
		if dbPath == "" {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(dbPath))
		if _, statErr := os.Stat(abs); statErr == nil {
			continue // normal — file exists at DB file_path
		}
		// File missing — search for the actual location.
		actualPath := findTaskFile(root, t.TaskID, searchDirs)
		drifts = append(drifts, domain.TaskDrift{
			TaskID:     t.TaskID,
			DBFilePath: dbPath,
			ActualPath: actualPath,
		})
		if !dryRun && actualPath != "" {
			_ = gs.UpdateTaskFilePath(t.TaskID, actualPath)
		}
	}
	return drifts, nil
}

// findTaskFile searches each searchDir for an .md file whose name
// starts with taskID and returns the first match's path
// (repo-relative).
func findTaskFile(root, taskID string, searchDirs []string) string {
	prefix := taskID + "-"
	for _, dir := range searchDirs {
		absDir := filepath.Join(root, filepath.FromSlash(dir))
		entries, err := os.ReadDir(absDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".md") {
				rel, _ := filepath.Rel(root, filepath.Join(absDir, name))
				return filepath.ToSlash(rel)
			}
		}
	}
	return ""
}

// UpdateFields selectively updates fields on the Sprint DB record.
// Only non-nil fields among title/goal/status/folder_path are
// UPDATEd. status updates can override the lifecycle rule, so the
// caller must confirm `--force` first.
// trac: HAR-CM004
func UpdateFields(sprintID string, title, goal, status, folderPath *string) error {
	rec, err := GetFromDB(sprintID)
	if err != nil {
		return err
	}
	newTitle := rec.Title
	if title != nil {
		newTitle = *title
	}
	newGoal := rec.Goal
	if goal != nil {
		newGoal = *goal
	}
	newStatus := rec.Status
	if status != nil {
		newStatus = *status
	}
	newFolder := rec.FolderPath
	if folderPath != nil {
		newFolder = *folderPath
	}
	var startedPtr, completedPtr *string
	if rec.StartedAt != "" {
		s := rec.StartedAt
		startedPtr = &s
	}
	if rec.CompletedAt != "" {
		s := rec.CompletedAt
		completedPtr = &s
	}
	return SaveToDB(sprintID, newTitle, newGoal, newStatus, newFolder, startedPtr, completedPtr)
}

// CheckCreateGuard returns a non-destructive warning when a DB
// record for the same sprint_id exists with a different title. When
// the caller wants to overwrite via --force, this function is
// skipped.
// trac: HAR-CM001
func CheckCreateGuard(sprintID, newTitle string) error {
	rec, err := GetFromDB(sprintID)
	if err != nil || rec == nil {
		return nil // not in DB — normal create path
	}
	if rec.Title == newTitle {
		return nil
	}
	return &apperr.RejectedError{
		EntityID: sprintID,
		Reason: fmt.Sprintf(
			"a DB record for Sprint %s already exists with a different title (DB=%q, new=%q). Possible parallel-worktree edit.",
			sprintID, rec.Title, newTitle),
		RecoveryHint: "to recover from the file, run `hstl sprint reconcile " + sprintID + "`; to overwrite, use the `--force` flag.",
	}
}
