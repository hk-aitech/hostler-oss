// Package ports — Gate Judgement Log port.
//
// Background: gate-layer verdicts (PASS / WARN / BLOCK / HARD_BLOCK / SKIP)
// produced by harness check / rules engine / task result verification /
// sprint stamp record only that they "happened" in the audit log; the
// data explaining "why we ruled that way" (input snippet, rule id,
// evidence) is scattered. The doc-review pre-existing 4-FAIL recurrence
// is direct evidence — there is no structure to capture repeated false
// positives.
//
// This port owns the detailed gate-verdict data; audit remains the
// owner of "events" (the two logs are linked through cross-references).
// It honours the port/adapter split — start with JSONL, swap to SQLite
// or remote sinks later if needed.
package ports

// Verdict enumerates gate verdicts. Modelled as string constants so it works
// in both log writes and query filters.
type Verdict string

const (
	VerdictPass      Verdict = "pass"
	VerdictWarn      Verdict = "warn"
	VerdictBlock     Verdict = "block"
	VerdictHardBlock Verdict = "hard_block"
	VerdictSkip      Verdict = "skip"
)

// JudgementRecord captures the full data for a single gate verdict.
// Flat-struct strategy (KB A023) — JSONL query oriented.
type JudgementRecord struct {
	ID              string            `json:"id"`                         // uuid or hash — uniquely identifies the record
	RecordedAt      string            `json:"recorded_at"`                // RFC3339
	GateType        string            `json:"gate_type"`                  // "harness" | "task_result" | "sprint_stamp" | "rules" | "doc_review"
	ItemID          string            `json:"item_id,omitempty"`          // "T###" | "sprint-NN" | "precommit.go.vet" etc.
	Verdict         Verdict           `json:"verdict"`                    // pass/warn/block/hard_block/skip
	RuleID          string            `json:"rule_id,omitempty"`          // e.g. "precommit.manifest.drift"
	RuleSeverity    string            `json:"rule_severity,omitempty"`    // "warn" | "block" | "hard_block"
	InputSnippet    string            `json:"input_snippet,omitempty"`    // excerpt of the inspected content (PII caution — recommend bounding the length)
	Expected        string            `json:"expected,omitempty"`         // expected value (optional)
	Actual          string            `json:"actual,omitempty"`           // observed value (optional)
	SuggestedAction string            `json:"suggested_action,omitempty"` // automatically suggested remediation
	EvidenceRefs    []string          `json:"evidence_refs,omitempty"`    // related file paths / commit SHAs / PR links
	Metadata        map[string]string `json:"metadata,omitempty"`         // caller-specific extensions (task_type, etc.)
}

// FalsePositiveLabel records that a user marked a verdict as a false positive.
// Accumulated labels feed rule-relaxation suggestions.
type FalsePositiveLabel struct {
	RecordID           string `json:"record_id"`
	LabeledBy          string `json:"labeled_by"`
	LabeledAt          string `json:"labeled_at"`
	Reason             string `json:"reason"`                         // recommend 10+ characters of concrete justification
	RuleAdjustmentHint string `json:"rule_adjustment_hint,omitempty"` // e.g. "relax regex / downgrade severity to warn"
}

// JudgementQuery filters Record lookups.
type JudgementQuery struct {
	Since    string  // ISO 8601 lower bound (inclusive)
	Until    string  // ISO 8601 upper bound (inclusive)
	GateType string  // empty string = all
	Verdict  Verdict // empty value = all
	ItemID   string  // empty string = all
	RuleID   string  // empty string = all
	Limit    int     // 0 = default (decided by adapter)
}

// JudgementStats — verdict counts + rule aggregation.
type JudgementStats struct {
	Total        int             `json:"total"`
	VerdictCount map[Verdict]int `json:"verdict_count"` // count per pass/warn/block/...
	RuleTopN     []RuleIDCount   `json:"rule_top_n"`    // top rule_id occurrence counts
	FPRatio      float64         `json:"fp_ratio"`      // labeled FP / Total (0.0~1.0)
	FPByRule     map[string]int  `json:"fp_by_rule"`    // rule_id → FP label count
}

// RuleIDCount — occurrence count per rule_id (a sortable slice element).
type RuleIDCount struct {
	RuleID string `json:"rule_id"`
	Count  int    `json:"count"`
}

// GateJudgementLog — port interface for the gate judgement log.
// Implementations may be backed by JSONL, SQLite, or any remote storage.
type GateJudgementLog interface {
	// Record appends a single verdict. When id is empty the adapter generates one.
	Record(r JudgementRecord) error

	// Query returns records matching filter, newest first.
	Query(filter JudgementQuery) ([]JudgementRecord, error)

	// MarkFalsePositive attaches an FP label to an existing record.
	MarkFalsePositive(recordID string, label FalsePositiveLabel) error

	// Stats aggregates verdicts and rules and returns the FP ratio.
	Stats(filter JudgementQuery) (*JudgementStats, error)
}
