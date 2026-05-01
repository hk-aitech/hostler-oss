// Package config defines the project-config.yaml schema.
//
// The hstl plugin manages per-project settings under
// .hstl/project-config.yaml. This file declares the Go types
// corresponding to that YAML schema.
//
// SSOT: .hstl-oss/project-config.yaml ( MD fallback fully removed;
// cutover).
//
//  JSON Schema auto-generation.
// When types.go changes, `go generate ./pkg/config/...` regenerates
// schemas/project-config/v2.0.0.schema.json. CI blocks any drift.
//
//go:generate go run ../../cmd/gen-schema
package config

// ProjectConfig represents the entire .hstl-oss/project-config.yaml schema
//
//
// All fields are optional — missing sections load as zero values / nil and
// the caller falls back to defaults or returns an empty result.
//
// JSON tags exist for google/jsonschema-go schema generation and every
// field has omitempty so all fields are marked optional. yaml.v3 only
// reads the yaml tag, so coexistence has no effect on YAML loading.
type ProjectConfig struct {
	// Version is the SemVer schema version.
	//
	// e.g. "2.0.0", "2.1.0", "3.0.0"
	//
	// Loader compatibility:
	//   - major mismatch → fail-fast candidate (warns,
	//     fails).
	//   - minor diff → warning + compatible load.
	//   - patch diff → ignored.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	//  removed 9 deprecated v1 fields, the dead DeploymentGroups field,
	// and 5 fields under ProjectMeta . Verified 0 backward-compat
	// consumers . The Migration Runner was removed too, leaving a
	// canonical v2-only schema.

	// Extends is the preset-inheritance list of Profile Composition
	// . Items use the form "builtin:<name>" or a relative path
	// like "profiles/<file>.yaml".
	Extends []string `yaml:"extends,omitempty" json:"extends,omitempty"`

	// Project holds project metadata.
	Project ProjectMeta `yaml:"project" json:"project,omitzero"`

	// Platform is the project's tech-platform tagged union .
	// The Kind field decides which nested struct is valid.
	Platform *Platform `yaml:"platform,omitempty" json:"platform,omitempty"`

	// DeploymentGroups defines deployment groups. No code consumer exists
	// today, but it is kept descriptive in the YAML so the schema accepts
	// it.
	DeploymentGroups []DeploymentGroup `yaml:"deployment_groups,omitempty" json:"deployment_groups,omitempty"`

	// Tasks holds Task-lifecycle settings for hstl Identity Layer 1
	// .
	Tasks *TasksConfig `yaml:"tasks,omitempty" json:"tasks,omitempty"`

	// Sprints holds Sprint-lifecycle settings for hstl Identity Layer 1
	// .
	Sprints *SprintsConfig `yaml:"sprints,omitempty" json:"sprints,omitempty"`

	// Skills is the per-skill configuration namespace (
	// Skill-First).
	Skills map[string]*SkillConfig `yaml:"skills,omitempty" json:"skills,omitempty"`

	// Briefing is the section definition for the project-briefing skill.
	Briefing *BriefingConfig `yaml:"briefing,omitempty" json:"briefing,omitempty"`

	// Registry holds resource-registry path settings .
	// YAML example:
	//   registry:
	//     path: /custom/path/to/registry.json
	Registry *RegistryConfig `yaml:"registry,omitempty" json:"registry,omitempty"`

	// Layout overrides the project directory layout .
	// When unset, the default hstl convention (works/sprints, works/tasks)
	// is used.
	// YAML example:
	//   layout:
	//     sprints_root: works/sprints
	//     tasks_root: works/tasks
	Layout *LayoutConfig `yaml:"layout,omitempty" json:"layout,omitempty"`

	// Output configures the CLI output format 
	// from hstl-003).
	// YAML example:
	//   output:
	//     default_format: json   # json | text | console
	Output *OutputConfig `yaml:"output,omitempty" json:"output,omitempty"`

	// PacingGuard configures the ai-pair-v1 Pacing Guard principle
	// . On a successful sprint:complete, a rest-recommend
	// reminder block is automatically appended at the end of the response,
	// nudging the user to consciously consider a pause.
	PacingGuard *PacingGuardConfig `yaml:"pacing_guard,omitempty" json:"pacing_guard,omitempty"`

	// Precommit holds precommit rule threshold settings .
	// branch-sync thresholds vary per project, so persisting them to
	// project-config.yaml supports team sharing, CI consistency, and
	// reproducibility for new developers. Env vars
	// (e.g. HSTL_BRANCH_SYNC_THRESHOLD) take precedence when set.
	Precommit *PrecommitConfig `yaml:"precommit,omitempty" json:"precommit,omitempty"`

	// SprintStart configures Sprint-start ceremony thresholds
	// . divergence_warn_commits is the WARN threshold
	// for dev↔sprint-base divergence. The
	// HSTL_SPRINT_START_DIVERGENCE_WARN_COMMITS env var takes precedence.
	SprintStart *SprintStartConfig `yaml:"sprint_start,omitempty" json:"sprint_start,omitempty"`
}

