// Package log — slog-based Logger adapter.
//
// Design principles:
// - wrap slog.Logger to implement ports.Logger.
// - emit through a **shared global writer** (atomic swappable).
// Default writer = os.Stderr. In JSON-output mode the
// root.go PersistentPreRunE swaps to SetGlobalOutput(io.Discard) to
// keep the stdout JSON payload free of stray writes.
// - level controlled by `HSTL_LOG_LEVEL` env var (debug/info/warn/error,
// default info).
// - format defaults to slog text handler (key=value); future Tasks may
// extend with `HSTL_LOG_FORMAT=json`.
// - JSON-mode debugging: `HOSTLER_LOG=stderr` env var disables the swap.
//
// Callers obtain instances via NewDefault() or, for tests,
// NewWith(handler) to inject an isolated handler.
package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// LogLevelEnvVar — env var that controls the log level (debug/info/warn/error).
// Follows the user-facing env-prefix convention (HSTL_). Declared via var
// instead of const because Go const cannot store the result of a function
// call or runtime concatenation, even when the inputs are themselves const.
var LogLevelEnvVar = brand.EnvPrefix + "LOG_LEVEL"

// LogOverrideEnvVar — opt-in env var that keeps Logger output even in JSON
// mode. When `HOSTLER_LOG=stderr` is set, SetGlobalOutput(io.Discard) is a
// no-op so debug logs continue to reach stderr.
const LogOverrideEnvVar = "HOSTLER_LOG"

// writerHolder wraps io.Writer for atomic.Pointer type consistency. Using
// atomic.Pointer[writerHolder] (instead of atomic.Value) avoids the
// concrete-type-mismatch panic that atomic.Value triggers when a different
// concrete type is stored after the first Store.
type writerHolder struct{ w io.Writer }

// globalWriter is the atomic writer holder shared by every NewDefault() Logger.
// JSON-mode confirmation in root.go swaps it to SetGlobalOutput(io.Discard).
var globalWriter atomic.Pointer[writerHolder]

// defaultStderrWriter is a marker writer that resolves os.Stderr lazily on
// every Write. Compatible with tests (e.g. Test) that reassign
// os.Stderr to a pipe.
type defaultStderrWriter struct{}

func (defaultStderrWriter) Write(p []byte) (int, error) { return os.Stderr.Write(p) }

var defaultStderr io.Writer = defaultStderrWriter{}

func init() {
	globalWriter.Store(&writerHolder{w: defaultStderr})
}

// SetGlobalOutput swaps the global Logger writer at runtime.
// At JSON-mode confirmation (root.go PersistentPreRunE), io.Discard is injected.
// When HOSTLER_LOG=stderr is set, the call is a no-op (debug opt-in).
// Passing nil or os.Stderr restores the defaultStderr marker (lazy reference).
func SetGlobalOutput(w io.Writer) {
	if strings.EqualFold(os.Getenv(LogOverrideEnvVar), "stderr") {
		return
	}
	if w == nil || w == os.Stderr {
		w = defaultStderr
	}
	globalWriter.Store(&writerHolder{w: w})
}

// ResetGlobalOutput — for tests / re-initialisation. Swaps the writer
// unconditionally. nil or os.Stderr restores the defaultStderr marker.
func ResetGlobalOutput(w io.Writer) {
	if w == nil || w == os.Stderr {
		w = defaultStderr
	}
	globalWriter.Store(&writerHolder{w: w})
}

// globalProxy is the writer proxy that re-reads globalWriter on every
// Write. slog handlers bind a writer at creation time, so without the
// proxy the initial value (os.Stderr) would be permanently fixed. When
// the value is the defaultStderr marker, it resolves os.Stderr live.
type globalProxy struct{}

func (globalProxy) Write(p []byte) (int, error) {
	h := globalWriter.Load()
	if h == nil || h.w == nil {
		return defaultStderr.Write(p)
	}
	return h.w.Write(p)
}

// Logger implements ports.Logger.
type Logger struct {
	slog *slog.Logger
}

// Compile marker.
var _ ports.Logger = (*Logger)(nil)

// NewDefault constructs the default Logger with globalProxy + env-driven level.
// Runtime SetGlobalOutput swaps to io.Discard in JSON mode.
// Not idempotent — callers manage instance lifetime; composition.go calls it once.
func NewDefault() *Logger {
	return NewWith(globalProxy{}, parseLevel(os.Getenv(LogLevelEnvVar)))
}

// NewWith creates a Logger with a custom writer + level (test / DI).
// A custom writer is unaffected by SetGlobalOutput.
func NewWith(w io.Writer, level slog.Level) *Logger {
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	return &Logger{slog: slog.New(h)}
}

// parseLevel converts an env string to slog.Level. Empty / invalid -> info.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	}
	return slog.LevelInfo
}

// Debug / Info / Warn / Error delegate to slog.
func (l *Logger) Debug(msg string, attrs ...any) { l.slog.Debug(msg, attrs...) }
func (l *Logger) Info(msg string, attrs ...any) { l.slog.Info(msg, attrs...) }
func (l *Logger) Warn(msg string, attrs ...any) { l.slog.Warn(msg, attrs...) }
func (l *Logger) Error(msg string, attrs ...any) { l.slog.Error(msg, attrs...) }
