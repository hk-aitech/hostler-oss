// Package domain — specialized renderers for Sprint result types.
package domain

import (
	"fmt"
	"io"
	"strings"
)

// ── SprintCreateResult ────────────────────────────────────────────────

func (r *SprintCreateResult) Text(w io.Writer) {
	fmt.Fprintf(w, "Created: %s\n", r.SprintID)
	fmt.Fprintf(w, "  Title:  %s\n", r.Title)
	fmt.Fprintf(w, "  Goal:   %s\n", r.Goal)
	fmt.Fprintf(w, "  Status: %s\n", r.Status)
	fmt.Fprintf(w, "  Folder: %s\n", r.FolderPath)
}

func (r *SprintCreateResult) Console(w io.Writer, noColor bool) {
	fmt.Fprintf(w, "%s %s %s\n",
		colorize("✓", ansiGreen, noColor),
		boldText("Created:", noColor),
		r.SprintID,
	)
	fmt.Fprintf(w, "  %s  %s\n", dimText("Title:", noColor), r.Title)
	fmt.Fprintf(w, "  %s   %s\n", dimText("Goal:", noColor), r.Goal)
	fmt.Fprintf(w, "  %s %s\n", dimText("Status:", noColor), statusColor(r.Status, noColor))
	fmt.Fprintf(w, "  %s %s\n", dimText("Folder:", noColor), dimText(r.FolderPath, noColor))
}

// ── SprintStartResult ─────────────────────────────────────────────────

func (r *SprintStartResult) Text(w io.Writer) {
	fmt.Fprintf(w, "Started: %s (status=%s)\n", r.SprintID, r.Status)
	fmt.Fprintf(w, "  Started:  %s\n", r.StartedAt)
	fmt.Fprintf(w, "  Folder:   %s\n", r.FolderPath)
}

func (r *SprintStartResult) Console(w io.Writer, noColor bool) {
	fmt.Fprintf(w, "%s %s %s  %s\n",
		colorize("▶", ansiGreen, noColor),
		boldText("Started:", noColor),
		r.SprintID,
		statusColor(r.Status, noColor),
	)
	fmt.Fprintf(w, "  %s  %s\n", dimText("Started:", noColor), r.StartedAt)
	fmt.Fprintf(w, "  %s   %s\n", dimText("Folder:", noColor), dimText(r.FolderPath, noColor))
}

// ── SprintCompleteResult ──────────────────────────────────────────────

func (r *SprintCompleteResult) Text(w io.Writer) {
	fmt.Fprintf(w, "Completed: %s (status=%s)\n", r.SprintID, r.Status)
	fmt.Fprintf(w, "  Completed: %s\n", r.CompletedAt)
	fmt.Fprintf(w, "  Folder:    %s\n", r.FolderPath)
}

func (r *SprintCompleteResult) Console(w io.Writer, noColor bool) {
	fmt.Fprintf(w, "%s %s %s  %s\n",
		colorize("✓", ansiGreen, noColor),
		boldText("Completed:", noColor),
		r.SprintID,
		statusColor(r.Status, noColor),
	)
	fmt.Fprintf(w, "  %s %s\n", dimText("Completed:", noColor), r.CompletedAt)
	fmt.Fprintf(w, "  %s    %s\n", dimText("Folder:", noColor), dimText(r.FolderPath, noColor))
}

// ── SprintDiscardResult ───────────────────────────────────────────────

func (r *SprintDiscardResult) Text(w io.Writer) {
	fmt.Fprintf(w, "Discarded: %s (%s → %s)\n", r.SprintID, r.PreviousStatus, r.Status)
	fmt.Fprintf(w, "  Reason:   %s\n", r.Reason)
	fmt.Fprintf(w, "  Folder:   %s\n", r.FolderPath)
	if len(r.ReturnedTasks) > 0 {
		fmt.Fprintf(w, "  Returned tasks: %s\n", strings.Join(r.ReturnedTasks, ", "))
	}
}

func (r *SprintDiscardResult) Console(w io.Writer, noColor bool) {
	fmt.Fprintf(w, "%s %s %s  %s→%s\n",
		colorize("✗", ansiYellow, noColor),
		boldText("Discarded:", noColor),
		r.SprintID,
		statusColor(r.PreviousStatus, noColor),
		statusColor(r.Status, noColor),
	)
	fmt.Fprintf(w, "  %s   %s\n", dimText("Reason:", noColor), r.Reason)
	fmt.Fprintf(w, "  %s   %s\n", dimText("Folder:", noColor), dimText(r.FolderPath, noColor))
	if len(r.ReturnedTasks) > 0 {
		fmt.Fprintf(w, "  %s %s\n", dimText("Returned:", noColor), strings.Join(r.ReturnedTasks, ", "))
	}
}

// ── ProgressResult ────────────────────────────────────────────────────

func (r *ProgressResult) Text(w io.Writer) {
	fmt.Fprintf(w, "%s\n", r.SprintID)
	fmt.Fprintf(w, "  Done:        %d\n", r.Done)
	fmt.Fprintf(w, "  In-Progress: %d\n", r.InProgress)
	fmt.Fprintf(w, "  Todo:        %d\n", r.Todo)
	fmt.Fprintf(w, "  Total:       %d (%d%%)\n", r.Total, r.Percent)
}

func (r *ProgressResult) Console(w io.Writer, noColor bool) {
	fmt.Fprintf(w, "%s\n", boldText(r.SprintID, noColor))
	bar := renderProgressBar(r.Percent, 20, noColor)
	fmt.Fprintf(w, "  %s  %s %d/%d (%d%%)\n",
		bar,
		dimText("done:", noColor),
		r.Done, r.Total, r.Percent,
	)
	fmt.Fprintf(w, "  %s %s  %s %s  %s %s\n",
		dimText("done:", noColor), colorize(fmt.Sprintf("%d", r.Done), ansiGreen, noColor),
		dimText("in-progress:", noColor), colorize(fmt.Sprintf("%d", r.InProgress), ansiYellow, noColor),
		dimText("todo:", noColor), colorize(fmt.Sprintf("%d", r.Todo), ansiGray, noColor),
	)
}

// renderProgressBar draws a width-cell progress bar (e.g. [████░░░░]).
func renderProgressBar(percent, width int, noColor bool) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := percent * width / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return "[" + colorize(bar, ansiGreen, noColor) + "]"
}
