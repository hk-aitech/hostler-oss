package cmd

// e2e test shared fixture plus task/sprint/harness suites.
//
// Design:
//   - each test uses an isolated project root from t.TempDir()
//   - git init + minimal .hstl/ directory + empty go.mod so the CLI sees a project
//   - environment variables HSTL_PROJECT_ROOT + HSTL_DB_PATH provide isolation
//   - calls the bin/hstl-oss subprocess (reuses runHstl)
//
// Follows the recursion-guard pattern (t.Chdir + isolated DB) used in rules_run_e2e.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// NewTempProject creates an isolated project root and sets the
// HSTL_PROJECT_ROOT + HSTL_DB_PATH environment variables. Returns the
// created project root path.
func NewTempProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// git init (audit log + some CLI commands assume git is present).
	cmd := exec.Command("git", "init", "-q", root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	// Minimal git identity (we never commit, but some CLI commands look it up).
	_ = exec.Command("git", "-C", root, "config", "user.email", "t606@test.local").Run()
	_ = exec.Command("git", "-C", root, "config", "user.name", "T606").Run()

	// Empty go.mod (so platform.kind auto-detects).
	_ = os.WriteFile(filepath.Join(root, "go.mod"), []byte("module e2etest\n\ngo 1.21\n"), 0o644)

	// .hstl/ directory.
	_ = os.MkdirAll(filepath.Join(root, ".hstl"), 0o755)

	// Environment isolation.
	t.Setenv("HSTL_PROJECT_ROOT", root)
	t.Setenv("HSTL_DB_PATH", filepath.Join(root, ".hstl", "hstl.db"))
	// Disable the summary heuristic (e2e validates CLI paths, not summary semantics).
	t.Setenv("HSTL_SUMMARY_POLICY", "off")
	// Disable the LLM re-judge after the subagent heuristic fails.
	t.Setenv("HSTL_SUMMARY_SUBAGENT", "off")

	return root
}

// hstlJSON invokes hstl as a subprocess, parses stdout as JSON, and
// returns the map. Attempts JSON parsing even when exitCode != 0 (error
// responses are also JSON).
func hstlJSON(t *testing.T, args ...string) (map[string]any, int) {
	t.Helper()
	if hstlBin == "" {
		t.Fatal("hstlBin not set")
	}
	fullArgs := append([]string{}, args...)
	if !contains(fullArgs, "-o") && !contains(fullArgs, "--output") {
		fullArgs = append(fullArgs, "-o", "json")
	}
	cmd := exec.Command(hstlBin, fullArgs...)
	out, err := cmd.CombinedOutput()
	exit := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exit = exitErr.ExitCode()
		} else {
			t.Fatalf("exec failed: %v\n%s", err, out)
		}
	}
	// Tolerate noise around the JSON (e.g. text-mode skip prompts) by extracting from the first `{` to the last `}`.
	text := string(out)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return nil, exit
	}
	var result map[string]any
	if jerr := json.Unmarshal([]byte(text[start:end+1]), &result); jerr != nil {
		t.Logf("failed to parse JSON (exit=%d): %v\nraw: %s", exit, jerr, text)
		return nil, exit
	}
	return result, exit
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// ===== Task series E2E =====

func TestT606_E2E_Task_Create_Missing_Title(t *testing.T) {
	skipIfNoBinary(t)
	NewTempProject(t)

	// Missing --title - cobra Required flag validation must fail it.
	_, exit := hstlJSON(t, "task", "create",
		"--summary", "verify create path without title",
		"--type", "feature",
	)
	if exit == 0 {
		t.Error("create succeeded without --title - expected an error")
	}
}

func TestT606_E2E_Task_Create_Get_RoundTrip(t *testing.T) {
	skipIfNoBinary(t)
	NewTempProject(t)

	// create
	createOut, exit := hstlJSON(t, "task", "create",
		"--title", "T606 round-trip test",
		"--summary", "round-trip test verifying that get returns the same data after create",
		"--type", "test",
		"--estimate", "S",
	)
	if exit != 0 || createOut == nil {
		t.Fatalf("task create failed: exit=%d", exit)
	}
	data, _ := createOut["data"].(map[string]any)
	taskID, _ := data["task_id"].(string)
	if taskID == "" {
		t.Fatalf("task_id not returned: %v", createOut)
	}

	// get
	getOut, exit := hstlJSON(t, "task", "get", taskID)
	if exit != 0 {
		t.Fatalf("task get failed: exit=%d", exit)
	}
	title, _ := getOut["title"].(string)
	if !strings.Contains(title, "T606 round-trip") {
		t.Errorf("round-trip title mismatch: %q", title)
	}
}

// ===== Sprint series E2E =====

func TestT606_E2E_Sprint_Create_List(t *testing.T) {
	skipIfNoBinary(t)
	NewTempProject(t)

	// sprint create
	_, exit := hstlJSON(t, "sprint", "create",
		"--id", "sprint-t606",
		"--title", "T606 e2e sprint",
		"--goal", "verify e2e sprint creation",
	)
	if exit != 0 {
		t.Fatalf("sprint create failed: exit=%d", exit)
	}

	// Confirm sprint-t606 appears in sprint list.
	listOut, exit := hstlJSON(t, "sprint", "list")
	if exit != 0 {
		t.Fatalf("sprint list failed: exit=%d", exit)
	}
	// Response shape varies between implementations - match by substring.
	raw, _ := json.Marshal(listOut)
	if !strings.Contains(string(raw), "sprint-t606") {
		t.Errorf("sprint-t606 missing from sprint list:\n%s", raw)
	}
}

// ===== Harness series E2E =====

func TestT606_E2E_Harness_Get_Empty(t *testing.T) {
	skipIfNoBinary(t)
	NewTempProject(t)

	// Create the Task.
	createOut, _ := hstlJSON(t, "task", "create",
		"--title", "T606 harness test",
		"--summary", "verify the initial harness get state - items created and unchecked",
		"--type", "chore",
	)
	data, _ := createOut["data"].(map[string]any)
	taskID, _ := data["task_id"].(string)
	if taskID == "" {
		t.Fatal("missing task_id")
	}

	// harness get - the chore type templates only criteria_checked.
	getOut, exit := hstlJSON(t, "harness", "get", "task", taskID)
	if exit != 0 {
		t.Fatalf("harness get failed: exit=%d", exit)
	}
	getData, _ := getOut["data"].(map[string]any)
	items, _ := getData["items"].([]any)
	if len(items) == 0 {
		t.Errorf("harness items empty — expected at least one (criteria_checked); got envelope:\n%v", getOut)
	}
}

// ===== CLI help regression - verify new subcommands are registered =====

func TestT606_E2E_Help_NewSubcommands(t *testing.T) {
	skipIfNoBinary(t)
	cases := []struct {
		args    []string
		keyword string
	}{
		{[]string{"task", "--help"}, "checkpoint"},
		{[]string{"task", "complete", "--help"}, "auto-check-criteria"},
		{[]string{"task", "complete", "--help"}, "interactive"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("help_%s_%s", tc.args[0], tc.keyword), func(t *testing.T) {
			cmd := exec.Command(hstlBin, tc.args...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("help failed: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), tc.keyword) {
				t.Errorf("%v output missing %q", tc.args, tc.keyword)
			}
		})
	}
}
