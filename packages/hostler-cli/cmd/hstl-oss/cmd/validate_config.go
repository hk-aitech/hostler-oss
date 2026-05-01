package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
)

// Flags specific to the validate-config subcommand.
var (
	printEffective bool
	validateFormat string
	validateFilter string
)

// validateFormatTable is the default table output format.
const validateFormatTable = "table"

// validateFormatJSON is the JSON output format.
const validateFormatJSON = "json"

// validateFormatText is the simple text output format.
const validateFormatText = "text"

// validateConfigCmd is the deprecated `hstl validate-config` alias.
//
// `hstl validate-config` is deprecated and dispatches internally to
// `hstl config show-effective`. Scheduled for removal.
//
// The legacy behavior (effective values + sources) is preserved. A
// deprecation warning is emitted to stderr on each invocation.
// `hstl config check` is the new inspection entry point; `show-effective`
// retains the original "print effective" responsibility.
var validateConfigCmd = &cobra.Command{
	Use:        "validate-config",
	Short:      "[deprecated] use hstl config check / hstl config show-effective instead",
	Deprecated: "scheduled for removal; use `hstl config check` or `hstl config show-effective`.",
	Long: `[DEPRECATED] hstl validate-config - scheduled for removal

This command is an alias of hstl config show-effective. The existing flags
behave the same and a deprecation warning is emitted on every invocation.

New commands:
  hstl config check           # problem detection + suggested commands
  hstl config show-effective  # effective values + source (the legacy --print-effective behavior)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(output.Stderr(), "warning: hstl validate-config is deprecated - use 'hstl config show-effective' or 'hstl config check'.")

		ec, _ := app.ConfigCollectEffective().(*config.EffectiveConfig)

		fields := ec.Fields
		if validateFilter != "" {
			fields = filterFields(fields, validateFilter)
		}

		switch validateFormat {
		case validateFormatJSON:
			return printValidateJSON(ec, fields)
		case validateFormatText:
			return printValidateText(ec, fields)
		case validateFormatTable:
			return printValidateTable(ec, fields)
		default:
			return fmt.Errorf("unknown --format value: %q (allowed: table/json/text)", validateFormat)
		}
	},
}

// filterFields returns only descriptors whose Name contains the filter substring.
func filterFields(fields []config.FieldDescriptor, filter string) []config.FieldDescriptor {
	lower := strings.ToLower(filter)
	out := make([]config.FieldDescriptor, 0, len(fields))
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f.Name), lower) {
			out = append(out, f)
		}
	}
	return out
}

// printValidateJSON emits the full EffectiveConfig as JSON.
func printValidateJSON(ec *config.EffectiveConfig, fields []config.FieldDescriptor) error {
	out := map[string]any{
		"yaml_path":       ec.YAMLPath,
		"yaml_exists":     ec.YAMLExists,
		"load_error":      ec.LoadError,
		"validation_ok":   ec.ValidationOK,
		"validation_msg":  ec.ValidationMsg,
		"validation_path": ec.ValidationPath, // T303
		"validation_hint": ec.ValidationHint, // T303
		"fields":          fields,
	}
	buf, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(buf))
	return nil
}

// printValidateText emits a simple per-line summary.
func printValidateText(ec *config.EffectiveConfig, fields []config.FieldDescriptor) error {
	printValidateHeader(ec)
	for _, f := range fields {
		detail := ""
		if f.SourceDetail != "" {
			detail = " (" + f.SourceDetail + ")"
		}
		fmt.Printf("- %s = %q  [%s%s]\n", f.Name, f.EffectiveValue, f.Source, detail)
	}
	return nil
}

// printValidateTable emits a table (per the interactive-ux-principles).
func printValidateTable(ec *config.EffectiveConfig, fields []config.FieldDescriptor) error {
	printValidateHeader(ec)

	// Compute column widths.
	const (
		minNameWidth   = 16
		minValueWidth  = 20
		minSourceWidth = 10
		maxValueWidth  = 60
	)
	nameWidth := minNameWidth
	valueWidth := minValueWidth
	sourceWidth := minSourceWidth
	for _, f := range fields {
		if w := len(f.Name); w > nameWidth {
			nameWidth = w
		}
		if w := len(f.EffectiveValue); w > valueWidth {
			valueWidth = w
		}
		if w := len(string(f.Source)); w > sourceWidth {
			sourceWidth = w
		}
	}
	if valueWidth > maxValueWidth {
		valueWidth = maxValueWidth
	}

	// Header
	fmt.Printf("=============================================\n")
	fmt.Printf("  Effective Configuration (%d fields)\n", len(fields))
	fmt.Printf("=============================================\n")
	fmt.Printf("  %-*s  %-*s  %-*s  %s\n", nameWidth, "FIELD", valueWidth, "VALUE", sourceWidth, "SOURCE", "DETAIL")
	fmt.Printf("  %s  %s  %s  %s\n", strings.Repeat("-", nameWidth), strings.Repeat("-", valueWidth), strings.Repeat("-", sourceWidth), strings.Repeat("-", 30))
	for _, f := range fields {
		val := f.EffectiveValue
		if len(val) > valueWidth {
			val = val[:valueWidth-3] + "..."
		}
		fmt.Printf("  %-*s  %-*s  %-*s  %s\n", nameWidth, f.Name, valueWidth, val, sourceWidth, f.Source, f.SourceDetail)
	}
	fmt.Printf("=============================================\n")
	return nil
}

// printValidateHeader emits the shared header (yaml path, load error, validation result).
//
// Surfaces the ValidationError Path / RecoveryHint in three lines so the
// user can act immediately. Previously only Message was shown on one line.
func printValidateHeader(ec *config.EffectiveConfig) {
	if ec.YAMLExists {
		fmt.Printf("yaml_path:   %s\n", ec.YAMLPath)
	} else {
		fmt.Println("yaml_path:   (missing - using built-in defaults)")
	}

	if ec.LoadError != "" {
		fmt.Printf("load_error:  %s\n", ec.LoadError)
	}
	if ec.ValidationOK {
		fmt.Println("validation:  OK")
	} else if ec.ValidationMsg != "" {
		fmt.Println("validation:  FAIL")
		if ec.ValidationPath != "" {
			fmt.Printf("  field:    %s\n", ec.ValidationPath)
		}
		fmt.Printf("  message:  %s\n", ec.ValidationMsg)
		if ec.ValidationHint != "" {
			fmt.Printf("  hint:     %s\n", ec.ValidationHint)
		}
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(validateConfigCmd)
	validateConfigCmd.Flags().BoolVar(&printEffective, "print-effective", true, "Print effective values (default true)")
	validateConfigCmd.Flags().StringVar(&validateFormat, "format", validateFormatTable, "Output format (table|json|text)")
	validateConfigCmd.Flags().StringVar(&validateFilter, "filter", "", "Field-name substring filter")
}
