// Package domain — specialized renderers for Task result types.
//
// Implements TextRenderer / ConsoleRenderer methods on the domain Result
// structs. The output package detects them via type assertion and replaces
// the generic %v fallback. JSON output is unchanged (the formatter branches separately).
package domain

import (
	"fmt"
	"io"
	"strings"
)

// ── GetResult ──────────────────────────────────────────────────────────

func (r *GetResult) Text(w io.Writer) {
	fmt.Fprintf(w, "%s  %s\n", r.TaskID, r.Title)
	fmt.Fprintf(w, "  Type:     %s\n", r.Type)
	fmt.Fprintf(w, "  Status:   %s\n", r.Status)
	fmt.Fprintf(w, "  Priority: %s\n", r.Priority)
	fmt.Fprintf(w, "  Estimate: %s\n", r.Estimate)
	if r.Sprint != nil {
		fmt.Fprintf(w, "  Sprint:   %v\n", r.Sprint)
	}
	if len(r.DependsOn) > 0 {
		fmt.Fprintf(w, "  Depends:  %s\n", strings.Join(r.DependsOn, ", "))
	}
	fmt.Fprintf(w, "  File:     %s\n", r.FilePath)
}

func (r *GetResult) Console(w io.Writer, noColor bool) {
	fmt.Fprintf(w, "%s  %s\n",
		boldText(r.TaskID, noColor),
		r.Title,
	)
	fmt.Fprintf(w, "  %s     %s\n", dimText("Type:", noColor), r.Type)
	fmt.Fprintf(w, "  %s   %s\n", dimText("Status:", noColor), statusColor(r.Status, noColor))
	fmt.Fprintf(w, "  %s %s\n", dimText("Priority:", noColor), priorityColor(r.Priority, noColor))
	fmt.Fprintf(w, "  %s %s\n", dimText("Estimate:", noColor), r.Estimate)
	if r.Sprint != nil {
		fmt.Fprintf(w, "  %s   %v\n", dimText("Sprint:", noColor), r.Sprint)
	}
	if len(r.DependsOn) > 0 {
		fmt.Fprintf(w, "  %s  %s\n", dimText("Depends:", noColor), strings.Join(r.DependsOn, ", "))
	}
	fmt.Fprintf(w, "  %s     %s\n", dimText("File:", noColor), dimText(r.FilePath, noColor))
}

// ── ListResult ─────────────────────────────────────────────────────────

func (r *ListResult) Text(w io.Writer) {
	if len(r.Tasks) == 0 {
		fmt.Fprintln(w, "(no tasks)")
		return
	}
	for _, t := range r.Tasks {
		sprint := "-"
		if t.Sprint != nil {
			sprint = fmt.Sprintf("%v", t.Sprint)
		}
		fmt.Fprintf(w, "%-6s  %-10s  %-12s  %-4s  %-12s  %s\n",
			t.TaskID, t.Type, t.Status, t.Priority, sprint, t.Title,
		)
	}
	fmt.Fprintf(w, "\nTotal: %d\n", r.Count)
}

func (r *ListResult) Console(w io.Writer, noColor bool) {
	if len(r.Tasks) == 0 {
		fmt.Fprintln(w, dimText("(no tasks)", noColor))
		return
	}
	header := fmt.Sprintf("%-6s  %-10s  %-12s  %-4s  %-12s  %s",
		"ID", "Type", "Status", "Pri", "Sprint", "Title")
	fmt.Fprintln(w, boldText(header, noColor))
	fmt.Fprintln(w, dimText(strings.Repeat("─", 80), noColor))
	for _, t := range r.Tasks {
		sprint := "-"
		if t.Sprint != nil {
			sprint = fmt.Sprintf("%v", t.Sprint)
		}
		fmt.Fprintf(w, "%-6s  %-10s  %-21s  %-13s  %-12s  %s\n",
			t.TaskID, t.Type,
			statusColor(t.Status, noColor),
			priorityColor(t.Priority, noColor),
			sprint, t.Title,
		)
	}
	fmt.Fprintf(w, "\n%s %d\n", dimText("Total:", noColor), r.Count)
}

// ── CreateResult ───────────────────────────────────────────────────────

func (r *CreateResult) Text(w io.Writer) {
	fmt.Fprintf(w, "Created: %s (status=%s)\n", r.TaskID, r.Status)
	fmt.Fprintf(w, "  File:    %s\n", r.FilePath)
	fmt.Fprintf(w, "  Created: %s\n", r.CreatedAt)
	for _, warn := range r.Warnings {
		fmt.Fprintf(w, "  ⚠ %s\n", warn)
	}
}

