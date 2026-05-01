// Package cmd — CLI subcommand flag cross-check test.
//
// Declares the expected flag set per subcommand and uses cobra's
// Flags().Lookup() to confirm each flag is actually registered. Adding a new
// subcommand or removing an existing flag without updating this test will
// trip the assertions, blocking the kind of doc/code drift where a flag was
// described in documentation but never registered in code (silent failure).
package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// expectedFlags lists the expected flag names per subcommand. Update this map
// when adding or removing a flag — the failure message points to which kind
// of update is required.
var expectedFlags = map[string][]string{
	// hstl-oss config subcommand tree.
	"config check":          {"format", "target"},
	"config fix":            {"dry-run", "yes", "target"},
	"config show-effective": {"format", "filter", "field", "with-source"},
	// migrate-mode subcommand.
	"config migrate-mode": {"to", "dry-run", "keep-legacy-symlink"},

	// Deprecated alias — slated for removal; delete this entry along with the
	// subcommand when it goes away.
	"validate-config": {"print-effective", "format", "filter"},
}

// TestFlagCrossCheck verifies that every flag declared in expectedFlags is
// actually registered on its subcommand.
func TestFlagCrossCheck(t *testing.T) {
	for cmdPath, flags := range expectedFlags {
		t.Run(cmdPath, func(t *testing.T) {
			cmd := findSubCommand(rootCmd, cmdPath)
			if cmd == nil {
				t.Fatalf("subcommand %q not found — confirm it is registered on rootCmd", cmdPath)
			}
			for _, flagName := range flags {
				if cmd.Flags().Lookup(flagName) == nil {
					t.Errorf("subcommand %q is missing flag --%s (add it in code)", cmdPath, flagName)
				}
			}
		})
	}
}

// TestNoUnexpectedFlags warns when a non-global flag exists on a subcommand
// but is absent from expectedFlags — this forces the map to be kept current
// when new flags are added.
func TestNoUnexpectedFlags(t *testing.T) {
	for cmdPath, expected := range expectedFlags {
		t.Run(cmdPath, func(t *testing.T) {
			cmd := findSubCommand(rootCmd, cmdPath)
			if cmd == nil {
				t.Skipf("subcommand %q not found", cmdPath)
				return
			}
			expectedSet := make(map[string]bool, len(expected))
			for _, f := range expected {
				expectedSet[f] = true
			}
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				// Skip global persistent flags inherited from rootCmd.
				if rootCmd.PersistentFlags().Lookup(f.Name) != nil {
					return
				}
				if !expectedSet[f.Name] {
					t.Errorf("subcommand %q has unexpected flag --%s (add it to expectedFlags)", cmdPath, f.Name)
				}
			})
		})
	}
}

// findSubCommand walks a space-separated path like "config migrate" through
// the subcommand tree. A single name resolves against rootCmd's direct
// children.
func findSubCommand(root *cobra.Command, path string) *cobra.Command {
	parts := splitSpaces(path)
	current := root
	for _, name := range parts {
		found := false
		for _, child := range current.Commands() {
			if child.Name() == name {
				current = child
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	if current == root {
		return nil
	}
	return current
}

// splitSpaces splits on whitespace (a minimal stand-in for strings.Fields).
func splitSpaces(s string) []string {
	var out []string
	cur := ""
	for _, ch := range s {
		if ch == ' ' || ch == '\t' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
