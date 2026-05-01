package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// hstl manifest get <dotted-name> tests.

// TestT334_ResolveDottedName_Valid verifies that a valid dotted-name resolves
// to the exact leaf cobra Command.
func TestT334_ResolveDottedName_Valid(t *testing.T) {
	tests := []string{
		"task.start",
		"task.complete",
		"sprint.create",
		"sprint.complete",
		"harness.auto-check",
		"harness.get",
	}
	for _, dotted := range tests {
		cmd, err := resolveDottedName(dotted)
		if err != nil {
			t.Errorf("%q: unexpected err = %v", dotted, err)
			continue
		}
		if cmd == nil {
			t.Errorf("%q: cmd is nil", dotted)
			continue
		}
		// The last dotted segment must match cmd.Name().
		parts := strings.Split(dotted, ".")
		leafPart := parts[len(parts)-1]
		if cmd.Name() != leafPart {
			t.Errorf("%q: cmd.Name() = %q, want %q", dotted, cmd.Name(), leafPart)
		}
	}
}

// TestT334_ResolveDottedName_Invalid verifies that an invalid dotted-name
// returns an error.
func TestT334_ResolveDottedName_Invalid(t *testing.T) {
	tests := []struct {
		dotted string
		wantErr string
	}{
		{"task.nonexistent", "not found"},
		{"nonexistent.anything", "not found"},
		{"", "empty"},
		{"task.start.extra.too.deep", "exceeds 3 levels"},
		{"task", "group"}, // task is a branch, not a leaf
	}
	for _, tt := range tests {
		_, err := resolveDottedName(tt.dotted)
		if err == nil {
			t.Errorf("%q: expected error, got nil", tt.dotted)
			continue
		}
		if !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%q: err = %q, want substring %q", tt.dotted, err.Error(), tt.wantErr)
		}
	}
}

// TestT334_SuggestSimilarNames verifies that an unknown dotted-name yields
// suggestions of similar leaves from the same group.
func TestT334_SuggestSimilarNames(t *testing.T) {
	suggestions := suggestSimilarNames("task.nonexistent")
	if len(suggestions) == 0 {
		t.Errorf("no suggestions for task.nonexistent")
	}
	// Every suggestion must start with task..
	for _, s := range suggestions {
		if !strings.HasPrefix(s, "task.") {
			t.Errorf("suggestion %q does not start with task.", s)
		}
	}
	// Maximum 5 suggestions.
	if len(suggestions) > 5 {
		t.Errorf("suggestion count = %d, want <= 5", len(suggestions))
	}
}

// TestT334_IsManifestSubcommand verifies the manifest-series subcommand
// detection (the reason manifest / manifest.get / manifest.validate etc. must
// be bypassed by the persistent handler).
func TestT334_IsManifestSubcommand(t *testing.T) {
	// manifest itself is true.
	if !isManifestSubcommand(manifestCmd) {
		t.Errorf("manifestCmd is a manifest subcommand")
	}
	// manifest get is also true.
	if !isManifestSubcommand(manifestGetCmd) {
		t.Errorf("manifestGetCmd is a manifest subcommand")
	}
	// task is false.
	var taskCmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "task" {
			taskCmd = sub
			break
		}
	}
	if taskCmd != nil && isManifestSubcommand(taskCmd) {
		t.Errorf("taskCmd is not in the manifest series")
	}
}
