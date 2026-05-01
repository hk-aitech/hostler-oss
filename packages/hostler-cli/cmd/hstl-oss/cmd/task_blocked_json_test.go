// Package cmd - tests for printTaskBlockedJSON.
//
// Verifies that printTaskBlockedJSON emits the BLOCKED payload to stdout
// in JSON mode and produces a structure parseable by jq across all three
// paths (strict result_check / harness gate / hotfix rollback). Regression
// guard for past silent-exit incidents.
package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/ceremony"
)

// fixedTestTime is a constant for deterministic timestamps in unit tests.
var fixedTestTime = time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)

// TestT390_PrintTaskBlockedJSON_StrictResultCheck - when strict
// result_check returns BLOCKED, stdout must carry a JSON payload that
// includes missing_files + policy + recovery_hint.
func TestT390_PrintTaskBlockedJSON_StrictResultCheck(t *testing.T) {
	captured := WithTestOutput(t, "json")
	output.SetCurrentFormat("json")
	t.Cleanup(func() { output.SetCurrentFormat(output.FormatJSON) })

	be := &apperr.BlockedError{
		EntityType: "task",
		EntityID:   "T9999",
		Message:    "1 file listed in the Task Result section is missing from git diff. Missing path: path/missing.go",
		SuggestedActions: []string{
			"Inspect the Result section in the Task body for typos or false paths",
		},
		Extra: map[string]any{
			"reason":        "task_result_unverified",
			"policy":        "strict",
			"missing_files": []string{"path/missing.go"},
			"recovery_hint": "1) check the Result section for typos and fix them, or 2) actually create/modify the missing file, stage it, and retry.",
		},
	}
	cer := &ceremony.TaskCompleteCeremony{
		ChangedFiles: []string{"src/other.go"},
		ResultSection: &ceremony.ResultSectionCheck{
			Exists:       true,
			MissingFiles: []string{"path/missing.go"},
		},
	}

	printTaskBlockedJSON("T9999", be, cer)

	stdout := captured.Stdout.String()
	if stdout == "" {
		t.Fatal("stdout 0 bytes - regression: BLOCKED JSON payload was not emitted to stdout")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse stdout JSON: %v, content=%q", err, stdout)
	}

	if payload["status"] != "blocked" {
		t.Errorf(`expected status="blocked", got=%v`, payload["status"])
	}
	if payload["task_id"] != "T9999" {
		t.Errorf(`expected task_id="T9999", got=%v`, payload["task_id"])
	}
	if payload["reason"] != "task_result_unverified" {
		t.Errorf(`expected reason="task_result_unverified", got=%v`, payload["reason"])
	}

	vr, ok := payload["verification_result"].(map[string]any)
	if !ok {
		t.Fatalf(`verification_result missing or not a map: %T`, payload["verification_result"])
	}
	if vr["policy"] != "strict" {
		t.Errorf(`expected verification_result.policy="strict", got=%v`, vr["policy"])
	}
	missing, ok := vr["missing_files"].([]any)
	if !ok || len(missing) != 1 || missing[0] != "path/missing.go" {
		t.Errorf(`expected verification_result.missing_files=["path/missing.go"], got=%v`, vr["missing_files"])
	}
	if hint, _ := vr["recovery_hint"].(string); hint == "" {
		t.Error(`verification_result.recovery_hint is empty`)
	}

	if _, ok := payload["timestamp"].(string); !ok {
		t.Error(`timestamp field is not a string`)
	}

	cerView, ok := payload["ceremony"].(map[string]any)
	if !ok {
		t.Fatalf(`ceremony field missing: %T`, payload["ceremony"])
	}
	cf, _ := cerView["changed_files"].([]any)
	if len(cf) != 1 || cf[0] != "src/other.go" {
		t.Errorf(`expected ceremony.changed_files=["src/other.go"], got=%v`, cerView["changed_files"])
	}
}

