// Package kb manages Knowledge Base card markdown files.
// Maintains category-specific markdown files and INDEX.md under
// docs/07-knowledge/.
package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
)

// ---------------------------------------------------------------------------
// Constants and prefix mapping
// ---------------------------------------------------------------------------

const KBRoot = "docs/07-knowledge"
const IndexFilename = "INDEX.md"

// _violationPatternsPrefix is the bold-header prefix used to identify the
// violation-pattern line inside a card body.
const _violationPatternsPrefix = "**Violation patterns**: "

// prefixKey is the internal key for a (category, subcategory) pair.
type prefixKey struct {
	category    string
	subcategory string // empty string = default for this category
}

// _prefixMap mirrors the same mapping used in Python.
var _prefixMap = map[prefixKey]string{
	{"mistakes", "code-review-findings"}: "M",
	{"mistakes", "doc-consistency"}:      "D",
	{"operations", "infrastructure"}:     "O",
	{"architecture", ""}:                 "A",
	{"api", ""}:                          "API",
	{"domain", ""}:                       "DOM",
	{"migration", ""}:                    "MIG",
}

// ---------------------------------------------------------------------------
// Card-header regular expressions
// ---------------------------------------------------------------------------

// _cardHeaderRE captures headers in the form "## M27: Title (2026-04-05, context)".
var _cardHeaderRE = regexp.MustCompile(
	`(?m)^##\s+([A-Z]+)(\d+):\s*(.+?)(?:\s*\((\d{4}-\d{2}-\d{2})(?:,\s*(.+?))?\))?\s*$`,
)

// _indexRowRE captures rows in the INDEX.md table.
var _indexRowRE = regexp.MustCompile(
	`(?m)^\|\s*([^|]+?)\s*\|\s*\[([^\]]+)\]\(([^)]+)\)\s*\|\s*(\d+)\s*\|`,
)

// _indexTotalRE captures the total-card-count line in INDEX.md.
var _indexTotalRE = regexp.MustCompile(
	`(?m)^\*\*Total cards\*\*:\s*\d+.*$`,
)

// _countReplaceRE rewrites the card-count column of an INDEX.md table row.
var _countReplaceRE = regexp.MustCompile(`(\|\s*\[[^\]]+\]\([^)]+\)\s*\|\s*)\d+(\s*\|)`)

// ---------------------------------------------------------------------------
// CardHeader
// ---------------------------------------------------------------------------

// CardHeader holds the parsed information from a card header.
type CardHeader struct {
	CardID            string // "M27" (category card) or "KB-LS-CATALOG-2026-04" (single-file)
	Prefix            string // "M" (category) or "KB" (single-file)
	Number            int    // 27 for category cards; 0 for single-file cards
	Title             string
	OccurredAt        string // "" if none
	Context           string // "" if none
	FilePath          string
	Kind              string   // "category" (default) or "single_file"
	ViolationPatterns []string // parsed from the **Violation patterns**: line in the card body
}

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// ResolvePrefix returns the card-ID prefix for a (category, subcategory)
// pair. Falls back to the uppercase first letter of the category when no
// mapping exists.
func ResolvePrefix(category, subcategory string) string {
	if v, ok := _prefixMap[prefixKey{category, subcategory}]; ok {
		return v
	}
	if v, ok := _prefixMap[prefixKey{category, ""}]; ok {
		return v
	}
	if len(category) > 0 {
		return strings.ToUpper(string([]rune(category)[0]))
	}
	return "C"
}

// GetKBFilePath returns the docs/07-knowledge/{category}/{subcategory}.md path.
// Returns an error when category or subcategory contains a path-escape sequence.
//
// Defends against the bug where a missing subcategory yields a `.md`-only
// filename — the empty value is rejected with an error.
func GetKBFilePath(projectRoot, category, subcategory string) (string, error) {
	if strings.TrimSpace(category) == "" {
		return "", fmt.Errorf("category is empty")
	}
	if strings.TrimSpace(subcategory) == "" {
		return "", fmt.Errorf("subcategory is empty (prevents creating a filename of %q)", ".md")
	}
	p := filepath.Join(projectRoot, KBRoot, category, subcategory+".md")
	rel, err := filepath.Rel(projectRoot, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid KB path: category=%q subcategory=%q", category, subcategory)
	}
	return p, nil
}