// PrecommitConfig is the precommit rule threshold namespace
//
type PrecommitConfig struct {
	// BranchSync configures the precommit.branch.sync rule threshold.
	BranchSync *BranchSyncConfig `yaml:"branch_sync,omitempty" json:"branch_sync,omitempty"`
}

// BranchSyncConfig configures precommit.branch.sync thresholds
//
//
// Pointer types are used so that "unset" is interpreted as "use default
// value", distinguishing the zero value (0 / false) from an explicit
// "0 or false" input.
type BranchSyncConfig struct {
	// Threshold is the BLOCK-trigger commit count. Default 30.
	// HSTL_BRANCH_SYNC_THRESHOLD takes precedence.
	Threshold *int `yaml:"threshold,omitempty" json:"threshold,omitempty"`

	// WarnRatio is the WARN threshold as a percentage of Threshold.
	// Default 80.
	// e.g. threshold=30, warn_ratio=80 → WARN ≥ 24, BLOCK ≥ 30.
	// HSTL_BRANCH_SYNC_WARN_RATIO takes precedence.
	WarnRatio *int `yaml:"warn_ratio,omitempty" json:"warn_ratio,omitempty"`

	// Enabled is the rule on/off switch. false is equivalent to
	// HSTL_BRANCH_SYNC=off. nil (unset) defaults to enabled=true. The
	// HSTL_BRANCH_SYNC=off env var takes precedence.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// StrictMode disables the sprint-*/hotfix- branch exemption (default
	// off). HSTL_BRANCH_SYNC_STRICT=1 takes precedence.
	StrictMode *bool `yaml:"strict_mode,omitempty" json:"strict_mode,omitempty"`
}

// SprintStartConfig configures Sprint-start ceremony settings
//
type SprintStartConfig struct {
	// DivergenceWarnCommits is the WARN threshold for dev↔sprint-base
	// divergence. Default 25.
	// HSTL_SPRINT_START_DIVERGENCE_WARN_COMMITS takes precedence.
	DivergenceWarnCommits *int `yaml:"divergence_warn_commits,omitempty" json:"divergence_warn_commits,omitempty"`
}

// GetBranchSyncConfig returns the BranchSync settings. nil-safe.
func (c *ProjectConfig) GetBranchSyncConfig() *BranchSyncConfig {
	if c == nil || c.Precommit == nil {
		return nil
	}
	return c.Precommit.BranchSync
}

// GetSprintStartConfig returns the SprintStart settings. nil-safe.
func (c *ProjectConfig) GetSprintStartConfig() *SprintStartConfig {
	if c == nil {
		return nil
	}
	return c.SprintStart
}

// PacingGuardConfig configures the ai-pair-v1 Pacing Guard principle .
type PacingGuardConfig struct {
	// ReminderOnSprintComplete controls whether a Pacing Guard reminder
	// block is emitted on sprint:complete success. nil or unset defaults
	// to true.
	ReminderOnSprintComplete *bool `yaml:"reminder_on_sprint_complete,omitempty" json:"reminder_on_sprint_complete,omitempty"`
}

