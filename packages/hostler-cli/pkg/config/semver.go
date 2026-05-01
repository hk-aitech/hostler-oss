// Package config — SemVer parser scaffolding (semver.go).
//
// SemVer parser for the top-level version field of project-config.yaml.
//
// schema_version was historically expressed as the integer 1; the v2
// schema replaces it with the SemVer-shaped top-level `version` string:
//
//	v1 (deprecated): schema_version: 1
//	v2 (new):        version: "2.0.0"
//
// The parser exposes Compatible() to express version compatibility:
//   - Identical          — completely identical
//   - CompatibleMinor    — major matches, only minor or below differs
//   - CompatiblePatch    — major/minor match, only patch differs
//   - IncompatibleMajor  — major mismatch (fail-fast trigger)
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// SchemaVersion is a SemVer major.minor.patch struct.
type SchemaVersion struct {
	Major uint
	Minor uint
	Patch uint
}

// ParseSchemaVersion parses a "M.m.p" string into a SchemaVersion.
//
// Allowed formats:
//
//	"2.0.0"     -> {2,0,0}
//	"1.2.3"     -> {1,2,3}
//	"10.20.30"  -> {10,20,30}
//
// Rejected formats:
//
//	"2"         -> error (major only)
//	"2.0"       -> error (patch missing)
//	"2.0.0.1"   -> error (four parts)
//	"v2.0.0"    -> error (no v prefix allowed)
//	""          -> error (empty string)
func ParseSchemaVersion(s string) (SchemaVersion, error) {
	if s == "" {
		return SchemaVersion{}, fmt.Errorf("empty version string")
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return SchemaVersion{}, fmt.Errorf("version format must be major.minor.patch: %q", s)
	}
	major, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return SchemaVersion{}, fmt.Errorf("parse major %q: %w", parts[0], err)
	}
	minor, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return SchemaVersion{}, fmt.Errorf("parse minor %q: %w", parts[1], err)
	}
	patch, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return SchemaVersion{}, fmt.Errorf("parse patch %q: %w", parts[2], err)
	}
	return SchemaVersion{
		Major: uint(major),
		Minor: uint(minor),
		Patch: uint(patch),
	}, nil
}

// String serialises in "M.m.p" form.
func (v SchemaVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compatibility is the relationship between two SchemaVersions.
type Compatibility int

const (
	// CompatibilityIdentical means both versions are completely identical.
	CompatibilityIdentical Compatibility = iota
	// CompatibilityPatch means major/minor match while only patch differs.
	CompatibilityPatch
	// CompatibilityMinor means major matches while only minor or below differs.
	CompatibilityMinor
	// CompatibilityIncompatible means major mismatch (fail-fast trigger).
	CompatibilityIncompatible
)

// String is the human-friendly representation of Compatibility.
func (c Compatibility) String() string {
	switch c {
	case CompatibilityIdentical:
		return "identical"
	case CompatibilityPatch:
		return "patch"
	case CompatibilityMinor:
		return "minor"
	case CompatibilityIncompatible:
		return "incompatible"
	default:
		return "unknown"
	}
}

// Compatible returns the compatibility relation between self and other.
//
// Rules:
//   - different major -> Incompatible
//   - same major, different minor -> Minor (backward compatible)
//   - same major/minor, different patch -> Patch
//   - all equal -> Identical
func (v SchemaVersion) Compatible(other SchemaVersion) Compatibility {
	if v.Major != other.Major {
		return CompatibilityIncompatible
	}
	if v.Minor != other.Minor {
		return CompatibilityMinor
	}
	if v.Patch != other.Patch {
		return CompatibilityPatch
	}
	return CompatibilityIdentical
}
