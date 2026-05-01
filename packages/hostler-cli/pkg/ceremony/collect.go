package ceremony

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/events"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/harness"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/sprint"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/store"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// --------------------------------------------------------------------------
// Sprint Start Ceremony
// --------------------------------------------------------------------------

// CollectSprintStart gathers ceremony data at sprint-start time.
func CollectSprintStart(sprintID string) *SprintStartCeremony {
	cer := &SprintStartCeremony{}

	// 1. Briefing data.
	cer.Briefing = collectSprintBriefing(sprintID)

	// 2. Design readiness.
	cer.DesignReadiness = collectDesignReadiness(sprintID)

	// 3. Reminders.
	cer.Reminders = config.GetReminders(string(events.EventSprintStart))

	return cer
}

// collectSprintBriefing gathers the Sprint briefing payload.
func collectSprintBriefing(sprintID string) *SprintBriefing {
	briefing := &SprintBriefing{SprintID: sprintID}

	// Look up Sprint metadata.
	rec, err := sprint.GetFromDB(sprintID)
	if err != nil {
		return briefing
	}
	briefing.Title = rec.Title
	briefing.Goal = rec.Goal

	// Look up the Tasks bound to the Sprint.
	sprintPtr := sprintID
	raw, err := task.List(&sprintPtr, nil)
	if err != nil {
		return briefing
	}

	lr, ok := raw.(*domain.ListResult)
	if !ok || lr == nil {
		return briefing
	}

	briefing.TaskCount = lr.Count
	for _, t := range lr.Tasks {
		var deps []string
		if len(t.DependsOn) > 0 {
			deps = t.DependsOn
		}
		briefing.Tasks = append(briefing.Tasks, TaskSummary{
			TaskID:    t.TaskID,
			Title:     t.Title,
			Type:      t.Type,
			Size:      t.Estimate,
			Priority:  t.Priority,
			Status:    t.Status,
			DependsOn: deps,
		})
	}

	return briefing
}

