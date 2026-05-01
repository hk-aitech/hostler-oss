// Package cmd - gate-log CLI subcommands.
//
// hstl gate-log record/query/stats/mark-fp - query and label gate judgement
// logs. An observability tool that surfaces accumulated false-positive
// patterns and provides the data backing rule-relaxation suggestions.
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

var gateLogCmd = &cobra.Command{
	Use:   "gate-log",
	Short: "Query and label gate judgement logs (observability)",
	Long: `Accumulate gate-layer judgement results from harness / rules /
task_result / sprint_stamp / etc. into JSONL logs, label false positives, and
aggregate statistics.

Subcommands:
  record     - record one entry from stdin JSON (bash bridge)
  query      - filter-based lookup
  stats      - verdict counts + rule top-N + FP ratio
  mark-fp    - attach a false-positive label to an existing record`,
}

// ── record (from stdin) ────────────────────────────────────────────

var gateLogRecordFromStdin bool

var gateLogRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record one judgement from stdin JSON (bash bridge)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !gateLogRecordFromStdin {
			return fmt.Errorf("--from-stdin is required (currently only the bash bridge mode is supported)")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		var r ports.JudgementRecord
		if err := json.Unmarshal(data, &r); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		if err := app.GateRecordDirect(r); err != nil {
			return fmt.Errorf("Record failed: %w", err)
		}
		_, _ = fmt.Fprintln(os.Stdout, "recorded")
		return nil
	},
}

// ── query ──────────────────────────────────────────────────────────

var (
	gateLogQuerySince    string
	gateLogQueryUntil    string
	gateLogQueryGateType string
	gateLogQueryVerdict  string
	gateLogQueryItemID   string
	gateLogQueryRuleID   string
	gateLogQueryLimit    int
	gateLogQueryOutput   string
)

var gateLogQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Filter-based judgement log lookup",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := ports.JudgementQuery{
			Since:    gateLogQuerySince,
			Until:    gateLogQueryUntil,
			GateType: gateLogQueryGateType,
			Verdict:  ports.Verdict(gateLogQueryVerdict),
			ItemID:   gateLogQueryItemID,
			RuleID:   gateLogQueryRuleID,
			Limit:    gateLogQueryLimit,
		}
		records, err := app.GateQuery(filter)
		if err != nil {
			return err
		}
		return outputJSONOrText(records, gateLogQueryOutput)
	},
}

// ── stats ──────────────────────────────────────────────────────────

var (
	gateLogStatsSince    string
	gateLogStatsGateType string
	gateLogStatsOutput   string
)

var gateLogStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "verdict counts + rule top-N + FP ratio",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := ports.JudgementQuery{
			Since:    gateLogStatsSince,
			GateType: gateLogStatsGateType,
		}
		stats, err := app.GateStats(filter)
		if err != nil {
			return err
		}
		return outputJSONOrText(stats, gateLogStatsOutput)
	},
}

// ── mark-fp ────────────────────────────────────────────────────────

var (
	gateLogMarkFPReason    string
	gateLogMarkFPRuleHint  string
	gateLogMarkFPLabeledBy string
)

var gateLogMarkFPCmd = &cobra.Command{
	Use:   "mark-fp <record-id>",
	Short: "Attach a false-positive label to a record",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recordID := args[0]
		if len(strings.TrimSpace(gateLogMarkFPReason)) < 10 {
			return fmt.Errorf("--reason is required (at least 10 characters)")
		}
		label := ports.FalsePositiveLabel{
			RecordID:           recordID,
			LabeledBy:          gateLogMarkFPLabeledBy,
			Reason:             gateLogMarkFPReason,
			RuleAdjustmentHint: gateLogMarkFPRuleHint,
		}
		if err := app.GateMarkFalsePositive(recordID, label); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "labeled: %s\n", recordID)
		return nil
	},
}

// ── shared output helper ───────────────────────────────────────────

func outputJSONOrText(v any, mode string) error {
	if mode == "json" || mode == "" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	// Text mode is caller-defined - MVP only emits JSON.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func init() {
	gateLogRecordCmd.Flags().BoolVar(&gateLogRecordFromStdin, "from-stdin", false, "Read JSON from stdin")

	gateLogQueryCmd.Flags().StringVar(&gateLogQuerySince, "since", "", "ISO 8601 (inclusive)")
	gateLogQueryCmd.Flags().StringVar(&gateLogQueryUntil, "until", "", "ISO 8601 (inclusive)")
	gateLogQueryCmd.Flags().StringVar(&gateLogQueryGateType, "gate-type", "", "harness|rules|task_result|sprint_stamp|doc_review")
	gateLogQueryCmd.Flags().StringVar(&gateLogQueryVerdict, "verdict", "", "pass|warn|block|hard_block|skip")
	gateLogQueryCmd.Flags().StringVar(&gateLogQueryItemID, "item-id", "", "entity/item id")
	gateLogQueryCmd.Flags().StringVar(&gateLogQueryRuleID, "rule-id", "", "rule id")
	gateLogQueryCmd.Flags().IntVar(&gateLogQueryLimit, "limit", 0, "Maximum number of rows")
	gateLogQueryCmd.Flags().StringVarP(&gateLogQueryOutput, "output", "o", "json", "Output format (json)")

	gateLogStatsCmd.Flags().StringVar(&gateLogStatsSince, "since", "", "ISO 8601")
	gateLogStatsCmd.Flags().StringVar(&gateLogStatsGateType, "gate-type", "", "harness|rules|...")
	gateLogStatsCmd.Flags().StringVarP(&gateLogStatsOutput, "output", "o", "json", "Output format")

	gateLogMarkFPCmd.Flags().StringVar(&gateLogMarkFPReason, "reason", "", "Reason for the label (at least 10 chars)")
	gateLogMarkFPCmd.Flags().StringVar(&gateLogMarkFPRuleHint, "rule-hint", "", "Rule relaxation suggestion")
	gateLogMarkFPCmd.Flags().StringVar(&gateLogMarkFPLabeledBy, "labeled-by", "user", "Labeling actor")
	_ = gateLogMarkFPCmd.MarkFlagRequired("reason")

	gateLogCmd.AddCommand(gateLogRecordCmd, gateLogQueryCmd, gateLogStatsCmd, gateLogMarkFPCmd)
	rootCmd.AddCommand(gateLogCmd)
}
