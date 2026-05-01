// Package gatejudgement — SQLite adapter.
//
// Implements ports.GateJudgementLog with the same interface as JSONLStore.
// Callers receive the store via factory injection, so swapping in this
// implementation does not affect callers.
//
// Tables:
// - gate_judgement_records : append-only JudgementRecord storage
// - gate_judgement_fp_labels : append-only FalsePositiveLabel storage
//
// Advantages over JSONL:
// - WHERE-clause filtering avoids full scans
// - shares the DB with audit_events → unified observability
// - eliminates git history bloat (preventing recurrence of the
// 200KB+ JSONL log issues observed in the originating monorepo)
package gatejudgement

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// SQLiteStore is the SQLite-backed GateJudgementLog implementation.
type SQLiteStore struct {
	db *sql.DB
	mu sync.Mutex
}

// NewSQLiteStore creates a store from the given *sql.DB and idempotently
// applies the schema. Typically pkg/db.GetDB() is injected.
func NewSQLiteStore(db *sql.DB) (*SQLiteStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	s := &SQLiteStore{db: db}
	if err := s.ensureSchema(); err != nil {
		return nil, fmt.Errorf("failed to apply gate-judgement schema: %w", err)
	}
	return s, nil
}

// Compile-time check that the port interface is satisfied.
var _ ports.GateJudgementLog = (*SQLiteStore)(nil)

const gateJudgementSchemaDDL = `
CREATE TABLE IF NOT EXISTS gate_judgement_records (
 id TEXT PRIMARY KEY,
 recorded_at TEXT NOT NULL,
 gate_type TEXT NOT NULL,
 item_id TEXT NOT NULL DEFAULT '',
 verdict TEXT NOT NULL,
 rule_id TEXT NOT NULL DEFAULT '',
 rule_severity TEXT NOT NULL DEFAULT '',
 input_snippet TEXT NOT NULL DEFAULT '',
 expected TEXT NOT NULL DEFAULT '',
 actual TEXT NOT NULL DEFAULT '',
 suggested_action TEXT NOT NULL DEFAULT '',
 evidence_refs TEXT NOT NULL DEFAULT '[]',
 metadata TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_gate_judgement_records_recorded_at
 ON gate_judgement_records (recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_gate_judgement_records_gate_verdict
 ON gate_judgement_records (gate_type, verdict);
CREATE INDEX IF NOT EXISTS idx_gate_judgement_records_rule_id
 ON gate_judgement_records (rule_id);

CREATE TABLE IF NOT EXISTS gate_judgement_fp_labels (
 record_id TEXT NOT NULL,
 labeled_by TEXT NOT NULL DEFAULT '',
 labeled_at TEXT NOT NULL,
 reason TEXT NOT NULL,
 rule_adjustment_hint TEXT NOT NULL DEFAULT '',
 PRIMARY KEY (record_id, labeled_at)
);
`

func (s *SQLiteStore) ensureSchema() error {
	_, err := s.db.Exec(gateJudgementSchemaDDL)
	return err
}

