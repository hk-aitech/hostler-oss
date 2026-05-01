// Package ports — ConfigProvider port.
//
// hstl-specific port. Reads runtime configuration resolved from
// project-config.yaml plus presets and the environment-variable
// cascade. Today `pkg/config/` handles YAML loading, canonical
// ordering, and preset merging.
//
// Verification pattern:
//
//	var _ ports.ConfigProvider = (*ConfigProviderAdapter)(nil)
//
// Return types are kept to plain values and maps; domain-specific structs
// (e.g. SprintCeremony) are returned as `any` so adapters can expose rich
// types directly.
package ports

// ConfigProvider exposes resolved project configuration.
// Currently implemented by `pkg/config`.
type ConfigProvider interface {
	// Reminders returns the reminder items for the given event
	// (task-start, sprint-complete, etc.). Returns an empty slice when the
	// configuration omits the event.
	Reminders(eventName string) []string

	// SkillExtensions returns the list of extension configuration keys for
	// the given skill.
	SkillExtensions(skillName string) []string

	// ProjectKey returns the unique key (hash, etc.) of the current project.
	// Sourced from project.key in project-config.yaml.
	ProjectKey() string

	// ProjectConfigSummary returns the canonical-ordered configuration
	// summary (map) for debugging / inspection.
	ProjectConfigSummary() map[string]any

	// TaskResultCheckPolicy returns the policy (strict | warn | off) used by
	// task_complete to verify files in the `## Results` section.
	TaskResultCheckPolicy() string

	// SprintStaleBodyDays returns the threshold (in days) for the Task body
	// gap evaluated at Sprint start.
	SprintStaleBodyDays() int

	// RegistryRuntimePath returns the runtime registry.json path.
	// Precedence: HSTL_REGISTRY_PATH env → registry.path yaml → "".
	// When empty, callers must fall back to
	// ~/{ProjectDirName}/data/{project_key}/registry.json.
	RegistryRuntimePath() string

	// RegistryDocsPath returns the override for the git-managed snapshot
	// registry.json path. Precedence: HSTL_REGISTRY_DOCS_PATH env →
	// registry.docs_path yaml → "". When empty, callers fall back to
	// {project_root}/{ProjectDirName}/data/registry.json.
	RegistryDocsPath() string
}
