package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/kb"
)

// kbCmd is the root of the kb subcommand.
var kbCmd = &cobra.Command{
	Use:   "kb",
	Short: "Manage the Knowledge Base (list/create/rebuild)",
}

// ---------------------------------------------------------------------------
// kb list
// ---------------------------------------------------------------------------

var (
	kbListCategory    string
	kbListKeyword     string
	kbListSince       string
	kbListSubcategory string // T238 (Sprint-16)
	kbListPrefix      string
	kbListLast        int
	kbListLimit       int
)

var kbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List KB cards",
	Long: `List KB cards. Filter flags can be combined.

Filters:
  --category           mistakes|architecture|operations|...
  --keyword            partial title match (use 'kb search' for full-text)
  --since              ISO-8601 date (OccurredAt >=)
  --subcategory        filename-based (e.g. persistence|methodology)
  --prefix             card ID prefix (A|M|O|I)
  --last N             most recent N by OccurredAt
  --limit N            truncate (sort-independent)

Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filtered, err := app.KBList(ports.KBListFilter{
			Category:    kbListCategory,
			Keyword:     kbListKeyword,
			Since:       kbListSince,
			Subcategory: kbListSubcategory,
			Prefix:      kbListPrefix,
			Last:        kbListLast,
			Limit:       kbListLimit,
		})
		if err != nil {
			Out.Error("failed to scan KB: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			Out.Print(filtered)
			return nil
		}

		headers := []string{"ID", "Title", "OccurredAt", "Context", "FilePath"}
		rows := make([][]string, 0, len(filtered))
		for _, c := range filtered {
			rows = append(rows, []string{c.CardID, c.Title, c.OccurredAt, c.Context, c.FilePath})
		}
		Out.Table(headers, rows)
		return nil
	},
}

// ---------------------------------------------------------------------------
// kb search
// ---------------------------------------------------------------------------

var (
	kbSearchMatchAll    bool
	kbSearchCategory    string
	kbSearchSubcategory string
	kbSearchPrefix      string
	kbSearchSince       string
	kbSearchLast        int
	kbSearchLimit       int
)

var kbSearchCmd = &cobra.Command{
	Use:   "search <keyword>...",
	Short: "Full-text search KB cards (title + body)",
	Long: `kb list --keyword only matches title substrings. kb search scans both
title and body and returns matches with matched_snippet.