// GetIndexPath returns the docs/07-knowledge/INDEX.md path.
func GetIndexPath(projectRoot string) string {
	return filepath.Join(projectRoot, KBRoot, IndexFilename)
}

// ---------------------------------------------------------------------------
// Card scanning
// ---------------------------------------------------------------------------

// ScanCardsInFile scans card headers in a single file and returns the
// CardHeader slice. Returns an empty slice when the file is missing.
// Each card section is parsed for its **Violation patterns**: line and
// the result is stored in ViolationPatterns.
func ScanCardsInFile(filePath string) ([]CardHeader, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []CardHeader{}, nil
		}
		return nil, fmt.Errorf("read file (%s): %w", filePath, err)
	}

	// Collect header match positions and indices to determine section ranges.
	headerIdxs := _cardHeaderRE.FindAllSubmatchIndex(data, -1)
	matches := _cardHeaderRE.FindAllSubmatch(data, -1)
	cards := make([]CardHeader, 0, len(matches))

	for i, m := range matches {
		prefix := string(m[1])
		numberStr := string(m[2])
		title := strings.TrimSpace(string(m[3]))
		occurredAt := strings.TrimSpace(string(m[4]))
		context := strings.TrimSpace(string(m[5]))

		// Determine the body range of this card section
		// (current header end -> next header start or EOF).
		sectionStart := headerIdxs[i][1]
		sectionEnd := len(data)
		if i+1 < len(headerIdxs) {
			sectionEnd = headerIdxs[i+1][0]
		}
		sectionBody := string(data[sectionStart:sectionEnd])

		violationPatterns := parseViolationPatterns(sectionBody)

		number, _ := strconv.Atoi(numberStr)
		cards = append(cards, CardHeader{
			CardID:            prefix + numberStr,
			Prefix:            prefix,
			Number:            number,
			Title:             title,
			OccurredAt:        occurredAt,
			Context:           context,
			FilePath:          filePath,
			Kind:              "category",
			ViolationPatterns: violationPatterns,
		})
	}
	return cards, nil
}

// parseViolationPatterns finds the **Violation patterns**: line in the card
// section body and returns the comma-separated patterns.
// Returns nil when no patterns are present.
func parseViolationPatterns(sectionBody string) []string {
	for _, line := range strings.Split(sectionBody, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, _violationPatternsPrefix) {
			raw := strings.TrimPrefix(trimmed, _violationPatternsPrefix)
			var patterns []string
			for _, p := range strings.Split(raw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					patterns = append(patterns, p)
				}
			}
			return patterns
		}
	}
	return nil
}

// ScanAllCards scans every .md file under docs/07-knowledge/ and returns
// the card list. Includes both `## M01`/`## A01`-style cards inside
// category files and single-file KB cards (filenames matching `KB-*.md`).
func ScanAllCards(projectRoot string) ([]CardHeader, error) {
	kbRoot := filepath.Join(projectRoot, KBRoot)
	if _, err := os.Stat(kbRoot); os.IsNotExist(err) {
		return []CardHeader{}, nil
	}

	var allCards []CardHeader
	err := filepath.Walk(kbRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if info.Name() == IndexFilename {
			return nil
		}
		// Detect single-file KB cards (filename starts with `KB-`).
		if strings.HasPrefix(info.Name(), "KB-") {
			if card, ok := parseSingleFileCard(path, info.Name()); ok {
				allCards = append(allCards, card)
				return nil
			}
		}
		cards, err := ScanCardsInFile(path)
		if err != nil {
			return err
		}
		allCards = append(allCards, cards...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("KB scan: %w", err)
	}
	return allCards, nil
}

// parseSingleFileCard converts a `KB-*.md` single-file card into a CardHeader.
// - CardID: filename without the `.md` suffix (e.g. `KB-LS-CATALOG-2026-04`)
// - Prefix: "KB"
// - Title: the first `# ` heading or CardID (fallback)
// Returns false on read failure so the caller can fall back to the category path.
func parseSingleFileCard(filePath, fileName string) (CardHeader, bool) {
	cardID := strings.TrimSuffix(fileName, ".md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return CardHeader{}, false
	}
	title := cardID
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			break
		}
	}
	return CardHeader{
		CardID:   cardID,
		Prefix:   "KB",
		Number:   0,
		Title:    title,
		FilePath: filePath,
		Kind:     "single_file",
	}, true
}

