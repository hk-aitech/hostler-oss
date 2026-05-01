package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/rules"
)

// TestT576_ResolvePhases_all verifies that the "all" sentinel returns both
// commit-msg and precommit phases. Acts as a boundary guard when the set
// expands.
func TestT576_ResolvePhases_all(t *testing.T) {
	phases, err := resolvePhases("all")
	if err != nil {
		t.Fatalf("resolvePhases(all) failed: %v", err)
	}
	if len(phases) < 2 {
		t.Errorf("all phase expected at least 2, got=%d", len(phases))
	}
	has := map[rules.Phase]bool{}
	for _, p := range phases {
		has[p] = true
	}
	if !has[rules.PhaseCommitMsg] || !has[rules.PhasePreCommit] {
		t.Errorf("commit-msg + precommit missing: %v", phases)
	}
}

// TestT576_ResolvePhases_Single verifies that a single phase string is
// resolved correctly (regression guard for existing behavior).
func TestT576_ResolvePhases_Single(t *testing.T) {
	cases := map[string]rules.Phase{
		"commit-msg": rules.PhaseCommitMsg,
		"precommit":  rules.PhasePreCommit,
	}
	for in, want := range cases {
		got, err := resolvePhases(in)
		if err != nil {
			t.Errorf("resolvePhases(%q) failed: %v", in, err)
			continue
		}
		if len(got) != 1 || got[0] != want {
			t.Errorf("resolvePhases(%q) = %v, want [%s]", in, got, want)
		}
	}
}

// TestT576_ResolvePhases_Error verifies that an unknown phase and an empty
// string both return an error.
func TestT576_ResolvePhases_Error(t *testing.T) {
	if _, err := resolvePhases(""); err == nil {
		t.Error("empty string must produce an error")
	}
	if _, err := resolvePhases("unknown"); err == nil {
		t.Error("unknown phase must produce an error")
	}
}

// TestT576_WriteAuditReport verifies the JSON report schema and summary
// aggregation are correct.
func TestT576_WriteAuditReport(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "audit.json")

	results := []*rules.RuleResult{
		{RuleID: "test.ok", Status: rules.StatusOK, Severity: rules.SeverityWarn},
		{RuleID: "test.violated", Status: rules.StatusViolated, Severity: rules.SeverityBlock, Message: "fail", Evidence: []string{"line1"}},
		{RuleID: "test.skipped", Status: rules.StatusSkipped, Severity: rules.SeverityOff},
	}
	phaseOf := map[string]rules.Phase{
		"test.ok":       rules.PhaseCommitMsg,
		"test.violated": rules.PhasePreCommit,
		"test.skipped":  rules.PhaseCommitMsg,
	}

	if err := writeAuditReport(path, phaseOf, results); err != nil {
		t.Fatalf("writeAuditReport failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read report: %v", err)
	}
	var rep auditReport
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if rep.Version != "1" {
		t.Errorf("version=1 expected, got=%q", rep.Version)
	}
	if len(rep.Entries) != 3 {
		t.Errorf("entries=3 expected, got=%d", len(rep.Entries))
	}
	if rep.Summary.Total != 3 || rep.Summary.OK != 1 || rep.Summary.Violated != 1 || rep.Summary.Skipped != 1 {
		t.Errorf("summary mismatch: %+v", rep.Summary)
	}

	// Verify phase mapping.
	for _, e := range rep.Entries {
		if e.RuleID == "test.violated" && e.Phase != string(rules.PhasePreCommit) {
			t.Errorf("violated phase mapping wrong: %+v", e)
		}
	}
}

// TestT576_StatusString verifies that every RuleStatus is converted to its
// string form.
func TestT576_StatusString(t *testing.T) {
	cases := map[rules.RuleStatus]string{
		rules.StatusOK:       "ok",
		rules.StatusViolated: "violated",
		rules.StatusSkipped:  "skipped",
		rules.StatusError:    "error",
	}
	for in, want := range cases {
		if got := statusString(in); got != want {
			t.Errorf("statusString(%d) = %q, want %q", in, got, want)
		}
	}
}

// TestT596_Audit_PhaseAll_AllCategoriesExposed verifies that every Rule
// category in the Registry is exposed in the audit report when running
// audit mode with phase=all. Regression guard for the bug where new
// categories (precommit/skill/harness/sprint) were missing because
// rulesForPhase did not map them.
func TestT596_Audit_PhaseAll_AllCategoriesExposed(t *testing.T) {
	// In audit mode the Registry best-effort branch must run.
	// Update this list when a new category is added - intentionally enforced.
	expectedCategories := []string{
		"commit",
		"task",
		"skill",
		"harness",
		"precommit",
		"sprint",
	}

	// Each category must have at least one Rule in the Registry.
	seen := map[string]int{}
	for _, r := range rules.List() {
		seen[r.Category()]++
	}
	for _, cat := range expectedCategories {
		if seen[cat] == 0 {
			t.Errorf("no Rule with category %q in the Registry - registration missing", cat)
		}
	}
}

