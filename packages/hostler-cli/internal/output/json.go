package output

import (
	"encoding/json"
	"fmt"
)

// JSONFormatter is the structured-output formatter that responds to the `-o json` flag.
// User-output only (mainly for machine-parsing consumers) — not for
// diagnostic / debug logging.
type JSONFormatter struct {
	cfg *Config
}

func (f *JSONFormatter) Success(message string, data any) {
	output := map[string]any{"status": "ok", "message": message}
	if data != nil {
		output["data"] = data
	}
	f.writeJSON(output)
}

func (f *JSONFormatter) Error(message string, category string, hint string) {
	output := map[string]any{
		"status":  "error",
		"message": message,
	}
	if category != "" {
		output["error_category"] = category
	}
	if hint != "" {
		output["recovery_hint"] = hint
	}
	f.writeJSON(output)
}

func (f *JSONFormatter) Warning(message string) {
	f.writeJSON(map[string]any{"status": "warning", "message": message})
}

func (f *JSONFormatter) Table(headers []string, rows [][]string) {
	items := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		item := make(map[string]string)
		for i, cell := range row {
			if i < len(headers) {
				item[headers[i]] = cell
			}
		}
		items = append(items, item)
	}
	f.writeJSON(items)
}

// Print writes raw data.
// Auto-wraps with the standard envelope (`{status, data, message}`).
// Default behaviour: if the marshaled data already contains a top-level
// "status" key, bypass the wrap; otherwise auto-wrap.
// When `--legacy-envelope` / `HOSTLER_CLI_ENVELOPE=legacy` is active the
// data is emitted raw (legacy behaviour).
func (f *JSONFormatter) Print(data any) {
	if f.cfg.LegacyEnvelope {
		f.writeJSON(data)
		return
	}
	// Convert via marshal → unmarshal so the status check covers both
	// structs and maps. Existing envelope structs (BlockedResponse /
	// OkResponse) bypass the wrap to avoid double-wrapping.
	b, err := json.Marshal(data)
	if err == nil {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			if _, hasStatus := m["status"]; hasStatus {
				f.writeJSON(m)
				return
			}
		}
	}
	// auto-wrap: promote flat list/get responses to the standard envelope.
	f.writeJSON(map[string]any{"status": "ok", "data": data})
}

func (f *JSONFormatter) writeJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(f.cfg.Err, "{\"error\": \"json marshal failed: %s\"}\n", err)
		return
	}
	_, _ = fmt.Fprintf(f.cfg.Out, "%s\n", b)
}
