package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
)

// ---------------------------------------------------------------------------
// hstl manifest validate <file>
// ---------------------------------------------------------------------------
//
// Compares an external manifest (YAML/JSON) against the actual CLI manifest
// and returns the drift.
// Design rationale: docs/08-references/standards/manifest-schema-contract.md, sec. 7
//
// Four drift types:
//   missing_in_cli       - declared externally but absent from the CLI
//   missing_in_external  - present in the CLI but absent from the external file
//   flag_mismatch        - flag name / required attribute mismatch
//   enum_mismatch        - enum value mismatch (follow-up)
//
// Exit codes:
//   0 - 0 drift entries
//   3 - 1 or more drift entries (pre-commit rule block trigger)
//   2 - failed to parse external file
//
// Supports two external file shapes:
//   1. manifest dump     - same shape as `hstl manifest` output ({commands: [...]})
//   2. manifest profile  - subset declaration ({commands: [{name, required_flags, flags}]})

// ValidateDrift is a single drift entry in the validate result.
type ValidateDrift struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

// ValidateResult is the JSON shape returned by validate.
type ValidateResult struct {
	Status     string          `json:"status"` // "ok" | "drift"
	DriftCount int             `json:"drift_count"`
	Drifts     []ValidateDrift `json:"drifts"`
}

// ExternalManifest is the minimal external-file shape (only the commands array is required).
type ExternalManifest struct {
	Commands []ExternalCommand `json:"commands" yaml:"commands"`
}

// ExternalCommand is a single command entry inside an external manifest.
type ExternalCommand struct {
	Name          string   `json:"name" yaml:"name"`
	RequiredFlags []string `json:"required_flags" yaml:"required_flags"`
	Flags         []string `json:"flags" yaml:"flags"`
}

var validateFromSkill string

var manifestValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Detect drift between an external manifest and the actual CLI manifest",
	Long: `Take an external manifest file (YAML/JSON) or a skill name and return the
drift relative to the actual CLI manifest.

Four drift types:
  missing_in_cli       - declared externally but absent from the CLI
  missing_in_external  - present in the CLI but absent from the external file
  flag_mismatch        - flag name / required attribute mismatch
  enum_mismatch        - enum value mismatch

Exit codes:
  0  0 drift entries
  3  1 or more drift entries (pre-commit rule block trigger)
  2  failed to parse external file

Examples:
  hstl manifest validate docs/08-references/standards/command-first-mapping.yaml
  hstl manifest validate /tmp/snapshot.json
  hstl manifest validate --from-skill task-management
  hstl manifest validate --from-skill sprint-management

Standard contract: docs/08-references/standards/manifest-schema-contract.md, sec. 7`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var external *ExternalManifest

		// --from-skill and the file positional argument are mutually exclusive.
		if validateFromSkill != "" {
			if len(args) > 0 {
				Out.Error("cannot use --from-skill together with <file>", apperr.CategoryInvalidInput.String(), "use only one")
				os.Exit(exitError)
				return nil
			}
			profile, err := buildProfileFromSkill(validateFromSkill)
			if err != nil {
				Out.Error("failed to build skill profile: "+err.Error(), apperr.CategoryInvalidInput.String(), "")
				os.Exit(exitError)
				return nil
			}
			external = profile
		} else {
			if len(args) == 0 {
				Out.Error("either <file> or --from-skill is required", apperr.CategoryInvalidInput.String(), "hstl manifest validate <file> or --from-skill <name>")
				os.Exit(exitError)
				return nil
			}
			filePath := args[0]
			loaded, err := loadExternalManifest(filePath)
			if err != nil {
				Out.Error("failed to load external manifest: "+err.Error(), apperr.CategoryInvalidInput.String(), "Check the file path / format (YAML/JSON)")
				os.Exit(exitError)
				return nil
			}
			external = loaded
		}

		drifts := compareManifests(external)

		// In --from-skill mode the profile is a subset, so we ignore
		// missing_in_external (covering only part of the CLI is the intended state).
		if validateFromSkill != "" {
			filtered := drifts[:0]
			for _, d := range drifts {
				if d.Type == "missing_in_external" {
					continue
				}
				filtered = append(filtered, d)
			}
			drifts = filtered
		}

		result := ValidateResult{
			DriftCount: len(drifts),
			Drifts:     drifts,
		}
		if len(drifts) == 0 {
			result.Status = "ok"
		} else {
			result.Status = "drift"
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			Out.Error("failed to emit validate result: "+err.Error(), apperr.CategoryInternal.String(), "")
			os.Exit(exitError)
		}

		if len(drifts) > 0 {
			os.Exit(exitBlocked)
		}
		return nil
	},
}

func init() {
	manifestValidateCmd.Flags().StringVar(&validateFromSkill, "from-skill", "",
		"Take a skill name and auto-generate a profile from skillCommandMap before validating")
	manifestCmd.AddCommand(manifestValidateCmd)
}

