// Package output provides the CLI output formatters (json/text/console).
//
// This package is **dedicated to user-facing output**. Diagnostic /
// debug logs and observability events should use stdlib log or the
// slog adapter. Misusing the Formatter family as a log sink lets the
// Quiet/Verbose flags affect diagnostic output and makes problems hard
// to trace.
//
// Three modes:
//   - json (default, AI-oriented)
//   - text (plain, pipe-friendly)
//   - console (colours, tables, human exploration)
//
// The previous "text" mode mixed emoji and tables; that role moved to
// console. "text" is now pure plain text and pipe-friendly.
package output

import (
	"io"
	"os"
)

// Format constants — valid Config.Format values plus aliases.
const (
	FormatJSON    = "json"
	FormatText    = "text"    // plain text (no colours / emojis)
	FormatConsole = "console" // colours / tables / emojis (human exploration)

	// Alias.
	FormatTextAlias    = "txt"
	FormatConsoleAlias = "con"

	// OutputFormatEnvVar — global output-format env var.
	OutputFormatEnvVar = "HOSTLER_OUTPUT_FORMAT"

	// OutputDefaultEnvVar overrides the default output mode via env.
	// Even after the default switched to json, users who prefer console
	// as the session default can opt in via
	// `HOSTLER_OUTPUT_DEFAULT=console`.
	OutputDefaultEnvVar = "HOSTLER_OUTPUT_DEFAULT"

	// DefaultFormat — built-in default. Aligned with the AI-first JSON
	// default. For human exploration, pass `-o console` explicitly or
	// opt in via `HOSTLER_OUTPUT_DEFAULT=console`.
	DefaultFormat = FormatJSON
)

// NormalizeFormat folds aliases onto canonical values.
// Known values pass through; aliases map to their canonical form;
// anything else returns an empty string.
func NormalizeFormat(f string) string {
	switch f {
	case FormatJSON:
		return FormatJSON
	case FormatText, FormatTextAlias:
		return FormatText
	case FormatConsole, FormatConsoleAlias:
		return FormatConsole
	default:
		return ""
	}
}

// Formatter is the CLI user-output formatter interface.
//
// In scope: user-facing output for `hstl *` commands (json / text /
// console).
// Out of scope (forbidden): observability logs, error traces, debug
// output. Those flow through stdlib log or the slog adapter so the
// user-output toggles (Quiet / NoColor) operate independently.
type Formatter interface {
	Success(message string, data any)
	Error(message string, category string, hint string)
	Warning(message string)
	Table(headers []string, rows [][]string)
	Print(data any) // raw data output
}

// Config carries output settings.
type Config struct {
	Format  string // "json" | "text" | "console"
	Quiet   bool
	Verbose bool
	NoColor bool
	Out     io.Writer // stdout
	Err     io.Writer // stderr

	// LegacyEnvelope toggles backward-compatible envelope behaviour.
	// When true, JSON `Print()` writes data verbatim (flat envelope).
	// When false (the default) a `status` key is auto-wrapped onto the
	// payload as `{status, data, message}` if missing. Activate via the
	// `--legacy-envelope` flag or `HOSTLER_CLI_ENVELOPE=legacy`.
	LegacyEnvelope bool
}

// LegacyEnvelopeEnvVar disables `Print()` auto-wrap.
// Only the literal value "legacy" is recognised; anything else keeps
// the standard envelope.
const LegacyEnvelopeEnvVar = "HOSTLER_CLI_ENVELOPE"

// DefaultConfig returns the default output settings.
// Falls back to console; HOSTLER_OUTPUT_DEFAULT overrides.
func DefaultConfig() *Config {
	return &Config{
		Format: ResolveDefault(),
		Out:    os.Stdout,
		Err:    os.Stderr,
	}
}

// ResolveDefault picks the built-in default plus the env override.
// Priority: HOSTLER_OUTPUT_DEFAULT (env) > DefaultFormat constant.
// Unknown env values fall back silently to the default.
func ResolveDefault() string {
	if env := os.Getenv(OutputDefaultEnvVar); env != "" {
		if v := NormalizeFormat(env); v != "" {
			return v
		}
	}
	return DefaultFormat
}

// currentFormat tracks the active Format (used to suppress stderr in
// JSON mode). root.go's resolveOutputFormat sets this via
// SetCurrentFormat.
var currentFormat = DefaultFormat

// SetCurrentFormat sets the active Format.
// Called from root.go PersistentPreRunE.
func SetCurrentFormat(format string) {
	if v := NormalizeFormat(format); v != "" {
		currentFormat = v
	}
}

// CurrentFormat returns the active Format.
func CurrentFormat() string { return currentFormat }

// IsJSONMode reports whether the active Format is JSON.
func IsJSONMode() bool { return currentFormat == FormatJSON }

// Stderr returns the io.Writer used for diagnostic output.
//   - JSON mode (`-o json`): io.Discard — keeps stderr clean so that
//     AI clients merging 2>&1 still see a single valid JSON stream.
//   - Otherwise: os.Stderr.
//
// Caller pattern: `fmt.Fprintf(output.Stderr(), "WARN: ...\n")` for
// user-friendly diagnostic messages. Pairs with the JSON payload's
// `warnings` field (cli-output-contract.md).
func Stderr() io.Writer {
	if IsJSONMode() {
		return io.Discard
	}
	return os.Stderr
}

// New constructs the formatter for the given config.
// Aliases are normalised before branching; an unknown format falls
// back to json.
func New(cfg *Config) Formatter {
	switch NormalizeFormat(cfg.Format) {
	case FormatJSON:
		return &JSONFormatter{cfg: cfg}
	case FormatText:
		return &TextFormatter{cfg: cfg}
	case FormatConsole:
		return &ConsoleFormatter{cfg: cfg}
	default:
		return &JSONFormatter{cfg: cfg}
	}
}
