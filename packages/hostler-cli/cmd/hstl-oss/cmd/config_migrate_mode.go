// Package cmd — T349 (Sprint-28) `hstl config migrate-mode` subcommand.
//
// Switches the project config folder between modes per ADR-042 T352
// revision's unified folder convention.
//
//	hstl config migrate-mode --to=hstl   # hostler-only mode (.hstl/)
//	hstl config migrate-mode --to=pry4   # ecosystem mode (.pry4/, completed in a future sprint)
//	hstl config migrate-mode --to=axl    # alternate mode (.axl/, completed in a future sprint)
//
// Current implementation scope (Sprint 28):
//   - rename `.hostler/` (legacy full name) → `.hstl/` (short primary)
//   - keep a legacy symlink (backward-compat option `--keep-legacy-symlink`)
//   - idempotent: no-op when only `.hstl/` exists
//   - strict discovery: BLOCK when conflicting folders coexist
//     (e.g. `.hstl/` + `.pry4/`)
//
// Out of scope (follow-up tasks):
//   - the actual implementations for the pry4/axl modes will be completed
//     in their respective starting sprints. For now they return a stub
//     guidance error.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// mode value constants.
const (
	configMigrateModeHstl = "hstl"
	configMigrateModePry4 = "pry4"
	configMigrateModeAxl  = "axl"
)

// allowedModes is the full list of values that may be passed to `--to`.
var allowedModes = []string{
	configMigrateModeHstl,
	configMigrateModePry4,
	configMigrateModeAxl,
}

// modeDirName maps each mode to its primary folder name (ADR-042 T352
// revision).
var modeDirName = map[string]string{
	configMigrateModeHstl: ".hstl",
	configMigrateModePry4: ".pry4",
	configMigrateModeAxl:  ".axl",
}

// --------- flags ---------

var (
	configMigrateModeTo                string
	configMigrateModeDryRun            bool
	configMigrateModeKeepLegacySymlink bool
)

var configMigrateModeCmd = &cobra.Command{
	Use:   "migrate-mode",
	Short: "Switch config folder mode (e.g. .hostler → .hstl)",
	Long: fmt.Sprintf(`Switches the project config folder between modes per
ADR-042 T352 revision's unified folder convention.

Supported modes:
  hstl   hostler-only — .hstl/   (current primary)
  pry4   ecosystem    — .pry4/   (completed in a future sprint)
  axl    alternate    — .axl/    (completed in a future sprint)

Current implementation scope:
  - only the hstl mode transition (.hostler/ → .hstl/) is fully supported
  - pry4/axl will be completed in their respective starting sprints —
    today they return a guidance error

Examples:
  %s config migrate-mode --to=hstl              # .hostler/ → .hstl/ (default)
  %s config migrate-mode --to=hstl --dry-run    # preview the changes
  %s config migrate-mode --to=hstl --keep-legacy-symlink

Strict discovery: when both the destination folder and the source folder
exist, the operation is BLOCKED. The user must remove one side manually
and retry (data-loss prevention).
`,
		brand.ShortName, brand.ShortName, brand.ShortName),
	RunE: runConfigMigrateMode,
}

// migrationPlan is the plan object built right before execution. Both
// dry-run and apply paths drive their common logic from this plan.
type migrationPlan struct {
	Mode          string   // target mode (hstl/pry4/axl)
	Root          string   // absolute project root
	Primary       string   // primary folder after migration (e.g. .hstl)
	PrimaryPath   string   // Root/Primary
	SourceDir     string   // rename source (e.g. .hostler). "" if absent.
	SourcePath    string   // Root/SourceDir
	Conflicts     []string // conflicting other-mode primaries (e.g. .pry4, .axl) — must not coexist
	AlreadyTarget bool     // Primary is already in the correct state
}

func runConfigMigrateMode(cmd *cobra.Command, args []string) error {
	plan, err := buildMigrationPlan()
	if err != nil {
		return err
	}

	// Both folders coexist → BLOCKED.
	if len(plan.Conflicts) > 0 {
		return fmt.Errorf(
			"strict CLI discovery violation — primary (%s) and conflicting folder(s) %v coexist. "+
				"manually remove or migrate the conflicting folder, then retry",
			plan.PrimaryPath, plan.Conflicts)
	}

	if plan.AlreadyTarget && plan.SourceDir == "" {
		fmt.Printf("mode:      %s\n", plan.Mode)
		fmt.Printf("primary:   %s (already in target state)\n", plan.PrimaryPath)
		fmt.Println("action:    no-op (idempotent)")
		return nil
	}

	// pry4/axl modes are stubs for now.
	if plan.Mode != configMigrateModeHstl {
		return fmt.Errorf(
			"--to=%s mode transition will be completed in its starting sprint. "+
				"Sprint 28 only supports the hstl mode transition",
			plan.Mode)
	}

	if configMigrateModeDryRun {
		return printMigrateModeDryRun(plan)
	}
	return applyMigrateMode(plan)
}

