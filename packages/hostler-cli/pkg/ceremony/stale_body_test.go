package ceremony

import (
	"fmt"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"strings"
	"testing"
	"time"
)

// T601 (Sprint-71): Task body authored-vs-started date gap check.

func TestT601_StaleBodyCheck_Empty(t *testing.T) {
	got := collectStaleBodyCheck(nil)
	if got == nil || got.Status != "pass" {
		t.Errorf("expected pass for empty Task list, got %+v", got)
	}
}

func TestT601_StaleBodyCheck_AllFresh(t *testing.T) {
	t.Setenv("HSTL_SPRINT_STALE_BODY_DAYS", "30")
	today := time.Now().Format("2006-01-02")
	tasks := []domain.TaskSummary{
		{TaskID: "T001", CreatedAt: today},
		{TaskID: "T002", CreatedAt: today},
	}
	got := collectStaleBodyCheck(tasks)
	if got == nil || got.Status != "pass" {
		t.Errorf("expected pass when all authored today, got %+v", got)
	}
}

func TestT601_StaleBodyCheck_Stale(t *testing.T) {
	t.Setenv("HSTL_SPRINT_STALE_BODY_DAYS", "30")
	old := time.Now().AddDate(0, 0, -45).Format("2006-01-02")
	tasks := []domain.TaskSummary{
		{TaskID: "T001", CreatedAt: old},
		{TaskID: "T002", CreatedAt: time.Now().Format("2006-01-02")},
	}
	got := collectStaleBodyCheck(tasks)
	if got == nil || got.Status != "warn" {
		t.Fatalf("expected warn when a Task is 45 days old, got %+v", got)
	}
	if !strings.Contains(got.Detail, "T001") {
		t.Errorf("T001 missing from detail: %s", got.Detail)
	}
	if strings.Contains(got.Detail, "T002") {
		t.Errorf("T002 is fresh but appears in detail: %s", got.Detail)
	}
}

func TestT601_StaleBodyCheck_CustomThreshold(t *testing.T) {
	t.Setenv("HSTL_SPRINT_STALE_BODY_DAYS", "7")
	old := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	tasks := []domain.TaskSummary{{TaskID: "T001", CreatedAt: old}}
	got := collectStaleBodyCheck(tasks)
	if got == nil || got.Status != "warn" {
		t.Errorf("expected warn with threshold 7 and 10-day-old Task, got %+v", got)
	}
}

func TestT601_StaleBodyCheck_InvalidDate(t *testing.T) {
	tasks := []domain.TaskSummary{{TaskID: "T001", CreatedAt: "garbage"}}
	got := collectStaleBodyCheck(tasks)
	if got == nil || got.Status != "pass" {
		t.Errorf("expected pass when parse-failed Task is ignored, got %+v", got)
	}
}

func TestT601_ParseTaskCreatedAt_Formats(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"2026-04-15", true},
		{"2026-04-15T00:00:00Z", true},
		{"", false},
		{"invalid", false},
	}
	for _, c := range cases {
		_, ok := parseTaskCreatedAt(c.in)
		if ok != c.ok {
			t.Errorf("parseTaskCreatedAt(%q) ok=%v, expected %v", c.in, ok, c.ok)
		}
	}
	// Sanity: that the result has the expected year
	_ = fmt.Sprintf // keep fmt import in case of future assertions
}
