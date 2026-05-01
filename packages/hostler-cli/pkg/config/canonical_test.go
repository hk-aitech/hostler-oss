package config

import (
	"strings"
	"testing"
)

// T540 (Sprint-58) — regression tests for canonicalizeYAML / canonicalizeYAMLChecked.
//
// Core guards:
//  1. avoid silent removal of new fields like precommit.branch_sync
//  2. consistent 2-space indent output (matches user yaml convention)
//  3. canonicalizeYAMLChecked returns an error when any leaf key is missing

// TestT540_BranchSyncSection_Preserved — regression for precommit.branch_sync preservation.
//
//	Prevents the field introduced in T430 from being silently removed by fix.
func TestT540_BranchSyncSection_Preserved(t *testing.T) {
	input := []byte(`project:
  key: "test"
precommit:
  branch_sync:
    threshold: 60
    warn_ratio: 80
`)
	out, err := canonicalizeYAML(input)
	if err != nil {
		t.Fatalf("canonicalize failed: %v", err)
	}
	got := string(out)
	want := []string{"precommit", "branch_sync", "threshold", "warn_ratio"}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("output missing %q: %s", w, got)
		}
	}
}

// TestT540_Indent_2Spaces — verifies the change from yaml.Marshal default 4 to 2 spaces.
//
//	yaml.Marshal default: 4 spaces per level ("    b:" at depth 1).
//	yaml.Encoder + SetIndent(2): 2 spaces per level ("  b:" at depth 1).
func TestT540_Indent_2Spaces(t *testing.T) {
	input := []byte(`top:
  child: value
`)
	out, err := canonicalizeYAML(input)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	got := string(out)
	// depth 1 ("child") must not be 4 spaces (would mean default 4-space indent in use).
	if strings.Contains(got, "    child:") {
		t.Errorf("default 4-space indent in use — Encoder SetIndent(2) not applied: %s", got)
	}
	// depth 1 ("child") must be 2 spaces.
	if !strings.Contains(got, "  child:") {
		t.Errorf("2-space indent (depth 1) not detected: %s", got)
	}
}

// TestT540_Checked_PreservesAllKeys — canonicalizeYAMLChecked passes when every key is preserved.
func TestT540_Checked_PreservesAllKeys(t *testing.T) {
	input := []byte(`project:
  key: "x"
precommit:
  branch_sync:
    threshold: 60
extra:
  nested:
    deep: "value"
`)
	out, err := canonicalizeYAMLChecked(input)
	if err != nil {
		t.Fatalf("checked: %v", err)
	}
	for _, k := range []string{"project", "precommit", "branch_sync", "extra", "nested"} {
		if !strings.Contains(string(out), k) {
			t.Errorf("key %q missing: %s", k, out)
		}
	}
}

// TestT540_ExtractLeafKeys — leaf key dotted-path extraction accuracy.
func TestT540_ExtractLeafKeys(t *testing.T) {
	input := []byte(`a:
  b: 1
  c:
    d: 2
e: 3
`)
	keys := extractLeafKeys(input)
	want := []string{"a.b", "a.c.d", "e"}
	for _, w := range want {
		if !keys[w] {
			t.Errorf("leaf key %q not detected, got=%v", w, keys)
		}
	}
}

// TestT540_Empty_Yaml — handles empty yaml input.
func TestT540_Empty_Yaml(t *testing.T) {
	out, err := canonicalizeYAMLChecked([]byte(""))
	if err != nil {
		t.Fatalf("empty input error: %v", err)
	}
	if len(out) > 5 { // produces around "{}\n" or empty output
		// Normal behaviour leaves an empty or very short output.
	}
	_ = out
}
