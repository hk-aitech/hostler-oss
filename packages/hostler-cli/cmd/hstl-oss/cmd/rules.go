package cmd

// rules.go — Rule Engine observability CLI (T570).
// Subcommands: list / explain / effective / validate
// SSOT: cli/pkg/rules (T567-T569)

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/rules"
)

const (
	rulesSchemaVersion = 1

	rulesIconAdvisory  = "ℹ"
	rulesIconWarn      = "⚠"
	rulesIconBlock     = "⛔"
	rulesIconHardBlock = "🔒"
	rulesIconOff       = "∅"
)

// rulesCmd is the parent command.
var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Rule Engine inspection / validation (list / explain / effective / validate)",
	Long: fmt.Sprintf(`%s Rule Engine (Sprint-64 Phase 1) observability CLI.

- list       — list every registered Rule (filter by category / enabled)
- explain    — show one Rule's cascade origin chain
- effective  — dump every effective rule (for CI / diff, --diff/--target/--policy)
- validate   — validate rules.yaml`, brand.ShortName),
}

// ── rules list ───────────────────────────────────────────────────────

var (
	rulesListCategory string
	rulesListEnabled  bool
)

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered Rules",
	Long:  "Prints every Rule registered in the global Registry, in alphabetical order. Filter with --category.",
	RunE: func(cmd *cobra.Command, args []string) error {
		allRules, _ := app.RulesList().([]rules.Rule)

		// category filter
		filtered := make([]rules.Rule, 0, len(allRules))
		for _, r := range allRules {
			if rulesListCategory != "" && r.Category() != rulesListCategory {
				continue
			}
			filtered = append(filtered, r)
		}

		// enabled filter: only rules whose current config severity is not Off.
		if rulesListEnabled {
			cfgRaw, err := app.RulesLoadConfig(app.ProjectRoot())
			if err != nil {
				return err
			}
			cfg, _ := cfgRaw.(*rules.EngineConfig)
			effRaw, err := app.RulesResolveEffective(cfg, app.ProjectRoot())
			if err != nil {
				return err
			}
			eff, _ := effRaw.(map[string]*rules.EffectiveRule)
			var enabledOnly []rules.Rule
			for _, r := range filtered {
				if e, ok := eff[r.ID()]; ok && e.Severity != rules.SeverityOff {
					enabledOnly = append(enabledOnly, r)
				}
			}
			filtered = enabledOnly
		}

		if outputFormat == "json" {
			out := make([]map[string]any, 0, len(filtered))
			for _, r := range filtered {
				out = append(out, map[string]any{
					"id":               r.ID(),
					"category":         r.Category(),
					"description":      r.Description(),
					"default_severity": r.DefaultSeverity().String(),
				})
			}
			Out.Print(map[string]any{
				"version": rulesSchemaVersion,
				"count":   len(out),
				"rules":   out,
			})
			return nil
		}

		headers := []string{"ID", "Category", "Default", "Description"}
		rows := make([][]string, 0, len(filtered))
		for _, r := range filtered {
			rows = append(rows, []string{r.ID(), r.Category(), r.DefaultSeverity().String(), r.Description()})
		}
		Out.Table(headers, rows)
		fmt.Fprintf(output.Stderr(), "\n%d rule(s)\n", len(filtered))
		return nil
	},
}

// ── rules explain ───────────────────────────────────────────────────

