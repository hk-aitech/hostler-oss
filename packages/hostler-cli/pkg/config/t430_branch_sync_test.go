package config

import (
	"testing"
)

// T430 (Sprint-36) — precommit.branch_sync + sprint_start.divergence_warn_commits
// config field parsing checks.

func TestT430_BranchSyncConfig_Parsing(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, `
project:
  key: test-proj
precommit:
  branch_sync:
    threshold: 60
    warn_ratio: 70
    enabled: true
    strict_mode: false
sprint_start:
  divergence_warn_commits: 50
`)
	cfg, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg nil")
	}
	bs := cfg.GetBranchSyncConfig()
	if bs == nil {
		t.Fatal("BranchSyncConfig nil — yaml parsing failed")
	}
	if bs.Threshold == nil || *bs.Threshold != 60 {
		t.Errorf("expected Threshold 60, got %v", bs.Threshold)
	}
	if bs.WarnRatio == nil || *bs.WarnRatio != 70 {
		t.Errorf("expected WarnRatio 70, got %v", bs.WarnRatio)
	}
	if bs.Enabled == nil || *bs.Enabled != true {
		t.Errorf("expected Enabled true, got %v", bs.Enabled)
	}
	if bs.StrictMode == nil || *bs.StrictMode != false {
		t.Errorf("expected StrictMode false, got %v", bs.StrictMode)
	}
	ss := cfg.GetSprintStartConfig()
	if ss == nil || ss.DivergenceWarnCommits == nil || *ss.DivergenceWarnCommits != 50 {
		t.Errorf("expected SprintStart.DivergenceWarnCommits 50, got %v", ss)
	}
}

func TestT430_BranchSyncConfig_Empty(t *testing.T) {
	tmp := setupTempRoot(t)
	writeTestYAML(t, tmp, `
project:
  key: test-proj
`)
	cfg, err := LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if cfg.GetBranchSyncConfig() != nil {
		t.Error("GetBranchSyncConfig must be nil when precommit section is missing")
	}
	if cfg.GetSprintStartConfig() != nil {
		t.Error("GetSprintStartConfig must be nil when sprint_start section is missing")
	}
}

// knownTopLevelKeys must include precommit / sprint_start so they are not
// flagged as unknown_top_level_key.
func TestT430_KnownTopLevelKeys_Includes(t *testing.T) {
	for _, key := range []string{"precommit", "sprint_start"} {
		if !knownTopLevelKeys[key] {
			t.Errorf("knownTopLevelKeys missing %q", key)
		}
	}
}
