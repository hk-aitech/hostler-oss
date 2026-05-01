// Package config — ConfigProvider adapter.
//
// ADR-001 §4.2 Phase C. Wraps the existing GetReminders / GetSkillExtensions /
// GetProjectKey / GetProjectConfigSummary / GetTaskResultCheckPolicy /
// GetSprintStaleBodyDays global functions in a struct so the result satisfies
// ports.ConfigProvider. Phase D will route the CLI through this adapter.
package config

import (
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/registry"
)

// Registry path override environment variables .
const (
	envRegistryRuntimePath = "HSTL_REGISTRY_PATH"
	envRegistryDocsPath    = "HSTL_REGISTRY_DOCS_PATH"
)

// init — registers a config reader with pkg/registry so
// registry.GetRuntimeRegistryPath can consult the YAML configuration
// (, pattern that avoids an import cycle).
func init() {
	registry.RegisterConfigReader(func(field string) string {
		cfg, err := LoadProjectConfig()
		if err != nil || cfg == nil {
			return ""
		}
		r := cfg.GetRegistry()
		if r == nil {
			return ""
		}
		switch field {
		case "runtime":
			return strings.TrimSpace(r.Path)
		case "docs":
			return strings.TrimSpace(r.DocsPath)
		}
		return ""
	})
}

// ConfigProviderAdapter is a ConfigProvider adapter built atop the global config functions.
type ConfigProviderAdapter struct{}

// NewConfigProviderAdapter constructs the default config adapter.
func NewConfigProviderAdapter() *ConfigProviderAdapter {
	return &ConfigProviderAdapter{}
}

// Reminders returns the reminder entries for the given event.
func (ConfigProviderAdapter) Reminders(eventName string) []string {
	return GetReminders(eventName)
}

// SkillExtensions returns the extension keys for the given skill.
func (ConfigProviderAdapter) SkillExtensions(skillName string) []string {
	return GetSkillExtensions(skillName)
}

// ProjectKey returns the unique key of the current project.
func (ConfigProviderAdapter) ProjectKey() string {
	return GetProjectKey()
}

// ProjectConfigSummary returns the canonical-sorted config summary map.
func (ConfigProviderAdapter) ProjectConfigSummary() map[string]any {
	return GetProjectConfigSummary()
}

// TaskResultCheckPolicy returns the task_complete result-section validation policy.
func (ConfigProviderAdapter) TaskResultCheckPolicy() string {
	return GetTaskResultCheckPolicy()
}

// SprintStaleBodyDays returns the day threshold for the Task body gap at Sprint start.
func (ConfigProviderAdapter) SprintStaleBodyDays() int {
	return GetSprintStaleBodyDays()
}

// RegistryRuntimePath returns the runtime registry.json path override
// . Order: env -> yaml -> "". Returns "" if yaml fails to load.
func (ConfigProviderAdapter) RegistryRuntimePath() string {
	if v := strings.TrimSpace(envalias.Lookup("REGISTRY_PATH")); v != "" {
		return v
	}
	cfg, err := LoadProjectConfig()
	if err != nil || cfg == nil {
		return ""
	}
	if r := cfg.GetRegistry(); r != nil {
		return strings.TrimSpace(r.Path)
	}
	return ""
}

// RegistryDocsPath returns the docs snapshot registry.json path override
// . Order: env -> yaml -> "".
func (ConfigProviderAdapter) RegistryDocsPath() string {
	if v := strings.TrimSpace(envalias.Lookup("REGISTRY_DOCS_PATH")); v != "" {
		return v
	}
	cfg, err := LoadProjectConfig()
	if err != nil || cfg == nil {
		return ""
	}
	if r := cfg.GetRegistry(); r != nil {
		return strings.TrimSpace(r.DocsPath)
	}
	return ""
}

// Compile-time check — ConfigProviderAdapter satisfies the ports.ConfigProvider contract.
var _ ports.ConfigProvider = (*ConfigProviderAdapter)(nil)