// buildProfileFromSkill builds a Sprint profile from a skillCommandMap
// entry. SKILL.md frontmatter trigger_commands are loaded lazily.
// Iterates skillMap[skill] prefixes -> collects leaf commands starting with
// each prefix -> converts to a profile.
func buildProfileFromSkill(skill string) (*ExternalManifest, error) {
	skillMap := getSkillCommandMap()
	prefixes, ok := skillMap[skill]
	if !ok {
		names := make([]string, 0, len(skillMap))
		for k := range skillMap {
			names = append(names, k)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("unknown skill: %q - available: %s", skill, strings.Join(names, ", "))
	}

	leaves := collectLeafCommands(rootCmd)
	em := &ExternalManifest{}
	seen := make(map[string]bool)
	for _, leaf := range leaves {
		path := leaf.CommandPath()
		matched := false
		for _, pfx := range prefixes {
			if strings.HasPrefix(path, pfx) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		dotted := commandPathToDotted(path)
		if seen[dotted] {
			continue
		}
		seen[dotted] = true
		em.Commands = append(em.Commands, ExternalCommand{Name: dotted})
	}
	return em, nil
}

// loadExternalManifest parses a YAML or JSON file into an ExternalManifest.
// Detects format by extension first; on failure, sniffs the content (tries the other format).
func loadExternalManifest(filePath string) (*ExternalManifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var em ExternalManifest

	// Try by extension first.
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &em); err == nil {
			return &em, nil
		}
		// fallback to JSON
		if err := json.Unmarshal(data, &em); err != nil {
			return nil, fmt.Errorf("failed to parse YAML/JSON: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &em); err == nil {
			return &em, nil
		}
		// fallback to YAML
		if err := yaml.Unmarshal(data, &em); err != nil {
			return nil, fmt.Errorf("failed to parse JSON/YAML: %w", err)
		}
	default:
		// No extension - content sniff.
		if err := json.Unmarshal(data, &em); err == nil {
			return &em, nil
		}
		if err := yaml.Unmarshal(data, &em); err != nil {
			return nil, fmt.Errorf("failed to parse both JSON and YAML: %w", err)
		}
	}

	return &em, nil
}

// compareManifests compares the external manifest against the actual CLI
// manifest and returns a sorted drift list (reproducible diff).
func compareManifests(external *ExternalManifest) []ValidateDrift {
	var drifts []ValidateDrift

	// Collect the actual CLI manifest.
	leaves := collectLeafCommands(rootCmd)
	cliCommands := make(map[string]*cobra.Command)
	for _, leaf := range leaves {
		// Normalize to dotted-name form (e.g. "hstl task start" -> "task.start").
		name := commandPathToDotted(leaf.CommandPath())
		cliCommands[name] = leaf
	}

	externalCommands := make(map[string]*ExternalCommand)
	for i := range external.Commands {
		ec := &external.Commands[i]
		// Normalize external file names too - support both "hstl task start" and "task.start".
		externalCommands[normalizeExternalName(ec.Name)] = ec
	}

	// missing_in_cli: declared externally but absent from the CLI.
	for extName := range externalCommands {
		if _, ok := cliCommands[extName]; !ok {
			drifts = append(drifts, ValidateDrift{
				Type:   "missing_in_cli",
				Path:   extName,
				Detail: "declared in external manifest but absent from the CLI",
			})
		}
	}

	// missing_in_external: present in the CLI but absent from the external file.
	for cliName := range cliCommands {
		if _, ok := externalCommands[cliName]; !ok {
			drifts = append(drifts, ValidateDrift{
				Type:   "missing_in_external",
				Path:   cliName,
				Detail: "present in the CLI but absent from the external manifest",
			})
		}
	}

	// flag_mismatch: present on both sides but the flag list differs.
	for extName, ec := range externalCommands {
		cliCmd, ok := cliCommands[extName]
		if !ok {
			continue
		}
		flagDrifts := compareFlags(extName, ec, cliCmd)
		drifts = append(drifts, flagDrifts...)
	}

	// Sort (reproducible).
	sort.Slice(drifts, func(i, j int) bool {
		if drifts[i].Type != drifts[j].Type {
			return drifts[i].Type < drifts[j].Type
		}
		return drifts[i].Path < drifts[j].Path
	})

	return drifts
}

// commandPathToDotted converts "hstl task start" -> "task.start".
// Drops the root prefix and replaces spaces with dots.
func commandPathToDotted(path string) string {
	parts := strings.Fields(path)
	if len(parts) > 0 {
		parts = parts[1:] // drop the binary root
	}
	return strings.Join(parts, ".")
}

// normalizeExternalName normalizes a command name from the external file
// into dotted-name form. Both shapes are supported:
//
//	"hstl task start" -> "task.start" (raw CLI form)
//	"task.start"      -> "task.start" (already normalized)
func normalizeExternalName(name string) string {
	// If it contains a space, it is the CLI path form -> commandPathToDotted.
	if strings.Contains(name, " ") {
		return commandPathToDotted(name)
	}
	return name
}

// compareFlags compares the flag lists of an ExternalCommand and a cobra
// Command and returns drift entries. When the profile has empty flags /
// required_flags, skip (profile is incomplete).
func compareFlags(name string, ec *ExternalCommand, cliCmd *cobra.Command) []ValidateDrift {
	var drifts []ValidateDrift

	// Empty flags means the profile opted out of flag comparison.
	if len(ec.Flags) == 0 && len(ec.RequiredFlags) == 0 {
		return nil
	}

	// Collect CLI flags.
	cliFlags := make(map[string]bool)
	cliRequired := make(map[string]bool)
	cliCmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		cliFlags[f.Name] = true
		if f.Annotations != nil {
			if _, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
				cliRequired[f.Name] = true
			}
		}
	})

	// Detect missing flags (declared externally but absent from the CLI).
	for _, f := range ec.Flags {
		if !cliFlags[f] {
			drifts = append(drifts, ValidateDrift{
				Type:   "flag_mismatch",
				Path:   name + " flags[" + f + "]",
				Detail: "external flag '" + f + "' is absent from the CLI",
			})
		}
	}

	// required_flags mismatch.
	for _, f := range ec.RequiredFlags {
		if !cliRequired[f] {
			drifts = append(drifts, ValidateDrift{
				Type:   "flag_mismatch",
				Path:   name + " required[" + f + "]",
				Detail: "external required flag '" + f + "' is not required in the CLI",
			})
		}
	}

	return drifts
}
