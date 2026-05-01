// Package domain — request DTOs and options for the Task service.
//
// ADR-001 §9 E2 — promoted the Request / Options types from pkg/task into
// internal/domain.
package domain

import "time"

// UpdateRequest — input for the task_update tool.
//
// Every field is a pointer: nil=no change, non-nil=change requested.
// An empty string / empty slice means "clear the value" (e.g. drop dependencies).
//
// Immutable fields (cannot be updated):
//   - id, created_at, file_path
//
// Sprint field caveats:
//   - Sprint: directly updates the DB sprint column (no file move) — used by
//     the drift-repair path. The standard flow that also moves the file is
//     task_assign_sprint / task_unassign_sprint.
//
// The status field is an escape hatch — use only for exceptional situations
// such as DB recovery or manual repair. Abnormal transitions are allowed
// with a WARN log.
type UpdateRequest struct {
	TaskID    string    // required
	Title     *string   // new title
	Type      *string   // feature/bugfix/hotfix/refactor/infra/docs/test/chore
	Priority  *string   // p0/p1/p2/p3
	Estimate  *string   // XS/S/M/L/XL
	Status    *string   // todo/in-progress/done/absorbed — escape hatch
	Sprint    *string   // direct DB sprint edit (no file move) — drift-repair path
	DependsOn *[]string // nil=no change, []=clear dependencies, [...]=replace
}

// HasUpdates reports whether at least one field has a requested update.
func (r *UpdateRequest) HasUpdates() bool {
	return r.Title != nil || r.Type != nil || r.Priority != nil || r.Estimate != nil ||
		r.Status != nil || r.Sprint != nil || r.DependsOn != nil
}

// DeleteOptions controls optional Delete behaviour.
type DeleteOptions struct {
	// OrphanFile — when true, drift Tasks (no DB row, only a file) are also
	// processed by the delete path. Default false keeps the legacy behaviour
	// (returns NotFoundError).
	OrphanFile bool
}

// SubagentOptions — options for InvokeSummaryJudge (preserves the pkg/task API).
type SubagentOptions struct {
	ProjectRoot string        // falls back to cwd or HSTL_PROJECT_ROOT when empty
	Model       string        // uses the default when empty
	Timeout     time.Duration // 0 uses the default timeout
	Disabled    bool          // when true, returns ok immediately (for tests)
}

// SummaryJudgeStatsOptions — options for AggregateSummaryJudgeLog.
type SummaryJudgeStatsOptions struct {
	Window int    // aggregate only the most recent N entries (0 uses the default window)
	Path   string // log file path (uses the default path when empty)
}

// HeuristicPreset — strictness level for summary validation.
type HeuristicPreset string

const (
	HeuristicPresetOff      HeuristicPreset = "off"
	HeuristicPresetLenient  HeuristicPreset = "lenient"
	HeuristicPresetModerate HeuristicPreset = "moderate"
	HeuristicPresetStrict   HeuristicPreset = "strict"
)

// HeuristicConfig — heuristic rule-chain settings (loaded from a preset).
type HeuristicConfig struct {
	Preset                   HeuristicPreset `yaml:"preset"`
	MinUniqueWords           int             `yaml:"min_unique_words"`
	ForbiddenSoloKeywords    []string        `yaml:"forbidden_solo_keywords"`
	RequiredPurposeConnector []string        `yaml:"required_purpose_connectors"`
	EnableLLMJudge           bool            `yaml:"enable_llm_judge"`
}