// collectDesignReadiness gathers the design-readiness check items.
func collectDesignReadiness(sprintID string) []ReadinessCheck {
	var checks []ReadinessCheck

	// Look up the Tasks bound to the Sprint.
	sprintPtr := sprintID
	raw, err := task.List(&sprintPtr, nil)
	if err != nil {
		return checks
	}
	lr, ok := raw.(*domain.ListResult)
	if !ok || lr == nil {
		return checks
	}

	// 1. Check whether any placeholder body remains.
	var placeholders []string
	for _, t := range lr.Tasks {
		if task.HasPlaceholderBody(t.TaskID) {
			placeholders = append(placeholders, t.TaskID)
		}
	}
	if len(placeholders) > 0 {
		// task assign already blocks placeholder bodies at Sprint-
		// assignment time, so this is an auxiliary warning (info).
		// If the normal path was followed, we should never reach
		// here.
		checks = append(checks, ReadinessCheck{
			Check:  "placeholder body remaining",
			Status: "info",
			Detail: strings.Join(placeholders, ", ") + " (task assign block already applied as the primary gate)",
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Check:  "placeholder body remaining",
			Status: "pass",
			Detail: "0 items",
		})
	}

	// 3. Check for incomplete dependent Tasks.
	var unmetDeps []string
	for _, t := range lr.Tasks {
		for _, depID := range t.DependsOn {
			depResult, depErr := task.Get(depID)
			if depErr != nil {
				continue
			}
			if gr, gOK := depResult.(*domain.GetResult); gOK && gr.Status != "done" {
				unmetDeps = append(unmetDeps, depID)
			}
		}
	}
	if len(unmetDeps) > 0 {
		checks = append(checks, ReadinessCheck{
			Check:  "dependent Task incomplete",
			Status: "info",
			Detail: strings.Join(unmetDeps, ", "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Check:  "dependent Tasks done",
			Status: "pass",
		})
	}

	// 4. Stale body check: gap between Task body write date and
	// kick-off date. When any Task exceeds the threshold, emit a
	// "review the body" warning.
	if stale := collectStaleBodyCheck(lr.Tasks); stale != nil {
		checks = append(checks, *stale)
	}

	// 5. Per-project extension verification scripts.
	// The hstl core does not hard-code domain keywords. When a
	// project registers script paths under
	// project-config.yaml's sprints.ceremony.design_checks,
	// each script is invoked as a subprocess; result lines are
	// parsed and accumulated as additional ReadinessCheck items.
	// The previous domain-keyword-based findTimeConstrainedTasks
	// helper was fully removed and replaced by this hook.
	if extra := runProjectDesignChecks(lr.Tasks); len(extra) > 0 {
		checks = append(checks, extra...)
	}

	// 6. Verify the existence of backtick paths in the Task body.
	// Prevents the "claimed but non-existent path" recurrence.
	// Only paths containing a slash are checked; missing files
	// produce a WARN (not BLOCK — the Task itself is allowed to
	// proceed).
	if check := collectBacktickRefCheck(lr.Tasks); check != nil {
		checks = append(checks, *check)
	}

	// 7. Auto-warn when Tasks are estimated L/XL (split trigger).
	// Enforces the L/XL split rule from estimate-guidelines as a
	// WARN — intentional L/XL is allowed but must be acknowledged
	// before proceeding.
	if check := collectLXLEstimateCheck(lr.Tasks); check != nil {
		checks = append(checks, *check)
	}

	// 8. Early warning for divergence vs main.
	// Empirically, with dev 34 commits ahead, the first commit on a
	// new sprint was blocked by precommit.branch.sync (BLOCK
	// threshold 30) and the AI had to merge first. When the divergence
	// reaches the WARN threshold (25), surface it in the briefing.
	if check := collectMainDivergenceCheck(); check != nil {
		checks = append(checks, *check)
	}

	return checks
}

// lxlExemptTypes — task types exempted from the L/XL split advisory.
// Rationale:
// spike: a single artefact (one comparison matrix or design doc)
// splitting is inefficient.
// docs: a single document (ADR amendment / guide) — splitting
// is inefficient.
// Other types (feature/refactor/bugfix/infra/test/chore) remain
// subject to the split advisory.
var lxlExemptTypes = map[string]string{
	"spike": "single artefact (comparison matrix / design doc)",
	"docs":  "single document (ADR amendment / guide)",
}

// collectLXLEstimateCheck returns a WARN when any Sprint Task has an
// L/XL estimate. Per the split guide, L/XL Tasks should be
// pre-split into 2~3 M Tasks.
// Per-task-type exemption: spike + docs are single-artefact units
// where splitting is inefficient -> exempted. Exempted items are
// shown separately in the detail with their rationale.
func collectLXLEstimateCheck(tasks []domain.TaskSummary) *ReadinessCheck {
	var lxl []string
	var exempt []string
	for _, t := range tasks {
		if t.Estimate != "L" && t.Estimate != "XL" {
			continue
		}
		if reason, ok := lxlExemptTypes[t.Type]; ok {
			exempt = append(exempt, fmt.Sprintf("%s(%s/%s — %s)", t.TaskID, t.Estimate, t.Type, reason))
			continue
		}
		lxl = append(lxl, fmt.Sprintf("%s(%s)", t.TaskID, t.Estimate))
	}
	if len(lxl) == 0 {
		detail := "L/XL estimates: 0"
		if len(exempt) > 0 {
			detail = fmt.Sprintf("L/XL estimates: 0 (subject to split advisory). Exempt %d (spike/docs): %s",
				len(exempt), strings.Join(exempt, ", "))
		}
		return &ReadinessCheck{
			Check:  "L/XL Task split trigger",
			Status: "pass",
			Detail: detail,
		}
	}
	detail := fmt.Sprintf("L/XL estimates: %d — pre-split into 2~3 M Tasks recommended: %s",
		len(lxl), strings.Join(lxl, ", "))
	if len(exempt) > 0 {
		detail += fmt.Sprintf(". Exempt %d (spike/docs): %s",
			len(exempt), strings.Join(exempt, ", "))
	}
	return &ReadinessCheck{
		Check:  "L/XL Task split trigger",
		Status: "warn",
		Detail: detail,
	}
}

// defaultStalePathPct — default stale-ratio threshold (30%).
// Override via the HSTL_READINESS_STALE_PATH_PCT env var (float,
// 0~100).
const defaultStalePathPct = 30.0

// stalePathPctThreshold reads the env-var override and returns the
// threshold in percent.
func stalePathPctThreshold() float64 {
	if v := envalias.Lookup("READINESS_STALE_PATH_PCT"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 0 {
			return n
		}
	}
	return defaultStalePathPct
}

// collectBacktickRefCheck verifies the existence of backtick-quoted
// paths in Sprint Task bodies. Returns WARN when the stale-ratio
// exceeds the threshold.
// Ratio: stale_count / total_ref_count * 100 >
// HSTL_READINESS_STALE_PATH_PCT (default 30).
func collectBacktickRefCheck(tasks []domain.TaskSummary) *ReadinessCheck {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	type staleRef struct {
		taskID string
		ref    string
	}
	var stales []staleRef
	totalRefs := 0
	for _, t := range tasks {
		details, detailsErr := task.Get(t.TaskID)
		if detailsErr != nil {
			continue
		}
		gr, ok := details.(*domain.GetResult)
		if !ok || gr == nil || gr.FilePath == "" {
			continue
		}
		absPath := gr.FilePath
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(cwd, gr.FilePath)
		}
		data, rerr := os.ReadFile(absPath)
		if rerr != nil {
			continue
		}
		refs := findBacktickRefs(string(data))
		totalRefs += len(refs)
		missing := checkBacktickRefsExist(cwd, refs)
		for _, m := range missing {
			stales = append(stales, staleRef{taskID: t.TaskID, ref: m})
		}
	}
	if totalRefs == 0 {
		return &ReadinessCheck{
			Check:  "Task body backtick paths",
			Status: "pass",
			Detail: "0 backtick path references",
		}
	}
	stalePct := float64(len(stales)) / float64(totalRefs) * 100.0
	threshold := stalePathPctThreshold()
	if len(stales) == 0 {
		return &ReadinessCheck{
			Check:  "Task body backtick paths",
			Status: "pass",
			Detail: fmt.Sprintf("stale 0 / total %d", totalRefs),
		}
	}
	// List up to 5 items in the detail.
	const maxShow = 5
	var lines []string
	for i, s := range stales {
		if i >= maxShow {
			lines = append(lines, fmt.Sprintf("... +%d more", len(stales)-maxShow))
			break
		}
		lines = append(lines, fmt.Sprintf("%s(%s)", s.taskID, s.ref))
	}
	status := "pass"
	label := "info"
	if stalePct > threshold {
		status = "warn"
		label = "warn"
	}
	_ = label
	return &ReadinessCheck{
		Check:  "Task body backtick paths",
		Status: status,
		Detail: fmt.Sprintf("stale %d / total %d (%.1f%% vs threshold %.1f%%) — %s",
			len(stales), totalRefs, stalePct, threshold, strings.Join(lines, "; ")),
	}
}

