package domain

import "testing"

// T836 ADR-001 A3 — TaskID / SprintID value-object unit tests.

func TestTaskID_String(t *testing.T) {
	id := TaskID("T001")
	if id.String() != "T001" {
		t.Errorf("TaskID.String = %q, want T001", id.String())
	}
}

func TestTaskID_IsZero_Empty(t *testing.T) {
	if !TaskID("").IsZero() {
		t.Error("an empty TaskID must report IsZero true")
	}
}

func TestTaskID_IsZero_NonEmpty(t *testing.T) {
	if TaskID("T001").IsZero() {
		t.Error("T001 must report IsZero false")
	}
}

func TestSprintID_String(t *testing.T) {
	id := SprintID("sprint-98")
	if id.String() != "sprint-98" {
		t.Errorf("SprintID.String = %q, want sprint-98", id.String())
	}
}

func TestSprintID_IsZero_Empty(t *testing.T) {
	if !SprintID("").IsZero() {
		t.Error("an empty SprintID must report IsZero true")
	}
}

func TestSprintID_IsZero_NonEmpty(t *testing.T) {
	if SprintID("sprint-98").IsZero() {
		t.Error("sprint-98 must report IsZero false")
	}
}
