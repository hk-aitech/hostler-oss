// Sprint deprecation path.
// What it does: transitions a Sprint from backlog or active into the
// discarded state. Moves the folder to works/sprints/discarded/<id>/,
// updates the SPRINT.md frontmatter, and records the audit event
// sprint.discarded. With the --return-tasks option, unfinished tasks
// (todo / in-progress) are unassigned automatically and returned to
// the backlog.
// Design constraints:
// completed Sprints cannot be discarded (the historical area is
// immutable).
// reason must be at least 10 characters (matches the reopen / delete
// convention).
// No HMAC signature (preserves the completed chain's integrity).

package sprint

import (
	"fmt"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

const (
	discardReasonMinLen = 10
	discardLocation     = "discarded"
)

// Discard transitions a Sprint into the discarded state.
// Caller passes reason (>= 10 chars), returnTasks (whether to
// auto-unassign unfinished tasks), and dbOnly (DB-drift cleanup
// without folder move). completed Sprints are rejected. After a
// successful transition, audit.LogEvent("sprint.discarded") is
// recorded.
// dbOnly=true skips MoveSprint and only sets the DB status to
// discarded. This is the official cleanup path when the folder was
// removed manually or lost to a git conflict, leaving DB drift.
// trac: HAR-CM004
func Discard(id, reason string, returnTasks, dbOnly bool) (*domain.SprintDiscardResult, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < discardReasonMinLen {
		return nil, &apperr.RejectedError{
			EntityID:       id,
			Reason:         fmt.Sprintf("reason must be at least %d characters", discardReasonMinLen),
			ProvidedLength: len(reason),
			RecoveryHint:   "supply a deprecation reason of at least 10 characters. Example: --reason \"absorbed by sprint-37 due to scope overlap\"",
		}
	}

	rec, err := GetFromDB(id)
	if err != nil {
		return nil, err
	}

	switch rec.Status {
	case "completed":
		return nil, &apperr.RejectedError{
			EntityID:     id,
			Reason:       "completed Sprints cannot be discarded (historical area is immutable)",
			CurrentState: rec.Status,
			RecoveryHint: "completed Sprints have no deprecation path. If completed by mistake, run `sprint update --status active --force` to revert, then discard.",
		}
	case discardLocation:
		return nil, &apperr.RejectedError{
			EntityID:     id,
			Reason:       "Sprint is already in the discarded state",
			CurrentState: rec.Status,
		}
	case "backlog", "active":
		// proceed
	default:
		return nil, &apperr.RejectedError{
			EntityID:     id,
			Reason:       fmt.Sprintf("status %s does not allow discard. Only backlog/active Sprints may be discarded.", rec.Status),
			CurrentState: rec.Status,
		}
	}

	fromLocation := rec.Status

	// -return-tasks: auto-unassign unfinished tasks.
	var returnedTasks []string
	if returnTasks {
		taskIDs, collectErr := collectUnfinishedTaskIDs(id)
		if collectErr != nil {
			return nil, fmt.Errorf("unfinished-task collection failed: %w", collectErr)
		}
		if len(taskIDs) > 0 {
			unassigned, unassignErr := unassignTasks(taskIDs)
			if unassignErr != nil {
				return nil, fmt.Errorf("task unassign failed: %w", unassignErr)
			}
			returnedTasks = unassigned
		}
	}

	var newRelPath string

	if dbOnly {
		// -db-only path — only update DB status, no folder move.
		// Used when the folder was deleted manually or lost in a git
		// conflict.
		if updateErr := UpdateStatusInDB(id, discardLocation, nil, nil, nil); updateErr != nil {
			return nil, fmt.Errorf("DB status update failed: %w", updateErr)
		}
	} else {
		// Standard path: folder move (prefer git mv, fall back to
		// os.Rename).
		newPath, moveErr := MoveSprint(id, fromLocation, discardLocation)
		if moveErr != nil {
			return nil, fmt.Errorf("Sprint folder move failed: %w", moveErr)
		}
		newRelPath = fileutil.ToRepoRelative(newPath)

		// SPRINT.md frontmatter status update.
		sprintMDPath := newPath + "/SPRINT.md"
		_ = UpdateSprintMDField(sprintMDPath, "status", discardLocation)

		// DB status update.
		if updateErr := UpdateStatusInDB(id, discardLocation, &newRelPath, nil, nil); updateErr != nil {
			return nil, fmt.Errorf("DB status update failed: %w", updateErr)
		}

		// Update Task file_path to follow the folder move.
		updateTaskPaths(id, fromLocation, discardLocation, "")
	}

	// CURRENT-FOCUS update — discarding the active Sprint returns to
	// idle.
	if fromLocation == "active" {
		_ = fileutil.UpdateCurrentFocus("", "", "sprint-idle")
	}

	// audit: sprint.discarded
	_ = audit.LogEvent("sprint.discarded", "sprint", id, "claude", map[string]any{
		"from_status":    fromLocation,
		"to_status":      discardLocation,
		"reason":         reason,
		"returned_tasks": returnedTasks,
		"return_tasks":   returnTasks,
		"db_only":        dbOnly,
	}, "")

	return &domain.SprintDiscardResult{
		SprintID:       id,
		PreviousStatus: fromLocation,
		Status:         discardLocation,
		Reason:         reason,
		FolderPath:     newRelPath,
		ReturnedTasks:  returnedTasks,
		DBOnly:         dbOnly,
	}, nil
}

// collectUnfinishedTaskIDs returns the IDs of todo / in-progress tasks
// that belong to the Sprint.
func collectUnfinishedTaskIDs(sprintID string) ([]string, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, err
	}
	sprintFilter := sprintID
	res, listErr := gs.ListTasks(&sprintFilter, nil)
	if listErr != nil || res == nil {
		return nil, listErr
	}
	var ids []string
	for _, t := range res.Tasks {
		if t.Status == "todo" || t.Status == "in-progress" {
			ids = append(ids, t.TaskID)
		}
	}
	return ids, nil
}

// unassignTasks unassigns the Tasks and returns the IDs that succeeded.
// pkg/task.UnassignSprint is not called from here to avoid the import
// cycle (pkg/task may use pkg/sprint).
func unassignTasks(taskIDs []string) ([]string, error) {
	gs, err := store.MustGet()
	if err != nil {
		return nil, err
	}
	var unassigned []string
	for _, id := range taskIDs {
		if updateErr := gs.UpdateTaskSprint(id, nil); updateErr == nil {
			unassigned = append(unassigned, id)
		}
	}
	return unassigned, nil
}
