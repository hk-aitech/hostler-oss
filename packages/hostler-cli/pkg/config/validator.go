// Package config — runtime schema validation.
//
// Returns errors early when a YAML file's structure does not match the Go
// types in types.go (wrong enum, missing fields, type mismatch, etc.). Uses
// the jsonschema-go Draft 2020-12 validator and infers the schema directly
// from the ProjectConfig type via SchemaForProjectConfig, so it works
// in-process without loading any file.
//
// Calling convention:
//
//	cfg, warnings, err := LoadProjectConfigValidated()
//
// - err != nil: fatal error such as IO/parse/schema-inference failure (cfg may be nil).
// - warnings != nil: validation findings (cfg loaded successfully; defer to caller).
package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"gopkg.in/yaml.v3"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/events"
)

// resolvedSchemaOnce / resolvedSchema is the lazy cache for the
// ProjectConfig schema. jsonschema.For + Resolve incur reflect cost, so we
// run them only on the first call.
var (
	resolvedSchemaOnce sync.Once
	resolvedSchema     *jsonschema.Resolved
	resolvedSchemaErr  error
)

// SchemaForProjectConfig returns the JSON Schema for the ProjectConfig
// type. Shared between cmd/gen-schema and the validator. Each call
// constructs a fresh instance so callers cannot influence one another.
//
// Generation pipeline:
//  1. jsonschema.For[ProjectConfig] — initial reflect-based schema.
//  2. enrichReminderPropertyNames — inject events.KnownEvents enum into the reminders key.
//  3. enrichBriefingExtractPattern — inject the DSL pattern into briefing.sections[].extract.
//  4. enrichPlatformKindEnum — inject the go/dotnet/python/node enum into platform.kind.
//
// Future extensions: progressively inject constraints that reflect alone
// cannot express.
func SchemaForProjectConfig() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[ProjectConfig](nil)
	if err != nil {
		return nil, fmt.Errorf("infer ProjectConfig schema: %w", err)
	}
	enrichReminderPropertyNames(schema)
	enrichBriefingExtractPattern(schema)
	enrichPlatformKindEnum(schema)
	return schema, nil
}

// enrichPlatformKindEnum injects the go/dotnet/python/node enum constraint
// into platform.kind (part of the v2 schema).
//
// Reflect-based inference produces only "type: string" for string fields,
// so the Platform kind enum constraint must be declared separately. With
// it, JSON Schema alone catches invalid kind values (e.g. "rust") in
// IDE/CI early. Runtime validation duplicates the check in
// validatePlatformTaggedUnion (defence in depth).
func enrichPlatformKindEnum(schema *jsonschema.Schema) {
	if schema == nil || schema.Properties == nil {
		return
	}
	platform, ok := schema.Properties["platform"]
	if !ok || platform == nil || platform.Properties == nil {
		return
	}
	kind, ok := platform.Properties["kind"]
	if !ok || kind == nil {
		return
	}
	kind.Enum = []any{
		PlatformKindGo,
		PlatformKindDotnet,
		PlatformKindPython,
		PlatformKindNode,
	}
}

// enrichReminderPropertyNames injects the events.KnownEvents enum into the
// reminders property keys. JSON Schema's propertyNames.enum is the
// standard mechanism for restricting object keys to a fixed set.
//
// google/jsonschema-go's reflect-based For[T] converts map[string]T into
// additionalProperties without producing a key enum, so this helper closes
// the gap.
func enrichReminderPropertyNames(schema *jsonschema.Schema) {
	if schema == nil || schema.Properties == nil {
		return
	}
	reminders, ok := schema.Properties["reminders"]
	if !ok || reminders == nil {
		return
	}
	keyEnum := make([]any, 0, len(events.KnownEventStrings()))
	for _, e := range events.KnownEventStrings() {
		keyEnum = append(keyEnum, e)
	}
	reminders.PropertyNames = &jsonschema.Schema{
		Enum: keyEnum,
	}
}

// enrichBriefingExtractPattern injects the DSL pattern constraint into
// briefing.sections[].extract.
//
// Reflect-based schema inference produces only `"type": "string"` for
// string fields, so DSL enum/pattern constraints must be injected
// explicitly. The briefing layout is BriefingConfig -> Sections
// ([]BriefingSection); descend through items.properties.extract to set
// the pattern.
func enrichBriefingExtractPattern(schema *jsonschema.Schema) {
	if schema == nil || schema.Properties == nil {
		return
	}
	briefing, ok := schema.Properties["briefing"]
	if !ok || briefing == nil || briefing.Properties == nil {
		return
	}
	sections, ok := briefing.Properties["sections"]
	if !ok || sections == nil || sections.Items == nil || sections.Items.Properties == nil {
		return
	}
	extract, ok := sections.Items.Properties["extract"]
	if !ok || extract == nil {
		return
	}
	extract.Pattern = ExtractPattern()
}

