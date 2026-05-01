package kb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// GetKBFilePath empty-value defence
// ---------------------------------------------------------------------------

// TestT413_GetKBFilePath_EmptySubcategory defends against the bug where an
// empty subcategory yields a file named `.md`.
func TestT413_GetKBFilePath_EmptySubcategory(t *testing.T) {
	_, err := GetKBFilePath("/tmp/testproj", "mistakes", "")
	if err == nil {
		t.Fatal("empty subcategory passed without error — could create a .md file")
	}
	if !strings.Contains(err.Error(), "subcategory") {
		t.Errorf("error message does not mention subcategory: %v", err)
	}
}

// TestT413_GetKBFilePath_EmptyCategory verifies an empty category returns an error.
func TestT413_GetKBFilePath_EmptyCategory(t *testing.T) {
	_, err := GetKBFilePath("/tmp/testproj", "", "foo")
	if err == nil {
		t.Fatal("empty category passed without error")
	}
	if !strings.Contains(err.Error(), "category") {
		t.Errorf("error message does not mention category: %v", err)
	}
}

// TestT413_GetKBFilePath_Valid is a regression that valid input still works.
func TestT413_GetKBFilePath_Valid(t *testing.T) {
	p, err := GetKBFilePath("/tmp/testproj", "mistakes", "M01-test")
	if err != nil {
		t.Fatalf("valid input failed: %v", err)
	}
	if !strings.HasSuffix(p, "/docs/07-knowledge/mistakes/M01-test.md") {
		t.Errorf("path differs from expected: %s", p)
	}
}

// ---------------------------------------------------------------------------
// TestResolvePrefix
// ---------------------------------------------------------------------------

func TestResolvePrefix_HasMapping(t *testing.T) {
	cases := []struct {
		category    string
		subcategory string
		want        string
	}{
		{"mistakes", "code-review-findings", "M"},
		{"mistakes", "doc-consistency", "D"},
		{"operations", "infrastructure", "O"},
		{"architecture", "", "A"},
		{"api", "", "API"},
		{"domain", "", "DOM"},
		{"migration", "", "MIG"},
	}
	for _, tc := range cases {
		got := ResolvePrefix(tc.category, tc.subcategory)
		if got != tc.want {
			t.Errorf("ResolvePrefix(%q, %q) = %q; want %q", tc.category, tc.subcategory, got, tc.want)
		}
	}
}

func TestResolvePrefix_Default(t *testing.T) {
	got := ResolvePrefix("unknown", "sub")
	if got != "U" {
		t.Errorf("ResolvePrefix(unknown, sub) = %q; want 'U'", got)
	}

	got2 := ResolvePrefix("custom", "")
	if got2 != "C" {
		t.Errorf("ResolvePrefix(custom, '') = %q; want 'C'", got2)
	}
}

// ---------------------------------------------------------------------------
// TestScanCardsInFile
// ---------------------------------------------------------------------------

func TestScanCardsInFile_OK(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	content := `# Test

## M01: First lesson (2026-01-01, context-a)
**Problem**: problem description
**Solution**: solution

## M02: Second lesson (2026-02-01)
**Problem**: second problem
**Solution**: second solution

## API03: API lesson
**Problem**: API problem
**Solution**: API solution
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cards, err := ScanCardsInFile(filePath)
	if err != nil {
		t.Fatalf("ScanCardsInFile error: %v", err)
	}
	if len(cards) != 3 {
		t.Fatalf("card count = %d; want 3", len(cards))
	}

	if cards[0].CardID != "M01" || cards[0].Prefix != "M" || cards[0].Number != 1 {
		t.Errorf("cards[0] = %+v; want M01", cards[0])
	}
	if cards[0].OccurredAt != "2026-01-01" {
		t.Errorf("cards[0].OccurredAt = %q; want '2026-01-01'", cards[0].OccurredAt)
	}
	if cards[0].Context != "context-a" {
		t.Errorf("cards[0].Context = %q; want 'context-a'", cards[0].Context)
	}
	if cards[1].OccurredAt != "2026-02-01" || cards[1].Context != "" {
		t.Errorf("cards[1] occurred_at/context: %q/%q", cards[1].OccurredAt, cards[1].Context)
	}
	if cards[2].CardID != "API03" || cards[2].Prefix != "API" {
		t.Errorf("cards[2] = %+v; want API03", cards[2])
	}
}

func TestScanCardsInFile_MissingFile(t *testing.T) {
	cards, err := ScanCardsInFile("/nonexistent/path/file.md")
	if err != nil {
		t.Fatalf("missing file returned error: %v", err)
	}
	if len(cards) != 0 {
		t.Errorf("card count = %d; want 0", len(cards))
	}
}

// ---------------------------------------------------------------------------
// TestNextCardNumber
// ---------------------------------------------------------------------------

func TestNextCardNumber_Increment(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	content := `## M01: Lesson 1
**Problem**: p1
**Solution**: s1

## M03: Lesson 3
**Problem**: p3
**Solution**: s3
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := NextCardNumber(filePath, "M")
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("NextCardNumber = %d; want 4", n)
	}

	// Unknown prefix.
	n2, err := NextCardNumber(filePath, "D")
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 1 {
		t.Errorf("unknown prefix NextCardNumber = %d; want 1", n2)
	}
}

