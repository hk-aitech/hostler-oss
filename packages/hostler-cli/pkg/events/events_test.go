package events

import "testing"

func TestKnownEvents_FourFixed(t *testing.T) {
	events := KnownEvents()
	if len(events) != 4 {
		t.Fatalf("KnownEvents count=%d (expected 4)", len(events))
	}
	expected := []Event{
		EventTaskStart,
		EventTaskComplete,
		EventSprintStart,
		EventSprintComplete,
	}
	for i, e := range expected {
		if events[i] != e {
			t.Errorf("KnownEvents[%d]=%s (expected %s)", i, events[i], e)
		}
	}
}

func TestKnownEventStrings_StringConversion(t *testing.T) {
	strs := KnownEventStrings()
	if len(strs) != 4 {
		t.Fatalf("KnownEventStrings count=%d", len(strs))
	}
	want := []string{"task.start", "task.complete", "sprint.start", "sprint.complete"}
	for i, s := range want {
		if strs[i] != s {
			t.Errorf("KnownEventStrings[%d]=%q (expected %q)", i, strs[i], s)
		}
	}
}

func TestKnownEvents_ReturnsIndependentCopy(t *testing.T) {
	e1 := KnownEvents()
	e1[0] = "mutated"
	e2 := KnownEvents()
	if e2[0] != EventTaskStart {
		t.Errorf("KnownEvents shares the internal slice: %s", e2[0])
	}
}

func TestIsKnown(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"task.start", true},
		{"task.complete", true},
		{"sprint.start", true},
		{"sprint.complete", true},
		{"task-start", false},   // kebab
		{"task_start", false},   // snake
		{"Task.start", false},   // case
		{"task.started", false}, // typo
		{"", false},
		{"unknown", false},
	}
	for _, c := range cases {
		if got := IsKnown(c.name); got != c.want {
			t.Errorf("IsKnown(%q)=%v (expected %v)", c.name, got, c.want)
		}
	}
}

func TestSuggest_CaseNormalisation(t *testing.T) {
	if got := Suggest("Task.Start"); got != EventTaskStart {
		t.Errorf("Suggest(Task.Start)=%s (expected task.start)", got)
	}
	if got := Suggest("TASK.COMPLETE"); got != EventTaskComplete {
		t.Errorf("Suggest(TASK.COMPLETE)=%s", got)
	}
}

func TestSuggest_KebabSnakeConversion(t *testing.T) {
	cases := map[string]Event{
		"task-start":      EventTaskStart,
		"task_start":      EventTaskStart,
		"sprint-complete": EventSprintComplete,
		"sprint_complete": EventSprintComplete,
	}
	for input, want := range cases {
		if got := Suggest(input); got != want {
			t.Errorf("Suggest(%q)=%s (expected %s)", input, got, want)
		}
	}
}

func TestSuggest_substring(t *testing.T) {
	// "task" should return whichever of task.start / task.complete matches first.
	if got := Suggest("task"); got != EventTaskStart {
		t.Errorf("Suggest(task)=%s (expected task.start — first match)", got)
	}
	// "sprint" similarly.
	if got := Suggest("sprint"); got != EventSprintStart {
		t.Errorf("Suggest(sprint)=%s (expected sprint.start)", got)
	}
}

func TestSuggest_NoMatch(t *testing.T) {
	if got := Suggest("totally-unrelated"); got != "" {
		t.Errorf("Suggest(totally-unrelated)=%q (expected empty string)", got)
	}
	if got := Suggest(""); got != "" {
		t.Errorf("Suggest(empty)=%q", got)
	}
}

func TestSuggest_ExactMatchWins(t *testing.T) {
	// When an exact match exists, dot-notation input returns unchanged.
	if got := Suggest("task.start"); got != EventTaskStart {
		t.Errorf("Suggest(task.start)=%s", got)
	}
}
