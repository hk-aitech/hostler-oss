package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/gatejudgement"
)

// hstlBin is the path to the test CLI binary.
var hstlBin string

func TestMain(m *testing.M) {
	// Prevent gatejudgement subdirectory pollution - block .hostler/ creation under pkg cwd.
	// Tests that exercise the store itself (e.g. gate_log_test) call SetGlobalStore
	// directly (store.Record), so SetDisabled has no effect on them.
	gatejudgement.SetDisabled(true)

	// Resolve the binary path from env var or the project bin/.
	hstlBin = os.Getenv("HSTL_BIN")
	if hstlBin == "" {
		// Search for bin/hstl-oss under the project root.
		wd, _ := os.Getwd()
		for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "bin", brand.BinaryName)
			if _, err := os.Stat(candidate); err == nil {
				hstlBin = candidate
				break
			}
		}
	}
	os.Exit(m.Run())
}

func skipIfNoBinary(t *testing.T) {
	t.Helper()
	if hstlBin == "" {
		t.Skip("hstl binary missing - run make build-all first")
	}
}

func runHstl(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(hstlBin, args...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec failed: %v", err)
		}
	}
	return string(out), exitCode
}

// ===== --help tests (every command) =====

func TestHelp_Root(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	for _, keyword := range []string{"task", "sprint", "kb", "audit", "registry", "version"} {
		if !strings.Contains(out, keyword) {
			t.Errorf("--help missing %q", keyword)
		}
	}
}

func TestHelp_Task(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "task", "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	for _, sub := range []string{"list", "get", "create", "start", "complete", "next", "delete", "reopen"} {
		if !strings.Contains(out, sub) {
			t.Errorf("task --help missing subcommand %q", sub)
		}
	}
}

func TestHelp_Sprint(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "sprint", "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	for _, sub := range []string{"list", "create", "start", "complete", "progress", "update"} {
		if !strings.Contains(out, sub) {
			t.Errorf("sprint --help missing subcommand %q", sub)
		}
	}
}

func TestHelp_KB(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "kb", "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	for _, sub := range []string{"list", "create", "rebuild"} {
		if !strings.Contains(out, sub) {
			t.Errorf("kb --help missing subcommand %q", sub)
		}
	}
}

func TestHelp_Registry(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "registry", "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	for _, sub := range []string{"list", "allocate"} {
		if !strings.Contains(out, sub) {
			t.Errorf("registry --help missing subcommand %q", sub)
		}
	}
}

func TestHelp_Harness(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "harness", "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	for _, sub := range []string{"get", "check", "check-all"} {
		if !strings.Contains(out, sub) {
			t.Errorf("harness --help missing subcommand %q", sub)
		}
	}
}

func TestHelp_Backlog(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "backlog", "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out, "sync") {
		t.Error("backlog --help missing the sync subcommand")
	}
}

// ===== Global flag tests =====

func TestGlobalFlag_Output(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "--help")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(out, "--output") {
		t.Error("--output flag missing")
	}
	if !strings.Contains(out, "--quiet") {
		t.Error("--quiet flag missing")
	}
	if !strings.Contains(out, "--verbose") {
		t.Error("--verbose flag missing")
	}
	if !strings.Contains(out, "--no-color") {
		t.Error("--no-color flag missing")
	}
}

// ===== version tests =====

func TestVersion_ExitZero(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "version")
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	// `version` emits a JSON envelope: {"data":{"commit":"…","date":"…","version":"…"},"status":"ok"}.
	for _, key := range []string{`"version"`, `"commit"`, `"date"`, `"status"`} {
		if !strings.Contains(out, key) {
			t.Errorf("version output missing key %s: %s", key, out)
		}
	}
}

// ===== exit code contract tests =====

func TestExitCode_UnknownCommand(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "nonexistent-command")
	// cobra returns an error for unknown commands.
	if code == 0 {
		t.Error("unknown command returned exit 0")
	}
}

func TestExitCode_TaskGetNotFound(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "task", "get", "T999999")
	if code == 0 {
		t.Error("missing Task returned exit 0")
	}
	if code != 1 {
		t.Logf("exit code: %d (expected 1 for NOT_FOUND)", code)
	}
}

