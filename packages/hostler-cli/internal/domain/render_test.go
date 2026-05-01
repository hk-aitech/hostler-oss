// Package domain — renderer unit tests.
//
// Verifies that the TextRenderer / ConsoleRenderer implementations are
// attached to the domain Result types and that the output package can
// dispatch them via type assertion. Snapshot-style verification.
package domain

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetResult_Text(t *testing.T) {
	r := &GetResult{
		TaskID:   "T999",
		Title:    "test task",
		Type:     "feature",
		Status:   "in-progress",
		Priority: "p1",
		Estimate: "M",
		FilePath: "works/tasks/T999.md",
	}
	var buf bytes.Buffer
	r.Text(&buf)
	out := buf.String()
	if !strings.Contains(out, "T999") || !strings.Contains(out, "test task") {
		t.Errorf("Text output is missing key fields: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("Text mode contains ANSI escape: %q", out)
	}
}

func TestGetResult_Console_NoColor(t *testing.T) {
	r := &GetResult{TaskID: "T999", Title: "x", Type: "f", Status: "done", Priority: "p0", Estimate: "S", FilePath: "f"}
	var buf bytes.Buffer
	r.Console(&buf, true)
	if strings.Contains(buf.String(), "\033[") {
		t.Errorf("noColor=true but ANSI escape was emitted: %q", buf.String())
	}
}

func TestGetResult_Console_WithColor(t *testing.T) {
	r := &GetResult{TaskID: "T999", Title: "x", Type: "f", Status: "done", Priority: "p0", Estimate: "S", FilePath: "f"}
	var buf bytes.Buffer
	r.Console(&buf, false)
	if !strings.Contains(buf.String(), "\033[") {
		t.Errorf("noColor=false but no ANSI escape was emitted: %q", buf.String())
	}
}

func TestListResult_Text_Empty(t *testing.T) {
	r := &ListResult{}
	var buf bytes.Buffer
	r.Text(&buf)
	if !strings.Contains(buf.String(), "no tasks") {
		t.Errorf("empty ListResult Text output: %q", buf.String())
	}
}

func TestListResult_Console(t *testing.T) {
	r := &ListResult{
		Tasks: []TaskSummary{
			{TaskID: "T001", Title: "first", Type: "feature", Status: "done", Priority: "p1"},
			{TaskID: "T002", Title: "second", Type: "bugfix", Status: "todo", Priority: "p2"},
		},
		Count: 2,
	}
	var buf bytes.Buffer
	r.Console(&buf, true)
	out := buf.String()
	if !strings.Contains(out, "T001") || !strings.Contains(out, "T002") {
		t.Errorf("ListResult Console output missing entries: %q", out)
	}
	if !strings.Contains(out, "Total:") {
		t.Errorf("Total line missing: %q", out)
	}
}

func TestCreateResult_Text(t *testing.T) {
	r := &CreateResult{TaskID: "T999", FilePath: "x.md", CreatedAt: "2026-04-20", Status: "todo"}
	var buf bytes.Buffer
	r.Text(&buf)
	if !strings.Contains(buf.String(), "T999") {
		t.Errorf("CreateResult Text: %q", buf.String())
	}
}

func TestTransitionResult_Text(t *testing.T) {
	r := &TransitionResult{
		TaskID: "T999", PreviousStatus: "todo", NewStatus: "in-progress",
		Title: "x", Type: "feature", Estimate: "M", Priority: "p1",
		CommitPrefix: "feat", WorkTicket: "WT-T999-deadbeef",
	}
	var buf bytes.Buffer
	r.Text(&buf)
	out := buf.String()
	if !strings.Contains(out, "T999") || !strings.Contains(out, "todo") || !strings.Contains(out, "in-progress") {
		t.Errorf("TransitionResult Text: %q", out)
	}
}

func TestReopenResult_Text(t *testing.T) {
	r := &ReopenResult{TaskID: "T999", PreviousStatus: "done", NewStatus: "todo", Reason: "test"}
	var buf bytes.Buffer
	r.Text(&buf)
	if !strings.Contains(buf.String(), "reopened") {
		t.Errorf("ReopenResult Text: %q", buf.String())
	}
}

func TestSprintCreateResult_Text(t *testing.T) {
	r := &SprintCreateResult{SprintID: "sprint-99", Title: "test", Goal: "g", Status: "backlog", FolderPath: "f"}
	var buf bytes.Buffer
	r.Text(&buf)
	if !strings.Contains(buf.String(), "sprint-99") {
		t.Errorf("SprintCreateResult: %q", buf.String())
	}
}

func TestSprintStartResult_Text(t *testing.T) {
	r := &SprintStartResult{SprintID: "sprint-99", Status: "active", StartedAt: "2026-04-20", FolderPath: "f"}
	var buf bytes.Buffer
	r.Text(&buf)
	if !strings.Contains(buf.String(), "Started") {
		t.Errorf("SprintStartResult: %q", buf.String())
	}
}



func TestSprintCompleteResult_Text(t *testing.T) {
	r := &SprintCompleteResult{SprintID: "sprint-99", Status: "completed", CompletedAt: "2026-04-20", FolderPath: "f"}
	var buf bytes.Buffer
	r.Text(&buf)
	if !strings.Contains(buf.String(), "Completed") {
		t.Errorf("SprintCompleteResult: %q", buf.String())
	}
}

func TestProgressResult_Console(t *testing.T) {
	r := &ProgressResult{SprintID: "sprint-99", Done: 5, InProgress: 2, Todo: 3, Total: 10, Percent: 50}
	var buf bytes.Buffer
	r.Console(&buf, true)
	out := buf.String()
	if !strings.Contains(out, "sprint-99") || !strings.Contains(out, "50%") {
		t.Errorf("ProgressResult Console: %q", out)
	}
	if !strings.Contains(out, "█") || !strings.Contains(out, "░") {
		t.Errorf("progress bar missing: %q", out)
	}
}

func TestHarnessGetResult_Text(t *testing.T) {
	r := &HarnessGetResult{
		EntityType: "task", EntityID: "T999", TemplateKey: "task:feature",
		Items: []HarnessItemView{
			{ID: "criteria_checked", Name: "criteria check", Required: true, Done: true},
			{ID: "build_passed", Name: "build", Required: true, Done: false},
		},
		RequiredTotal: 2, RequiredDone: 1, Blocked: true,
	}
	var buf bytes.Buffer
	r.Text(&buf)
	out := buf.String()
	if !strings.Contains(out, "T999") {
		t.Errorf("HarnessGetResult.Text missing TaskID: %q", out)
	}
	if !strings.Contains(out, "[x]") || !strings.Contains(out, "[ ]") {
		t.Errorf("check marks missing: %q", out)
	}
}
