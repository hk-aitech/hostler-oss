package gatejudgement_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/gatejudgement"
)

// SQLite adapter unit tests.

func newTestSQLiteStore(t *testing.T) *gatejudgement.SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "gj-test.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sqlite open failed: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	s, err := gatejudgement.NewSQLiteStore(conn)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	return s
}

func TestT877_SQLiteStore_RecordAndQuery(t *testing.T) {
	s := newTestSQLiteStore(t)

	now := time.Now().UTC().Format(time.RFC3339)
	rec := ports.JudgementRecord{
		RecordedAt:   now,
		GateType:     "harness",
		ItemID:       "T872",
		Verdict:      ports.VerdictPass,
		RuleID:       "criteria_checked",
		InputSnippet: "5 completion criteria satisfied",
		EvidenceRefs: []string{"works/tasks/T872.md"},
		Metadata:     map[string]string{"sprint": "sprint-104"},
	}
	if err := s.Record(rec); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	out, err := s.Query(ports.JudgementQuery{GateType: "harness"})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 Query result, got %d", len(out))
	}
	got := out[0]
	if got.ItemID != "T872" || got.Verdict != ports.VerdictPass {
		t.Errorf("Query field mismatch: %+v", got)
	}
	if len(got.EvidenceRefs) != 1 || got.EvidenceRefs[0] != "works/tasks/T872.md" {
		t.Errorf("EvidenceRefs restore failed: %v", got.EvidenceRefs)
	}
	if got.Metadata["sprint"] != "sprint-104" {
		t.Errorf("Metadata restore failed: %v", got.Metadata)
	}
	if got.ID == "" {
		t.Error("auto-generated ID is empty")
	}
}

func TestT877_SQLiteStore_QueryFilters(t *testing.T) {
	s := newTestSQLiteStore(t)

	now := time.Now().UTC().Format(time.RFC3339)
	for i, v := range []ports.Verdict{ports.VerdictPass, ports.VerdictWarn, ports.VerdictBlock} {
		if err := s.Record(ports.JudgementRecord{
			RecordedAt: now,
			GateType:   "harness",
			ItemID:     "T100",
			Verdict:    v,
			RuleID:     "r" + string(rune('1'+i)),
		}); err != nil {
			t.Fatalf("Record failed #%d: %v", i, err)
		}
	}

	warnOnly, err := s.Query(ports.JudgementQuery{Verdict: ports.VerdictWarn})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(warnOnly) != 1 || warnOnly[0].Verdict != ports.VerdictWarn {
		t.Errorf("verdict filter failed: %+v", warnOnly)
	}
}

func TestT877_SQLiteStore_FalsePositiveLabelAndStats(t *testing.T) {
	s := newTestSQLiteStore(t)

	rec := ports.JudgementRecord{
		ID:         "gj_test_001",
		RecordedAt: time.Now().UTC().Format(time.RFC3339),
		GateType:   "rules",
		Verdict:    ports.VerdictBlock,
		RuleID:     "precommit.manifest.drift",
	}
	if err := s.Record(rec); err != nil {
		t.Fatalf("Record failed: %v", err)
	}
	if err := s.MarkFalsePositive("gj_test_001", ports.FalsePositiveLabel{
		Reason: "user-intended drift — whitelist",
	}); err != nil {
		t.Fatalf("MarkFalsePositive failed: %v", err)
	}

	stats, err := s.Stats(ports.JudgementQuery{})
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.Total != 1 {
		t.Errorf("expected Total 1, got %d", stats.Total)
	}
	if stats.VerdictCount[ports.VerdictBlock] != 1 {
		t.Errorf("expected block count 1, got %d", stats.VerdictCount[ports.VerdictBlock])
	}
	if stats.FPRatio <= 0 {
		t.Errorf("expected FPRatio > 0, got %v", stats.FPRatio)
	}
}

func TestT877_NewStore_DefaultIsJSONL(t *testing.T) {
	t.Setenv(gatejudgement.StoreEnvVar, "")
	root := t.TempDir()
	s := gatejudgement.NewStore(root)
	if _, ok := s.(*gatejudgement.JSONLStore); !ok {
		t.Errorf("expected default JSONL, got type %T", s)
	}
}
