package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// Regression tests for the dedicated implementations that replaced the
// samples.go bridge.

func writeTaskFile(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "T999-fake.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestT589_ResultFilesExist_ExtractPaths — verify that backtick paths in
// the `### Artifacts` section are extracted and compared against the git
// diff stub, with missing files being reported.
func TestT589_ResultFilesExist_ExtractPaths(t *testing.T) {
	body := `---
id: T999
---

# T999 fake

## Result

### Artifacts

- ` + "`cli/pkg/foo/bar.go`" + ` (modified)
- ` + "`cli/pkg/foo/bar_test.go`" + ` (new)
`
	path := writeTaskFile(t, body)

	// git diff stub: only one file is staged; the other is missing.
	orig := gitChangedFilesSinceTaskStart
	gitChangedFilesSinceTaskStart = func() (map[string]struct{}, error) {
		return map[string]struct{}{"cli/pkg/foo/bar.go": {}}, nil
	}
	defer func() { gitChangedFilesSinceTaskStart = orig }()

	r, _ := Get("task.result_section.files_exist")
	res := r.Check(&RuleContext{SelfFilePath: path})
	if res.Status != StatusViolated {
		t.Errorf("status=%v, want Violated", res.Status)
	}
	if len(res.Evidence) != 1 || res.Evidence[0] != "cli/pkg/foo/bar_test.go" {
		t.Errorf("evidence=%v, expected single missing file", res.Evidence)
	}
}

// TestT589_ResultFilesExist_FullMatch — when every path is present in the
// git diff, the result is OK.
func TestT589_ResultFilesExist_FullMatch(t *testing.T) {
	body := `---
id: T999
---

# T999

## Result

### Artifacts

- ` + "`a.go`" + `
- ` + "`b.go`" + `
`
	path := writeTaskFile(t, body)
	orig := gitChangedFilesSinceTaskStart
	gitChangedFilesSinceTaskStart = func() (map[string]struct{}, error) {
		return map[string]struct{}{"a.go": {}, "b.go": {}}, nil
	}
	defer func() { gitChangedFilesSinceTaskStart = orig }()

	r, _ := Get("task.result_section.files_exist")
	res := r.Check(&RuleContext{SelfFilePath: path})
	if res.Status != StatusOK {
		t.Errorf("status=%v, want OK, evidence=%v", res.Status, res.Evidence)
	}
}

// TestT589_ResultFilesExist_NoSection_Skip — with no `### Artifacts`
// section, the result is Skipped.
func TestT589_ResultFilesExist_NoSection_Skip(t *testing.T) {
	body := "# T999\n\n## Purpose\n\nSome description only.\n"
	path := writeTaskFile(t, body)
	r, _ := Get("task.result_section.files_exist")
	res := r.Check(&RuleContext{SelfFilePath: path})
	if res.Status != StatusSkipped {
		t.Errorf("status=%v, want Skipped", res.Status)
	}
}

// TestT589_BodyNotPlaceholder_ShortBody_Violated — body shorter than 200
// chars violates the rule.
func TestT589_BodyNotPlaceholder_ShortBody_Violated(t *testing.T) {
	body := "---\nid: T999\n---\n\n# T999\n\n## Purpose\n\nshort.\n"
	path := writeTaskFile(t, body)
	r, _ := Get("task.body.not_placeholder")
	res := r.Check(&RuleContext{SelfFilePath: path})
	if res.Status != StatusViolated {
		t.Errorf("status=%v, want Violated", res.Status)
	}
}

// TestT589_BodyNotPlaceholder_SufficientBody_OK — sufficiently long body
// with few markers is OK.
func TestT589_BodyNotPlaceholder_SufficientBody_OK(t *testing.T) {
	long := "This is a sufficiently long Task body description. The purpose and requirements are clear and almost no placeholder markers appear. "
	body := "---\nid: T999\n---\n\n# T999\n\n## Purpose\n\n"
	for i := 0; i < 10; i++ {
		body += long
	}
	path := writeTaskFile(t, body)
	r, _ := Get("task.body.not_placeholder")
	res := r.Check(&RuleContext{SelfFilePath: path})
	if res.Status != StatusOK {
		t.Errorf("status=%v, want OK, evidence=%v", res.Status, res.Evidence)
	}
}

// TestT589_BodyNotPlaceholder_TooManyMarkers_Violated — exceeding the
// marker-ratio threshold is a violation.
func TestT589_BodyNotPlaceholder_TooManyMarkers_Violated(t *testing.T) {
	body := "---\nid: T999\n---\n\n# T999\n\n## Purpose\n\n"
	// many short markers — content_len > 200 with a high marker ratio
	body += "TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO TODO\n"
	path := writeTaskFile(t, body)
	r, _ := Get("task.body.not_placeholder")
	res := r.Check(&RuleContext{SelfFilePath: path})
	if res.Status != StatusViolated {
		t.Errorf("status=%v, want Violated, evidence=%v", res.Status, res.Evidence)
	}
}

// TestT589_StripFrontmatter — regression test for the frontmatter-strip
// utility.
func TestT589_StripFrontmatter(t *testing.T) {
	in := "---\nkey: value\n---\nbody"
	got := stripFrontmatter(in)
	if got != "body" {
		t.Errorf("stripFrontmatter=%q, want %q", got, "body")
	}
	// Without frontmatter, returns the input unchanged.
	if got := stripFrontmatter("no frontmatter"); got != "no frontmatter" {
		t.Errorf("no-frontmatter passthrough failed: %q", got)
	}
}
