// Package cmd - the `hstl config` subcommand tree.
//
// All config operations are unified under a single namespace:
//
//	hstl config                    # = check (default)
//	hstl config check              # inspect - problem list + per-category suggested commands
//	hstl config fix                # auto-repair small issues (migration excluded)
//	hstl config fix --dry-run      # preview
//	hstl config fix --yes          # non-interactive
//	hstl config migrate            # structural change (dry-run by default)
//	hstl config migrate --apply    # actually apply
//	hstl config show-effective     # effective values + source
//
// The legacy `hstl validate-config` / `hstl migrate` aliases remain
// deprecated and are scheduled for removal. Their implementations reuse the
// runners in this file.
//
// Design principles (2026-04-12 user feedback):
//  1. verb = subcommand, modifier = flag (`fix` is a subcommand, `--yes` is a flag)
//  2. fix and migrate are separate - structural changes only run when called explicitly
//  3. `--dry-run` provides preview (no separate `diff` subcommand)
//
// trac: OPS-QR001
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
)

// configFormatText / configFormatJSON - shared output format constants.
const (
	configFormatText = "text"
	configFormatJSON = "json"
)

// --------- Shared flags (used across subcommands) ---------

var (
	configCheckFormat string
	configCheckTarget string

	configFixDryRun bool
	configFixYes    bool
	configFixTarget string

	// configShowEffective* mirror the existing validate-config flags.
	configShowFormat string
	configShowFilter string
	configShowField  string
)

// ---------------------------------------------------------------------------
// hstl config (root)
// ---------------------------------------------------------------------------

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Unified entry point for project-config.yaml check / fix / migrate",
	Long: fmt.Sprintf(`%s config - unified management command for project-config.yaml.

Subcommands:
  check           inspect - problem list + per-category suggested commands
  fix             auto-repair small issues (migration excluded; interactive)
  migrate         structural change (v1 -> v2; dry-run by default; needs --apply)
  show-effective  print current effective values + source

Running without arguments behaves like 'check'.

Design principles:
  - fix never auto-handles migration category problems.
  - Structural changes must be invoked via '%s config migrate --apply'.

Examples:
  %s config                        # = check
  %s config check --format json
  %s config fix --dry-run
  %s config fix --yes
  %s config migrate                # default dry-run
  %s config migrate --apply
  %s config show-effective --filter task_result`,
		brand.ShortName, brand.ShortName,
		brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName,
		brand.ShortName, brand.ShortName, brand.ShortName),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior = check (no args).
		return runConfigCheck(cmd, args)
	},
}

// ---------------------------------------------------------------------------
// hstl config check
// ---------------------------------------------------------------------------

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Inspect project-config.yaml - problem list + suggested commands",
	Long: fmt.Sprintf(`Inspect project-config.yaml and detect problems.

Per-category recommendations:
  migration   -> run separately via '%s config migrate --apply' (excluded from fix)
  deprecation -> '%s config fix'
  schema      -> '%s config fix' (when auto-repair is possible)
  normalize   -> '%s config fix'`,
		brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName),
	RunE: runConfigCheck,
}

func runConfigCheck(cmd *cobra.Command, args []string) error {
	target, err := resolveConfigTarget(configCheckTarget)
	if err != nil {
		return err
	}
	raw, err := app.ConfigDetectProblems(target)
	if err != nil {
		return err
	}
	report, _ := raw.(*config.ProblemReport)

	if configCheckFormat == configFormatJSON {
		return printJSON(report)
	}
	return printCheckText(report)
}

func printCheckText(report *config.ProblemReport) error {
	if !report.Exists {
		fmt.Printf("yaml_path:   (missing - %s)\n", report.YAMLPath)
		fmt.Println("validation:  OK target file missing")
		return nil
	}

	fmt.Printf("yaml_path:   %s\n", report.YAMLPath)
	if !report.HasProblems() {
		fmt.Println("validation:  OK")
		return nil
	}

	fmt.Printf("validation:  warn %d problems found\n\n", len(report.Problems))
	for _, p := range report.Problems {
		fmt.Printf("[%s/%s] %s\n", p.Category, p.Code, p.Message)
		if p.Path != "" {
			fmt.Printf("  path:    %s\n", p.Path)
		}
		fmt.Printf("  severity: %s\n", p.Severity)
		if p.RecoveryHint != "" {
			fmt.Printf("  recovery: %s\n", p.RecoveryHint)
		}
		if len(p.Commands) > 0 {
			fmt.Println("  suggested:")
			for _, c := range p.Commands {
				fmt.Printf("    %s\n", c)
			}
		}
		if p.Category == config.CategoryMigration {
			fmt.Printf("  warn: this problem is not handled by '%s config fix'.\n", brand.ShortName)
		}
		fmt.Println()
	}

	fixable := report.FixableProblems()
	migrations := report.MigrationProblems()
	fmt.Printf("Summary: %d fixable (config fix target) / %d migration (config migrate target)\n",
		len(fixable), len(migrations))
	return nil
}

