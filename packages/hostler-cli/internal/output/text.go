package output

import (
	"fmt"
	"strings"
)

// TextFormatter is the plain-text output formatter. It produces output
// suited to pipes and scripts — no color, emoji, or box-drawing characters.
// ADR-003 separated the historic "text" mode (which mixed emoji and tables)
// into ConsoleFormatter, leaving this slot for true plain text.
//
// Comparison:
//   - ConsoleFormatter — color, emoji, box-drawing (for human exploration)
//   - TextFormatter    — plain (for pipes / scripts / machine helpers)
//   - JSONFormatter    — structured (default, AI-oriented)
type TextFormatter struct {
	cfg *Config
}

func (f *TextFormatter) Success(message string, data any) {
	if f.cfg.Quiet {
		if data != nil {
			_, _ = fmt.Fprintf(f.cfg.Out, "%v\n", data)
		}
		return
	}
	_, _ = fmt.Fprintf(f.cfg.Err, "OK: %s\n", message)
	if data != nil && f.cfg.Verbose {
		_, _ = fmt.Fprintf(f.cfg.Out, "%v\n", data)
	}
}

func (f *TextFormatter) Error(message string, category string, hint string) {
	_, _ = fmt.Fprintf(f.cfg.Err, "ERROR: %s\n", message)
	if category != "" {
		_, _ = fmt.Fprintf(f.cfg.Err, "category: %s\n", category)
	}
	if hint != "" {
		_, _ = fmt.Fprintf(f.cfg.Err, "hint: %s\n", hint)
	}
}

func (f *TextFormatter) Warning(message string) {
	_, _ = fmt.Fprintf(f.cfg.Err, "WARN: %s\n", message)
}

// Table renders rows in plain tab-separated form (no colours or box
// drawing). pipe / awk / cut friendly.
func (f *TextFormatter) Table(headers []string, rows [][]string) {
	_, _ = fmt.Fprintln(f.cfg.Out, strings.Join(headers, "\t"))
	for _, row := range rows {
		_, _ = fmt.Fprintln(f.cfg.Out, strings.Join(row, "\t"))
	}
}

func (f *TextFormatter) Print(data any) {
	_, _ = fmt.Fprintf(f.cfg.Out, "%v\n", data)
}
