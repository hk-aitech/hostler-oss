// Package migration — migrate_project.go
// Orchestrator that decomposes MigrateProject into four sub-steps.
// The original 944-line single function was split into
// parseMigrationPlan / validateMigration / applyMigrationSteps /
// reportMigrationResult.
// Why the split (deep audit):
// Resolves the issue where the test boundary was the whole
// function.
// Each step exposes a separately invocable signature -> unit
// testable.
// Reduces MigrateProject's body to orchestration responsibility.
package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// migrationPlan holds the "before-load" state collected by
// parseMigrationPlan: file scan/parse, registry merge, and Sprint scan
// results.
type migrationPlan struct {
	parsedTasks       map[string]map[string]any
	parsedSprints     map[string]map[string]any
	sprintCounts      map[string]int
	registryOnlyCount int
	maxTaskSeq        int
	scannedCount      int
	skippedReasons    map[string]int
	warnings          []string
}

// parseMigrationPlan executes MigrateProject Steps 1~4 to gather the
// state before DB loading (also reused by analyse / dry-run paths).
func parseMigrationPlan(repoRoot string) *migrationPlan {
	plan := &migrationPlan{
		parsedTasks:    map[string]map[string]any{},
		parsedSprints:  map[string]map[string]any{},
		sprintCounts:   map[string]int{"active": 0, "backlog": 0, "completed": 0, "superseded": 0},
		skippedReasons: map[string]int{},
	}

	// Step 1: detect double-nested directories and warn.
	plan.warnings = append(plan.warnings, detectDoubleNestedSprints(repoRoot)...)

	// Step 2: scan and parse Task files.
	parseTaskFilesInto(repoRoot, plan)

	// Step 3: merge with the registry.
	mergeRegistryTasks(repoRoot, plan)

	// Step 4: scan Sprint folders.
	scanSprintFiles(repoRoot, plan)

	return plan
}

// detectDoubleNestedSprints returns warnings for
// works/sprints/<loc>/<sprint-id>/<sprint-id>/ double-nested
// directories.
func detectDoubleNestedSprints(repoRoot string) []string {
	var warnings []string
	for _, location := range []string{"active", "backlog", "completed", "superseded"} {
		sprintBase := filepath.Join(repoRoot, "works", "sprints", location)
		entries, err := os.ReadDir(sprintBase)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			nested := filepath.Join(sprintBase, e.Name(), e.Name())
			if info, err := os.Stat(nested); err == nil && info.IsDir() {
				warnings = append(warnings, fmt.Sprintf(
					"[double-nested directory detected] works/sprints/%s/%s/%s/ — abnormal nesting structure. Task files under that path are still scanned.",
					location, e.Name(), e.Name(),
				))
			}
		}
	}
	return warnings
}

