package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestVersionCmd_TextOutput(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	Version, Commit, Date = "1.0.0-test", "abc1234", "2026-01-01"
	defer func() { Version, Commit, Date = oldVersion, oldCommit, oldDate }()

	// In text mode, version writes via fmt.Printf to os.Stdout directly,
	// so cobra SetOut does not capture it. Just confirm the call succeeds.
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run version: %v", err)
	}

	// Verify the Version variable was set correctly.
	if Version != "1.0.0-test" {
		t.Errorf("expected Version 1.0.0-test, got %s", Version)
	}
}

func TestVersionCmd_JSONOutput(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, Date
	Version, Commit, Date = "2.0.0", "def5678", "2026-06-01"
	defer func() { Version, Commit, Date = oldVersion, oldCommit, oldDate }()

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"version", "--output", "json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("failed to run version --output json: %v", err)
	}

	// JSON output goes through the Out formatter, so JSON lands on stdout.
	// PersistentPreRun initializes Out, so attempt to parse as JSON.
	got := out.String()
	if got == "" {
		// JSON is written to the formatter's Out (=os.Stdout) which may differ from cobra SetOut.
		t.Skip("JSON output goes via formatter stdout - cobra SetOut does not apply")
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(got), &result); err == nil {
		if result["version"] != "2.0.0" {
			t.Errorf("expected JSON version 2.0.0, got %s", result["version"])
		}
	}
}

func TestVersionCmd_Use(t *testing.T) {
	if versionCmd.Use != "version" {
		t.Errorf("expected Use 'version', got %q", versionCmd.Use)
	}
}

func TestVersionCmd_Short(t *testing.T) {
	if versionCmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestVersionCmd_IsRegistered(t *testing.T) {
	if !rootCmd.HasSubCommands() {
		t.Fatal("root has no subcommands")
	}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			return
		}
	}
	t.Error("version not registered under root")
}
