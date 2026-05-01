// Package config — nested-key drift detector.
//
// The previous detector checked only top-level YAML keys. Nested-key typos (e.g.
// `platform.dotnet.bounded_context` vs. the correct `bounded_contexts`) went
// undetected and silently dropped, allowing drift to accumulate even after a
// v1->v2 migration.
//
// Implementation strategy:
// 1. Walk the ProjectConfig struct via reflect to precompute the set of
// allowed YAML key paths (`allowedNestedKeyPaths`).
// 2. Unmarshal the YAML file into `map[string]any` and collect the set of
// paths actually used the same way.
// 3. Return the difference (used minus allowed) as "unknown nested keys".
//
// Constraints:
// - Free-form map fields (e.g. `reminders.<event>` and slice-of-string
// fields like `sprints.ceremony.design_checks`) are not walked into.
// When the reflect type is a map or a slice-of-string, the children are
// considered "user-managed" and skipped.
// - For tagged unions (`platform.kind` -> go/dotnet/python/node) the
// allowed set includes nested struct fields for every kind, so
// platform.go and platform.dotnet are both allowed simultaneously.
// Semantic validation lives in validatePlatformTaggedUnion, so this is
// not a duplicate check.
package config

import (
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// allowedNestedKeyPaths is the set of every nested YAML key path allowed by
// ProjectConfig (dot notation), e.g. "platform", "platform.kind",
// "platform.dotnet", "platform.dotnet.solution_file".
//
// Computed only once (no sync.Once — the scale does not warrant deferred
// init at package load); the first call walks reflect and caches the result.
var allowedNestedKeyPaths map[string]bool

// freeFormNestedPaths is the set of map-typed YAML paths automatically
// collected via reflect . Children below these paths are user-managed
// so the nested-key scan stops walking once it reaches them.
var freeFormNestedPaths map[string]bool

// nestedKeyPathSeparator is the YAML path separator.
const nestedKeyPathSeparator = "."

// ensureAllowedNestedKeyPaths populates allowedNestedKeyPaths and
// freeFormNestedPaths a single time. Not exported, for test isolation.
func ensureAllowedNestedKeyPaths() map[string]bool {
	if allowedNestedKeyPaths != nil {
		return allowedNestedKeyPaths
	}
	paths := make(map[string]bool)
	freeForm := make(map[string]bool)
	walkStructForYAMLPaths("", reflect.TypeOf(ProjectConfig{}), paths, freeForm)
	allowedNestedKeyPaths = paths
	freeFormNestedPaths = freeForm
	return paths
}

// walkStructForYAMLPaths recursively walks a Go struct and accumulates yaml-
// tag-based key paths into paths. Ptr and Slice-of-Struct unwrap via Elem
// and descend into the struct. Map and Slice-of-primitive halt the child
// walk.
func walkStructForYAMLPaths(prefix string, typ reflect.Type, paths, freeForm map[string]bool) {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tag := strings.Split(f.Tag.Get("yaml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		path := tag
		if prefix != "" {
			path = prefix + nestedKeyPathSeparator + tag
		}
		paths[path] = true

		ft := f.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		switch ft.Kind() {
		case reflect.Struct:
			walkStructForYAMLPaths(path, ft, paths, freeForm)
		case reflect.Slice:
			et := ft.Elem()
			for et.Kind() == reflect.Ptr {
				et = et.Elem()
			}
			if et.Kind() == reflect.Struct {
				walkStructForYAMLPaths(path, et, paths, freeForm)
			}
			// slice-of-string / slice-of-int: no child walk.
		case reflect.Map:
			// map[string]X uses free-form keys, so child traversal must stop here.
			freeForm[path] = true
		}
	}
}

// isFreeFormParent reports whether free-form keys are allowed beneath the
// given path. Looks up the freeFormNestedPaths set populated by
// walkStructForYAMLPaths; map[string]X fields are auto-included.
func isFreeFormParent(path string) bool {
	ensureAllowedNestedKeyPaths()
	return freeFormNestedPaths[path]
}

// collectYAMLKeyPaths unmarshals the YAML bytes into a map and collects the
// set of used key paths. Stops descending below any free-form parent.
func collectYAMLKeyPaths(data []byte) ([]string, error) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	var out []string
	walkMapForPaths("", root, &out)
	sort.Strings(out)
	return out, nil
}

// walkMapForPaths recursively walks a map[string]any and accumulates paths.
func walkMapForPaths(prefix string, m map[string]any, out *[]string) {
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + nestedKeyPathSeparator + k
		}
		*out = append(*out, path)
		if isFreeFormParent(path) {
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			walkMapForPaths(path, val, out)
		case []any:
			for _, item := range val {
				if sub, ok := item.(map[string]any); ok {
					walkMapForPaths(path, sub, out)
				}
			}
		}
	}
}

// FindUnknownNestedKeys returns the nested key paths in the YAML bytes that
// are not defined by the ProjectConfig schema .
//
// Top-level keys are already covered by findUnknownTopLevelKeys, so this
// includes only depths >= 1.
func FindUnknownNestedKeys(data []byte) ([]string, error) {
	used, err := collectYAMLKeyPaths(data)
	if err != nil {
		return nil, err
	}
	allowed := ensureAllowedNestedKeyPaths()
	var unknown []string
	for _, p := range used {
		if !strings.Contains(p, nestedKeyPathSeparator) {
			continue // top-level — handled by
		}
		if !allowed[p] {
			unknown = append(unknown, p)
		}
	}
	return unknown, nil
}
