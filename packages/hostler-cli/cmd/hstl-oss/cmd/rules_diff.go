package cmd

// rules_diff.go - `hstl rules diff <before> <after>` subcommand.
//
// rules audit mode emits JSON reports, but comparing before/after a migration
// was manual. This subcommand matches two audit reports by rule_id, shows
// status/severity changes, and returns an exit code based on whether any
// difference exists, providing a CI-friendly diff check.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

const (
	// rulesDiffExitChanged is the exit code returned when the reports differ.
	rulesDiffExitChanged = 1
)

var rulesDiffCmd = &cobra.Command{
	Use:   "diff <before.json> <after.json>",
	Short: "Diff two audit reports - compare Rule results before and after a change",
	Long: `Match two audit reports (output of rules run --audit --report) by rule_id
and display status/severity changes.

Categories:
  added       - rule absent from before, present in after
  removed     - rule present in before, absent from after
  status      - same rule, status changed (ok -> violated etc.)
  severity    - same rule, only severity changed (advisory -> block etc.)
  unchanged   - no change

exit code: 1 when there are differences, 0 when identical (CI-friendly).`,
	Example: fmt.Sprintf(`  %s rules diff /tmp/before.json /tmp/after.json
  %s rules run --phase all --audit --report before.json
  # ... apply changes ...
  %s rules run --phase all --audit --report after.json
  %s rules diff before.json after.json`,
		brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName),
	Args: cobra.ExactArgs(2),
	RunE: runRulesDiff,
}

// rulesDiffEntry is one row of the diff output.
type rulesDiffEntry struct {
	RuleID         string `json:"rule_id"`
	Kind           string `json:"kind"` // added/removed/status/severity/unchanged
	BeforeStatus   string `json:"before_status,omitempty"`
	AfterStatus    string `json:"after_status,omitempty"`
	BeforeSeverity string `json:"before_severity,omitempty"`
	AfterSeverity  string `json:"after_severity,omitempty"`
}

// rulesDiffReport is the final diff payload.
type rulesDiffReport struct {
	Entries []rulesDiffEntry `json:"entries"`
	Summary struct {
		Added     int `json:"added"`
		Removed   int `json:"removed"`
		Status    int `json:"status"`
		Severity  int `json:"severity"`
		Unchanged int `json:"unchanged"`
	} `json:"summary"`
}

func runRulesDiff(cmd *cobra.Command, args []string) error {
	beforePath, afterPath := args[0], args[1]

	before, err := loadAuditReport(beforePath)
	if err != nil {
		return fmt.Errorf("failed to load before report: %w", err)
	}
	after, err := loadAuditReport(afterPath)
	if err != nil {
		return fmt.Errorf("failed to load after report: %w", err)
	}

	diff := computeRulesDiff(before, after)

	if outputFormat == "json" {
		Out.Print(diff)
	} else {
		renderRulesDiffText(cmd.OutOrStderr(), diff)
	}

	if diff.Summary.Added+diff.Summary.Removed+diff.Summary.Status+diff.Summary.Severity > 0 {
		os.Exit(rulesDiffExitChanged)
	}
	return nil
}

// loadAuditReport reads an audit report file.
func loadAuditReport(path string) (*auditReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rep auditReport
	if err := json.Unmarshal(data, &rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

// computeRulesDiff matches the entries of two reports by rule_id and computes
// the diff.
func computeRulesDiff(before, after *auditReport) *rulesDiffReport {
	beforeMap := indexAuditEntries(before)
	afterMap := indexAuditEntries(after)

	rep := &rulesDiffReport{}

	// Walk before -> classify as removed / status / severity / unchanged.
	var ids []string
	for id := range beforeMap {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		b := beforeMap[id]
		a, found := afterMap[id]
		if !found {
			rep.Entries = append(rep.Entries, rulesDiffEntry{
				RuleID:         id,
				Kind:           "removed",
				BeforeStatus:   b.Status,
				BeforeSeverity: b.Severity,
			})
			rep.Summary.Removed++
			continue
		}
		switch {
		case b.Status != a.Status:
			rep.Entries = append(rep.Entries, rulesDiffEntry{
				RuleID:         id,
				Kind:           "status",
				BeforeStatus:   b.Status,
				AfterStatus:    a.Status,
				BeforeSeverity: b.Severity,
				AfterSeverity:  a.Severity,
			})
			rep.Summary.Status++
		case b.Severity != a.Severity:
			rep.Entries = append(rep.Entries, rulesDiffEntry{
				RuleID:         id,
				Kind:           "severity",
				BeforeSeverity: b.Severity,
				AfterSeverity:  a.Severity,
			})
			rep.Summary.Severity++
		default:
			rep.Summary.Unchanged++
		}
	}

	// rules only in after -> added.
	var addedIDs []string
	for id := range afterMap {
		if _, ok := beforeMap[id]; !ok {
			addedIDs = append(addedIDs, id)
		}
	}
	sort.Strings(addedIDs)
	for _, id := range addedIDs {
		a := afterMap[id]
		rep.Entries = append(rep.Entries, rulesDiffEntry{
			RuleID:        id,
			Kind:          "added",
			AfterStatus:   a.Status,
			AfterSeverity: a.Severity,
		})
		rep.Summary.Added++
	}

	return rep
}

// indexAuditEntries converts entries into a map keyed by rule_id.
func indexAuditEntries(rep *auditReport) map[string]auditEntry {
	m := make(map[string]auditEntry, len(rep.Entries))
	for _, e := range rep.Entries {
		m[e.RuleID] = e
	}
	return m
}

// renderRulesDiffText emits a human-readable diff.
func renderRulesDiffText(w interface{ Write(p []byte) (int, error) }, diff *rulesDiffReport) {
	for _, e := range diff.Entries {
		line := ""
		switch e.Kind {
		case "added":
			line = fmt.Sprintf("[+] %-40s (after: %s/%s)\n", e.RuleID, e.AfterStatus, e.AfterSeverity)
		case "removed":
			line = fmt.Sprintf("[-] %-40s (before: %s/%s)\n", e.RuleID, e.BeforeStatus, e.BeforeSeverity)
		case "status":
			line = fmt.Sprintf("[~] %-40s status %s -> %s\n", e.RuleID, e.BeforeStatus, e.AfterStatus)
		case "severity":
			line = fmt.Sprintf("[~] %-40s severity %s -> %s\n", e.RuleID, e.BeforeSeverity, e.AfterSeverity)
		}
		if line != "" {
			_, _ = w.Write([]byte(line))
		}
	}
	summary := fmt.Sprintf("Summary: added=%d removed=%d status=%d severity=%d unchanged=%d\n",
		diff.Summary.Added, diff.Summary.Removed, diff.Summary.Status, diff.Summary.Severity, diff.Summary.Unchanged)
	_, _ = w.Write([]byte(summary))
}

func init() {
	rulesCmd.AddCommand(rulesDiffCmd)
}
