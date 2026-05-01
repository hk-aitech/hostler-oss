// Package harnessdefaults is a standalone utility package that parses
// the embedded harness_defaults.json. pkg/db and pkg/sprint both depend
// on it without introducing a cyclic dependency, so default Harness
// settings can be read freely.
package harnessdefaults

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed schemas/harness_defaults.json
var defaultsJSON []byte

// LoadSprintPhaseNames returns the sprint:default item names from
// harness_defaults.json in order.
func LoadSprintPhaseNames() ([]string, error) {
	var defaults map[string]json.RawMessage
	if err := json.Unmarshal(defaultsJSON, &defaults); err != nil {
		return nil, fmt.Errorf("harness_defaults.json parse failed: %w", err)
	}
	raw, ok := defaults["sprint:default"]
	if !ok {
		return nil, fmt.Errorf("harness_defaults.json missing sprint:default key")
	}
	var items []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("sprint:default parse failed: %w", err)
	}
	names := make([]string, 0, len(items))
	for _, it := range items {
		names = append(names, it.Name)
	}
	return names, nil
}

// DefaultsJSON returns the raw harness_defaults.json bytes.
// pkg/db uses this during DB initialisation.
func DefaultsJSON() []byte {
	return defaultsJSON
}
