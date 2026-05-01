package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
)

// hstl version bump subcommand.
//
// Atomically updates the version in plugin.json, marketplace.json, and CLAUDE.md
// (3 files) plus prepends a CHANGELOG.md entry skeleton. plugin.json is the SSOT.

const (
	versionBumpPluginJSONRel      = ".claude-plugin/plugin.json"
	versionBumpMarketplaceJSONRel = ".claude-plugin/marketplace.json"
	versionBumpClaudeMDRel        = "CLAUDE.md"
	versionBumpChangelogRel       = "CHANGELOG.md"
	versionBumpPluginName         = brand.ProductName
	versionBumpDateFormat         = "2006-01-02"
)

var (
	versionBumpMajor  bool
	versionBumpMinor  bool
	versionBumpPatch  bool
	versionBumpDryRun bool
)

var versionBumpCmd = &cobra.Command{
	Use:   "bump",
	Short: "Atomically bump plugin.json/marketplace.json/CLAUDE.md versions and prepend a CHANGELOG entry skeleton",
	Long: `Bump the hstl plugin version using semver rules.

SSOT is .claude-plugin/plugin.json. The three files (plus CHANGELOG) are updated
atomically so scripts/check-version-sync.sh keeps passing.

Examples:
  hstl version bump --minor         # 3.3.0 -> 3.4.0
  hstl version bump --patch         # 3.3.0 -> 3.3.1
  hstl version bump --major         # 3.3.0 -> 4.0.0
  hstl version bump --minor --dry-run
`,
	RunE: runVersionBump,
}

func init() {
	versionBumpCmd.Flags().BoolVar(&versionBumpMajor, "major", false, "Increment major version (X.0.0)")
	versionBumpCmd.Flags().BoolVar(&versionBumpMinor, "minor", false, "Increment minor version (x.Y.0)")
	versionBumpCmd.Flags().BoolVar(&versionBumpPatch, "patch", false, "Increment patch version (x.y.Z)")
	versionBumpCmd.Flags().BoolVar(&versionBumpDryRun, "dry-run", false, "Print the plan without modifying files")
	versionCmd.AddCommand(versionBumpCmd)
}

type versionBumpResult struct {
	OldVersion string   `json:"old_version"`
	NewVersion string   `json:"new_version"`
	BumpType   string   `json:"bump_type"`
	DryRun     bool     `json:"dry_run"`
	Updated    []string `json:"updated_files"`
}

func runVersionBump(cmd *cobra.Command, args []string) error {
	bumpType, err := versionBumpSelectType()
	if err != nil {
		return err
	}

	root := app.ProjectRoot()

	pluginPath := filepath.Join(root, versionBumpPluginJSONRel)
	oldVer, err := readPluginJSONVersion(pluginPath)
	if err != nil {
		return err
	}

	newVer, err := bumpSemver(oldVer, bumpType)
	if err != nil {
		return err
	}

	result := versionBumpResult{
		OldVersion: oldVer,
		NewVersion: newVer,
		BumpType:   bumpType,
		DryRun:     versionBumpDryRun,
		Updated: []string{
			versionBumpPluginJSONRel,
			versionBumpMarketplaceJSONRel,
			versionBumpClaudeMDRel,
			versionBumpChangelogRel,
		},
	}

	if versionBumpDryRun {
		emitVersionBumpResult(result)
		return nil
	}

	if err := writePluginJSONVersion(pluginPath, newVer); err != nil {
		return err
	}
	if err := writeMarketplaceJSONVersion(filepath.Join(root, versionBumpMarketplaceJSONRel), newVer); err != nil {
		return err
	}
	if err := writeClaudeMDVersion(filepath.Join(root, versionBumpClaudeMDRel), newVer); err != nil {
		return err
	}
	if err := prependChangelogEntry(filepath.Join(root, versionBumpChangelogRel), newVer); err != nil {
		return err
	}

	emitVersionBumpResult(result)
	return nil
}

