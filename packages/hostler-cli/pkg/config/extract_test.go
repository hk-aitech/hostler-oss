package config

import (
	"regexp"
	"testing"
)

func TestParseExtract_EmptyStringAllowed(t *testing.T) {
	expr, err := ParseExtract("")
	if err != nil {
		t.Fatalf("empty string must be allowed: %v", err)
	}
	if expr.Kind != "" {
		t.Errorf("expected empty Kind for empty string, got %q", expr.Kind)
	}
}

func TestParseExtract_LiteralPasses(t *testing.T) {
	cases := []struct {
		in   string
		want ExtractLiteral
	}{
		{"first-heading-block", ExtractLiteralFirstHeadingBlock},
		{"markdown-table", ExtractLiteralMarkdownTable},
	}
	for _, c := range cases {
		expr, err := ParseExtract(c.in)
		if err != nil {
			t.Errorf("ParseExtract(%q) error: %v", c.in, err)
			continue
		}
		if expr.Kind != ExtractKindLiteral {
			t.Errorf("%q Kind=%s (expected literal)", c.in, expr.Kind)
		}
		if expr.Literal != c.want {
			t.Errorf("%q Literal=%s (expected %s)", c.in, expr.Literal, c.want)
		}
	}
}

func TestParseExtract_section_heading(t *testing.T) {
	valid := []string{"section:# Overview", "section:## Requirements", "section:###### deep"}
	for _, s := range valid {
		expr, err := ParseExtract(s)
		if err != nil {
			t.Errorf("ParseExtract(%q) error: %v", s, err)
			continue
		}
		if expr.Kind != ExtractKindSection {
			t.Errorf("%q Kind=%s", s, expr.Kind)
		}
	}

	invalid := []string{"section:", "section:no-hash", "section:#nospace", "section:####### too-deep"}
	for _, s := range invalid {
		if _, err := ParseExtract(s); err == nil {
			t.Errorf("ParseExtract(%q) expected error", s)
		}
	}
}

func TestParseExtract_file_head(t *testing.T) {
	expr, err := ParseExtract("file:head(20)")
	if err != nil {
		t.Fatalf("file:head(20) error: %v", err)
	}
	if expr.Kind != ExtractKindFileHead || expr.FileHeadLines != 20 {
		t.Errorf("expr=%+v", expr)
	}

	invalid := []string{"file:head()", "file:head(0)", "file:head(-5)", "file:head(1001)", "file:head(abc)", "file:head(20", "file:head"}
	for _, s := range invalid {
		if _, err := ParseExtract(s); err == nil {
			t.Errorf("ParseExtract(%q) expected error", s)
		}
	}
}

func TestParseExtract_UnknownForm(t *testing.T) {
	invalid := []string{"unknown", "xyz:abc", "section: no-hash"}
	for _, s := range invalid {
		_, err := ParseExtract(s)
		if err == nil {
			t.Errorf("ParseExtract(%q) expected error", s)
		}
	}
}

func TestIsValidExtract(t *testing.T) {
	cases := map[string]bool{
		"":                    true, // empty string allowed
		"section:## Overview": true,
		"first-heading-block": true,
		"markdown-table":      true,
		"file:head(50)":       true,
		"unknown":             false,
		"mcp:project_status":  false,
	}
	for s, want := range cases {
		if got := IsValidExtract(s); got != want {
			t.Errorf("IsValidExtract(%q)=%v (expected %v)", s, got, want)
		}
	}
}

func TestExtractPattern_RegexMatches(t *testing.T) {
	// The regex injected into JSON Schema must match ParseExtract's results.
	re := regexp.MustCompile(ExtractPattern())
	valid := []string{
		"first-heading-block",
		"markdown-table",
		"section:## Overview",
		"file:head(20)",
	}
	for _, s := range valid {
		if !re.MatchString(s) {
			t.Errorf("regex rejected valid value %q", s)
		}
	}
	invalid := []string{"unknown", "mcp:project_status", "section:nospace", "file:head(0)"}
	for _, s := range invalid {
		if re.MatchString(s) {
			t.Errorf("regex accepted invalid value %q", s)
		}
	}
}
