package sprint

// Auto-generated default template for the Sprint Dev-verification Task body.
// Helper invoked when `hstl sprint create --auto-verify` is on. Creates a
// Dev-verification Task and injects a generic checklist body into the
// file. The Sprint's implementation Task list may still be empty at
// sprint-create time, so only generic items are injected (design
// decision).

import (
	"fmt"
	"os"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

// ── Constants ──
const (
	autoVerifyTaskType     = "infra"
	autoVerifyTaskPriority = "p1"
	autoVerifyTaskEstimate = "S"

	autoVerifyBodyStartHeading = "## Type Tags"
	autoVerifyBodyEndHeading   = "## Scope Limits"
	autoVerifyBodyFallbackEnd  = "## References"
)

// autoVerifyTaskTitleTemplate is the title format for the Sprint
// release-verification Task.
const autoVerifyTaskTitleTemplate = "%s Dev environment release verification"

// autoVerifyBodyTemplate is the body inserted between ## Type Tags and
// just before ## Scope Limits. Includes a leading newline so it follows
// the "## Type Tags" block naturally.
// Argument 1: Sprint ID. Argument 2: Sprint goal supplemental line
// (pass "" when empty).
const autoVerifyBodyTemplate = `
## Purpose

Integration-verify the implementation results of the %s Sprint in the
Dev environment.%s

## Requirements

- [ ] Per-Task manual smoke test — directly check 1~2 core features of
  each Task in this Sprint.
- [ ] Regression test — confirm no existing functionality is missing.
- [ ] ` + "`cd cli && go test -count=1 ./...`" + ` — full PASS.
- [ ] ` + "`cd cli && make install`" + ` — succeeds + binary version verified.
- [ ] ` + "`bash scripts/integration-test.sh`" + ` — passes.
- [ ] manifest / harness / heading lint drift = 0.

## Done Criteria

- [ ] All 6 verification items above PASS.
- [ ] Bugs / issues found during Dev verification are immediately
  registered as hotfix Tasks.

`

// AppendAutoVerifyTask registers a Dev-verification Task for the given
// sprintID and injects the body template. Returns the task.Create
// result as-is. File-write failures are wrapped as warnings while the
// Task creation itself is preserved as success (body-template injection
// is best-effort).
func AppendAutoVerifyTask(sprintID string) (*domain.CreateResult, error) {
	title := fmt.Sprintf(autoVerifyTaskTitleTemplate, sprintID)
	raw, err := task.Create(title, autoVerifyTaskType, sprintID, autoVerifyTaskPriority, autoVerifyTaskEstimate, "" /*summary=auto-generated*/, nil)
	if err != nil {
		return nil, fmt.Errorf("auto-verify Task creation failed: %w", err)
	}
	result, ok := raw.(*domain.CreateResult)
	if !ok || result == nil {
		return nil, fmt.Errorf("auto-verify Task result type error")
	}

	// Look up the Sprint goal so it can be included in the body.
	// Failure preserves Task creation success (best-effort).
	goal := ""
	if rec, gErr := GetFromDB(sprintID); gErr == nil && rec != nil {
		goal = rec.Goal
	}

	if writeErr := InjectAutoVerifyBody(result.FilePath, sprintID, goal); writeErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("body-template injection failed (file remains a placeholder): %v", writeErr))
	}
	return result, nil
}

// autoVerifyTaskTitleSuffix is the suffix used to identify the Dev
// verification Task by title. Must match the format produced by
// AppendAutoVerifyTask.
const autoVerifyTaskTitleSuffix = "Dev environment release verification"

// refreshDevVerifyTaskBody is invoked at sprint-start time and finds
// the in-Sprint Dev verification Task (whose title ends with
// "... Dev environment release verification"), regenerating its body
// with the Sprint's actual implementation Task list. No-op when no
// Dev-verification Task exists. Body-regeneration failures are
// returned as errors but the call site treats them as warn-level.
func refreshDevVerifyTaskBody(sprintID string) error {
	// Look up the Task list within this Sprint.
	sprintPtr := sprintID
	raw, err := task.List(&sprintPtr, nil)
	if err != nil {
		return err
	}
	lr, ok := raw.(*domain.ListResult)
	if !ok || lr == nil {
		return nil
	}

	// Separate the Dev-verification Task from the implementation Tasks.
	var devVerify *domain.TaskSummary
	var implTasks []domain.TaskSummary
	for i := range lr.Tasks {
		t := lr.Tasks[i]
		if strings.HasSuffix(t.Title, autoVerifyTaskTitleSuffix) {
			devVerify = &t
		} else {
			implTasks = append(implTasks, t)
		}
	}
	if devVerify == nil {
		return nil // No auto-verify Task — no-op.
	}

	// Look up the Task file path.
	getRaw, err := task.Get(devVerify.TaskID)
	if err != nil {
		return err
	}
	gr, ok := getRaw.(*domain.GetResult)
	if !ok || gr == nil || gr.FilePath == "" {
		return nil
	}
	absPath := gr.FilePath
	if !strings.HasPrefix(absPath, "/") {
		absPath = fileutil.GetProjectRoot() + "/" + absPath
	}

	// Look up the Sprint goal so it can be included in the body.
	goal := ""
	if rec, gErr := GetFromDB(sprintID); gErr == nil && rec != nil {
		goal = rec.Goal
	}

	return InjectAutoVerifyBodyWithTasks(absPath, sprintID, goal, implTasks)
}

