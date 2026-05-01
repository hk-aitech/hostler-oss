// Package domain — Harness Result specialized renderers.
package domain

import (
	"fmt"
	"io"
	"strings"
)

// ── HarnessGetResult ──────────────────────────────────────────────────

func (r *HarnessGetResult) Text(w io.Writer) {
	fmt.Fprintf(w, "%s/%s  template=%s  required=%d/%d  blocked=%v\n",
		r.EntityType, r.EntityID, r.TemplateKey,
		r.RequiredDone, r.RequiredTotal, r.Blocked,
	)
	for _, item := range r.Items {
		mark := "[ ]"
		if item.Done {
			mark = "[x]"
		}
		req := "(opt)"
		if item.Required {
			req = "(req)"
		}
		fmt.Fprintf(w, "  %s %-30s %s %s\n", mark, item.ID, req, item.Name)
	}
}

func (r *HarnessGetResult) Console(w io.Writer, noColor bool) {
	blockedTag := colorize("✓", ansiGreen, noColor)
	if r.Blocked {
		blockedTag = colorize("✗", ansiRed, noColor)
	}
	fmt.Fprintf(w, "%s %s/%s  %s %s  %s %d/%d  %s %s\n",
		blockedTag,
		boldText(r.EntityType, noColor), boldText(r.EntityID, noColor),
		dimText("template:", noColor), r.TemplateKey,
		dimText("required:", noColor), r.RequiredDone, r.RequiredTotal,
		dimText("blocked:", noColor), boolColor(r.Blocked, noColor),
	)
	fmt.Fprintln(w, dimText(strings.Repeat("─", 70), noColor))
	for _, item := range r.Items {
		mark := colorize("[ ]", ansiGray, noColor)
		if item.Done {
			mark = colorize("[x]", ansiGreen, noColor)
		}
		req := dimText("(opt)", noColor)
		if item.Required {
			req = colorize("(req)", ansiYellow, noColor)
		}
		fmt.Fprintf(w, "  %s %-30s %s  %s\n", mark, item.ID, req, item.Name)
	}
}

// boolColor — true=green, false=red.
func boolColor(b bool, noColor bool) string {
	if b {
		return colorize("true", ansiRed, noColor) // blocked=true is the danger state
	}
	return colorize("false", ansiGreen, noColor)
}