// ===== Subcommand missing-arg tests =====

func TestTaskGet_NoArgs(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "task", "get")
	if code == 0 {
		t.Error("task get succeeded with no args - violates ExactArgs(1)")
	}
}

func TestTaskStart_NoArgs(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "task", "start")
	if code == 0 {
		t.Error("task start succeeded with no args - violates ExactArgs(1)")
	}
}

func TestTaskComplete_NoArgs(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "task", "complete")
	if code == 0 {
		t.Error("task complete succeeded with no args - violates ExactArgs(1)")
	}
}

func TestTaskDelete_NoReason(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "task", "delete", "T001")
	if code == 0 {
		t.Error("task delete succeeded without --reason")
	}
	if !strings.Contains(out, "reason") {
		t.Logf("error message did not mention reason: %s", out)
	}
}

func TestTaskReopen_NoReason(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "task", "reopen", "T001")
	if code == 0 {
		t.Error("task reopen succeeded without --reason")
	}
	if !strings.Contains(out, "reason") {
		t.Logf("error message did not mention reason: %s", out)
	}
}

func TestSprintStart_NoArgs(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "sprint", "start")
	if code == 0 {
		t.Error("sprint start succeeded with no args - violates ExactArgs(1)")
	}
}

func TestSprintProgress_NoArgs(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "sprint", "progress")
	if code == 0 {
		t.Error("sprint progress succeeded with no args - violates ExactArgs(1)")
	}
}

func TestSprintCreate_NoFlags(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "sprint", "create")
	if code == 0 {
		t.Error("sprint create succeeded without required flags")
	}
	if !strings.Contains(out, "id") || !strings.Contains(out, "title") {
		t.Logf("required flag guidance missing: %s", out)
	}
}

func TestTaskCreate_NoFlags(t *testing.T) {
	skipIfNoBinary(t)
	out, code := runHstl(t, "task", "create")
	if code == 0 {
		t.Error("task create succeeded without required flags")
	}
	if !strings.Contains(out, "title") {
		t.Logf("required flag guidance missing: %s", out)
	}
}

// ===== harness subcommand tests =====

func TestHarnessGet_NoArgs(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "harness", "get")
	if code == 0 {
		t.Error("harness get succeeded with no args")
	}
}

func TestHarnessCheck_NoArgs(t *testing.T) {
	skipIfNoBinary(t)
	_, code := runHstl(t, "harness", "check")
	if code == 0 {
		t.Error("harness check succeeded with no args")
	}
}

// ===== Verify --help passes on every top-level command =====

func TestAllTopLevelCommands_HelpExitZero(t *testing.T) {
	skipIfNoBinary(t)
	commands := []string{
		"task", "sprint", "kb", "audit", "registry",
		"backlog", "harness", "version",
	}
	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			_, code := runHstl(t, cmd, "--help")
			if code != 0 {
				t.Errorf("%s --help exit code %d (expected 0)", cmd, code)
			}
		})
	}
}

// ===== Pipeline combination tests =====