// OutputConfig configures the CLI output format .
//
// Precedence: CLI flag → HSTL_OUTPUT_FORMAT env → output.default_format →
// built-in default (json, AI-first).
type OutputConfig struct {
	// DefaultFormat is the default CLI output format.
	// Values: "json" | "text" | "console". The aliases "txt" / "con" are
	// accepted for "text" / "console" respectively.
	DefaultFormat string `yaml:"default_format,omitempty" json:"default_format,omitempty"`
}

// GetOutputDefaultFormat returns OutputConfig.DefaultFormat. nil-safe.
// When empty, the caller must fall back to the built-in default.
func (c *ProjectConfig) GetOutputDefaultFormat() string {
	if c == nil || c.Output == nil {
		return ""
	}
	return c.Output.DefaultFormat
}

// GetPacingGuardReminderOnSprintComplete returns whether the Pacing Guard
// reminder is emitted on sprint:complete . When unset or
// when c is nil, defaults to true (opt-out — preventing fatigue is the
// default).
func (c *ProjectConfig) GetPacingGuardReminderOnSprintComplete() bool {
	if c == nil || c.PacingGuard == nil || c.PacingGuard.ReminderOnSprintComplete == nil {
		return true
	}
	return *c.PacingGuard.ReminderOnSprintComplete
}

// LayoutConfig overrides the project directory layout .
//
// Purpose: distinguish the hstl convention (works/) from the pure hostler
// convention (hostler/) or special monorepo paths. When unset, defaults
// to works/ (hstl-compatible).
//
// Callers obtain default-resolved values via GetLayout() or
// ResolvedSprintsRoot() / ResolvedTasksRoot().
type LayoutConfig struct {
	// SprintsRoot is the Sprint root directory (relative to project root).
	// Default: "works/sprints"
	SprintsRoot string `yaml:"sprints_root,omitempty" json:"sprints_root,omitempty"`

	// TasksRoot is the Task root directory (location of unassigned tasks,
	// relative to project root).
	// Default: "works/tasks"
	TasksRoot string `yaml:"tasks_root,omitempty" json:"tasks_root,omitempty"`
}

// GetLayout returns the Layout settings. nil-safe.
func (c *ProjectConfig) GetLayout() *LayoutConfig {
	if c == nil {
		return nil
	}
	return c.Layout
}

// ResolvedSprintsRoot returns the Sprint root path (default if no config).
func (c *ProjectConfig) ResolvedSprintsRoot() string {
	if c == nil || c.Layout == nil || c.Layout.SprintsRoot == "" {
		return "works/sprints"
	}
	return c.Layout.SprintsRoot
}

// ResolvedTasksRoot returns the Task root path (default if no config).
func (c *ProjectConfig) ResolvedTasksRoot() string {
	if c == nil || c.Layout == nil || c.Layout.TasksRoot == "" {
		return "works/tasks"
	}
	return c.Layout.TasksRoot
}

// RegistryConfig holds the resource-registry path settings .
//
// Provides Runtime and Docs path overrides via YAML. Precedence:
//  1. HSTL_REGISTRY_PATH / HSTL_REGISTRY_DOCS_PATH env vars
//  2. registry.path / registry.docs_path (YAML)
//  3. defaults (~/{brand.ProjectDirName}/data/{project_key}/registry.json )
type RegistryConfig struct {
	// Path overrides the runtime registry.json location.
	// Default: ~/{brand.ProjectDirName}/data/{project_key}/registry.json
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	// DocsPath overrides the git-managed snapshot path.
	// Default: {project_root}/{brand.ProjectDirName}/data/registry.json
	DocsPath string `yaml:"docs_path,omitempty" json:"docs_path,omitempty"`
}

// GetRegistry returns the Registry settings. nil-safe.
func (c *ProjectConfig) GetRegistry() *RegistryConfig {
	if c == nil {
		return nil
	}
	return c.Registry
}