// parseTaskFilesInto globs works/sprints/**/tasks/T*.md plus
// works/tasks/T*.md and stores the results in parsedTasks (Step 2).
func parseTaskFilesInto(repoRoot string, plan *migrationPlan) {
	type scanSpec struct {
		forcedStatus string
		pattern      string
	}
	scanSpecs := []scanSpec{
		{"", "works/sprints/completed/*/tasks/T*.md"},
		{"", "works/sprints/active/*/tasks/T*.md"},
		{"", "works/sprints/backlog/*/tasks/T*.md"},
		{"superseded", "works/sprints/superseded/*/tasks/T*.md"},
		// Double-nested directory support.
		{"", "works/sprints/completed/*/*/tasks/T*.md"},
		{"", "works/sprints/active/*/*/tasks/T*.md"},
		{"", "works/sprints/backlog/*/*/tasks/T*.md"},
		{"superseded", "works/sprints/superseded/*/*/tasks/T*.md"},
		{"", "works/tasks/T*.md"},
	}
	type taskFileEntry struct {
		path         string
		forcedStatus string
	}
	seenFiles := map[string]bool{}
	var taskFileEntries []taskFileEntry
	for _, spec := range scanSpecs {
		matches, err := filepath.Glob(filepath.Join(repoRoot, spec.pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			if !seenFiles[m] {
				seenFiles[m] = true
				taskFileEntries = append(taskFileEntries, taskFileEntry{path: m, forcedStatus: spec.forcedStatus})
			}
		}
	}
	plan.scannedCount = len(taskFileEntries)
	for _, entry := range taskFileEntries {
		result := parseTaskFile(entry.path, repoRoot)
		if result == nil {
			plan.skippedReasons["frontmatter_parse_failed"]++
			relPath := entry.path
			if rel, err := filepath.Rel(repoRoot, entry.path); err == nil {
				relPath = rel
			}
			plan.warnings = append(plan.warnings, fmt.Sprintf("parse failed: %s", relPath))
			continue
		}
		if entry.forcedStatus != "" {
			result["status"] = entry.forcedStatus
		}
		tid, ok := result["task_id"].(string)
		if !ok || tid == "" {
			plan.warnings = append(plan.warnings, fmt.Sprintf("invalid task_id: %v", result["file_path"]))
			continue
		}
		if _, exists := plan.parsedTasks[tid]; exists {
			plan.warnings = append(plan.warnings, fmt.Sprintf("duplicate Task ID %s: %s (previous file overwritten)", tid, result["file_path"]))
		}
		plan.parsedTasks[tid] = result
	}
}

// mergeRegistryTasks merges history items from
// task-id-registry.json that have no corresponding file as archived
// Tasks (Step 3).
func mergeRegistryTasks(repoRoot string, plan *migrationPlan) {
	registryPath := filepath.Join(repoRoot, "works", "data", "task", "task-id-registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		plan.warnings = append(plan.warnings, fmt.Sprintf("task-id-registry.json missing: %s", registryPath))
	} else {
		var registryData map[string]any
		if unmarshalErr := json.Unmarshal(data, &registryData); unmarshalErr != nil {
			plan.warnings = append(plan.warnings, fmt.Sprintf("registry parse failed: %v", unmarshalErr))
		} else {
			if v, ok := registryData["lastId"]; ok {
				switch n := v.(type) {
				case float64:
					plan.maxTaskSeq = int(n)
				case int:
					plan.maxTaskSeq = n
				}
			}
			if history, ok := registryData["history"]; ok {
				if histList, ok := history.([]any); ok {
					for _, item := range histList {
						entry, ok := item.(map[string]any)
						if !ok {
							continue
						}
						rawID := ""
						if id, ok := entry["id"]; ok && id != nil {
							rawID = fmt.Sprintf("%v", id)
						}
						tid := normalizeTaskID(rawID)
						if tid == "" {
							continue
						}
						if _, exists := plan.parsedTasks[tid]; !exists {
							plan.registryOnlyCount++
							titleStr := ""
							if t, ok := entry["title"]; ok && t != nil {
								titleStr = fmt.Sprintf("%v", t)
							}
							createdAt := "unknown"
							if c, ok := entry["createdAt"]; ok && c != nil {
								createdAt = fmt.Sprintf("%v", c)
							}
							var sprint *string
							if s, ok := entry["sprint"]; ok && s != nil {
								sv := fmt.Sprintf("%v", s)
								sprint = &sv
							}
							plan.parsedTasks[tid] = map[string]any{
								"task_id":    tid,
								"title":      titleStr,
								"type":       "unknown",
								"sprint":     sprint,
								"status":     "archived",
								"priority":   (*string)(nil),
								"estimate":   (*string)(nil),
								"file_path":  "",
								"depends_on": (*string)(nil),
								"created_at": createdAt,
							}
						}
					}
				}
			}
		}
	}
	// Find the maximum sequence number across the parsed Tasks.
	reTaskSeq := regexp.MustCompile(`^T(\d+)$`)
	for tid := range plan.parsedTasks {
		if m := reTaskSeq.FindStringSubmatch(tid); m != nil {
			seq, _ := strconv.Atoi(m[1])
			if seq > plan.maxTaskSeq {
				plan.maxTaskSeq = seq
			}
		}
	}
}

// scanSprintFiles globs SPRINT.md per location into parsedSprints
// (Step 4).
func scanSprintFiles(repoRoot string, plan *migrationPlan) {
	for _, location := range []string{"active", "backlog", "completed", "superseded"} {
		pattern := filepath.Join(repoRoot, "works", "sprints", location, "*", "SPRINT.md")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, sprintMD := range matches {
			result := parseSprintFile(sprintMD, location, repoRoot)
			if result == nil {
				plan.warnings = append(plan.warnings, fmt.Sprintf("Sprint parse failed: %s", sprintMD))
				continue
			}
			sprintID := result["sprint_id"].(string)
			plan.parsedSprints[sprintID] = result
			if _, ok := plan.sprintCounts[location]; ok {
				plan.sprintCounts[location]++
			}
		}
	}
}

// reportMigrationPlan converts the plan data into a MigrateResult
// (analysis-aggregation step). It is populated using dryRun and the
// pre-load values; applyMigrationSteps fills in Loaded/Orphaned later.
func reportMigrationPlan(plan *migrationPlan, repoRoot string, dryRun bool) *MigrateResult {
	totalSkipped := 0
	for _, v := range plan.skippedReasons {
		totalSkipped += v
	}
	return &MigrateResult{
		DryRun: dryRun,
		Source: repoRoot,
		Tasks: MigrateTaskStats{
			Scanned:        plan.scannedCount,
			RegisteredOnly: plan.registryOnlyCount,
			Loaded:         0,
			Skipped:        totalSkipped,
			SkippedReasons: plan.skippedReasons,
			TotalToLoad:    len(plan.parsedTasks),
		},
		Sprints: MigrateSprintStats{
			Scanned:    len(plan.parsedSprints),
			Active:     plan.sprintCounts["active"],
			Backlog:    plan.sprintCounts["backlog"],
			Completed:  plan.sprintCounts["completed"],
			Superseded: plan.sprintCounts["superseded"],
			Loaded:     0,
		},
		Counters: MigrateCounters{
			Task: plan.maxTaskSeq,
		},
		Warnings: plan.warnings,
		Errors:   []string{},
	}
}

// applyMigrationSteps loads the plan into the actual DB (Steps 6~7).
// Only invoked on the dryRun=false path. Loaded/Orphaned counts and
// updated warnings/errors are reflected in the result.
func applyMigrationSteps(plan *migrationPlan, repoRoot string, result *MigrateResult) error {
	if err := db.InitDB(); err != nil {
		return fmt.Errorf("DB initialisation failed: %w", err)
	}
	gs, err := store.MustGet()
	if err != nil {
		return fmt.Errorf("store initialisation failed: %w", err)
	}
	tasksLoaded, sprintsLoaded, errs := insertPlanIntoDB(gs, plan)

	orphanedTasks := reconcileOrphanedTasks(gs, plan.parsedTasks, &plan.warnings)

	// Audit log.
	_ = audit.LogEvent(
		"migration.completed",
		"project",
		filepath.Base(repoRoot),
		"cli",
		map[string]any{
			"tasks":          tasksLoaded,
			"sprints":        sprintsLoaded,
			"source":         repoRoot,
			"task_counter":   plan.maxTaskSeq,
			"orphaned_tasks": orphanedTasks,
		},
		"",
	)

	result.Tasks.Loaded = tasksLoaded
	result.Tasks.Orphaned = len(orphanedTasks)
	result.Sprints.Loaded = sprintsLoaded
	result.Warnings = plan.warnings
	result.Errors = errs
	return nil
}

// insertPlanIntoDB INSERTs OR REPLACEs parsedTasks / parsedSprints /
// ufcCounters via the GraphStore. Returns the loaded counts plus an
// error-message list.
func insertPlanIntoDB(gs ports.GraphStore, plan *migrationPlan) (int, int, []string) {
	var errs []string

	tasksLoaded := 0
	for _, taskData := range plan.parsedTasks {
		tidStr, _ := taskData["task_id"].(string)
		taskNum := 0
		if n, err := fmt.Sscanf(tidStr, "T%d", &taskNum); err != nil || n != 1 {
			taskNum = 0
		}

		taskRec := &ports.TaskRecord{
			TaskID:    tidStr,
			Title:     stringVal(taskData["title"]),
			Type:      stringVal(taskData["type"]),
			Sprint:    nilableStringVal(taskData["sprint"]),
			Status:    stringVal(taskData["status"]),
			Priority:  nilableStringVal(taskData["priority"]),
			Estimate:  nilableStringVal(taskData["estimate"]),
			FilePath:  stringVal(taskData["file_path"]),
			DependsOn: nilableStringVal(taskData["depends_on"]),
			CreatedAt: stringVal(taskData["created_at"]),
		}
		if err := gs.UpsertTaskInsert(taskRec, taskNum); err != nil {
			errs = append(errs, fmt.Sprintf("Task %s load failed: %v", tidStr, err))
			continue
		}
		tasksLoaded++
	}

	sprintsLoaded := 0
	for _, sprintData := range plan.parsedSprints {
		sprintRec := &ports.SprintRecord{
			SprintID:    stringVal(sprintData["sprint_id"]),
			Title:       stringVal(sprintData["title"]),
			Status:      stringVal(sprintData["status"]),
			FolderPath:  stringVal(sprintData["folder_path"]),
			StartedAt:   nilableStringVal(sprintData["started_at"]),
			CompletedAt: nilableStringVal(sprintData["completed_at"]),
			Goal:        nilableStringVal(sprintData["goal"]),
		}
		if err := gs.UpsertSprintInsert(sprintRec); err != nil {
			errs = append(errs, fmt.Sprintf("Sprint %s load failed: %v", sprintRec.SprintID, err))
			continue
		}
		sprintsLoaded++
	}

	if plan.maxTaskSeq > 0 {
		if err := gs.SetCounterMax("task", int64(plan.maxTaskSeq)); err != nil {
			errs = append(errs, fmt.Sprintf("task counter set failed: %v", err))
		}
	}

	return tasksLoaded, sprintsLoaded, errs
}

// stringVal converts an `any` to a string; returns "" on nil.
func stringVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// nilableStringVal converts an `any` to a string; returns "" for nil
// or empty values.
func nilableStringVal(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case *string:
		if s == nil {
			return ""
		}
		return *s
	}
	return fmt.Sprintf("%v", v)
}

// reconcileOrphanedTasks switches active / in-progress Task DB
// records that were not parsed to archived (Step 7). New warnings are
// appended to plan.warnings.
func reconcileOrphanedTasks(gs ports.GraphStore, parsedTasks map[string]map[string]any, warnings *[]string) []string {
	scannedTaskIDs := map[string]bool{}
	for tid := range parsedTasks {
		scannedTaskIDs[tid] = true
	}

	nonArchived, err := gs.GetNonArchivedTasks()
	var orphanedTasks []string
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("orphan-record lookup error: %v", err))
	} else {
		for _, t := range nonArchived {
			if !scannedTaskIDs[t.TaskID] {
				orphanedTasks = append(orphanedTasks, t.TaskID)
			}
		}
	}

	if len(orphanedTasks) > 0 {
		if archiveErr := gs.ArchiveTasks(orphanedTasks); archiveErr == nil {
			*warnings = append(*warnings, fmt.Sprintf(
				"[orphan record cleanup] %d Task(s) without files transitioned to archived: %s",
				len(orphanedTasks), strings.Join(orphanedTasks, ", "),
			))
		}
	}
	return orphanedTasks
}
