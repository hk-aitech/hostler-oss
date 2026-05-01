package rules

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// embed-script regression — verifies that the embedded rule scripts are
// bundled into the binary and that writeEmbeddedScript writes them to
// a temporary file that bash can execute.

func TestEmbeddedRuleScriptsAllPresent(t *testing.T) {
	expected := []string{
		"scripts/check-branch-sync.sh",
		"scripts/check-version-sync.sh",
		"scripts/check-manifest-sync.sh",
		"scripts/check-harness-skill-drift.sh",
		"scripts/check-sprint-phase-drift.sh",
	}
	for _, key := range expected {
		body, ok := embeddedRuleScripts[key]
		if !ok {
			t.Errorf("missing embedded entry: %s", key)
			continue
		}
		if len(body) == 0 {
			t.Errorf("embedded body is empty: %s", key)
		}
		if !strings.HasPrefix(body, "#!/usr/bin/env bash") && !strings.HasPrefix(body, "#!/bin/bash") {
			t.Errorf("missing embedded shebang: %s — first 20 chars: %q", key, body[:min(20, len(body))])
		}
	}
}

func TestWriteEmbeddedScriptProducesExecutable(t *testing.T) {
	path, err := writeEmbeddedScript("scripts/check-branch-sync.sh")
	if err != nil {
		t.Fatalf("writeEmbeddedScript failed: %v", err)
	}
	defer os.Remove(path)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat temp file: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("temp file is not executable: mode=%v", info.Mode().Perm())
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp file: %v", err)
	}
	if !strings.Contains(string(body), "HSTL_BRANCH_SYNC") {
		t.Errorf("embed body damaged — missing HSTL_BRANCH_SYNC")
	}
}

func TestWriteEmbeddedScriptNotFound(t *testing.T) {
	_, err := writeEmbeddedScript("scripts/nonexistent.sh")
	if err == nil {
		t.Fatalf("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "embedded script not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestEmbeddedScriptRunsInArbitraryDir guards against a regression where
// running from a temporary directory of an external project caused exit
// 127 because the script was missing — we now embed the script and
// invoke it by absolute path.
func TestEmbeddedScriptRunsInArbitraryDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hstl-rule-exec-test-*")
	if err != nil {
		t.Fatalf("create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath, err := writeEmbeddedScript("scripts/check-branch-sync.sh")
	if err != nil {
		t.Fatalf("writeEmbeddedScript failed: %v", err)
	}
	defer os.Remove(scriptPath)

	cmd := exec.Command("bash", scriptPath, "--help")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "HSTL_BRANCH_SYNC=off")
	// In opt-out mode, exit 0 is expected.
	err = cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 127 {
			t.Fatalf("regression — exit 127 (command not found)")
		}
	}
}
