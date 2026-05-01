// Package config — canonical.go: deterministic YAML re-emission helpers.
//
// canonicalizeYAML was introduced as part of the Migration Runner and
// split out into its own file when the v1 migration runner was removed.
// config fix uses it to reorder keys canonically.
//
// yaml.Marshal's default 4-space indent clashed with the user's yaml
// convention (typically 2 spaces) and made the fix diff look destructive.
// Switched to Encoder + SetIndent(2) for consistent 2-space output and
// added a key-preservation sanity check.
package config

import (
	"bytes"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// CurrentSchemaMajor is the current major value of the top-level version field.
// Pinned to 2 because v2 is the only supported major.
const CurrentSchemaMajor uint = 2

// canonicalIndentSpaces — indent for canonical output (defaults to 2 spaces).
const canonicalIndentSpaces = 2

// canonicalizeYAML re-emits the supplied yaml bytes in deterministic form.
// All MappingNode keys are sorted alphabetically.
//
// Instead of yaml.Marshal's default 4-space indent, uses Encoder +
// SetIndent(2) to match the user's yaml convention (2 spaces). Prevents
// the perception that fix is destructive when only the indent changed.
func canonicalizeYAML(data []byte) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse yaml.Node: %w", err)
	}
	sortYAMLMappingsRecursive(&doc)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(canonicalIndentSpaces)
	if err := enc.Encode(&doc); err != nil {
		_ = enc.Close()
		return nil, fmt.Errorf("yaml.Encoder: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("yaml.Encoder close: %w", err)
	}
	return buf.Bytes(), nil
}

// canonicalizeYAMLChecked is the key-preserving sanity-check variant of canonicalizeYAML.
//
//	Guards against 'silent key removal'. Confirms that the
//	canonical result's leaf-key set is a superset of the original's. Returns
//	an error when keys are missing, preventing fix from silently losing user
//	settings.
func canonicalizeYAMLChecked(data []byte) ([]byte, error) {
	out, err := canonicalizeYAML(data)
	if err != nil {
		return nil, err
	}
	original := extractLeafKeys(data)
	canonical := extractLeafKeys(out)
	var missing []string
	for k := range original {
		if !canonical[k] {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("canonicalize result is missing original leaf keys: %v", missing)
	}
	return out, nil
}

// extractLeafKeys returns every leaf key (mapping keys whose value is a
// scalar) of the supplied yaml bytes as a dotted-path set. Used by the
// key-preservation sanity check.
func extractLeafKeys(data []byte) map[string]bool {
	keys := make(map[string]bool)
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return keys
	}
	walkLeafKeys(&doc, "", keys)
	return keys
}

// walkLeafKeys recursively walks the yaml.Node tree and collects leaf keys
// as dotted paths (e.g. "precommit.branch_sync.threshold").
func walkLeafKeys(node *yaml.Node, prefix string, keys map[string]bool) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, c := range node.Content {
			walkLeafKeys(c, prefix, keys)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			k := node.Content[i].Value
			path := k
			if prefix != "" {
				path = prefix + "." + k
			}
			v := node.Content[i+1]
			if v.Kind == yaml.MappingNode || v.Kind == yaml.SequenceNode {
				walkLeafKeys(v, path, keys)
			} else {
				keys[path] = true
			}
		}
	case yaml.SequenceNode:
		for i, c := range node.Content {
			path := fmt.Sprintf("%s[%d]", prefix, i)
			walkLeafKeys(c, path, keys)
		}
	}
}

// sortYAMLMappingsRecursive sorts every MappingNode in the yaml.Node tree
// alphabetically by key.
func sortYAMLMappingsRecursive(node *yaml.Node) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, c := range node.Content {
			sortYAMLMappingsRecursive(c)
		}
	case yaml.MappingNode:
		n := len(node.Content) / 2
		if n > 1 {
			type kv struct {
				k, v *yaml.Node
			}
			pairs := make([]kv, n)
			for i := 0; i < n; i++ {
				pairs[i] = kv{node.Content[2*i], node.Content[2*i+1]}
			}
			sort.SliceStable(pairs, func(i, j int) bool {
				return pairs[i].k.Value < pairs[j].k.Value
			})
			for i, p := range pairs {
				node.Content[2*i] = p.k
				node.Content[2*i+1] = p.v
			}
		}
		for i := 1; i < len(node.Content); i += 2 {
			sortYAMLMappingsRecursive(node.Content[i])
		}
	case yaml.SequenceNode:
		for _, c := range node.Content {
			sortYAMLMappingsRecursive(c)
		}
	}
}
