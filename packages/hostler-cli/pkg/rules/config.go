package rules

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// presetFS holds the built-in preset bundle.
//
//go:embed presets/*.yaml presets/manifest.json
var presetFS embed.FS

const (
	// configFileName is the project-relative path of the config file.
	configFileName = ".hstl/rules.yaml"

	// presetScheme is the preset://hostler/{name} extends URI scheme.
	presetScheme = "preset://hostler/"

	// envVarPrefix is the environment-variable override prefix.
	// HSTL_RULE_<ID>=<severity>.
	envVarPrefix = "HSTL_RULE_"

	// defaultPreset is the fallback preset when rules.yaml is absent.
	defaultPreset = "recommended"

	// maxExtendsDepth is the maximum depth of the extends chain (second
	// line of defence against cycles).
	maxExtendsDepth = 16
)

// EngineConfig is the schema of rules.yaml.
type EngineConfig struct {
	Version  int               `yaml:"version"`
	Extends  []string          `yaml:"extends"`
	Policies []PolicyConfig    `yaml:"policies"`
	Rules    map[string]string `yaml:"rules"` // id → severity | off
	Targets  []TargetConfig    `yaml:"targets"`
}

// PolicyConfig is a named bundle of Rules.
type PolicyConfig struct {
	Name              string              `yaml:"name"`
	EnforcementLevel  Severity            `yaml:"enforcement_level"`
	Includes          []string            `yaml:"includes"`
	SeverityOverrides map[string]Severity `yaml:"severity_overrides"`
}

// TargetConfig is the list of Policies applied to a glob path.
type TargetConfig struct {
	Paths    []string `yaml:"paths"`
	Policies []string `yaml:"policies"`
}

// EffectiveRule is one entry of the cascade-resolved result.
type EffectiveRule struct {
	RuleID        string       // rule ID (e.g. task.result_section.files_exist)
	Severity      Severity     // final severity
	Origin        Origin       // origin (final decision layer)
	PreviousChain []ChainEntry // decision history per layer (rules effective rationale)
}

// Origin identifies the layer that made the final decision.
type Origin struct {
	Layer  string // preset | policy | rules | env
	Source string // concrete name (preset name, policy name, "rules.yaml", env var key)
}

// ChainEntry is a single element of PreviousChain.
type ChainEntry struct {
	Layer    string
	Source   string
	Severity Severity
}

// LoadConfig loads rules.yaml from the project root. When the file is
// absent, it returns the built-in recommended preset as an EngineConfig
// (extends: [preset://hostler/recommended]).
func LoadConfig(projectRoot string) (*EngineConfig, error) {
	path := filepath.Join(projectRoot, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &EngineConfig{
				Version: 1,
				Extends: []string{presetScheme + defaultPreset},
			}, nil
		}
		return nil, fmt.Errorf("read rules.yaml: %w", err)
	}
	var cfg EngineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse rules.yaml (%s): %w", path, err)
	}
	return &cfg, nil
}

// PresetManifest is the schema of presets/manifest.json.
type PresetManifest struct {
	Revision    string   `json:"revision"`
	BuiltAt     string   `json:"built_at"`
	PresetCount int      `json:"preset_count"`
	RuleCount   int      `json:"rule_count"`
	Presets     []string `json:"presets"`
	Source      string   `json:"source"`
	Description string   `json:"description"`
}

// LoadPresetManifest returns the manifest of the built-in preset bundle.
func LoadPresetManifest() (*PresetManifest, error) {
	data, err := presetFS.ReadFile("presets/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("load manifest.json: %w", err)
	}
	var m PresetManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest.json: %w", err)
	}
	return &m, nil
}

// LoadConfigBytes parses an EngineConfig from yaml bytes (used for tests and
// extends resolution).
func LoadConfigBytes(data []byte) (*EngineConfig, error) {
	var cfg EngineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml parse: %w", err)
	}
	return &cfg, nil
}

// resolveExtend loads a single extends URI: preset:// / file:// / relative
// path / https://.
func resolveExtend(uri string, baseDir string) (*EngineConfig, error) {
	switch {
	case strings.HasPrefix(uri, presetScheme):
		name := strings.TrimPrefix(uri, presetScheme)
		data, err := presetFS.ReadFile("presets/" + name + ".yaml")
		if err != nil {
			return nil, fmt.Errorf("load built-in preset %q: %w", uri, err)
		}
		return LoadConfigBytes(data)
	case strings.HasPrefix(uri, "file://"):
		path := strings.TrimPrefix(uri, "file://")
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("load file:// extend %q: %w", uri, err)
		}
		return LoadConfigBytes(data)
	case strings.HasPrefix(uri, "https://") || strings.HasPrefix(uri, "http://"):
		return nil, fmt.Errorf("remote extends not supported: %s", uri)
	default:
		// relative path (e.g. "./team-rules.yaml")
		path := uri
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, uri)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("load extends path %q: %w", uri, err)
		}
		return LoadConfigBytes(data)
	}
}