var rulesExplainCmd = &cobra.Command{
	Use:   "explain <rule_id>",
	Short: "Print one Rule's cascade origin chain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ruleID := args[0]
		rRaw, ok := app.RulesGet(ruleID)
		var r rules.Rule
		if ok {
			r, _ = rRaw.(rules.Rule)
		}
		// If absent from the Registry, r == nil — show only the cascade
		// result.

		cfgRaw, err := app.RulesLoadConfig(app.ProjectRoot())
		if err != nil {
			return err
		}
		cfg, _ := cfgRaw.(*rules.EngineConfig)
		effRaw, err := app.RulesResolveEffective(cfg, app.ProjectRoot())
		if err != nil {
			return err
		}
		eff, _ := effRaw.(map[string]*rules.EffectiveRule)
		e, found := eff[ruleID]
		if !found {
			Out.Error(fmt.Sprintf("Rule not found: %s", ruleID), apperr.CategoryNotFound.String(), "")
			os.Exit(exitError)
		}

		if outputFormat == "json" {
			Out.Print(map[string]any{
				"version":          rulesSchemaVersion,
				"id":               ruleID,
				"description":      ruleDescription(r),
				"default_severity": ruleDefaultSeverity(r),
				"current_severity": e.Severity.String(),
				"origin":           map[string]any{"layer": e.Origin.Layer, "source": e.Origin.Source},
				"previous_chain":   chainEntriesToJSON(e.PreviousChain),
			})
			return nil
		}

		fmt.Printf("ID: %s\n", ruleID)
		fmt.Printf("Description: %s\n", ruleDescription(r))
		fmt.Printf("Default Severity: %s\n", ruleDefaultSeverity(r))
		fmt.Printf("Current Severity: %s  (from %s.%s)\n", e.Severity, e.Origin.Layer, e.Origin.Source)
		fmt.Println("Origin Chain:")
		for i, c := range e.PreviousChain {
			fmt.Printf("  %d. %-12s %-30s → %s\n", i+1, c.Layer, c.Source, c.Severity)
		}
		fmt.Printf("  ★ WINNER    %-30s → %s\n", e.Origin.Layer+"."+e.Origin.Source, e.Severity)
		return nil
	},
}

// ── rules effective ─────────────────────────────────────────────────

var (
	rulesEffectivePolicy string
	rulesEffectiveDiff   string
	rulesEffectiveTarget string
)

var rulesEffectiveCmd = &cobra.Command{
	Use:   "effective",
	Short: "Dump every effective rule (after cascade resolution)",
	Long: fmt.Sprintf(`Loads the project's rules.yaml, completes cascade resolution,
and prints every effective rule along with its origin chain. Filter with the
CI/diff/target options.

Examples:
  %s rules effective                              # everything
  %s rules effective -o json                      # for CI pipelines
  %s rules effective --policy hostler/task-quality-core
  %s rules effective --diff preset://hostler/strict # diff vs strict preset
  %s rules effective --target works/tasks/T567.md # path-glob filter`,
		brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName, brand.ShortName),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgRaw, err := app.RulesLoadConfig(app.ProjectRoot())
		if err != nil {
			return err
		}
		cfg, _ := cfgRaw.(*rules.EngineConfig)
		effRaw, err := app.RulesResolveEffective(cfg, app.ProjectRoot())
		if err != nil {
			return err
		}
		eff, _ := effRaw.(map[string]*rules.EffectiveRule)

		// --policy filter: keep only Rules listed in the inline policy's
		// SeverityOverrides.
		if rulesEffectivePolicy != "" {
			eff = filterByPolicy(cfg, eff, rulesEffectivePolicy)
		}

		// --target filter: limit to policies matched by the glob.
		if rulesEffectiveTarget != "" {
			policies := cfg.PoliciesForPath(rulesEffectiveTarget)
			if len(policies) > 0 {
				policySet := make(map[string]bool)
				for _, p := range policies {
					policySet[p] = true
				}
				filtered := make(map[string]*rules.EffectiveRule)
				for id, r := range eff {
					// Keep only rules listed in SeverityOverrides for
					// policies whose paths matched the target.
					for _, p := range cfg.Policies {
						if !policySet[p.Name] {
							continue
						}
						if _, has := p.SeverityOverrides[id]; has {
							filtered[id] = r
							break
						}
					}
				}
				eff = filtered
			}
		}

		// --diff: compare against another preset/config.
		if rulesEffectiveDiff != "" {
			other, err := loadConfigFromURI(rulesEffectiveDiff)
			if err != nil {
				return err
			}
			otherEffRaw, err := app.RulesResolveEffective(other, app.ProjectRoot())
			if err != nil {
				return err
			}
			otherEff, _ := otherEffRaw.(map[string]*rules.EffectiveRule)
			return printDiff(eff, otherEff, rulesEffectiveDiff)
		}

		if outputFormat == "json" {
			Out.Print(buildEffectiveJSON(cfg, eff))
			return nil
		}
		printEffectiveText(eff)
		return nil
	},
}

// ── rules validate ──────────────────────────────────────────────────

var rulesValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate rules.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgRaw, err := app.RulesLoadConfig(app.ProjectRoot())
		if err != nil {
			Out.Error("load failed: "+err.Error(), apperr.CategoryConfigInvalid.String(), "")
			os.Exit(exitError)
		}
		cfg, _ := cfgRaw.(*rules.EngineConfig)

		var errs []string
		_, err = app.RulesResolveEffective(cfg, app.ProjectRoot())
		if err != nil {
			errs = append(errs, err.Error())
		}

		// Verify rule IDs in policies.includes exist.
		registry := make(map[string]bool)
		allRules, _ := app.RulesList().([]rules.Rule)
		for _, r := range allRules {
			registry[r.ID()] = true
		}
		for _, p := range cfg.Policies {
			for _, id := range p.Includes {
				if !registry[id] {
					errs = append(errs, fmt.Sprintf("policy %q: missing rule %q", p.Name, id))
				}
			}
		}

		// Validate target globs.
		for i, t := range cfg.Targets {
			for _, pattern := range t.Paths {
				if _, err := matchGlobValidate(pattern); err != nil {
					errs = append(errs, fmt.Sprintf("targets[%d]: invalid glob %q: %v", i, pattern, err))
				}
			}
		}

		if outputFormat == "json" {
			Out.Print(map[string]any{
				"version": rulesSchemaVersion,
				"valid":   len(errs) == 0,
				"errors":  errs,
			})
			if len(errs) > 0 {
				os.Exit(exitError)
			}
			return nil
		}

		if len(errs) == 0 {
			fmt.Println("✅ rules.yaml is valid")
			return nil
		}
		fmt.Fprintln(output.Stderr(), "❌ rules.yaml validation failed:")
		for _, e := range errs {
			fmt.Fprintln(output.Stderr(), "  - "+e)
		}
		os.Exit(exitError)
		return nil
	},
}

// ── helpers ──────────────────────────────────────────────────────────

func ruleDescription(r rules.Rule) string {
	if r == nil {
		return "(not registered in Registry)"
	}
	return r.Description()
}

func ruleDefaultSeverity(r rules.Rule) string {
	if r == nil {
		return "unknown"
	}
	return r.DefaultSeverity().String()
}

func chainEntriesToJSON(chain []rules.ChainEntry) []map[string]any {
	out := make([]map[string]any, 0, len(chain))
	for _, c := range chain {
		out = append(out, map[string]any{
			"layer":    c.Layer,
			"source":   c.Source,
			"severity": c.Severity.String(),
		})
	}
	return out
}

func filterByPolicy(cfg *rules.EngineConfig, eff map[string]*rules.EffectiveRule, policyName string) map[string]*rules.EffectiveRule {
	for _, p := range cfg.Policies {
		if p.Name != policyName {
			continue
		}
		filtered := make(map[string]*rules.EffectiveRule)
		for id := range p.SeverityOverrides {
			if r, ok := eff[id]; ok {
				filtered[id] = r
			}
		}
		return filtered
	}
	return map[string]*rules.EffectiveRule{}
}

func loadConfigFromURI(uri string) (*rules.EngineConfig, error) {
	if strings.HasPrefix(uri, "preset://") {
		return &rules.EngineConfig{Extends: []string{uri}}, nil
	}
	// File path.
	data, err := os.ReadFile(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to load diff target: %w", err)
	}
	out, err := app.RulesLoadConfigBytes(data)
	if err != nil {
		return nil, err
	}
	cfg, _ := out.(*rules.EngineConfig)
	return cfg, nil
}

func printDiff(cur, other map[string]*rules.EffectiveRule, otherURI string) error {
	type diffEntry struct {
		ID      string
		Current string
		Other   string
		Changed string
	}
	var diffs []diffEntry

	allIDs := make(map[string]bool)
	for id := range cur {
		allIDs[id] = true
	}
	for id := range other {
		allIDs[id] = true
	}
	ids := make([]string, 0, len(allIDs))
	for id := range allIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		curSev, otherSev := "absent", "absent"
		if r, ok := cur[id]; ok {
			curSev = r.Severity.String()
		}
		if r, ok := other[id]; ok {
			otherSev = r.Severity.String()
		}
		if curSev == otherSev {
			continue
		}
		diffs = append(diffs, diffEntry{ID: id, Current: curSev, Other: otherSev, Changed: curSev + " → " + otherSev})
	}

	if outputFormat == "json" {
		out := make([]map[string]any, 0, len(diffs))
		for _, d := range diffs {
			out = append(out, map[string]any{"id": d.ID, "current": d.Current, "other": d.Other})
		}
		Out.Print(map[string]any{
			"version":    rulesSchemaVersion,
			"diff_with":  otherURI,
			"diff_count": len(diffs),
			"diffs":      out,
		})
		return nil
	}

	if len(diffs) == 0 {
		fmt.Println("✅ no differences")
		return nil
	}
	fmt.Printf("diff with %s (%d entries):\n", otherURI, len(diffs))
	for _, d := range diffs {
		fmt.Printf("  %-50s  %s\n", d.ID, d.Changed)
	}
	return nil
}

