package rules

import (
	"strings"
	"testing"
)

// commit-convention 3-rule unit tests.

func runCheck(t *testing.T, id string, extra map[string]any) *RuleResult {
	t.Helper()
	r, ok := Get(id)
	if !ok {
		t.Fatalf("Rule %s not registered", id)
	}
	ctx := &RuleContext{Extra: extra}
	return r.Check(ctx)
}

// commit.message.korean tests removed: the fixtures required Korean
// text to exercise the rule, and the OSS sweep policy forbids Korean
// in source. The rule itself remains registered for downstream users
// who need it.

func TestCommitMessageKorean_CoAuthored_Excluded(t *testing.T) {
	// only a trailer with a short body → skip (below minimum length).
	msg := "fix\n\nCo-Authored-By: Someone <x@y.z>"
	res := runCheck(t, "commit.message.korean", map[string]any{"commit_msg": msg})
	if res.Status != StatusSkipped {
		t.Errorf("body too short → expected skip, got status=%d", res.Status)
	}
}

func TestCommitMessageKorean_NoMessage_Skip(t *testing.T) {
	res := runCheck(t, "commit.message.korean", map[string]any{})
	if res.Status != StatusSkipped {
		t.Errorf("missing commit_msg → expected skip, got %d", res.Status)
	}
}

func TestCommitTaskIDPresent_TaskIDFound(t *testing.T) {
	msg := "fix(T575): work in progress"
	res := runCheck(t, "commit.task_id.present", map[string]any{"commit_msg": msg})
	if res.Status != StatusOK {
		t.Errorf("T575 present but check did not pass: %d %s", res.Status, res.Message)
	}
}

func TestCommitTaskIDPresent_TaskIDMissing(t *testing.T) {
	msg := "chore: misc work"
	res := runCheck(t, "commit.task_id.present", map[string]any{"commit_msg": msg})
	if res.Status != StatusViolated {
		t.Errorf("no T-id but not flagged: %d", res.Status)
	}
}

// Restore the literal via diff text including the file header in the
// production path. This file is `_test.go` and is auto-excluded by the
// secret scanner's path exception, even when it appears in a staged diff
// (regression guard).

func TestCommitFilesNoSecrets_AWSKey_Blocked(t *testing.T) {
	diff := "diff --git a/cli/pkg/config/foo.go b/cli/pkg/config/foo.go\n" +
		"--- a/cli/pkg/config/foo.go\n" +
		"+++ b/cli/pkg/config/foo.go\n" +
		"@@ -1,1 +1,2 @@\n" +
		"+const key = \"" + "AKIA" + "IOSFODNN7EXAMPLE\"\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusViolated {
		t.Errorf("AWS key passed: %d", res.Status)
	}
	if len(res.Evidence) == 0 || !strings.Contains(res.Evidence[0], "aws_access_key") {
		t.Errorf("evidence missing aws_access_key: %v", res.Evidence)
	}
}

func TestCommitFilesNoSecrets_PrivateKey_Blocked(t *testing.T) {
	diff := "diff --git a/keys/id_rsa b/keys/id_rsa\n" +
		"--- a/keys/id_rsa\n" +
		"+++ b/keys/id_rsa\n" +
		"+-----" + "BEGIN RSA PRIVATE KEY-----\n" +
		"+MIIEpAIB...\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusViolated {
		t.Errorf("private key passed: %d", res.Status)
	}
}

func TestCommitFilesNoSecrets_CleanDiff_Passes(t *testing.T) {
	diff := "diff --git a/cli/pkg/clean/foo.go b/cli/pkg/clean/foo.go\n" +
		"--- a/cli/pkg/clean/foo.go\n" +
		"+++ b/cli/pkg/clean/foo.go\n" +
		"+func foo() { return 42 }\n+var bar = \"hello\"\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusOK {
		t.Errorf("clean diff did not return OK: status=%d evidence=%v", res.Status, res.Evidence)
	}
}

func TestCommitFilesNoSecrets_APIKey_Assignment_Blocked(t *testing.T) {
	diff := "diff --git a/cli/pkg/api/foo.go b/cli/pkg/api/foo.go\n" +
		"--- a/cli/pkg/api/foo.go\n" +
		"+++ b/cli/pkg/api/foo.go\n" +
		"+" + "api_key" + ` = "` + "sk_" + "live_" + "abcdefghij1234567890abcdef" + `"` + "\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusViolated {
		t.Errorf("api_key assignment not detected: %d", res.Status)
	}
}

