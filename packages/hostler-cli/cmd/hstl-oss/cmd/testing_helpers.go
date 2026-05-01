// Package cmd - test-only output injection helper.
//
// Background: an earlier `hstl rules` e2e test failed because the cmd
// package's Out (output.Formatter) was constructed in PersistentPreRun with a
// direct reference to os.Stdout, so cobra's rootCmd.SetOut could not capture
// it. The fix introduces an outputOverride hook in root.go and provides a
// helper that lets tests inject buffers.
//
// This helper is only called from _test.go files (shared between unit and
// integration tests), but the filename omits the `_test.go` suffix so other
// test packages can reference it. The runtime effect only triggers when the
// testing.TB argument is non-nil, so it is safe - production binary command
// paths never invoke it.
package cmd

import (
	"bytes"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
)

// CapturedOutput is the stdout/stderr buffer pair returned by WithTestOutput.
type CapturedOutput struct {
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}

// WithTestOutput replaces the cmd package's global outputOverride with a
// buffer-backed Formatter for tests, restoring it automatically when the test
// ends. rootCmd.Execute detects outputOverride during PersistentPreRun and
// reuses it.
//
// format must be "text" or "json". Remaining Config fields use defaults.
//
// Example usage:
//
//	captured := cmd.WithTestOutput(t, "json")
//	rootCmd.SetArgs([]string{"rules", "list", "-o", "json"})
//	if err := rootCmd.Execute(); err != nil { t.Fatal(err) }
//	var result map[string]any
//	if err := json.Unmarshal(captured.Stdout.Bytes(), &result); err != nil { ... }
func WithTestOutput(tb testing.TB, format string) *CapturedOutput {
	tb.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cfg := &output.Config{
		Format: format,
		Out:    stdout,
		Err:    stderr,
	}
	formatter := output.New(cfg)
	prevOverride := outputOverride
	prevOut := Out
	outputOverride = formatter
	// Set Out immediately for unit tests that call Out.Success directly.
	// In the cobra.Execute path PersistentPreRun detects outputOverride and
	// assigns the same value.
	Out = formatter
	tb.Cleanup(func() {
		outputOverride = prevOverride
		Out = prevOut
	})
	return &CapturedOutput{Stdout: stdout, Stderr: stderr}
}
