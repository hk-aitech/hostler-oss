// Package config — briefing.sections[].extract DSL parser and validator ( I09).
//
// Resolves D10 from the research doc (extract value is a mini DSL but lacks a
// spec) by formalising the valid extract values as `ExtractKind` enum + parser.
//
// DSL form (EBNF):
//
//	extract ::= literal | "section:" heading | "file:head(" digits ")"
//	literal ::= "first-heading-block" | "markdown-table"
//	heading ::= "#"{1,6} " " text
//	digits ::= [1-9] [0-9]*
//	text ::= arbitrary string (whitespace allowed, no newlines)
//
// Authoritative spec: docs/05-reports/extract-dsl.md
package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ExtractKind represents the top-level kind of the extract DSL.
type ExtractKind string

const (
	// ExtractKindLiteral — argument-less literal keywords like first-heading-block, markdown-table.
	ExtractKindLiteral ExtractKind = "literal"

	// ExtractKindSection — `section:{heading}` form. heading is a markdown
	// heading line itself (e.g. `## Overview`).
	ExtractKindSection ExtractKind = "section"

	// ExtractKindFileHead — `file:head(N)` form. N is a positive integer
	// (same meaning as the head command: first N lines of the file).
	ExtractKindFileHead ExtractKind = "file_head"
)

// ExtractLiteral enumerates the argument-less literal keywords.
type ExtractLiteral string

const (
	ExtractLiteralFirstHeadingBlock ExtractLiteral = "first-heading-block"
	ExtractLiteralMarkdownTable ExtractLiteral = "markdown-table"
)

// knownLiterals is the set of allowed literal keywords. When adding a new
// literal, update this slice and the spec at extract-dsl.md together.
var knownLiterals = []ExtractLiteral{
	ExtractLiteralFirstHeadingBlock,
	ExtractLiteralMarkdownTable,
}

// prefixes
const (
	sectionPrefix = "section:"
	fileHeadPrefixOpen = "file:head("
	fileHeadSuffix = ")"
)

// maxFileHeadLines caps N in file:head(N). An unbounded N risks memory abuse
// when the value is a typo, so an upper bound is enforced. In real briefing
// section usage a few dozen lines is enough.
const maxFileHeadLines = 1000

// sectionHeadingRe — heading validity for section:{heading}: 1-6 `#` + space + text.
// The text portion may include any character set (no script restrictions).
var sectionHeadingRe = regexp.MustCompile(`^#{1,6} \S.*$`)

// ExtractExpr is the structured representation of a parsed extract value.
type ExtractExpr struct {
	Kind ExtractKind

	// Literal — only valid when Kind == ExtractKindLiteral.
	Literal ExtractLiteral

	// Heading — only valid when Kind == ExtractKindSection.
	Heading string

	// FileHeadLines — only valid when Kind == ExtractKindFileHead.
	FileHeadLines int
}

// ErrInvalidExtract is the sentinel error returned on parse failure.
var ErrInvalidExtract = errors.New("invalid extract DSL")

// ParseExtract parses a briefing.sections[].extract value.
// An empty string is valid (means dynamic section extraction is disabled) —
// returns (zero ExtractExpr, nil).
func ParseExtract(s string) (ExtractExpr, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ExtractExpr{}, nil
	}

	// 1. literal
	for _, lit := range knownLiterals {
		if trimmed == string(lit) {
			return ExtractExpr{Kind: ExtractKindLiteral, Literal: lit}, nil
		}
	}

	// 2. section:{heading}
	if heading, ok := strings.CutPrefix(trimmed, sectionPrefix); ok {
		if !sectionHeadingRe.MatchString(heading) {
			return ExtractExpr{}, fmt.Errorf("%w: section heading must be '#' 1-6 + space + text (got: %q)", ErrInvalidExtract, heading)
		}
		return ExtractExpr{Kind: ExtractKindSection, Heading: heading}, nil
	}

	// 3. file:head(N)
	if inner, ok := strings.CutPrefix(trimmed, fileHeadPrefixOpen); ok {
		n, nerr := parseFileHeadArgs(inner)
		if nerr != nil {
			return ExtractExpr{}, nerr
		}
		return ExtractExpr{Kind: ExtractKindFileHead, FileHeadLines: n}, nil
	}

	return ExtractExpr{}, fmt.Errorf("%w: unknown form (got: %q). valid forms: %s",
		ErrInvalidExtract, trimmed, strings.Join(knownExtractFormats(), ", "))
}

// parseFileHeadArgs parses the "N)" string after the file:head( prefix.
func parseFileHeadArgs(inner string) (int, error) {
	if !strings.HasSuffix(inner, fileHeadSuffix) {
		return 0, fmt.Errorf("%w: file:head(N) is missing the closing parenthesis", ErrInvalidExtract)
	}
	numStr := strings.TrimSuffix(inner, fileHeadSuffix)
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("%w: N in file:head(N) must be an integer (got: %q)", ErrInvalidExtract, numStr)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%w: N in file:head(N) must be positive (got: %d)", ErrInvalidExtract, n)
	}
	if n > maxFileHeadLines {
		return 0, fmt.Errorf("%w: N in file:head(N) must be <= %d (got: %d)", ErrInvalidExtract, maxFileHeadLines, n)
	}
	return n, nil
}

// IsValidExtract reports whether parsing succeeds, without returning the
// error. Used by schema validation and similar callers.
func IsValidExtract(s string) bool {
	_, err := ParseExtract(s)
	return err == nil
}

// knownExtractFormats returns the list of valid forms used in error messages.
func knownExtractFormats() []string {
	out := []string{
		sectionPrefix + "{heading}",
		fileHeadPrefixOpen + "N" + fileHeadSuffix,
	}
	for _, lit := range knownLiterals {
		out = append(out, string(lit))
	}
	return out
}

// extractPattern is the regex used by the JSON Schema `pattern` field. A
// single anchored regex covers all four forms above. JSON Schema regex is
// ECMA-262, and this regex stays within that subset.
//
// Whitespace is mandatory: a space follows `#` after section:, and the
// numeric form is required inside file:head(N).
const extractPattern = `^(first-heading-block|markdown-table|section:#{1,6} \S.*|file:head\([1-9][0-9]*\))$`

// ExtractPattern returns the regex string used for JSON Schema generation and validation.
func ExtractPattern() string { return extractPattern }