// path exception: secret literals on _test.go paths are not scanned.
func TestCommitFilesNoSecrets_TestFile_PathSkip(t *testing.T) {
	diff := "diff --git a/cli/pkg/rules/samples_commit_test.go b/cli/pkg/rules/samples_commit_test.go\n" +
		"--- a/cli/pkg/rules/samples_commit_test.go\n" +
		"+++ b/cli/pkg/rules/samples_commit_test.go\n" +
		"+const testKey = \"" + "AKIA" + "IOSFODNN7EXAMPLE\"\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusOK {
		t.Errorf("_test.go path exception failed: status=%d evidence=%v", res.Status, res.Evidence)
	}
	if len(res.Evidence) == 0 || !strings.Contains(res.Evidence[0], "path_skipped=1") {
		t.Errorf("evidence missing path_skipped=1: %v", res.Evidence)
	}
}

// path exception: testdata/ subtrees are also skipped.
func TestCommitFilesNoSecrets_Testdata_PathSkip(t *testing.T) {
	diff := "diff --git a/cli/pkg/foo/testdata/bad.txt b/cli/pkg/foo/testdata/bad.txt\n" +
		"--- a/cli/pkg/foo/testdata/bad.txt\n" +
		"+++ b/cli/pkg/foo/testdata/bad.txt\n" +
		"+" + "AKIA" + "IOSFODNN7EXAMPLE\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusOK {
		t.Errorf("testdata/ path exception failed: status=%d", res.Status)
	}
}

// line marker: lines containing `// nosecret` are skipped individually.
func TestCommitFilesNoSecrets_NosecretMarker_Skip(t *testing.T) {
	diff := "diff --git a/cli/pkg/demo/demo.go b/cli/pkg/demo/demo.go\n" +
		"--- a/cli/pkg/demo/demo.go\n" +
		"+++ b/cli/pkg/demo/demo.go\n" +
		"+const k = \"" + "AKIA" + "IOSFODNN7EXAMPLE\" // nosecret — example for docs\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusOK {
		t.Errorf("nosecret marker skip failed: status=%d evidence=%v", res.Status, res.Evidence)
	}
	if len(res.Evidence) == 0 || !strings.Contains(res.Evidence[0], "marker_skipped=1") {
		t.Errorf("evidence missing marker_skipped=1: %v", res.Evidence)
	}
}

// the line marker has no effect on adjacent lines: only the marked line is
// skipped.
func TestCommitFilesNoSecrets_Marker_AdjacentLineUnaffected(t *testing.T) {
	diff := "diff --git a/cli/pkg/demo/demo.go b/cli/pkg/demo/demo.go\n" +
		"--- a/cli/pkg/demo/demo.go\n" +
		"+++ b/cli/pkg/demo/demo.go\n" +
		"+const a = \"" + "AKIA" + "IOSFODNN7EXAMPLE\" // nosecret\n" +
		"+const b = \"" + "AKIA" + "AAAAAAAAAAAAAAAA\"\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusViolated {
		t.Errorf("second line should be blocked: status=%d", res.Status)
	}
	if len(res.Evidence) == 0 {
		t.Errorf("violated but evidence is empty")
	}
}

// `#`-style markers are also recognised (shell/yaml support).
func TestCommitFilesNoSecrets_HashMarker_Skip(t *testing.T) {
	diff := "diff --git a/scripts/deploy.sh b/scripts/deploy.sh\n" +
		"--- a/scripts/deploy.sh\n" +
		"+++ b/scripts/deploy.sh\n" +
		"+KEY=\"" + "AKIA" + "IOSFODNN7EXAMPLE\"  # nosecret\n"
	res := runCheck(t, "commit.files.no_secrets", map[string]any{"staged_diff": diff})
	if res.Status != StatusOK {
		t.Errorf("# nosecret marker failed: status=%d", res.Status)
	}
}

func TestCommitFilesNoSecrets_NoDiff_Skip(t *testing.T) {
	res := runCheck(t, "commit.files.no_secrets", map[string]any{})
	if res.Status != StatusSkipped {
		t.Errorf("no diff → expected skip, got %d", res.Status)
	}
}
