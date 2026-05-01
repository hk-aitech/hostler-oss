package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/registry"
)

// ---------------------------------------------------------------------------
// Flag variables
// ---------------------------------------------------------------------------

var (
	registryListType string

	registryAllocType       string
	registryAllocName       string
	registryAllocOwner      string
	registryAllocValue      int
	registryAllocRangeStart int
	registryAllocRangeEnd   int
	registryAllocEnv        string
	registryAllocNote       string

	registryCheckType       string
	registryCheckValue      int
	registryCheckRangeStart int
	registryCheckRangeEnd   int
	registryCheckEnv        string
)

// ---------------------------------------------------------------------------
// Top-level registry command
// ---------------------------------------------------------------------------

var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage the resource registry (list/allocate)",
}

// ---------------------------------------------------------------------------
// registry list
// ---------------------------------------------------------------------------

var registryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources in the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := registry.ListResources(registryListType)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			Out.Print(result)
			return nil
		}

		headers := []string{"Name", "Type", "Description", "Allocations"}
		rows := make([][]string, 0, len(result.Resources))
		for _, r := range result.Resources {
			rows = append(rows, []string{
				r.Name,
				r.Type,
				r.Description,
				strconv.Itoa(r.AllocationCount),
			})
		}

		Out.Print(fmt.Sprintf("Project: %s  (updated: %s)", result.Project, result.UpdatedAt))
		Out.Table(headers, rows)
		return nil
	},
}

// ---------------------------------------------------------------------------
// registry allocate
// ---------------------------------------------------------------------------