// NextCardNumber returns the next number for the given prefix in a file.
// Returns 1 when the file is missing or no matching card exists.
func NextCardNumber(filePath, prefix string) (int, error) {
	cards, err := ScanCardsInFile(filePath)
	if err != nil {
		return 0, err
	}
	maxNum := 0
	for _, c := range cards {
		if c.Prefix == prefix && c.Number > maxNum {
			maxNum = c.Number
		}
	}
	return maxNum + 1, nil
}

// ---------------------------------------------------------------------------
// Card rendering
// ---------------------------------------------------------------------------

// RenderCardMarkdown produces the standard hstl KB card markdown.
//
// Format:
//
//	\n## {cardID}: {title}{meta_suffix}\n**Problem**: {problem}\n...
//
// When violationPatterns is non-empty, the **Violation patterns**: line is
// inserted directly before the **Category** line.
func RenderCardMarkdown(cardID, title, occurredAt, context, problem, cause, solution, prevention string, refs []string, violationPatterns []string, category string) string {
	var metaParts []string
	if occurredAt != "" {
		metaParts = append(metaParts, occurredAt)
	}
	if context != "" {
		metaParts = append(metaParts, context)
	}

	metaSuffix := ""
	if len(metaParts) > 0 {
		metaSuffix = " (" + strings.Join(metaParts, ", ") + ")"
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("## %s: %s%s", cardID, title, metaSuffix))
	lines = append(lines, fmt.Sprintf("**Problem**: %s", problem))

	if cause != "" {
		lines = append(lines, fmt.Sprintf("**Cause**: %s", cause))
	}

	lines = append(lines, fmt.Sprintf("**Solution**: %s", solution))

	if prevention != "" {
		lines = append(lines, fmt.Sprintf("**Prevention**: %s", prevention))
	}

	if len(refs) > 0 {
		lines = append(lines, fmt.Sprintf("**Related**: %s", strings.Join(refs, ", ")))
	}

	if len(violationPatterns) > 0 {
		lines = append(lines, fmt.Sprintf("%s%s", _violationPatternsPrefix, strings.Join(violationPatterns, ", ")))
	}

	if category != "" {
		lines = append(lines, fmt.Sprintf("**Category**: %s", category))
	}

	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Card append
// ---------------------------------------------------------------------------

// AppendCardToFile appends card markdown to the end of a file.
// Creates the directory and a title (# {stem}) header automatically when
// the file is missing.
func AppendCardToFile(filePath, cardMarkdown string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// New file — auto-generate the title section.
		stem := strings.TrimSuffix(filepath.Base(filePath), ".md")
		title := toTitle(strings.ReplaceAll(stem, "-", " "))
		initial := fmt.Sprintf("# %s\n\n", title)
		return os.WriteFile(filePath, []byte(initial+cardMarkdown), 0o644)
	}

	existing, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	content := string(existing)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(filePath, []byte(content+cardMarkdown), 0o644)
}

// toTitle uppercases the first character of every word.
func toTitle(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// ---------------------------------------------------------------------------
// INDEX.md management
// ---------------------------------------------------------------------------

// EnsureIndexFile creates INDEX.md from a default template when it is missing.
// Returns the INDEX.md path.
func EnsureIndexFile(projectRoot string) (string, error) {
	indexPath := GetIndexPath(projectRoot)
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
			return "", fmt.Errorf("create KB directory: %w", err)
		}
		content := "# Knowledge Base\n\n" +
			"Records lessons accumulated during project development by category.\n\n" +
			"## Categories\n\n" +
			"| Category | File | Cards | Targets |\n" +
			"|----------|------|-------|---------|\n" +
			"\n" +
			"**Total cards**: 0\n"
		if err := os.WriteFile(indexPath, []byte(content), 0o644); err != nil {
			return "", fmt.Errorf("create INDEX.md: %w", err)
		}
	}
	return indexPath, nil
}

