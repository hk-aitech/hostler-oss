// Package cmd - hstl task reconcile-ticket command.
//
// Parallel worktrees plus concurrent hstl invocations can let the
// tasks.work_ticket column drift from the frontmatter `work_ticket` field.
// Dry-run mode reports the drift; --fix reconciles the DB to match the file
// SSOT and emits an audit event.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/task"
)

var (
	reconcileTicketDryRun bool
	reconcileTicketFix    bool
)

var taskReconcileTicketCmd = &cobra.Command{
	Use:   "reconcile-ticket",
	Short: "Reconcile the tasks.work_ticket column with frontmatter (diagnose/repair drift)",
	Long: fmt.Sprintf(`%s diagnoses and repairs drift between the work_ticket column in
the tasks table and the work_ticket field in each task file's frontmatter.

Categories:
  - missing_in_db   : present in frontmatter, empty in DB
  - missing_in_file : present in DB, absent from frontmatter (file may be missing)
  - mismatch        : both present but the values differ
  - ok              : identical (no drift)

Defaults to dry-run, which only prints a report. Pass --fix to update the DB
based on the file-side SSOT.`, brand.ProductName),
	Example: brand.Examplef(
		`task reconcile-ticket`,
		`task reconcile-ticket --fix`,
		`task reconcile-ticket -o json | jq '.data.drifts'`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()

		// Dry-run by default - updates only run when --fix is provided.
		dryRun := !reconcileTicketFix
		if reconcileTicketDryRun {
			dryRun = true
		}

		result, err := task.ReconcileTickets(db.GetDB(), dryRun, reconcileTicketFix)
		if err != nil {
			Out.Error("reconcile failed: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		msg := fmt.Sprintf("reconcile-ticket - total=%d ok=%d drift=%d",
			result.TotalTasks, result.OkCount, result.DriftCount)
		if result.FixApplied {
			msg += fmt.Sprintf(" fixed=%d", result.FixedCount)
		} else if result.DriftCount > 0 {
			msg += " (dry-run - to recover, add the --fix flag)"
		}

		Out.Success(msg, result)
		return nil
	},
}

func init() {
	taskReconcileTicketCmd.Flags().BoolVar(&reconcileTicketDryRun, "dry-run", false, "Diagnose drift without making changes")
	taskReconcileTicketCmd.Flags().BoolVar(&reconcileTicketFix, "fix", false, "Update DB to match the file-side SSOT")
	taskCmd.AddCommand(taskReconcileTicketCmd)
}
