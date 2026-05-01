// Package rules — Rule/Policy/Config Engine.
//
// # 4-layer structure
//
//	Rule     — atomic validation unit (Rule interface)
//	Policy   — named bundle of Rules (PolicyConfig)
//	Config   — project declaration (rules.yaml → EngineConfig)
//	Preset   — distribution bundle (embed.FS: presets/*.yaml)
//
// # Usage example
//
//	cfg, err := rules.LoadConfig(projectRoot)
//	if err != nil { ... }
//	eff, err := rules.ResolveEffective(cfg, projectRoot)
//	for id, r := range eff {
//	    fmt.Printf("%s: %s (from %s)\n", id, r.Severity, r.Origin.Layer)
//	}
//
// # Severity cascade priority
//
//  1. extends chain (preset → file)
//  2. policies.severity_overrides
//  3. rules top-level override
//  4. HSTL_RULE_<ID> environment variable
//
// hard-block does not allow env override (non-relaxable shield).
package rules
