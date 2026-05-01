package cmd

// rules_exec.go — T316 (Sprint-23): `hstl rules exec <rule-id>` runs a
// single Rule.
//
// The existing `hstl rules run --phase <phase>` runs Rules in phase
// batches. This command runs a single rule-id directly so embedded-script
// Rules (precommit.branch.sync, precommit.version.sync, etc.) can be
// invoked from the CLI.
//
// Use cases:
//   - replace skill-document guidance like
//     `bash scripts/check-branch-sync.sh` with this command
//   - fix the exit-127 problem in external projects where the embedded
//     script does not exist on disk
//   - debug a single Rule — verify just one rule without running the
//     whole phase
//
// exit code convention:
//   0 — Rule Status=OK or Skipped
//   3 — Rule Status=Violated (block / hard-block)
//   2 — rule-id is not registered

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/rules"
)

var rulesExecCmd = &cobra.Command{
	Use:   "exec <rule-id>",
	Short: "Run a single Rule (replaces the bash scripts/... guidance in skill docs)",
	Long: `Run a single Rule with a minimal RuleContext. Primarily targets
embedded-script Rules (precommit.branch.sync, precommit.version.sync, etc.).

The existing 'rules run --phase <phase>' runs the entire phase — this
command runs one rule.

exit code:
  0 — Rule Status=OK or Skipped
  3 — Rule Status=Violated
  2 — rule-id not registered`,
	Example: fmt.Sprintf(`  %s rules exec precommit.branch.sync
  %s rules exec precommit.version.sync
  %s rules exec precommit.manifest.drift`,
		brand.ShortName, brand.ShortName, brand.ShortName),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruleID := args[0]
		raw, ok := app.RulesGet(ruleID)
		if !ok {
			Out.Error(
				fmt.Sprintf("rule %q not registered", ruleID),
				apperr.CategoryNotFound.String(),
				fmt.Sprintf("run '%s rules list' to see registered rules", brand.ShortName),
			)
			os.Exit(2)
		}
		r, ok := raw.(rules.Rule)
		if !ok {
			Out.Error("RulesGet returned an unexpected type", "", "")
			os.Exit(exitError)
		}
		// app.ProjectRoot() auto-detects — even when cwd is a subdirectory
		// (packages/hostler-cli, etc.) it returns the project root, so
		// the .git check inside the Rule passes.
		root := app.ProjectRoot()
		if root == "" {
			root, _ = os.Getwd()
		}
		ctx := &rules.RuleContext{ProjectRoot: root}
		res := r.Check(ctx)

		switch res.Status {
		case rules.StatusOK:
			Out.Success(fmt.Sprintf("[%s] PASS", ruleID), map[string]any{
				"rule_id":  ruleID,
				"status":   "pass",
				"duration": res.Duration.String(),
			})
			return nil
		case rules.StatusSkipped:
			fmt.Fprintf(output.Stderr(), "[%s] SKIPPED (shouldRun=false or precondition unmet)\n", ruleID)
			Out.Success(fmt.Sprintf("[%s] SKIPPED", ruleID), map[string]any{
				"rule_id": ruleID,
				"status":  "skip",
			})
			return nil
		case rules.StatusViolated:
			fmt.Fprintf(output.Stderr(), "[%s] VIOLATED: %s\n", ruleID, res.Message)
			for _, e := range res.Evidence {
				fmt.Fprintf(output.Stderr(), "  %s\n", e)
			}
			Out.Error(
				fmt.Sprintf("[%s] %s", ruleID, res.Message),
				apperr.CategoryBlocked.String(),
				"Inspect the Rule evidence, fix the root cause, and rerun",
			)
			os.Exit(exitBlocked)
		case rules.StatusError:
			fmt.Fprintf(output.Stderr(), "[%s] ERROR: %s\n", ruleID, res.Message)
			if res.Err != nil {
				fmt.Fprintf(output.Stderr(), "  err: %v\n", res.Err)
			}
			Out.Error(res.Message, "RULE_EXEC_ERROR", "")
			os.Exit(exitError)
		}
		return nil
	},
}

func init() {
	rulesCmd.AddCommand(rulesExecCmd)
}
