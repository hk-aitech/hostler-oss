// Package task — static-heuristic layer for summary validation.
// A pluggable heuristic chain that runs before the existing LLM HTTP
// validator. Each rule is registered by name and presets
// (strict / moderate / lenient / off) select the combination. Magic
// numbers come from config rather than being hard-coded as Go constants
// (per the `no-magic-numbers` policy).
// Flow:
//	HeuristicCheck(summary, cfg) -> (issues, ok)
//	Apply each rule in order; any failure appends to issues. Empty
//	issues -> ok.
// Integration: invoked from `task create` in cmd/hstl-oss/cmd/task.go right
// after the length check.
package task

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/domain"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"gopkg.in/yaml.v3"
)

//go:embed presets/summary-validation/*.yaml
var summaryPresetsFS embed.FS

// HeuristicPreset — alias of domain.HeuristicPreset.
type HeuristicPreset = domain.HeuristicPreset

// Preset constants — re-export the domain HeuristicPreset values for
// backward compatibility.
const (
	PresetOff      = domain.HeuristicPresetOff
	PresetLenient  = domain.HeuristicPresetLenient
	PresetModerate = domain.HeuristicPresetModerate
	PresetStrict   = domain.HeuristicPresetStrict
)

// HeuristicConfig — alias of domain.HeuristicConfig.
type HeuristicConfig = domain.HeuristicConfig

// presetDefaults maps preset names to default settings, used as a
// fallback when no YAML preset is found. Magic numbers live in one place
// and may be overridden by YAML.
var presetDefaults = map[HeuristicPreset]HeuristicConfig{
	PresetStrict: {
		Preset:         PresetStrict,
		MinUniqueWords: 5,
		ForbiddenSoloKeywords: []string{
			"fix", "update", "refactor", "cleanup", "test", "tidy",
			"work", "add", "improve", "implement",
		},
		RequiredPurposeConnector: []string{
			"because", "to fix", "so that", "since", "due to",
			"in order to", "to address", "to ensure", "to prevent",
			"to verify", "to improve",
		},
		EnableLLMJudge: true,
	},
	PresetModerate: {
		Preset:                   PresetModerate,
		MinUniqueWords:           4,
		ForbiddenSoloKeywords:    []string{"fix", "update", "tweak"},
		RequiredPurposeConnector: nil, // connectors are optional in moderate
		EnableLLMJudge:           false,
	},
	PresetLenient: {
		Preset:                   PresetLenient,
		MinUniqueWords:           0,
		ForbiddenSoloKeywords:    []string{"fix", "update"}, // minimum filter
		RequiredPurposeConnector: nil,
		EnableLLMJudge:           false,
	},
	PresetOff: {
		Preset: PresetOff,
	},
}

// ResolvePreset returns the config for the given preset name.
// Lookup priority: project override
// (.hstl-oss/summary-validation/<name>.yaml; legacy .hstl/, .hostler/ also accepted)
//	> embed YAML (presets/summary-validation/<name>.yaml)
//	> Go-map fallback (presetDefaults).
// Unknown names fall back to strict.
// trac: HAR-CM008
func ResolvePreset(name string) HeuristicConfig {
	p := HeuristicPreset(strings.ToLower(strings.TrimSpace(name)))
	if p == "" {
		p = PresetStrict
	}
	// 1) Project-local override.
	if cfg, ok := loadProjectPreset(string(p)); ok {
		return cfg
	}
	// 2) Embed YAML.
	if cfg, ok := loadEmbedPreset(string(p)); ok {
		return cfg
	}
	// 3) Go-map fallback.
	if cfg, ok := presetDefaults[p]; ok {
		return cfg
	}
	return presetDefaults[PresetStrict]
}

// loadEmbedPreset reads an embedded YAML preset and converts it to a
// config.
func loadEmbedPreset(name string) (HeuristicConfig, bool) {
	path := fmt.Sprintf("presets/summary-validation/%s.yaml", name)
	data, err := summaryPresetsFS.ReadFile(path)
	if err != nil {
		return HeuristicConfig{}, false
	}
	return parsePresetYAML(data)
}

