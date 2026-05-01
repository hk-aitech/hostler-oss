// Package backlog — full BACKLOG.md regeneration.
// Re-renders works/tasks/BACKLOG.md from the DB tasks table from
// scratch. Used as a one-shot recovery in environments that have
// many existing Task files but no index file (typical when imported
// to or from a sibling tool). Distinct from per-row updates
// (fileutil.UpdateBacklogMD) — this is the "full regenerate"
// scenario.
package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// RebuildMDResult is the result of a RebuildMD call.
type RebuildMDResult struct {
	BackupPath string `json:"backup_path,omitempty"`
	FilePath   string `json:"file_path"`
	Count      int    `json:"count"`
	DryRun     bool   `json:"dry_run,omitempty"`
	// WouldBackup carries the planned backup path during DryRun when
	// an existing file is present.
	WouldBackup string `json:"would_backup,omitempty"`
}

// RebuildMDOptions is the option struct for RebuildMD.
type RebuildMDOptions struct {
	// IncludeInProgress, when true, also includes in-progress Tasks in
	// the unassigned section.
	IncludeInProgress bool
	// DryRun, when true, returns only the computed result without
	// writing the file.
	DryRun bool
	// Backup, when true, copies the existing BACKLOG.md to .bak. The
	// default is false to avoid `.bak` accumulation in automation
	// environments; manual runs opt in via --backup.
	Backup bool
}

// rebuildMDHeader is the BACKLOG.md skeleton header used during
// regeneration.
const rebuildMDHeader = `# Backlog

The project's unassigned Task list (Tasks not bound to a Sprint).
` + "`hstl task create`" + ` / ` + "`hstl task assign`" + ` keep this file in sync
automatically. Use ` + "`hstl backlog rebuild-md`" + ` to regenerate it from
scratch.

## Unassigned Tasks

| ID | title | size | type | priority | depends_on |
|----|------|------|------|----------|------------|
`

// priorityOrder maps a priority string to an integer for sorting.
var priorityOrder = map[string]int{
	"p0": 0,
	"p1": 1,
	"p2": 2,
	"p3": 3,
}

// RebuildMD reads the DB tasks table and regenerates
// works/tasks/BACKLOG.md. If a BACKLOG.md already exists, it is
// backed up to `.bak` (with --backup) before being overwritten.
func RebuildMD(opts RebuildMDOptions) (*RebuildMDResult, error) {
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised")
	}

	// Look up unassigned (and optionally in-progress) Tasks.
	backlogTasks, err := gs.GetBacklogTasks(opts.IncludeInProgress)
	if err != nil {
		return nil, fmt.Errorf("tasks lookup failed: %w", err)
	}

	type row struct {
		id, title, taskType, estimate, priority string
	}
	var items []row
	for _, t := range backlogTasks {
		r := row{
			id:       t.TaskID,
			title:    t.Title,
			taskType: t.Type,
			estimate: t.Estimate,
			priority: t.Priority,
		}
		if r.taskType == "" {
			r.taskType = "chore"
		}
		if r.estimate == "" {
			r.estimate = "M"
		}
		if r.priority == "" {
			r.priority = "p2"
		}
		items = append(items, r)
	}

	// Sort: priority ascending -> task_id numeric ascending.
	sort.SliceStable(items, func(i, j int) bool {
		pi := priorityOrder[items[i].priority]
		pj := priorityOrder[items[j].priority]
		if pi != pj {
			return pi < pj
		}
		return taskIDNumeric(items[i].id) < taskIDNumeric(items[j].id)
	})

	// Resolve the file path.
	root := fileutil.GetProjectRoot()
	backlogDir := filepath.Join(root, "works", "tasks")
	backlogPath := filepath.Join(backlogDir, "BACKLOG.md")

	result := &RebuildMDResult{FilePath: backlogPath, Count: len(items), DryRun: opts.DryRun}

	// DryRun: return only the computed result without filesystem
	// changes.
	if opts.DryRun {
		if opts.Backup {
			if _, err := os.Stat(backlogPath); err == nil {
				result.WouldBackup = backlogPath + ".bak"
			}
		}
		return result, nil
	}

	if err := os.MkdirAll(backlogDir, 0o755); err != nil {
		return nil, fmt.Errorf("BACKLOG.md directory creation failed: %w", err)
	}

	// Back up the existing file — only on opt-in via --backup.
	if opts.Backup {
		if existing, err := os.ReadFile(backlogPath); err == nil {
			backupPath := backlogPath + ".bak"
			if err := os.WriteFile(backupPath, existing, 0o644); err != nil {
				return nil, fmt.Errorf("existing BACKLOG.md backup failed: %w", err)
			}
			result.BackupPath = backupPath
		}
	}

	// Regenerate.
	var body strings.Builder
	body.WriteString(rebuildMDHeader)
	for _, it := range items {
		fmt.Fprintf(&body, "| %s | %s | %s | %s | %s | — |\n",
			it.id, it.title, it.estimate, it.taskType, it.priority)
	}
	if err := os.WriteFile(backlogPath, []byte(body.String()), 0o644); err != nil {
		return nil, fmt.Errorf("BACKLOG.md write failed: %w", err)
	}

	return result, nil
}

// taskIDNumeric extracts the numeric portion from a ""-formatted
// Task ID. On failure, returns a very large number so the entry sorts
// to the end.
func taskIDNumeric(id string) int {
	if len(id) < 2 || id[0] != 'T' {
		return 1 << 30
	}
	n := 0
	for _, c := range id[1:] {
		if c < '0' || c > '9' {
			return 1 << 30
		}
		n = n*10 + int(c-'0')
	}
	return n
}
