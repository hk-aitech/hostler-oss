package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rules run e2e coverage.
//
// The executeRules / parallelismForPhase helpers replaced the inline loop
// in runRulesRun. This test suite exercises that path through real cobra
// execution to guard against regressions.

// TestT587_RulesRun_CommitMsg_MsgFile verifies that the commit-msg phase
// reads --msg-file and forwards the contents to the
// commit.message.korean / commit.task_id.present rules. Covers the
// sequentialParallelism path plus the populateExtra branch.
func TestT587_RulesRun_CommitMsg_MsgFile(t *testing.T) {
	captured := WithTestOutput(t, "text")
	tmp := t.TempDir()
	msgFile := filepath.Join(tmp, "COMMIT_EDITMSG")
	// Sufficient Korean ratio + contains a Task ID -> both rules expected to pass.
	msg := "feat(T587): cmd/cli e2e test coverage expanded - sufficient body content\n"
	if err := os.WriteFile(msgFile, []byte(msg), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"rules", "run", "--phase", "commit-msg", "--msg-file", msgFile})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	// runRulesRun calls os.Exit(3) on block, so we only test the happy path here.
	// Rule output goes through cobra OutOrStderr directly while WithTestOutput
	// only captures the formatter path, so this test only checks that we exit
	// without error (the execution path traverses populateExtra + executeRules).
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("rules run commit-msg failed: %v\nstderr: %s", err, captured.Stderr.String())
	}
}

// TestT587_RulesRun_CommitMsg_MsgFile_Missing_Error - without --msg-file,
// populateExtra must return an error and fail the run. The error
// propagates because we are not in audit mode.
func TestT587_RulesRun_CommitMsg_MsgFile_Missing_Error(t *testing.T) {
	_ = WithTestOutput(t, "text")
	// Package-global flag values may persist across test runs. Set
	// --msg-file= to an empty value explicitly to force the empty-string
	// branch in populateExtra.
	rootCmd.SetArgs([]string{"rules", "run", "--phase", "commit-msg", "--msg-file="})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when --msg-file is missing")
	}
	if !strings.Contains(err.Error(), "msg-file") {
		t.Errorf("expected error message to mention msg-file: %v", err)
	}
}

// TestT587_RulesRun_PhaseAll_Audit_JSON verifies that --phase all --audit
// --report produces a JSON report and that the writeAuditReport schema is
// valid. Covers writeAuditReport plus the best-effort registry walk.
//
// Note: `--phase all` also runs the precommit Rules, and
// `precommit.go.test_short` invokes `go test ./... -short` internally; if
// the test suite enters that path, it will recursively re-run itself - a
// fork bomb. To prevent that, t.Chdir into an empty TempDir without staged
// files so `precommitStagedFiles` returns empty, leaving every precommit
// Rule in Skipped status. The registry walk and writeAuditReport path are
// still exercised.
func TestT587_RulesRun_PhaseAll_Audit_JSON(t *testing.T) {
	captured := WithTestOutput(t, "text")
	tmp := t.TempDir()
	// Recursion guard: cwd into an empty non-git directory. The precommit
	// Rule shouldRun calls git diff --cached --name-only to list staged
	// files; without a git repo the result is empty -> everything Skipped.
	t.Chdir(tmp)

	reportPath := filepath.Join(tmp, "audit.json")
	msgFile := filepath.Join(tmp, "msg")
	if err := os.WriteFile(msgFile, []byte("feat(T587): audit mode coverage expanded\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{
		"rules", "run",
		"--phase", "all",
		"--audit",
		"--msg-file", msgFile,
		"--report", reportPath,
	})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rules run phase=all audit failed: %v\nstderr: %s", err, captured.Stderr.String())
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("failed to read audit report: %v", err)
	}
	var rep auditReport
	if err := json.Unmarshal(data, &rep); err != nil {
		t.Fatalf("failed to parse audit report JSON: %v", err)
	}
	if rep.Version != "1" {
		t.Errorf("audit report version = %q, want %q", rep.Version, "1")
	}
	if rep.Summary.Total == 0 {
		t.Error("audit report summary.total = 0 (registry walk may not be running)")
	}
}

// TestT587_RulesRun_InvalidPhase_Error - resolvePhases must return an
// error for an unsupported phase string.
func TestT587_RulesRun_InvalidPhase_Error(t *testing.T) {
	_ = WithTestOutput(t, "text")
	rootCmd.SetArgs([]string{"rules", "run", "--phase", "invalid-xyz"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid phase")
	}
	if !strings.Contains(err.Error(), "phase") {
		t.Errorf("expected error message to mention phase: %v", err)
	}
}

// TestT587_RulesValidate_E2E - the rules validate subcommand verifies the
// consistency of the current project's rules.yaml (or built-in
// defaults). Covers the no-error happy path.
func TestT587_RulesValidate_E2E(t *testing.T) {
	captured := WithTestOutput(t, "text")
	rootCmd.SetArgs([]string{"rules", "validate"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	// validate should exit 0 with no error on success.
	err := rootCmd.Execute()
	if err != nil {
		// Fail if a healthy project produces an error.
		t.Fatalf("rules validate failed: %v\nstderr: %s", err, captured.Stderr.String())
	}
}

// TestT587_RulesExplain_E2E - rules explain <rule_id> must print the
// cascade origin chain for the given Rule.
func TestT587_RulesExplain_E2E(t *testing.T) {
	captured := WithTestOutput(t, "json")
	rootCmd.SetArgs([]string{"rules", "explain", "commit.message.korean", "-o", "json"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rules explain failed: %v\nstderr: %s", err, captured.Stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(captured.Stdout.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse rules explain JSON: %v\nraw: %q", err, firstN(captured.Stdout.String(), 200))
	}
	// envelope wrap - payload lives under .data.
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("envelope data field missing: %v", result)
	}
	if _, ok := data["id"]; !ok {
		t.Errorf("data.id key missing: %v", data)
	}
	if _, ok := data["default_severity"]; !ok {
		t.Errorf("data.default_severity key missing: %v", data)
	}
}