// buildMigrationPlan reads the current filesystem state and assembles a
// migration plan.
func buildMigrationPlan() (*migrationPlan, error) {
	if configMigrateModeTo == "" {
		return nil, errors.New("--to flag is required (hstl|pry4|axl)")
	}
	if _, ok := modeDirName[configMigrateModeTo]; !ok {
		return nil, fmt.Errorf(
			"unknown --to value %q (allowed: %v)",
			configMigrateModeTo, allowedModes)
	}

	root := app.ProjectRoot()
	primary := modeDirName[configMigrateModeTo]
	primaryPath := filepath.Join(root, primary)

	plan := &migrationPlan{
		Mode:        configMigrateModeTo,
		Root:        root,
		Primary:     primary,
		PrimaryPath: primaryPath,
	}

	// Check Primary existence (excluding symlinks: those are handled
	// separately for legacy compatibility).
	if info, err := os.Lstat(primaryPath); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			plan.AlreadyTarget = true
		}
	}

	// Pick the first existing legacy candidate as the SourceDir.
	for _, legacy := range brand.LegacyProjectDirNames {
		p := filepath.Join(root, legacy)
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		// A symlink is not a source (it's only kept as backward-compat
		// after a previous rename).
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		plan.SourceDir = legacy
		plan.SourcePath = p
		break
	}

	// Conflicts — any other-mode primary (.pry4, .axl, etc.) that exists
	// as a real directory besides our Primary.
	for mode, dir := range modeDirName {
		if mode == configMigrateModeTo {
			continue
		}
		p := filepath.Join(root, dir)
		info, err := os.Lstat(p)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		plan.Conflicts = append(plan.Conflicts, dir)
	}

	return plan, nil
}

// printMigrateModeDryRun prints only the planned changes to stdout.
func printMigrateModeDryRun(plan *migrationPlan) error {
	fmt.Printf("mode:      %s\n", plan.Mode)
	fmt.Printf("primary:   %s\n", plan.PrimaryPath)
	if plan.SourceDir != "" {
		fmt.Printf("source:    %s\n", plan.SourcePath)
		fmt.Printf("plan:      rename %s → %s\n", plan.SourceDir, plan.Primary)
	} else if plan.AlreadyTarget {
		fmt.Println("plan:      no-op (already primary)")
	} else {
		fmt.Println("plan:      create empty .hstl/ (no source)")
	}
	if configMigrateModeKeepLegacySymlink && plan.SourceDir != "" {
		fmt.Printf("symlink:   %s → %s (backward compat)\n", plan.SourceDir, plan.Primary)
	}
	fmt.Println("dry-run — no files were modified.")
	return nil
}

// applyMigrateMode performs the actual rename.
func applyMigrateMode(plan *migrationPlan) error {
	if plan.AlreadyTarget && plan.SourceDir == "" {
		fmt.Printf("mode:      %s\n", plan.Mode)
		fmt.Println("action:    no-op (idempotent)")
		return nil
	}

	// rename: source → primary.
	if plan.SourceDir != "" && !plan.AlreadyTarget {
		if err := os.Rename(plan.SourcePath, plan.PrimaryPath); err != nil {
			return fmt.Errorf("rename %s → %s failed: %w", plan.SourcePath, plan.PrimaryPath, err)
		}
		fmt.Printf("renamed:   %s → %s\n", plan.SourceDir, plan.Primary)
	} else if !plan.AlreadyTarget {
		// no source + no primary → create empty folder.
		if err := os.MkdirAll(plan.PrimaryPath, 0o755); err != nil {
			return fmt.Errorf("mkdir %s failed: %w", plan.PrimaryPath, err)
		}
		fmt.Printf("created:   %s (empty)\n", plan.PrimaryPath)
	}

	// Optional: create the legacy symlink (`.hostler` → `.hstl`).
	if configMigrateModeKeepLegacySymlink && plan.SourceDir != "" {
		if err := os.Symlink(plan.Primary, plan.SourcePath); err != nil {
			// Already exists is OK (no separate symlink-replace logic
			// required).
			if !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("symlink %s → %s failed: %w", plan.SourceDir, plan.Primary, err)
			}
		} else {
			fmt.Printf("symlink:   %s → %s (backward compat)\n", plan.SourceDir, plan.Primary)
		}
	}

	fmt.Println("action:    completed")
	return nil
}

func init() {
	configMigrateModeCmd.Flags().StringVar(
		&configMigrateModeTo, "to", "",
		"target mode (hstl|pry4|axl)")
	configMigrateModeCmd.Flags().BoolVar(
		&configMigrateModeDryRun, "dry-run", false,
		"only print planned changes, do not modify files")
	configMigrateModeCmd.Flags().BoolVar(
		&configMigrateModeKeepLegacySymlink, "keep-legacy-symlink", false,
		"after rename, keep a symlink at the legacy name (e.g. .hostler) pointing to the new folder")
	configCmd.AddCommand(configMigrateModeCmd)
}