// collectStaleBodyCheck flags Tasks whose write date is at least
// GetSprintStaleBodyDays() days older than today. Tasks that pass and
// dates that fail to parse are ignored. CreatedAt format follows the
// Task frontmatter's "YYYY-MM-DD" or RFC3339.
func collectStaleBodyCheck(tasks []domain.TaskSummary) *ReadinessCheck {
	threshold := config.GetSprintStaleBodyDays()
	now := time.Now()
	var stale []string
	for _, t := range tasks {
		created, ok := parseTaskCreatedAt(t.CreatedAt)
		if !ok {
			continue
		}
		gap := int(now.Sub(created).Hours() / hoursPerDay)
		if gap >= threshold {
			stale = append(stale, fmt.Sprintf("%s(%d days)", t.TaskID, gap))
		}
	}
	if len(stale) == 0 {
		return &ReadinessCheck{
			Check:  "Task body write-date gap",
			Status: "pass",
			Detail: fmt.Sprintf("under %d-day threshold", threshold),
		}
	}
	return &ReadinessCheck{
		Check:  "Task body review recommended",
		Status: "warn",
		Detail: fmt.Sprintf("over %d-day threshold: %s — body may be a stale snapshot", threshold, strings.Join(stale, ", ")),
	}
}

// hoursPerDay is the hours -> days conversion constant used by
// collectStaleBodyCheck. Avoids magic numbers in the body.
const hoursPerDay = 24