// TasksConfig is the Task-lifecycle namespace for hstl Identity Layer 1
//
//
// YAML path:
//
//	tasks:
//	  events:            # moved from reminders.task.*
//	    start:    [...]
//	    complete: [...]
//	  result_check:      # moved from task_result_check
//	    policy: strict
//	    section_titles: [Result, Deliverables]
type TasksConfig struct {
	// Events maps each event to a list of reminder items.
	// keys: "start" | "complete" (runtime event ID is "task.{key}").
	Events map[string][]string `yaml:"events,omitempty" json:"events,omitempty"`

	// ResultCheck is the task_complete result-section validation policy.
	ResultCheck *TaskResultCheckConfig `yaml:"result_check,omitempty" json:"result_check,omitempty"`
}

// SprintsConfig is the Sprint-lifecycle namespace for hstl Identity Layer 1
//
//
// YAML path:
//
//	sprints:
//	  events:            # moved from reminders.sprint.*
//	    start:    [...]
//	    complete: [...]
//	  ceremony:          # moved from sprint_ceremony
//	    start:    [...]
//	    complete: [...]
type SprintsConfig struct {
	// Events maps each event to a list of reminder items.
	// keys: "start" | "complete" (runtime event ID is "sprint.{key}").
	Events map[string][]string `yaml:"events,omitempty" json:"events,omitempty"`

	// Ceremony is the start/complete checklist for a Sprint.
	Ceremony *SprintCeremony `yaml:"ceremony,omitempty" json:"ceremony,omitempty"`

	// SizeMode is the Sprint Tier classification .
	// Four tiers: "solo" (default) | "standard" | "pair" | "squad".
	// When unset, interpreted as "solo" (default for autonomous solo-AI
	// workflow).
	// SSOT: docs/08-references/standards/sprint-tier-spec.md
	SizeMode string `yaml:"size_mode,omitempty" json:"size_mode,omitempty"`

	// SizeOverride explicitly overrides SizeMode defaults .
	// nil fields use the SizeMode default. Partial overrides are allowed.
	SizeOverride *SprintSizeOverride `yaml:"size_override,omitempty" json:"size_override,omitempty"`

	// CompleteMode controls the sprint:complete ceremony mode.
	// "confirm" (default) | "lite". lite enables an automatic stamp
	// fast-path when there are zero KB+Task candidates.
	// Precedence: CLI flag --lite > env HSTL_SPRINT_COMPLETE_MODE >
	// config value > built-in "confirm".
	// SSOT: docs/08-references/standards/sprint-tier-spec.md (separate
	// SSOT review pending).
	CompleteMode string `yaml:"complete_mode,omitempty" json:"complete_mode,omitempty"`
}

// SprintSizeOverride overrides SizeMode defaults .
//
// YAML example:
//
//	sprints:
//	  size_mode: standard
//	  size_override:
//	    max_task_count: 10   # standard default 7 → 10
//	    max_capacity: 25     # standard default 20 → 25
type SprintSizeOverride struct {
	// MinTaskCount is the minimum Task count of a Sprint (inclusive).
	// nil falls back to the SizeMode default.
	MinTaskCount *int `yaml:"min_task_count,omitempty" json:"min_task_count,omitempty"`

	// MaxTaskCount is the maximum Task count of a Sprint (inclusive).
	// nil falls back to the SizeMode default. 0 means "unlimited".
	MaxTaskCount *int `yaml:"max_task_count,omitempty" json:"max_task_count,omitempty"`

	// MaxCapacity is the maximum Sprint capacity (sum of points).
	// nil falls back to the SizeMode default. 0 means "unlimited".
	MaxCapacity *int `yaml:"max_capacity,omitempty" json:"max_capacity,omitempty"`
}

// GetSprintCeremony returns the Sprint ceremony settings.
func (c *ProjectConfig) GetSprintCeremony() *SprintCeremony {
	if c == nil || c.Sprints == nil {
		return nil
	}
	return c.Sprints.Ceremony
}