// InjectAutoVerifyBodyWithTasks is the extended form of
// InjectAutoVerifyBody. It lists the Sprint's implementation Tasks as
// checkboxes inside the generated ## Requirements section. When
// implTasks is empty, behaves identically to InjectAutoVerifyBody and
// only injects the generic template.
func InjectAutoVerifyBodyWithTasks(filePath, sprintID, goal string, implTasks []domain.TaskSummary) error {
	if len(implTasks) == 0 {
		return InjectAutoVerifyBody(filePath, sprintID, goal)
	}
	if filePath == "" {
		return fmt.Errorf("empty file path")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(data)

	startIdx := strings.Index(content, autoVerifyBodyStartHeading)
	if startIdx < 0 {
		return fmt.Errorf("%q section missing", autoVerifyBodyStartHeading)
	}
	after := content[startIdx+len(autoVerifyBodyStartHeading):]
	relNext := strings.Index(after, "\n## ")
	if relNext < 0 {
		return fmt.Errorf("end of ## Type Tags block not found")
	}
	bodyStart := startIdx + len(autoVerifyBodyStartHeading) + relNext

	endIdx := strings.Index(content[bodyStart:], "\n"+autoVerifyBodyEndHeading)
	if endIdx < 0 {
		endIdx = strings.Index(content[bodyStart:], "\n"+autoVerifyBodyFallbackEnd)
	}
	if endIdx < 0 {
		return fmt.Errorf("end anchor (%q or %q) missing", autoVerifyBodyEndHeading, autoVerifyBodyFallbackEnd)
	}
	endAbs := bodyStart + endIdx

	// Task list rendering. Each row identifies the originating Task
	// (`T### — title`) — manual smoke test instructions live in the
	// Dev verification Task's surrounding Done Criteria copy, not on
	// every row, to keep the per-Task line stable for diffing.
	var taskLines []string
	for _, t := range implTasks {
		taskLines = append(taskLines, fmt.Sprintf("- [ ] %s — %s", t.TaskID, t.Title))
	}
	taskSection := strings.Join(taskLines, "\n")

	// When goal is set, append a "Sprint goal: ..." line in the
	// ## Purpose section.
	goalSuffix := ""
	if goal != "" {
		goalSuffix = "\n\nSprint goal: " + goal
	}

	body := fmt.Sprintf(`
## Purpose

Integration-verify the implementation results of the %s Sprint in the
Dev environment.%s

## Requirements

%s
- [ ] Regression test — confirm no existing functionality is missing.
- [ ] `+"`go test -count=1 ./...`"+` — full PASS.
- [ ] `+"`make install`"+` — succeeds + binary version verified.
- [ ] manifest / harness / heading lint drift = 0.

## Done Criteria

- [ ] Every verification item above passes.
- [ ] Bugs / issues found during Dev verification are immediately
  registered as hotfix Tasks.

`, sprintID, goalSuffix, taskSection)

	replaced := content[:bodyStart] + body + content[endAbs+1:]
	return os.WriteFile(filePath, []byte(replaced), 0o644)
}

// InjectAutoVerifyBody replaces the segment between the ## Type Tags
// block and just before ## Scope Limits in the generated Task file
// with a generic verification template. Returns an error if either
// anchor is missing.
func InjectAutoVerifyBody(filePath, sprintID, goal string) error {
	if filePath == "" {
		return fmt.Errorf("empty file path")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(data)

	// Find the end of the ## Type Tags block: the first "## " heading
	// after the blank line that follows ## Type Tags.
	startIdx := strings.Index(content, autoVerifyBodyStartHeading)
	if startIdx < 0 {
		return fmt.Errorf("%q section missing", autoVerifyBodyStartHeading)
	}
	// Skip past the ## Type Tags block content and locate the next
	// "## ".
	after := content[startIdx+len(autoVerifyBodyStartHeading):]
	relNext := strings.Index(after, "\n## ")
	if relNext < 0 {
		return fmt.Errorf("end of ## Type Tags block not found")
	}
	bodyStart := startIdx + len(autoVerifyBodyStartHeading) + relNext

	// End anchor: prefer ## Scope Limits, fall back to ## References.
	endIdx := strings.Index(content[bodyStart:], "\n"+autoVerifyBodyEndHeading)
	if endIdx < 0 {
		endIdx = strings.Index(content[bodyStart:], "\n"+autoVerifyBodyFallbackEnd)
	}
	if endIdx < 0 {
		return fmt.Errorf("end anchor (%q or %q) missing", autoVerifyBodyEndHeading, autoVerifyBodyFallbackEnd)
	}
	endAbs := bodyStart + endIdx

	// When goal is set, append a "Sprint goal: ..." line in the
	// ## Purpose section.
	goalSuffix := ""
	if goal != "" {
		goalSuffix = "\n\nSprint goal: " + goal
	}
	replaced := content[:bodyStart] + fmt.Sprintf(autoVerifyBodyTemplate, sprintID, goalSuffix) + content[endAbs+1:]
	return os.WriteFile(filePath, []byte(replaced), 0o644)
}