// parseTaskCreatedAt parses the created_at value from Task
// frontmatter. Allowed formats: RFC3339 ("2026-04-15T00:00:00Z") or
// date-only ("2026-04-15").
func parseTaskCreatedAt(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// runProjectDesignChecks runs the scripts registered under
// sprints.ceremony.design_checks in project-config.yaml sequentially
// and returns the results as a ReadinessCheck list.
// Contract:
// stdin: JSON of the Sprint's Tasks ({"tasks": [{task_id, title,
// type, ...}]}).
// stdout: one check per line, "check_name|status|detail"
// (status in {pass, warn, info, fail}; detail optional; empty
// lines ignored).
// exit code 0: success; non-zero: script failure (shown as warn).
// per-script timeout: designCheckTimeout.
func runProjectDesignChecks(tasks []domain.TaskSummary) []ReadinessCheck {
	cfg, err := config.LoadProjectConfig()
	if err != nil || cfg == nil || cfg.Sprints == nil || cfg.Sprints.Ceremony == nil {
		return nil
	}
	scripts := cfg.Sprints.Ceremony.DesignChecks
	if len(scripts) == 0 {
		return nil
	}

	payload := buildDesignCheckPayload(tasks)
	root := fileutil.GetProjectRoot()

	var results []ReadinessCheck
	for _, scriptPath := range scripts {
		rc := executeDesignCheckScript(root, scriptPath, payload)
		results = append(results, rc...)
	}
	return results
}

// designCheckTimeout is the per-script execution timeout for the
// design_check scripts. Caps long I/O waits.
const designCheckTimeoutSec = 10

// buildDesignCheckPayload builds the JSON payload passed to a
// script's stdin.
func buildDesignCheckPayload(tasks []domain.TaskSummary) []byte {
	type payloadTask struct {
		TaskID   string `json:"task_id"`
		Title    string `json:"title"`
		Type     string `json:"type"`
		Size     string `json:"size"`
		Priority string `json:"priority"`
		Status   string `json:"status"`
		FilePath string `json:"file_path"`
	}
	items := make([]payloadTask, 0, len(tasks))
	for _, t := range tasks {
		filePath := ""
		if raw, err := task.Get(t.TaskID); err == nil {
			if gr, ok := raw.(*domain.GetResult); ok && gr != nil {
				filePath = gr.FilePath
			}
		}
		items = append(items, payloadTask{
			TaskID:   t.TaskID,
			Title:    t.Title,
			Type:     t.Type,
			Size:     t.Estimate,
			Priority: t.Priority,
			Status:   t.Status,
			FilePath: filePath,
		})
	}
	data, _ := json.Marshal(map[string]any{"tasks": items})
	return data
}

// executeDesignCheckScript runs one script and parses its stdout
// into ReadinessCheck items. A script failure is reported as a
// single warn item.
func executeDesignCheckScript(root, scriptPath string, stdin []byte) []ReadinessCheck {
	abs := scriptPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, scriptPath)
	}
	ctx, cancel := context.WithTimeout(context.Background(), designCheckTimeoutSec*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, abs)
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		return []ReadinessCheck{{
			Check:  "design_check: " + scriptPath,
			Status: "warn",
			Detail: "script run failed: " + err.Error(),
		}}
	}
	return parseDesignCheckOutput(scriptPath, string(out))
}

// parseDesignCheckOutput parses each stdout line as
// "check|status|detail". Lines with an invalid format are ignored.
func parseDesignCheckOutput(scriptPath, out string) []ReadinessCheck {
	var results []ReadinessCheck
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		status := strings.TrimSpace(parts[1])
		switch status {
		case "pass", "warn", "info", "fail":
		default:
			continue
		}
		rc := ReadinessCheck{
			Check:  strings.TrimSpace(parts[0]),
			Status: status,
		}
		if len(parts) == 3 {
			rc.Detail = strings.TrimSpace(parts[2])
		}
		results = append(results, rc)
	}
	if len(results) == 0 {
		results = append(results, ReadinessCheck{
			Check:  "design_check: " + scriptPath,
			Status: "pass",
		})
	}
	return results
}

// --------------------------------------------------------------------------
// Sprint Complete Ceremony
// --------------------------------------------------------------------------

// CollectSprintComplete gathers ceremony data at sprint-complete time.
func CollectSprintComplete(sprintID string) *SprintCompleteCeremony {
	cer := &SprintCompleteCeremony{}

	// Harness 10-Phase state
	state, err := harness.HarnessGet("sprint", sprintID)
	if err == nil {
		for _, item := range state.Items {
			status := "pending"
			if item.Done {
				status = "done"
			} else if item.Required {
				status = "blocked"
			}
			cer.Harness = append(cer.Harness, PhaseStatus{
				Phase:  item.ID,
				Status: status,
				Detail: item.Evidence,
			})
		}
	}

	return cer
}

