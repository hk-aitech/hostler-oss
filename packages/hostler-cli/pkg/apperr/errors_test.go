package apperr_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// ---------------------------------------------------------------------------
// NotFoundError
// ---------------------------------------------------------------------------

func TestNotFoundError_Error_WithMessage(t *testing.T) {
	err := &apperr.NotFoundError{
		EntityType: "task",
		EntityID:   "T001",
		Message:    "Task not found.",
	}
	if got := err.Error(); got != "Task not found." {
		t.Errorf("Error() = %q, want %q", got, "Task not found.")
	}
}

func TestNotFoundError_Error_DefaultMessage(t *testing.T) {
	err := &apperr.NotFoundError{
		EntityType: "task",
		EntityID:   "T001",
	}
	want := "task T001 not found."
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestNotFoundError_Category(t *testing.T) {
	err := &apperr.NotFoundError{}
	if got := err.Category(); got != "NOT_FOUND" {
		t.Errorf("Category() = %q, want NOT_FOUND", got)
	}
}

func TestNotFoundError_ErrorsAs(t *testing.T) {
	base := error(&apperr.NotFoundError{EntityType: "task", EntityID: "T099", RecoveryHint: "check task_list."})
	wrapped := fmt.Errorf("outer: %w", base)

	var nfe *apperr.NotFoundError
	if !errors.As(wrapped, &nfe) {
		t.Fatal("errors.As failed for NotFoundError")
	}
	if nfe.EntityID != "T099" {
		t.Errorf("EntityID = %q, want T099", nfe.EntityID)
	}
	if nfe.RecoveryHint != "check task_list." {
		t.Errorf("RecoveryHint = %q", nfe.RecoveryHint)
	}
}

// ---------------------------------------------------------------------------
// BlockedError
// ---------------------------------------------------------------------------

func TestBlockedError_Error_WithMessage(t *testing.T) {
	err := &apperr.BlockedError{
		Message: "3 items unfinished",
	}
	if got := err.Error(); got != "3 items unfinished" {
		t.Errorf("Error() = %q", got)
	}
}

func TestBlockedError_Error_DefaultMessage(t *testing.T) {
	err := &apperr.BlockedError{
		UncheckedItems: []apperr.UncheckedItem{
			{ID: "build_passed", Name: "build passing", Required: true},
			{ID: "tests_passed", Name: "tests passing", Required: true},
		},
	}
	want := "2 items unfinished"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestBlockedError_Category(t *testing.T) {
	err := &apperr.BlockedError{}
	if got := err.Category(); got != "BLOCKED" {
		t.Errorf("Category() = %q, want BLOCKED", got)
	}
}

func TestBlockedError_ErrorsAs(t *testing.T) {
	base := &apperr.BlockedError{
		EntityID: "T042",
		UncheckedItems: []apperr.UncheckedItem{
			{ID: "build_passed", Name: "build passing", Required: true},
		},
	}
	var be *apperr.BlockedError
	if !errors.As(base, &be) {
		t.Fatal("errors.As failed for BlockedError")
	}
	if len(be.UncheckedItems) != 1 {
		t.Errorf("UncheckedItems len = %d, want 1", len(be.UncheckedItems))
	}
}

// ---------------------------------------------------------------------------
// RejectedError
// ---------------------------------------------------------------------------

func TestRejectedError_Error(t *testing.T) {
	err := &apperr.RejectedError{
		Reason: "reason must be at least 10 characters",
	}
	if got := err.Error(); got != "reason must be at least 10 characters" {
		t.Errorf("Error() = %q", got)
	}
}

func TestRejectedError_Category(t *testing.T) {
	err := &apperr.RejectedError{}
	if got := err.Category(); got != "REJECTED" {
		t.Errorf("Category() = %q, want REJECTED", got)
	}
}

func TestRejectedError_ErrorsAs(t *testing.T) {
	base := &apperr.RejectedError{
		Reason:       "only todo state can be deleted",
		CurrentState: "in-progress",
		EntityID:     "T010",
	}
	var re *apperr.RejectedError
	if !errors.As(base, &re) {
		t.Fatal("errors.As failed for RejectedError")
	}
	if re.CurrentState != "in-progress" {
		t.Errorf("CurrentState = %q", re.CurrentState)
	}
}

// ---------------------------------------------------------------------------
// InvalidStateError
// ---------------------------------------------------------------------------

func TestInvalidStateError_Error_WithMessage(t *testing.T) {
	err := &apperr.InvalidStateError{
		Message: "state transition not allowed: Task already complete.",
	}
	if got := err.Error(); got != "state transition not allowed: Task already complete." {
		t.Errorf("Error() = %q", got)
	}
}

func TestInvalidStateError_Error_DefaultMessage(t *testing.T) {
	err := &apperr.InvalidStateError{
		CurrentState:  "done",
		ExpectedState: "in-progress",
	}
	want := "state transition not allowed: current=done, expected=in-progress"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestInvalidStateError_Category(t *testing.T) {
	err := &apperr.InvalidStateError{}
	if got := err.Category(); got != "INVALID_STATE" {
		t.Errorf("Category() = %q, want INVALID_STATE", got)
	}
}

func TestInvalidStateError_ErrorsAs(t *testing.T) {
	base := &apperr.InvalidStateError{
		EntityID:      "T020",
		CurrentState:  "todo",
		ExpectedState: "in-progress",
		RecoveryHint:  "call task_start first.",
	}
	var ise *apperr.InvalidStateError
	if !errors.As(base, &ise) {
		t.Fatal("errors.As failed for InvalidStateError")
	}
	if ise.RecoveryHint != "call task_start first." {
		t.Errorf("RecoveryHint = %q", ise.RecoveryHint)
	}
}

// ---------------------------------------------------------------------------
// Categorizer interface
// ---------------------------------------------------------------------------

func TestCategorizer_Interface(t *testing.T) {
	cases := []struct {
		err     apperr.Categorizer
		wantCat string
	}{
		{&apperr.NotFoundError{}, apperr.CategoryNotFound.String()},
		{&apperr.BlockedError{}, apperr.CategoryBlocked.String()},
		{&apperr.RejectedError{}, apperr.CategoryRejected.String()},
		{&apperr.InvalidStateError{}, apperr.CategoryInvalidState.String()},
		{&apperr.InvalidInputError{}, apperr.CategoryInvalidInput.String()},
		{&apperr.HMACVerifyError{}, apperr.CategoryHMACVerifyFailed.String()},
		{&apperr.ConfigInvalidError{}, apperr.CategoryConfigInvalid.String()},
	}
	for _, c := range cases {
		if got := c.err.Category(); got != c.wantCat {
			t.Errorf("%T.Category() = %q, want %q", c.err, got, c.wantCat)
		}
	}
}

// ---------------------------------------------------------------------------
// T210 — categories enum + MarshalJSON standardisation
// ---------------------------------------------------------------------------

func TestCategoriesEnum_AllCategories(t *testing.T) {
	cats := apperr.AllCategories()
	if len(cats) < 8 {
		t.Errorf("AllCategories len=%d, want >=8", len(cats))
	}
	// Confirm the five core categories are present.
	want := []string{
		apperr.CategoryNotFound.String(),
		apperr.CategoryBlocked.String(),
		apperr.CategoryRejected.String(),
		apperr.CategoryHMACVerifyFailed.String(),
		apperr.CategoryConfigInvalid.String(),
	}
	set := map[string]bool{}
	for _, c := range cats {
		set[c.String()] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("AllCategories missing: %s", w)
		}
	}
}

func TestNotFoundError_MarshalJSON(t *testing.T) {
	e := &apperr.NotFoundError{
		EntityType:   "task",
		EntityID:     "T999",
		RecoveryHint: "check via task list",
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["status"] != "error" {
		t.Errorf("status: %v", m["status"])
	}
	if m["error_category"] != "NOT_FOUND" {
		t.Errorf("error_category: %v", m["error_category"])
	}
	if m["recovery_hint"] != "check via task list" {
		t.Errorf("recovery_hint: %v", m["recovery_hint"])
	}
}

func TestBlockedError_MarshalJSON(t *testing.T) {
	e := &apperr.BlockedError{
		EntityID:         "T999",
		UncheckedItems:   []apperr.UncheckedItem{{ID: "build_passed", Required: true}},
		SuggestedActions: []string{"go build ./...", "go test ./..."},
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["error_category"] != "BLOCKED" {
		t.Errorf("error_category: %v", m["error_category"])
	}
	if m["recovery_hint"] != "go build ./..." {
		t.Errorf("recovery_hint (first action): %v", m["recovery_hint"])
	}
	if _, ok := m["unchecked_items"]; !ok {
		t.Error("unchecked_items field missing")
	}
}

func TestRejectedError_MarshalJSON(t *testing.T) {
	e := &apperr.RejectedError{
		Reason:       "summary too short",
		RecoveryHint: "write at least 10 characters",
		EntityID:     "T999",
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["error_category"] != "REJECTED" {
		t.Errorf("error_category: %v", m["error_category"])
	}
	if m["recovery_hint"] != "write at least 10 characters" {
		t.Errorf("recovery_hint: %v", m["recovery_hint"])
	}
	if m["entity_id"] != "T999" {
		t.Errorf("entity_id: %v", m["entity_id"])
	}
}

func TestInvalidInputError_MarshalJSON(t *testing.T) {
	e := &apperr.InvalidInputError{
		Field:        "title",
		Reason:       "required",
		RecoveryHint: "add the --title flag",
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["error_category"] != "INVALID_INPUT" {
		t.Errorf("error_category: %v", m["error_category"])
	}
	if m["field"] != "title" {
		t.Errorf("field: %v", m["field"])
	}
}

func TestHMACVerifyError_MarshalJSON(t *testing.T) {
	e := &apperr.HMACVerifyError{
		EntityType:   "task",
		EntityID:     "T999",
		FilePath:     "works/tasks/T999.md",
		Expected:     "abcd",
		Computed:     "ef01",
		RecoveryHint: "task rotate-hmac-secret",
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["error_category"] != "HMAC_VERIFY_FAILED" {
		t.Errorf("error_category: %v", m["error_category"])
	}
	if m["expected_prefix"] != "abcd" {
		t.Errorf("expected_prefix: %v", m["expected_prefix"])
	}
}

func TestConfigInvalidError_MarshalJSON(t *testing.T) {
	e := &apperr.ConfigInvalidError{
		FilePath:     ".hostler/project-config.yaml",
		Field:        "harness_required",
		Reason:       "must be array",
		RecoveryHint: "check the yaml format",
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["error_category"] != "CONFIG_INVALID" {
		t.Errorf("error_category: %v", m["error_category"])
	}
	if m["file_path"] != ".hostler/project-config.yaml" {
		t.Errorf("file_path: %v", m["file_path"])
	}
}

func TestInvalidStateError_MarshalJSON(t *testing.T) {
	e := &apperr.InvalidStateError{
		EntityID:      "T999",
		CurrentState:  "done",
		ExpectedState: "in-progress",
		RecoveryHint:  "reopen first",
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["error_category"] != "INVALID_STATE" {
		t.Errorf("error_category: %v", m["error_category"])
	}
	if m["current_state"] != "done" {
		t.Errorf("current_state: %v", m["current_state"])
	}
}
