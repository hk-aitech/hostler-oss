package output

import (
	"fmt"
	"strings"
)

// ConsoleFormatter renders console output for human exploration
// (color, tables, emoji). ADR-003 — separates the historically conflated
// TextFormatter responsibilities by moving the rich output into console mode.
// User-output only — not for diagnostic / debug logging.
type ConsoleFormatter struct {
	cfg *Config
}

func (f *ConsoleFormatter) Success(message string, data any) {
	if f.cfg.Quiet {
		if data != nil {
			_, _ = fmt.Fprintf(f.cfg.Out, "%v\n", data)
		}
		return
	}
	_, _ = fmt.Fprintf(f.cfg.Err, "✓ %s\n", message)
	if data != nil && f.cfg.Verbose {
		_, _ = fmt.Fprintf(f.cfg.Out, "%v\n", data)
	}
}

func (f *ConsoleFormatter) Error(message string, category string, hint string) {
	_, _ = fmt.Fprintf(f.cfg.Err, "✗ Error: %s\n", message)
	if category != "" {
		_, _ = fmt.Fprintf(f.cfg.Err, "  Category: %s\n", category)
	}
	if hint != "" {
		_, _ = fmt.Fprintf(f.cfg.Err, "  → %s\n", hint)
	}
}

func (f *ConsoleFormatter) Warning(message string) {
	_, _ = fmt.Fprintf(f.cfg.Err, "⚠ Warning: %s\n", message)
}

func (f *ConsoleFormatter) Table(headers []string, rows [][]string) {
	// compute column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// emit header
	for i, h := range headers {
		_, _ = fmt.Fprintf(f.cfg.Out, "%-*s", widths[i]+2, h)
	}
	_, _ = fmt.Fprintln(f.cfg.Out)

	// separator (box-drawing)
	for _, w := range widths {
		_, _ = fmt.Fprintf(f.cfg.Out, "%s  ", strings.Repeat("─", w))
	}
	_, _ = fmt.Fprintln(f.cfg.Out)

	// data rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				_, _ = fmt.Fprintf(f.cfg.Out, "%-*s", widths[i]+2, cell)
			}
		}
		_, _ = fmt.Fprintln(f.cfg.Out)
	}
}

func (f *ConsoleFormatter) Print(data any) {
	// Prefer ConsoleRenderer, then TextRenderer fallback, otherwise generic %v.
	if r, ok := data.(ConsoleRenderer); ok {
		r.Console(f.cfg.Out, f.cfg.NoColor)
		return
	}
	if r, ok := data.(TextRenderer); ok {
		r.Text(f.cfg.Out)
		return
	}
	_, _ = fmt.Fprintf(f.cfg.Out, "%v\n", data)
}