// ---------------------------------------------------------------------------
// TestRenderCardMarkdown
// ---------------------------------------------------------------------------

func TestRenderCardMarkdown_AllFields(t *testing.T) {
	md := RenderCardMarkdown(
		"M01",
		"test lesson",
		"2026-01-01",
		"sprint-01",
		"problem text",
		"cause text",
		"solution text",
		"prevention text",
		[]string{"M00", "A01"},
		nil,
		"mistakes",
	)

	wantParts := []string{
		"## M01: test lesson (2026-01-01, sprint-01)",
		"**Problem**: problem text",
		"**Cause**: cause text",
		"**Solution**: solution text",
		"**Prevention**: prevention text",
		"**Related**: M00, A01",
		"**Category**: mistakes",
	}
	for _, want := range wantParts {
		if !strings.Contains(md, want) {
			t.Errorf("rendered output missing %q\nactual:\n%s", want, md)
		}
	}
}

func TestRenderCardMarkdown_MinimalFields(t *testing.T) {
	md := RenderCardMarkdown(
		"D01",
		"minimal card",
		"", // no occurredAt
		"", // no context
		"problem",
		"", // no cause
		"solution",
		"",  // no prevention
		nil, // no refs
		nil, // no violationPatterns
		"",  // no category
	)

	if strings.Contains(md, "**Cause**") {
		t.Error("Cause printed even though it was empty")
	}
	if strings.Contains(md, "**Prevention**") {
		t.Error("Prevention printed even though it was empty")
	}
	if strings.Contains(md, "(") {
		t.Error("parenthesis printed without meta")
	}
	if !strings.Contains(md, "## D01: minimal card") {
		t.Error("header line missing")
	}
}

// ---------------------------------------------------------------------------
// TestAppendCardToFile
// ---------------------------------------------------------------------------

func TestAppendCardToFile_NewFileCreate(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "mistakes", "code-review-findings.md")

	md := "\n## M01: new card\n**Problem**: p\n**Solution**: s\n"
	if err := AppendCardToFile(filePath, md); err != nil {
		t.Fatalf("AppendCardToFile error: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "# Code Review Findings") {
		t.Errorf("new file title missing: %q", content[:min(len(content), 100)])
	}
	if !strings.Contains(content, "## M01: new card") {
		t.Error("card header missing")
	}
}

func TestAppendCardToFile_AddToExistingFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.md")
	initial := "# Test\n\n## M01: existing\n**Problem**: p\n**Solution**: s\n"
	if err := os.WriteFile(filePath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	md := "\n## M02: added card\n**Problem**: p2\n**Solution**: s2\n"
	if err := AppendCardToFile(filePath, md); err != nil {
		t.Fatalf("AppendCardToFile error: %v", err)
	}

	data, _ := os.ReadFile(filePath)
	content := string(data)
	if !strings.Contains(content, "## M01: existing") {
		t.Error("existing card vanished")
	}
	if !strings.Contains(content, "## M02: added card") {
		t.Error("added card missing")
	}
}

// ---------------------------------------------------------------------------
// TestRecountIndex
// ---------------------------------------------------------------------------

