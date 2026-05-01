// trac: HAR-CM018,HAR-CM019,HAR-CM020,HAR-QR001
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
)

// ---------------------------------------------------------------------------
// Flag variables
// ---------------------------------------------------------------------------

var harnessEvidence string

// ---------------------------------------------------------------------------
// printHarnessState renders the harness state as an ASCII box.
// ---------------------------------------------------------------------------

func printHarnessState(state *domain.HarnessGetResult) {
	separator := strings.Repeat("=", 45)
	fmt.Fprintln(output.Stderr(), separator)
	fmt.Fprintf(output.Stderr(), "  Harness Gate - %s\n", state.EntityID)
	fmt.Fprintln(output.Stderr(), separator)
	fmt.Fprintln(output.Stderr())

	for _, item := range state.Items {
		mark := " "
		if item.Done {
			mark = "x"
		}
		fmt.Fprintf(output.Stderr(), "  [%s] %s - %s\n", mark, item.ID, item.Name)
	}

	fmt.Fprintln(output.Stderr())
	unchecked := state.RequiredTotal - state.RequiredDone
	statusLabel := "OK"
	if state.Blocked {
		statusLabel = "BLOCKED"
	}
	fmt.Fprintf(output.Stderr(), "  Unchecked: %d / Total: %d  -> %s\n", unchecked, state.RequiredTotal, statusLabel)
	fmt.Fprintln(output.Stderr(), separator)
}

// printHarnessBlocked emits the unchecked items and resolution commands when BLOCKED.
func printHarnessBlocked(entityType, entityID string, state *domain.HarnessGetResult) {
	separator := strings.Repeat("=", 45)
	fmt.Fprintln(output.Stderr(), separator)
	_, _ = fmt.Fprintf(output.Stderr(), "  %s BLOCKED - %s\n", strings.ToUpper(entityType[:1])+entityType[1:], entityID)
	fmt.Fprintln(output.Stderr(), separator)
	fmt.Fprintln(output.Stderr())

	hasUnchecked := false
	for _, item := range state.Items {
		if item.Required && !item.Done {
			fmt.Fprintf(output.Stderr(), "  [ ] %s - %s\n", item.ID, item.Name)
			hasUnchecked = true
		}
	}

	if !hasUnchecked {
		fmt.Fprintln(output.Stderr(), "  (no unchecked items)")
	}

	fmt.Fprintln(output.Stderr())
	fmt.Fprintln(output.Stderr(), "  How to resolve:")
	for _, item := range state.Items {
		if item.Required && !item.Done {
			fmt.Fprintf(output.Stderr(), "    %s harness check %s %s %s --evidence \"evidence\"\n",
				brand.ShortName, entityType, entityID, item.ID)
		}
	}
	fmt.Fprintf(output.Stderr(), "  Clear all: %s harness check-all %s %s\n", brand.ShortName, entityType, entityID)
	fmt.Fprintln(output.Stderr(), separator)
}

// ---------------------------------------------------------------------------
// Top-level harness command
// ---------------------------------------------------------------------------

var harnessCmd = &cobra.Command{
	Use:   "harness",
	Short: "Inspect the Harness Gate and check items",
	Long:  "Inspect the Task/Sprint completion checklist (Harness Gate) and mark items done.",
}

// ---------------------------------------------------------------------------
// harness get
// ---------------------------------------------------------------------------

