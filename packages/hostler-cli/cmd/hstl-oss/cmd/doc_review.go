// Package cmd - the `hstl doc-review` subcommand.
// Checks the Task bodies in a Sprint for placeholder markers and stale
// backtick paths.
// stale_path logic is delegated to pkg/ceremony to avoid duplicate-
// implementation drift.
// Switched to FindBacktickRefsInArtifacts (ArtifactOnly mode): the previous
// FullScan over the Task body raised false positives for paths inside
// narrative sub-headings (design decisions / verification, etc.). Now we
// only verify entries under "## Result -> ### Artifacts" - i.e., the
// existence of files the Task actually produced.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
)

var (
	docReviewSprint string
)

var docReviewCmd = &cobra.Command{
	Use:   "doc-review",
	Short: "Check Sprint Task bodies for placeholders and stale backtick paths",
	Long: `Automation entry point for sprint complete Phase 1 (doc-review).
Scans every works/sprints/active/<sprint>/tasks/T*.md file for residual
placeholder bodies and stale backtick paths (missing files). JSON output
is supported.

Usage:
  hstl doc-review --sprint sprint-21
  hstl doc-review --sprint sprint-21 -o json

Exit codes:
  0: pass (0 issues)
  3: BLOCKED (1 or more placeholder / stale entries)`,
	RunE: runDocReview,
}

type docReviewIssue struct {
	TaskID string `json:"task_id"`
	File   string `json:"file"`
	Kind   string `json:"kind"` // "placeholder" | "stale_path"
	Detail string `json:"detail"`
}

type docReviewResult struct {
	Sprint       string           `json:"sprint"`
	TasksScanned int              `json:"tasks_scanned"`
	Issues       []docReviewIssue `json:"issues"`
	IssueCount   int              `json:"issue_count"`
}

var docReviewPlaceholderMarkers = []string{
	"{this Task", "{Requirement", "{Criterion", "{TODO", "{Placeholder", "{One-line summary",
}

func runDocReview(_ *cobra.Command, _ []string) error {
	if docReviewSprint == "" {
		return fmt.Errorf("--sprint is required")
	}

	sprintDir := filepath.Join("works", "sprints", "active", docReviewSprint, "tasks")
	if _, err := os.Stat(sprintDir); err != nil {
		// fallback: backlog / completed
		for _, alt := range []string{"backlog", "completed"} {
			cand := filepath.Join("works", "sprints", alt, docReviewSprint, "tasks")
			if _, ce := os.Stat(cand); ce == nil {
				sprintDir = cand
				break
			}
		}
	}
	if _, err := os.Stat(sprintDir); err != nil {
		return fmt.Errorf("failed to locate sprint directory: %s", docReviewSprint)
	}

	result := docReviewResult{Sprint: docReviewSprint}
	entries, err := os.ReadDir(sprintDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks directory: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(sprintDir, e.Name())
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			continue
		}
		result.TasksScanned++
		taskID := extractTaskID(e.Name())
		content := string(body)

		// Placeholder check - excludes narrative sub-headings so the scanner
		// does not recursively BLOCK itself when a self-describing document
		// quotes the marker literal. Skips `## Result` -> `### Decisions` /
		// `### Verification` / `### Commits` / `### Follow-up Tasks`. The
		// remaining sections (Purpose / Requirements / Done Criteria / Scope /
		// References / Artifacts) stay in scope.
		scanContent := ceremony.ExtractPlaceholderScanContent(content)
		for _, m := range docReviewPlaceholderMarkers {
			if strings.Contains(scanContent, m) {
				result.Issues = append(result.Issues, docReviewIssue{
					TaskID: taskID, File: path, Kind: "placeholder",
					Detail: fmt.Sprintf("marker %q still present", m),
				})
			}
		}

		// Stale backtick-path check delegated to pkg/ceremony.
		// Shares the 12-stage prefix list (cli/ + packages/hostler-cli/ +
		// packages/hostler-plugin/ etc.) plus normalizeRef (trim :line
		// suffix) and isLikelyPathRef (slash required).
		//
		// FindBacktickRefsInArtifacts only scans entries under
		// "## Result -> ### Artifacts". Path mentions inside narrative
		// sections (design decisions / verification / scope limits / etc.)
		// are excluded as false positives. For full-scan over the Task body,
		// call ceremony.FindBacktickRefs directly (used at sprint start).
		refs := ceremony.FindBacktickRefsInArtifacts(content)
		stale := ceremony.CheckBacktickRefsExist(".", refs)
		for _, s := range stale {
			result.Issues = append(result.Issues, docReviewIssue{
				TaskID: taskID, File: path, Kind: "stale_path",
				Detail: fmt.Sprintf("backtick path %q does not exist", s),
			})
		}
	}

	result.IssueCount = len(result.Issues)

	if outputFormat == "json" {
		Out.Print(result)
	} else {
		fmt.Printf("doc-review sprint=%s tasks=%d issues=%d\n",
			result.Sprint, result.TasksScanned, result.IssueCount)
		for _, iss := range result.Issues {
			fmt.Printf("  [%s] %s - %s\n", iss.Kind, iss.TaskID, iss.Detail)
		}
	}

	if result.IssueCount > 0 {
		os.Exit(3)
	}
	return nil
}

func extractTaskID(fileName string) string {
	// Extract "T297" from "T297-xxx.md".
	idx := strings.Index(fileName, "-")
	if idx < 0 {
		return strings.TrimSuffix(fileName, ".md")
	}
	return fileName[:idx]
}

func init() {
	docReviewCmd.Flags().StringVar(&docReviewSprint, "sprint", "", "Target Sprint ID (required)")
	rootCmd.AddCommand(docReviewCmd)
}