// ---------------------------------------------------------------------------
// hstl config fix
// ---------------------------------------------------------------------------

var configFixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Auto-repair small config problems (migration excluded)",
	Long: fmt.Sprintf(`Auto-repair AutoFixable problems found by DetectConfigProblems.

Core rule: this command never processes migration category problems.
Structural changes must run via '%s config migrate --apply'.

Flags:
  --dry-run   print the planned changes without modifying files (check + plan)
  --yes       skip the confirmation prompt (CI-friendly)`, brand.ShortName),
	RunE: runConfigFix,
}

func runConfigFix(cmd *cobra.Command, args []string) error {
	target, err := resolveConfigTarget(configFixTarget)
	if err != nil {
		return err
	}

	raw, err := app.ConfigDetectProblems(target)
	if err != nil {
		return err
	}
	report, _ := raw.(*config.ProblemReport)
	if !report.Exists {
		fmt.Printf("yaml_path: (missing - %s)\n", report.YAMLPath)
		fmt.Println("fix:       skipped - target file missing")
		return nil
	}

	fixable := report.FixableProblems()
	migrations := report.MigrationProblems()

	fmt.Printf("yaml_path: %s\n", report.YAMLPath)
	fmt.Printf("Inspection: %d fixable / %d migration\n\n", len(fixable), len(migrations))

	// Surface migration items first (never auto-processed).
	if len(migrations) > 0 {
		fmt.Println("migration category problems (not eligible for fix):")
		for _, p := range migrations {
			fmt.Printf("  [%s] %s\n", p.Code, p.Message)
		}
		fmt.Printf("  -> handle via: %s config migrate --apply\n", brand.ShortName)
		fmt.Println()
	}

	if len(fixable) == 0 {
		fmt.Println("fix: no auto-repairable problems to apply.")
		return nil
	}

	// dry-run: print the planned changes only.
	if configFixDryRun {
		rawFix, ferr := app.ConfigApplyFixes(report, true)
		if ferr != nil {
			return ferr
		}
		result, _ := rawFix.(*config.FixResult)
		return printFixDryRun(result, fixable)
	}

	// Interactive: confirm unless --yes.
	if !configFixYes {
		fmt.Println("These problems will be auto-repaired:")
		for _, p := range fixable {
			fmt.Printf("  [%s/%s] %s\n", p.Category, p.Code, p.Message)
		}
		fmt.Print("\nApply? [y/N] ")
		if !confirmYes() {
			fmt.Println("canceled.")
			return nil
		}
	}

	rawFix, err := app.ConfigApplyFixes(report, false)
	if err != nil {
		return err
	}
	result, _ := rawFix.(*config.FixResult)
	if result.Applied {
		fmt.Printf("OK %d problems fixed (%s)\n", len(result.FixedProblems), result.YAMLPath)
	} else {
		fmt.Println("no changes (target file no-op).")
	}
	return nil
}

// printFixDryRun emits the "what would change" report in dry-run mode.
func printFixDryRun(result *config.FixResult, fixable []config.ConfigProblem) error {
	fmt.Println("dry-run - no files were modified.")
	fmt.Println()
	fmt.Printf("Problems to apply (%d):\n", len(fixable))
	for _, p := range fixable {
		fmt.Printf("  [%s/%s] %s\n", p.Category, p.Code, p.Message)
	}
	fmt.Println()

	// OriginalBytes vs FixedBytes diff.
	if len(result.OriginalBytes) == 0 && len(result.FixedBytes) == 0 {
		fmt.Println("(no changes)")
		return nil
	}
	diff := app.ConfigRenderDiff(result.OriginalBytes, result.FixedBytes)
	fmt.Println("--- diff ---")
	fmt.Print(diff)
	return nil
}