var harnessGetCmd = &cobra.Command{
	Use:   "get <entity-type> <entity-id>",
	Short: "Inspect the Harness Gate state",
	Long: `Inspect the Harness Gate checklist state. entity-type: task | sprint.
With --output json the per-item done/evidence is emitted as structured data.

See also: harness check (clear an item), task complete / sprint complete (gate verification)`,
	Example: brand.Examplef(
		`harness get task T137`,
		`harness get sprint sprint-19 -o json`,
	),
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityType := args[0]
		entityID := args[1]

		if entityType != "task" && entityType != "sprint" {
			Out.Error("entity-type must be task or sprint", apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		raw, err := app.HarnessGet(entityType, entityID)
		if err != nil {
			var nfe *apperr.NotFoundError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}
		state := raw.(*domain.HarnessGetResult)

		if outputFormat == "json" {
			Out.Print(state)
			return nil
		}

		printHarnessState(state)
		return nil
	},
}

// ---------------------------------------------------------------------------
// harness check
// ---------------------------------------------------------------------------

var harnessCheckCmd = &cobra.Command{
	Use:   "check <entity-type> <entity-id> <item-id>",
	Short: "Mark a Harness Gate item as done",
	Long: `Mark an individual Harness Gate item as done. entity-type: task | sprint.

See also: harness get (inspect state), harness check-all (clear all items)`,
	Example: brand.Examplef(
		`harness check task T137 build_passed --evidence "go build ./... succeeded"`,
		`harness check sprint sprint-19 phase5_retro --evidence "wrote 3 KPT cards"`,
	),
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityType := args[0]
		entityID := args[1]
		itemID := args[2]

		if entityType != "task" && entityType != "sprint" {
			Out.Error("entity-type must be task or sprint", apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		result, err := app.HarnessCheck(entityType, entityID, itemID, harnessEvidence, "cli")
		if err != nil {
			var nfe *apperr.NotFoundError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}

		Out.Success(fmt.Sprintf("[%s] %s checked", entityID, itemID), result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// harness check-all
// ---------------------------------------------------------------------------

var harnessCheckAllCmd = &cobra.Command{
	Use:   "check-all <entity-type> <entity-id>",
	Short: "Mark every unchecked Harness Gate item as done",
	Long: `Mark every required-but-unchecked item as done at once. entity-type: task | sprint.

See also: harness get (inspect state), harness check (clear a single item)`,
	Example: brand.Examplef(
		`harness check-all task T137 --evidence "all checks complete"`,
		`harness check-all sprint sprint-19 --evidence "10-Phase complete"`,
	),
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entityType := args[0]
		entityID := args[1]

		if entityType != "task" && entityType != "sprint" {
			Out.Error("entity-type must be task or sprint", apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}

		mustInitDB()
		defer db.Close()

		raw, err := app.HarnessGet(entityType, entityID)
		if err != nil {
			var nfe *apperr.NotFoundError
			if errors.As(err, &nfe) {
				Out.Error(nfe.Error(), nfe.Category(), nfe.RecoveryHint)
			} else {
				Out.Error(err.Error(), "", "")
			}
			os.Exit(exitError)
		}
		state := raw.(*domain.HarnessGetResult)

		checkedCount := 0
		for _, item := range state.Items {
			if item.Required && !item.Done {
				_, checkErr := app.HarnessCheck(entityType, entityID, item.ID, harnessEvidence, "cli")
				if checkErr != nil {
					Out.Warning(fmt.Sprintf("[%s] check failed: %v", item.ID, checkErr))
					continue
				}
				fmt.Fprintf(output.Stderr(), "  [x] %s - %s\n", item.ID, item.Name)
				checkedCount++
			}
		}

		if checkedCount == 0 {
			Out.Print("No required items remain unchecked.")
		} else {
			Out.Success(fmt.Sprintf("%d items checked", checkedCount), map[string]any{
				"entity_type":   entityType,
				"entity_id":     entityID,
				"checked_count": checkedCount,
			})
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	// --evidence flag (shared by check and check-all).
	harnessCheckCmd.Flags().StringVar(&harnessEvidence, "evidence", "", "Evidence (e.g. \"build log URL\")")
	harnessCheckAllCmd.Flags().StringVar(&harnessEvidence, "evidence", "", "Evidence applied uniformly to every item")

	harnessCmd.AddCommand(harnessGetCmd)
	harnessCmd.AddCommand(harnessCheckCmd)
	harnessCmd.AddCommand(harnessCheckAllCmd)

	rootCmd.AddCommand(harnessCmd)
}
