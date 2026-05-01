// Package cmd - the hstl project init-docs subcommand.
//
// Implements the project-structure I12 triad. Creates the docs/ tree
// idempotently in new projects following the ADR-002 10-slot prefix
// standard. Existing files are never overwritten.
//
// Default: hstl project init-docs
// Preview: hstl project init-docs --dry-run -o json | jq '.would_create_files'
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// docsPrefixSpec describes one entry in the docs/ 10-slot prefix layout
// (ADR-002). The SSOT lives in skills/project-structure/SKILL.md; this list
// is a copy.
type docsPrefixSpec struct {
	Dir   string // docs/{Dir}
	Title string // INDEX.md title
	Desc  string // INDEX.md leading paragraph description
}

// docsPrefixes is the list of prefix directories that init-docs creates.
// Based on the ADR-002 10-slot standard (currently 9 slots in use; slot 09 reserved).
// To add or reorder a prefix, update skills/project-structure/SKILL.md and
// ADR-002 first to avoid drift.
var docsPrefixes = []docsPrefixSpec{
	{"00-project", "Project definition", "Top-level project definition: PDD, roadmap, implementation plan, system overview, quality attributes, domain glossary."},
	{"01-requirements", "Requirements", "SRS, FRS, NFR, requirement traceability matrix (RTM)."},
	{"02-architecture", "Architecture decisions and views", "Architecture decision records in the ADR-NNN-title.md form, plus key views."},
	{"03-design", "Design documents", "Architecture, domain, feature, UX, API, and deployment design documents."},
	{"04-guides", "Guides", "How-to, troubleshooting, runbooks, and playbooks for development, setup, and operations."},
	{"05-operations", "Operations specs", "SLO definitions, FMA, monitoring strategy, data-management policy."},
	{"06-reports", "Reports", "Analyses, measurements, audits, checklists, spike outputs."},
	{"07-knowledge", "Knowledge base", "API gotchas, architecture decision retrospectives, domain knowledge, operational lessons, mistakes (recurrence-prevention) cards."},
	{"08-references", "External references", "External API docs, research / domain materials, external specification mappings."},
}

// docsIndexTopTemplate is the skeleton body for docs/index.md.
const docsIndexTopTemplate = `# Project documents

docs/ follows the ADR-002 ` + "`docs-numbering-standard`" + ` 10-slot prefix convention.
Use each prefix directory's ` + "`INDEX.md`" + ` as the entry point.

## Prefix

| # | Directory | Purpose |
|---|-----------|---------|
{{ROWS}}

Detailed rules: ` + "`skills/project-structure/SKILL.md`" + ` (SSOT).
`

// docsIndexPrefixTemplate is the skeleton for each prefix INDEX.md.
const docsIndexPrefixTemplate = `# {{TITLE}}

{{DESC}}

## Contents

<!-- Update this list when documents are added. There is no auto-generation script. -->

- (empty)
`

// InitDocsResult is the init-docs invocation result.
type InitDocsResult struct {
	ProjectRoot      string   `json:"project_root"`
	CreatedDirs      []string `json:"created_dirs"`
	CreatedFiles     []string `json:"created_files"`
	SkippedExisting  []string `json:"skipped_existing"`
	DryRun           bool     `json:"dry_run,omitempty"`
	WouldCreateDirs  []string `json:"would_create_dirs,omitempty"`
	WouldCreateFiles []string `json:"would_create_files,omitempty"`
}

var projectInitDocsDryRun bool