// --------------------------------------------------------------------------
// Task Start Ceremony
// --------------------------------------------------------------------------

// CollectTaskStart gathers ceremony data at task-start time.
func CollectTaskStart(taskID string) *TaskStartCeremony {
	cer := &TaskStartCeremony{}

	// Briefing.
	cer.Briefing = collectTaskBriefing(taskID)

	// Reminders.
	cer.Reminders = config.GetReminders(string(events.EventTaskStart))

	return cer
}

// collectTaskBriefing gathers the Task briefing payload.
func collectTaskBriefing(taskID string) *TaskBriefing {
	briefing := &TaskBriefing{TaskID: taskID}

	raw, err := task.Get(taskID)
	if err != nil {
		return briefing
	}

	gr, ok := raw.(*domain.GetResult)
	if !ok || gr == nil {
		return briefing
	}

	briefing.Title = gr.Title
	briefing.Type = gr.Type
	briefing.Estimate = gr.Estimate
	briefing.Priority = gr.Priority
	if gr.Sprint != nil {
		if s, sOK := gr.Sprint.(string); sOK {
			briefing.Sprint = s
		}
	}
	briefing.DependsOn = gr.DependsOn

	// Commit-prefix mapping.
	prefixMap := map[string]string{
		"feature":  "feat",
		"bugfix":   "fix",
		"hotfix":   "hotfix",
		"docs":     "docs",
		"refactor": "refactor",
		"infra":    "infra",
		"test":     "test",
		"chore":    "chore",
	}
	if p, pOK := prefixMap[gr.Type]; pOK {
		briefing.CommitPrefix = p
	}

	return briefing
}

// --------------------------------------------------------------------------
// Task Complete Ceremony
// --------------------------------------------------------------------------

// CollectTaskComplete gathers ceremony data at task-complete time.
func CollectTaskComplete(taskID string) *TaskCompleteCeremony {
	cer := &TaskCompleteCeremony{}

	// 1. Changed-file list from git diff.
	cer.ChangedFiles = collectChangedFiles()

	// 2. Harness Gate state.
	cer.HarnessGate = collectHarnessGate(taskID)

	// 3. Result-section verification.
	cer.ResultSection = collectResultSection(taskID)

	return cer
}

// collectChangedFiles gathers the changed-file list.
// Previously this used a ceremony-local implementation that only
// inspected the working tree, but the strict verification logic
// (pkg/task.getGitChangedFiles) now unions "working tree + N most
// recent commits", which caused drift. We now reuse
// pkg/task.GetGitChangedFiles to share the same source.
func collectChangedFiles() []string {
	return task.GetGitChangedFiles()
}

// collectHarnessGate gathers the Task's Harness Gate state.
func collectHarnessGate(taskID string) *HarnessGate {
	gate := &HarnessGate{Status: "pass"}

	state, err := harness.HarnessGet("task", taskID)
	if err != nil {
		return gate
	}

	if state.Blocked {
		gate.Status = "blocked"
		for _, item := range state.Items {
			if item.Required && !item.Done {
				gate.UncheckedItems = append(gate.UncheckedItems, item.ID)
			}
		}
	}

	return gate
}

// resultSectionMarker is the markdown heading pattern that
// identifies the result section.
const resultSectionMarker = "## result"

// collectResultSection verifies whether the result section exists in
// the Task file.
func collectResultSection(taskID string) *ResultSectionCheck {
	check := &ResultSectionCheck{Exists: false}

	gs := store.Get()
	if gs == nil {
		return check
	}

	filePath, err := gs.GetTaskFilePath(taskID)
	if err != nil || filePath == "" {
		return check
	}

	absPath, resolveErr := task.ResolveTaskPath(taskID, filePath)
	if resolveErr != nil {
		return check
	}

	root := fileutil.GetProjectRoot()
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, absPath)
	}

	data, readErr := os.ReadFile(absPath)
	if readErr != nil {
		return check
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, resultSectionMarker) {
			check.Exists = true
			break
		}
	}

	return check
}