// RecountIndex rescans docs/07-knowledge/ and updates the card-count
// column and total-card line in INDEX.md.
func RecountIndex(projectRoot string) (*RecountResult, error) {
	indexPath, err := EnsureIndexFile(projectRoot)
	if err != nil {
		return nil, err
	}
	kbRoot := filepath.Join(projectRoot, KBRoot)

	// 1. Collect cards per file.
	fileCardCounts := make(map[string]int)
	err = filepath.Walk(kbRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if info.Name() == IndexFilename {
			return nil
		}
		cards, err := ScanCardsInFile(path)
		if err != nil {
			return err
		}
		if len(cards) > 0 {
			rel, err := filepath.Rel(kbRoot, path)
			if err != nil {
				return err
			}
			fileCardCounts[filepath.ToSlash(rel)] = len(cards)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("KB scan: %w", err)
	}

	totalCards := 0
	for _, cnt := range fileCardCounts {
		totalCards += cnt
	}

	// 2. Read INDEX.md and rewrite the card-count fields.
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("read INDEX.md: %w", err)
	}

	lines := splitLines(string(data))
	filesUpdated := []string{}
	rowsFound := make(map[string]bool)

	// Match each row against _indexRowRE.
	for i, line := range lines {
		m := _indexRowRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// m[1]=category, m[2]=display, m[3]=rel_path, m[4]=old_count
		relPath := m[3]
		rowsFound[relPath] = true

		newCount := fileCardCounts[relPath]
		oldCount, _ := strconv.Atoi(m[4])
		if newCount != oldCount {
			updatedLine := _countReplaceRE.ReplaceAllString(line, fmt.Sprintf("${1}%d${2}", newCount))
			lines[i] = updatedLine
			filesUpdated = append(filesUpdated, relPath)
		}
	}

	// 3. Refresh the total-card line.
	today := time.Now().Format("2006-01-02")
	totalLine := fmt.Sprintf("**Total cards**: %d (as of %s)", totalCards, today)
	totalReplaced := false
	for i, line := range lines {
		if _indexTotalRE.MatchString(line) {
			lines[i] = totalLine
			totalReplaced = true
			break
		}
	}
	if !totalReplaced {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, totalLine)
	}

	// 4. Auto-add files missing from the INDEX.
	var missingInIndex []string
	for relPath := range fileCardCounts {
		if !rowsFound[relPath] {
			missingInIndex = append(missingInIndex, relPath)
		}
	}

	addedToIndex := []string{}
	if len(missingInIndex) > 0 {
		// Find the position of the last table row.
		lastTableRowIdx := -1
		for i, line := range lines {
			if _indexRowRE.MatchString(line) {
				lastTableRowIdx = i
			}
		}

		for _, relPath := range sortedStrings(missingInIndex) {
			parts := strings.Split(filepath.ToSlash(relPath), "/")
			category := "uncategorized"
			if len(parts) > 1 {
				category = parts[0]
			}
			displayName := strings.TrimSuffix(filepath.Base(relPath), ".md")

			// Use the first card title as the description.
			description := displayName
			fullPath := filepath.Join(kbRoot, filepath.FromSlash(relPath))
			cards, _ := ScanCardsInFile(fullPath)
			if len(cards) > 0 {
				description = cards[0].Title
			}

			count := fileCardCounts[relPath]
			newRow := fmt.Sprintf("| %s | [%s](%s) | %d | %s |", category, displayName, relPath, count, description)

			if lastTableRowIdx >= 0 {
				lastTableRowIdx++
				newLines := make([]string, 0, len(lines)+1)
				newLines = append(newLines, lines[:lastTableRowIdx]...)
				newLines = append(newLines, newRow)
				newLines = append(newLines, lines[lastTableRowIdx:]...)
				lines = newLines
			} else {
				// Insert after the divider (|---).
				inserted := false
				for i, line := range lines {
					if strings.HasPrefix(strings.TrimSpace(line), "|---") {
						newLines := make([]string, 0, len(lines)+1)
						newLines = append(newLines, lines[:i+1]...)
						newLines = append(newLines, newRow)
						newLines = append(newLines, lines[i+1:]...)
						lines = newLines
						lastTableRowIdx = i + 1
						inserted = true
						break
					}
				}
				if !inserted {
					lines = append(lines, newRow)
					lastTableRowIdx = len(lines) - 1
				}
			}
			addedToIndex = append(addedToIndex, relPath)
		}

		// Write back after the inserts.
		if err := os.WriteFile(indexPath, []byte(joinLines(lines)), 0o644); err != nil {
			return nil, fmt.Errorf("write INDEX.md: %w", err)
		}
	} else {
		// Persist the line updates even when no rows were added.
		if err := os.WriteFile(indexPath, []byte(joinLines(lines)), 0o644); err != nil {
			return nil, fmt.Errorf("write INDEX.md: %w", err)
		}
	}

	return &RecountResult{
		TotalCards:     totalCards,
		FilesScanned:   len(fileCardCounts),
		FilesUpdated:   filesUpdated,
		MissingInIndex: missingInIndex,
		AddedToIndex:   addedToIndex,
	}, nil
}

