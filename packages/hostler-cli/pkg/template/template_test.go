package template_test

import (
	"strings"
	"testing"
	"time"

	tmpl "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/template"
)

// TestRenderTaskTemplate_default verifies the basic Task template body.
func TestRenderTaskTemplate_default(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T009", "Sample Task", "feature", "sprint-01",
		"p1", "M",
		[]string{"T001"},
		"",
	)

	checks := []struct {
		desc string
		want string
	}{
		{"id frontmatter", "id: T009"},
		{"title frontmatter", `title: "Sample Task"`},
		{"type frontmatter", "type: feature"},
		{"sprint frontmatter", "sprint: sprint-01"},
		{"status frontmatter", "status: todo"},
		{"priority frontmatter", "priority: p1"},
		{"estimate frontmatter", "estimate: M"},
		{"depends_on", `depends_on: ["T001"]`},
		{"created date", "created: " + time.Now().Format("2006-01-02")},
		{"h1 heading", "# T009 Sample Task"},
		{"type-tags section", "## Type Tags"},
		{"new structure option", "New structure"},
		{"existing structure option", "Existing structure update"},
		{"verification option", "Measurement / verification / deployment"},
		{"mixed option", "Mixed"},
		{"purpose section", "## Purpose"},
		{"requirements section", "## Requirements"},
		{"done-criteria section", "## Done Criteria"},
		{"references section", "## References"},
		{"frontmatter delimiter", "---"},
	}

	for _, c := range checks {
		if !strings.Contains(result, c.want) {
			t.Errorf("[%s] expected %q in result\nresult:\n%s", c.desc, c.want, result)
		}
	}
}

// TestRenderTaskTemplate_backlog verifies that an empty sprint becomes "backlog".
func TestRenderTaskTemplate_backlog(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T010", "Backlog Task", "chore", "",
		"p2", "S", nil, "",
	)

	if !strings.Contains(result, "sprint: backlog") {
		t.Errorf("expected 'sprint: backlog' for empty sprint\nresult:\n%s", result)
	}
}

// TestRenderTaskTemplate_emptyDeps verifies that nil depends_on serialises to "[]".
func TestRenderTaskTemplate_emptyDeps(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T011", "No Deps", "docs", "sprint-02",
		"p3", "XS", nil, "",
	)

	if !strings.Contains(result, "depends_on: []") {
		t.Errorf("expected empty depends_on '[]'\nresult:\n%s", result)
	}
}

// TestRenderTaskTemplate_frontmatterDelimiter verifies the file starts with a
// frontmatter block.
func TestRenderTaskTemplate_frontmatterDelimiter(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T012", "Frontmatter Test", "feature", "sprint-01",
		"p0", "L", nil, "",
	)

	if !strings.HasPrefix(result, "---\n") {
		t.Errorf("file must start with '---\\n'")
	}

	count := strings.Count(result[:strings.Index(result, "\n# ")], "---")
	if count < 2 {
		t.Errorf("frontmatter close '---' missing\nresult:\n%s", result[:100])
	}
}

// TestRenderTaskTemplate_typeMatrix verifies that every supported type renders.
func TestRenderTaskTemplate_typeMatrix(t *testing.T) {
	types := []string{"feature", "bugfix", "hotfix", "refactor", "infra", "docs", "test", "chore"}
	for _, taskType := range types {
		result := tmpl.RenderTaskTemplate(
			"T100", "type test", taskType, "",
			"p2", "M", nil, "",
		)
		if !strings.Contains(result, "type: "+taskType) {
			t.Errorf("type=%s missing from template", taskType)
		}
	}
}

// TestRenderTaskTemplate_hotfixRollback verifies that a hotfix Task gets the
// ## Rollback section and other types do not.
func TestRenderTaskTemplate_hotfixRollback(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T999", "hotfix test", "hotfix", "",
		"p1", "S", nil, "",
	)
	if !strings.Contains(result, "## Rollback") {
		t.Errorf("hotfix type missing '## Rollback' section\nresult:\n%s", result)
	}

	featureResult := tmpl.RenderTaskTemplate(
		"T998", "feature test", "feature", "",
		"p1", "S", nil, "",
	)
	if strings.Contains(featureResult, "## Rollback") {
		t.Errorf("feature type must not include '## Rollback'")
	}
}

// TestRenderTaskTemplate_priorityMatrix verifies that all priorities + estimates render.
func TestRenderTaskTemplate_priorityMatrix(t *testing.T) {
	priorities := []string{"p0", "p1", "p2", "p3"}
	estimates := []string{"XS", "S", "M", "L", "XL"}
	for _, p := range priorities {
		for _, e := range estimates {
			result := tmpl.RenderTaskTemplate(
				"T200", "priority test", "feature", "",
				p, e, nil, "",
			)
			if !strings.Contains(result, "priority: "+p) {
				t.Errorf("priority=%s missing from template", p)
			}
			if !strings.Contains(result, "estimate: "+e) {
				t.Errorf("estimate=%s missing from template", e)
			}
		}
	}
}

