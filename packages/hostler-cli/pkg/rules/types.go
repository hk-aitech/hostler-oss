// Package rules — Rule/Policy/Config Engine.
//
// 4-layer architecture:
//
//	Rule     — atomic validation unit (atom)
//	Policy   — named bundle of Rules (bundle)
//	Config   — project-declared active Policies + overrides (rules.yaml)
//	Preset   — distribution-ready built-in Rule/Policy bundle (embed FS)
package rules

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Severity is the enforcement level of a Rule violation. The taxonomy is
// inspired by ESLint and HashiCorp Sentinel.
//
//	SeverityOff       : rule disabled (not checked)
//	SeverityAdvisory  : informational (log only)
//	SeverityWarn      : warning (does not block)
//	SeverityBlock     : default block (relaxable via policy)
//	SeverityHardBlock : non-relaxable block (env/config override forbidden)
type Severity int

const (
	SeverityOff Severity = iota
	SeverityAdvisory
	SeverityWarn
	SeverityBlock
	SeverityHardBlock
)

// ParseSeverity converts the string "off|advisory|warn|block|hard-block" to a
// Severity. Case-insensitive. Unknown values return an error.
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off", "":
		return SeverityOff, nil
	case "advisory", "info":
		return SeverityAdvisory, nil
	case "warn", "warning":
		return SeverityWarn, nil
	case "block", "error":
		return SeverityBlock, nil
	case "hard-block", "hardblock", "fatal":
		return SeverityHardBlock, nil
	default:
		return SeverityOff, fmt.Errorf("unknown severity: %q (off|advisory|warn|block|hard-block)", s)
	}
}

// String serialises Severity (used for config output and origin trace).
func (s Severity) String() string {
	switch s {
	case SeverityOff:
		return "off"
	case SeverityAdvisory:
		return "advisory"
	case SeverityWarn:
		return "warn"
	case SeverityBlock:
		return "block"
	case SeverityHardBlock:
		return "hard-block"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// UnmarshalYAML is a custom yaml.v3 unmarshaller. Decodes string → Severity.
func (s *Severity) UnmarshalYAML(node *yaml.Node) error {
	var raw string
	if err := node.Decode(&raw); err != nil {
		return err
	}
	v, err := ParseSeverity(raw)
	if err != nil {
		return err
	}
	*s = v
	return nil
}
