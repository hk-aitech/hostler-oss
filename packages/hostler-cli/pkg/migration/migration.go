// Package migration migrates a file-based project to a SQLite DB.
// Task file frontmatter parsed -> tasks table.
// SPRINT.md frontmatter parsed -> sprints table.
// task-id-registry.json merged -> includes archived Tasks whose
// files are missing.
package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
)

// --------------------------------------------------------------------------
// shared types
// --------------------------------------------------------------------------

// BootstrapInfo carries the result of "is migration required".
type BootstrapInfo struct {
	RegistryPath string `json:"registry_path"`
	LastID       int    `json:"last_id"`
	TotalCount   int    `json:"total_count"`
	Reason       string `json:"reason"`
}

// --------------------------------------------------------------------------
// Task ID normalisation
// --------------------------------------------------------------------------

// normalizeTaskID normalises Task IDs in various formats to the
// 'T{NNN}' 3-digit format.
// Supported formats:
// "", "T1", "T42" -> "", "", "".
// "123" (pure digits) -> "".
// Returns the empty string when conversion is not possible.
func normalizeTaskID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Pure digits.
	if n, err := strconv.Atoi(s); err == nil {
		return fmt.Sprintf("T%03d", n)
	}
	// String starting with T.
	re := regexp.MustCompile(`(?i)^[Tt](\d+)$`)
	if m := re.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return fmt.Sprintf("T%03d", n)
	}
	return ""
}

// --------------------------------------------------------------------------
// Task file parse
// --------------------------------------------------------------------------

// parseTaskFile parses the Task markdown file's frontmatter and
// returns it as a map. Returns nil on parse failure.
func parseTaskFile(filePath, repoRoot string) map[string]any {
	meta, err := fileutil.ReadTaskFrontmatter(filePath)
	if err != nil {
		return nil
	}

	rawID := meta["id"]
	taskID := normalizeTaskID(fmt.Sprintf("%v", rawID))
	if rawID == nil || taskID == "" {
		return nil
	}

	// depends_on process
	var dependsOn []string
	switch v := meta["depends_on"].(type) {
	case []any:
		for _, d := range v {
			dependsOn = append(dependsOn, fmt.Sprintf("%v", d))
		}
	case string:
		if v != "" {
			for _, x := range strings.Split(v, ",") {
				if t := strings.TrimSpace(x); t != "" {
					dependsOn = append(dependsOn, t)
				}
			}
		}
	}

	// file_path: relative path
	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		relPath = filePath
	}

	// created_at
	var createdAt string
	if c := meta["created"]; c != nil {
		createdAt = fmt.Sprintf("%v", c)
	}
	if createdAt == "" {
		createdAt = "unknown"
	}

	// status normalisation
	status := "todo"
	if s, ok := meta["status"]; ok && s != nil {
		status = strings.ToLower(fmt.Sprintf("%v", s))
	}

	// priority normalisation
	var priority *string
	if p, ok := meta["priority"]; ok && p != nil {
		s := strings.ToLower(fmt.Sprintf("%v", p))
		priority = &s
	}

	// estimate
	var estimate *string
	if e, ok := meta["estimate"]; ok && e != nil {
		s := fmt.Sprintf("%v", e)
		estimate = &s
	}

	// type
	taskType := "feature"
	if t, ok := meta["type"]; ok && t != nil {
		taskType = fmt.Sprintf("%v", t)
	}

	// sprint
	var sprint *string
	if s, ok := meta["sprint"]; ok && s != nil {
		sv := fmt.Sprintf("%v", s)
		if strings.ToLower(sv) != "backlog" {
			sprint = &sv
		}
	}

	// depends_on JSON
	var dependsOnJSON *string
	if len(dependsOn) > 0 {
		b, _ := json.Marshal(dependsOn)
		s := string(b)
		dependsOnJSON = &s
	}

	// title
	title := ""
	if t, ok := meta["title"]; ok && t != nil {
		title = fmt.Sprintf("%v", t)
	}

	return map[string]any{
		"task_id":    taskID,
		"title":      title,
		"type":       taskType,
		"sprint":     sprint,
		"status":     status,
		"priority":   priority,
		"estimate":   estimate,
		"file_path":  relPath,
		"depends_on": dependsOnJSON,
		"created_at": createdAt,
	}
}

