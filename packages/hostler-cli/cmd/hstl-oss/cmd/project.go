// Package cmd - the hstl project namespace subcommand.
//
// Currently provides only init-skeleton, invoked from the project:init
// command's generate phase to idempotently create the initial works/ and
// brand.ProjectDirName skeleton. Augments without overwriting existing files.
// trac: OPS-CM003
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project management subcommand",
}

// InitSkeletonResult is the init-skeleton invocation result.
type InitSkeletonResult struct {
	ProjectRoot     string   `json:"project_root"`
	CreatedDirs     []string `json:"created_dirs"`
	CreatedFiles    []string `json:"created_files"`
	SkippedExisting []string `json:"skipped_existing"`
	DryRun          bool     `json:"dry_run,omitempty"`
	// WouldCreateDirs / WouldCreateFiles are the "to be created" lists in DryRun.
	// Only populated when DryRun=true.
	WouldCreateDirs  []string `json:"would_create_dirs,omitempty"`
	WouldCreateFiles []string `json:"would_create_files,omitempty"`
}

var projectInitSkeletonDryRun bool

var projectInitSkeletonCmd = &cobra.Command{
	Use:   "init-skeleton",
	Short: "Idempotently create the initial works/ + project-config skeleton",
	Long: `Idempotently create the following under the project root:

  works/tasks/                              (directory)
  works/sprints/{backlog,active,completed}  (directories)
  works/tasks/BACKLOG.md                    (skeleton; only when missing)
  works/CURRENT-FOCUS.md                    (idle; only when missing)
  ` + brand.ProjectDirName + `/                                   (directory)

Existing files are never overwritten. The expected usage is to be invoked
from the project:init command's generate stage, but standalone invocation
is also supported.
Use --dry-run to preview the planned creation list without making changes.

See also: the project:init command (full interactive initialization flow)`,
	Example: brand.Examplef(
		`project init-skeleton`,
		`project init-skeleton --dry-run -o json | jq '.would_create_files'`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		result, err := runInitSkeleton(projectInitSkeletonDryRun)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}
		var msg string
		if projectInitSkeletonDryRun {
			msg = fmt.Sprintf("[dry-run] skeleton would be created (%d files / %d directories) - no changes applied",
				len(result.WouldCreateFiles), len(result.WouldCreateDirs))
		} else {
			msg = fmt.Sprintf("skeleton created (new %d files / %d directories)",
				len(result.CreatedFiles), len(result.CreatedDirs))
		}
		Out.Success(msg, result)
		return nil
	},
}

// skeletonDirs is the list of directories created idempotently.
var skeletonDirs = []string{
	filepath.Join("works", "tasks"),
	filepath.Join("works", "sprints", "backlog"),
	filepath.Join("works", "sprints", "active"),
	filepath.Join("works", "sprints", "completed"),
	brand.ProjectDirName,
}

// skeletonBacklogHeader is the initial BACKLOG.md header. Mirrors the
// shape produced by UpdateBacklogMD. Duplicated locally to avoid a circular
// dependency between the project package and fileutil.
const skeletonBacklogHeader = `# Backlog

Unassigned Task list for the project (no Sprint). Auto-synced when ` + "`hstl task create`" + ` /
` + "`hstl task assign`" + ` runs. Run ` + "`hstl backlog rebuild-md`" + ` for a full rebuild.

## Unassigned Tasks

| ID | Title | Size | Type | Priority | Dependencies |
|----|-------|------|------|----------|--------------|
`

func runInitSkeleton(dryRun bool) (*InitSkeletonResult, error) {
	root := app.ProjectRoot()
	if root == "" {
		return nil, fmt.Errorf("failed to detect project root")
	}
	result := &InitSkeletonResult{ProjectRoot: root, DryRun: dryRun}

	// Process directories.
	for _, rel := range skeletonDirs {
		full := filepath.Join(root, rel)
		existed := dirExists(full)
		if dryRun {
			if !existed {
				result.WouldCreateDirs = append(result.WouldCreateDirs, rel)
			} else {
				result.SkippedExisting = append(result.SkippedExisting, rel)
			}
			continue
		}
		if err := os.MkdirAll(full, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", rel, err)
		}
		if !existed {
			result.CreatedDirs = append(result.CreatedDirs, rel)
		}
	}

	// BACKLOG.md (only when missing).
	backlogPath := filepath.Join(root, "works", "tasks", "BACKLOG.md")
	if _, err := os.Stat(backlogPath); os.IsNotExist(err) {
		if dryRun {
			result.WouldCreateFiles = append(result.WouldCreateFiles, "works/tasks/BACKLOG.md")
		} else {
			if wErr := os.WriteFile(backlogPath, []byte(skeletonBacklogHeader), 0o644); wErr != nil {
				return nil, fmt.Errorf("failed to create BACKLOG.md: %w", wErr)
			}
			result.CreatedFiles = append(result.CreatedFiles, "works/tasks/BACKLOG.md")
		}
	} else {
		result.SkippedExisting = append(result.SkippedExisting, "works/tasks/BACKLOG.md")
	}

	// CURRENT-FOCUS.md
	focusPath := filepath.Join(root, "works", "CURRENT-FOCUS.md")
	if _, err := os.Stat(focusPath); os.IsNotExist(err) {
		if dryRun {
			result.WouldCreateFiles = append(result.WouldCreateFiles, "works/CURRENT-FOCUS.md")
		} else {
			if upErr := app.UpdateCurrentFocus("", "", "idle"); upErr != nil {
				return nil, fmt.Errorf("failed to create CURRENT-FOCUS.md: %w", upErr)
			}
			result.CreatedFiles = append(result.CreatedFiles, "works/CURRENT-FOCUS.md")
		}
	} else {
		result.SkippedExisting = append(result.SkippedExisting, "works/CURRENT-FOCUS.md")
	}

	// project-config.yaml - pinProjectKeyToYAML trigger.
	// Even under DryRun, DetectProjectKey would normally be invoked, but the
	// pin function is a no-op when the YAML already exists. Skip the call in
	// dryRun to fully suppress side effects.
	if !dryRun {
		_ = db.DetectProjectKey()
	}
	yamlRel := filepath.Join(brand.ProjectDirName, "project-config.yaml")
	yamlPath := filepath.Join(root, yamlRel)
	if _, err := os.Stat(yamlPath); err == nil {
		result.SkippedExisting = append(result.SkippedExisting, yamlRel)
	} else if dryRun {
		result.WouldCreateFiles = append(result.WouldCreateFiles, yamlRel)
	}

	return result, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func init() {
	projectInitSkeletonCmd.Flags().BoolVar(&projectInitSkeletonDryRun, "dry-run", false, "Return only the planned creation list without making changes")
	projectCmd.AddCommand(projectInitSkeletonCmd)
	rootCmd.AddCommand(projectCmd)
}