// ResolveEffective computes the final severity of every Rule ID by applying
// the cascade priority.
//
// Priority (low → high):
//  1. rules from the extends chain (recursively flattened)
//  2. Policies.SeverityOverrides (current config)
//  3. Rules top-level override (current config)
//  4. environment variable HSTL_RULE_<ID>=<severity>
//
// Intermediate values from each layer are recorded in
// EffectiveRule.PreviousChain to support explain/effective output.
func ResolveEffective(cfg *EngineConfig, baseDir string) (map[string]*EffectiveRule, error) {
	result := make(map[string]*EffectiveRule)
	visited := make(map[string]bool)
	if err := applyCascade(cfg, baseDir, result, visited, 0); err != nil {
		return nil, err
	}
	// Layer 4: env var override
	applyEnvOverrides(result)
	return result, nil
}

// applyCascade recursively flattens the extends chain and overlays the
// current config's rules and policies on top.
func applyCascade(cfg *EngineConfig, baseDir string, result map[string]*EffectiveRule, visited map[string]bool, depth int) error {
	if depth > maxExtendsDepth {
		return fmt.Errorf("extends depth exceeded (> %d) — possible cycle", maxExtendsDepth)
	}
	// 1. extends first (lowest precedence)
	for _, uri := range cfg.Extends {
		if visited[uri] {
			return fmt.Errorf("cyclic extends detected: %s (already visited)", uri)
		}
		visited[uri] = true
		child, err := resolveExtend(uri, baseDir)
		if err != nil {
			return err
		}
		if err := applyCascade(child, baseDir, result, visited, depth+1); err != nil {
			return err
		}
		// Layer label: preset:// → preset, otherwise file extends.
		layer := "preset"
		if !strings.HasPrefix(uri, presetScheme) {
			layer = "extends"
		}
		// child's rules are already reflected in result by the recursive
		// applyCascade call; re-tag the child's top-level rules as the
		// preset layer here so the parent sees them attributed to the
		// child source.
		if err := tagLastAs(result, layer, uri, child.Rules); err != nil {
			return err
		}
		delete(visited, uri)
	}
	// 2. current config's policies.severity_overrides
	for _, p := range cfg.Policies {
		for id, sev := range p.SeverityOverrides {
			setEffective(result, id, sev, "policy", p.Name)
		}
	}
	// 3. current config's top-level rules override
	for id, raw := range cfg.Rules {
		sev, err := ParseSeverity(raw)
		if err != nil {
			return fmt.Errorf("rules[%s] severity parse: %w", id, err)
		}
		setEffective(result, id, sev, "rules", "rules.yaml")
	}
	return nil
}

// tagLastAs re-tags child rules entries as the preset layer immediately
// after extends resolution. The child's applyCascade records its top-level
// rules under the "rules" layer; from the parent's perspective these are
// preset-layer contributions and are reclassified here.
func tagLastAs(result map[string]*EffectiveRule, layer, source string, childRules map[string]string) error {
	for id := range childRules {
		eff, ok := result[id]
		if !ok {
			continue
		}
		// Re-tag only when the last entry was added by the child as the
		// "rules" layer.
		if n := len(eff.PreviousChain); n > 0 {
			last := &eff.PreviousChain[n-1]
			if last.Layer == "rules" && last.Source == "rules.yaml" {
				last.Layer = layer
				last.Source = source
			}
		}
		if eff.Origin.Layer == "rules" && eff.Origin.Source == "rules.yaml" {
			eff.Origin.Layer = layer
			eff.Origin.Source = source
		}
	}
	return nil
}

// setEffective applies a new severity to a Rule ID and records it in
// PreviousChain.
func setEffective(result map[string]*EffectiveRule, id string, sev Severity, layer, source string) {
	eff, ok := result[id]
	if !ok {
		eff = &EffectiveRule{RuleID: id}
		result[id] = eff
	}
	if eff.Origin.Layer != "" {
		eff.PreviousChain = append(eff.PreviousChain, ChainEntry{
			Layer:    eff.Origin.Layer,
			Source:   eff.Origin.Source,
			Severity: eff.Severity,
		})
	}
	eff.Severity = sev
	eff.Origin = Origin{Layer: layer, Source: source}
}

// applyEnvOverrides applies HSTL_RULE_<ID>=<severity> as the final
// override. The "." in a Rule ID is replaced with "_" to form the
// environment-variable name.
func applyEnvOverrides(result map[string]*EffectiveRule) {
	for id, eff := range result {
		idSuffix := strings.ToUpper(strings.ReplaceAll(id, ".", "_"))
		envKey := envVarPrefix + idSuffix
		raw := os.Getenv(envKey)
		if raw == "" {
			continue
		}
		sev, err := ParseSeverity(raw)
		if err != nil {
			continue // ignore invalid values to avoid silently disabling checks
		}
		// Hard-block does not allow env override.
		if eff.Severity == SeverityHardBlock {
			continue
		}
		eff.PreviousChain = append(eff.PreviousChain, ChainEntry{
			Layer: eff.Origin.Layer, Source: eff.Origin.Source, Severity: eff.Severity,
		})
		eff.Severity = sev
		eff.Origin = Origin{Layer: "env", Source: envKey}
	}
}

// PoliciesForPath returns the names of the Policies that apply to the given
// path by glob-matching against TargetConfig.Paths.
func (cfg *EngineConfig) PoliciesForPath(path string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, t := range cfg.Targets {
		for _, pattern := range t.Paths {
			matched, err := filepath.Match(pattern, path)
			if err != nil || !matched {
				continue
			}
			for _, p := range t.Policies {
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
			break
		}
	}
	sort.Strings(out)
	return out
}