// --------------------------------------------------------------------------
// Sprint file parse
// --------------------------------------------------------------------------

// cleanDate normalises a date string. "~" and future dates become nil.
func cleanDate(val any) *string {
	if val == nil {
		return nil
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", val))
	if s == "" || s == "~" {
		return nil
	}
	// Future dates are planned schedules, not actual times.
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	if t.After(time.Now()) {
		return nil
	}
	return &s
}

// parseSprintFile parses the SPRINT.md frontmatter and returns it as
// a map. Sprint state is decided by folder location — preferred over
// frontmatter status.
func parseSprintFile(sprintMDPath, location, repoRoot string) map[string]any {
	meta, err := fileutil.ReadTaskFrontmatter(sprintMDPath)
	if err != nil {
		return nil
	}

	// sprint_id
	sprintID := ""
	if id, ok := meta["id"]; ok && id != nil {
		sprintID = fmt.Sprintf("%v", id)
	}
	if sprintID == "" {
		// Extracted from folder name.
		sprintID = filepath.Base(filepath.Dir(sprintMDPath))
	}

	// State: folder location takes priority.
	statusMap := map[string]string{
		"active":     "active",
		"backlog":    "backlog",
		"completed":  "completed",
		"superseded": "superseded",
	}
	status := location
	if mapped, ok := statusMap[location]; ok {
		status = mapped
	}

	// folder_path: relative path
	dirPath := filepath.Dir(sprintMDPath)
	folderPath, err := filepath.Rel(repoRoot, dirPath)
	if err != nil {
		folderPath = dirPath
	}

	// started_at: 'started' preferred, otherwise 'start_date'
	startedAt := cleanDate(meta["started"])
	if startedAt == nil {
		startedAt = cleanDate(meta["start_date"])
	}

	// completed_at: 'completed' preferred, otherwise 'end_date'
	completedAt := cleanDate(meta["completed"])
	if completedAt == nil {
		completedAt = cleanDate(meta["end_date"])
	}

	// status=completed but no completed_at -> fall back to file mtime.
	if status == "completed" && completedAt == nil {
		if info, err := os.Stat(sprintMDPath); err == nil {
			s := info.ModTime().Format("2006-01-02")
			completedAt = &s
		}
	}

	// title
	title := sprintID
	if t, ok := meta["title"]; ok && t != nil {
		title = fmt.Sprintf("%v", t)
	}

	// goal
	var goal *string
	if g, ok := meta["goal"]; ok && g != nil {
		s := fmt.Sprintf("%v", g)
		goal = &s
	}

	return map[string]any{
		"sprint_id":    sprintID,
		"title":        title,
		"status":       status,
		"folder_path":  folderPath,
		"started_at":   startedAt,
		"completed_at": completedAt,
		"goal":         goal,
	}
}

// --------------------------------------------------------------------------
// Bootstrap detection
// --------------------------------------------------------------------------

// CheckBootstrapNeeded checks whether migration is required in O(1).
// Conditions (migration required when both are met):
// 1. The SQLite tasks table is empty.
// 2. {projectRoot}/works/data/task/task-id-registry.json exists.
// Returns nil when migration is not required.
func CheckBootstrapNeeded(projectRoot string) (*BootstrapInfo, error) {
	// 1. DB tasks count.
	gs := store.Get()
	if gs == nil {
		return nil, fmt.Errorf("DB is not initialised (call db.InitDB() + store.Init() first)")
	}

	allTasks, err := gs.GetAllTasksSync()
	if err != nil {
		return nil, fmt.Errorf("tasks COUNT lookup failed: %w", err)
	}
	if len(allTasks) > 0 {
		return nil, nil // Data already present.
	}

	// 2. task-id-registry.json file.
	registryPath := filepath.Join(projectRoot, "works", "data", "task", "task-id-registry.json")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return nil, nil // Brand-new project.
	}

	// 3. Build a registry summary to return.
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return &BootstrapInfo{
			RegistryPath: registryPath,
			Reason:       fmt.Sprintf("registry file found but read failed: %v", err),
		}, nil
	}

	var registry map[string]any
	if err := json.Unmarshal(data, &registry); err != nil {
		return &BootstrapInfo{
			RegistryPath: registryPath,
			Reason:       fmt.Sprintf("registry file found but parse failed: %v", err),
		}, nil
	}

	lastID := 0
	if v, ok := registry["lastId"]; ok {
		switch n := v.(type) {
		case float64:
			lastID = int(n)
		case int:
			lastID = n
		}
	}
	totalCount := 0
	if v, ok := registry["totalCount"]; ok {
		switch n := v.(type) {
		case float64:
			totalCount = int(n)
		case int:
			totalCount = n
		}
	}

	return &BootstrapInfo{
		RegistryPath: registryPath,
		LastID:       lastID,
		TotalCount:   totalCount,
		Reason: fmt.Sprintf(
			"existing project detected: task-id-registry.json present (lastId=T%03d, totalCount=%d items)",
			lastID, totalCount,
		),
	}, nil
}