// TestT390_PrintTaskBlockedJSON_HarnessUnchecked - when the Harness Gate
// returns BLOCKED for unchecked items, the payload must include
// unchecked_items plus a suggested_next_action with the harness check
// invocation.
func TestT390_PrintTaskBlockedJSON_HarnessUnchecked(t *testing.T) {
	captured := WithTestOutput(t, "json")
	output.SetCurrentFormat("json")
	t.Cleanup(func() { output.SetCurrentFormat(output.FormatJSON) })

	be := &apperr.BlockedError{
		EntityType: "task",
		EntityID:   "T1001",
		Message:    "2 items remain unchecked",
		UncheckedItems: []apperr.UncheckedItem{
			{ID: "criteria_checked", Name: "Done criteria review", Required: true},
			{ID: "code_review", Name: "Code review", Required: true},
		},
		SuggestedActions: []string{
			"Run harness_get(entity_type='task', entity_id='T1001') to inspect the available item_id values",
		},
		Extra: map[string]any{
			"template_key":       "task:default",
			"available_item_ids": []string{"criteria_checked", "build_passed", "tests_passed", "code_review"},
		},
	}

	printTaskBlockedJSON("T1001", be, nil)

	var payload map[string]any
	if err := json.Unmarshal(captured.Stdout.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	items, ok := payload["unchecked_items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf(`expected unchecked_items length 2, got=%v`, payload["unchecked_items"])
	}
	first, _ := items[0].(map[string]any)
	if first["id"] != "criteria_checked" {
		t.Errorf(`expected unchecked_items[0].id="criteria_checked", got=%v`, first["id"])
	}

	suggested, _ := payload["suggested_next_action"].(string)
	if suggested == "" {
		t.Error("suggested_next_action is empty")
	}
	if !containsAll(suggested, []string{"harness check", "task", "T1001", "criteria_checked"}) {
		t.Errorf(`suggested_next_action missing required tokens: %q`, suggested)
	}

	if _, ok := payload["ceremony"]; ok {
		// ceremony was nil so the field must not be present in the payload.
		t.Error("ceremony field present even though input was nil")
	}
}

// TestT390_PrintTaskBlockedJSON_ConsoleMode_NoStdout - console mode must
// leave stdout empty (no-op). The stderr banner is handled by the caller's
// printBlockedError, so it is not verified here.
func TestT390_PrintTaskBlockedJSON_ConsoleMode_NoStdout(t *testing.T) {
	captured := WithTestOutput(t, "console")
	output.SetCurrentFormat("console")
	t.Cleanup(func() { output.SetCurrentFormat(output.FormatJSON) })

	be := &apperr.BlockedError{
		EntityType: "task",
		EntityID:   "T1002",
		Message:    "blocked",
	}

	printTaskBlockedJSON("T1002", be, nil)

	if out := captured.Stdout.String(); out != "" {
		t.Errorf("console mode must leave stdout empty (no-op), got=%q", out)
	}
}

// TestT390_BuildTaskBlockedPayload_Deterministic - the pure function
// buildTaskBlockedPayload yields a deterministic timestamp from the
// injected time and runs without panicking on empty SuggestedActions /
// empty UncheckedItems.
func TestT390_BuildTaskBlockedPayload_Deterministic(t *testing.T) {
	be := &apperr.BlockedError{
		EntityType: "task",
		EntityID:   "T0",
		Message:    "some failure",
	}
	p := buildTaskBlockedPayload("T0", be, nil, fixedTestTime)

	if p.Status != "blocked" || p.TaskID != "T0" {
		t.Errorf("default field mismatch: %+v", p)
	}
	if p.Timestamp != "2026-04-23T10:00:00Z" {
		t.Errorf(`timestamp not deterministic: %q`, p.Timestamp)
	}
	if p.VerificationResult != nil {
		t.Errorf(`expected verification_result nil when Extra is nil, got=%v`, p.VerificationResult)
	}
	if p.Ceremony != nil {
		t.Errorf(`expected ceremony nil when cer is nil, got=%v`, p.Ceremony)
	}
}

// TestT390_DeriveBlockedReason_Priority - Extra["reason"] > Category > default.
func TestT390_DeriveBlockedReason_Priority(t *testing.T) {
	// Extra["reason"] takes precedence.
	beExtra := &apperr.BlockedError{
		Extra: map[string]any{"reason": "task_result_unverified"},
	}
	if got := deriveBlockedReason(beExtra); got != "task_result_unverified" {
		t.Errorf("Extra reason did not win: %q", got)
	}

	// Without Extra, fall back to Category.
	beCat := &apperr.BlockedError{}
	if got := deriveBlockedReason(beCat); got == "" {
		t.Error("Category fallback failed: empty string")
	}
}

// containsAll reports whether s contains every entry in substrs.
func containsAll(s string, substrs []string) bool {
	for _, ss := range substrs {
		if !strings.Contains(s, ss) {
			return false
		}
	}
	return true
}