// GetTaskResultCheck returns the Task result-section validation policy.
func (c *ProjectConfig) GetTaskResultCheck() *TaskResultCheckConfig {
	if c == nil || c.Tasks == nil {
		return nil
	}
	return c.Tasks.ResultCheck
}

// ProjectMeta holds project metadata.
//
//  removed dead fields with zero consumers (Name,
// IdentitySource, ADRDir, SolutionFile). Only Key is
// referenced (by GetProjectKey()).
type ProjectMeta struct {
	// Key is the project identifier (e.g. "my-plugin").
	// Must match the HSTL_PROJECT env var.
	Key string `yaml:"key" json:"key,omitempty"`

	// Facets is the facet definition map for ADR-047 Project Context
	// 15-Facet. The actual strict parsing is done separately in
	// pkg/facet/yaml_load.go via yamlConfigV2; this field is just a
	// placeholder map so the config validator does not reject
	// project.facets as an unknown property  dogfood —
	// fix discovered when registering the Plugin SDK dev-server-state
	// facet).
	Facets map[string]any `yaml:"facets,omitempty" json:"facets,omitempty"`
}

// Platform is the project's tech-platform tagged union .
//
// Kind decides which nested struct is valid:
//   - "go"     → Go field is valid
//   - "dotnet" → Dotnet field is valid
//   - "python" → Python field is valid
//   - "node"   → Node field is valid
//
// Validate() verifies that Kind matches the populated nested struct.
type Platform struct {
	// Kind is the platform type. Required.
	Kind string `yaml:"kind" json:"kind,omitempty"`

	// Go is Go-project-only configuration. Valid only when Kind=="go".
	Go *PlatformGo `yaml:"go,omitempty" json:"go,omitempty"`

	// Dotnet is .NET-project-only configuration. Valid only when
	// Kind=="dotnet".
	Dotnet *PlatformDotnet `yaml:"dotnet,omitempty" json:"dotnet,omitempty"`

	// Python is Python-project-only configuration. Valid only when
	// Kind=="python".
	Python *PlatformPython `yaml:"python,omitempty" json:"python,omitempty"`

	// Node is Node.js/TypeScript-project-only configuration. Valid only
	// when Kind=="node".
	Node *PlatformNode `yaml:"node,omitempty" json:"node,omitempty"`
}

// PlatformGo is Go-project-only configuration.
type PlatformGo struct {
	// Module is the go.mod module path (e.g. "github.com/example/plugin").
	Module string `yaml:"module,omitempty" json:"module,omitempty"`

	// Lint is the list of Go lint tools to run.
	// e.g. ["gofmt", "govet", "staticcheck"]
	Lint []string `yaml:"lint,omitempty" json:"lint,omitempty"`
}

// PlatformDotnet is .NET-project-only configuration.
//
// absorbed five existing top-level fields into this
// struct: solution_file, bounded_contexts, shared_projects,
// structure_rules, roslyn_analyzers.
type PlatformDotnet struct {
	// SolutionFile is the .NET solution file path (e.g. "Project.slnx").
	SolutionFile string `yaml:"solution_file,omitempty" json:"solution_file,omitempty"`

	// BoundedContexts is the BC list (1:1 mapped to .NET projects).
	BoundedContexts []BoundedContext `yaml:"bounded_contexts,omitempty" json:"bounded_contexts,omitempty"`

	// SharedProjects is the list of cross-cutting non-BC projects
	// (SharedKernel, Contracts, etc.).
	SharedProjects []SharedProject `yaml:"shared_projects,omitempty" json:"shared_projects,omitempty"`

	// StructureRules is the list of project structure rules
	// (e.g. BC = project 1:1).
	StructureRules []StructureRule `yaml:"structure_rules,omitempty" json:"structure_rules,omitempty"`

	// RoslynAnalyzers is the list of .NET Roslyn static-analysis rules.
	RoslynAnalyzers []RoslynAnalyzer `yaml:"roslyn_analyzers,omitempty" json:"roslyn_analyzers,omitempty"`
}

