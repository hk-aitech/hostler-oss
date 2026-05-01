package cmd

import (
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/rules"
)

// TestSeverityIcon_Mapping - icon mapping works correctly (pure function, no side effects).
func TestSeverityIcon_Mapping(t *testing.T) {
	cases := map[rules.Severity]string{
		rules.SeverityAdvisory:  rulesIconAdvisory,
		rules.SeverityWarn:      rulesIconWarn,
		rules.SeverityBlock:     rulesIconBlock,
		rules.SeverityHardBlock: rulesIconHardBlock,
		rules.SeverityOff:       rulesIconOff,
	}
	for sev, want := range cases {
		if got := severityIcon(sev); got != want {
			t.Errorf("severityIcon(%v) = %q, want %q", sev, got, want)
		}
	}
}

// TestRuleDescription_nil - default description value for a nil Rule.
func TestRuleDescription_nil(t *testing.T) {
	if ruleDescription(nil) == "" {
		t.Error("nil Rule description is an empty string")
	}
	if ruleDefaultSeverity(nil) != "unknown" {
		t.Errorf("nil default severity = %q, want unknown", ruleDefaultSeverity(nil))
	}
}

// TestChainEntriesToJSON - serializes a ChainEntry slice.
func TestChainEntriesToJSON(t *testing.T) {
	entries := []rules.ChainEntry{
		{Layer: "preset", Source: "recommended", Severity: rules.SeverityBlock},
		{Layer: "env", Source: "HSTL_RULE_X", Severity: rules.SeverityWarn},
	}
	out := chainEntriesToJSON(entries)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0]["layer"] != "preset" || out[0]["severity"] != "block" {
		t.Errorf("entry 0 = %+v", out[0])
	}
}

// TestFilterByPolicy_Hit - the policy filter only returns matching rules.
func TestFilterByPolicy_Hit(t *testing.T) {
	cfg := &rules.EngineConfig{
		Policies: []rules.PolicyConfig{
			{Name: "team/core", SeverityOverrides: map[string]rules.Severity{
				"a.b.c": rules.SeverityBlock,
				"x.y.z": rules.SeverityWarn,
			}},
		},
	}
	eff := map[string]*rules.EffectiveRule{
		"a.b.c":          {RuleID: "a.b.c", Severity: rules.SeverityBlock},
		"x.y.z":          {RuleID: "x.y.z", Severity: rules.SeverityWarn},
		"unrelated.rule": {RuleID: "unrelated.rule", Severity: rules.SeverityBlock},
	}
	out := filterByPolicy(cfg, eff, "team/core")
	if len(out) != 2 {
		t.Errorf("filter result = %d, want 2", len(out))
	}
	if _, ok := out["unrelated.rule"]; ok {
		t.Error("unrelated rule remains after filter")
	}
}

// TestFilterByPolicy_Miss - an unknown policy name returns an empty result.
func TestFilterByPolicy_Miss(t *testing.T) {
	cfg := &rules.EngineConfig{}
	out := filterByPolicy(cfg, nil, "nonexistent")
	if len(out) != 0 {
		t.Errorf("expected empty result, got %d", len(out))
	}
}

// TestLoadConfigFromURI_preset - preset:// wraps into EngineConfig.Extends.
func TestLoadConfigFromURI_preset(t *testing.T) {
	cfg, err := loadConfigFromURI("preset://hostler/recommended")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(cfg.Extends) != 1 || cfg.Extends[0] != "preset://hostler/recommended" {
		t.Errorf("extends = %v", cfg.Extends)
	}
}

// TestBuildEffectiveJSON_Schema - required fields of the effective JSON.
func TestBuildEffectiveJSON_Schema(t *testing.T) {
	cfg := &rules.EngineConfig{
		Extends:  []string{"preset://hostler/recommended"},
		Policies: []rules.PolicyConfig{{Name: "hostler/task-quality-core"}},
	}
	eff := map[string]*rules.EffectiveRule{
		"task.a": {RuleID: "task.a", Severity: rules.SeverityBlock, Origin: rules.Origin{Layer: "preset", Source: "recommended"}},
	}
	out := buildEffectiveJSON(cfg, eff)
	if out["version"] != rulesSchemaVersion {
		t.Errorf("version = %v", out["version"])
	}
	if _, ok := out["rules"].([]map[string]any); !ok {
		t.Error("rules slice type mismatch")
	}
	if _, ok := out["extends_chain"].([]string); !ok {
		t.Error("extends_chain type mismatch")
	}
}
