// Package ceremony defines data and structures collected for Sprint /
// Task lifecycle ceremonies. When the CLI --with-ceremony flag is set,
// the collector functions in this package run and the result is attached
// to the ceremony field of the JSON response.
package ceremony

// SprintStartCeremony populates the ceremony field of the
// `sprint start --with-ceremony` response.
type SprintStartCeremony struct {
	Briefing        *SprintBriefing  `json:"briefing"`
	DesignReadiness []ReadinessCheck `json:"design_readiness"`
	Reminders       []string         `json:"reminders"`
}

// SprintBriefing is the briefing payload returned at Sprint start.
type SprintBriefing struct {
	SprintID  string        `json:"sprint_id"`
	Title     string        `json:"title"`
	Goal      string        `json:"goal"`
	Tasks     []TaskSummary `json:"tasks"`
	TaskCount int           `json:"task_count"`
}

// TaskSummary is the per-Task summary used in briefings.
type TaskSummary struct {
	TaskID    string   `json:"task_id"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Size      string   `json:"size"`
	Priority  string   `json:"priority"`
	Status    string   `json:"status"`
	DependsOn []string `json:"depends_on,omitempty"`
}

// ReadinessCheck is one design-readiness check.
type ReadinessCheck struct {
	Check  string `json:"check"`
	Status string `json:"status"` // "pass", "warn", "info"
	Detail string `json:"detail,omitempty"`
}

// SprintCompleteCeremony populates the ceremony field of the
// `sprint complete --with-ceremony` response.
type SprintCompleteCeremony struct {
	Harness      []PhaseStatus `json:"harness"`
	KBCandidates []string      `json:"kb_candidates,omitempty"`
	// PacingGuard is the rest-recommendation reminder block emitted
	// immediately after a successful Sprint complete. Populated only when
	// the config flag pacing_guard.reminder_on_sprint_complete is true and
	// the Sprint complete succeeded.
	PacingGuard []string `json:"pacing_guard,omitempty"`
}

// PhaseStatus is the status of one Harness phase.
type PhaseStatus struct {
	Phase  string `json:"phase"`
	Status string `json:"status"` // "done", "pending", "blocked"
	Detail string `json:"detail,omitempty"`
}

// TaskStartCeremony populates the ceremony field of the
// `task start --with-ceremony` response.
type TaskStartCeremony struct {
	Briefing  *TaskBriefing `json:"briefing"`
	Reminders []string      `json:"reminders"`
}

// TaskBriefing is the briefing payload returned at Task start.
type TaskBriefing struct {
	TaskID       string   `json:"task_id"`
	Title        string   `json:"title"`
	Type         string   `json:"type"`
	Estimate     string   `json:"estimate"`
	Priority     string   `json:"priority"`
	Sprint       string   `json:"sprint,omitempty"`
	DependsOn    []string `json:"depends_on,omitempty"`
	CommitPrefix string   `json:"commit_prefix"`
}

// TaskCompleteCeremony populates the ceremony field of the
// `task complete --with-ceremony` response.
type TaskCompleteCeremony struct {
	ChangedFiles  []string            `json:"changed_files"`
	HarnessGate   *HarnessGate        `json:"harness_gate"`
	ResultSection *ResultSectionCheck `json:"result_section"`
}

// HarnessGate is the Harness Gate status at Task completion.
type HarnessGate struct {
	Status         string   `json:"status"` // "pass", "blocked"
	UncheckedItems []string `json:"unchecked_items,omitempty"`
}

// ResultSectionCheck reports whether the Task Result section exists.
type ResultSectionCheck struct {
	Exists       bool     `json:"exists"`
	MissingFiles []string `json:"missing_files,omitempty"`
}
