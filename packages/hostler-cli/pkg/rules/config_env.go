// project-config.yaml → subprocess env injection wrapper.
//
// Purpose: when a precommit rule spawns a shell script, project-specific
// configuration (e.g. precommit.branch_sync) is forwarded as environment
// variables. Priority:
//
//   1. environment variables already set (one-off developer override)
//   2. project-config.yaml (team-shared persistent settings)
//   3. defaults internal to the shell script
//
// When the env is already set, the config value is not injected (the
// original is preserved).

package rules

import (
	"os"
	"strconv"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
)

// buildPrecommitEnv returns the list of additional environment variables
// for a given precommit rule ID. For each rule, the relevant config
// section is consulted and only env values that are not already set are
// filled in.
//
// rule mappings:
//   - precommit.branch.sync → precommit.branch_sync (threshold, warn_ratio, enabled, strict_mode)
//
// Returns: slice of "KEY=VALUE" strings. An empty result keeps the env
// inherited as-is.
func buildPrecommitEnv(ruleID string) []string {
	switch ruleID {
	case "precommit.branch.sync":
		return branchSyncEnv()
	}
	return nil
}

// branchSyncEnv builds the env slice for the precommit.branch.sync rule.
// Returns an empty slice when the config fails to load (the rule then
// runs with its defaults).
func branchSyncEnv() []string {
	cfg, err := config.LoadProjectConfig()
	if err != nil || cfg == nil {
		return nil
	}
	bs := cfg.GetBranchSyncConfig()
	if bs == nil {
		return nil
	}

	var envs []string
	injectIntEnv := func(envKey string, ptr *int) {
		if ptr == nil {
			return
		}
		if _, set := os.LookupEnv(envKey); set {
			return // env already set — env takes precedence
		}
		envs = append(envs, envKey+"="+strconv.Itoa(*ptr))
	}
	injectBoolEnv := func(envKey, trueValue, falseValue string, ptr *bool) {
		if ptr == nil {
			return
		}
		if _, set := os.LookupEnv(envKey); set {
			return
		}
		if *ptr {
			envs = append(envs, envKey+"="+trueValue)
		} else {
			envs = append(envs, envKey+"="+falseValue)
		}
	}

	injectIntEnv("HSTL_BRANCH_SYNC_THRESHOLD", bs.Threshold)
	injectIntEnv("HSTL_BRANCH_SYNC_WARN_RATIO", bs.WarnRatio)
	// Enabled=false → HSTL_BRANCH_SYNC=off (the shell script's opt-out
	// form). Enabled=true matches the default, so it is not injected
	// (reduces noise).
	if bs.Enabled != nil && !*bs.Enabled {
		if _, set := os.LookupEnv("HSTL_BRANCH_SYNC"); !set {
			envs = append(envs, "HSTL_BRANCH_SYNC=off")
		}
	}
	// StrictMode=true → HSTL_BRANCH_SYNC_STRICT=1.
	injectBoolEnv("HSTL_BRANCH_SYNC_STRICT", "1", "0", bs.StrictMode)

	return envs
}
