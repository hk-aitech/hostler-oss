package rules

import (
	"time"
)

// G2: expose the 10-step Sprint Phase entries in the Rule Registry.
//
// Inventory G2 / D03 noted that the Sprint Phase 10 steps were defined in
// three places (harness_defaults.json, defaultCeremonyComplete in
// pkg/sprint/sprint.go, and commands/sprint/complete.md). This file wraps
// each phase as a `sprint.phase.<step>` Rule to expose them via audit
// reports and `hstl rules list`.
//
// SSOT consolidation (removing defaultCeremonyComplete and centralising on
// a JSON-loading wrapper) is handled in the D03 follow-up. This task only
// adds Registry exposure.

// sprintPhaseItems defines the 10 steps (kept in sync with
// harness_defaults.json sprint_ceremony).
var sprintPhaseItems = []struct {
	itemID      string
	description string
}{
	{"phase1_doc_review", "Phase 1 — doc-review --sprint"},
	{"phase2_code_review", "Phase 2 — code-review --sprint"},
	{"phase3_work_audit", "Phase 3 — work-audit (implementation + workflow)"},
	{"phase4_cross_check", "Phase 4 — doc-cross-check (conditional)"},
	{"phase5_retro", "Phase 5 — retro (KPT)"},
	{"phase6_learned", "Phase 6 — learned (KB)"},
	{"phase7_dev_deploy", "Phase 7 — Dev environment deployment verification"},
	{"phase8_build", "Phase 8 — build / test / install"},
	{"phase9_followup_tasks", "Phase 9 — derive follow-up Tasks"},
	{"phase10_skill_update", "Phase 10 — skill / docs update audit"},
}

const sprintPhaseRuleCategory = "sprint"

type sprintPhaseWrapperRule struct {
	itemID      string
	description string
}

func (r sprintPhaseWrapperRule) ID() string                { return "sprint.phase." + r.itemID }
func (r sprintPhaseWrapperRule) Category() string          { return sprintPhaseRuleCategory }
func (r sprintPhaseWrapperRule) Description() string       { return r.description }
func (r sprintPhaseWrapperRule) DefaultSeverity() Severity { return SeverityAdvisory }

func (r sprintPhaseWrapperRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}
	// observability-only — actual enforcement runs in the sprint complete cascade.
	res.Status = StatusSkipped
	res.Duration = time.Since(start)
	return res
}

func init() {
	for _, item := range sprintPhaseItems {
		Register(sprintPhaseWrapperRule{itemID: item.itemID, description: item.description})
	}
}

// RegisterSprintPhase is the public entry point that lets external plugins
// add extra entries to the Cascade Phase list. It must be called after the
// init() that registers the built-in 10 phases (or it can register an
// unrelated independent item).
//
// itemID       : a unique identifier such as "phase11_custom_verify"
//                (the phaseN prefix is recommended).
// description  : human-readable description.
//
// Registered entries are visible via `hstl rules list` and appear in audit
// reports. Actual enforcement requires a separate update to
// harness_defaults.json plus the sprint:complete cascade logic — this
// function only handles Registry exposure.
func RegisterSprintPhase(itemID, description string) {
	Register(sprintPhaseWrapperRule{itemID: itemID, description: description})
}
