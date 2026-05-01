package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// ---------------------------------------------------------------------------
// manifest subcommand
// ---------------------------------------------------------------------------
//
// `hstl manifest` matches the persistent `--manifest` flag and serves as the
// parent shell for follow-on subcommands such as get and validate.
//
// Design rationale: docs/08-references/standards/manifest-schema-contract.md, sec. 2
//
// Without arguments:
//   hstl manifest           -> full manifest (= hstl --manifest)
//   hstl manifest --schema  -> full draft-07 schema (= hstl --schema)
//
// Subcommands:
//   hstl manifest get <dotted>
//   hstl manifest validate <file>

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "CLI manifest / JSON Schema introspection",
	Long: `hstl manifest prints the CLI subcommand tree and the JSON Schema.

Equivalents to the persistent flags:
  hstl --manifest           <=>  hstl manifest
  hstl --schema             <=>  hstl manifest --schema
  hstl task --manifest      <=>  (task group only)
  hstl task start --schema  <=>  (single leaf JSON Schema draft-07)

Used for dynamic binding from Claude tool use, MCP servers, and shell completion.
Standard contract: docs/08-references/standards/manifest-schema-contract.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If a persistent flag is set, PersistentPreRun already handled it.
		if manifestFlag || schemaFlag {
			return nil
		}

		// No arguments: emit the full manifest (with group/skill filters applied).
		manifest, err := generateManifest(manifestGroup, manifestSkill)
		if err != nil {
			Out.Error("manifest filter error: "+err.Error(), apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(manifest); err != nil {
			Out.Error("failed to emit manifest: "+err.Error(), apperr.CategoryInternal.String(), "")
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(manifestCmd)
}