// runHstlSplit runs the binary and returns separated stdout/stderr.
func runHstlSplit(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(hstlBin, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// runHstlIsolated runs the hstl binary in a tmpDir-based isolated env.
// Used as a regression-guard runner to verify stdout/stderr separation
// for write commands (task create, kb create, ...) without touching the real project DB.
func runHstlIsolated(t *testing.T, tmpDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(hstlBin, args...)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		"HSTL_PROJECT_ROOT="+tmpDir,
		"HSTL_DB_PATH="+filepath.Join(tmpDir, "hstl.db"),
		// Lift the strict summary heuristic for fixture-driven integration tests —
		// the strict preset is meant for real authored Tasks, not test fixtures.
		"HSTL_SUMMARY_POLICY=off",
		"HSTL_SUMMARY_SUBAGENT=off",
	)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

func TestPipeline_TaskList_JSONValid(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "task", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("task list -o json exit code %d", code)
	}
	if !json.Valid([]byte(stdout)) {
		t.Errorf("stdout is not valid JSON: %s", stdout[:min(len(stdout), 200)])
	}
}

func TestPipeline_TaskList_JSONSchema(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "task", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var envelope struct {
		Status string                     `json:"status"`
		Data   map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if _, ok := envelope.Data["tasks"]; !ok {
		t.Error("missing tasks key in data envelope")
	}
	if _, ok := envelope.Data["count"]; !ok {
		t.Error("missing count key in data envelope")
	}
}

func TestPipeline_SprintList_JSONValid(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "sprint", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("sprint list -o json exit code %d", code)
	}
	if !json.Valid([]byte(stdout)) {
		t.Errorf("stdout is not valid JSON: %s", stdout[:min(len(stdout), 200)])
	}
}

func TestPipeline_SprintList_JSONSchema(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "sprint", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var envelope struct {
		Status string                     `json:"status"`
		Data   map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if _, ok := envelope.Data["sprints"]; !ok {
		t.Error("missing sprints key in data envelope")
	}
	if _, ok := envelope.Data["count"]; !ok {
		t.Error("missing count key in data envelope")
	}
}

func TestPipeline_Version_JSONValid(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "version", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("version -o json exit code %d", code)
	}
	if !json.Valid([]byte(stdout)) {
		t.Errorf("stdout is not valid JSON: %s", stdout[:min(len(stdout), 200)])
	}
}

func TestPipeline_Quiet_StderrSilent(t *testing.T) {
	skipIfNoBinary(t)
	_, stderr, code := runHstlSplit(t, "task", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Errorf("--quiet still produced stderr output: %s", stderr[:min(len(stderr), 200)])
	}
}

func TestPipeline_JSONUnmarshal_TaskList(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "task", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var result struct {
		Tasks []struct {
			TaskID   string `json:"task_id"`
			Title    string `json:"title"`
			Type     string `json:"type"`
			Status   string `json:"status"`
			Priority string `json:"priority"`
		} `json:"tasks"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if result.Count != len(result.Tasks) {
		t.Errorf("count(%d) != len(tasks)(%d)", result.Count, len(result.Tasks))
	}
	for i, task := range result.Tasks {
		if task.TaskID == "" {
			t.Errorf("tasks[%d].task_id is empty", i)
		}
	}
}

// TestT479_TaskCreate_StdoutPureJSON verifies that task create -o json's stdout
// is not polluted by warning messages (body authoring reminder) and contains only pure JSON
// (regression guard). Runs in an isolated tmpDir without touching the real DB.
func TestT479_TaskCreate_StdoutPureJSON(t *testing.T) {
	skipIfNoBinary(t)
	tmpDir := t.TempDir()
	// Minimal works structure.
	for _, d := range []string{
		filepath.Join(tmpDir, "works", "tasks"),
		filepath.Join(tmpDir, "works", "sprints", "backlog"),
		filepath.Join(tmpDir, "works", "sprints", "active"),
		filepath.Join(tmpDir, "works", "data", "task"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to mkdir %s: %v", d, err)
		}
	}

	stdout, stderr, code := runHstlIsolated(t, tmpDir,
		"task", "create",
		"--title", "T479 stdout purity verification",
		"--summary", "regression test verifying that task create stdout is pure JSON",
		"--type", "chore",
		"-o", "json",
	)
	if code != 0 {
		t.Fatalf("task create exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		t.Fatalf("stdout is empty")
	}
	if !json.Valid([]byte(trimmed)) {
		t.Errorf("stdout is not valid JSON:\n%s", trimmed)
	}
	// stderr should carry the warning (reminder).
	if !strings.Contains(stderr, "warn") && !strings.Contains(stderr, "created") {
		t.Logf("stderr warning not detected (informational): %s", stderr)
	}
	// stdout must never include the warning emoji.
	if strings.Contains(trimmed, "⚠") {
		t.Errorf("stdout leaked the warning emoji (must only appear on stderr):\n%s", trimmed)
	}
}

func TestPipeline_StdoutStderr_Separation(t *testing.T) {
	skipIfNoBinary(t)
	commands := []struct {
		name string
		args []string
	}{
		{"task list", []string{"task", "list", "-o", "json", "-q"}},
		{"sprint list", []string{"sprint", "list", "-o", "json", "-q"}},
		{"version", []string{"version", "-o", "json", "-q"}},
		{"task next", []string{"task", "next", "-o", "json", "-q"}},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, _ := runHstlSplit(t, tc.args...)
			stdout = strings.TrimSpace(stdout)
			stderr = strings.TrimSpace(stderr)

			if stdout == "" {
				t.Errorf("%s: stdout is empty", tc.name)
				return
			}
			if !json.Valid([]byte(stdout)) {
				t.Errorf("%s: stdout is not valid JSON", tc.name)
			}
			if stderr != "" {
				t.Logf("%s: stderr not empty (informational): %s", tc.name, stderr[:min(len(stderr), 100)])
			}
		})
	}
}

// ===== CLI-Hook contract tests =====
// Verify the required JSON output fields used by jq expressions in hooks
// (session-context.sh, workflow-validator.sh). When the format changes,
// these tests fail and force the hooks to be updated.

func TestContract_TaskList_HookFields(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "task", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Hook contract: .tasks[].task_id, .tasks[].title, .tasks[].status, .tasks | length
	var result struct {
		Tasks []struct {
			TaskID string `json:"task_id"`
			Title  string `json:"title"`
			Status string `json:"status"`
		} `json:"tasks"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Hook contract violation: failed to parse task list JSON: %v", err)
	}
	for i, task := range result.Tasks {
		if task.TaskID == "" {
			t.Errorf("Hook contract violation: tasks[%d].task_id is empty", i)
		}
		if task.Title == "" {
			t.Errorf("Hook contract violation: tasks[%d].title is empty", i)
		}
		if task.Status == "" {
			t.Errorf("Hook contract violation: tasks[%d].status is empty", i)
		}
	}
}

func TestContract_SprintList_HookFields(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "sprint", "list", "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	// Hook contract: .sprints[].sprint_id, .sprints | length
	var result struct {
		Sprints []struct {
			SprintID string `json:"sprint_id"`
		} `json:"sprints"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Hook contract violation: failed to parse sprint list JSON: %v", err)
	}
	for i, s := range result.Sprints {
		if s.SprintID == "" {
			t.Errorf("Hook contract violation: sprints[%d].sprint_id is empty", i)
		}
	}
}

func TestContract_SprintProgress_HookFields(t *testing.T) {
	skipIfNoBinary(t)
	// sprint progress needs an active Sprint, so look one up first.
	listOut, _, code := runHstlSplit(t, "sprint", "list", "-o", "json", "-q", "--status", "active")
	if code != 0 {
		t.Skip("sprint list failed")
	}
	var listResult struct {
		Sprints []struct {
			SprintID string `json:"sprint_id"`
		} `json:"sprints"`
	}
	if err := json.Unmarshal([]byte(listOut), &listResult); err != nil || len(listResult.Sprints) == 0 {
		t.Skip("no active Sprint - skipping contract test")
	}
	sid := listResult.Sprints[0].SprintID

	stdout, _, _ := runHstlSplit(t, "sprint", "progress", sid, "-o", "json", "-q")
	// Hook contract: .done, .total
	var result struct {
		Done  *int `json:"done"`
		Total *int `json:"total"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Hook contract violation: failed to parse sprint progress JSON: %v", err)
	}
	if result.Done == nil {
		t.Error("Hook contract violation: missing done field")
	}
	if result.Total == nil {
		t.Error("Hook contract violation: missing total field")
	}
}

func TestContract_TaskList_StatusFilter(t *testing.T) {
	skipIfNoBinary(t)
	// Verify that the hook can use the --status done filter.
	stdout, _, code := runHstlSplit(t, "task", "list", "-o", "json", "-q", "--status", "done")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var result struct {
		Tasks []struct {
			TaskID string `json:"task_id"`
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("Hook contract violation: failed to parse task list --status done: %v", err)
	}
	for i, task := range result.Tasks {
		if task.Status != "done" {
			t.Errorf("Hook contract violation: --status done filter returned tasks[%d].status = %q", i, task.Status)
		}
	}
}

func TestPipeline_SprintProgress_JSONValid(t *testing.T) {
	skipIfNoBinary(t)
	// Look up an active Sprint.
	listOut, _, _ := runHstlSplit(t, "sprint", "list", "-o", "json", "-q", "--status", "active")
	var listResult struct {
		Sprints []struct {
			SprintID string `json:"sprint_id"`
		} `json:"sprints"`
	}
	if err := json.Unmarshal([]byte(listOut), &listResult); err != nil || len(listResult.Sprints) == 0 {
		t.Skip("no active Sprint")
	}
	sid := listResult.Sprints[0].SprintID

	stdout, _, code := runHstlSplit(t, "sprint", "progress", sid, "-o", "json", "-q")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !json.Valid([]byte(stdout)) {
		t.Errorf("stdout is not valid JSON: %s", stdout[:min(len(stdout), 200)])
	}
	var result struct {
		SprintID string `json:"sprint_id"`
		Done     int    `json:"done"`
		Total    int    `json:"total"`
		Percent  int    `json:"percent"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if result.SprintID != sid {
		t.Errorf("sprint_id mismatch: %q != %q", result.SprintID, sid)
	}
}

func TestPipeline_TaskList_FilterCombo(t *testing.T) {
	skipIfNoBinary(t)
	stdout, _, code := runHstlSplit(t, "task", "list", "-o", "json", "-q", "--status", "done")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var result struct {
		Tasks []struct {
			Status string `json:"status"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	for i, task := range result.Tasks {
		if task.Status != "done" {
			t.Errorf("tasks[%d].status = %q, expected done", i, task.Status)
		}
	}
}

// TestT630_TaskAssign_PlaceholderBody_ReturnsError verifies that assigning
// a placeholder-body Task to a sprint returns exit code 1 + JSON status "error".
// Reproduces a downstream incident — even on failure the response
// status was "ok" with "assigned" displayed.
func TestT630_TaskAssign_PlaceholderBody_ReturnsError(t *testing.T) {
	skipIfNoBinary(t)
	tmpDir := t.TempDir()
	for _, d := range []string{
		filepath.Join(tmpDir, "works", "tasks"),
		filepath.Join(tmpDir, "works", "sprints", "backlog"),
		filepath.Join(tmpDir, "works", "sprints", "active"),
		filepath.Join(tmpDir, "works", "data", "task"),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to mkdir %s: %v", d, err)
		}
	}

	// Create the Sprint.
	_, _, code := runHstlIsolated(t, tmpDir, "sprint", "create",
		"--id", "sprint-t630", "--title", "T630 test Sprint", "--goal", "T630 test")
	if code != 0 {
		t.Skip("sprint create failed - environment unsupported")
	}

	// Create a Task with a placeholder body.
	stdout, _, code := runHstlIsolated(t, tmpDir, "task", "create",
		"--title", "T630 placeholder task", "--summary", "test Task verifying placeholder-body assign block",
		"--type", "chore", "-o", "json")
	if code != 0 {
		t.Fatalf("task create failed exit %d", code)
	}
	var created struct {
		Data struct {
			TaskID string `json:"task_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &created); err != nil {
		t.Fatalf("failed to parse task create JSON: %v\n%s", err, stdout)
	}
	taskID := created.Data.TaskID

	// Extract a sprint_id from sprint list.
	listOut, _, _ := runHstlIsolated(t, tmpDir, "sprint", "list", "-o", "json", "-q")
	var sprintList struct {
		Sprints []struct {
			SprintID string `json:"sprint_id"`
		} `json:"sprints"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(listOut)), &sprintList); err != nil || len(sprintList.Sprints) == 0 {
		t.Skip("failed to parse sprint list or no sprints")
	}
	sprintID := sprintList.Sprints[0].SprintID

	// Attempting assign with a placeholder must produce exit 1 + JSON status "error".
	assignOut, _, assignCode := runHstlIsolated(t, tmpDir, "task", "assign", taskID,
		"--sprint", sprintID, "-o", "json")

	if assignCode == 0 {
		t.Errorf("T630: placeholder Task assign returned exit 0 - expected exit 1\nstdout: %s", assignOut)
	}

	// stdout must be valid JSON.
	trimmed := strings.TrimSpace(assignOut)
	if trimmed != "" && json.Valid([]byte(trimmed)) {
		var resp struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal([]byte(trimmed), &resp); err == nil {
			if resp.Status == "ok" {
				t.Errorf("T630: JSON status = %q, expected error - placeholder Task assign reported ok", resp.Status)
			}
		}
	}
}