// loadResolvedSchema infers and resolves the validation schema once and caches it.
func loadResolvedSchema() (*jsonschema.Resolved, error) {
	resolvedSchemaOnce.Do(func() {
		schema, err := SchemaForProjectConfig()
		if err != nil {
			resolvedSchemaErr = err
			return
		}
		resolved, err := schema.Resolve(nil)
		if err != nil {
			resolvedSchemaErr = fmt.Errorf("resolve ProjectConfig schema: %w", err)
			return
		}
		resolvedSchema = resolved
	})
	return resolvedSchema, resolvedSchemaErr
}

// ValidateProjectConfigYAML parses YAML bytes and validates them against the schema.
//
// Returns:
//   - parseErr: YAML syntax error (cannot validate; fatal).
//   - validationErr: schema violation (cfg is still returned — equivalent to warn mode).
//
// Validation findings are wrapped in ValidationError with a recovery hint.
//
// When extends is present, the presets are loaded and deep-merged before validation.
func ValidateProjectConfigYAML(data []byte) (*ProjectConfig, *ValidationError, error) {
	// 1. resolve extends and merge presets
	raw, mergeErr := applyExtendsAndMerge(data)
	if mergeErr != nil {
		return nil, nil, fmt.Errorf("resolve extends: %w", mergeErr)
	}

	// 2. merged map -> YAML -> typed ProjectConfig
	finalYAML, err := yaml.Marshal(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("re-marshal merged result: %w", err)
	}
	var cfg ProjectConfig
	if err := yaml.Unmarshal(finalYAML, &cfg); err != nil {
		return nil, nil, fmt.Errorf("unmarshal YAML to ProjectConfig: %w", err)
	}

	// 3. skip validation for an empty file (validating a nil map is meaningless)
	if raw == nil {
		return &cfg, nil, nil
	}

	// 4. load schema and validate
	resolved, err := loadResolvedSchema()
	if err != nil {
		return &cfg, nil, err
	}

	// 5. convert YAML to a JSON-compatible view
	// yaml.v3 may produce map[any]any, so normalise to a JSON-friendly shape.
	normalized := normalizeYAMLValue(raw)

	// Multi-error collection. Run every validation step rather than
	// returning early; if there is more than one finding, attach them as
	// Causes to the top-level ValidationError. Backward-compatible
	// `verr != nil` and `verr.Error()` formatting still work.
	var causes []ValidationError

	if err := resolved.Validate(normalized); err != nil {
		causes = append(causes, ValidationError{
			Path:         "",
			Message:      err.Error(),
			RecoveryHint: "Update project-config.yaml to match the schema in types.go, or check the $schema version.",
		})
	}

	// Migration Runner removed. v2 only.

	// version major fail-fast check.
	// Apply the same rule as the loader path (loadProjectConfigOnce) to
	// the validator path (validate-config CLI / MCP config_inspect) so a
	// manually edited version: "3.0.0" file is not reported as valid only
	// to fail-fast at runtime.
	if vErr := checkVersionMajor(&cfg); vErr != nil {
		causes = append(causes, ValidationError{
			Path:         "version",
			Message:      vErr.Error(),
			RecoveryHint: "Run hstl config migrate --apply to migrate the file to the latest format.",
		})
	}

	// Platform Tagged Union semantic validation.
	// JSON Schema oneOf is awkward to express the Kind <-> nested struct
	// integrity, so it is checked separately. Skip when Platform is unset.
	if verr := validatePlatformTaggedUnion(&cfg); verr != nil {
		causes = append(causes, *verr)
	}

	if len(causes) == 0 {
		return &cfg, nil, nil
	}
	if len(causes) == 1 {
		// Single error: return it as before (Causes stays nil).
		single := causes[0]
		return &cfg, &single, nil
	}
	// Multiple errors: use the first as the top-level message and store all in Causes.
	return &cfg, &ValidationError{
		Path:         "",
		Message:      fmt.Sprintf("%d configuration violations found (see Causes for details)", len(causes)),
		RecoveryHint: "Inspect each Cause's RecoveryHint and update project-config.yaml accordingly.",
		Causes:       causes,
	}, nil
}

