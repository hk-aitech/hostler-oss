// Package task — work_ticket reconcile.
// Background: with parallel worktrees plus concurrent hstl invocations,
// the tasks.work_ticket column can drift from the frontmatter
// `work_ticket` field. The backlog sync only fills work_ticket on INSERT
// and does not cover the UPDATE path, so when a task start in another
// worktree updates work_ticket, the other worktree's DB drifts.
// Fix: compare every task's frontmatter work_ticket with DB
// tasks.work_ticket, classify drift (missing_in_db / missing_in_file /
// mismatch) and, with --fix, update the DB using the file as the SSOT.
package task

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TicketDrift describes a single task's work_ticket drift entry.
type TicketDrift struct {
	TaskID    string `json:"task_id"`
	FilePath  string `json:"file_path"`
	DBTicket  string `json:"db_ticket"`
	FileTicke string `json:"file_ticket"`
	Kind      string `json:"kind"` // "missing_in_db" | "missing_in_file" | "mismatch" | "ok"
}

// ReconcileResult aggregates the reconcile run.
type ReconcileResult struct {
	TotalTasks int           `json:"total_tasks"`
	OkCount    int           `json:"ok_count"`
	DriftCount int           `json:"drift_count"`
	Drifts     []TicketDrift `json:"drifts"`
	FixedCount int           `json:"fixed_count"`
	DryRun     bool          `json:"dry_run"`
	FixApplied bool          `json:"fix_applied"`
}

// ReconcileTickets verifies work_ticket consistency for every task.
// Parameters:
// database: connection to the tasks table.
// dryRun: when true, only report drift; when false plus
// applyFix=true, update the DB.
// applyFix: --fix flag — update the DB using the file as the SSOT.
// Drift classification:
// "missing_in_db": frontmatter has work_ticket but DB is empty.
// "missing_in_file": DB has work_ticket but frontmatter is missing.
// "mismatch": both are present but disagree.
// "ok": values match.
// trac: HAR-CM005
func ReconcileTickets(database *sql.DB, dryRun, applyFix bool) (*ReconcileResult, error) {
	type taskRow struct {
		taskID, filePath, dbTicket string
	}
	// Step 1: collect all rows up-front (avoids UPDATE / SELECT cursor
	// conflicts).
	rows, err := database.Query(`SELECT task_id, file_path, COALESCE(work_ticket, '') FROM tasks ORDER BY task_id`)
	if err != nil {
		return nil, fmt.Errorf("tasks query failed: %w", err)
	}
	var collected []taskRow
	for rows.Next() {
		var r taskRow
		if scanErr := rows.Scan(&r.taskID, &r.filePath, &r.dbTicket); scanErr != nil {
			rows.Close()
			return nil, fmt.Errorf("row scan failed: %w", scanErr)
		}
		collected = append(collected, r)
	}
	if cErr := rows.Err(); cErr != nil {
		rows.Close()
		return nil, cErr
	}
	rows.Close()

	result := &ReconcileResult{DryRun: dryRun, FixApplied: applyFix && !dryRun}

	for _, r := range collected {
		taskID, filePath, dbTicket := r.taskID, r.filePath, r.dbTicket
		result.TotalTasks++

		fileTicket, fmErr := extractFrontmatterWorkTicket(filePath)
		if fmErr != nil {
			// File missing / read failure — classify as drift
			// (missing_in_file).
			drift := TicketDrift{
				TaskID:    taskID,
				FilePath:  filePath,
				DBTicket:  dbTicket,
				FileTicke: "",
				Kind:      "missing_in_file",
			}
			result.Drifts = append(result.Drifts, drift)
			result.DriftCount++
			continue
		}

		drift := TicketDrift{
			TaskID:    taskID,
			FilePath:  filePath,
			DBTicket:  dbTicket,
			FileTicke: fileTicket,
		}
		switch {
		case dbTicket == fileTicket:
			drift.Kind = "ok"
			result.OkCount++
			continue
		case dbTicket == "" && fileTicket != "":
			drift.Kind = "missing_in_db"
		case dbTicket != "" && fileTicket == "":
			drift.Kind = "missing_in_file"
		default:
			drift.Kind = "mismatch"
		}
		result.Drifts = append(result.Drifts, drift)
		result.DriftCount++

		// -fix: update the DB using the file as the SSOT (skip
		// missing_in_file because there is no source).
		if applyFix && !dryRun && drift.Kind != "missing_in_file" {
			if _, uErr := database.Exec(
				`UPDATE tasks SET work_ticket = ? WHERE task_id = ?`,
				fileTicket, taskID,
			); uErr == nil {
				result.FixedCount++
			}
		}
	}
	return result, nil
}

// extractFrontmatterWorkTicket extracts the work_ticket field from the
// YAML frontmatter of a markdown file. Missing field returns an empty
// string + nil error.
func extractFrontmatterWorkTicket(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if lineNum == 1 {
			if strings.TrimSpace(line) != "---" {
				return "", nil
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			break
		}
		if !inFrontmatter {
			break
		}
		if rest, ok := strings.CutPrefix(line, "work_ticket:"); ok {
			val := strings.TrimSpace(rest)
			val = strings.Trim(val, `"'`)
			return val, nil
		}
	}
	return "", nil
}
