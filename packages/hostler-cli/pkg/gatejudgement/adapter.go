// Package gatejudgement — GateJudgementLog JSONL adapter.
//
// JSONL append store + full-scan-based query (MVP). When future query
// efficiency is exhausted, swap in a SQLite adapter — callers are
// unaffected as long as the port interface is preserved.
//
// Default file path: .hstl-oss/gate-judgement.jsonl (relative to projectRoot).
// Override via the HSTL_GATE_LOG_PATH environment variable.
//
// FP labels: appended to a separate file .hstl-oss/gate-judgement-fp.jsonl
// (the original record is untouched, preserving the log-structured rule).
package gatejudgement

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// JSONLStore is the JSONL append adapter.
type JSONLStore struct {
	recordPath string // .hstl-oss/gate-judgement.jsonl
	fpPath     string // .hstl-oss/gate-judgement-fp.jsonl
	mu         sync.Mutex
}

const (
	// fileMode / dirMode — JSONL file permissions.
	gateLogFileMode = 0o644
	gateLogDirMode  = 0o755

	// GateLogPathEnvVar — env var that overrides the record file path.
	GateLogPathEnvVar = "HSTL_GATE_LOG_PATH"

	// DefaultQueryLimit — default value for Query filter.Limit.
	DefaultQueryLimit = 1000

	// DefaultInputSnippetMaxLen — default max length for InputSnippet (privacy/size guard).
	DefaultInputSnippetMaxLen = 500

	// InputSnippetMaxLenEnvVar — override for the max length.
	InputSnippetMaxLenEnvVar = "HSTL_GATE_LOG_SNIPPET_MAX"

	// recordIDBytes — number of random bytes used for the auto-generated ID (hex doubles it).
	recordIDBytes = 8

	// defaultGateLogRelPath — default relative path.
	defaultGateLogRelPath = brand.ProjectDirName + "/gate-judgement.jsonl"
)

// NewJSONLStore creates a JSONL adapter rooted at projectRoot. When
// projectRoot is empty, fileutil.GetProjectRoot() is consulted
// (HSTL_PROJECT_ROOT -> git toplevel -> cwd). A plain cwd fallback would
// create a fresh state directory in subdirectories like docker/, fragmenting
// the data.
func NewJSONLStore(projectRoot string) *JSONLStore {
	root := projectRoot
	if root == "" {
		root = fileutil.GetProjectRoot()
	}
	recordPath := resolvePath(root, defaultGateLogRelPath, "GATE_LOG_PATH")
	fpPath := filepath.Join(filepath.Dir(recordPath), "gate-judgement-fp.jsonl")
	return &JSONLStore{
		recordPath: recordPath,
		fpPath:     fpPath,
	}
}

// NewJSONLStoreWithPaths is for tests — paths injected directly.
func NewJSONLStoreWithPaths(recordPath, fpPath string) *JSONLStore {
	return &JSONLStore{recordPath: recordPath, fpPath: fpPath}
}

// Compile-time marker — confirms port interface compliance.
var _ ports.GateJudgementLog = (*JSONLStore)(nil)

// resolvePath prefers env, otherwise resolves relative to root.
// envKey is the prefix-less suffix used by envalias.Lookup.
func resolvePath(root, relDefault, envKey string) string {
	if v := envalias.Lookup(envKey); v != "" {
		return v
	}
	return filepath.Join(root, relDefault)
}

// Record appends a judgement. ID is generated when empty. RecordedAt
// defaults to UTC now when empty.
func (s *JSONLStore) Record(r ports.JudgementRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.ID == "" {
		id, err := generateRecordID()
		if err != nil {
			return fmt.Errorf("generate record ID: %w", err)
		}
		r.ID = id
	}
	if r.RecordedAt == "" {
		r.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	}
	r.InputSnippet = truncateSnippet(r.InputSnippet)

	if err := os.MkdirAll(filepath.Dir(s.recordPath), gateLogDirMode); err != nil {
		return fmt.Errorf("create gate-log directory: %w", err)
	}
	return appendJSONLine(s.recordPath, r)
}

// Query scans by filter. Reads the entire JSONL, applies the filter,
// sorts newest-first, then applies the limit.
func (s *JSONLStore) Query(filter ports.JudgementQuery) ([]ports.JudgementRecord, error) {
	records, err := readAllRecords(s.recordPath)
	if err != nil {
		return nil, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultQueryLimit
	}

	filtered := make([]ports.JudgementRecord, 0, len(records))
	for _, r := range records {
		if !matchFilter(r, filter) {
			continue
		}
		filtered = append(filtered, r)
	}

	// Newest-first sort (string compare on RecordedAt — RFC3339 means lexical == temporal).
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].RecordedAt > filtered[j].RecordedAt
	})

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

