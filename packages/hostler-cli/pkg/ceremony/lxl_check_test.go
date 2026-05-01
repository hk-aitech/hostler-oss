package ceremony

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"strings"
	"testing"
)

// TestCollectLXLEstimateCheck_warn (T730) returns warn whenever there is one
// or more L/XL Task.
func TestCollectLXLEstimateCheck_warn(t *testing.T) {
	tasks := []domain.TaskSummary{
		{TaskID: "T001", Estimate: "S"},
		{TaskID: "T002", Estimate: "L"},
		{TaskID: "T003", Estimate: "M"},
		{TaskID: "T004", Estimate: "XL"},
	}
	check := collectLXLEstimateCheck(tasks)
	if check == nil {
		t.Fatal("check nil")
		return // nolint (SA5011 guard)
	}
	if check.Status != "warn" {
		t.Errorf("Status: got %q, want warn", check.Status)
	}
	if !strings.Contains(check.Detail, "T002(L)") {
		t.Errorf("Detail missing T002(L): %s", check.Detail)
	}
	if !strings.Contains(check.Detail, "T004(XL)") {
		t.Errorf("Detail missing T004(XL): %s", check.Detail)
	}
	if !strings.Contains(check.Detail, "M Tasks") {
		t.Errorf("Detail missing pre-split guidance: %s", check.Detail)
	}
}

// TestCollectLXLEstimateCheck_pass (T730) returns pass when only S/M/XS
// Tasks are present.
func TestCollectLXLEstimateCheck_pass(t *testing.T) {
	tasks := []domain.TaskSummary{
		{TaskID: "T001", Estimate: "XS"},
		{TaskID: "T002", Estimate: "S"},
		{TaskID: "T003", Estimate: "M"},
		{TaskID: "T004", Estimate: "S"},
	}
	check := collectLXLEstimateCheck(tasks)
	if check == nil {
		t.Fatal("check nil")
		return // nolint (SA5011 guard)
	}
	if check.Status != "pass" {
		t.Errorf("Status: got %q, want pass", check.Status)
	}
	if !strings.Contains(check.Detail, "L/XL estimates: 0") {
		t.Errorf("Detail does not call out 0 L/XL: %s", check.Detail)
	}
}

// TestCollectLXLEstimateCheck_exemptSpikeDocs (T628, Sprint-65) — spike/docs
// L/XL Tasks are exempt from the split advisory. Only L/XL Tasks of other
// types (feature/refactor) should warn.
func TestCollectLXLEstimateCheck_exemptSpikeDocs(t *testing.T) {
	tasks := []domain.TaskSummary{
		{TaskID: "T604", Estimate: "L", Type: "spike"},   // exempt
		{TaskID: "T621", Estimate: "L", Type: "docs"},    // exempt
		{TaskID: "T606", Estimate: "L", Type: "feature"}, // subject to advisory
	}
	check := collectLXLEstimateCheck(tasks)
	if check == nil {
		t.Fatal("check nil")
		return // nolint (SA5011 guard)
	}
	if check.Status != "warn" {
		t.Errorf("Status: got %q, want warn (T606 feature L remains)", check.Status)
	}
	if !strings.Contains(check.Detail, "T606(L)") {
		t.Errorf("Detail missing T606(L) (should be subject to split advisory): %s", check.Detail)
	}
	if strings.Contains(check.Detail, "T604(L) ") || strings.Contains(check.Detail, ", T604(L)") {
		t.Errorf("T604 spike incorrectly listed in split advisory: %s", check.Detail)
	}
	if !strings.Contains(check.Detail, "Exempt") {
		t.Errorf("Detail missing exemption marker: %s", check.Detail)
	}
	if !strings.Contains(check.Detail, "T604") || !strings.Contains(check.Detail, "spike") {
		t.Errorf("Detail exemption section missing T604 spike: %s", check.Detail)
	}
	if !strings.Contains(check.Detail, "T621") || !strings.Contains(check.Detail, "docs") {
		t.Errorf("Detail exemption section missing T621 docs: %s", check.Detail)
	}
}

// TestCollectLXLEstimateCheck_allExempt (T628) — when every L/XL is
// spike/docs the check passes.
func TestCollectLXLEstimateCheck_allExempt(t *testing.T) {
	tasks := []domain.TaskSummary{
		{TaskID: "T604", Estimate: "L", Type: "spike"},
		{TaskID: "T621", Estimate: "L", Type: "docs"},
		{TaskID: "T605", Estimate: "M", Type: "feature"},
	}
	check := collectLXLEstimateCheck(tasks)
	if check == nil {
		t.Fatal("check nil")
		return // nolint (SA5011 guard)
	}
	if check.Status != "pass" {
		t.Errorf("Status: got %q, want pass (no items subject to split advisory)", check.Status)
	}
	if !strings.Contains(check.Detail, "Exempt 2") {
		t.Errorf("Detail missing 'Exempt 2' marker: %s", check.Detail)
	}
}
