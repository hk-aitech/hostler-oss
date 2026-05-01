// Package cmd - dependency helpers for hooks doctor.
//
// Split out from hooks.go to isolate the yaml_loader / rules dependencies in a
// single file. This is the only hooks path where the cmd package depends on
// pkg/config or pkg/rules.
package cmd

import (
	"fmt"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/rules"
)

// loadProjectConfigSilent returns the LoadProjectConfig result silently.
//
// rootHint is currently unused (LoadProjectConfig resolves paths internally
// from cwd). Reserved as an argument for a future root override.
func loadProjectConfigSilent(rootHint string) (*config.ProjectConfig, error) {
	_ = rootHint // explicitly ignored - reserved for future expansion
	return config.LoadProjectConfig()
}

// countPrecommitRules counts effective rules whose ID has the "precommit." prefix.
//
// Uses the rules.LoadConfig + ResolveEffective flow. Wraps errors on failure.
func countPrecommitRules(projectRoot string) (int, error) {
	cfg, err := rules.LoadConfig(projectRoot)
	if err != nil {
		return 0, fmt.Errorf("rules.LoadConfig: %w", err)
	}
	eff, err := rules.ResolveEffective(cfg, projectRoot)
	if err != nil {
		return 0, fmt.Errorf("rules.ResolveEffective: %w", err)
	}
	count := 0
	for id := range eff {
		if strings.HasPrefix(id, "precommit.") {
			count++
		}
	}
	return count, nil
}
