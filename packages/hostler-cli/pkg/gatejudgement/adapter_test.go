// Package gatejudgement — JSONL adapter unit tests.
package gatejudgement

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// newTestStore creates an isolated store for tests.
func newTestStore(t *testing.T) (*JSONLStore, string) {
	t.Helper()
	tmp := t.TempDir()
	recordPath := filepath.Join(tmp, ".hostler", "gate.jsonl")
	fpPath := filepath.Join(tmp, ".hostler", "gate-fp.jsonl")
	return NewJSONLStoreWithPaths(recordPath, fpPath), tmp
}

func TestT781_Record_AutoID(t *testing.T) {
	s, _ := newTestStore(t)

	r := ports.JudgementRecord{
		GateType: "harness",
		ItemID:   "T001",
		Verdict:  ports.VerdictBlock,
		RuleID:   "task.harness",
	}
	if err := s.Record(r); err != nil {
		t.Fatalf("Record: %v", err)
	}
	records, err := s.Query(ports.JudgementQuery{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected record count 1, got %d", len(records))
	}
	if records[0].ID == "" {
		t.Error("auto-generated ID missing")
	}
	if records[0].RecordedAt == "" {
		t.Error("RecordedAt auto-set missing")
	}
}

func TestT781_Query_Filter(t *testing.T) {
	s, _ := newTestStore(t)

	_ = s.Record(ports.JudgementRecord{GateType: "harness", ItemID: "T001", Verdict: ports.VerdictPass, RuleID: "a"})
	_ = s.Record(ports.JudgementRecord{GateType: "harness", ItemID: "T002", Verdict: ports.VerdictBlock, RuleID: "b"})
	_ = s.Record(ports.JudgementRecord{GateType: "rules", ItemID: "precommit.go.vet", Verdict: ports.VerdictWarn, RuleID: "precommit.go.vet"})

	// GateType filter.
	recs, _ := s.Query(ports.JudgementQuery{GateType: "harness"})
	if len(recs) != 2 {
		t.Errorf("expected GateType=harness count 2, got %d", len(recs))
	}

	// Verdict filter.
	recs, _ = s.Query(ports.JudgementQuery{Verdict: ports.VerdictBlock})
	if len(recs) != 1 || recs[0].ItemID != "T002" {
		t.Errorf("Verdict=block filter failed: %v", recs)
	}

	// RuleID filter.
	recs, _ = s.Query(ports.JudgementQuery{RuleID: "precommit.go.vet"})
	if len(recs) != 1 {
		t.Errorf("RuleID filter failed: %d", len(recs))
	}
}

func TestT781_MarkFP_And_Stats(t *testing.T) {
	s, _ := newTestStore(t)

	// Record 3 entries.
	_ = s.Record(ports.JudgementRecord{GateType: "rules", ItemID: "a", Verdict: ports.VerdictBlock, RuleID: "precommit.go.vet"})
	_ = s.Record(ports.JudgementRecord{GateType: "rules", ItemID: "b", Verdict: ports.VerdictWarn, RuleID: "precommit.go.vet"})
	_ = s.Record(ports.JudgementRecord{GateType: "rules", ItemID: "c", Verdict: ports.VerdictPass, RuleID: "precommit.go.staticcheck"})

	recs, _ := s.Query(ports.JudgementQuery{})
	if len(recs) != 3 {
		t.Fatalf("expected 3, got %d", len(recs))
	}

	// 1 FP label.
	if err := s.MarkFalsePositive(recs[0].ID, ports.FalsePositiveLabel{Reason: "pre-existing noise"}); err != nil {
		t.Fatalf("MarkFalsePositive: %v", err)
	}

	// Stats.
	stats, err := s.Stats(ports.JudgementQuery{})
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.Total != 3 {
		t.Errorf("expected Total 3, got %d", stats.Total)
	}
	if stats.VerdictCount[ports.VerdictBlock] != 1 {
		t.Errorf("expected block count 1, got %d", stats.VerdictCount[ports.VerdictBlock])
	}
	if stats.VerdictCount[ports.VerdictWarn] != 1 {
		t.Errorf("expected warn count 1, got %d", stats.VerdictCount[ports.VerdictWarn])
	}
	if len(stats.RuleTopN) != 2 {
		t.Errorf("expected rule top-N 2, got %d", len(stats.RuleTopN))
	}
	// Top rule is precommit.go.vet (2 entries).
	if stats.RuleTopN[0].RuleID != "precommit.go.vet" || stats.RuleTopN[0].Count != 2 {
		t.Errorf("expected top rule go.vet(2), got %v", stats.RuleTopN[0])
	}
	// FP ratio.
	wantRatio := 1.0 / 3.0
	if stats.FPRatio < wantRatio-0.01 || stats.FPRatio > wantRatio+0.01 {
		t.Errorf("expected FP ratio ~0.333, got %f", stats.FPRatio)
	}
}

func TestT781_MarkFP_ShortReason_REJECT(t *testing.T) {
	// At the adapter level only an empty reason is rejected (the 10-character requirement is enforced at the CLI).
	s, _ := newTestStore(t)
	_ = s.Record(ports.JudgementRecord{GateType: "harness", ItemID: "T001", Verdict: ports.VerdictBlock})
	recs, _ := s.Query(ports.JudgementQuery{})
	if err := s.MarkFalsePositive(recs[0].ID, ports.FalsePositiveLabel{Reason: ""}); err == nil {
		t.Error("empty reason must be rejected")
	}
	if err := s.MarkFalsePositive("", ports.FalsePositiveLabel{Reason: "valid reason"}); err == nil {
		t.Error("empty recordID must be rejected")
	}
}

func TestT781_Query_EmptyFile_NoError(t *testing.T) {
	s, _ := newTestStore(t)
	// Query without creating the file.
	recs, err := s.Query(ports.JudgementQuery{})
	if err != nil {
		t.Errorf("missing file should not be an error: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("expected empty result, got %d", len(recs))
	}
}

func TestT781_InputSnippet_Truncation(t *testing.T) {
	s, _ := newTestStore(t)

	longInput := strings.Repeat("x", 2000)
	_ = s.Record(ports.JudgementRecord{
		GateType:     "harness",
		Verdict:      ports.VerdictBlock,
		InputSnippet: longInput,
	})
	recs, _ := s.Query(ports.JudgementQuery{})
	if len(recs) != 1 {
		t.Fatal("no record")
	}
	if !strings.HasSuffix(recs[0].InputSnippet, "...[truncated]") {
		t.Errorf("snippet truncation missing: len=%d", len(recs[0].InputSnippet))
	}
	if len(recs[0].InputSnippet) > DefaultInputSnippetMaxLen+20 {
		t.Errorf("snippet too long: %d", len(recs[0].InputSnippet))
	}
}

func TestT781_GlobalStore_Disabled(t *testing.T) {
	ResetGlobalStore()
	defer ResetGlobalStore()

	s, _ := newTestStore(t)
	SetGlobalStore(s)

	SetDisabled(true)
	defer SetDisabled(false)

	RecordJudgement(ports.JudgementRecord{
		GateType: "harness",
		Verdict:  ports.VerdictBlock,
	})
	recs, _ := s.Query(ports.JudgementQuery{})
	if len(recs) != 0 {
		t.Errorf("must not record while disabled: %d", len(recs))
	}
}

func TestT781_GlobalStore_SamplingPass_Skip(t *testing.T) {
	ResetGlobalStore()
	defer ResetGlobalStore()

	s, _ := newTestStore(t)
	SetGlobalStore(s)

	// PASS sampling rate 0 -> all skipped.
	os.Setenv(PassSampleRateEnvVar, "0")
	defer os.Unsetenv(PassSampleRateEnvVar)

	for i := 0; i < 50; i++ {
		RecordJudgement(ports.JudgementRecord{
			GateType: "harness",
			Verdict:  ports.VerdictPass,
		})
	}
	recs, _ := s.Query(ports.JudgementQuery{})
	if len(recs) != 0 {
		t.Errorf("PASS sampling 0 should skip all, got %d", len(recs))
	}
}

func TestT781_GlobalStore_WarnBlock_AlwaysRecorded(t *testing.T) {
	ResetGlobalStore()
	defer ResetGlobalStore()

	s, _ := newTestStore(t)
	SetGlobalStore(s)

	// WARN/BLOCK/HARD_BLOCK ignore sampling and are always recorded.
	os.Setenv(PassSampleRateEnvVar, "0")
	defer os.Unsetenv(PassSampleRateEnvVar)

	verdicts := []ports.Verdict{
		ports.VerdictWarn, ports.VerdictBlock, ports.VerdictHardBlock,
	}
	for _, v := range verdicts {
		for i := 0; i < 3; i++ {
			RecordJudgement(ports.JudgementRecord{
				GateType: "harness",
				Verdict:  v,
			})
		}
	}
	recs, _ := s.Query(ports.JudgementQuery{})
	if len(recs) != 9 {
		t.Errorf("expected WARN/BLOCK/HARD_BLOCK total 9, got %d", len(recs))
	}
}

func TestT781_PortInterface_ComplianceMarker(t *testing.T) {
	// Compile-time marker — adapter.go contains `var _ ports.GateJudgementLog = (*JSONLStore)(nil)`.
	var _ ports.GateJudgementLog = &JSONLStore{}
}

// Test that, even when called from a subdirectory with empty projectRoot,
// the JSONL adapter creates .hstl-oss/gate-judgement.jsonl based on
// HSTL_PROJECT_ROOT (the project root) rather than the CWD (preventing fragmentation).
func TestISS20260419_005_NewJSONLStore_ProjectRootFromSubdir(t *testing.T) {
	projectRoot := t.TempDir()
	subdir := filepath.Join(projectRoot, "docker")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	t.Setenv("HSTL_PROJECT_ROOT", projectRoot)
	t.Chdir(subdir) // simulate invocation from a subdirectory

	// Disable the HSTL_GATE_LOG_PATH override for this test (we are testing the lookup path).
	t.Setenv(GateLogPathEnvVar, "")

	s := NewJSONLStore("")
	if err := s.Record(ports.JudgementRecord{
		GateType: "harness",
		ItemID:   "ISS-REGRESSION",
		Verdict:  ports.VerdictBlock,
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	expected := filepath.Join(projectRoot, ".hstl-oss", "gate-judgement.jsonl")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("not created under the project root .hstl-oss/: %v", err)
	}
	forbidden := filepath.Join(subdir, ".hstl-oss", "gate-judgement.jsonl")
	if _, err := os.Stat(forbidden); err == nil {
		t.Errorf("erroneously created under the subdirectory .hstl-oss/ — fragmentation regressed: %s", forbidden)
	}
}

// Same guarantee for the global singleton — second path of the same regression.
func TestISS20260419_005_GlobalStore_ProjectRootFromSubdir(t *testing.T) {
	ResetGlobalStore()
	defer ResetGlobalStore()

	projectRoot := t.TempDir()
	subdir := filepath.Join(projectRoot, "deep", "nested")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	t.Setenv("HSTL_PROJECT_ROOT", projectRoot)
	t.Chdir(subdir)
	t.Setenv(GateLogPathEnvVar, "")

	SetDisabled(false)
	os.Setenv(PassSampleRateEnvVar, "1.0") // record everything
	defer os.Unsetenv(PassSampleRateEnvVar)

	RecordJudgement(ports.JudgementRecord{
		GateType: "harness",
		ItemID:   "ISS-REGRESSION-GLOBAL",
		Verdict:  ports.VerdictBlock,
	})

	expected := filepath.Join(projectRoot, ".hstl-oss", "gate-judgement.jsonl")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("GlobalStore must also use the project root: %v", err)
	}
	forbidden := filepath.Join(subdir, ".hstl-oss", "gate-judgement.jsonl")
	if _, err := os.Stat(forbidden); err == nil {
		t.Errorf("GlobalStore wrongly created under the subdirectory .hstl-oss/: %s", forbidden)
	}
}
