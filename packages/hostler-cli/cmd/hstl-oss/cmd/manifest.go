// trac: OPS-CM001,OPS-FT001,OPS-FT002,OPS-FT003,OPS-FT004,OPS-OP001
// trac: OPS-OP002,OPS-OP003,OPS-OP004,OPS-OP005,OPS-OP006,OPS-OP007,OPS-OP008
// trac: OPS-QR002
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// manifestSchemaURI is the JSON Schema draft-07 spec URI.
const manifestSchemaURI = "http://json-schema.org/draft-07/schema#"

// manifestAnnotationCeremony is the Annotations key marking a ceremony-bearing command.
const manifestAnnotationCeremony = "ceremony"

// manifestAnnotationCommandRef is the Annotations key for the corresponding Command/Skill path.
const manifestAnnotationCommandRef = "command_ref"

// pluginManifestPath is the source for the semantic version used in the
// manifest Version field. Filling Version with the build-time -ldflags
// commit hash made manifest.json drift on every commit, causing false
// positives in check-manifest. The plugin.json version (e.g., "3.0.0")
// changes only at release time, dramatically improving the drift
// signal-to-noise ratio.
const pluginManifestPath = ".claude-plugin/plugin.json"

// pluginManifestUnknown is the fallback value when plugin.json cannot be read.
const pluginManifestUnknown = "unknown"

// readPluginVersion reads the version field from .claude-plugin/plugin.json.
// On missing file or parse failure, falls back gracefully to
// pluginManifestUnknown - manifest generation must continue, so we do not
// promote this to an error.
func readPluginVersion() string {
	root := app.ProjectRoot()
	if root == "" {
		return pluginManifestUnknown
	}
	data, err := os.ReadFile(filepath.Join(root, pluginManifestPath))
	if err != nil {
		return pluginManifestUnknown
	}
	var meta struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &meta); err != nil || meta.Version == "" {
		return pluginManifestUnknown
	}
	return meta.Version
}

// ---------------------------------------------------------------------------
// Output structures
// ---------------------------------------------------------------------------

// ManifestOutput is the top-level structure of the --manifest output.
type ManifestOutput struct {
	Schema   string        `json:"$schema"`
	Version  string        `json:"version"`
	Commands []CommandMeta `json:"commands"`
}

// CommandMeta is the metadata for each command.
type CommandMeta struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	PositionalArgs []PositionalArg `json:"positional_args,omitempty"`
	Args           *CommandArgs    `json:"args,omitempty"`
	Output         *CommandArgs    `json:"output,omitempty"`
	Ceremony       bool            `json:"ceremony"`
	CeremonySchema *CommandArgs    `json:"ceremony_schema,omitempty"`
	CommandRef     string          `json:"command_ref,omitempty"`
	CLIUsage       string          `json:"cli_usage"`
}

// PositionalArg is positional-argument metadata.
type PositionalArg struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
}

// CommandArgs is the JSON Schema for command arguments.
type CommandArgs struct {
	Type       string                 `json:"type"`
	Properties map[string]ArgProperty `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// ArgProperty is the JSON Schema property for an individual argument.
type ArgProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     any    `json:"default,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
}

// ---------------------------------------------------------------------------
// --schema output structures (JSON Schema draft-07 standard)
// ---------------------------------------------------------------------------

// SchemaOutput is the top-level --schema output structure for root/branch scopes.
// Aligns with the Root object in manifest-schema-contract.md, sec. 3.
type SchemaOutput struct {
	Schema   string       `json:"$schema"`
	Title    string       `json:"title"`
	Type     string       `json:"type"`
	Version  string       `json:"cli_version"`
	Commands []LeafSchema `json:"commands"`
}

