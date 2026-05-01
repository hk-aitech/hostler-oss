// Package domain — ANSI helper for the specialized renderers in *_render.go.
//
// Color helpers used when domain Result types implement TextRenderer /
// ConsoleRenderer. Emits ANSI escapes directly without depending on the
// output package. Color is used in Console mode only; Text mode stays plain.
package domain

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
	ansiGray   = "\033[90m"
)

// colorize omits the escape when noColor=true.
func colorize(s, code string, noColor bool) string {
	if noColor {
		return s
	}
	return code + s + ansiReset
}

// statusColor — color per Task / Sprint status.
func statusColor(status string, noColor bool) string {
	switch status {
	case "done", "completed":
		return colorize(status, ansiGreen, noColor)
	case "in-progress", "active":
		return colorize(status, ansiYellow, noColor)
	case "blocked", "error":
		return colorize(status, ansiRed, noColor)
	case "todo", "backlog":
		return colorize(status, ansiGray, noColor)
	default:
		return status
	}
}

// priorityColor — color per p0~p3 priority.
func priorityColor(p string, noColor bool) string {
	switch p {
	case "p0":
		return colorize(p, ansiRed, noColor)
	case "p1":
		return colorize(p, ansiYellow, noColor)
	case "p2":
		return colorize(p, ansiCyan, noColor)
	case "p3":
		return colorize(p, ansiGray, noColor)
	default:
		return p
	}
}

// boldText — bold for headers.
func boldText(s string, noColor bool) string {
	return colorize(s, ansiBold, noColor)
}

// dimText — dim/gray for secondary information.
func dimText(s string, noColor bool) string {
	return colorize(s, ansiGray, noColor)
}
