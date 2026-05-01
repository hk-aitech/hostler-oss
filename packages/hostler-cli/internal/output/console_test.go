package output

import (
	"bytes"
	"strings"
	"testing"
)

func newConsoleFormatter() (*ConsoleFormatter, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: FormatConsole, Out: out, Err: errBuf}
	return &ConsoleFormatter{cfg: cfg}, out, errBuf
}

func TestConsoleFormatter_Success(t *testing.T) {
	f, out, errBuf := newConsoleFormatter()
	f.Success("operation complete", "result-data")

	if !strings.Contains(errBuf.String(), "operation complete") {
		t.Errorf("stderr missing success message: %s", errBuf.String())
	}
	if out.Len() != 0 {
		t.Errorf("verbose=false but stdout received data: %s", out.String())
	}
}

func TestConsoleFormatter_Success_Verbose(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: FormatConsole, Out: out, Err: errBuf, Verbose: true}
	f := &ConsoleFormatter{cfg: cfg}

	f.Success("done", "verbose-data")

	if !strings.Contains(out.String(), "verbose-data") {
		t.Errorf("verbose mode missing data on stdout: %s", out.String())
	}
}

func TestConsoleFormatter_Success_Quiet(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: FormatConsole, Out: out, Err: errBuf, Quiet: true}
	f := &ConsoleFormatter{cfg: cfg}

	f.Success("done", "quiet-data")

	if errBuf.Len() != 0 {
		t.Errorf("quiet mode wrote to stderr: %s", errBuf.String())
	}
	if !strings.Contains(out.String(), "quiet-data") {
		t.Errorf("quiet mode missing data on stdout: %s", out.String())
	}
}

func TestConsoleFormatter_Error(t *testing.T) {
	f, _, errBuf := newConsoleFormatter()
	f.Error("operation failed", "NOT_FOUND", "retry task_get")

	got := errBuf.String()
	if !strings.Contains(got, "Error: operation failed") {
		t.Errorf("missing error message: %s", got)
	}
	if !strings.Contains(got, "Category: NOT_FOUND") {
		t.Errorf("missing category: %s", got)
	}
	if !strings.Contains(got, "retry task_get") {
		t.Errorf("missing hint: %s", got)
	}
}

func TestConsoleFormatter_Warning(t *testing.T) {
	f, _, errBuf := newConsoleFormatter()
	f.Warning("warning item")

	if !strings.Contains(errBuf.String(), "Warning: warning item") {
		t.Errorf("missing warning message: %s", errBuf.String())
	}
}

func TestConsoleFormatter_Table(t *testing.T) {
	f, out, _ := newConsoleFormatter()
	headers := []string{"ID", "Title"}
	rows := [][]string{
		{"T001", "first"},
		{"T002", "second"},
	}
	f.Table(headers, rows)

	got := out.String()
	if !strings.Contains(got, "ID") || !strings.Contains(got, "Title") {
		t.Errorf("missing headers: %s", got)
	}
	if !strings.Contains(got, "T001") || !strings.Contains(got, "T002") {
		t.Errorf("missing row data: %s", got)
	}
	if !strings.Contains(got, "─") {
		t.Errorf("missing divider line: %s", got)
	}
}

func TestConsoleFormatter_Print(t *testing.T) {
	f, out, _ := newConsoleFormatter()
	f.Print("raw output")

	if !strings.Contains(out.String(), "raw output") {
		t.Errorf("Print produced no output: %s", out.String())
	}
}
