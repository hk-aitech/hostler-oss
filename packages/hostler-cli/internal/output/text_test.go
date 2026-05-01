package output

import (
	"bytes"
	"strings"
	"testing"
)

func newTextFormatter() (*TextFormatter, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: FormatText, Out: out, Err: errBuf}
	return &TextFormatter{cfg: cfg}, out, errBuf
}

// asciiOnly is a heuristic that ensures plain-text output does not
// contain emojis, box-drawing chars, or ANSI escapes. The core
// invariant of plain mode.
func asciiOnly(s string) bool {
	for _, r := range s {
		if r > 0x7F {
			return false
		}
		if r == 0x1B { // ANSI escape
			return false
		}
	}
	return true
}

func TestTextFormatter_Success(t *testing.T) {
	f, _, errBuf := newTextFormatter()
	f.Success("operation complete", "result-data")

	got := errBuf.String()
	if !strings.Contains(got, "OK: operation complete") {
		t.Errorf("stderr missing OK prefix: %s", got)
	}
	if strings.Contains(got, "✓") {
		t.Errorf("plain mode but emoji present: %s", got)
	}
}

func TestTextFormatter_Success_Verbose(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: FormatText, Out: out, Err: errBuf, Verbose: true}
	f := &TextFormatter{cfg: cfg}

	f.Success("done", "verbose-data")

	if !strings.Contains(out.String(), "verbose-data") {
		t.Errorf("verbose mode missing data on stdout: %s", out.String())
	}
}

func TestTextFormatter_Success_Quiet(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: FormatText, Out: out, Err: errBuf, Quiet: true}
	f := &TextFormatter{cfg: cfg}

	f.Success("done", "quiet-data")

	if errBuf.Len() != 0 {
		t.Errorf("quiet mode wrote to stderr: %s", errBuf.String())
	}
	if !strings.Contains(out.String(), "quiet-data") {
		t.Errorf("quiet mode missing data on stdout: %s", out.String())
	}
}

func TestTextFormatter_Error(t *testing.T) {
	f, _, errBuf := newTextFormatter()
	f.Error("failed message", "NOT_FOUND", "retry task_get")

	got := errBuf.String()
	if !strings.Contains(got, "ERROR: failed message") {
		t.Errorf("missing ERROR prefix: %s", got)
	}
	if !strings.Contains(got, "category: NOT_FOUND") {
		t.Errorf("missing category: %s", got)
	}
	if !strings.Contains(got, "hint: retry task_get") {
		t.Errorf("missing hint: %s", got)
	}
	if strings.Contains(got, "✗") || strings.Contains(got, "→") {
		t.Errorf("plain mode but emoji/arrow present: %s", got)
	}
	if !asciiOnly(got) {
		t.Errorf("plain mode but non-ASCII characters present: %s", got)
	}
}

func TestTextFormatter_Warning(t *testing.T) {
	f, _, errBuf := newTextFormatter()
	f.Warning("plain warning")

	got := errBuf.String()
	if !strings.Contains(got, "WARN: plain warning") {
		t.Errorf("missing WARN prefix: %s", got)
	}
	if strings.Contains(got, "⚠") {
		t.Errorf("plain mode but emoji present: %s", got)
	}
}

func TestTextFormatter_Table_PlainTabSeparated(t *testing.T) {
	f, out, _ := newTextFormatter()
	headers := []string{"ID", "Title"}
	rows := [][]string{
		{"T001", "first"},
		{"T002", "second"},
	}
	f.Table(headers, rows)

	got := out.String()
	// tab-separated check
	if !strings.Contains(got, "ID\tTitle") {
		t.Errorf("header is not tab-separated: %q", got)
	}
	if !strings.Contains(got, "T001\tfirst") {
		t.Errorf("row is not tab-separated: %q", got)
	}
	// box-drawing forbidden
	if strings.Contains(got, "─") {
		t.Errorf("plain mode but box-drawing present: %s", got)
	}
}

func TestTextFormatter_Print(t *testing.T) {
	f, out, _ := newTextFormatter()
	f.Print("raw output")

	if !strings.Contains(out.String(), "raw output") {
		t.Errorf("Print produced no output: %s", out.String())
	}
}