// TestRenderTaskTemplate_resultSubHeadings verifies that the standard Task
// result section contains the three required sub-headings.
func TestRenderTaskTemplate_resultSubHeadings(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T578a", "result section", "feature", "",
		"p2", "M", nil, "",
	)
	required := []string{"## Result", "### Design Decisions", "### Artifacts", "### Verification"}
	for _, section := range required {
		if !strings.Contains(result, section) {
			t.Errorf("template missing section %q", section)
		}
	}
}

// TestRenderTaskTemplate_spikeResultTemplate verifies that a spike Task renders
// the spike-specific sub-headings instead of the standard sub-headings.
func TestRenderTaskTemplate_spikeResultTemplate(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T455t", "spike result template", "spike", "",
		"p3", "S", nil, "",
	)

	required := []string{
		"## Result",
		"### Subject",
		"### Method",
		"### Conclusion",
		"### Recommendation",
	}
	for _, section := range required {
		if !strings.Contains(result, section) {
			t.Errorf("spike template missing section %q\nresult:\n%s", section, result)
		}
	}

	notExpected := []string{
		"### Artifacts",
		"### Design Decisions",
	}
	for _, section := range notExpected {
		if strings.Contains(result, section) {
			t.Errorf("spike template must not include sub-heading %q\nresult:\n%s", section, result)
		}
	}

	featureResult := tmpl.RenderTaskTemplate(
		"T455f", "feature regression", "feature", "",
		"p3", "S", nil, "",
	)
	if strings.Contains(featureResult, "### Subject") {
		t.Errorf("feature type must not include spike-only '### Subject'\nresult:\n%s", featureResult)
	}
	if !strings.Contains(featureResult, "### Artifacts") {
		t.Errorf("feature type missing '### Artifacts' (regression)\nresult:\n%s", featureResult)
	}
}

// TestRenderTaskTemplate_multipleDeps verifies that multiple depends_on entries
// render correctly.
func TestRenderTaskTemplate_multipleDeps(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T300", "multi deps", "feature", "sprint-01",
		"p1", "M",
		[]string{"T001", "T002", "T003"},
		"",
	)

	if !strings.Contains(result, "T001") {
		t.Error("T001 missing from depends_on")
	}
	if !strings.Contains(result, "T002") {
		t.Error("T002 missing from depends_on")
	}
}

// TestRenderTaskTemplate_summaryPlaceholder verifies that an empty summary
// substitutes the placeholder text in the ## Summary section.
func TestRenderTaskTemplate_summaryPlaceholder(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T400", "summary placeholder test", "feature", "",
		"p2", "M", nil, "",
	)
	if !strings.Contains(result, "## Summary") {
		t.Error("## Summary section missing")
	}
	if !strings.Contains(result, "{One-line summary") {
		t.Errorf("expected placeholder text for empty summary\nresult:\n%s", result)
	}
}

// TestRenderTaskTemplate_summaryInjection verifies that a non-empty summary is
// injected into the ## Summary section and the placeholder is gone.
func TestRenderTaskTemplate_summaryInjection(t *testing.T) {
	summary := "Add an API endpoint — performance / 50% latency reduction is the success bar"
	result := tmpl.RenderTaskTemplate(
		"T401", "summary injection test", "feature", "",
		"p2", "M", nil, summary,
	)
	if !strings.Contains(result, "## Summary") {
		t.Error("## Summary section missing")
	}
	if !strings.Contains(result, summary) {
		t.Errorf("summary value missing from template\nresult:\n%s", result)
	}
	if strings.Contains(result, "{One-line summary") {
		t.Error("placeholder text must not remain when summary is injected")
	}
}

// TestRenderTaskTemplate_summaryPosition verifies that ## Summary comes
// immediately after ## Type Tags and before ## Purpose.
func TestRenderTaskTemplate_summaryPosition(t *testing.T) {
	result := tmpl.RenderTaskTemplate(
		"T402", "summary position test", "feature", "",
		"p2", "M", nil, "",
	)
	idxTypeTags := strings.Index(result, "## Type Tags")
	idxSummary := strings.Index(result, "## Summary")
	idxPurpose := strings.Index(result, "## Purpose")

	if idxTypeTags < 0 || idxSummary < 0 || idxPurpose < 0 {
		t.Fatalf("required sections missing — type tags=%d, summary=%d, purpose=%d",
			idxTypeTags, idxSummary, idxPurpose)
	}
	if idxTypeTags >= idxSummary {
		t.Errorf("## Summary appears before ## Type Tags (type tags=%d, summary=%d)", idxTypeTags, idxSummary)
	}
	if idxSummary >= idxPurpose {
		t.Errorf("## Purpose appears before ## Summary (summary=%d, purpose=%d)", idxSummary, idxPurpose)
	}
}
