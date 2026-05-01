// trac: WRK-CM001,WRK-CM002,WRK-QR004
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/backlog"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
)

// ---------------------------------------------------------------------------
// Flag variables
// ---------------------------------------------------------------------------

var backlogSyncDryRun bool
var backlogSyncAllowStatusRegression bool

// ---------------------------------------------------------------------------
// Top-level backlog command
// ---------------------------------------------------------------------------

var backlogCmd = &cobra.Command{
	Use:   "backlog",
	Short: "Manage Backlog consistency (sync)",
}

// ---------------------------------------------------------------------------
// backlog sync
// ---------------------------------------------------------------------------

var backlogSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Verify and auto-repair file / DB / BACKLOG.md consistency",
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		raw, err := app.BacklogSyncWithOptions(backlogSyncDryRun, backlogSyncAllowStatusRegression)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}
		result, _ := raw.(*backlog.SyncResult)

		if outputFormat == "json" {
			Out.Print(result)
			return nil
		}

		// Summary output
		s := result.Summary
		mode := "apply"
		if s.DryRun {
			mode = "dry-run"
		}
		Out.Print(fmt.Sprintf(
			"backlog sync (%s) - scanned: %d, issues: %d, auto-fixed: %d, manual: %d",
			mode, s.TaskFilesScanned, s.TotalIssues, s.AutoFixed, s.ManualRequired,
		))

		if len(result.Issues) > 0 {
			headers := []string{"Type", "TaskID", "AutoFixable", "Message"}
			rows := make([][]string, 0, len(result.Issues))
			for _, issue := range result.Issues {
				autofix := "no"
				if issue.AutoFixable {
					autofix = "yes"
				}
				rows = append(rows, []string{
					issue.Type,
					issue.TaskID,
					autofix,
					issue.Message,
				})
			}
			Out.Table(headers, rows)
		}

		if len(result.Fixes) > 0 {
			Out.Print(fmt.Sprintf("auto-fixed %d:", len(result.Fixes)))
			for _, fix := range result.Fixes {
				Out.Print(fmt.Sprintf("  [%s] %s", fix.Type, fix.TaskID))
			}
		}

		return nil
	},
}

// ---------------------------------------------------------------------------
// backlog rebuild-md
// ---------------------------------------------------------------------------

var backlogRebuildIncludeInProgress bool
var backlogRebuildDryRun bool
var backlogRebuildBackup bool // default false, --backup is opt-in

var backlogRebuildMDCmd = &cobra.Command{
	Use:   "rebuild-md",
	Short: "Rebuild works/tasks/BACKLOG.md from scratch",
	Long: `Look up unassigned Tasks in the DB tasks table and rewrite
works/tasks/BACKLOG.md from scratch. By default it overwrites without a
backup. Pass --backup to opt in to creating a .bak file.

Use cases:
- environments with many file-based Tasks but no BACKLOG.md index
- bulk recovery after manual edits drift the BACKLOG.md
- ad-hoc runs where you need the original preserved (then add --backup)

See also: backlog sync (row-level file <-> DB consistency repair)`,
	Example: brand.Examplef(
		"backlog rebuild-md",
		"backlog rebuild-md --include-in-progress -o json",
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		raw, err := app.BacklogRebuildMD(backlog.RebuildMDOptions{
			IncludeInProgress: backlogRebuildIncludeInProgress,
			DryRun:            backlogRebuildDryRun,
			Backup:            backlogRebuildBackup,
		})
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}
		result, _ := raw.(*backlog.RebuildMDResult)

		msg := fmt.Sprintf("BACKLOG.md rebuilt (%d Tasks)", result.Count)
		if backlogRebuildDryRun {
			msg = fmt.Sprintf("[dry-run] BACKLOG.md will be rebuilt (%d Tasks) - no changes applied", result.Count)
		}
		Out.Success(msg, result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	backlogSyncCmd.Flags().BoolVar(&backlogSyncDryRun, "dry-run", false, "Detect issues without modifying anything")
	backlogSyncCmd.Flags().BoolVar(&backlogSyncAllowStatusRegression, "allow-status-regression", false, "Allow status regression (e.g. done->todo). Opt in for intentional reopens; audit events are recorded.")
	backlogRebuildMDCmd.Flags().BoolVar(&backlogRebuildIncludeInProgress, "include-in-progress", false, "Also include in-progress Tasks in the unassigned section")
	backlogRebuildMDCmd.Flags().BoolVar(&backlogRebuildDryRun, "dry-run", false, "Return the list that would be generated without modifying files")
	backlogRebuildMDCmd.Flags().BoolVar(&backlogRebuildBackup, "backup", false, "Back up the existing BACKLOG.md to .bak (default false, opt-in)")

	backlogCmd.AddCommand(backlogSyncCmd)
	backlogCmd.AddCommand(backlogRebuildMDCmd)

	rootCmd.AddCommand(backlogCmd)
}