// Record appends a judgement. If ID is empty, it is randomly generated; if
// RecordedAt is empty, UTC now is used.
func (s *SQLiteStore) Record(r ports.JudgementRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.ID == "" {
		id, err := generateRecordID()
		if err != nil {
			return fmt.Errorf("failed to generate record ID: %w", err)
		}
		r.ID = id
	}
	if r.RecordedAt == "" {
		r.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	}
	r.InputSnippet = truncateSnippet(r.InputSnippet)

	evRefs, _ := json.Marshal(r.EvidenceRefs)
	if len(evRefs) == 0 {
		evRefs = []byte("[]")
	}
	meta, _ := json.Marshal(r.Metadata)
	if len(meta) == 0 {
		meta = []byte("{}")
	}

	_, err := s.db.Exec(`
INSERT INTO gate_judgement_records
 (id, recorded_at, gate_type, item_id, verdict, rule_id, rule_severity,
 input_snippet, expected, actual, suggested_action, evidence_refs, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, r.ID, r.RecordedAt, r.GateType, r.ItemID, string(r.Verdict), r.RuleID,
		r.RuleSeverity, r.InputSnippet, r.Expected, r.Actual,
		r.SuggestedAction, string(evRefs), string(meta))
	if err != nil {
		return fmt.Errorf("gate_judgement_records INSERT failed: %w", err)
	}
	return nil
}

// Query returns records matching the filter (newest first).
func (s *SQLiteStore) Query(filter ports.JudgementQuery) ([]ports.JudgementRecord, error) {
	where, args := buildWhereClause(filter)
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultQueryLimit
	}
	q := `SELECT id, recorded_at, gate_type, item_id, verdict, rule_id, rule_severity,
		input_snippet, expected, actual, suggested_action, evidence_refs, metadata
		FROM gate_judgement_records ` + where + `
		ORDER BY recorded_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("gate_judgement_records SELECT failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []ports.JudgementRecord
	for rows.Next() {
		var r ports.JudgementRecord
		var verdictStr, evRefsJSON, metaJSON string
		if err := rows.Scan(&r.ID, &r.RecordedAt, &r.GateType, &r.ItemID,
			&verdictStr, &r.RuleID, &r.RuleSeverity, &r.InputSnippet,
			&r.Expected, &r.Actual, &r.SuggestedAction,
			&evRefsJSON, &metaJSON); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		r.Verdict = ports.Verdict(verdictStr)
		_ = json.Unmarshal([]byte(evRefsJSON), &r.EvidenceRefs)
		_ = json.Unmarshal([]byte(metaJSON), &r.Metadata)
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkFalsePositive appends an FP label.
func (s *SQLiteStore) MarkFalsePositive(recordID string, label ports.FalsePositiveLabel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if recordID == "" {
		return fmt.Errorf("recordID is required")
	}
	if label.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	if label.LabeledAt == "" {
		label.LabeledAt = time.Now().UTC().Format(time.RFC3339)
	}
	if label.LabeledBy == "" {
		label.LabeledBy = "user"
	}

	_, err := s.db.Exec(`
INSERT INTO gate_judgement_fp_labels
 (record_id, labeled_by, labeled_at, reason, rule_adjustment_hint)
VALUES (?, ?, ?, ?, ?)
`, recordID, label.LabeledBy, label.LabeledAt, label.Reason, label.RuleAdjustmentHint)
	if err != nil {
		return fmt.Errorf("gate_judgement_fp_labels INSERT failed: %w", err)
	}
	return nil
}

// Stats reports verdict counts, rule top-N, and FP ratio.
func (s *SQLiteStore) Stats(filter ports.JudgementQuery) (*ports.JudgementStats, error) {
	where, args := buildWhereClause(filter)

	stats := &ports.JudgementStats{
		VerdictCount: map[ports.Verdict]int{},
		FPByRule: map[string]int{},
	}

	// verdict count + total
	rows, err := s.db.Query(`SELECT verdict, COUNT(*) FROM gate_judgement_records `+where+` GROUP BY verdict`, args...)
	if err != nil {
		return nil, fmt.Errorf("verdict aggregation failed: %w", err)
	}
	for rows.Next() {
		var v string
		var cnt int
		if err := rows.Scan(&v, &cnt); err != nil {
			_ = rows.Close()
			return nil, err
		}
		stats.VerdictCount[ports.Verdict(v)] = cnt
		stats.Total += cnt
	}
	_ = rows.Close()

	// rule top-N
	ruleRows, err := s.db.Query(`
SELECT rule_id, COUNT(*) FROM gate_judgement_records `+where+`
 AND rule_id != '' GROUP BY rule_id`, args...)
	if err != nil {
		return nil, fmt.Errorf("rule_id aggregation failed: %w", err)
	}
	for ruleRows.Next() {
		var id string
		var cnt int
		if err := ruleRows.Scan(&id, &cnt); err != nil {
			_ = ruleRows.Close()
			return nil, err
		}
		stats.RuleTopN = append(stats.RuleTopN, ports.RuleIDCount{RuleID: id, Count: cnt})
	}
	_ = ruleRows.Close()
	sort.Slice(stats.RuleTopN, func(i, j int) bool {
		return stats.RuleTopN[i].Count > stats.RuleTopN[j].Count
	})

	// FP label aggregation — record IDs limited to the filter scope.
	fpRows, err := s.db.Query(`
SELECT l.record_id, r.rule_id FROM gate_judgement_fp_labels l
JOIN gate_judgement_records r ON r.id = l.record_id
`+strings.ReplaceAll(where, "WHERE", "WHERE")+``, args...)
	if err != nil {
		return nil, fmt.Errorf("fp aggregation failed: %w", err)
	}
	fpCount := 0
	for fpRows.Next() {
		var recID, ruleID string
		if err := fpRows.Scan(&recID, &ruleID); err != nil {
			_ = fpRows.Close()
			return nil, err
		}
		fpCount++
		if ruleID != "" {
			stats.FPByRule[ruleID]++
		}
	}
	_ = fpRows.Close()
	if stats.Total > 0 {
		stats.FPRatio = float64(fpCount) / float64(stats.Total)
	}
	return stats, nil
}

// buildWhereClause translates a JudgementQuery into a SQLite WHERE clause.
func buildWhereClause(f ports.JudgementQuery) (string, []any) {
	var clauses []string
	var args []any
	if f.Since != "" {
		clauses = append(clauses, "recorded_at >= ?")
		args = append(args, f.Since)
	}
	if f.Until != "" {
		clauses = append(clauses, "recorded_at <= ?")
		args = append(args, f.Until)
	}
	if f.GateType != "" {
		clauses = append(clauses, "gate_type = ?")
		args = append(args, f.GateType)
	}
	if f.Verdict != "" {
		clauses = append(clauses, "verdict = ?")
		args = append(args, string(f.Verdict))
	}
	if f.ItemID != "" {
		clauses = append(clauses, "item_id = ?")
		args = append(args, f.ItemID)
	}
	if f.RuleID != "" {
		clauses = append(clauses, "rule_id = ?")
		args = append(args, f.RuleID)
	}
	if len(clauses) == 0 {
		return "WHERE 1=1", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