// BuildBootstrapResponse builds the BOOTSTRAP_REQUIRED response.
func BuildBootstrapResponse(bootstrap *BootstrapInfo, projectRoot string) *BootstrapResponse {
	totalMsg := "registry"
	if bootstrap.TotalCount > 0 {
		totalMsg = fmt.Sprintf("approximately %d Task history items", bootstrap.TotalCount)
	}

	return &BootstrapResponse{
		Status:       "BOOTSTRAP_REQUIRED",
		Reason:       bootstrap.Reason,
		RegistryPath: bootstrap.RegistryPath,
		LastID:       bootstrap.LastID,
		TotalCount:   bootstrap.TotalCount,
		Message: fmt.Sprintf(
			"existing project data detected. "+
				"%s must be imported into the SQLite DB first.\n"+
				"Recommended order:\n"+
				"  1. dry-run: project_migrate(source_repo='%s', dry_run=True)\n"+
				"  2. user reviews and approves\n"+
				"  3. actual load: project_migrate(source_repo='%s', dry_run=False)\n"+
				"  4. retry this tool",
			totalMsg, projectRoot, projectRoot,
		),
		SuggestedAction: BootstrapAction{
			Tool: "project_migrate",
			Args: BootstrapActionArgs{
				SourceRepo: projectRoot,
				DryRun:     true,
			},
		},
	}
}

// --------------------------------------------------------------------------
// Main migration function
// --------------------------------------------------------------------------

// MigrateProject migrates a file-based project to SQLite.
// sourceRepo: source project root (includes works/).
// dryRun: true performs analysis only; false performs the actual load.
// The 944-LOC single function was broken into a 4-step sub-function
// orchestrator. The real logic is delegated to the parseMigrationPlan
// > reportMigrationPlan -> applyMigrationSteps helpers in
// migrate_project.go.
func MigrateProject(sourceRepo string, dryRun bool) (*MigrateResult, error) {
	repoRoot, err := filepath.Abs(sourceRepo)
	if err != nil {
		return nil, fmt.Errorf("path conversion failed: %w", err)
	}
	plan := parseMigrationPlan(repoRoot)
	result := reportMigrationPlan(plan, repoRoot, dryRun)
	if dryRun {
		return result, nil
	}
	if err := applyMigrationSteps(plan, repoRoot, result); err != nil {
		return nil, err
	}
	return result, nil
}