func (r *CreateResult) Console(w io.Writer, noColor bool) {
	fmt.Fprintf(w, "%s %s %s\n",
		colorize("✓", ansiGreen, noColor),
		boldText("Created:", noColor),
		r.TaskID,
	)
	fmt.Fprintf(w, "  %s    %s\n", dimText("File:", noColor), dimText(r.FilePath, noColor))
	fmt.Fprintf(w, "  %s  %s\n", dimText("Status:", noColor), statusColor(r.Status, noColor))
	for _, warn := range r.Warnings {
		fmt.Fprintf(w, "  %s %s\n", colorize("⚠", ansiYellow, noColor), warn)
	}
}

// ── TransitionResult (shared by start/complete/reopen) ────────────────

func (r *TransitionResult) Text(w io.Writer) {
	fmt.Fprintf(w, "%s: %s → %s\n", r.TaskID, r.PreviousStatus, r.NewStatus)
	fmt.Fprintf(w, "  Title:    %s\n", r.Title)
	fmt.Fprintf(w, "  Type:     %s (commit prefix: %s)\n", r.Type, r.CommitPrefix)
	fmt.Fprintf(w, "  Estimate: %s  Priority: %s\n", r.Estimate, r.Priority)
	if r.Sprint != nil {
		fmt.Fprintf(w, "  Sprint:   %v\n", r.Sprint)
	}
	if r.WorkTicket != "" {
		fmt.Fprintf(w, "  Ticket:   %s\n", r.WorkTicket)
	}
	for _, rem := range r.Reminders {
		fmt.Fprintf(w, "  • %s\n", rem)
	}
	if r.SuggestedNextAction != "" {
		fmt.Fprintf(w, "\n→ %s\n", r.SuggestedNextAction)
	}
}

func (r *TransitionResult) Console(w io.Writer, noColor bool) {
	arrow := colorize("→", ansiCyan, noColor)
	fmt.Fprintf(w, "%s  %s %s %s\n",
		boldText(r.TaskID, noColor),
		statusColor(r.PreviousStatus, noColor),
		arrow,
		statusColor(r.NewStatus, noColor),
	)
	fmt.Fprintf(w, "  %s    %s\n", dimText("Title:", noColor), r.Title)
	fmt.Fprintf(w, "  %s     %s  %s %s\n",
		dimText("Type:", noColor), r.Type,
		dimText("commit:", noColor), r.CommitPrefix,
	)
	fmt.Fprintf(w, "  %s %s  %s %s\n",
		dimText("Estimate:", noColor), r.Estimate,
		dimText("Pri:", noColor), priorityColor(r.Priority, noColor),
	)
	if r.Sprint != nil {
		fmt.Fprintf(w, "  %s   %v\n", dimText("Sprint:", noColor), r.Sprint)
	}
	if r.WorkTicket != "" {
		fmt.Fprintf(w, "  %s   %s\n", dimText("Ticket:", noColor), dimText(r.WorkTicket, noColor))
	}
	for _, rem := range r.Reminders {
		fmt.Fprintf(w, "  %s %s\n", colorize("•", ansiYellow, noColor), rem)
	}
	if r.SuggestedNextAction != "" {
		fmt.Fprintf(w, "\n%s %s\n", colorize("→", ansiGreen, noColor), r.SuggestedNextAction)
	}
}

// ── ReopenResult ───────────────────────────────────────────────────────

func (r *ReopenResult) Text(w io.Writer) {
	fmt.Fprintf(w, "%s: reopened (%s → %s)\n", r.TaskID, r.PreviousStatus, r.NewStatus)
	fmt.Fprintf(w, "  Reason:  %s\n", r.Reason)
	fmt.Fprintf(w, "  History: appended=%v  HarnessReset: %v\n", r.HistoryAppended, r.HarnessReset)
}

func (r *ReopenResult) Console(w io.Writer, noColor bool) {
	arrow := colorize("→", ansiCyan, noColor)
	fmt.Fprintf(w, "%s  %s %s %s  %s\n",
		boldText(r.TaskID, noColor),
		statusColor(r.PreviousStatus, noColor),
		arrow,
		statusColor(r.NewStatus, noColor),
		colorize("(reopened)", ansiYellow, noColor),
	)
	fmt.Fprintf(w, "  %s  %s\n", dimText("Reason:", noColor), r.Reason)
	fmt.Fprintf(w, "  %s appended=%v  harness_reset=%v\n",
		dimText("History:", noColor), r.HistoryAppended, r.HarnessReset)
}
