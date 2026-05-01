package sprint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestT681_NextAvailableSprintID_BaseIDPattern — when baseID matches the
// sprint-N pattern, finds an unoccupied ID starting from N+1.
func TestT681_NextAvailableSprintID_BaseIDPattern(t *testing.T) {
	occupied := map[string]string{
		"sprint-72": "/wt-a",
		"sprint-73": "/wt-b",
		"sprint-74": "/wt-c",
		"sprint-75": "/wt-d",
		"sprint-76": "/wt-e",
		"sprint-77": "/wt-f",
	}
	got := nextAvailableSprintID(occupied, "sprint-72")
	if got != "sprint-78" {
		t.Errorf("expected sprint-78, got %s", got)
	}
}

// TestT681_NextAvailableSprintID_NoOccupied — returns baseID+1 when
// nothing is occupied.
func TestT681_NextAvailableSprintID_NoOccupied(t *testing.T) {
	occupied := map[string]string{}
	got := nextAvailableSprintID(occupied, "sprint-77")
	if got != "sprint-78" {
		t.Errorf("expected sprint-78, got %s", got)
	}
}

// TestT681_NextAvailableSprintID_GapInOccupied — returns the first gap
// when the occupied range has a gap in it.
func TestT681_NextAvailableSprintID_GapInOccupied(t *testing.T) {
	occupied := map[string]string{
		"sprint-78": "/wt-a",
		// sprint-79 unoccupied
		"sprint-80": "/wt-b",
	}
	got := nextAvailableSprintID(occupied, "sprint-78")
	if got != "sprint-79" {
		t.Errorf("expected sprint-79 (first gap), got %s", got)
	}
}

// TestT681_ScanOccupied_FromTempDirs — creates sprint folders under a
// temp dir and confirms scanOccupiedSprintIDs recognises them correctly.
func TestT681_ScanOccupied_FromTempDirs(t *testing.T) {
	wt := t.TempDir()
	mustMkdir(t, filepath.Join(wt, "works/sprints/backlog/sprint-78"))
	mustMkdir(t, filepath.Join(wt, "works/sprints/active/sprint-79"))
	mustMkdir(t, filepath.Join(wt, "works/sprints/completed/sprint-80"))
	mustMkdir(t, filepath.Join(wt, "works/sprints/discarded/sprint-81"))
	// noise — must not match
	mustMkdir(t, filepath.Join(wt, "works/sprints/backlog/not-a-sprint"))
	mustMkdir(t, filepath.Join(wt, "works/sprints/backlog/sprint-XXX"))

	occupied := scanOccupiedSprintIDs([]string{wt})

	expected := []string{"sprint-78", "sprint-79", "sprint-80", "sprint-81"}
	for _, id := range expected {
		if _, ok := occupied[id]; !ok {
			t.Errorf("expected %s in occupied set, missing", id)
		}
	}
	if _, ok := occupied["not-a-sprint"]; ok {
		t.Error("noise dir 'not-a-sprint' incorrectly matched")
	}
	if _, ok := occupied["sprint-XXX"]; ok {
		t.Error("noise dir 'sprint-XXX' incorrectly matched")
	}
}

// TestT681_CheckConflict_OptOut — returns nil when env
// HSTL_SPRINT_ID_CONFLICT_CHECK=off. envalias auto-prepends HSTL_ so the
// actual env var is HSTL_<IDConflictCheckEnv>.
func TestT681_CheckConflict_OptOut(t *testing.T) {
	t.Setenv("HSTL_"+IDConflictCheckEnv, "off")
	err := CheckMultiWorktreeIDConflict("sprint-78")
	if err != nil {
		t.Errorf("expected nil on opt-out, got: %v", err)
	}
}

// TestT681_CheckConflict_GracefulOnGitFailure — even when the parent
// directory is not a git checkout, returns nil (skip) without panicking.
// (When git worktree list actually fails, listWorktreePaths returns an
// error and CheckMultiWorktreeIDConflict prints a stderr WARN and returns
// nil.)
func TestT681_CheckConflict_GracefulOnGitFailure(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	// t.TempDir() may itself live inside a git repo, so only verify
	// graceful behaviour.
	err := CheckMultiWorktreeIDConflict("sprint-9999")
	if err != nil && !strings.Contains(err.Error(), "occupied") {
		t.Errorf("expected graceful skip, got unexpected error: %v", err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
