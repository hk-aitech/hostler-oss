package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func newJSONFormatter() (*JSONFormatter, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: "json", Out: out, Err: errBuf}
	return &JSONFormatter{cfg: cfg}, out, errBuf
}

func parseJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON parse failed: %v\ninput: %s", err, string(data))
	}
	return result
}

func TestJSONFormatter_Success(t *testing.T) {
	f, out, _ := newJSONFormatter()
	f.Success("done", map[string]string{"key": "value"})

	result := parseJSON(t, out.Bytes())
	if result["status"] != "ok" {
		t.Errorf("status: want ok, got %v", result["status"])
	}
	if result["message"] != "done" {
		t.Errorf("message: want 'done', got %v", result["message"])
	}
	if result["data"] == nil {
		t.Error("data field missing")
	}
}

func TestJSONFormatter_Success_NilData(t *testing.T) {
	f, out, _ := newJSONFormatter()
	f.Success("done", nil)

	result := parseJSON(t, out.Bytes())
	if _, exists := result["data"]; exists {
		t.Errorf("data field present despite nil data: %v", result)
	}
}

func TestJSONFormatter_Error(t *testing.T) {
	f, out, _ := newJSONFormatter()
	f.Error("failure", "NOT_FOUND", "retry")

	result := parseJSON(t, out.Bytes())
	if result["status"] != "error" {
		t.Errorf("status: want error, got %v", result["status"])
	}
	if result["error_category"] != "NOT_FOUND" {
		t.Errorf("error_category: want NOT_FOUND, got %v", result["error_category"])
	}
	if result["recovery_hint"] != "retry" {
		t.Errorf("recovery_hint: want 'retry', got %v", result["recovery_hint"])
	}
}

func TestJSONFormatter_Warning(t *testing.T) {
	f, out, _ := newJSONFormatter()
	f.Warning("watch out")

	result := parseJSON(t, out.Bytes())
	if result["status"] != "warning" {
		t.Errorf("status: want warning, got %v", result["status"])
	}
}

func TestJSONFormatter_Table(t *testing.T) {
	f, out, _ := newJSONFormatter()
	headers := []string{"ID", "Name"}
	rows := [][]string{{"T1", "Alpha"}, {"T2", "Beta"}}
	f.Table(headers, rows)

	var items []map[string]string
	if err := json.Unmarshal(out.Bytes(), &items); err != nil {
		t.Fatalf("Table JSON parse failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("row count: want 2, got %d", len(items))
	}
	if items[0]["ID"] != "T1" || items[1]["Name"] != "Beta" {
		t.Errorf("Table data mismatch: %v", items)
	}
}

// Print performs envelope-standard auto-wrap.
// Top-level data without a "status" is promoted to {status: "ok", data: ...}.
func TestJSONFormatter_Print(t *testing.T) {
	f, out, _ := newJSONFormatter()
	f.Print(map[string]int{"count": 42})

	result := parseJSON(t, out.Bytes())
	if result["status"] != "ok" {
		t.Errorf("envelope status: want 'ok', got %v", result["status"])
	}
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("envelope data field missing: %v", result)
	}
	if data["count"] != float64(42) {
		t.Errorf("data.count: want 42, got %v", data["count"])
	}
}

// When data already carries a "status" key the value is passed through
// without auto-wrap.
func TestJSONFormatter_Print_BypassEnvelopeWhenStatusPresent(t *testing.T) {
	f, out, _ := newJSONFormatter()
	f.Print(map[string]any{"status": "error", "message": "existing"})

	result := parseJSON(t, out.Bytes())
	if result["status"] != "error" {
		t.Errorf("expected the existing status to be preserved, got %v", result["status"])
	}
	if _, nested := result["data"]; nested {
		t.Errorf("auto-wrap must be suppressed when status is already present, got %v", result)
	}
}

// When --legacy-envelope (Config.LegacyEnvelope=true) is set the
// formatter emits raw JSON.
func TestJSONFormatter_Print_LegacyEnvelope(t *testing.T) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cfg := &Config{Format: FormatJSON, Out: out, Err: errBuf, LegacyEnvelope: true}
	f := &JSONFormatter{cfg: cfg}
	f.Print(map[string]int{"count": 42})

	result := parseJSON(t, out.Bytes())
	if result["count"] != float64(42) {
		t.Errorf("legacy mode expected raw passthrough, got %v", result)
	}
	if _, wrapped := result["data"]; wrapped {
		t.Errorf("auto-wrap must be suppressed in legacy mode, got %v", result)
	}
}