// loadProjectPreset attempts the project-root override
// `.hstl-oss/summary-validation/<name>.yaml` (user override; legacy
// `.hstl/`, `.hostler/` directories are also consulted as a read-only
// fallback). Returns false when not present.
func loadProjectPreset(name string) (HeuristicConfig, bool) {
	root := envalias.Lookup("PROJECT_ROOT")
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return HeuristicConfig{}, false
		}
		root = wd
	}
	dirs := append([]string{brand.ProjectDirName}, brand.LegacyProjectDirNames...)
	for _, dir := range dirs {
		path := filepath.Join(root, dir, "summary-validation", name+".yaml")
		data, err := os.ReadFile(path)
		if err == nil {
			return parsePresetYAML(data)
		}
	}
	return HeuristicConfig{}, false
}

// parsePresetYAML parses the YAML bytes into a HeuristicConfig.
func parsePresetYAML(data []byte) (HeuristicConfig, bool) {
	var cfg HeuristicConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return HeuristicConfig{}, false
	}
	// Normalise the preset field.
	cfg.Preset = HeuristicPreset(strings.ToLower(strings.TrimSpace(string(cfg.Preset))))
	if cfg.Preset == "" {
		cfg.Preset = PresetStrict
	}
	return cfg, true
}

// HeuristicCheck applies the rule chain in cfg to summary.
// Returns issues (one message per violation) and ok (true when issues is
// empty). preset=off always returns ok.
// trac: HAR-CM008
func HeuristicCheck(summary string, cfg HeuristicConfig) (issues []string, ok bool) {
	if cfg.Preset == PresetOff {
		return nil, true
	}

	trimmed := strings.TrimSpace(summary)

	// min_unique_words
	if cfg.MinUniqueWords > 0 {
		if n := countUniqueWords(trimmed); n < cfg.MinUniqueWords {
			issues = append(issues, fmt.Sprintf(
				"unique words %d < minimum %d — summary is too fragmented (preset=%s)",
				n, cfg.MinUniqueWords, cfg.Preset))
		}
	}

	// forbidden_solo_keywords
	if matched := matchForbiddenSolo(trimmed, cfg.ForbiddenSoloKeywords); matched != "" {
		issues = append(issues, fmt.Sprintf(
			"the summary cannot consist of the lone keyword %q — state what / why / success criterion (preset=%s)",
			matched, cfg.Preset))
	}

	// required_purpose_connectors
	if len(cfg.RequiredPurposeConnector) > 0 && !containsAny(trimmed, cfg.RequiredPurposeConnector) {
		issues = append(issues, fmt.Sprintf(
			"no connector that ties purpose / reason / success criteria together (e.g. because / to fix / so that / in order to). preset=%s",
			cfg.Preset))
	}

	return issues, len(issues) == 0
}

// countUniqueWords counts unique lowercase tokens after splitting on
// whitespace and punctuation.
func countUniqueWords(s string) int {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	set := make(map[string]struct{}, len(fields))
	for _, w := range fields {
		if len(w) == 0 {
			continue
		}
		set[strings.ToLower(w)] = struct{}{}
	}
	return len(set)
}

// matchForbiddenSolo decides whether the summary is composed of a
// forbidden keyword alone or near-alone. Rather than strict equality, it
// matches "summary contains the keyword and has <=2 other meaningful
// words".
func matchForbiddenSolo(s string, forbidden []string) string {
	lower := strings.ToLower(s)
	meaningfulWords := countUniqueWords(s)
	for _, kw := range forbidden {
		klow := strings.ToLower(kw)
		if !strings.Contains(lower, klow) {
			continue
		}
		// The keyword is present; verify the rest of the summary is
		// trivial (few meaningful words).
		if meaningfulWords <= len(strings.Fields(kw))+2 {
			return kw
		}
	}
	return ""
}

// containsAny reports whether s contains any of the keywords.
func containsAny(s string, keywords []string) bool {
	low := strings.ToLower(s)
	for _, kw := range keywords {
		if strings.Contains(low, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// Reserved for future ASCII-whitespace / connector-word caches.
var _ = regexp.MustCompile