// LeafSchema is the JSON Schema draft-07 representation of a single command.
// Directly compatible with the Claude tool-use tools[] element form
// (name + description + input_schema). Aligns with the Leaf object in
// manifest-schema-contract.md, sec. 3.
type LeafSchema struct {
	Schema       string      `json:"$schema,omitempty"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	InputSchema  *InputSpec  `json:"input_schema"`
	OutputSchema *OutputSpec `json:"output_schema,omitempty"`
}

// InputSpec is the draft-07 standard input schema (type=object, properties, required).
type InputSpec struct {
	Type       string                 `json:"type"`
	Properties map[string]ArgProperty `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// OutputSpec is the output schema layered on top of the CLI Output Contract JSON envelope.
type OutputSpec struct {
	Type       string                 `json:"type"`
	Properties map[string]ArgProperty `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// ---------------------------------------------------------------------------
// Flags (promoted to persistent)
// ---------------------------------------------------------------------------

var manifestFlag bool
var schemaFlag bool
var manifestGroup string
var manifestSkill string

// ---------------------------------------------------------------------------
// Skill -> command path prefix mapping
// ---------------------------------------------------------------------------

// legacySkillCommandMap is the hardcoded mapping used as fallback. It is
// consulted when the SKILL.md frontmatter `trigger_commands` is empty or the
// plugin directory cannot be located. Prevents breaking changes and keeps
// the expectations of dozens of existing tests intact. Consider removing
// once the SKILL.md convention stabilizes.
var legacySkillCommandMap = map[string][]string{
	"task-management":    {brand.ShortName + " task", brand.ShortName + " harness", brand.ShortName + " backlog"},
	"sprint-management":  {brand.ShortName + " sprint", brand.ShortName + " task assign", brand.ShortName + " task unassign"},
	"learned":            {brand.ShortName + " kb"},
	"project-resources":  {brand.ShortName + " registry"},
	"project-management": {brand.ShortName + " br", brand.ShortName + " brf", brand.ShortName + " brief", brand.ShortName + " context"},
}

// skillCommandMapOnce + cachedSkillCommandMap implement sync.Once lazy load.
var (
	skillCommandMapOnce   sync.Once
	cachedSkillCommandMap map[string][]string
)

// skillFrontmatter is the minimal schema for the SKILL.md yaml frontmatter.
type skillFrontmatter struct {
	Name            string   `yaml:"name"`
	TriggerCommands []string `yaml:"trigger_commands"`
}

// getSkillCommandMap returns the skill -> command path prefix list mapping.
// On first call, loads SKILL.md frontmatters; on failure, uses the legacy
// fallback. sync.Once limits the load to once per process lifetime.
func getSkillCommandMap() map[string][]string {
	skillCommandMapOnce.Do(func() {
		loaded, err := loadSkillCommandMap()
		if err != nil || len(loaded) == 0 {
			cachedSkillCommandMap = legacySkillCommandMap
			return
		}
		// Inject any legacy-only entries from the fallback.
		// (Prevents regressions during the progressive migration of SKILL.md frontmatter.)
		for k, v := range legacySkillCommandMap {
			if _, ok := loaded[k]; !ok {
				loaded[k] = v
			}
		}
		cachedSkillCommandMap = loaded
	})
	return cachedSkillCommandMap
}

// resetSkillCommandMapForTest resets sync.Once for test isolation.
// Production code must not call this.
func resetSkillCommandMapForTest() {
	skillCommandMapOnce = sync.Once{}
	cachedSkillCommandMap = nil
}

// findPluginSkillsDir locates the absolute path of
// packages/hostler-plugin/skills relative to ProjectRoot. Returns the empty
// string if not present (triggering the legacy fallback).
func findPluginSkillsDir() string {
	root := app.ProjectRoot()
	if root == "" {
		return ""
	}
	candidate := filepath.Join(root, "packages", "hostler-plugin", "skills")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

// loadSkillCommandMap collects trigger_commands from each SKILL.md
// frontmatter and returns a mapping skillName -> ["hstl <verb>", ...].
//
// dotted-name conversion:
//   - "task.*"          -> "hstl task" (dotted prefix -> space prefix)
//   - "sprint.complete" -> "hstl sprint complete"
//   - "task.assign"     -> "hstl task assign"
func loadSkillCommandMap() (map[string][]string, error) {
	skillsDir := findPluginSkillsDir()
	if skillsDir == "" {
		return nil, fmt.Errorf("plugin skills directory not found - using legacy fallback")
	}
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}
	result := make(map[string][]string)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "_") {
			continue // internal categories like _shared / _frozen
		}
		skillMdPath := filepath.Join(skillsDir, name, "SKILL.md")
		triggers, err := parseSkillTriggerCommands(skillMdPath)
		if err != nil || len(triggers) == 0 {
			continue
		}
		prefixes := make([]string, 0, len(triggers))
		seen := make(map[string]struct{}, len(triggers))
		for _, t := range triggers {
			p := convertDottedToPrefix(t)
			if p == "" {
				continue
			}
			if _, dup := seen[p]; dup {
				continue
			}
			seen[p] = struct{}{}
			prefixes = append(prefixes, p)
		}
		if len(prefixes) > 0 {
			sort.Strings(prefixes)
			result[name] = prefixes
		}
	}
	return result, nil
}

// parseSkillTriggerCommands reads the yaml frontmatter from a SKILL.md file
// and returns the trigger_commands array. Returns an empty slice when the
// frontmatter is missing.
func parseSkillTriggerCommands(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Extract the frontmatter: starts with `---` and runs to the second `---`.
	text := string(data)
	if !strings.HasPrefix(text, "---") {
		return nil, nil
	}
	rest := strings.TrimPrefix(text, "---")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, nil
	}
	fm := rest[:end]
	var parsed skillFrontmatter
	if err := yaml.Unmarshal([]byte(fm), &parsed); err != nil {
		return nil, err
	}
	return parsed.TriggerCommands, nil
}

// convertDottedToPrefix converts a dotted-name such as "task.*" /
// "sprint.complete" into a command path prefix such as "hstl task" /
// "hstl sprint complete". Strips the wildcard `*` and replaces dots with spaces.
func convertDottedToPrefix(dotted string) string {
	dotted = strings.TrimSpace(dotted)
	if dotted == "" {
		return ""
	}
	// Strip the trailing `.*` wildcard.
	dotted = strings.TrimSuffix(dotted, ".*")
	// Other `*` does not need special handling (we restrict to dotted prefixes).
	parts := strings.Split(dotted, ".")
	return brand.ShortName + " " + strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// Flag/argument name -> regex pattern mapping
// ---------------------------------------------------------------------------

// argPatternMap defines regex patterns for flag / argument names.
var argPatternMap = map[string]string{
	"id":            `^sprint-\d+$`,
	"sprint":        `^sprint-\d+$`,
	"sprint_id":     `^sprint-\d+$`,
	"task_id":       `^T\d+$`,
	"status":        `^(todo|in-progress|done|active|completed|backlog)$`,
	"type":          `^(feature|bugfix|hotfix|refactor|infra|docs|test|chore)$`,
	"priority":      `^p[0-3]$`,
	"estimate":      `^(XS|S|M|L|XL)$`,
	"entity_type":   `^(task|sprint)$`,
	"target_status": `^(todo|in-progress)$`,
}

// positionalArgPatternMap defines regex patterns for positional arg names.
var positionalArgPatternMap = map[string]string{
	"id":      `^(sprint-\d+|T\d+)$`,
	"task-id": `^T\d+$`,
	"item-id": `^[a-z_]+$`,
}

// ---------------------------------------------------------------------------
// Positional argument parsing
// ---------------------------------------------------------------------------

// positionalArgRe extracts <arg> patterns from cmd.Use.
var positionalArgRe = regexp.MustCompile(`<([^>]+)>`)

// parsePositionalArgs extracts positional arguments from cmd.Use.
// Example: "start <id>" -> [{Name: "id", Required: true}]
func parsePositionalArgs(cmd *cobra.Command) []PositionalArg {
	use := cmd.Use
	matches := positionalArgRe.FindAllStringSubmatch(use, -1)
	if len(matches) == 0 {
		return nil
	}

	args := make([]PositionalArg, 0, len(matches))
	for _, m := range matches {
		argName := m[1]
		arg := PositionalArg{
			Name:     argName,
			Required: cmd.Args != nil, // required when something like cobra.ExactArgs is set
		}

		// Description and pattern mappings for positional args.
		arg.Description = positionalArgDescription(cmd.CommandPath(), argName)
		arg.Pattern = positionalArgPattern(cmd.CommandPath(), argName)

		args = append(args, arg)
	}

	return args
}

// positionalArgPattern returns the regex pattern for a positional argument.
func positionalArgPattern(cmdPath string, argName string) string {
	switch {
	case strings.Contains(cmdPath, "sprint") && argName == "id":
		return `^sprint-\d+$`
	case strings.Contains(cmdPath, "task") && argName == "id":
		return `^T\d+$`
	case argName == "entity-type":
		return `^(task|sprint)$`
	case argName == "entity-id":
		return `^(sprint-\d+|T\d+)$`
	case argName == "item-id":
		return positionalArgPatternMap["item-id"]
	}
	return ""
}

// positionalArgDescription returns the description for a positional argument.
func positionalArgDescription(cmdPath string, argName string) string {
	switch {
	case strings.Contains(cmdPath, "sprint") && argName == "id":
		return "Sprint ID (e.g., sprint-19)"
	case strings.Contains(cmdPath, "task") && argName == "id":
		return "Task ID (e.g., T137)"
	case strings.Contains(cmdPath, "harness") && argName == "entity-type":
		return "Entity type (task or sprint)"
	case strings.Contains(cmdPath, "harness") && argName == "entity-id":
		return "Entity ID (e.g., T137, sprint-19)"
	case strings.Contains(cmdPath, "harness") && argName == "item-id":
		return "Harness item ID (e.g., criteria_checked)"
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// Output schema definitions
// ---------------------------------------------------------------------------

// outputSchemaMap defines the output JSON Schema for each command path.
var outputSchemaMap = map[string]*CommandArgs{
	brand.ShortName + " sprint list": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"sprints": {Type: "array", Description: "Sprint list [{sprint_id, title, status, goal, started_at, completed_at}]"},
			"count":   {Type: "integer", Description: "Number of Sprints"},
		},
	},
	brand.ShortName + " sprint create": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"sprint_id":   {Type: "string", Description: "Created Sprint ID"},
			"folder_path": {Type: "string", Description: "Sprint directory path"},
			"status":      {Type: "string", Description: "Sprint status (backlog)"},
		},
	},
	brand.ShortName + " sprint start": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"sprint_id":   {Type: "string", Description: "Sprint ID"},
			"status":      {Type: "string", Description: "Sprint status (active)"},
			"started_at":  {Type: "string", Description: "Start timestamp"},
			"folder_path": {Type: "string", Description: "Sprint directory path"},
		},
	},
	brand.ShortName + " sprint complete": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"sprint_id":    {Type: "string", Description: "Sprint ID"},
			"status":       {Type: "string", Description: "Sprint status (completed)"},
			"completed_at": {Type: "string", Description: "Completion timestamp"},
			"folder_path":  {Type: "string", Description: "Sprint directory path"},
		},
	},
	brand.ShortName + " sprint progress": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"sprint_id":   {Type: "string", Description: "Sprint ID"},
			"done":        {Type: "integer", Description: "Done Task count"},
			"in_progress": {Type: "integer", Description: "In-progress Task count"},
			"todo":        {Type: "integer", Description: "Pending Task count"},
			"total":       {Type: "integer", Description: "Total Task count"},
			"percent":     {Type: "integer", Description: "Completion ratio (0-100)"},
		},
	},
	brand.ShortName + " sprint update": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"sprint_id":      {Type: "string", Description: "Sprint ID"},
			"updated_fields": {Type: "array", Description: "Updated fields"},
		},
	},
	brand.ShortName + " task list": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"tasks": {Type: "array", Description: "Task list [{task_id, title, type, sprint, status, priority, estimate, depends_on}]"},
			"count": {Type: "integer", Description: "Number of Tasks"},
		},
	},
	brand.ShortName + " task get": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id":    {Type: "string", Description: "Task ID"},
			"title":      {Type: "string", Description: "Task title"},
			"type":       {Type: "string", Description: "Task type"},
			"sprint":     {Type: "string", Description: "Sprint ID (null means BACKLOG)"},
			"status":     {Type: "string", Description: "Task status"},
			"priority":   {Type: "string", Description: "Priority"},
			"estimate":   {Type: "string", Description: "Estimate"},
			"depends_on": {Type: "array", Description: "Dependency Task IDs"},
			"file_path":  {Type: "string", Description: "Task file path"},
		},
	},
	brand.ShortName + " task create": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id":    {Type: "string", Description: "Created Task ID"},
			"file_path":  {Type: "string", Description: "Task file path"},
			"created_at": {Type: "string", Description: "Creation timestamp"},
		},
	},
	brand.ShortName + " task start": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id":    {Type: "string", Description: "Task ID"},
			"status":     {Type: "string", Description: "Task status (in-progress)"},
			"started_at": {Type: "string", Description: "Start timestamp"},
		},
	},
	brand.ShortName + " task complete": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id":      {Type: "string", Description: "Task ID"},
			"status":       {Type: "string", Description: "Task status (done)"},
			"completed_at": {Type: "string", Description: "Completion timestamp"},
		},
	},
	brand.ShortName + " task next": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id":  {Type: "string", Description: "Suggested Task ID"},
			"title":    {Type: "string", Description: "Task title"},
			"type":     {Type: "string", Description: "Task type"},
			"sprint":   {Type: "string", Description: "Sprint ID"},
			"priority": {Type: "string", Description: "Priority"},
		},
	},
	brand.ShortName + " task delete": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id": {Type: "string", Description: "Deleted Task ID"},
			"status":  {Type: "string", Description: "Deletion status (deleted)"},
			"reason":  {Type: "string", Description: "Deletion reason"},
		},
	},
	brand.ShortName + " task reopen": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id":         {Type: "string", Description: "Task ID"},
			"previous_status": {Type: "string", Description: "Previous status"},
			"new_status":      {Type: "string", Description: "New status"},
		},
	},
	brand.ShortName + " task update": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"task_id":        {Type: "string", Description: "Task ID"},
			"updated_fields": {Type: "array", Description: "Updated field names"},
			"old_values":     {Type: "object", Description: "Old values {field: value}"},
			"new_values":     {Type: "object", Description: "New values {field: value}"},
		},
	},
	brand.ShortName + " task assign": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"assigned":  {Type: "array", Description: "Assigned Task IDs"},
			"sprint_id": {Type: "string", Description: "Assigned Sprint ID"},
			"count":     {Type: "integer", Description: "Number of assigned Tasks"},
		},
	},
	brand.ShortName + " task unassign": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"unassigned": {Type: "array", Description: "Detached Task IDs"},
			"count":      {Type: "integer", Description: "Number of detached Tasks"},
		},
	},
	brand.ShortName + " registry check": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"available":  {Type: "boolean", Description: "Whether the value is available"},
			"conflicts":  {Type: "array", Description: "Conflicting allocations [{owner, value, env}] (when available=false)"},
			"owner":      {Type: "string", Description: "Conflicting owner (when available=false)"},
			"suggestion": {Type: "integer", Description: "Suggested value when no conflict exists"},
		},
	},
	brand.ShortName + " registry sync": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"synced":     {Type: "boolean", Description: "Whether the sync was performed"},
			"diff_count": {Type: "integer", Description: "Number of changed entries"},
			"docs_path":  {Type: "string", Description: "Path of the synced docs/registry.json"},
		},
	},
}

// ---------------------------------------------------------------------------
// Ceremony schema definitions
// ---------------------------------------------------------------------------

// ceremonySchemaMap defines the JSON Schema for the ceremony object on
// ceremony-bearing commands.
var ceremonySchemaMap = map[string]*CommandArgs{
	brand.ShortName + " sprint create": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"briefing":         {Type: "object", Description: "Sprint briefing {sprint_id, title, goal, tasks[], task_count}"},
			"design_readiness": {Type: "array", Description: "Design readiness [{check, status, detail}]"},
			"reminders":        {Type: "array", Description: "Reminder strings"},
		},
	},
	brand.ShortName + " sprint start": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"briefing":         {Type: "object", Description: "Sprint briefing {sprint_id, title, goal, tasks[], task_count}"},
			"design_readiness": {Type: "array", Description: "Design readiness [{check, status, detail}]"},
			"reminders":        {Type: "array", Description: "Reminder strings"},
		},
	},
	brand.ShortName + " sprint complete": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"harness":       {Type: "array", Description: "10-Phase Harness [{phase, status, detail}]"},
			"kb_candidates": {Type: "array", Description: "KB candidate strings"},
		},
	},
	brand.ShortName + " task create": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"briefing":  {Type: "object", Description: "Task briefing {task_id, title, type, estimate, priority, sprint, depends_on, commit_prefix}"},
			"reminders": {Type: "array", Description: "Reminder strings"},
		},
	},
	brand.ShortName + " task start": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"briefing":  {Type: "object", Description: "Task briefing {task_id, title, type, estimate, priority, sprint, depends_on, commit_prefix}"},
			"reminders": {Type: "array", Description: "Reminder strings"},
		},
	},
	brand.ShortName + " task complete": {
		Type: "object",
		Properties: map[string]ArgProperty{
			"changed_files":  {Type: "array", Description: "Changed file paths"},
			"harness_gate":   {Type: "object", Description: "Harness Gate {status, unchecked_items[]}"},
			"result_section": {Type: "object", Description: "Result section verification {exists, missing_files[]}"},
		},
	},
}

// ---------------------------------------------------------------------------
// Manifest collection logic
// ---------------------------------------------------------------------------

// collectLeafCommands walks the Cobra command tree recursively and returns leaf commands only.
func collectLeafCommands(cmd *cobra.Command) []*cobra.Command {
	if !cmd.HasSubCommands() {
		// Exclude built-in commands like help / completion.
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}
		return []*cobra.Command{cmd}
	}

	var leaves []*cobra.Command
	for _, sub := range cmd.Commands() {
		leaves = append(leaves, collectLeafCommands(sub)...)
	}
	return leaves
}

// buildCommandMeta builds a CommandMeta from a Cobra command.
func buildCommandMeta(cmd *cobra.Command) CommandMeta {
	meta := CommandMeta{
		Name:        cmd.CommandPath(),
		Description: cmd.Short,
		CLIUsage:    cmd.CommandPath() + " --json",
	}

	// Parse positional arguments.
	meta.PositionalArgs = parsePositionalArgs(cmd)

	// Extract ceremony-related metadata from Annotations.
	if cmd.Annotations != nil {
		if cmd.Annotations[manifestAnnotationCeremony] == "true" {
			meta.Ceremony = true
			meta.CLIUsage = cmd.CommandPath() + " --json --with-ceremony"
		}
		if ref, ok := cmd.Annotations[manifestAnnotationCommandRef]; ok {
			meta.CommandRef = ref
		}
	}

	// Convert flags into JSON Schema properties.
	args := &CommandArgs{
		Type:       "object",
		Properties: make(map[string]ArgProperty),
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Exclude global / hidden flags.
		if f.Hidden {
			return
		}
		prop := ArgProperty{
			Description: f.Usage,
		}
		switch f.Value.Type() {
		case "bool":
			prop.Type = "boolean"
		case "int", "int64":
			prop.Type = "integer"
		default:
			prop.Type = "string"
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			prop.Default = f.DefValue
		}

		// Pattern mapping.
		if pattern, ok := argPatternMap[f.Name]; ok {
			prop.Pattern = pattern
		}

		args.Properties[f.Name] = prop
	})

	// Collect required flags.
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// cobra.MarkFlagRequired records its mark in Annotations internally.
		if f.Annotations != nil {
			if _, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
				args.Required = append(args.Required, f.Name)
			}
		}
	})

	if len(args.Properties) > 0 {
		meta.Args = args
	}

	// Output schema.
	if schema, ok := outputSchemaMap[cmd.CommandPath()]; ok {
		meta.Output = schema
	}

	// Ceremony schema.
	if meta.Ceremony {
		if schema, ok := ceremonySchemaMap[cmd.CommandPath()]; ok {
			meta.CeremonySchema = schema
		}
	}

	return meta
}

// buildLeafSchema builds a draft-07 LeafSchema from a single cobra command.
// Compatible with the Claude tool-use tools[] element shape
// (name + description + input_schema).
func buildLeafSchema(cmd *cobra.Command) LeafSchema {
	leaf := LeafSchema{
		Name:        cmd.CommandPath(),
		Description: cmd.Short,
	}

	// input_schema: positional args → required + properties, flags → properties
	input := &InputSpec{
		Type:       "object",
		Properties: make(map[string]ArgProperty),
	}

	// positional args
	positionalArgs := parsePositionalArgs(cmd)
	for _, pa := range positionalArgs {
		prop := ArgProperty{
			Type:        "string",
			Description: pa.Description,
			Pattern:     pa.Pattern,
		}
		input.Properties[pa.Name] = prop
		if pa.Required {
			input.Required = append(input.Required, pa.Name)
		}
	}

	// flags
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		prop := ArgProperty{
			Description: f.Usage,
		}
		switch f.Value.Type() {
		case "bool":
			prop.Type = "boolean"
		case "int", "int64":
			prop.Type = "integer"
		case "float32", "float64":
			prop.Type = "number"
		case "stringSlice", "stringArray":
			prop.Type = "array"
		case "duration":
			prop.Type = "string"
		default:
			prop.Type = "string"
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "[]" {
			prop.Default = f.DefValue
		}
		if pattern, ok := argPatternMap[f.Name]; ok {
			prop.Pattern = pattern
		}
		input.Properties[f.Name] = prop

		// Collect required flags (cobra.MarkFlagRequired).
		if f.Annotations != nil {
			if _, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
				input.Required = append(input.Required, f.Name)
			}
		}
	})

	leaf.InputSchema = input

	// output_schema: CLI Output Contract JSON envelope base plus each command's data schema.
	output := &OutputSpec{
		Type: "object",
		Properties: map[string]ArgProperty{
			"status":   {Type: "string", Description: "status (ok|error|blocked)"},
			"data":     {Type: "object", Description: "subcommand-specific payload"},
			"warnings": {Type: "array", Description: "warning messages"},
			"error":    {Type: "object", Description: "error object (when status=error)"},
		},
		Required: []string{"status"},
	}
	// Apply each command's outputSchemaMap to the data field.
	if dataSchema, ok := outputSchemaMap[cmd.CommandPath()]; ok {
		props := make(map[string]ArgProperty, len(dataSchema.Properties)+4)
		for k, v := range output.Properties {
			props[k] = v
		}
		props["data"] = ArgProperty{
			Type:        "object",
			Description: "subcommand-specific payload - " + joinSchemaKeys(dataSchema),
		}
		output.Properties = props
	}
	leaf.OutputSchema = output

	return leaf
}

// joinSchemaKeys returns the comma-separated list of property keys for a
// CommandArgs (used in output_schema.data descriptions).
func joinSchemaKeys(args *CommandArgs) string {
	if args == nil || len(args.Properties) == 0 {
		return "generic"
	}
	keys := make([]string, 0, len(args.Properties))
	for k := range args.Properties {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

// filterByGroup returns only commands whose first subcommand name matches group.
// Example: group="task" -> only commands shaped like brand.ShortName + " task *".
func filterByGroup(leaves []*cobra.Command, group string) []*cobra.Command {
	prefix := brand.ShortName + " " + group
	var result []*cobra.Command
	for _, leaf := range leaves {
		if strings.HasPrefix(leaf.CommandPath(), prefix) {
			result = append(result, leaf)
		}
	}
	return result
}

// filterBySkill returns only commands belonging to the specified skill.
// The mapping is consulted from SKILL.md frontmatter trigger_commands or
// the legacy fallback. Returns an error for unknown skill names.
func filterBySkill(leaves []*cobra.Command, skill string) ([]*cobra.Command, error) {
	skillMap := getSkillCommandMap()
	prefixes, ok := skillMap[skill]
	if !ok {
		names := make([]string, 0, len(skillMap))
		for k := range skillMap {
			names = append(names, k)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("unknown skill: %q\navailable skills: %s", skill, strings.Join(names, ", "))
	}

	var result []*cobra.Command
	for _, leaf := range leaves {
		path := leaf.CommandPath()
		for _, pfx := range prefixes {
			if strings.HasPrefix(path, pfx) {
				result = append(result, leaf)
				break
			}
		}
	}
	return result, nil
}

// generateManifest builds the full (or filtered) manifest.
// When group is non-empty, only that group is included; when skill is
// non-empty, only that skill is included.
func generateManifest(group, skill string) (*ManifestOutput, error) {
	leaves := collectLeafCommands(rootCmd)

	if group != "" {
		leaves = filterByGroup(leaves, group)
	}

	if skill != "" {
		var err error
		leaves, err = filterBySkill(leaves, skill)
		if err != nil {
			return nil, err
		}
	}

	manifest := &ManifestOutput{
		Schema:  manifestSchemaURI,
		Version: readPluginVersion(),
	}

	for _, leaf := range leaves {
		meta := buildCommandMeta(leaf)
		manifest.Commands = append(manifest.Commands, meta)
	}

	return manifest, nil
}

// ---------------------------------------------------------------------------
// init - register --manifest / --schema as persistent flags
// ---------------------------------------------------------------------------

func init() {
	// Promoted to PersistentFlags so every subcommand inherits.
	// Where the flag is set = the output scope (root/branch/leaf recursion).
	rootCmd.PersistentFlags().BoolVar(&manifestFlag, "manifest", false,
		"Print the manifest of the subtree rooted at this command node (persistent - inherited by every subcommand)")
	rootCmd.PersistentFlags().BoolVar(&schemaFlag, "schema", false,
		"Print the JSON Schema draft-07 for the subtree rooted at this command node (persistent; Claude tool use / MCP compatible)")
	rootCmd.PersistentFlags().StringVar(&manifestGroup, "group", "",
		"Filter manifest output by group (the first subcommand name, e.g. task, sprint)")
	rootCmd.PersistentFlags().StringVar(&manifestSkill, "skill", "",
		"Filter manifest output by skill name (e.g. task-management, sprint-management)")

	// Extend PersistentPreRun to handle --manifest / --schema.
	origPreRun := rootCmd.PersistentPreRun
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if origPreRun != nil {
			origPreRun(cmd, args)
		}

		// Scope handling based on where the flag was set - the cmd argument is the scope.
		if manifestFlag || schemaFlag {
			// Bypass the persistent handler for manifest get / validate, etc.
			// They resolve a dotted-name in their own RunE and return the schema.
			if isManifestSubcommand(cmd) && cmd != manifestCmd {
				return
			}
			handleManifestOrSchema(cmd)
		}
	}

	// Add a Run for the root command - help or manifest when invoked without a subcommand.
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if manifestFlag || schemaFlag {
			// Already handled by PersistentPreRun.
			return nil
		}
		return cmd.Help()
	}

	// installManifestSchemaSupport runs from Execute() in root.go because
	// cobra.OnInitialize runs after the c.Runnable() check, missing the
	// timing window for injecting RunE on group cmds.
}

// installManifestSchemaSupport injects --manifest/--schema support across the
// entire cobra command tree.
//
// Injection:
// 1. Wrap each cmd.Args so that, when manifestFlag/schemaFlag is set, the positional check is skipped.
// 2. Provide a default RunE on group cmds that lack one - run cmd.Help() when no flags are set.
//
// rootCmd already sets its own RunE in init, so we skip it.
func installManifestSchemaSupport(cmd *cobra.Command) {
	// 1. Wrap Args - capture the original Args via closure.
	if cmd != rootCmd {
		origArgs := cmd.Args
		cmd.Args = func(c *cobra.Command, args []string) error {
			if manifestFlag || schemaFlag {
				return nil
			}
			if origArgs != nil {
				return origArgs(c, args)
			}
			return nil
		}

		// 2. Inject a default RunE on groups missing one.
		if cmd.Run == nil && cmd.RunE == nil {
			cmd.RunE = func(c *cobra.Command, args []string) error {
				if manifestFlag || schemaFlag {
					// Already handled by PersistentPreRun.
					return nil
				}
				return c.Help()
			}
		}
	}

	// recurse
	for _, sub := range cmd.Commands() {
		installManifestSchemaSupport(sub)
	}
}

// isManifestSubcommand reports whether the given cmd belongs to the manifest
// subcommand family (manifest.get, manifest.validate, etc.). Such commands
// handle scope in their own RunE, so the persistent handler must not
// intercept them.
func isManifestSubcommand(cmd *cobra.Command) bool {
	if cmd == manifestCmd {
		return true
	}
	for parent := cmd.Parent(); parent != nil; parent = parent.Parent() {
		if parent == manifestCmd {
			return true
		}
	}
	return false
}

// handleManifestOrSchema processes --manifest / --schema based on scope.
// The cmd argument is the "where the flag was set" anchor; only leaves under
// that node are collected (recursion principle).
func handleManifestOrSchema(cmd *cobra.Command) {
	// Scope: cmd=root -> entire tree; cmd=leaf -> single command; cmd=branch -> subtree.
	scopeLeaves := collectLeafCommands(cmd)

	// group / skill filters apply only to root/branch calls (a leaf call already returns 1).
	if manifestGroup != "" && len(scopeLeaves) > 1 {
		scopeLeaves = filterByGroup(scopeLeaves, manifestGroup)
	}
	if manifestSkill != "" && len(scopeLeaves) > 1 {
		filtered, err := filterBySkill(scopeLeaves, manifestSkill)
		if err != nil {
			Out.Error("manifest filter error: "+err.Error(), apperr.CategoryInvalidInput.String(), "")
			os.Exit(exitError)
		}
		scopeLeaves = filtered
	}

	if schemaFlag {
		// JSON Schema draft-07 standard output.
		emitSchema(cmd, scopeLeaves)
	} else {
		// Existing --manifest: emit the custom ManifestOutput structure.
		emitManifest(scopeLeaves)
	}
	os.Exit(exitSuccess)
}

// emitManifest builds and prints a ManifestOutput from the given leaves.
func emitManifest(leaves []*cobra.Command) {
	manifest := &ManifestOutput{
		Schema:  manifestSchemaURI,
		Version: readPluginVersion(),
	}
	for _, leaf := range leaves {
		manifest.Commands = append(manifest.Commands, buildCommandMeta(leaf))
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		Out.Error("failed to emit manifest: "+err.Error(), apperr.CategoryInternal.String(), "")
		os.Exit(exitError)
	}
}

// emitSchema emits the draft-07 standard JSON Schema.
// For a single leaf, a single object (without the commands array) is emitted
// for Claude tool-use compatibility. For root/branch calls, includes the
// commands array plus title/type/cli_version metadata.
func emitSchema(cmd *cobra.Command, leaves []*cobra.Command) {
	// Single leaf - Claude tool-use tools[] element form.
	if len(leaves) == 1 && leaves[0] == cmd {
		leafSchema := buildLeafSchema(leaves[0])
		leafSchema.Schema = manifestSchemaURI // leaf-alone call: emit $schema (draft-07 valid)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(leafSchema); err != nil {
			Out.Error("failed to emit schema: "+err.Error(), apperr.CategoryInternal.String(), "")
			os.Exit(exitError)
		}
		return
	}

	// Branch/root - group schema with the commands array.
	schemaOut := SchemaOutput{
		Schema:   manifestSchemaURI,
		Title:    brand.ShortName + " " + cmd.CommandPath(),
		Type:     "object",
		Version:  readPluginVersion(),
		Commands: make([]LeafSchema, 0, len(leaves)),
	}
	for _, leaf := range leaves {
		schemaOut.Commands = append(schemaOut.Commands, buildLeafSchema(leaf))
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(schemaOut); err != nil {
		Out.Error("failed to emit schema: "+err.Error(), apperr.CategoryInternal.String(), "")
		os.Exit(exitError)
	}
}

// ---------------------------------------------------------------------------
// Register ceremony Annotations on sprint/task commands
// ---------------------------------------------------------------------------

// RegisterCeremonyAnnotations adds Annotations to the six ceremony-bearing commands.
// Called from init().
func RegisterCeremonyAnnotations() {
	annotations := map[*cobra.Command]map[string]string{
		sprintCreateCmd: {
			manifestAnnotationCeremony:   "true",
			manifestAnnotationCommandRef: "hstl:sprint:create",
		},
		sprintStartCmd: {
			manifestAnnotationCeremony:   "true",
			manifestAnnotationCommandRef: "hstl:sprint:start",
		},
		sprintCompleteCmd: {
			manifestAnnotationCeremony:   "true",
			manifestAnnotationCommandRef: "hstl:sprint:complete",
		},
		taskCreateCmd: {
			manifestAnnotationCeremony:   "true",
			manifestAnnotationCommandRef: "hstl:task:create",
		},
		taskStartCmd: {
			manifestAnnotationCeremony:   "true",
			manifestAnnotationCommandRef: "hstl:task:start",
		},
		taskCompleteCmd: {
			manifestAnnotationCeremony:   "true",
			manifestAnnotationCommandRef: "hstl:task:complete",
		},
	}

	for cmd, annots := range annotations {
		if cmd.Annotations == nil {
			cmd.Annotations = make(map[string]string)
		}
		for k, v := range annots {
			cmd.Annotations[k] = v
		}
	}
}
