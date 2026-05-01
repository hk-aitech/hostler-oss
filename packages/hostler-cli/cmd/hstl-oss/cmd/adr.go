package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var adrCmd = &cobra.Command{
	Use:   "adr",
	Short: "ADR governance (validate)",
}

var adrValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the ADR frontmatter status field",
	Long: `Validates the frontmatter 'status' field in docs/02-architecture/adrs/ADR-*.md.

Allowed enum (lowercase):
  proposed | accepted | deprecated | superseded | rejected

Blocking conditions:
  - missing status field
  - value outside the enum
  - uppercase usage (Accepted, Proposed, etc. - lowercase enforced)`,
	RunE: runADRValidate,
}

func init() {
	adrCmd.AddCommand(adrValidateCmd)
	rootCmd.AddCommand(adrCmd)
}

type adrValidateIssue struct {
	Path    string `json:"path"`
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type adrValidateResult struct {
	Total   int                `json:"total"`
	Pass    int                `json:"pass"`
	Fail    int                `json:"fail"`
	Issues  []adrValidateIssue `json:"issues"`
	Summary map[string]int     `json:"status_distribution"`
}

var adrAllowedStatus = map[string]bool{
	"proposed":   true,
	"accepted":   true,
	"deprecated": true,
	"superseded": true,
	"rejected":   true,
}

func runADRValidate(cmd *cobra.Command, args []string) error {
	root := projectRootOrCwd()
	dir := filepath.Join(root, "docs", "02-architecture", "adrs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to access ADR directory %s: %w", dir, err)
	}

	res := adrValidateResult{Summary: map[string]int{}}
	var paths []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "ADR-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)
	res.Total = len(paths)

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			res.Issues = append(res.Issues, adrValidateIssue{Path: p, Field: "file", Value: "", Message: "read failed: " + err.Error()})
			res.Fail++
			continue
		}
		fm, ok := extractYAMLFrontmatter(data)
		if !ok {
			res.Issues = append(res.Issues, adrValidateIssue{Path: p, Field: "frontmatter", Value: "", Message: "frontmatter missing"})
			res.Fail++
			continue
		}
		var meta map[string]any
		if err := yaml.Unmarshal(fm, &meta); err != nil {
			res.Issues = append(res.Issues, adrValidateIssue{Path: p, Field: "frontmatter", Value: "", Message: "yaml parse failed: " + err.Error()})
			res.Fail++
			continue
		}
		statusVal, exists := meta["status"]
		if !exists {
			res.Issues = append(res.Issues, adrValidateIssue{Path: p, Field: "status", Value: "", Message: "status field missing"})
			res.Fail++
			continue
		}
		statusStr, isStr := statusVal.(string)
		if !isStr {
			res.Issues = append(res.Issues, adrValidateIssue{Path: p, Field: "status", Value: fmt.Sprintf("%v", statusVal), Message: "status is not a string"})
			res.Fail++
			continue
		}
		if statusStr != strings.ToLower(statusStr) {
			res.Issues = append(res.Issues, adrValidateIssue{Path: p, Field: "status", Value: statusStr, Message: "lowercase enum required"})
			res.Fail++
			continue
		}
		if !adrAllowedStatus[statusStr] {
			res.Issues = append(res.Issues, adrValidateIssue{Path: p, Field: "status", Value: statusStr, Message: "value outside allowed enum (proposed|accepted|deprecated|superseded|rejected)"})
			res.Fail++
			continue
		}
		res.Pass++
		res.Summary[statusStr]++
	}

	Out.Print(res)
	if res.Fail > 0 {
		Out.Error(fmt.Sprintf("ADR validate failed %d/%d", res.Fail, res.Total), "ADR_STATUS_INVALID", "Add a status field to each ADR frontmatter: proposed|accepted|deprecated|superseded|rejected")
		os.Exit(exitError)
	}
	return nil
}

func extractYAMLFrontmatter(data []byte) ([]byte, bool) {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		return nil, false
	}
	rest := s[4:]
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return nil, false
	}
	return []byte(rest[:idx]), true
}

func projectRootOrCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".hstl")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
		dir = parent
	}
}