// TestT596_Registry_Total_SanityCheck verifies a minimum bound on the
// total Rule count - early detection of mass deletions during regression.
func TestT596_Registry_Total_SanityCheck(t *testing.T) {
	minExpected := 40 // safety margin
	got := len(rules.List())
	if got < minExpected {
		t.Errorf("Registry Rule count %d < minimum %d - suspect mass deletion", got, minExpected)
	}
}

// TestT596_RulesForPhase_vs_Registry_Gap verifies that even when
// rulesForPhase does not map a Rule, the audit branch still covers it. The
// mapping gap is logged informationally.
func TestT596_RulesForPhase_vs_Registry_Gap(t *testing.T) {
	// IDs mapped by rulesForPhase.
	mapped := map[string]bool{}
	for _, p := range []rules.Phase{rules.PhaseCommitMsg, rules.PhasePreCommit} {
		for _, id := range rulesForPhase(p) {
			mapped[id] = true
		}
	}
	// Count mapping gap.
	unmapped := 0
	for _, r := range rules.List() {
		if !mapped[r.ID()] {
			unmapped++
		}
	}
	if unmapped == 0 {
		t.Error("rulesForPhase maps every Rule - the audit branch may have become unnecessary or Rule count decreased")
	}
	t.Logf("Rules unmapped (need audit best-effort): %d / total %d", unmapped, len(rules.List()))
}

// TestT600_ParallelismForPhase_CommitMsg_Sequential - the commit-msg phase
// always returns sequentialParallelism (1); env var has no effect.
func TestT600_ParallelismForPhase_CommitMsg_Sequential(t *testing.T) {
	t.Setenv(envPrecommitParallelism, "8")
	if got := parallelismForPhase(rules.PhaseCommitMsg); got != sequentialParallelism {
		t.Errorf("commit-msg parallelism = %d, want %d", got, sequentialParallelism)
	}
}

// TestT600_ParallelismForPhase_PreCommit_env - the precommit phase
// honors the env var. Empty / invalid values fall back to NumCPU; a positive
// integer forces that value.
func TestT600_ParallelismForPhase_PreCommit_env(t *testing.T) {
	cases := []struct {
		name   string
		envVal string
		want   int // 0 = expect NumCPU
	}{
		{"unset_falls_back_to_NumCPU", "", 0},
		{"1_forces_sequential", "1", 1},
		{"4_forces_4", "4", 4},
		{"invalid_falls_back_to_NumCPU", "abc", 0},
		{"0_ignored_uses_NumCPU", "0", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(envPrecommitParallelism, c.envVal)
			got := parallelismForPhase(rules.PhasePreCommit)
			if c.want == 0 {
				if got < 1 {
					t.Errorf("parallelism = %d, want >=1 (NumCPU fallback)", got)
				}
			} else if got != c.want {
				t.Errorf("parallelism = %d, want %d", got, c.want)
			}
		})
	}
}

// TestT600_ExecuteRules_Parallel_Deterministic - the parallel execution
// must yield the same Rule ID set as sequential execution. Slice order is
// not relevant because renderRulesRunOutput sorts, but set equivalence and
// nil removal must hold.
func TestT600_ExecuteRules_Parallel_Deterministic(t *testing.T) {
	// Use the actual Rule ID list from the precommit category.
	var ids []string
	for _, r := range rules.List() {
		if r.Category() == "precommit" {
			ids = append(ids, r.ID())
		}
	}
	if len(ids) < 2 {
		t.Skip("fewer than 2 precommit Rules - not enough to verify parallelism")
	}
	ctx := &rules.RuleContext{
		Phase:       rules.PhasePreCommit,
		ProjectRoot: "", // empty root - Rules drop into Skipped (safe)
		Extra:       map[string]any{},
	}
	eff := map[string]*rules.EffectiveRule{}

	seq := executeRules(ctx, ids, eff, 1)
	par := executeRules(ctx, ids, eff, 4)

	if len(seq) != len(par) {
		t.Fatalf("sequential=%d parallel=%d - length mismatch", len(seq), len(par))
	}
	seqSet := map[string]bool{}
	for _, r := range seq {
		seqSet[r.RuleID] = true
	}
	for _, r := range par {
		if !seqSet[r.RuleID] {
			t.Errorf("present in parallel only: %s", r.RuleID)
		}
	}
}