// validatePlatformTaggedUnion validates Kind <-> nested struct integrity for
// the Platform Tagged Union.
//
// Rules:
//  1. Platform unset (nil) -> skip validation (backward-compat mode).
//  2. Platform set:
//     - Kind must be one of {go, dotnet, python, node}.
//     - Only the nested struct corresponding to Kind may be populated.
//     - All other Kind nested structs must be nil.
//
// Example errors:
//   - Kind="go" + Dotnet populated -> "platform.kind=go but platform.dotnet is set".
//   - Kind="invalid" -> "platform.kind must be one of go/dotnet/python/node".
func validatePlatformTaggedUnion(cfg *ProjectConfig) *ValidationError {
	if cfg == nil || cfg.Platform == nil {
		return nil
	}
	p := cfg.Platform

	// Validate Kind
	validKinds := map[string]bool{
		PlatformKindGo:     true,
		PlatformKindDotnet: true,
		PlatformKindPython: true,
		PlatformKindNode:   true,
	}
	if !validKinds[p.Kind] {
		return &ValidationError{
			Path:    "platform.kind",
			Message: fmt.Sprintf("platform.kind must be one of go/dotnet/python/node (got: %q)", p.Kind),
			RecoveryHint: "Set project-config.yaml's platform.kind to match the project type. " +
				"Go -> 'go', .NET -> 'dotnet', Python -> 'python', Node.js/TypeScript -> 'node'.",
		}
	}

	// Cross-check Kind against the populated nested struct.
	mismatches := []string{}
	if p.Kind != PlatformKindGo && p.Go != nil {
		mismatches = append(mismatches, "platform.go")
	}
	if p.Kind != PlatformKindDotnet && p.Dotnet != nil {
		mismatches = append(mismatches, "platform.dotnet")
	}
	if p.Kind != PlatformKindPython && p.Python != nil {
		mismatches = append(mismatches, "platform.python")
	}
	if p.Kind != PlatformKindNode && p.Node != nil {
		mismatches = append(mismatches, "platform.node")
	}
	if len(mismatches) > 0 {
		return &ValidationError{
			Path: "platform",
			Message: fmt.Sprintf("platform.kind=%q but nested fields for other kinds are set: %v",
				p.Kind, mismatches),
			RecoveryHint: fmt.Sprintf("Keep only platform.%s to match platform.kind=%q and remove %v.",
				p.Kind, p.Kind, mismatches),
		}
	}

	return nil
}

// ValidationError structures information about a schema violation. Maps to
// MCP tool responses with error_category=INVALID_STATE and recovery_hint.
//
// Causes is added for multi-error collection. The top-level
// ValidationError carries the summary message; each individual violation
// goes into Causes. Existing call sites stay backward compatible if they
// only use Error() / Message / RecoveryHint.
type ValidationError struct {
	Path         string            // JSON Pointer (empty string means root)
	Message      string            // violation message returned by jsonschema-go
	RecoveryHint string            // user-facing recovery guidance
	Causes       []ValidationError // multi-error collection (nil for single error)
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if len(e.Causes) == 0 {
		if e.Path == "" {
			return e.Message
		}
		return fmt.Sprintf("%s at %s", e.Message, e.Path)
	}
	// Multi-error: summary plus one line per cause.
	lines := make([]string, 0, len(e.Causes)+1)
	lines = append(lines, e.Message)
	for i, c := range e.Causes {
		cMsg := c.Message
		if c.Path != "" {
			cMsg = fmt.Sprintf("%s at %s", c.Message, c.Path)
		}
		lines = append(lines, fmt.Sprintf("  %d. %s", i+1, cMsg))
	}
	return strings.Join(lines, "\n")
}

// LoadProjectConfigValidated adds schema validation to the LoadProjectConfig path.
// Missing file returns (nil, nil, nil); parse errors go to err and schema
// violations to ValidationError.
//
// LoadProjectConfig itself does not consume the validation result —
// kept separate for backward compatibility. Callers opt in to validation
// only when they need it.
func LoadProjectConfigValidated() (*ProjectConfig, *ValidationError, error) {
	path := findProjectConfigYAML()
	if path == "" {
		return nil, nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read YAML (%s): %w", path, err)
	}
	return ValidateProjectConfigYAML(data)
}

// normalizeYAMLValue normalises yaml.v3 values into a JSON-compatible shape.
//
// yaml.v3 maps may use keys typed as any (interface{}), but the JSON Schema
// validator expects map[string]any. Converts recursively.
func normalizeYAMLValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = normalizeYAMLValue(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[fmt.Sprint(k)] = normalizeYAMLValue(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = normalizeYAMLValue(item)
		}
		return out
	default:
		return v
	}
}

// ResetValidatorCache resets the resolved-schema cache for test isolation.
func ResetValidatorCache() {
	resolvedSchemaOnce = sync.Once{}
	resolvedSchema = nil
	resolvedSchemaErr = nil
}