Multiple keyword support:
  --match-all (default off): all keywords must appear
  --match-any (default behavior): any keyword matches

Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.`,
	Example: brand.Examplef(
		`kb search "counter bump"`,
		`kb search bump rename --match-all`,
		`kb search drift --category operations --last 5`,
	),
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Call the adapter directly (skip the app-layer wrapper for this single subcommand).
		adapter := kb.KBFSStore{}
		results, err := adapter.SearchCards(args, kbSearchMatchAll, ports.KBListFilter{
			Category:    kbSearchCategory,
			Subcategory: kbSearchSubcategory,
			Prefix:      kbSearchPrefix,
			Since:       kbSearchSince,
			Last:        kbSearchLast,
			Limit:       kbSearchLimit,
		})
		if err != nil {
			Out.Error("failed to search KB: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			// JSON includes the snippet.
			out := make([]map[string]any, 0, len(results))
			for _, r := range results {
				out = append(out, map[string]any{
					"card_id":          r.Card.CardID,
					"title":            r.Card.Title,
					"occurred_at":      r.Card.OccurredAt,
					"file_path":        r.Card.FilePath,
					"matched_snippet":  r.Snippet,
					"matched_keywords": r.Matched,
				})
			}
			Out.Print(map[string]any{
				"results":  out,
				"count":    len(out),
				"keywords": args,
			})
			return nil
		}

		headers := []string{"ID", "Title", "Matched", "Snippet"}
		rows := make([][]string, 0, len(results))
		for _, r := range results {
			snippet := r.Snippet
			if len(snippet) > 60 {
				snippet = snippet[:57] + "..."
			}
			rows = append(rows, []string{r.Card.CardID, r.Card.Title, fmt.Sprintf("%d", r.Matched), snippet})
		}
		Out.Table(headers, rows)
		return nil
	},
}

// ---------------------------------------------------------------------------
// kb create
// ---------------------------------------------------------------------------

var (
	kbCreateCategory          string
	kbCreateSubcategory       string
	kbCreateTitle             string
	kbCreateProblem           string
	kbCreateSolution          string
	kbCreateContext           string
	kbCreateCause             string
	kbCreatePrevention        string
	kbCreateRefs              string
	kbCreateViolationPatterns string
)

var kbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a KB card",
	RunE: func(cmd *cobra.Command, args []string) error {
		if kbCreateCategory == "" {
			Out.Error("--category flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}
		if kbCreateTitle == "" {
			Out.Error("--title flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}
		// Guard against the bug where an empty subcategory yields a filename of just `.md`.
		if kbCreateSubcategory == "" {
			Out.Error("--subcategory flag is required (prevents the filename being just .md)", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}
		if kbCreateProblem == "" {
			Out.Error("--problem flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}
		if kbCreateSolution == "" {
			Out.Error("--solution flag is required", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}

		var refs []string
		if kbCreateRefs != "" {
			for _, r := range strings.Split(kbCreateRefs, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					refs = append(refs, r)
				}
			}
		}

		var violationPatterns []string
		if kbCreateViolationPatterns != "" {
			for _, p := range strings.Split(kbCreateViolationPatterns, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					violationPatterns = append(violationPatterns, p)
				}
			}
		}

		record, err := app.KBCreate(ports.KBCardInput{
			Category:          kbCreateCategory,
			Subcategory:       kbCreateSubcategory,
			Title:             kbCreateTitle,
			Problem:           kbCreateProblem,
			Solution:          kbCreateSolution,
			Context:           kbCreateContext,
			Cause:             kbCreateCause,
			Prevention:        kbCreatePrevention,
			Refs:              refs,
			ViolationPatterns: violationPatterns,
		})
		if err != nil {
			Out.Error("failed to create KB card: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		result := map[string]any{
			"card_id":   record.CardID,
			"title":     record.Title,
			"category":  kbCreateCategory,
			"file_path": record.FilePath,
		}
		if len(violationPatterns) > 0 {
			result["violation_patterns"] = violationPatterns
		}
		Out.Success("KB card created", result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// kb rebuild
// ---------------------------------------------------------------------------

var kbRebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild the KB index",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := app.KBRebuild()
		if err != nil {
			Out.Error("failed to rebuild KB index: "+err.Error(), "", "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			Out.Print(result)
			return nil
		}

		Out.Success(fmt.Sprintf("KB index rebuilt - total %d entries, %d files scanned", result.TotalCards, result.FilesScanned), result)
		return nil
	},
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	// kb list flags
	kbListCmd.Flags().StringVar(&kbListCategory, "category", "", "Category filter (e.g. mistakes)")
	kbListCmd.Flags().StringVar(&kbListKeyword, "keyword", "", "Title keyword substring match")
	kbListCmd.Flags().StringVar(&kbListSince, "since", "", "Date filter (e.g. 2026-01-01)")
	kbListCmd.Flags().StringVar(&kbListSubcategory, "subcategory", "", "Filename-based (e.g. persistence)")
	kbListCmd.Flags().StringVar(&kbListPrefix, "prefix", "", "Card ID prefix (A|M|O|I)")
	kbListCmd.Flags().IntVar(&kbListLast, "last", 0, "Most recent N by OccurredAt")
	kbListCmd.Flags().IntVar(&kbListLimit, "limit", 0, "Maximum number of rows")

	// kb search flags
	kbSearchCmd.Flags().BoolVar(&kbSearchMatchAll, "match-all", false, "Require all keywords (default: any)")
	kbSearchCmd.Flags().StringVar(&kbSearchCategory, "category", "", "Category filter")
	kbSearchCmd.Flags().StringVar(&kbSearchSubcategory, "subcategory", "", "Filename-based")
	kbSearchCmd.Flags().StringVar(&kbSearchPrefix, "prefix", "", "Card ID prefix")
	kbSearchCmd.Flags().StringVar(&kbSearchSince, "since", "", "ISO-8601 since")
	kbSearchCmd.Flags().IntVar(&kbSearchLast, "last", 0, "Most recent N")
	kbSearchCmd.Flags().IntVar(&kbSearchLimit, "limit", 0, "Truncate")

	// kb create flags
	kbCreateCmd.Flags().StringVar(&kbCreateCategory, "category", "", "Category (e.g. mistakes, decisions) (required)")
	kbCreateCmd.Flags().StringVar(&kbCreateSubcategory, "subcategory", "", "Subcategory (e.g. code-review) (required)")
	kbCreateCmd.Flags().StringVar(&kbCreateTitle, "title", "", "Card title (required)")
	kbCreateCmd.Flags().StringVar(&kbCreateProblem, "problem", "", "Problem description (required)")
	kbCreateCmd.Flags().StringVar(&kbCreateSolution, "solution", "", "Solution (required)")
	kbCreateCmd.Flags().StringVar(&kbCreateContext, "context", "", "Context where the issue occurred")
	kbCreateCmd.Flags().StringVar(&kbCreateCause, "cause", "", "Cause analysis")
	kbCreateCmd.Flags().StringVar(&kbCreatePrevention, "prevention", "", "Prevention guidance")
	kbCreateCmd.Flags().StringVar(&kbCreateRefs, "refs", "", "Comma-separated reference links")
	kbCreateCmd.Flags().StringVar(&kbCreateViolationPatterns, "violation-patterns", "", "Comma-separated grep patterns (optional)")
	for _, flag := range []string{"category", "subcategory", "title", "problem", "solution"} {
		if err := kbCreateCmd.MarkFlagRequired(flag); err != nil {
			// stderr + exit 1 instead of panic - this is a developer error
			// inside init() (e.g. flag name typo) but we surface a clear
			// message instead of a stack trace.
			fmt.Fprintf(os.Stderr, "init(kb create): MarkFlagRequired(%q) failed: %v\n", flag, err)
			os.Exit(1)
		}
	}

	// Register subcommands.
	kbCmd.AddCommand(kbListCmd)
	kbCmd.AddCommand(kbSearchCmd)
	kbCmd.AddCommand(kbCreateCmd)
	kbCmd.AddCommand(kbRebuildCmd)
	rootCmd.AddCommand(kbCmd)
}