// PlatformPython is Python-project-only configuration.
type PlatformPython struct {
	// Package is the package name from pyproject.toml.
	Package string `yaml:"package,omitempty" json:"package,omitempty"`

	// Lint is the list of Python lint tools to run
	// (e.g. ["ruff", "mypy"]).
	Lint []string `yaml:"lint,omitempty" json:"lint,omitempty"`
}

// PlatformNode is Node.js/TypeScript-project-only configuration.
type PlatformNode struct {
	// Package is the package.json name field.
	Package string `yaml:"package,omitempty" json:"package,omitempty"`

	// Lint is the list of JS/TS lint tools to run
	// (e.g. ["eslint", "biome"]).
	Lint []string `yaml:"lint,omitempty" json:"lint,omitempty"`
}

// GetBoundedContexts returns BoundedContexts (v2-only).
func (c *ProjectConfig) GetBoundedContexts() []BoundedContext {
	if c == nil || c.Platform == nil || c.Platform.Dotnet == nil {
		return nil
	}
	return c.Platform.Dotnet.BoundedContexts
}

// GetSharedProjects returns SharedProjects.
func (c *ProjectConfig) GetSharedProjects() []SharedProject {
	if c == nil || c.Platform == nil || c.Platform.Dotnet == nil {
		return nil
	}
	return c.Platform.Dotnet.SharedProjects
}

// GetStructureRules returns StructureRules.
func (c *ProjectConfig) GetStructureRules() []StructureRule {
	if c == nil || c.Platform == nil || c.Platform.Dotnet == nil {
		return nil
	}
	return c.Platform.Dotnet.StructureRules
}

// GetRoslynAnalyzers returns RoslynAnalyzers.
func (c *ProjectConfig) GetRoslynAnalyzers() []RoslynAnalyzer {
	if c == nil || c.Platform == nil || c.Platform.Dotnet == nil {
		return nil
	}
	return c.Platform.Dotnet.RoslynAnalyzers
}

// GetSolutionFile returns SolutionFile.
func (c *ProjectConfig) GetSolutionFile() string {
	if c == nil || c.Platform == nil || c.Platform.Dotnet == nil {
		return ""
	}
	return c.Platform.Dotnet.SolutionFile
}

// IsGo reports whether the project uses the Go platform.
func (c *ProjectConfig) IsGo() bool {
	return c != nil && c.Platform != nil && c.Platform.Kind == PlatformKindGo
}

// IsDotnet reports whether the project uses the .NET platform.
func (c *ProjectConfig) IsDotnet() bool {
	return c != nil && c.Platform != nil && c.Platform.Kind == PlatformKindDotnet
}

// IsPython reports whether the project uses the Python platform.
func (c *ProjectConfig) IsPython() bool {
	return c != nil && c.Platform != nil && c.Platform.Kind == PlatformKindPython
}

// IsNode reports whether the project uses the Node.js/TypeScript platform.
func (c *ProjectConfig) IsNode() bool {
	return c != nil && c.Platform != nil && c.Platform.Kind == PlatformKindNode
}

// Platform kind constants.
const (
	PlatformKindGo     = "go"
	PlatformKindDotnet = "dotnet"
	PlatformKindPython = "python"
	PlatformKindNode   = "node"
)

// BoundedContext is a BC definition.
type BoundedContext struct {
	// ID is the BC abbreviation (e.g. "COL", "STR", "TRD").
	ID string `yaml:"id" json:"id,omitempty"`

	// Name is the full BC name (e.g. "Collection", "Strategy").
	Name string `yaml:"name" json:"name,omitempty"`

	// Project is the .NET project name (e.g. "MyProject.Collection").
	Project string `yaml:"project" json:"project,omitempty"`

	// Description is the BC description.
	Description string `yaml:"description" json:"description,omitempty"`
}