func versionBumpSelectType() (string, error) {
	count := 0
	picked := ""
	if versionBumpMajor {
		count++
		picked = "major"
	}
	if versionBumpMinor {
		count++
		picked = "minor"
	}
	if versionBumpPatch {
		count++
		picked = "patch"
	}
	if count == 0 {
		return "", fmt.Errorf("flag required: one of --major / --minor / --patch")
	}
	if count > 1 {
		return "", fmt.Errorf("--major / --minor / --patch cannot be combined")
	}
	return picked, nil
}

func bumpSemver(old, bumpType string) (string, error) {
	parts := strings.Split(old, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("not a semver string: %q", old)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return "", fmt.Errorf("failed to parse semver numbers: %q", old)
		}
		nums[i] = n
	}
	switch bumpType {
	case "major":
		nums[0]++
		nums[1] = 0
		nums[2] = 0
	case "minor":
		nums[1]++
		nums[2] = 0
	case "patch":
		nums[2]++
	}
	return fmt.Sprintf("%d.%d.%d", nums[0], nums[1], nums[2]), nil
}

func readPluginJSONVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", fmt.Errorf("failed to parse %s JSON: %w", path, err)
	}
	v, ok := obj["version"].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("%s has no .version field", path)
	}
	return v, nil
}

func writePluginJSONVersion(path, newVer string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Replace only the version field on a per-line basis (preserves formatting).
	re := regexp.MustCompile(`("version"\s*:\s*)"[^"]+"`)
	updated := re.ReplaceAll(data, []byte(`${1}"`+newVer+`"`))
	return os.WriteFile(path, updated, 0o644)
}

func writeMarketplaceJSONVersion(path, newVer string) error {
	// Load JSON, update the hstl entry's version, then re-serialize.
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("failed to parse marketplace.json: %w", err)
	}
	plugins, ok := obj["plugins"].([]any)
	if !ok {
		return fmt.Errorf("marketplace.json has no .plugins array")
	}
	found := false
	for _, p := range plugins {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if name, _ := pm["name"].(string); name == versionBumpPluginName {
			pm["version"] = newVer
			found = true
		}
	}
	if !found {
		return fmt.Errorf("marketplace.json has no plugin entry %q", versionBumpPluginName)
	}
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func writeClaudeMDVersion(path, newVer string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`(\*\*Version\*\*:\s*)[0-9]+\.[0-9]+\.[0-9]+(\s*\([^)]*\))?`)
	today := time.Now().Format(versionBumpDateFormat)
	replacement := fmt.Sprintf(`${1}%s (%s)`, newVer, today)
	updated := re.ReplaceAll(data, []byte(replacement))
	return os.WriteFile(path, updated, 0o644)
}

func prependChangelogEntry(path, newVer string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	today := time.Now().Format(versionBumpDateFormat)
	header := "# Changelog\n"
	body := string(data)
	if !strings.HasPrefix(body, header) {
		return fmt.Errorf("CHANGELOG.md does not start with header %q", strings.TrimSpace(header))
	}
	rest := strings.TrimPrefix(body, header)
	entry := fmt.Sprintf("\n## v%s (%s)\n\n### Added\n\n- TODO: fill in this entry manually.\n\n", newVer, today)
	out := header + entry + strings.TrimLeft(rest, "\n")
	return os.WriteFile(path, []byte(out), 0o644)
}

func emitVersionBumpResult(r versionBumpResult) {
	if outputFormat == "json" {
		Out.Print(r)
		return
	}
	mode := ""
	if r.DryRun {
		mode = " (dry-run)"
	}
	fmt.Printf("version bump%s: %s -> %s (%s)\n", mode, r.OldVersion, r.NewVersion, r.BumpType)
	for _, f := range r.Updated {
		fmt.Printf("  - %s\n", f)
	}
}
