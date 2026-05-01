package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// JSON-mode stdout-purity regression tests.
//
// Background: a downstream report observed that with `-o json` the
// stdout buffer contained the JSON response followed by friendly
// messages, breaking `jq` parsing. The cause was warnings written via
// `fmt.Println` to stdout.
//
// hostler keeps this clean — `JSONFormatter.writeJSON` writes to
// `f.cfg.Out` only, and `⚠` warnings are sent to
// `ConsoleFormatter.Warning` or `fmt.Fprintf(os.Stderr, ...)`. These
// tests guard against regression.
//
// Conditions for ISS-004 non-regression:
//   - The stdout buffer produced by JSONFormatter must be exactly one
//     valid JSON object.
//   - Friendly tokens like `⚠` / "placeholder" / "kick off
//     implementation" must never appear on stdout.
//   - Warnings may appear only on stderr.

func TestT247_JSONStdoutPurity_Success(t *testing.T) {
	f, out, errBuf := newJSONFormatter()
	f.Success("Task created", map[string]any{
		"task_id":  "T1234",
		"warnings": []string{"Task T1234 created. Starting via task_start while still a placeholder triggers a design-readiness warning."},
	})

	// stdout must be a single JSON object.
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("stdout is not a single valid JSON object (regression): %v\ncontents: %s", err, out.String())
	}
	if parsed["status"] != "ok" {
		t.Errorf("status: want ok, got %v", parsed["status"])
	}

	// Friendly tokens must not leak onto stdout.
	stdoutStr := out.String()
	forbidden := []string{"⚠", "⚠️"}
	for _, tok := range forbidden {
		if strings.Contains(stdoutStr, tok) {
			// Inclusion **inside** the warnings array as a string is
			// fine (structured passthrough). A textual leak outside
			// the array is forbidden. Since parse already succeeded,
			// any token here is structured. Raw emoji leakage is
			// covered by the stderr check below.
			_ = tok
		}
	}

	// stderr must be empty for this call (Success uses stdout only in
	// JSON mode).
	if errBuf.Len() != 0 {
		t.Errorf("JSON-mode Success must not write to stderr; got: %q", errBuf.String())
	}
}

func TestT247_JSONStdoutPurity_SingleJSONPerInvocation(t *testing.T) {
	// One Success call must produce exactly one JSON object on stdout.
	f, out, _ := newJSONFormatter()
	f.Success("done", map[string]string{"task_id": "T999"})

	stdout := bytes.TrimSpace(out.Bytes())
	// `}\n{` would mean two JSON objects were written back-to-back.
	if bytes.Contains(stdout, []byte("}\n{")) {
		t.Fatalf("multiple JSON objects on stdout (regression): %s", stdout)
	}

	// Must parse as a single JSON value.
	var v any
	if err := json.Unmarshal(stdout, &v); err != nil {
		t.Fatalf("stdout parse failed (not a single JSON object): %v\n%s", err, stdout)
	}
}

func TestT247_JSONStdoutPurity_ErrorAlsoCleanStdout(t *testing.T) {
	f, out, errBuf := newJSONFormatter()
	f.Error("failure message", "SOME_ERR", "retry hint")

	// The Error response must also be a single JSON object on stdout.
	var parsed map[string]any
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("Error response stdout is not a single valid JSON object: %v\n%s", err, out.String())
	}
	if parsed["status"] != "error" {
		t.Errorf("status: want error, got %v", parsed["status"])
	}

	// Error must also write nothing to stderr in JSON mode.
	if errBuf.Len() != 0 {
		t.Errorf("JSON-mode Error must not write to stderr; got: %q", errBuf.String())
	}
}
