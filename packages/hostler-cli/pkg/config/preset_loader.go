// Package config — Profile Composition preset loader (preset_loader.go).
//
// Resolves the `extends: [builtin:<name>, ...]` field by loading and merging
// the embedded preset YAML files. Preset files are embedded into the Go
// binary, and nested extends are resolved recursively up to maxExtendsDepth
// (cycles are blocked with a clear error).
//
// Merge order (ESLint flat v10 "order matters"):
//  1. The first item in the extends array is the base.
//  2. The next item deep-merges on top (later wins).
//  3. The final YAML body wins overall.
//
// References: a synthesis of Renovate preset, ESLint extends arrays, and
// the Nx 3-layer merge model.
package config

import (
	"embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed presets/*.yaml
var presetFS embed.FS

// builtinPrefix is the reference prefix for built-in presets.
const builtinPrefix = "builtin:"

// maxExtendsDepth caps the nesting depth of extends.
// Acts as a humane limit even when no cycle is involved.
const maxExtendsDepth = 5

// LoadPreset takes a single preset name and returns the YAML parsed as map[string]any.
//
// Inputs:
//   - "builtin:hostler-base"   -> loads presets/hostler-base.yaml from the embed FS
//   - "builtin:go-plugin"      -> presets/go-plugin.yaml
//   - "builtin:dotnet-multi-bc" -> presets/dotnet-multi-bc.yaml
//
// Returns an error when the preset name is unknown or the file is missing.
// Non-builtin references (e.g. relative paths) are out of scope at the
// moment — extension is planned for a follow-up phase.
func LoadPreset(ref string) (map[string]any, error) {
	if !strings.HasPrefix(ref, builtinPrefix) {
		return nil, fmt.Errorf("unsupported preset reference %q (only %q prefix supported)", ref, builtinPrefix)
	}
	name := strings.TrimPrefix(ref, builtinPrefix)
	if name == "" {
		return nil, fmt.Errorf("empty preset name after prefix")
	}
	path := "presets/" + name + ".yaml"
	data, err := presetFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("preset %q not found: %w", ref, err)
	}
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse preset %q failed: %w", ref, err)
	}
	return out, nil
}

// ResolveExtends recursively resolves the extends chain and returns the
// merged result.
//
// Inputs:
//   - extends: list of references to resolve
//   - depth:   current recursion depth (callers pass 0)
//   - visited: cycle-detection set (callers may pass nil; managed internally)
//
// Behaviour:
//  1. Loads each ref via LoadPreset.
//  2. If a loaded preset has its own extends field, resolves it recursively (depth+1).
//  3. Combines them in order using mergeMaps (later wins).
//
// Returns: the merged map[string]any plus an error.
func ResolveExtends(extends []string, depth int, visited map[string]bool) (map[string]any, error) {
	if len(extends) == 0 {
		return nil, nil
	}
	if depth > maxExtendsDepth {
		return nil, fmt.Errorf("extends depth %d exceeded the max of %d — possible cycle", depth, maxExtendsDepth)
	}
	if visited == nil {
		visited = make(map[string]bool)
	}
	result := map[string]any{}
	for _, ref := range extends {
		if visited[ref] {
			return nil, fmt.Errorf("extends cycle detected: %q", ref)
		}
		visited[ref] = true
		presetMap, err := LoadPreset(ref)
		if err != nil {
			return nil, err
		}
		// Resolve the preset's own extends first.
		if nestedExtends, ok := presetMap["extends"]; ok {
			if nestedList, isList := toStringSlice(nestedExtends); isList {
				delete(presetMap, "extends") // remove the field after resolving
				nestedResult, err := ResolveExtends(nestedList, depth+1, visited)
				if err != nil {
					return nil, err
				}
				result = mergeMaps(result, nestedResult)
			}
		}
		result = mergeMaps(result, presetMap)
		// Re-referencing the same ref in a sibling chain is allowed (visited
		// is not path-based). Pointer equality alone is insufficient; here
		// we only guarantee "no duplicate visits within the same resolve call".
		delete(visited, ref)
	}
	return result, nil
}

// toStringSlice attempts to convert an interface{} value to []string.
// extends comes back as []any when loaded via YAML, so a type conversion is needed.
func toStringSlice(v any) ([]string, bool) {
	if v == nil {
		return nil, false
	}
	switch xs := v.(type) {
	case []string:
		return xs, true
	case []any:
		out := make([]string, 0, len(xs))
		for _, item := range xs {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}