// SharedProject is a non-BC, cross-cutting project.
type SharedProject struct {
	// ID is the project identifier (e.g. "SharedKernel", "Contracts").
	ID string `yaml:"id" json:"id,omitempty"`

	// Description is the project description.
	Description string `yaml:"description" json:"description,omitempty"`
}

// StructureRule is a project structure rule.
type StructureRule struct {
	// ID is the rule identifier (e.g. "bc-project-1to1").
	ID string `yaml:"id" json:"id,omitempty"`

	// Rule is the one-line summary of the rule.
	Rule string `yaml:"rule" json:"rule,omitempty"`

	// Detail is the detailed description.
	Detail string `yaml:"detail" json:"detail,omitempty"`

	// ADR is the related ADR identifier (e.g. "ADR-014").
	ADR string `yaml:"adr" json:"adr,omitempty"`
}

// DeploymentGroup is a deployment-group definition.
type DeploymentGroup struct {
	// ID is the group identifier.
	ID string `yaml:"id" json:"id,omitempty"`

	// Name is the group display name.
	Name string `yaml:"name" json:"name,omitempty"`

	// Path describes the deployment path (e.g. "LocalDev → Prod direct").
	Path string `yaml:"path" json:"path,omitempty"`

	// Process describes the process (including ports).
	Process string `yaml:"process" json:"process,omitempty"`
}

// RoslynAnalyzer is a .NET Roslyn static-analysis rule definition.
type RoslynAnalyzer struct {
	// ID is the rule ID ((e.g. example string)).
	ID string `yaml:"id" json:"id,omitempty"`

	// Rule is the rule description.
	Rule string `yaml:"rule" json:"rule,omitempty"`

	// Severity is the severity (Error, Warning, Info).
	Severity string `yaml:"severity" json:"severity,omitempty"`
}

// SprintCeremony customizes Sprint start/complete procedures.
//
// Each item is a list of strings rendered as checkbox lines by
// RenderCeremonyMD. The "{sprint_id}" token inside an item is replaced
// with the actual Sprint ID (e.g. "sprint-22").
type SprintCeremony struct {
	// Start is the list of start-procedure items.
	// e.g. ["git checkout -b {sprint_id}", "run sprint:start", ...]
	Start []string `yaml:"start" json:"start,omitempty"`

	// Complete is the list of complete-procedure (sprint:complete Phase)
	// items.
	// e.g. ["Phase 1: doc-review --sprint", "Phase 2: code-review --sprint", ...]
	Complete []string `yaml:"complete" json:"complete,omitempty"`

	// DesignChecks is the list of project-specific extension script paths
	// to run during the sprint_start ceremony's design_readiness step
	// .
	//
	// Each path is relative to the project root and must be executable.
	// ceremony/collect.go runs them as subprocesses, passing the Task list
	// JSON on stdin and parsing each stdout line as
	// "check|status|detail" into a ReadinessCheck (status:
	// pass|warn|info|fail).
	//
	// e.g.
	//   sprints:
	//     ceremony:
	//       design_checks:
	//         - .hstl-oss/scripts/check-time-constraints.sh
	//
	// Goal: avoid hard-coding project-specific rules into hstl core; defer
	// project-specific validation to external scripts (refactor).
	DesignChecks []string `yaml:"design_checks,omitempty" json:"design_checks,omitempty"`

	// StaleBodyDays is the threshold (in days) sprint_start ceremony uses
	// to judge "Task body authored ↔ start gap" .
	// Default 30. If Task.CreatedAt vs the sprint.start time differs by at
	// least this many days, a "body re-review recommended" warning is
	// added to readiness check.
	//
	// e.g. a Task authored in starting in — its body
	// is a snapshot from several Sprints back, so part of the
	// requirements may already be resolved by intervening changes.
	StaleBodyDays int `yaml:"stale_body_days,omitempty" json:"stale_body_days,omitempty"`
}

