package rules

import (
	"time"
)

// G1: register the 16 task-type harness items from harness_defaults.json as
// observability-only entries in the Rule Registry.
//
// The G1 / D01 inventory found that the 16 task-type harness items defined
// only in JSON were never exposed to the Rule Engine Registry. This file
// wraps each item as a `harness.task.<item>` Rule so that audit reports and
// `hstl rules list` can see them. Actual enforcement still flows through the
// existing harness.HarnessCheck path.
//
// The 10 Sprint-phase items (phase1~phase10) belong to G2 and are not
// included here. context_acknowledged was added to the auto-check
// allowlist; this file simply provides additional Registry exposure (no
// duplicate registration — different ID prefix).

// ── constants ────────────────────────────────────────────────────────

// harnessTaskItems lists the item IDs used by each task type in
// harness_defaults.json.
// jq -r '.[] | .[] | .id' cli/pkg/db/schemas/harness_defaults.json | sort -u
// after excluding the phase* (Sprint phase) entries.
var harnessTaskItems = []struct {
	itemID      string
	description string
}{
	{"build_passed", "build passes"},
	{"change_record", "change record (infra)"},
	{"code_review", "code-review performed"},
	{"context_acknowledged", "context call ack"},
	{"coverage_checked", "test coverage checked"},
	{"criteria_checked", "all done-criteria checkboxes ticked"},
	{"deploy_verified", "Dev deployment verified (infra)"},
	{"deprecation_cleanup_gate", "deprecation cleanup gate (archive residue check)"},
	{"doc_review", "doc review (docs)"},
	{"evaluation_scope", "evaluation scope defined (spike)"},
	{"findings_documented", "findings documented (spike)"},
	{"lint_passed", "lint passes (go vet + staticcheck)"},
	{"reproduction", "reproduction steps recorded (bugfix/hotfix)"},
	{"rollback_plan", "rollback plan (hotfix/infra)"},
	{"root_cause", "root cause analysis (bugfix/hotfix)"},
	{"tests_passed", "tests pass"},
}

// harnessRuleCategory is the Category used by Rules in this file.
const harnessRuleCategory = "harness"

// ── harnessWrapperRule — common implementation for the 16 Rules ──────

type harnessWrapperRule struct {
	itemID      string
	description string
}

func (r harnessWrapperRule) ID() string                { return "harness.task." + r.itemID }
func (r harnessWrapperRule) Category() string          { return harnessRuleCategory }
func (r harnessWrapperRule) Description() string       { return r.description }
func (r harnessWrapperRule) DefaultSeverity() Severity { return SeverityAdvisory }

func (r harnessWrapperRule) Check(ctx *RuleContext) *RuleResult {
	start := time.Now()
	res := &RuleResult{RuleID: r.ID(), Severity: r.DefaultSeverity()}
	// observability-only — actual enforcement goes through harness.HarnessCheck.
	// This Rule guarantees Registry exposure only and always returns Skipped
	// (it acts as a "harness-item-definition inventory" in audit reports).
	res.Status = StatusSkipped
	res.Duration = time.Since(start)
	return res
}

func init() {
	for _, item := range harnessTaskItems {
		Register(harnessWrapperRule{itemID: item.itemID, description: item.description})
	}
}
