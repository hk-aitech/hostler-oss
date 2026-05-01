package response_test

import (
	"encoding/json"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/response"
)

func TestOk(t *testing.T) {
	r := response.Ok(map[string]any{"task_id": "T001", "title": "test"})
	if r.Status != "ok" {
		t.Errorf("status field error: %v", r.Status)
	}
	if r.Data["task_id"] != "T001" {
		t.Errorf("task_id field error: %v", r.Data["task_id"])
	}
}

func TestBlocked_DefaultHint(t *testing.T) {
	r := response.Blocked([]string{"build_passed", "tests_passed"}, "")
	if r.Status != "BLOCKED" {
		t.Errorf("status field error: %v", r.Status)
	}
	if len(r.UncheckedItems) != 2 {
		t.Errorf("unchecked_items length error: %d", len(r.UncheckedItems))
	}
	if r.RecoveryHint != "run harness_check" {
		t.Errorf("default recovery_hint error: %v", r.RecoveryHint)
	}
}

func TestBlocked_UserHint(t *testing.T) {
	r := response.Blocked([]string{"build_passed"}, "run the build first")
	if r.RecoveryHint != "run the build first" {
		t.Errorf("user recovery_hint error: %v", r.RecoveryHint)
	}
}

func TestRejected_Default(t *testing.T) {
	r := response.Rejected("missing_required_field", "", nil)
	if r.Status != "REJECTED" {
		t.Errorf("status field error: %v", r.Status)
	}
	if r.Reason != "missing_required_field" {
		t.Errorf("reason field error: %v", r.Reason)
	}
	if r.Suggestion != "" {
		t.Error("suggestion field must be empty")
	}
}

func TestRejected_WithSuggestion(t *testing.T) {
	r := response.Rejected("invalid_state", "use task:complete", nil)
	if r.Suggestion != "use task:complete" {
		t.Errorf("suggestion field error: %v", r.Suggestion)
	}
}

func TestError_Default(t *testing.T) {
	r := response.Error("DB connection failed", "", "", nil)
	if r.Status != "error" {
		t.Errorf("status field error: %v", r.Status)
	}
	if r.ErrorCategory != "INTERNAL_ERROR" {
		t.Errorf("default error_category error: %v", r.ErrorCategory)
	}
	if r.RecoveryHint != "" {
		t.Error("recovery_hint field must be empty")
	}
}

func TestError_AllFields(t *testing.T) {
	r := response.Error("Task not found", "NOT_FOUND", "check via task_list", map[string]any{"task_id": "T999"})
	if r.ErrorCategory != "NOT_FOUND" {
		t.Errorf("error_category field error: %v", r.ErrorCategory)
	}
	if r.RecoveryHint != "check via task_list" {
		t.Errorf("recovery_hint field error: %v", r.RecoveryHint)
	}
	if r.Extra["task_id"] != "T999" {
		t.Errorf("extra field (task_id) error: %v", r.Extra["task_id"])
	}
}

func TestOk_EmptyData(t *testing.T) {
	r := response.Ok(nil)
	if r.Status != "ok" {
		t.Errorf("nil data did not yield status=ok: %v", r.Status)
	}
}

func TestOk_MultipleFields(t *testing.T) {
	r := response.Ok(map[string]any{
		"task_id":  "T001",
		"sprint":   "sprint-01",
		"priority": "p0",
	})
	if r.Status != "ok" {
		t.Errorf("status field error: %v", r.Status)
	}
	if r.Data["sprint"] != "sprint-01" {
		t.Errorf("sprint field error: %v", r.Data["sprint"])
	}
	if r.Data["priority"] != "p0" {
		t.Errorf("priority field error: %v", r.Data["priority"])
	}
	if r.Data["task_id"] != "T001" {
		t.Errorf("task_id field error: %v", r.Data["task_id"])
	}
}

func TestBlocked_EmptyItems(t *testing.T) {
	r := response.Blocked([]string{}, "")
	if r.Status != "BLOCKED" {
		t.Errorf("status field error: %v", r.Status)
	}
	if len(r.UncheckedItems) != 0 {
		t.Errorf("empty unchecked_items length: %d", len(r.UncheckedItems))
	}
}

func TestRejected_Detail(t *testing.T) {
	r := response.Rejected("sprint_already_exists", "",
		map[string]any{"found_location": "completed", "sprint_id": "sprint-01"})
	if r.Status != "REJECTED" {
		t.Errorf("status field error: %v", r.Status)
	}
	if r.Reason != "sprint_already_exists" {
		t.Errorf("reason field error: %v", r.Reason)
	}
	if r.Extra["found_location"] != "completed" {
		t.Errorf("extra field error: %v", r.Extra["found_location"])
	}
}

func TestError_SprintError(t *testing.T) {
	r := response.Error("Sprint not found", "NOT_FOUND", "check via sprint_list",
		map[string]any{"sprint_id": "sprint-99"})
	if r.Status != "error" {
		t.Errorf("status field error: %v", r.Status)
	}
	if r.Extra["sprint_id"] != "sprint-99" {
		t.Errorf("sprint_id field error: %v", r.Extra["sprint_id"])
	}
}

// JSON serialisation backward-compat checks.
func TestOk_JSONCompat(t *testing.T) {
	r := response.Ok(map[string]any{"task_id": "T001"})
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("JSON serialise failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("JSON deserialise failed: %v", err)
	}
	if m["status"] != "ok" {
		t.Errorf("JSON status error: %v", m["status"])
	}
}

func TestBlocked_JSONCompat(t *testing.T) {
	r := response.Blocked([]string{"build_passed"}, "run the build")
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("JSON serialise failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("JSON deserialise failed: %v", err)
	}
	if m["status"] != "BLOCKED" {
		t.Errorf("JSON status error: %v", m["status"])
	}
	if m["recovery_hint"] != "run the build" {
		t.Errorf("JSON recovery_hint error: %v", m["recovery_hint"])
	}
}

func TestError_JSONCompat(t *testing.T) {
	r := response.Error("failed", "NOT_FOUND", "retry", map[string]any{"id": "T1"})
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("JSON serialise failed: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("JSON deserialise failed: %v", err)
	}
	if m["error_category"] != "NOT_FOUND" {
		t.Errorf("JSON error_category error: %v", m["error_category"])
	}
}
