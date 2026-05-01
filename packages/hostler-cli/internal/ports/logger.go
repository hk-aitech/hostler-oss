// Package ports — Logger port.
//
// The CLI previously had no structured logging library; each call site
// used ad hoc `log.Printf` formatting, which hurt parsing and
// observability. Consolidates onto a slog-based adapter while requiring
// callers to know only this port interface.
//
// No dedicated method is exposed for "log and exit" paths à la log.Fatal —
// callers should use the two-step pattern of logging at Error and then
// calling os.Exit (avoids polluting the logger).
package ports

// Logger — structured-logging port.
// attrs follow slog convention as alternating (key, value) pairs, e.g.
// ("task_id", "T100").
type Logger interface {
	Debug(msg string, attrs ...any)
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, attrs ...any)
}
