// trac: HAR-CM015
package rules

import (
	"strings"
	"testing"
)

func TestT588_UUIDFrontmatter_Pattern(t *testing.T) {
	good := []string{
		"019dc973-2e17-7179-865c-8e56f72d0097",
		"00000000-0000-7000-8000-000000000000",
		"ffffffff-ffff-7fff-bfff-ffffffffffff",
		"019DC973-2E17-7179-865C-8E56F72D0097",
	}
	bad := []string{
		"",
		"invalid",
		"019dc973-2e17-4179-865c-8e56f72d0097",       // version 4 (not 7)
		"019dc973-2e17-7179-c65c-8e56f72d0097",       // variant c (not 8/9/a/b)
		"019dc9732e17-7179-865c-8e56f72d0097",        // missing dash
		"019dc973-2e17-7179-865c-8e56f72d0097-extra", // too long
	}
	for _, g := range good {
		if !uuidV7Pattern.MatchString(g) {
			t.Errorf("good UUID v7 %q failed to match", g)
		}
	}
	for _, b := range bad {
		if uuidV7Pattern.MatchString(b) {
			t.Errorf("bad UUID %q matched", b)
		}
	}
}

func TestT588_UUIDFrontmatter_TargetFilter(t *testing.T) {
	cases := []struct {
		path  string
		match bool
	}{
		{"docs/02-architecture/adrs/ADR-099.md", true},
		{"docs/03-design/frs/foo.md", true},
		{"docs/03-design/ux/scenarios/foo.md", true},
		{"docs/04-guides/bar.md", true},
		{"docs/08-references/standards/baz.md", true},
		{"docs/07-knowledge/mistakes/m.md", false}, // KB card body — activate after 1-file-1-card migration
		{"packages/hostler-cli/README.md", true},
		// exceptions
		{"docs/07-knowledge/INDEX.md", false},
		{"docs/07-knowledge/README.md", false},
		// non-targets
		{"docs/00-project/vision.md", false},
		{"works/sprints/active/sprint-77/SPRINT.md", false},
		{"works/sprints/active/sprint-77/tasks/T660.md", false},
		{"works/tasks/BACKLOG.md", false},
		{"archive/docs/old.md", false},
		{"main.go", false},
	}
	out := filterUUIDTargets(extractStr(cases))
	expected := map[string]bool{}
	for _, c := range cases {
		if c.match {
			expected[c.path] = true
		}
	}
	if len(out) != len(expected) {
		t.Errorf("filtered count = %d, want %d (out=%v)", len(out), len(expected), out)
	}
	for _, p := range out {
		if !expected[p] {
			t.Errorf("unexpected match: %s", p)
		}
	}
}

func extractStr(cases []struct {
	path  string
	match bool
}) []string {
	out := make([]string, len(cases))
	for i, c := range cases {
		out[i] = c.path
	}
	return out
}

func TestT588_ExtractFrontmatterField(t *testing.T) {
	body := "---\nuuid: 019dc973-2e17-7179-865c-8e56f72d0097\ntitle: foo\n---\n\nbody\n"
	got := extractFrontmatterField(body, "uuid")
	want := "019dc973-2e17-7179-865c-8e56f72d0097"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if extractFrontmatterField(body, "missing") != "" {
		t.Errorf("missing key did not return empty string")
	}
	if extractFrontmatterField("# no frontmatter\n", "uuid") != "" {
		t.Errorf("no-frontmatter did not return empty string")
	}
}

func TestT588_UUIDFrontmatter_RuleViolation_MissingUUID(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/04-guides/foo.md": "---\ntitle: foo\n---\n\n# Foo\nbody.\n",
	})
	stubStaged(t, []string{"docs/04-guides/foo.md"})
	r, _ := Get(uuidFrontmatterRuleID)
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusViolated {
		t.Fatalf("expected Violated, got %v", res.Status)
	}
	found := false
	for _, e := range res.Evidence {
		if strings.Contains(e, "uuid field missing") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'field missing' evidence, got %v", res.Evidence)
	}
}

func TestT588_UUIDFrontmatter_RuleViolation_BadFormat(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/04-guides/foo.md": "---\nuuid: invalid-uuid\ntitle: foo\n---\n\n# Foo\n",
	})
	stubStaged(t, []string{"docs/04-guides/foo.md"})
	r, _ := Get(uuidFrontmatterRuleID)
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusViolated {
		t.Fatalf("expected Violated, got %v", res.Status)
	}
	found := false
	for _, e := range res.Evidence {
		if strings.Contains(e, "format violation") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'format violation' evidence, got %v", res.Evidence)
	}
}

func TestT588_UUIDFrontmatter_RuleOK(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/04-guides/foo.md": "---\nuuid: 019dc973-2e17-7179-865c-8e56f72d0097\ntitle: foo\n---\n\n# Foo\n",
	})
	stubStaged(t, []string{"docs/04-guides/foo.md"})
	r, _ := Get(uuidFrontmatterRuleID)
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusOK {
		t.Errorf("expected OK, got %v evidence=%v", res.Status, res.Evidence)
	}
}

func TestT588_UUIDFrontmatter_NonTarget_Skipped(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"works/tasks/T999.md": "---\nid: T999\n---\n\n",
	})
	stubStaged(t, []string{"works/tasks/T999.md"})
	r, _ := Get(uuidFrontmatterRuleID)
	res := r.Check(&RuleContext{ProjectRoot: root})
	if res.Status != StatusSkipped {
		t.Errorf("expected Skipped (non-target path), got %v", res.Status)
	}
}
