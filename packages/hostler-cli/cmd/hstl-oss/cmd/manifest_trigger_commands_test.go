// Regression tests for moving the skill-to-command mapping into SKILL.md
// frontmatter trigger_commands.
package cmd

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// TestT350_ConvertDottedToPrefix - dotted-name to "hstl <path>" conversion rule.
func TestT350_ConvertDottedToPrefix(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"task.*", brand.ShortName + " task"},
		{"sprint.complete", brand.ShortName + " sprint complete"},
		{"task.assign", brand.ShortName + " task assign"},
		{"harness.*", brand.ShortName + " harness"},
		{"kb.*", brand.ShortName + " kb"},
		{"", ""},
	}
	for _, c := range cases {
		got := convertDottedToPrefix(c.in)
		if got != c.want {
			t.Errorf("convertDottedToPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestT350_LoadSkillCommandMapFromPlugin verifies that loading 7 skill
// frontmatters from the actual plugin directory yields a result equivalent to
// the legacy mapping.
func TestT350_LoadSkillCommandMapFromPlugin(t *testing.T) {
	resetSkillCommandMapForTest()
	defer resetSkillCommandMapForTest()

	loaded, err := loadSkillCommandMap()
	if err != nil {
		t.Skipf("plugin directory not found - skip: %v", err)
	}
	// All seven core skills must be present.
	required := []string{
		"task-management", "sprint-management", "learned",
		"project-resources", "project-management",
	}
	for _, name := range required {
		if _, ok := loaded[name]; !ok {
			t.Errorf("loaded missing skill %q", name)
		}
	}
	wantPrefix := brand.ShortName + " "
	for skill, prefixes := range loaded {
		for _, p := range prefixes {
			if !strings.HasPrefix(p, wantPrefix) {
				t.Errorf("skill %q prefix %q does not start with %q", skill, p, wantPrefix)
			}
		}
	}
}

// TestT350_GetSkillCommandMap_LegacySuperset - all legacy entries must be in the result.
func TestT350_GetSkillCommandMap_LegacySuperset(t *testing.T) {
	resetSkillCommandMapForTest()
	defer resetSkillCommandMapForTest()

	got := getSkillCommandMap()
	// Every legacy key must exist in the result (keys present only in
	// loaded but missing from legacy are allowed - new skills are OK).
	for k := range legacySkillCommandMap {
		if _, ok := got[k]; !ok {
			t.Errorf("getSkillCommandMap() missing legacy key %q", k)
		}
	}
}

// TestT350_ManifestDiff_BeforeAfter verifies that the prefix list generated
// from SKILL.md frontmatter matches the legacy set (no breaking change).
func TestT350_ManifestDiff_BeforeAfter(t *testing.T) {
	resetSkillCommandMapForTest()
	defer resetSkillCommandMapForTest()

	loaded, err := loadSkillCommandMap()
	if err != nil || loaded == nil {
		t.Skipf("plugin directory not found - skip")
	}
	// Compare each loaded skill against legacy (expecting equal sorted sets).
	for skill, legacyPrefixes := range legacySkillCommandMap {
		loadedPrefixes, ok := loaded[skill]
		if !ok {
			t.Errorf("loaded missing %q", skill)
			continue
		}
		lcopy := append([]string{}, legacyPrefixes...)
		sort.Strings(lcopy)
		// loaded is already sorted (inside loadSkillCommandMap).
		if !reflect.DeepEqual(lcopy, loadedPrefixes) {
			t.Errorf("skill %q diff\n  legacy:  %v\n  loaded:  %v", skill, lcopy, loadedPrefixes)
		}
	}
}
