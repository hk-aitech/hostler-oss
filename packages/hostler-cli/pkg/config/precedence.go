// Package config — I10 Precedence Matrix + effective config collector.
//
// Centralises per-field precedence rules (env > yaml > default) in one place
// and assembles an EffectiveConfig snapshot so the validate-config CLI can
// report "which value came from where".
//
// Design principles:
// - Precedence rules are scattered across each getter, but the
// FieldDescriptor array in this file enumerates every field
// declaratively as a single matrix.
// - When a new env variable or YAML field is added, register it in this
// matrix as well so validate-config exposes it to the user.
// - This file calls the getter functions to collect the actual effective
// value, preserving the single-source rule (getter = truth).
package config

import (
	"os"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/events"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/template"
)

// FieldSource indicates where the effective value originated.
type FieldSource string

const (
	// SourceEnv — derived from an environment variable.
	SourceEnv FieldSource = "env"
	// SourceYAML — derived from project-config.yaml.
	SourceYAML FieldSource = "yaml"
	// SourceTemplate — derived from a templates/ file under brand.ProjectDirName.
	SourceTemplate FieldSource = "template"
	// SourceDefault — built-in default value.
	SourceDefault FieldSource = "default"
	// SourceUnset — no configured value (nil/empty).
	SourceUnset FieldSource = "unset"
)

// FieldDescriptor carries the precedence info and current effective value of a single config field.
type FieldDescriptor struct {
	// Name — display name shown to the user (e.g. "task_result_check.policy").
	Name string

	// Description — one-line description.
	Description string

	// EnvVars — list of environment variables that may override this field (in priority order).
	EnvVars []string

	// YAMLPath — path inside project-config.yaml (dot notation).
	YAMLPath string

	// Default — string representation of the built-in default.
	Default string

	// EffectiveValue — value actually used by the current process.
	EffectiveValue string

	// Source — origin of EffectiveValue.
	Source FieldSource

	// SourceDetail — supplemental info such as the env var name or yaml path.
	SourceDetail string
}

// EffectiveConfig holds the descriptors for every field plus aggregate metadata.
type EffectiveConfig struct {
	Fields []FieldDescriptor
	YAMLPath string // absolute path of the loaded yaml ("" when none)
	YAMLExists bool
	LoadError string // YAML parse error ("" when none)
	ValidationOK bool // schema validation result
	ValidationMsg string // validation failure message

	// : expose ValidationError Path / RecoveryHint separately.
	// ValidationMsg comes from Error() (Path+Message combined), but the CLI
	// needs separate field/message/hint lines, so split fields are required.
	ValidationPath string `json:"validation_path,omitempty"` // JSON Pointer (empty for root)
	ValidationHint string `json:"validation_hint,omitempty"` // user-facing recovery hint
}

// CollectEffective gathers the effective config for the current process.
// Reused by the validate-config CLI and the MCP tool config_inspect.
func CollectEffective() *EffectiveConfig {
	ec := &EffectiveConfig{}

	// YAML load state
	ec.YAMLPath = findProjectConfigYAML()
	ec.YAMLExists = ec.YAMLPath != ""
	if ec.YAMLExists {
		if _, verr, err := LoadProjectConfigValidated(); err != nil {
			ec.LoadError = err.Error()
		} else if verr != nil {
			ec.ValidationMsg = verr.Message
			ec.ValidationPath = verr.Path
			ec.ValidationHint = verr.RecoveryHint
		} else {
			ec.ValidationOK = true
		}
	} else {
		ec.ValidationOK = true // no file => nothing to validate
	}

	// Per-field collection
	ec.Fields = append(ec.Fields, collectTaskResultCheckPolicy())
	ec.Fields = append(ec.Fields, collectTaskResultSectionTitles())
	projectKeyField := collectProjectKey()
	ec.Fields = append(ec.Fields, projectKeyField)
	ec.Fields = append(ec.Fields, collectProjectRoot())
	ec.Fields = append(ec.Fields, collectRemindersEvents())
	ec.Fields = append(ec.Fields, collectBriefingSections())
	// — surface DB paths (KB A002 dual-DB safeguard).
	ec.Fields = append(ec.Fields, collectDBPathGlobal(projectKeyField.EffectiveValue))
	ec.Fields = append(ec.Fields, collectDBPathLocal())

	return ec
}

// ---------------------------------------------------------------------------
// Per-field collectors — each function must mirror the getter's precedence.
// ---------------------------------------------------------------------------