var registryAllocateCmd = &cobra.Command{
	Use:   "allocate",
	Short: "Allocate a resource",
	RunE: func(cmd *cobra.Command, args []string) error {
		if registryAllocType == "" {
			Out.Error("--type flag is required", apperr.CategoryInvalidState.String(), "Specify either unique or range.")
			os.Exit(exitError)
		}
		if registryAllocName == "" {
			Out.Error("--name flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}
		if registryAllocOwner == "" {
			Out.Error("--owner flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}

		var valuePtr *int
		if cmd.Flags().Changed("value") {
			valuePtr = &registryAllocValue
		}

		var rangeStartPtr, rangeEndPtr *int
		if cmd.Flags().Changed("range-start") && cmd.Flags().Changed("range-end") {
			rangeStartPtr = &registryAllocRangeStart
			rangeEndPtr = &registryAllocRangeEnd
		}

		result, err := registry.Allocate(
			registryAllocType,
			registryAllocName,
			registryAllocOwner,
			valuePtr,
			rangeStartPtr,
			rangeEndPtr,
			registryAllocEnv,
			registryAllocNote,
		)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			Out.Print(result)
			return nil
		}

		switch r := result.(type) {
		case registry.AllocateResult:
			Out.Success(fmt.Sprintf("resource allocated: %s", r.Resource), result)
		case registry.AllocateConflictResult:
			Out.Error(r.Message, apperr.CategoryConflict.String(), fmt.Sprintf("%d conflicts - use a different value.", len(r.Conflicts)))
			os.Exit(exitError)
		default:
			Out.Print(result)
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// registry check
// ---------------------------------------------------------------------------

var registryCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Pre-check resource availability (conflict check before allocate)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if registryCheckType == "" {
			Out.Error("--type flag is required", apperr.CategoryInvalidState.String(), "Specify either unique or range.")
			os.Exit(exitError)
		}

		// resource_kind auto-determination.
		resourceKind := "unique"
		if cmd.Flags().Changed("range-start") || cmd.Flags().Changed("range-end") {
			resourceKind = "range"
		}

		var valuePtr *int
		if cmd.Flags().Changed("value") {
			valuePtr = &registryCheckValue
		}

		var rangeStartPtr, rangeEndPtr *int
		if cmd.Flags().Changed("range-start") {
			rangeStartPtr = &registryCheckRangeStart
		}
		if cmd.Flags().Changed("range-end") {
			rangeEndPtr = &registryCheckRangeEnd
		}

		result, err := registry.Check(resourceKind, registryCheckType, valuePtr, rangeStartPtr, rangeEndPtr, registryCheckEnv)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			if result.Status == "CONFLICT" {
				Out.Print(map[string]any{
					"available": false,
					"conflicts": result.Conflicts,
				})
			} else {
				Out.Print(map[string]any{
					"available":  true,
					"owner":      nil,
					"suggestion": result.Suggestion,
				})
			}
			return nil
		}

		if result.Status == "CONFLICT" {
			Out.Error(
				fmt.Sprintf("%d conflicts found", len(result.Conflicts)),
				apperr.CategoryConflict.String(),
				"Use a different value or run registry list to inspect the state.",
			)
			headers := []string{"Owner", "Value", "Env"}
			rows := make([][]string, 0, len(result.Conflicts))
			for _, c := range result.Conflicts {
				rows = append(rows, []string{c.Owner, fmt.Sprintf("%v", c.Value), c.Env})
			}
			Out.Table(headers, rows)
			os.Exit(exitError)
		}

		Out.Success(fmt.Sprintf("available (suggested: %v)", result.Suggestion), result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// registry sync
// ---------------------------------------------------------------------------

var registrySyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync the runtime registry to docs/registry.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		root := app.ProjectRoot()
		result, err := registry.SyncToProject(root)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			Out.Print(result)
			return nil
		}

		if result.Synced {
			Out.Success(fmt.Sprintf("synced: %s (%d changes)", result.DocsPath, result.DiffCount), result)
		} else {
			Out.Print(fmt.Sprintf("nothing to sync or runtime file missing: %s", result.DocsPath))
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	// registry list flags
	registryListCmd.Flags().StringVar(&registryListType, "type", "", "Filter by resource type (port|event_range|ufc, etc.)")

	// registry allocate flags
	registryAllocateCmd.Flags().StringVar(&registryAllocType, "type", "", "Resource type: unique|range (required)")
	registryAllocateCmd.Flags().StringVar(&registryAllocName, "name", "", "Resource name (required)")
	registryAllocateCmd.Flags().StringVar(&registryAllocOwner, "owner", "", "Allocation owner (required)")
	registryAllocateCmd.Flags().IntVar(&registryAllocValue, "value", 0, "Allocation value (unique type; auto-suggested if omitted)")
	registryAllocateCmd.Flags().IntVar(&registryAllocRangeStart, "range-start", 0, "Range start (range type)")
	registryAllocateCmd.Flags().IntVar(&registryAllocRangeEnd, "range-end", 0, "Range end (range type)")
	registryAllocateCmd.Flags().StringVar(&registryAllocEnv, "env", "", "Environment (e.g. dev|prod)")
	registryAllocateCmd.Flags().StringVar(&registryAllocNote, "note", "", "Note")

	// registry check flags
	registryCheckCmd.Flags().StringVar(&registryCheckType, "type", "", "Resource name to check (e.g. ports, logger_event_ranges) (required)")
	registryCheckCmd.Flags().IntVar(&registryCheckValue, "value", 0, "Value to check (unique type)")
	registryCheckCmd.Flags().IntVar(&registryCheckRangeStart, "range-start", 0, "Start value (range type)")
	registryCheckCmd.Flags().IntVar(&registryCheckRangeEnd, "range-end", 0, "End value (range type)")
	registryCheckCmd.Flags().StringVar(&registryCheckEnv, "env", "", "Environment scope (unique type allows independent envs)")

	// Register subcommands
	registryCmd.AddCommand(registryListCmd)
	registryCmd.AddCommand(registryAllocateCmd)
	registryCmd.AddCommand(registryCheckCmd)
	registryCmd.AddCommand(registrySyncCmd)

	rootCmd.AddCommand(registryCmd)
}
