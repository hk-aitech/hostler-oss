// Package cmd - tests for printSprintBlockedJSON.
//
// Verifies that printSprintBlockedJSON writes the JSON payload on stdout,
// keeps it separated from stderr, and prevents the past regression where
// stdout was 0 bytes on sprint completion failure.
package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
)

// TestT355_PrintSprintBlockedJSON_StdoutPayload - on BLOCKED, stdout must
// carry status=blocked + missing_phases + suggested_next_action as JSON, and
// the result must be parseable with jq.
func TestT355_PrintSprintBlockedJSON_StdoutPayload(t *testing.T) {
	captured := WithTestOutput(t, "json")

	state := &domain.HarnessGetResult{
		EntityType:  "sprint",
		EntityID:   "sprint-28",
		Blocked:    true,
		RequiredTotal: 3,
		RequiredDone: 1,
		Items: []domain.HarnessItemView{
			{ID: "phase1_doc_review", Name: "Phase 1: doc-review", Required: true, Done: false},
			{ID: "phase5_retro", Name: "Phase 5: retro (KPT)", Required: true, Done: false},
			{ID: "phase3_audit", Name: "Phase 3: work-audit", Required: true, Done: true},
		},
	}

	printSprintBlockedJSON("sprint-28", state)

	stdout := captured.Stdout.String()
	if stdout == "" {
		t.Fatal("stdout 0 bytes - regression: BLOCKED stdout must not be empty")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to parse stdout JSON: %v, content=%q", err, stdout)
	}

	if payload["status"] != "blocked" {
		t.Errorf(`expected payload["status"]="blocked", got=%v`, payload["status"])
	}
	if payload["sprint_id"] != "sprint-28" {
		t.Errorf(`expected payload["sprint_id"]="sprint-28", got=%v`, payload["sprint_id"])
	}

	missing, ok := payload["missing_phases"].([]any)
	if !ok {
		t.Fatalf(`payload["missing_phases"] is not an array: %T`, payload["missing_phases"])
	}
	if len(missing) != 2 {
		t.Errorf("expected missing_phases length 2, got=%d", len(missing))
	}

	// Done=true items (phase3_audit) must be excluded.
	first, _ := missing[0].(map[string]any)
	if first["id"] != "phase1_doc_review" {
		t.Errorf(`expected missing_phases[0].id="phase1_doc_review", got=%v`, first["id"])
	}

	suggested, _ := payload["suggested_next_action"].(string)
	if !strings.Contains(suggested, "phase1_doc_review") {
		t.Errorf(`expected suggested_next_action to contain the first unchecked phase, got=%q`, suggested)
	}
	if !strings.Contains(suggested, "sprint-28") {
		t.Errorf(`expected suggested_next_action to contain the sprint_id, got=%q`, suggested)
	}
}

// TestT355_PrintSprintBlockedJSON_EmptyMissing - when every Phase is
// complete, missing_phases must be an empty array and
// suggested_next_action an empty string (defensive invocation must produce
// JSON without panicking).
func TestT355_PrintSprintBlockedJSON_EmptyMissing(t *testing.T) {
	captured := WithTestOutput(t, "json")

	state := &domain.HarnessGetResult{
		EntityID: "sprint-99",
		Items:  []domain.HarnessItemView{},
	}

	printSprintBlockedJSON("sprint-99", state)

	var payload map[string]any
	if err := json.Unmarshal(captured.Stdout.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	missing, _ := payload["missing_phases"].([]any)
	if len(missing) != 0 {
		t.Errorf("expected missing_phases length 0, got=%d", len(missing))
	}
	if payload["suggested_next_action"] != "" {
		t.Errorf(`expected suggested_next_action="", got=%v`, payload["suggested_next_action"])
	}
}
