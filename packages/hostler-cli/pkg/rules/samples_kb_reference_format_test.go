// trac: HAR-CM015
package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupKBRefTree writes a tree of N files under a t.TempDir-based root,
// including a .git stub.
func setupKBRefTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runKBRef(t *testing.T, root string) *RuleResult {
	t.Helper()
	r, ok := Get(kbReferenceFormatRuleID)
	if !ok {
		t.Fatalf("Rule %s not registered", kbReferenceFormatRuleID)
	}
	return r.Check(&RuleContext{ProjectRoot: root})
}

// Pattern A — `KB W001` form is BLOCKed.
func TestT525_KBRef_PatternA_KB_Bare_Violation(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/guide.md": "# guide\n\nKB W001 registered.\n",
	})
	stubStaged(t, []string{"docs/guide.md"})
	res := runKBRef(t, root)
	if res.Status != StatusViolated {
		t.Fatalf("status=%v want Violated", res.Status)
	}
	found := false
	for _, e := range res.Evidence {
		if strings.Contains(e, "A.kb_bare") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected A.kb_bare in evidence, got %v", res.Evidence)
	}
}

// Pattern B — `[KB:W001]` markdown-link form is BLOCKed.
//
// The fixture path uses docs/ rather than works/tasks/T (the latter is in
// the allow list because of placeholder body reminder false positives).
func TestT525_KBRef_PatternB_KB_Link_Violation(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/sample.md": "# sample\n\n[KB:M001] reference.\n",
	})
	stubStaged(t, []string{"docs/sample.md"})
	res := runKBRef(t, root)
	if res.Status != StatusViolated {
		t.Fatalf("status=%v want Violated", res.Status)
	}
	found := false
	for _, e := range res.Evidence {
		if strings.Contains(e, "B.kb_link") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected B.kb_link in evidence, got %v", res.Evidence)
	}
}

// Canonical form — file path + ID combination passes.
func TestT525_KBRef_FullReference_OK(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/guide.md": "# guide\n\n`docs/07-knowledge/mistakes/refactoring.md#m001` reference.\n",
	})
	stubStaged(t, []string{"docs/guide.md"})
	res := runKBRef(t, root)
	if res.Status != StatusOK {
		t.Errorf("expected OK, got %v evidence=%v", res.Status, res.Evidence)
	}
}

// allow — KB card body itself.
func TestT525_KBRef_AllowedKBCard(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/07-knowledge/mistakes/refactoring.md": "# refactoring KB\n\nKB M001 definition.\n",
	})
	stubStaged(t, []string{"docs/07-knowledge/mistakes/refactoring.md"})
	res := runKBRef(t, root)
	if res.Status != StatusSkipped {
		t.Errorf("expected Skipped, got %v", res.Status)
	}
}

// allow — KB-card-id-governance ADR.
func TestT525_KBRef_AllowedADR046(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/02-architecture/adrs/ADR-046-kb-card-id-governance.md": "# ADR-046\n\nNegative example: `KB W001`.\n",
	})
	stubStaged(t, []string{"docs/02-architecture/adrs/ADR-046-kb-card-id-governance.md"})
	res := runKBRef(t, root)
	if res.Status != StatusSkipped {
		t.Errorf("expected Skipped, got %v", res.Status)
	}
}

// frontmatter region KB IDs are ignored.
func TestT525_KBRef_FrontmatterBoundary(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"docs/guide.md": "---\nkb_ref: KB W001\n---\n\n# Body\nLegitimate content.\n",
	})
	stubStaged(t, []string{"docs/guide.md"})
	res := runKBRef(t, root)
	if res.Status != StatusOK {
		t.Errorf("expected OK (frontmatter excluded), got %v evidence=%v", res.Status, res.Evidence)
	}
}

// Regression guard — detect substring collisions between the allow list
// and test fixtures. Adding the 'works/tasks/T' substring to the allow
// list previously caused existing 'works/tasks/Tnnn.md' fixtures to skip,
// regressing PatternB. This guard verifies that no fixture path contains
// any allow substring.
func TestT525_KBRef_AllowListAndFixtureNoCollision(t *testing.T) {
	// PatternA/B violation fixture paths — collision with the allow list
	// causes a status=Skipped regression.
	violationFixtures := []string{
		"docs/guide.md",  // PatternA
		"docs/sample.md", // PatternB
	}
	for _, fx := range violationFixtures {
		for _, sub := range kbReferenceAllowedSubpaths {
			if strings.Contains(fx, sub) {
				t.Errorf("fixture %q collides with allow substring %q — PatternA/B regression risk", fx, sub)
			}
		}
	}
}

// non-.md files skip.
func TestT525_KBRef_NonMD_Skipped(t *testing.T) {
	root := setupKBRefTree(t, map[string]string{
		"main.go": "package main\n// KB W001 in code\n",
	})
	stubStaged(t, []string{"main.go"})
	res := runKBRef(t, root)
	if res.Status != StatusSkipped {
		t.Errorf("expected Skipped (non-.md), got %v", res.Status)
	}
}