// SkillConfig is a single skill's configuration namespace
// (Skill-First).
//
// only uses the Rules field; future fields will be added
// incrementally:
//   - Extends  []string   — preset inheritance
//   - Sections []BriefingSection — moved from project-briefing skill
//   - Enabled  bool       — skill on/off
//
// YAML example:
//
//	skills:
//	  code-review:
//	    rules:
//	      - "Go standard — must pass gofmt and go vet"
//	      - "Detect magic constants"
type SkillConfig struct {
	// Rules is the list of skill rule strings. Same format as the legacy
	// skill_extensions.{name}; the Migration Runner ports automatically.
	Rules []string `yaml:"rules,omitempty" json:"rules,omitempty"`
}

// GetSkillRules returns a list of rule strings for the given skill
// (backward-compat helper).
//
//  v2-only — references only Skills.{name}.Rules.
func (c *ProjectConfig) GetSkillRules(name string) []string {
	if c == nil || name == "" || c.Skills == nil {
		return nil
	}
	if sc, ok := c.Skills[name]; ok && sc != nil {
		return sc.Rules
	}
	return nil
}

// BriefingConfig is the section definition of the project-briefing skill.
//
// merged the standalone briefing-config.yaml into the
// project-config.yaml.briefing field, and fully removed
// the standalone file, leaving this struct as the SSOT. Used together
// with the dynamic aggregation strategy (extract) introduced in 
type BriefingConfig struct {
	// Sections is the list of sections to include in the briefing
	// (rendered in order).
	Sections []BriefingSection `yaml:"sections" json:"sections,omitempty"`
}

// BriefingSection is a single briefing section definition.
type BriefingSection struct {
	// ID is the section identifier
	// (e.g. "identity", "architecture", "current_sprint").
	ID string `yaml:"id" json:"id,omitempty"`

	// Enabled controls whether the section is active (false hides it).
	Enabled bool `yaml:"enabled" json:"enabled,omitempty"`

	// Source is the section's data source path (multiple paths comma-
	// separated).
	Source string `yaml:"source" json:"source,omitempty"`

	// Type is the section type (static / semi-static / dynamic, etc.).
	// Default "static".
	Type string `yaml:"type,omitempty" json:"type,omitempty"`

	// Extract is the dynamic-extraction strategy (defined in).
	// e.g. "section:## Goals", "first-heading-block", "markdown-table",
	// "file:head(50)"
	Extract string `yaml:"extract,omitempty" json:"extract,omitempty"`

	// MaxItems is the maximum number of items to render
	// (0 means unlimited).
	MaxItems int `yaml:"max_items,omitempty" json:"max_items,omitempty"`

	// HighlightCount is the number of top items to highlight.
	HighlightCount int `yaml:"highlight_count,omitempty" json:"highlight_count,omitempty"`

	// RecentCount is the number of recent items (for dynamic sections).
	RecentCount int `yaml:"recent_count,omitempty" json:"recent_count,omitempty"`
}

// TaskResultCheckConfig is the task_complete result-section validation
// policy.
//
// First fully used by  The HSTL_TASK_RESULT_CHECK_POLICY and
// HSTL_TASK_RESULT_SECTION_TITLES env vars take precedence.
type TaskResultCheckConfig struct {
	// Policy is the validation policy. Values: "off" | "warn" | "strict".
	// Default (when this field is empty): "strict".
	Policy string `yaml:"policy" json:"policy,omitempty"`

	// SectionTitles is the list of markdown headings recognized as
	// result sections.
	// Default (when this field is empty): ["Result", "Deliverables"].
	SectionTitles []string `yaml:"section_titles" json:"section_titles,omitempty"`

	// DriftLessPaths is the list of path patterns considered "drift-less
	// regenerated" by strict validation: when the file actually exists
	// and the suffix matches, it is excluded from the missing list even
	// if absent from git diff . Prevents false-positive
	// BLOCKS for files like manifest.json / skill-trigger-index.md whose
	// regenerated output is identical and therefore produces no diff.
	DriftLessPaths []string `yaml:"drift_less_paths,omitempty" json:"drift_less_paths,omitempty"`
}