func collectTaskResultCheckPolicy() FieldDescriptor {
	d := FieldDescriptor{
		Name: "task_result_check.policy",
		Description: "task_complete result-section validation policy",
		EnvVars: []string{"HSTL_TASK_RESULT_CHECK_POLICY"},
		YAMLPath: "task_result_check.policy",
		Default: defaultTaskResultCheckPolicy,
	}
	if v := strings.TrimSpace(envalias.Lookup("TASK_RESULT_CHECK_POLICY")); v != "" {
		d.EffectiveValue = strings.ToLower(v)
		d.Source = SourceEnv
		d.SourceDetail = "HSTL_TASK_RESULT_CHECK_POLICY"
		return d
	}
	cfg, _ := LoadProjectConfig()
	if rc := cfg.GetTaskResultCheck(); rc != nil && rc.Policy != "" {
		d.EffectiveValue = rc.Policy
		d.Source = SourceYAML
		d.SourceDetail = "tasks.result_check.policy"
		return d
	}
	d.EffectiveValue = defaultTaskResultCheckPolicy
	d.Source = SourceDefault
	return d
}

func collectTaskResultSectionTitles() FieldDescriptor {
	d := FieldDescriptor{
		Name: "task_result_check.section_titles",
		Description: "list of markdown headings recognised as result sections",
		EnvVars: []string{"HSTL_TASK_RESULT_SECTION_TITLES"},
		YAMLPath: "tasks.result_check.section_titles",
		Default: strings.Join(defaultTaskResultSectionTitles, ","),
	}
	if v := strings.TrimSpace(envalias.Lookup("TASK_RESULT_SECTION_TITLES")); v != "" {
		d.EffectiveValue = v
		d.Source = SourceEnv
		d.SourceDetail = "HSTL_TASK_RESULT_SECTION_TITLES"
		return d
	}
	cfg, _ := LoadProjectConfig()
	if rc := cfg.GetTaskResultCheck(); rc != nil && len(rc.SectionTitles) > 0 {
		deduped := dedupPreservingOrder(rc.SectionTitles)
		d.EffectiveValue = strings.Join(deduped, ",")
		d.Source = SourceYAML
		d.SourceDetail = d.YAMLPath
		return d
	}
	d.EffectiveValue = d.Default
	d.Source = SourceDefault
	return d
}

func collectProjectKey() FieldDescriptor {
	d := FieldDescriptor{
		Name: "project.key",
		Description: "project identifier key (DB isolation key)",
		EnvVars: []string{"HSTL_PROJECT"},
		YAMLPath: "project.key",
		Default: "(git remote hash)",
	}
	if v := strings.TrimSpace(envalias.Lookup("PROJECT")); v != "" {
		d.EffectiveValue = v
		d.Source = SourceEnv
		d.SourceDetail = "HSTL_PROJECT"
		return d
	}
	cfg, _ := LoadProjectConfig()
	if cfg != nil && cfg.Project.Key != "" {
		d.EffectiveValue = cfg.Project.Key
		d.Source = SourceYAML
		d.SourceDetail = d.YAMLPath
		return d
	}
	d.EffectiveValue = d.Default
	d.Source = SourceDefault
	return d
}

func collectProjectRoot() FieldDescriptor {
	d := FieldDescriptor{
		Name: "project_root",
		Description: "project root directory (file-search base)",
		EnvVars: []string{"HSTL_PROJECT_ROOT"},
		YAMLPath: "(N/A — env-only)",
		Default: "(cwd)",
	}
	if v := strings.TrimSpace(envalias.Lookup("PROJECT_ROOT")); v != "" {
		d.EffectiveValue = v
		d.Source = SourceEnv
		d.SourceDetail = "HSTL_PROJECT_ROOT"
		return d
	}
	if cwd, err := os.Getwd(); err == nil {
		d.EffectiveValue = cwd
		d.Source = SourceDefault
		d.SourceDetail = "cwd"
	}
	return d
}

