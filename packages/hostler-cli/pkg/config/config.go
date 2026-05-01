// Package config handles project configuration parsing.
//
// The single SSOT is project-config.yaml under brand.ProjectDirName
// (handled by yaml_loader.go). Earlier markdown-based formats are no
// longer supported; YAML is the only recognised input.
package config

import (
	"errors"
	"strconv"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/template"
)

// configLogger routes through pkg/log. Package-level singleton.
var configLogger = pkglog.NewDefault()

// GetReminders returns the reminder entries for a given event.
//
// Priority (Content-as-Template split):
//  1. {brand.ProjectDirName}/templates/reminders/{eventName}.md (highest priority)
//  2. project-config.yaml reminders.{eventName} (backward compat)
//  3. otherwise an empty slice
//
// I03: eventName expects the dot-notation value of events.KnownEvents
// (e.g. events.EventTaskStart = "task.start"). A raw string also works, but
// callers should prefer the `events.Event` constants when possible.
//
// Once per load, when project-config.yaml has a reminders key that does not
// match events.IsKnown, a warning plus a typo suggestion is logged.
func GetReminders(eventName string) []string {
	if eventName == "" {
		return []string{}
	}

	// First priority: templates/reminders/{event}.md
	if items, _, err := template.LoadReminderContent(eventName); err == nil && len(items) > 0 {
		return items
	} else if err != nil && !errors.Is(err, template.ErrTemplateNotFound) {
		configLogger.Warn(brand.LogPrefix+" reminder template load failed — YAML fallback",
			"event", eventName, "error", err.Error())
	}

	// yaml reminders fallback removed. templates/ is the SSOT.
	return []string{}
}

// GetSkillExtensions returns the per-skill extension entries.
//
// Priority:
//  1. project-config.yaml skills.{skillName}.rules (v2 namespace, preferred)
//  2. skill_extensions.{skillName} (v1 deprecated, fallback)
//  3. otherwise an empty slice
//
// The function name is preserved for compatibility with consumer code
// before this rename. New code should use the cfg.GetSkillRules(name) method.
func GetSkillExtensions(skillName string) []string {
	if skillName == "" {
		return []string{}
	}
	cfg, err := LoadProjectConfig()
	if err != nil || cfg == nil {
		return []string{}
	}
	rules := cfg.GetSkillRules(skillName)
	if rules == nil {
		return []string{}
	}
	return rules
}

// GetProjectKey returns project.key from project-config.yaml.
// Returns an empty string when the file is missing or the key is empty.
func GetProjectKey() string {
	cfg, err := LoadProjectConfig()
	if err != nil || cfg == nil {
		return ""
	}
	return cfg.Project.Key
}

// GetProjectConfigSummary reports whether project-config exists and supplies basic info.
//
// Returned shape:
//
//	{
//	  exists: true,
//	  format: "yaml",
//	  path:   "...",
//	  has_bc_structure: bool,
//	  has_reminders: bool,
//	  has_skill_extensions: bool,
//	  has_sprint_ceremony: bool,
//	  has_briefing: bool,
//	}
//
// Returns nil when the file is missing.
func GetProjectConfigSummary() map[string]any {
	path := findProjectConfigYAML()
	if path == "" {
		return nil
	}
	cfg, err := LoadProjectConfig()
	if err != nil {
		return map[string]any{
			"exists":      true,
			"format":      "yaml",
			"path":        brand.ProjectDirName + "/" + filepathBaseFromAbs(path),
			"parse_error": err.Error(),
		}
	}
	if cfg == nil {
		return nil
	}
	// has_bc_structure checks for BC presence both before and after the
	// Platform absorption. True when either the deprecated top-level
	// BoundedContexts or Platform.Dotnet.BoundedContexts has values.
	hasBC := len(cfg.GetBoundedContexts()) > 0
	// platform_kind is the Kind value when Platform is set, otherwise empty.
	platformKind := ""
	if cfg.Platform != nil {
		platformKind = cfg.Platform.Kind
	}
	return map[string]any{
		"exists":           true,
		"format":           "yaml",
		"path":             brand.ProjectDirName + "/" + filepathBaseFromAbs(path),
		"platform_kind":    platformKind,
		"has_bc_structure": hasBC,
		"has_briefing":     cfg.Briefing != nil,
	}
}

// filepathBaseFromAbs returns only the last path component of an absolute path.
// Defined as a separate helper so the path/filepath import stays in yaml_loader.go.
func filepathBaseFromAbs(p string) string {
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return p
	}
	return p[idx+1:]
}

// GetSprintCeremony returns the SprintCeremony definition (Sprints.Ceremony only).
func GetSprintCeremony() *SprintCeremony {
	cfg, err := LoadProjectConfig()
	if err != nil || cfg == nil {
		return nil
	}
	return cfg.GetSprintCeremony()
}

// Default validation policy and section titles. Declared as constants
// to comply with the no-magic-constant rule.
const (
	defaultTaskResultCheckPolicy = "strict" // validation aims to block non-compliance — enforce by default
	defaultSprintStaleBodyDays   = 30       // Task body authoring vs. start gap threshold
)

