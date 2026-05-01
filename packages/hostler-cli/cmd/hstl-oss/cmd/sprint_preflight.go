// Package cmd - the `hstl sprint preflight-scope` subcommand.
// Automatically measures the scope (direct cmd -> pkg calls) when planning
// port conversion refactor tasks.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/portpreflight"
)

const (
	// preflightDefaultCmdDir is the default cmd directory path; can be
	// overridden with --cmd-dir.
	preflightDefaultCmdDir = "cmd/hstl-oss/cmd"
)

var (
	preflightPkg    string
	preflightCmdDir string
	preflightAllPkg bool // walk every pkg/ subdirectory
)

var sprintPreflightCmd = &cobra.Command{
	Use:   "preflight-scope",
	Short: "Measure port-conversion scope - statistics for direct cmd -> pkg/X calls",
	Long: `A measurement tool that resolves scope estimation error when planning
port conversion refactor tasks. Aggregates the files, functions, and
frequency with which the cmd directory directly calls public (exported)
functions of pkg/X. composition.go and _test.go are excluded.

Paste the measurement straight into the Requirements section of the Task and
use it as evidence for the "0 direct function calls" completion criterion.
Replaces the manual grep in Playbook Step 1.

Examples:
  hstl sprint preflight-scope --pkg mailbox
  hstl sprint preflight-scope --pkg config -o json
  hstl sprint preflight-scope --pkg task --cmd-dir cmd/hstl-oss/cmd`,
	RunE: runSprintPreflight,
}

func init() {
	sprintPreflightCmd.Flags().StringVar(&preflightPkg, "pkg", "",
		"Name of the pkg to analyze (mutually exclusive with --all-pkg)")
	sprintPreflightCmd.Flags().StringVar(&preflightCmdDir, "cmd-dir", "",
		"Path to the cmd directory (defaults to "+preflightDefaultCmdDir+" under the project root)")
	sprintPreflightCmd.Flags().BoolVar(&preflightAllPkg, "all-pkg", false,
		"Iterate every package under pkg/ for analysis")

	// Register as a sprint subcommand.
	sprintCmd.AddCommand(sprintPreflightCmd)
}

func runSprintPreflight(_ *cobra.Command, _ []string) error {
	if preflightPkg == "" && !preflightAllPkg {
		return fmt.Errorf("either --pkg or --all-pkg is required")
	}
	if preflightPkg != "" && preflightAllPkg {
		return fmt.Errorf("--pkg and --all-pkg are mutually exclusive")
	}
	cmdDir := preflightCmdDir
	if cmdDir == "" {
		root := app.ProjectRoot()
		cmdDir = filepath.Join(root, preflightDefaultCmdDir)
	} else if !filepath.IsAbs(cmdDir) {
		abs, err := filepath.Abs(cmdDir)
		if err != nil {
			return fmt.Errorf("failed to convert to absolute path: %w", err)
		}
		cmdDir = abs
	}

	if _, err := os.Stat(cmdDir); err != nil {
		return fmt.Errorf("failed to check cmd directory: %w", err)
	}

	if preflightAllPkg {
		return runSprintPreflightAllPkg(cmdDir)
	}

	result, err := portpreflight.Analyze(preflightPkg, cmdDir)
	if err != nil {
		return err
	}

	if outputFormat == "json" {
		Out.Print(result)
		return nil
	}

	renderPreflightText(result)
	return nil
}

// runSprintPreflightAllPkg walks every directory under pkg/.
func runSprintPreflightAllPkg(cmdDir string) error {
	pkgRoot := filepath.Join(app.ProjectRoot(), "packages/hostler-cli/pkg")
	entries, err := os.ReadDir(pkgRoot)
	if err != nil {
		return fmt.Errorf("failed to read pkg/ directory: %w", err)
	}

	type pkgSummary struct {
		Package        string `json:"package"`
		FilesScanned   int    `json:"files_scanned"`
		FilesWithCalls int    `json:"files_with_calls"`
		CallsTotal     int    `json:"calls_total"`
		FunctionsCount int    `json:"functions_count"`
	}
	var summaries []pkgSummary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 0 && name[0] == '_' {
			continue
		}
		r, aerr := portpreflight.Analyze(name, cmdDir)
		if aerr != nil {
			continue
		}
		summaries = append(summaries, pkgSummary{
			Package:        r.Package,
			FilesScanned:   r.FilesScanned,
			FilesWithCalls: r.FilesWithCalls,
			CallsTotal:     r.CallsTotal,
			FunctionsCount: r.FunctionsCount,
		})
	}

	if outputFormat == "json" {
		Out.Print(summaries)
		return nil
	}

	fmt.Println("=============================================")
	fmt.Println("  Port Preflight Scope - all-pkg")
	fmt.Println("=============================================")
	fmt.Printf("  %-20s %8s %8s %8s %8s\n", "package", "scanned", "w/calls", "calls", "funcs")
	for _, s := range summaries {
		fmt.Printf("  %-20s %8d %8d %8d %8d\n",
			s.Package, s.FilesScanned, s.FilesWithCalls, s.CallsTotal, s.FunctionsCount)
	}
	fmt.Printf("\n  Analyzed %d packages total\n", len(summaries))
	return nil
}

// renderPreflightText emits the text-mode rendering directly to stdout.
func renderPreflightText(r *portpreflight.Result) {
	fmt.Println("=============================================")
	fmt.Printf("  Port Preflight Scope - pkg/%s\n", r.Package)
	fmt.Println("=============================================")
	fmt.Println()
	fmt.Printf("cmd directory: %s\n", r.CmdDir)
	fmt.Printf("Files scanned: %d (composition.go, _test.go excluded)\n", r.FilesScanned)
	fmt.Printf("Files with calls: %d / total calls: %d / unique functions: %d\n",
		r.FilesWithCalls, r.CallsTotal, r.FunctionsCount)
	fmt.Println()

	if r.CallsTotal == 0 {
		fmt.Println("No direct calls - standby port (docs-only).")
		return
	}

	fmt.Println("Calls per function:")
	for _, f := range r.Functions {
		fmt.Printf("  %-30s %d\n", f.Name, f.Count)
	}
	fmt.Println()
	fmt.Println("Calls per file:")
	for _, f := range r.Files {
		fmt.Printf("  %-40s %d\n", f.Path, f.Count)
	}

	// Size hint (Playbook Step 1).
	fmt.Println()
	fmt.Printf("Size hint: ")
	switch {
	case r.FunctionsCount == 0:
		fmt.Println("XS (no calls)")
	case r.FunctionsCount <= 5 && r.FilesWithCalls <= 2:
		fmt.Println("S (single ServiceAdapter recommended)")
	case r.FunctionsCount >= 10 || r.FilesWithCalls >= 3:
		fmt.Println("M/L (consider splitting - watch G2 gotcha)")
	default:
		fmt.Println("S~M")
	}
}