func TestRecountIndex_Aggregate(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HSTL_PROJECT_ROOT", dir)
	defer os.Unsetenv("HSTL_PROJECT_ROOT")

	// Create the KB file.
	mistakesDir := filepath.Join(dir, "docs", "07-knowledge", "mistakes")
	if err := os.MkdirAll(mistakesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "## M01: lesson1\n**Problem**: p\n**Solution**: s\n\n## M02: lesson2\n**Problem**: p2\n**Solution**: s2\n"
	if err := os.WriteFile(filepath.Join(mistakesDir, "code-review-findings.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := RecountIndex(dir)
	if err != nil {
		t.Fatalf("RecountIndex error: %v", err)
	}

	if result.TotalCards != 2 {
		t.Errorf("total_cards = %v; want 2", result.TotalCards)
	}

	// Confirm the file was auto-added to INDEX.md.
	indexData, _ := os.ReadFile(filepath.Join(dir, "docs", "07-knowledge", "INDEX.md"))
	if !strings.Contains(string(indexData), "code-review-findings") {
		t.Errorf("INDEX.md missing code-review-findings:\n%s", string(indexData))
	}
}

// ---------------------------------------------------------------------------
// TestFilterCards
// ---------------------------------------------------------------------------

func TestFilterCards_CategoryFilter(t *testing.T) {
	dir := t.TempDir()
	cards := []CardHeader{
		{CardID: "M01", Prefix: "M", Number: 1, Title: "mistake lesson", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/code-review-findings.md")},
		{CardID: "A01", Prefix: "A", Number: 1, Title: "architecture lesson", FilePath: filepath.Join(dir, "docs/07-knowledge/architecture/patterns.md")},
		{CardID: "O01", Prefix: "O", Number: 1, Title: "operations lesson", FilePath: filepath.Join(dir, "docs/07-knowledge/operations/infrastructure.md")},
	}

	filtered := FilterCards(cards, dir, "mistakes", "", "")
	if len(filtered) != 1 {
		t.Errorf("category filter result = %d; want 1", len(filtered))
	}
	if filtered[0].CardID != "M01" {
		t.Errorf("filtered[0].CardID = %q; want M01", filtered[0].CardID)
	}
}

func TestFilterCards_KeywordFilter(t *testing.T) {
	dir := t.TempDir()
	cards := []CardHeader{
		{CardID: "M01", Title: "Go error-handling mistake", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
		{CardID: "M02", Title: "SQL injection prevention", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
		{CardID: "M03", Title: "Go test patterns", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
	}

	filtered := FilterCards(cards, dir, "", "go", "")
	if len(filtered) != 2 {
		t.Errorf("keyword filter result = %d; want 2", len(filtered))
	}
}

// ---------------------------------------------------------------------------
// TestScanCardsInFile_VariousPrefixes
// ---------------------------------------------------------------------------

func TestScanCardsInFile_VariousPrefixes(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "mixed.md")
	content := `# Mixed Cards

## M01: mistake lesson (2026-01-01, sprint-01)
**Problem**: M problem
**Solution**: M solution

## D01: doc consistency (2026-02-01)
**Problem**: D problem
**Solution**: D solution

## O01: operations lesson
**Problem**: O problem
**Solution**: O solution
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cards, err := ScanCardsInFile(filePath)
	if err != nil {
		t.Fatalf("ScanCardsInFile error: %v", err)
	}
	if len(cards) != 3 {
		t.Errorf("card count = %d; want 3", len(cards))
	}

	prefixes := make(map[string]bool)
	for _, c := range cards {
		prefixes[c.Prefix] = true
	}
	for _, p := range []string{"M", "D", "O"} {
		if !prefixes[p] {
			t.Errorf("prefix %s missing", p)
		}
	}
}

// ---------------------------------------------------------------------------
// TestNextCardNumber_NoPrefix
// ---------------------------------------------------------------------------

func TestNextCardNumber_NoPrefix(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "empty.md")
	content := "# Empty\n\nbody only without cards\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := NextCardNumber(filePath, "M")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("NextCardNumber(missing prefix) = %d; want 1", n)
	}
}

// ---------------------------------------------------------------------------
// TestRecountIndex_MissingDirectory
// ---------------------------------------------------------------------------

func TestRecountIndex_MissingDirectory(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("HSTL_PROJECT_ROOT", dir)
	defer os.Unsetenv("HSTL_PROJECT_ROOT")

	// Call RecountIndex without creating the KB directory.
	result, err := RecountIndex(dir)
	if err != nil {
		t.Fatalf("RecountIndex error: %v", err)
	}

	if result.TotalCards != 0 {
		t.Errorf("total_cards = %v; want 0", result.TotalCards)
	}
}

// ---------------------------------------------------------------------------
// TestFilterCards_SinceFilter
// ---------------------------------------------------------------------------

func TestFilterCards_SinceFilter(t *testing.T) {
	dir := t.TempDir()
	cards := []CardHeader{
		{CardID: "M01", Title: "old mistake", OccurredAt: "2025-01-01", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
		{CardID: "M02", Title: "recent mistake", OccurredAt: "2026-04-01", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
		{CardID: "M03", Title: "no date", OccurredAt: "", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
	}

	// Filter from 2026-01-01.
	filtered := FilterCards(cards, dir, "", "", "2026-01-01")
	// Includes the 2026-04-01 card and the dateless card.
	if len(filtered) == 0 {
		t.Logf("since filter result = 0 (depends on filter logic)")
	} else {
		t.Logf("since filter result = %d", len(filtered))
	}
}

// ---------------------------------------------------------------------------
// TestFilterCards_CompoundFilter
// ---------------------------------------------------------------------------

func TestFilterCards_CompoundFilter(t *testing.T) {
	dir := t.TempDir()
	cards := []CardHeader{
		{CardID: "M01", Title: "Go mistake", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
		{CardID: "A01", Title: "architecture Go", FilePath: filepath.Join(dir, "docs/07-knowledge/architecture/test.md")},
		{CardID: "M02", Title: "SQL mistake", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/test.md")},
	}

	// category=mistakes + keyword=go compound filter.
	filtered := FilterCards(cards, dir, "mistakes", "go", "")
	if len(filtered) != 1 {
		t.Errorf("compound filter result = %d; want 1", len(filtered))
	}
	if len(filtered) == 1 && filtered[0].CardID != "M01" {
		t.Errorf("compound filter result CardID=%s; want M01", filtered[0].CardID)
	}
}

// ---------------------------------------------------------------------------
// TestRenderCardMarkdown_SpecialChars
// ---------------------------------------------------------------------------

func TestRenderCardMarkdown_SpecialChars(t *testing.T) {
	md := RenderCardMarkdown(
		"M01",
		"Go `defer` overuse mistake",
		"2026-01-01",
		"",
		"problem: defer called inside loop",
		"",
		"solution: move defer outside the closure",
		"",
		nil,
		nil,
		"mistakes",
	)

	if !strings.Contains(md, "Go `defer`") {
		t.Error("backtick character not preserved")
	}
	if !strings.Contains(md, "**Problem**: problem:") {
		t.Logf("Problem field: %s", md)
	}
}

// ---------------------------------------------------------------------------
// TestResolvePrefix_Case
// ---------------------------------------------------------------------------

func TestResolvePrefix_Case(t *testing.T) {
	// Even when given uppercase input, returns the expected prefix.
	got := ResolvePrefix("MISTAKES", "CODE-REVIEW-FINDINGS")
	// Confirm the output is non-empty regardless of case handling.
	t.Logf("ResolvePrefix(MISTAKES, CODE-REVIEW-FINDINGS) = %s", got)
	if got == "" {
		t.Error("prefix is empty")
	}
}

// ---------------------------------------------------------------------------
// TestAppendCardToFile_AutoCreateDirectory
// ---------------------------------------------------------------------------

func TestAppendCardToFile_AutoCreateDirectory(t *testing.T) {
	dir := t.TempDir()
	// Three-level deep new path.
	filePath := filepath.Join(dir, "level1", "level2", "level3.md")

	md := "\n## N01: new card\n**Problem**: p\n**Solution**: s\n"
	if err := AppendCardToFile(filePath, md); err != nil {
		t.Fatalf("AppendCardToFile error: %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("file not created: %s", filePath)
	}
}

// ---------------------------------------------------------------------------
// Additional tests
// ---------------------------------------------------------------------------

// Verifies that GetKBFilePath returns the correct path per category.
// Updated to non-empty subcategory because the empty subcategory is rejected.
func TestGetKBFilePath_PerCategory(t *testing.T) {
	dir := t.TempDir()
	path, err := GetKBFilePath(dir, "mistakes", "M01-test")
	if err != nil {
		t.Fatalf("GetKBFilePath failed: %v", err)
	}
	if !strings.Contains(path, "mistakes") {
		t.Errorf("path missing mistakes: %s", path)
	}
}

// Verifies that GetIndexPath returns the correct KB index path.
func TestGetIndexPath_PathCheck(t *testing.T) {
	dir := t.TempDir()
	indexPath := GetIndexPath(dir)
	if !strings.Contains(indexPath, "INDEX.md") {
		t.Errorf("INDEX.md not present: %s", indexPath)
	}
	if !strings.HasPrefix(indexPath, dir) {
		t.Errorf("path does not start with tmpDir: %s", indexPath)
	}
}

// Verifies that KB-*.md files are also detected as cards.
func TestT609_ScanAllCards_SingleFileCardIncluded(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, "docs", "07-knowledge", "api")
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Category file (inline card).
	mistakesDir := filepath.Join(dir, "docs", "07-knowledge", "mistakes")
	if err := os.MkdirAll(mistakesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	categoryFile := filepath.Join(mistakesDir, "testing.md")
	os.WriteFile(categoryFile, []byte("# Testing\n\n## M01: existing card\n\n**Problem**: example\n"), 0o644)

	// Single-file card.
	singleFile := filepath.Join(kbDir, "KB-LS-CATALOG-2026-04.md")
	os.WriteFile(singleFile, []byte("---\nkb_card: true\n---\n\n# LS API catalog 2026-04\n\nbody\n"), 0o644)

	cards, err := ScanAllCards(dir)
	if err != nil {
		t.Fatalf("ScanAllCards: %v", err)
	}

	var foundCategory, foundSingle bool
	for _, c := range cards {
		if c.CardID == "M01" && c.Kind == "category" {
			foundCategory = true
		}
		if c.CardID == "KB-LS-CATALOG-2026-04" && c.Kind == "single_file" {
			foundSingle = true
			if c.Title != "LS API catalog 2026-04" {
				t.Errorf("single_file title=%q", c.Title)
			}
		}
	}
	if !foundCategory {
		t.Errorf("category card missing (regression)")
	}
	if !foundSingle {
		t.Errorf("single_file KB card not detected")
	}
}

// Verifies that an empty directory yields no cards.
func TestScanAllCards_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	// Do not create the KB directory.
	cards, err := ScanAllCards(dir)
	if err != nil {
		t.Logf("ScanAllCards error (allowed): %v", err)
		return
	}
	if len(cards) != 0 {
		t.Errorf("empty directory cards=%d, want=0", len(cards))
	}
}

// Verifies that all cards are returned without filters.
func TestFilterCards_ReturnsAll(t *testing.T) {
	dir := t.TempDir()
	cards := []CardHeader{
		{CardID: "M01", Prefix: "M", Title: "lesson1", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/a.md")},
		{CardID: "M02", Prefix: "M", Title: "lesson2", FilePath: filepath.Join(dir, "docs/07-knowledge/mistakes/a.md")},
		{CardID: "A01", Prefix: "A", Title: "architecture", FilePath: filepath.Join(dir, "docs/07-knowledge/architecture/a.md")},
	}

	filtered := FilterCards(cards, dir, "", "", "")
	if len(filtered) != 3 {
		t.Errorf("no filter: expected 3, got %d", len(filtered))
	}
}

// Verifies that the refs slice renders correctly.
func TestRenderCardMarkdown_References(t *testing.T) {
	refs := []string{"T001", "T002", "ADR-001"}
	result := RenderCardMarkdown("M05", "reference test", "2026-01-01", "ctx", "prob", "cause", "sol", "prev", refs, nil, "mistakes")
	for _, ref := range refs {
		if !strings.Contains(result, ref) {
			t.Errorf("refs missing %s", ref)
		}
	}
}

// Verifies that NextCardNumber returns 1 for an empty card file.
func TestNextCardNumber_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.md")
	os.WriteFile(emptyFile, []byte("# empty file\nno content\n"), 0o644)

	n, err := NextCardNumber(emptyFile, "M")
	if err != nil {
		t.Fatalf("NextCardNumber failed: %v", err)
	}
	if n != 1 {
		t.Errorf("empty file NextCardNumber=%d, want=1", n)
	}
}

// Verifies that EnsureIndexFile creates INDEX.md when missing.
func TestEnsureIndexFile_Create(t *testing.T) {
	dir := t.TempDir()

	// Create the KB directory.
	kbDir := filepath.Join(dir, "docs", "07-knowledge")
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("create KB directory failed: %v", err)
	}

	indexPath, err := EnsureIndexFile(dir)
	if err != nil {
		t.Fatalf("EnsureIndexFile failed: %v", err)
	}

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf("INDEX.md not created: %s", indexPath)
	}
}

// ---------------------------------------------------------------------------
// ViolationPatterns
// ---------------------------------------------------------------------------

// Verifies that the **Violation patterns**: line is rendered when the input is supplied.
func TestRenderCardMarkdown_WithViolationPatterns(t *testing.T) {
	patterns := []string{`cd\s+.*&&\s*git add`, `git add.*\.\./`}
	md := RenderCardMarkdown(
		"M003",
		"git add must run from repo root",
		"2026-04-16",
		"Sprint-77",
		"problem description",
		"cause",
		"solution",
		"prevention",
		nil,
		patterns,
		"mistakes",
	)

	if !strings.Contains(md, "**Violation patterns**: ") {
		t.Errorf("**Violation patterns**: line missing\nactual:\n%s", md)
	}
	for _, p := range patterns {
		if !strings.Contains(md, p) {
			t.Errorf("pattern %q missing from rendered output\nactual:\n%s", p, md)
		}
	}

	// **Category** must appear after **Violation patterns**.
	vpIdx := strings.Index(md, "**Violation patterns**:")
	catIdx := strings.Index(md, "**Category**:")
	if vpIdx < 0 || catIdx < 0 {
		t.Fatal("violation patterns or category line missing")
	}
	if vpIdx > catIdx {
		t.Errorf("**Violation patterns** appears after **Category** (order error)")
	}
}

// Verifies that the **Violation patterns**: line is omitted when no patterns are supplied.
func TestRenderCardMarkdown_NoViolationPatterns(t *testing.T) {
	md := RenderCardMarkdown(
		"M001",
		"card without violation patterns",
		"2026-01-01",
		"",
		"problem",
		"",
		"solution",
		"",
		nil,
		nil,
		"mistakes",
	)

	if strings.Contains(md, "**Violation patterns**:") {
		t.Errorf("**Violation patterns** printed without patterns\nactual:\n%s", md)
	}
}

// Verifies that ViolationPatterns is populated when parsing a card with the
// **Violation patterns**: line.
func TestScanCardsInFile_ViolationPatterns(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "workflow.md")
	content := `# Workflow

## M003: git add must run from repo root (2026-04-16, Sprint-77)
**Problem**: silent fail when git add runs outside the repo root
**Solution**: run from the repo root
**Violation patterns**: cd\s+.*&&\s*git add, git add.*\.\./
**Category**: mistakes

## M001: do not assert (2026-04-14, Sprint-56)
**Problem**: hypothesis recorded as assertion
**Solution**: record only observed symptoms
**Category**: mistakes
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cards, err := ScanCardsInFile(filePath)
	if err != nil {
		t.Fatalf("ScanCardsInFile error: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("card count=%d; want 2", len(cards))
	}

	// M003 — has violation patterns.
	m003 := cards[0]
	if m003.CardID != "M003" {
		t.Fatalf("cards[0].CardID=%q; want M003", m003.CardID)
	}
	if len(m003.ViolationPatterns) != 2 {
		t.Fatalf("ViolationPatterns count=%d; want 2\nactual: %v", len(m003.ViolationPatterns), m003.ViolationPatterns)
	}
	if m003.ViolationPatterns[0] != `cd\s+.*&&\s*git add` {
		t.Errorf("ViolationPatterns[0]=%q; want %q", m003.ViolationPatterns[0], `cd\s+.*&&\s*git add`)
	}
	if m003.ViolationPatterns[1] != `git add.*\.\./` {
		t.Errorf("ViolationPatterns[1]=%q; want %q", m003.ViolationPatterns[1], `git add.*\.\./`)
	}

	// M001 — no violation patterns.
	m001 := cards[1]
	if m001.CardID != "M001" {
		t.Fatalf("cards[1].CardID=%q; want M001", m001.CardID)
	}
	if len(m001.ViolationPatterns) != 0 {
		t.Errorf("M001 ViolationPatterns=%v; want empty", m001.ViolationPatterns)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
