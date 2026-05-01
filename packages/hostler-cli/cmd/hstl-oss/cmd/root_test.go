package cmd

import (
	"bytes"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/output"
)

func TestRootCmd_Execute(t *testing.T) {
	rootCmd.SetArgs([]string{})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root command failed to run: %v", err)
	}
}

// TestRootCmd_UnknownFlag_ReturnsError verifies that even with
// SilenceErrors=true, Execute() returns the error so main.go can take
// responsibility for printing it to stderr.
func TestRootCmd_UnknownFlag_ReturnsError(t *testing.T) {
	rootCmd.SetArgs([]string{"--definitely-unknown-flag"})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown flag - got nil")
	}
	// cobra returns an error whose message contains "unknown flag".
	if err.Error() == "" {
		t.Error("error message is empty - main.go has nothing to print")
	}
}

func TestRootCmd_HasVersionSubcommand(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("version subcommand not registered")
	}
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	// --json added.
	flags := []string{"output", "json", "quiet", "verbose", "dir", "no-color"}
	for _, name := range flags {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("global flag %q not registered", name)
		}
	}
}

// TestT533_JSONFlagAliasesOutputJSON verifies that --json behaves
// identically to -o json.
func TestT533_JSONFlagAliasesOutputJSON(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantFormat string
	}{
		{
			name:    "default (no flag)",
			args:    []string{},
			wantFormat: output.DefaultFormat, // json (AI-first)
		},
		{
			name:    "--output json",
			args:    []string{"--output", "json"},
			wantFormat: "json",
		},
		{
			name:    "-o json",
			args:    []string{"-o", "json"},
			wantFormat: "json",
		},
		{
			name:    "--json only",
			args:    []string{"--json"},
			wantFormat: "json",
		},
		{
			name:    "--json with -o text (--json wins)",
			args:    []string{"-o", "text", "--json"},
			wantFormat: "json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset flag state - default is json (ADR-038).
			outputFormat = output.DefaultFormat
			jsonShortcut = false

			// Parse only (no actual Execute).
			fs := rootCmd.PersistentFlags()
			if err := fs.Parse(tc.args); err != nil {
				t.Fatalf("flag parse failed: %v", err)
			}

			// Simulate the conversion logic from PersistentPreRun.
			if jsonShortcut {
				outputFormat = outputFormatJSON
			}

			if outputFormat != tc.wantFormat {
				t.Errorf("outputFormat = %q, want %q (args=%v)",
					outputFormat, tc.wantFormat, tc.args)
			}
		})
	}
}

func TestRootCmd_OutputFlagDefault(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("output")
	if f == nil {
		t.Fatal("output flag missing")
		return // nolint (SA5011 guard - reinforces staticcheck recognition of t.Fatal)
	}
	// Default is json (AI-first, per ADR-038).
	if f.DefValue != output.DefaultFormat {
		t.Errorf("expected output default %q, got %q", output.DefaultFormat, f.DefValue)
	}
}

func TestRootCmd_SilenceErrors(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Error("SilenceErrors is false")
	}
	if !rootCmd.SilenceUsage {
		t.Error("SilenceUsage is false")
	}
}
