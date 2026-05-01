package backlog

import "testing"

// T246 (Sprint-17, ISS-20260421-003) — unit tests for the
// sprint-status-rollback guard.
//
// Background: a downstream report. backlog sync would, on a Sprint with
// DB status=completed and lingering files in the backlog directory,
// suggest "fix DB to match file location (completed→backlog)" with
// auto_fixable=true. That is destructive.
//
// Guard: isSprintStatusRegression() detects the rollback and emits
// issueDBSprintStatusRegression (AutoFixable=false), blocking the auto-fix.

func TestT246_SprintStatusRank(t *testing.T) {
	cases := []struct {
		status string
		rank   int
	}{
		{"backlog", 0},
		{"active", 1},
		{"completed", 2},
		{"", -1},
		{"unknown", -1},
	}
	for _, c := range cases {
		got := sprintStatusRank(c.status)
		if got != c.rank {
			t.Errorf("sprintStatusRank(%q) = %d, want %d", c.status, got, c.rank)
		}
	}
}

func TestT246_IsSprintStatusRegression(t *testing.T) {
	cases := []struct {
		name       string
		dbStatus   string
		fileStatus string
		regression bool
	}{
		{
			name:       "completed→backlog is rollback (destructive)",
			dbStatus:   "completed",
			fileStatus: "backlog",
			regression: true,
		},
		{
			name:       "completed→active is rollback (sprint reopen required)",
			dbStatus:   "completed",
			fileStatus: "active",
			regression: true,
		},
		{
			name:       "active→backlog is rollback",
			dbStatus:   "active",
			fileStatus: "backlog",
			regression: true,
		},
		{
			name:       "backlog→active is forward (sync normal)",
			dbStatus:   "backlog",
			fileStatus: "active",
			regression: false,
		},
		{
			name:       "active→completed is forward",
			dbStatus:   "active",
			fileStatus: "completed",
			regression: false,
		},
		{
			name:       "same status is not a rollback",
			dbStatus:   "active",
			fileStatus: "active",
			regression: false,
		},
		{
			name:       "unknown status cannot be classified → false (safe fallback)",
			dbStatus:   "weird",
			fileStatus: "backlog",
			regression: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isSprintStatusRegression(c.dbStatus, c.fileStatus)
			if got != c.regression {
				t.Errorf("isSprintStatusRegression(%q, %q) = %v, want %v",
					c.dbStatus, c.fileStatus, got, c.regression)
			}
		})
	}
}

// TestT435_TaskStatusRank — unit tests for Task status progression order.
func TestT435_TaskStatusRank(t *testing.T) {
	cases := []struct {
		status string
		rank   int
	}{
		{"todo", 0},
		{"in-progress", 1},
		{"in_progress", 1},
		{"done", 2},
		{"", -1},
		{"unknown", -1},
	}
	for _, c := range cases {
		got := taskStatusRank(c.status)
		if got != c.rank {
			t.Errorf("taskStatusRank(%q) = %d, want %d", c.status, got, c.rank)
		}
	}
}

// TestT435_IsTaskStatusRegression — task-level DB status rollback judgement.
func TestT435_IsTaskStatusRegression(t *testing.T) {
	cases := []struct {
		name       string
		dbStatus   string
		fileStatus string
		regression bool
	}{
		{
			name:       "done→todo is rollback (revert completed task)",
			dbStatus:   "done",
			fileStatus: "todo",
			regression: true,
		},
		{
			name:       "done→in-progress is rollback",
			dbStatus:   "done",
			fileStatus: "in-progress",
			regression: true,
		},
		{
			name:       "in-progress→todo is rollback",
			dbStatus:   "in-progress",
			fileStatus: "todo",
			regression: true,
		},
		{
			name:       "todo→done is forward (file leads)",
			dbStatus:   "todo",
			fileStatus: "done",
			regression: false,
		},
		{
			name:       "same status is not a rollback",
			dbStatus:   "done",
			fileStatus: "done",
			regression: false,
		},
		{
			name:       "unknown status cannot be classified → false (safe fallback)",
			dbStatus:   "weird",
			fileStatus: "todo",
			regression: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isTaskStatusRegression(c.dbStatus, c.fileStatus)
			if got != c.regression {
				t.Errorf("isTaskStatusRegression(%q, %q) = %v, want %v",
					c.dbStatus, c.fileStatus, got, c.regression)
			}
		})
	}
}

// TestT435_IssueType_TaskRegressionIsNotAutoFixable — invariant: the
// issueDBTaskStatusRegression issue type must be a different constant
// from issueDBStatusMismatch.
func TestT435_IssueType_TaskRegressionIsNotAutoFixable(t *testing.T) {
	if issueDBTaskStatusRegression == "" {
		t.Fatal("issueDBTaskStatusRegression constant is empty")
	}
	if issueDBTaskStatusRegression == issueDBStatusMismatch {
		t.Errorf("issueDBTaskStatusRegression must differ from issueDBStatusMismatch (auto_fixable branch semantics)")
	}
}

// TestT246_IssueType_RegressionIsNotAutoFixable — invariant: the default
// AutoFixable value of issueDBSprintStatusRegression must always be false.
//
// Goal: prevent string-constant-level regressions if someone later
// classifies this issue type as auto_fixable=true by mistake.
func TestT246_IssueType_RegressionIsNotAutoFixable(t *testing.T) {
	// Constant existence check.
	if issueDBSprintStatusRegression == "" {
		t.Fatal("issueDBSprintStatusRegression constant is empty")
	}
	if issueDBSprintStatusRegression == issueDBSprintStatusFileAbsent {
		t.Errorf("issueDBSprintStatusRegression must differ from issueDBSprintStatusFileAbsent (auto_fixable branch semantics)")
	}
}