// MarkFalsePositive appends an FP label (log-structured — the original record is untouched).
func (s *JSONLStore) MarkFalsePositive(recordID string, label ports.FalsePositiveLabel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if recordID == "" {
		return fmt.Errorf("recordID required")
	}
	if label.Reason == "" {
		return fmt.Errorf("reason required")
	}
	label.RecordID = recordID
	if label.LabeledAt == "" {
		label.LabeledAt = time.Now().UTC().Format(time.RFC3339)
	}
	if label.LabeledBy == "" {
		label.LabeledBy = "user"
	}

	if err := os.MkdirAll(filepath.Dir(s.fpPath), gateLogDirMode); err != nil {
		return fmt.Errorf("create fp-log directory: %w", err)
	}
	return appendJSONLine(s.fpPath, label)
}

// Stats returns verdict counts + rule top-N + FP ratio.
func (s *JSONLStore) Stats(filter ports.JudgementQuery) (*ports.JudgementStats, error) {
	records, err := readAllRecords(s.recordPath)
	if err != nil {
		return nil, err
	}

	stats := &ports.JudgementStats{
		VerdictCount: map[ports.Verdict]int{},
		FPByRule:     map[string]int{},
	}
	ruleCount := map[string]int{}
	filteredIDs := map[string]bool{}
	filteredRuleByID := map[string]string{}
	for _, r := range records {
		if !matchFilter(r, filter) {
			continue
		}
		stats.Total++
		stats.VerdictCount[r.Verdict]++
		if r.RuleID != "" {
			ruleCount[r.RuleID]++
			filteredRuleByID[r.ID] = r.RuleID
		}
		filteredIDs[r.ID] = true
	}

	// rule top-N (return all; the caller trims for display).
	stats.RuleTopN = make([]ports.RuleIDCount, 0, len(ruleCount))
	for id, cnt := range ruleCount {
		stats.RuleTopN = append(stats.RuleTopN, ports.RuleIDCount{RuleID: id, Count: cnt})
	}
	sort.Slice(stats.RuleTopN, func(i, j int) bool {
		return stats.RuleTopN[i].Count > stats.RuleTopN[j].Count
	})

	// Aggregate FP labels — only record IDs within the filter range.
	fpLabels, _ := readAllFPLabels(s.fpPath) // ignore missing file
	fpCount := 0
	for _, lbl := range fpLabels {
		if !filteredIDs[lbl.RecordID] {
			continue
		}
		fpCount++
		if rid, ok := filteredRuleByID[lbl.RecordID]; ok {
			stats.FPByRule[rid]++
		}
	}
	if stats.Total > 0 {
		stats.FPRatio = float64(fpCount) / float64(stats.Total)
	}
	return stats, nil
}

// -- internal helpers --

// matchFilter reports whether the record satisfies the filter conditions.
func matchFilter(r ports.JudgementRecord, f ports.JudgementQuery) bool {
	if f.Since != "" && r.RecordedAt < f.Since {
		return false
	}
	if f.Until != "" && r.RecordedAt > f.Until {
		return false
	}
	if f.GateType != "" && r.GateType != f.GateType {
		return false
	}
	if f.Verdict != "" && r.Verdict != f.Verdict {
		return false
	}
	if f.ItemID != "" && r.ItemID != f.ItemID {
		return false
	}
	if f.RuleID != "" && r.RuleID != f.RuleID {
		return false
	}
	return true
}

// generateRecordID returns a random hex ID.
func generateRecordID() (string, error) {
	b := make([]byte, recordIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "gj_" + hex.EncodeToString(b), nil
}

// truncateSnippet truncates to the env-var or default max length.
func truncateSnippet(s string) string {
	if s == "" {
		return s
	}
	maxLen := DefaultInputSnippetMaxLen
	if v := envalias.Lookup("GATE_LOG_SNIPPET_MAX"); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			maxLen = n
		}
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}

// appendJSONLine appends one JSONL line (race safety guaranteed by the caller's mu).
func appendJSONLine(path string, v any) error {
	line, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("JSON marshal: %w", err)
	}
	line = append(line, '\n')
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, gateLogFileMode)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// readAllRecords parses every JSONL line. Returns an empty slice when the file is missing (no error).
func readAllRecords(path string) ([]ports.JudgementRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var records []ports.JudgementRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1MB/line
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r ports.JudgementRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue // skip damaged lines (observability-only)
		}
		records = append(records, r)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return records, nil
}

// readAllFPLabels parses every FP JSONL line. Returns an empty slice when the file is missing.
func readAllFPLabels(path string) ([]ports.FalsePositiveLabel, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var labels []ports.FalsePositiveLabel
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 512*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var lbl ports.FalsePositiveLabel
		if err := json.Unmarshal([]byte(line), &lbl); err != nil {
			continue
		}
		labels = append(labels, lbl)
	}
	return labels, sc.Err()
}
