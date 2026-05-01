package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// ---------------------------------------------------------------------------
// hstl manifest get <dotted-name>
// ---------------------------------------------------------------------------
//
// Pin-point query for a single subcommand's manifest or schema.
// Design rationale: docs/08-references/standards/manifest-schema-contract.md, sec. 5
//
// dotted-name convention (up to 3 levels):
//   task.start          -> hstl task start
//   sprint.complete     -> hstl sprint complete
//   harness.auto-check  -> hstl harness auto-check
//   manifest.get        -> hstl manifest get (self-reference)
//
// Output:
//   With --schema: LeafSchema (Claude tool use compatible)
//   Otherwise:     CommandMeta (existing manifest format)

var manifestGetCmd = &cobra.Command{
	Use:   "get <dotted-name>",
	Short: "Look up the manifest or schema for a single subcommand by dotted-name",
	Long: `Pin-point query for a single subcommand's manifest or schema by dotted-name.

Examples:
  hstl manifest get task.start              -> CommandMeta (existing format)
  hstl manifest get task.start --schema     -> LeafSchema (draft-07, Claude tool use)
  hstl manifest get sprint.complete
  hstl manifest get harness.auto-check

Useful for dynamically registering a single Claude tool-use entry without the
full manifest payload, or for emitting an MCP server tools/list entry.

dotted-name supports up to 3 levels (group.subcommand.leaf).
A missing dotted-name returns exit 2 plus a recovery_hint.

Standard contract: docs/08-references/standards/manifest-schema-contract.md, sec. 5`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dotted := args[0]
		target, err := resolveDottedName(dotted)
		if err != nil {
			// exit 3 + recovery_hint per the manifest-schema-contract drift policy.
			suggestions := suggestSimilarNames(dotted)
			hint := ""
			if len(suggestions) > 0 {
				hint = fmt.Sprintf("similar names: %s", strings.Join(suggestions, ", "))
			}
			Out.Error("dotted-name '"+dotted+"' not found: "+err.Error(), apperr.CategoryInvalidInput.String(), hint)
			os.Exit(exitBlocked)
			return nil // unreachable
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		if schemaFlag {
			leaf := buildLeafSchema(target)
			leaf.Schema = manifestSchemaURI
			return enc.Encode(leaf)
		}
		// Without --schema, emit the existing CommandMeta format.
		meta := buildCommandMeta(target)
		return enc.Encode(meta)
	},
}

func init() {
	manifestCmd.AddCommand(manifestGetCmd)
}

// resolveDottedName resolves a dotted-name such as "task.start" into a cobra
// command, walking up to 3 levels. Returns an error when not found.
func resolveDottedName(dotted string) (*cobra.Command, error) {
	if dotted == "" {
		return nil, fmt.Errorf("empty dotted-name")
	}

	parts := strings.Split(dotted, ".")
	if len(parts) > 3 {
		return nil, fmt.Errorf("dotted-name exceeds 3 levels (input: %d)", len(parts))
	}

	current := rootCmd
	for i, part := range parts {
		found := false
		for _, sub := range current.Commands() {
			if sub.Name() == part {
				current = sub
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("level %d (%q) not found within %q", i, part, current.CommandPath())
		}
	}

	// Reached a leaf: if HasSubCommands() it is a branch, not a leaf.
	if current.HasSubCommands() {
		return nil, fmt.Errorf("%q is a group command (not a leaf) - please specify a more specific dotted-name", current.CommandPath())
	}

	return current, nil
}

// suggestSimilarNames returns similar leaf names for an invalid dotted-name.
// Simple prefix matching - smarter heuristics like Levenshtein are follow-up.
func suggestSimilarNames(dotted string) []string {
	leaves := collectLeafCommands(rootCmd)

	// Convert the dotted prefix to a cmd path for sub-tree search.
	firstPart := strings.Split(dotted, ".")[0]
	expectedPrefix := brand.ShortName + " " + firstPart

	var suggestions []string
	for _, leaf := range leaves {
		path := leaf.CommandPath()
		if strings.HasPrefix(path, expectedPrefix) {
			// "hstl task start" -> "task.start"
			trimmed := strings.TrimPrefix(path, brand.ShortName+" ")
			suggestions = append(suggestions, strings.ReplaceAll(trimmed, " ", "."))
		}
	}

	sort.Strings(suggestions)
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	return suggestions
}
