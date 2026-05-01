package log_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
)

// pkg/log adapter unit tests.

func TestT800_Logger_InfoLevelDefault(t *testing.T) {
	var buf bytes.Buffer
	lg := log.NewWith(&buf, slog.LevelInfo)
	lg.Info("task started", "task_id", "T100")
	out := buf.String()
	if !strings.Contains(out, "task started") {
		t.Errorf("Info message missing: %q", out)
	}
	if !strings.Contains(out, "task_id=T100") {
		t.Errorf("structured attr missing: %q", out)
	}
}

func TestT800_Logger_DebugFilteredByLevel(t *testing.T) {
	var buf bytes.Buffer
	lg := log.NewWith(&buf, slog.LevelWarn)
	lg.Debug("should be filtered")
	lg.Info("should be filtered too")
	lg.Warn("this should appear")
	out := buf.String()
	if strings.Contains(out, "should be filtered") {
		t.Errorf("Debug/Info printed at Warn level: %q", out)
	}
	if !strings.Contains(out, "this should appear") {
		t.Errorf("Warn message missing: %q", out)
	}
}

func TestT800_Logger_EnvVarNamePrefix(t *testing.T) {
	// In the hostler fork, brand.EnvPrefix is HOSTLER_, so LogLevelEnvVar
	// is "HOSTLER_LOG_LEVEL". The contract is that the value must be
	// derived dynamically from brand.EnvPrefix rather than a hardcoded string.
	wantSuffix := "LOG_LEVEL"
	if !strings.HasSuffix(log.LogLevelEnvVar, wantSuffix) {
		t.Errorf("LogLevelEnvVar = %q, want suffix %q", log.LogLevelEnvVar, wantSuffix)
	}
}

func TestT800_Logger_NewDefault_DoesNotPanic(t *testing.T) {
	// The default logger writes to os.Stderr so content checks are hard.
	// Confirm the instance is created without panic/error (smoke test).
	lg := log.NewDefault()
	if lg == nil {
		t.Fatal("NewDefault() = nil")
	}
	lg.Info("smoke test from T800")
}