// collectRemindersEvents aggregates per-event reminders sources (template
// vs yaml) into a single summary entry. Computes template / yaml / unset
// for each event and renders the result as "event=source" pairs .
func collectRemindersEvents() FieldDescriptor {
	d := FieldDescriptor{
		Name: "reminders (template vs yaml)",
		Description: "per-event reminders load source (templates first, YAML fallback)",
		EnvVars: nil,
		YAMLPath: "reminders.*",
		Default: "(none)",
	}
	var parts []string
	allSources := map[FieldSource]int{}
	for _, e := range events.KnownEvents() {
		src := reminderSourceFor(string(e))
		allSources[src]++
		parts = append(parts, string(e)+"="+string(src))
	}
	d.EffectiveValue = strings.Join(parts, " ")
	// summary source: all template -> template, mix -> mixed, all unset -> unset
	switch {
	case allSources[SourceTemplate] == len(events.KnownEvents()):
		d.Source = SourceTemplate
		d.SourceDetail = brand.ProjectDirName + "/templates/reminders/*.md"
	case allSources[SourceUnset] == len(events.KnownEvents()):
		d.Source = SourceUnset
	default:
		d.Source = "mixed"
		d.SourceDetail = brand.ProjectDirName + "/templates/reminders/ + yaml reminders"
	}
	return d
}

// reminderSourceFor decides the load source for a single event's reminders (: templates only).
func reminderSourceFor(eventName string) FieldSource {
	if items, _, err := loadReminderTemplate(eventName); err == nil && len(items) > 0 {
		return SourceTemplate
	}
	return SourceUnset
}

// loadReminderTemplate is a thin indirection around template.LoadReminderContent.
// Tests can swap this variable out when mocking is required.
var loadReminderTemplate = func(eventName string) ([]string, any, error) {
	items, meta, err := template.LoadReminderContent(eventName)
	return items, meta, err
}

// collectBriefingSections summarises the enabled count of briefing.sections.
func collectBriefingSections() FieldDescriptor {
	d := FieldDescriptor{
		Name: "briefing.sections",
		Description: "project-briefing section count (enabled/total)",
		YAMLPath: "briefing.sections",
		Default: "(none)",
	}
	cfg, _ := LoadProjectConfig()
	if cfg == nil || cfg.Briefing == nil || len(cfg.Briefing.Sections) == 0 {
		d.EffectiveValue = d.Default
		d.Source = SourceUnset
		return d
	}
	enabled := 0
	for _, s := range cfg.Briefing.Sections {
		if s.Enabled {
			enabled++
		}
	}
	d.EffectiveValue = briefingSummary(enabled, len(cfg.Briefing.Sections))
	d.Source = SourceYAML
	d.SourceDetail = d.YAMLPath
	return d
}

// collectDBPathGlobal — , KB A002 dual-DB safeguard.
// The machine-global DB path actually read by hstl. Detects HSTL_DB_PATH
// override. projectKey is supplied by the already-resolved
// collectProjectKey to avoid a circular reference.
func collectDBPathGlobal(projectKey string) FieldDescriptor {
	d := FieldDescriptor{
		Name: "db.path.global",
		Description: "machine-global SQLite DB path read by hstl (SSOT)",
		EnvVars: []string{"HSTL_DB_PATH"},
		// : machine-global is `~/.hostler/data/{key}/hstl.db` (matches pkg/db GetDBPath).
		Default: "~/.hostler/data/{project.key}/hstl.db",
	}
	// HSTL_DB_PATH overrides the default location.
	if v := strings.TrimSpace(envalias.Lookup("DB_PATH")); v != "" {
		d.EffectiveValue = v
		d.Source = SourceEnv
		d.SourceDetail = "HSTL_DB_PATH"
		return d
	}
	home, _ := os.UserHomeDir()
	d.EffectiveValue = home + "/.hostler/data/" + projectKey + "/hstl.db"
	d.Source = SourceDefault
	d.SourceDetail = "~/.hostler/data/" + projectKey + "/hstl.db"
	return d
}

// collectDBPathLocal — , KB A002.
// repo-local secondary DB path. : routes through
// brand.ProjectDirName (`.hstl/`). Not the SSOT — synced cache only.
func collectDBPathLocal() FieldDescriptor {
	localRel := brand.ProjectDirName + "/db.sqlite" // `.hstl/db.sqlite`
	d := FieldDescriptor{
		Name: "db.path.local",
		Description: "repo-local secondary DB path (" + localRel + ") — not the SSOT",
		YAMLPath: "",
		Default: localRel,
	}
	root, _ := os.Getwd()
	d.EffectiveValue = root + "/" + localRel
	d.Source = SourceDefault
	d.SourceDetail = localRel + " (secondary copy)"
	return d
}

// briefingSummary returns the "N/M" summary string. Split out for test readability.
func briefingSummary(enabled, total int) string {
	return intToStr(enabled) + "/" + intToStr(total) + " enabled"
}

// intToStr converts int->string without fmt.Sprintf to minimise dependencies.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