var projectInitDocsCmd = &cobra.Command{
	Use:   "init-docs",
	Short: "Idempotently create the docs/ 10-slot prefix tree and INDEX.md skeletons",
	Long: `Idempotently create the ADR-002 10-slot prefix directories and per-prefix
INDEX.md skeletons under docs/. Also creates the top-level docs/index.md entry point.

Existing files are never overwritten. Use --dry-run to preview the planned
creation list without modifying anything.

SSOT: skills/project-structure/SKILL.md (per ADR-002).
Implements the project-structure I12 triad.`,
	Example: brand.Examplef(
		`project init-docs`,
		`project init-docs --dry-run -o json | jq '.would_create_files'`,
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := runInitDocs(projectInitDocsDryRun)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}
		var msg string
		if projectInitDocsDryRun {
			msg = fmt.Sprintf("[dry-run] docs tree would be created (%d files / %d directories) - no changes applied",
				len(result.WouldCreateFiles), len(result.WouldCreateDirs))
		} else {
			msg = fmt.Sprintf("docs tree created (new %d files / %d directories, %d preserved)",
				len(result.CreatedFiles), len(result.CreatedDirs), len(result.SkippedExisting))
		}
		Out.Success(msg, result)
		return nil
	},
}

func runInitDocs(dryRun bool) (*InitDocsResult, error) {
	root := app.ProjectRoot()
	if root == "" {
		return nil, fmt.Errorf("failed to detect project root")
	}
	result := &InitDocsResult{ProjectRoot: root, DryRun: dryRun}

	// Ensure the docs/ root.
	if err := ensureDir(root, "docs", dryRun, result); err != nil {
		return nil, err
	}

	// Each prefix directory + INDEX.md.
	for _, spec := range docsPrefixes {
		rel := filepath.Join("docs", spec.Dir)
		if err := ensureDir(root, rel, dryRun, result); err != nil {
			return nil, err
		}

		indexRel := filepath.Join(rel, "INDEX.md")
		body := renderIndexPrefix(spec)
		if err := ensureFile(root, indexRel, body, dryRun, result); err != nil {
			return nil, err
		}
	}

	// Top-level docs/index.md.
	topBody := renderIndexTop()
	if err := ensureFile(root, filepath.Join("docs", "index.md"), topBody, dryRun, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ensureDir idempotently creates the directory at rel. In DryRun, only records into WouldCreateDirs.
func ensureDir(root, rel string, dryRun bool, result *InitDocsResult) error {
	full := filepath.Join(root, rel)
	if dirExists(full) {
		result.SkippedExisting = append(result.SkippedExisting, rel)
		return nil
	}
	if dryRun {
		result.WouldCreateDirs = append(result.WouldCreateDirs, rel)
		return nil
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", rel, err)
	}
	result.CreatedDirs = append(result.CreatedDirs, rel)
	return nil
}

// ensureFile creates the file at rel with body only when missing.
func ensureFile(root, rel, body string, dryRun bool, result *InitDocsResult) error {
	full := filepath.Join(root, rel)
	if _, err := os.Stat(full); err == nil {
		result.SkippedExisting = append(result.SkippedExisting, rel)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat file %s: %w", rel, err)
	}
	if dryRun {
		result.WouldCreateFiles = append(result.WouldCreateFiles, rel)
		return nil
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		return fmt.Errorf("failed to create file %s: %w", rel, err)
	}
	result.CreatedFiles = append(result.CreatedFiles, rel)
	return nil
}

// renderIndexTop renders the docs/index.md skeleton.
func renderIndexTop() string {
	var rowsBuilder strings.Builder
	for i, p := range docsPrefixes {
		if i > 0 {
			rowsBuilder.WriteString("\n")
		}
		fmt.Fprintf(&rowsBuilder, "| %02d | `%s/` | %s |", i, p.Dir, p.Desc)
	}
	return strings.ReplaceAll(docsIndexTopTemplate, "{{ROWS}}", rowsBuilder.String())
}

// renderIndexPrefix renders the per-prefix INDEX.md skeleton.
func renderIndexPrefix(spec docsPrefixSpec) string {
	out := strings.ReplaceAll(docsIndexPrefixTemplate, "{{TITLE}}", spec.Title)
	return strings.ReplaceAll(out, "{{DESC}}", spec.Desc)
}

func init() {
	projectInitDocsCmd.Flags().BoolVar(&projectInitDocsDryRun, "dry-run", false, "Return only the planned creation list without making changes")
	projectCmd.AddCommand(projectInitDocsCmd)
}