func buildEffectiveJSON(cfg *rules.EngineConfig, eff map[string]*rules.EffectiveRule) map[string]any {
	rulesList := make([]map[string]any, 0, len(eff))
	ids := make([]string, 0, len(eff))
	for id := range eff {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		r := eff[id]
		rulesList = append(rulesList, map[string]any{
			"id":       id,
			"severity": r.Severity.String(),
			"origin": map[string]any{
				"layer":  r.Origin.Layer,
				"source": r.Origin.Source,
			},
			"previous_chain": chainEntriesToJSON(r.PreviousChain),
		})
	}
	policyNames := make([]string, 0, len(cfg.Policies))
	for _, p := range cfg.Policies {
		policyNames = append(policyNames, p.Name)
	}
	return map[string]any{
		"version":       rulesSchemaVersion,
		"config_source": configFileName(),
		"extends_chain": cfg.Extends,
		"policies":      policyNames,
		"rules":         rulesList,
		"summary": map[string]any{
			"total": len(eff),
		},
	}
}

func configFileName() string {
	return ".hstl/rules.yaml"
}

func printEffectiveText(eff map[string]*rules.EffectiveRule) {
	// Group by category (the first ID segment).
	byCategory := make(map[string][]*rules.EffectiveRule)
	for _, r := range eff {
		cat := strings.SplitN(r.RuleID, ".", 2)[0]
		byCategory[cat] = append(byCategory[cat], r)
	}
	cats := make([]string, 0, len(byCategory))
	for c := range byCategory {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	for _, cat := range cats {
		fmt.Printf("\n[%s]\n", cat)
		list := byCategory[cat]
		sort.Slice(list, func(i, j int) bool { return list[i].RuleID < list[j].RuleID })
		for _, r := range list {
			fmt.Printf("  %s %-55s (%s.%s)\n",
				severityIcon(r.Severity), r.RuleID, r.Origin.Layer, r.Origin.Source)
		}
	}
	fmt.Printf("\n%d rule(s)\n", len(eff))
}

func severityIcon(s rules.Severity) string {
	switch s {
	case rules.SeverityAdvisory:
		return rulesIconAdvisory
	case rules.SeverityWarn:
		return rulesIconWarn
	case rules.SeverityBlock:
		return rulesIconBlock
	case rules.SeverityHardBlock:
		return rulesIconHardBlock
	default:
		return rulesIconOff
	}
}

// matchGlobValidate calls filepath.Match as a dry-run to validate the
// pattern only.
func matchGlobValidate(pattern string) (bool, error) {
	_, err := filepathMatch(pattern, "dummy")
	return err == nil, err
}

func filepathMatch(pattern, name string) (bool, error) {
	// wrapper: separated for test-only DI.
	return filepathMatchImpl(pattern, name)
}

// filepathMatchImpl is the actual filepath.Match call; tests can
// override it.
var filepathMatchImpl = func(pattern, name string) (bool, error) {
	return defaultFilepathMatch(pattern, name)
}

func init() {
	rulesListCmd.Flags().StringVar(&rulesListCategory, "category", "", "category filter (task|sprint|commit|...)")
	rulesListCmd.Flags().BoolVar(&rulesListEnabled, "enabled", false, "only rules whose current severity is not off")

	rulesEffectiveCmd.Flags().StringVar(&rulesEffectivePolicy, "policy", "", "filter by a specific policy")
	rulesEffectiveCmd.Flags().StringVar(&rulesEffectiveDiff, "diff", "", "print severity differences against another preset/config")
	rulesEffectiveCmd.Flags().StringVar(&rulesEffectiveTarget, "target", "", "only rules applied via path-glob match")

	rulesCmd.AddCommand(rulesListCmd)
	rulesCmd.AddCommand(rulesExplainCmd)
	rulesCmd.AddCommand(rulesEffectiveCmd)
	rulesCmd.AddCommand(rulesValidateCmd)
	rootCmd.AddCommand(rulesCmd)
}
