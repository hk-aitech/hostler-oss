// Package domain — Task summary DTO.
package domain

// TaskSummary is the list / recommendation projection of a Task.
type TaskSummary struct {
	TaskID    string   `json:"task_id"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Sprint    any      `json:"sprint"`
	Status    string   `json:"status,omitempty"`
	Priority  string   `json:"priority"`
	Estimate  string   `json:"estimate"`
	DependsOn []string `json:"depends_on"`
	CreatedAt string   `json:"created_at"`

	// TemplateOutdated reports whether the task body is missing the required
	// sections produced by the current RenderTaskTemplate. Existing tasks are
	// left untouched; the flag exists only as a soft signal to authors.
	TemplateOutdated bool `json:"template_outdated,omitempty"`
}