// UpdateIndexSummary appends a new card summary to the summary column
// (the 4th column) of the matching file row in INDEX.md.
// Returns (updated bool, error).
func UpdateIndexSummary(projectRoot, relPath, cardID, title string) (bool, error) {
	indexPath, err := EnsureIndexFile(projectRoot)
	if err != nil {
		return false, err
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return false, fmt.Errorf("read INDEX.md: %w", err)
	}

	lines := splitLines(string(data))
	updated := false

	for i, line := range lines {
		m := _indexRowRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rowPath := m[3]
		if rowPath != relPath {
			continue
		}

		// Append the summary just before the row's last `|`.
		// Row format: "| ... | ... | N | existing summary |".
		parts := strings.Split(strings.TrimSuffix(line, "\n"), "|")
		if len(parts) >= 5 {
			// parts[4] = " existing summary " (last column)
			summary := strings.TrimRight(parts[4], " ")
			newEntry := fmt.Sprintf(" + **%s(%s)**", title, cardID)
			parts[4] = summary + newEntry + " "
			lines[i] = strings.Join(parts, "|")
			updated = true
		}
		break
	}

	if updated {
		if err := os.WriteFile(indexPath, []byte(joinLines(lines)), 0o644); err != nil {
			return false, fmt.Errorf("write INDEX.md: %w", err)
		}
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// Filter utilities
// ---------------------------------------------------------------------------

// FilterCards filters a card list by category, keyword, and date.
func FilterCards(cards []CardHeader, projectRoot, category, keyword, since string) []CardHeader {
	kbRoot := filepath.Join(projectRoot, KBRoot)
	var result []CardHeader

	for _, card := range cards {
		if category != "" {
			rel, err := filepath.Rel(kbRoot, card.FilePath)
			if err != nil {
				continue
			}
			parts := strings.SplitN(filepath.ToSlash(rel), "/", 2)
			if parts[0] != category {
				continue
			}
		}
		if keyword != "" && !strings.Contains(strings.ToLower(card.Title), strings.ToLower(keyword)) {
			continue
		}
		if since != "" && card.OccurredAt != "" && card.OccurredAt < since {
			continue
		}
		result = append(result, card)
	}
	return result
}

// ---------------------------------------------------------------------------
// Internal utilities
// ---------------------------------------------------------------------------

// splitLines splits a string into lines.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// Drop the trailing empty entry (handles end-of-file newline).
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// joinLines joins lines into a single string.
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// sortedStrings returns a sorted copy of the string slice.
func sortedStrings(ss []string) []string {
	sorted := make([]string, len(ss))
	copy(sorted, ss)
	// Simple bubble sort (small data).
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

// GetProjectRootForKB returns the project root for the KB service.
// Delegates to fileutil.GetProjectRoot().
func GetProjectRootForKB() string {
	return fileutil.GetProjectRoot()
}
