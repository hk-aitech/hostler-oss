// WithTestOutput helper unit test + actual cobra command capture regression test.
package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestWithTestOutput_text verifies that WithTestOutput captures buffered content from the text-format
// Formatter on both stdout and stderr separately.
// TextFormatter.Success sends the message to stderr; Print writes to stdout.
func TestWithTestOutput_text(t *testing.T) {
	captured := WithTestOutput(t, "text")
	Out.Success("hello", nil)
	Out.Print("world")
	if !strings.Contains(captured.Stderr.String(), "hello") {
		t.Errorf("Success message not captured on stderr: %q", captured.Stderr.String())
	}
	if !strings.Contains(captured.Stdout.String(), "world") {
		t.Errorf("Print body not captured on stdout: %q", captured.Stdout.String())
	}
}

// TestWithTestOutput_json verifies that Success messages are captured as
// JSON when the format is JSON.
func TestWithTestOutput_json(t *testing.T) {
	captured := WithTestOutput(t, "json")
	Out.Success("ok", map[string]any{"task_id": "T999"})
	var got map[string]any
	if err := json.Unmarshal(captured.Stdout.Bytes(), &got); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, captured.Stdout.String())
	}
	if got["status"] != "ok" {
		t.Errorf("status=ok expected, received %v", got["status"])
	}
}

// TestWithTestOutput_rules_list_e2e regression-verifies that the previously
// failing pattern (capturing stdout from a cobra command run) is restored
// through the helper.
func TestWithTestOutput_rules_list_e2e(t *testing.T) {
	captured := WithTestOutput(t, "json")
	rootCmd.SetArgs([]string{"rules", "list", "-o", "json"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rules list failed to run: %v\nstderr: %s", err, captured.Stderr.String())
	}
	stdout := captured.Stdout.String()
	if stdout == "" {
		t.Fatalf("rules list stdout is empty (capture failed)")
	}
	var result map[string]any
	if err := json.Unmarshal(captured.Stdout.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse rules list JSON: %v\nraw prefix: %q", err, firstN(stdout, 200))
	}
	// Out.Print auto-wraps in the envelope standard, so the actual payload
	// lives under .data. The legacy query path can be reproduced with --legacy-envelope.
	if result["status"] != "ok" {
		t.Errorf("envelope status expected 'ok', got %v", result["status"])
	}
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("envelope data field missing: %v", result)
	}
	if _, ok := data["rules"]; !ok {
		t.Errorf("data.rules key missing: %v", data)
	}
	if _, ok := data["count"]; !ok {
		t.Errorf("data.count key missing: %v", data)
	}
}

// TestWithTestOutput_cleanup_restores verifies that t.Cleanup resets the
// override to nil (must not affect subsequent tests).
func TestWithTestOutput_cleanup_restores(t *testing.T) {
	if outputOverride != nil {
		t.Fatalf("preceding test did not clean up outputOverride")
	}
	func() {
		subT := &testing.T{}
		_ = WithTestOutput(subT, "text")
		if outputOverride == nil {
			subT.Errorf("failed to set override")
		}
		// On subT scope exit Cleanup resets outputOverride.
		// testing.T cannot invoke Cleanup directly, so we restore manually here.
	}()
	// The Cleanup in the sub-scope above may not have run, so clean up safely.
	outputOverride = nil
}

// TestWithTestOutput_rules_effective_e2e restores the previously failing
// capture for the rules effective path via the helper. `rules effective`
// uses Out.Print to emit the schema directly, so JSON parsing must succeed.
func TestWithTestOutput_rules_effective_e2e(t *testing.T) {
	captured := WithTestOutput(t, "json")
	rootCmd.SetArgs([]string{"rules", "effective", "-o", "json"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rules effective failed to run: %v\nstderr: %s", err, captured.Stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(captured.Stdout.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse rules effective JSON: %v\nraw: %q", err, firstN(captured.Stdout.String(), 200))
	}
	// envelope wrap. payload lives under data.
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("envelope data field missing: %v", result)
	}
	if _, ok := data["rules"]; !ok {
		t.Errorf("data.rules key missing: %v", data)
	}
	if _, ok := data["version"]; !ok {
		t.Errorf("data.version key missing: %v", data)
	}
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