// defaultDriftLessPaths is the default list of drift-less regenerated files (drift-less safeguard).
// Used when the project configuration leaves the value empty.
var defaultDriftLessPaths = []string{
	"docs/generated/manifest.json",
	"docs/03-design/skill-trigger-index.md",
}

// GetTaskResultDriftLessPaths returns the path patterns that strict validation
// treats as "exclude from missing when the file exists, even if absent from
// the git diff" (drift-less safeguard).
// Priority: project-config.yaml -> defaults. No environment variable override.
func GetTaskResultDriftLessPaths() []string {
	cfg, err := LoadProjectConfig()
	if err == nil && cfg != nil {
		if rc := cfg.GetTaskResultCheck(); rc != nil && len(rc.DriftLessPaths) > 0 {
			return rc.DriftLessPaths
		}
	}
	return defaultDriftLessPaths
}

// GetSprintStaleBodyDays returns the day threshold (in days) for the
// sprint_start ceremony's "review Task body again" recommendation.
// Priority: env -> yaml -> default(30).
func GetSprintStaleBodyDays() int {
	if v := strings.TrimSpace(envalias.Lookup("SPRINT_STALE_BODY_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	cfg, err := LoadProjectConfig()
	if err == nil && cfg != nil {
		if sc := cfg.GetSprintCeremony(); sc != nil && sc.StaleBodyDays > 0 {
			return sc.StaleBodyDays
		}
	}
	return defaultSprintStaleBodyDays
}

// defaultTaskResultSectionTitles is the default list of markdown headings recognised as result sections.
var defaultTaskResultSectionTitles = []string{"Result", "Deliverables"}

// validTaskResultCheckPolicies is the set of valid policy values.
var validTaskResultCheckPolicies = map[string]bool{
	"off":    true,
	"warn":   true,
	"strict": true,
}

// GetTaskResultCheckPolicy returns the result-section validation policy for task_complete.
//
// Priority:
//  1. environment variable HSTL_TASK_RESULT_CHECK_POLICY (off|warn|strict)
//  2. project-config.yaml task_result_check.policy
//  3. default "strict"
//
// Unknown values fall back to the default.
func GetTaskResultCheckPolicy() string {
	// 1. environment variable wins
	if v := strings.TrimSpace(envalias.Lookup("TASK_RESULT_CHECK_POLICY")); v != "" {
		v = strings.ToLower(v)
		if validTaskResultCheckPolicies[v] {
			return v
		}
	}

	// 2. YAML config (consults Tasks.ResultCheck only)
	cfg, err := LoadProjectConfig()
	if err == nil && cfg != nil {
		if rc := cfg.GetTaskResultCheck(); rc != nil {
			v := strings.ToLower(strings.TrimSpace(rc.Policy))
			if validTaskResultCheckPolicies[v] {
				return v
			}
		}
	}

	// 3. default
	return defaultTaskResultCheckPolicy
}

// GetTaskResultSectionTitles returns the list of markdown headings recognised as result sections.
//
// Priority:
//  1. environment variable HSTL_TASK_RESULT_SECTION_TITLES (comma-separated, e.g. "Result,Deliverables,Outcome")
//  2. project-config.yaml task_result_check.section_titles
//  3. default ["Result", "Deliverables"]
//
// Empty and duplicate entries are removed (deduped). When the environment
// variable is empty or whitespace, the next priority takes over.
//
// dedup background: defends against duplicates accumulating because
// merge.go concatenates arrays when the extends preset (builtin:hostler-base)
// supplies defaults that overlap with the project's own value. This function
// is the SSOT entry point, so callers always see deduped output.
func GetTaskResultSectionTitles() []string {
	// 1. environment variable wins
	if v := strings.TrimSpace(envalias.Lookup("TASK_RESULT_SECTION_TITLES")); v != "" {
		titles := splitAndTrim(v, ",")
		if deduped := dedupPreservingOrder(titles); len(deduped) > 0 {
			return deduped
		}
	}

	// 2. YAML config (consults Tasks.ResultCheck only)
	cfg, err := LoadProjectConfig()
	if err == nil && cfg != nil {
		if rc := cfg.GetTaskResultCheck(); rc != nil && len(rc.SectionTitles) > 0 {
			var titles []string
			for _, t := range rc.SectionTitles {
				t = strings.TrimSpace(t)
				if t != "" {
					titles = append(titles, t)
				}
			}
			if deduped := dedupPreservingOrder(titles); len(deduped) > 0 {
				return deduped
			}
		}
	}

	// 3. default (return a copy so callers cannot mutate it)
	out := make([]string, len(defaultTaskResultSectionTitles))
	copy(out, defaultTaskResultSectionTitles)
	return out
}

// dedupPreservingOrder removes duplicates from a string slice while
// preserving first-seen order. Set + append pattern; returns nil for
// nil or empty input.
func dedupPreservingOrder(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// splitAndTrim splits on sep, trims whitespace from each part, and discards empties.
func splitAndTrim(s, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