// confirmYes reads y/Y from stdin and returns whether the user confirmed.
func confirmYes() bool {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

// ---------------------------------------------------------------------------
// hstl config show-effective
// ---------------------------------------------------------------------------

var configShowEffectiveCmd = &cobra.Command{
	Use: "show-effective",
	// Aligns with the catalog SSOT name. To preserve existing callers,
	// show-effective is the primary Use and effective is declared as an alias.
	Aliases: []string{"effective"},
	Short:   "Print the current effective config values with their sources",
	Long: fmt.Sprintf(`Collect the current process's effective configuration field by field and print it.
Each field reports which source applied (env / yaml / default).

Returns the same result as the legacy '%s validate-config --print-effective'.

Precedence matrix: docs/04-guides/development/project-config-precedence.md

When --field is provided, returns JSON with only that field's effective
value + source:
  { "field": "platform.kind", "value": "go", "source": "yaml" }

Examples:
  %s config show-effective
  %s config show-effective --filter task_result
  %s config show-effective --format json
  %s config show-effective --field platform.kind
  %s config show-effective --field version`,
		brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName),
	RunE: func(cmd *cobra.Command, args []string) error {
		ec, _ := app.ConfigCollectEffective().(*config.EffectiveConfig)

		// --field: single-field lookup - returns { "field", "value", "source" } JSON.
		if configShowField != "" {
			return printSingleField(ec.Fields, configShowField)
		}

		fields := ec.Fields
		if configShowFilter != "" {
			fields = filterFields(fields, configShowFilter)
		}
		switch configShowFormat {
		case validateFormatJSON:
			return printValidateJSON(ec, fields)
		case validateFormatText:
			return printValidateText(ec, fields)
		case validateFormatTable, "":
			return printValidateTable(ec, fields)
		default:
			return fmt.Errorf("unknown --format value: %q (allowed: table/json/text)", configShowFormat)
		}
	},
}

// printSingleField finds the field whose dot-notation name matches exactly
// and writes its { "field", "value", "source" } JSON to stdout.
func printSingleField(fields []config.FieldDescriptor, name string) error {
	for _, f := range fields {
		if f.Name == name {
			return printJSON(map[string]any{
				"field":  f.Name,
				"value":  f.EffectiveValue,
				"source": string(f.Source),
			})
		}
	}
	return fmt.Errorf("unknown field: %q (run %s config show-effective for the full list)", name, brand.ShortName)
}

// ---------------------------------------------------------------------------
// Shared utilities
// ---------------------------------------------------------------------------

// defaultConfigRelPath returns the default config file path relative to the
// project root. Prefers the primary directory (brand.ProjectDirName),
// falling back to brand.LegacyProjectDirNames in order.
func defaultConfigRelPath(root string) string {
	candidates := append([]string{brand.ProjectDirName}, brand.LegacyProjectDirNames...)
	for _, dir := range candidates {
		p := filepath.Join(root, dir, "project-config.yaml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Even without the primary file, return the primary path - the caller decides existence.
	return filepath.Join(root, brand.ProjectDirName, "project-config.yaml")
}

// resolveConfigTarget falls back to the default project-config.yaml path
// when the target flag is empty. Existence is handled by
// DetectConfigProblems, so we only resolve the path here.
func resolveConfigTarget(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	root := app.ProjectRoot()
	return defaultConfigRelPath(root), nil
}

// printJSON marshals v with two-space indent and writes it to stdout.
func printJSON(v any) error {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(buf))
	return nil
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	// Register on the root.
	rootCmd.AddCommand(configCmd)

	// check
	configCheckCmd.Flags().StringVar(&configCheckFormat, "format", configFormatText, "Output format (text|json)")
	configCheckCmd.Flags().StringVar(&configCheckTarget, "target", "", "Target YAML file (default: brand project-config.yaml, with legacy alternates as fallback)")
	configCmd.AddCommand(configCheckCmd)

	// fix
	configFixCmd.Flags().BoolVar(&configFixDryRun, "dry-run", false, "Print planned changes without modifying files")
	configFixCmd.Flags().BoolVar(&configFixYes, "yes", false, "Skip the confirmation prompt (CI-friendly)")
	configFixCmd.Flags().StringVar(&configFixTarget, "target", "", "Target YAML file (default: brand project-config.yaml, with legacy alternates as fallback)")
	configCmd.AddCommand(configFixCmd)

	// show-effective
	configShowEffectiveCmd.Flags().StringVar(&configShowFormat, "format", validateFormatTable, "Output format (table|json|text)")
	configShowEffectiveCmd.Flags().StringVar(&configShowFilter, "filter", "", "Field-name substring filter")
	configShowEffectiveCmd.Flags().StringVar(&configShowField, "field", "", "Single field name (dot notation). When set, returns only that field's value+source as JSON.")
	// Source tags are emitted by default already, so this flag is a no-op,
	// but we keep it as a recognized flag because external docs/skills may
	// invoke `--with-source` explicitly.
	configShowEffectiveCmd.Flags().Bool("with-source", false, "Include source tags in output (default true; compatibility flag)")
	configCmd.AddCommand(configShowEffectiveCmd)
}
