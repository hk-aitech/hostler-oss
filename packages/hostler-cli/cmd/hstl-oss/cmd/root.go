// Package cmd defines the cobra commands for the hstl CLI.
package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
)

var (
	// Injected at build time via -ldflags.
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Global flags.
//
// T207 (ADR-038, ported back from hstl T864/ADR-003): three-mode
// reclassification + json default.
//   - `-o, --output` is the primary input channel (default json)
//   - `--json` / `--text` / `--console` are explicit shortcuts
//   - cascade support for HSTL_OUTPUT_FORMAT env / yaml
//     output.default_format
var (
	outputFormat    string
	jsonShortcut    bool // T533
	textShortcut    bool // T207 — explicit plain-text shortcut
	consoleShortcut bool // T207 — explicit console (color/table) shortcut
	quiet           bool
	verbose         bool
	projectDir      string
	noColor         bool
	legacyEnvelope  bool // T447 (Sprint-38) — disable envelope auto-wrap
)

// outputFormatJSON is the json output format constant (T533). Lower
// layers (sprint_stamp, summary_judge, etc.) compare against this
// value.
const outputFormatJSON = output.FormatJSON

// Out is the current output formatter.
var Out output.Formatter

// outputOverride is a hook for tests to substitute Out (T579).
// When non-nil, PersistentPreRun uses this formatter instead of the
// default stdout formatter. Production code does not touch this; only
// the test helper (WithTestOutput) sets it.
var outputOverride output.Formatter

var rootCmd = &cobra.Command{
	Use:   brand.ShortName,
	Short: brand.ProductName + " — unified work-management CLI for development projects",
	Long: brand.ProductName + ` is a unified CLI tool that provides Sprint/Task
lifecycle, knowledge capture, code review, operational design, and AI
agent collaboration.`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// T579: when tests set outputOverride, use it as-is.
		if outputOverride != nil {
			Out = outputOverride
			return
		}
		resolved := resolveOutputFormat(cmd)
		outputFormat = resolved
		cfg := output.DefaultConfig()
		cfg.Format = resolved
		cfg.Quiet = quiet
		cfg.Verbose = verbose
		cfg.NoColor = noColor
		// T447 (Sprint-38): envelope-standardization backward-compat.
		// Disable JSON Print() auto-wrap when --legacy-envelope flag (higher
		// priority) or HOSTLER_CLI_ENVELOPE=legacy env is set. Default
		// applies the envelope standard.
		if legacyEnvelope || os.Getenv(output.LegacyEnvelopeEnvVar) == "legacy" {
			cfg.LegacyEnvelope = true
		}
		Out = output.New(cfg)
		// T292: register the active format — output.Stderr() uses it to
		// detect JSON mode.
		output.SetCurrentFormat(resolved)
		// T250 (Sprint-23): once JSON mode is determined, swap pkg/log
		// Logger's global writer to io.Discard so all NewDefault()
		// loggers stop polluting stderr. HOSTLER_LOG=stderr opts out
		// (handled inside pkglog).
		if resolved == output.FormatJSON {
			pkglog.SetGlobalOutput(io.Discard)
		} else {
			pkglog.SetGlobalOutput(os.Stderr)
		}
	},
}

// resolveOutputFormat decides the final format per ADR-038 cascade
// rules.
//
// Precedence:
//  1. explicit shortcut bool flags (--json / --text / --console) — on
//     conflict, json > console > text.
//  2. an explicitly specified -o/--output value.
//  3. the HSTL_OUTPUT_FORMAT env var.
//  4. `.hostler/project-config.yaml` output.default_format.
//  5. the built-in default (json, AI-first).
//
// Unrecognized values fall back to the default.
func resolveOutputFormat(cmd *cobra.Command) string {
	// 1. shortcut bool — when multiple are set, explicit precedence is
	// json > console > text.
	switch {
	case jsonShortcut:
		return output.FormatJSON
	case consoleShortcut:
		return output.FormatConsole
	case textShortcut:
		return output.FormatText
	}

	// 2. -o/--output explicit value (only when set, not the default).
	if cmd.Flags() != nil && cmd.Flags().Changed("output") {
		if v := output.NormalizeFormat(outputFormat); v != "" {
			return v
		}
	}

	// 3. env var. HOSTLER_OUTPUT_FORMAT (canonical).
	if env := os.Getenv(output.OutputFormatEnvVar); env != "" {
		if v := output.NormalizeFormat(env); v != "" {
			return v
		}
	}

	// 4. yaml output.default_format. Silently skip on load failure /
	// absence.
	if cfg, err := config.LoadProjectConfig(); err == nil && cfg != nil {
		if v := output.NormalizeFormat(cfg.GetOutputDefaultFormat()); v != "" {
			return v
		}
	}

	// 5. default (honors the HOSTLER_OUTPUT_DEFAULT env override).
	return output.ResolveDefault()
}

func init() {
	// T207 — default json (ADR-038 AI-first). Valid values:
	// json | text | console (aliases: txt/con).
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", output.DefaultFormat, "output format (json|text|console)")
	// T533: --json is an alias for -o json.
	rootCmd.PersistentFlags().BoolVar(&jsonShortcut, "json", false, "JSON output (same as -o json, default)")
	// T207: --text / --console shortcuts.
	rootCmd.PersistentFlags().BoolVar(&textShortcut, "text", false, "Plain-text output (same as -o text; no color/emoji)")
	rootCmd.PersistentFlags().BoolVar(&consoleShortcut, "console", false, "Console output (same as -o console; with color/tables)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Minimal output (data only)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().StringVar(&projectDir, "dir", "", "Project directory (default: cwd)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color")
	rootCmd.PersistentFlags().BoolVar(&legacyEnvelope, "legacy-envelope", false, "Disable JSON envelope standardization (T447) — return list/get responses flat (tasks/sprints/items at the top level) instead of {status, data, message} auto-wrap. Equivalent to HOSTLER_CLI_ENVELOPE=legacy env.")
}

// Execute runs the root command.
func Execute() error {
	// Register Annotations on ceremony-bearing commands (used by
	// --manifest output).
	RegisterCeremonyAnnotations()

	// T333 (Sprint 25): inject manifest/schema support once all
	// subcommands are registered. cobra.OnInitialize is called after
	// c.Runnable() checks, so RunE on a group cmd must be injected
	// beforehand. The Execute entry point is the safest spot.
	installManifestSchemaSupport(rootCmd)

	return rootCmd.Execute()
}
