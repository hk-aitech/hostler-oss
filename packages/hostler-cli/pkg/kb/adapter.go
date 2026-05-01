// Package kb — KBStore adapter.
//
// ADR-001 §4.2 Phase C. Wraps the existing pkg/kb globals (GetKBFilePath,
// ResolvePrefix, NextCardNumber, RenderCardMarkdown, AppendCardToFile,
// ScanAllCards, FilterCards, RecountIndex) so they satisfy the
// ports.KBStore interface. In Phase D the `hstl kb` CLI commands route
// through this adapter.
package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// KBFSStore is the KBStore adapter backed by the docs/07-knowledge/ filesystem.
type KBFSStore struct{}

// NewKBFSStore constructs the default fs-based KBStore.
func NewKBFSStore() *KBFSStore { return &KBFSStore{} }

// CreateCard adds a card to the category/subcategory file and returns the
// issued metadata. Consolidates the existing CLI `kb create` flow into a
// port method. Accepts the full input (context/cause/prevention/refs/
// violationPatterns).
func (KBFSStore) CreateCard(input ports.KBCardInput) (*ports.KBCardRecord, error) {
	projectRoot := GetProjectRootForKB()
	filePath, err := GetKBFilePath(projectRoot, input.Category, input.Subcategory)
	if err != nil {
		return nil, fmt.Errorf("resolve KB file path: %w", err)
	}
	prefix := ResolvePrefix(input.Category, input.Subcategory)
	num, err := NextCardNumber(filePath, prefix)
	if err != nil {
		return nil, fmt.Errorf("allocate card number: %w", err)
	}
	cardID := fmt.Sprintf("%s%03d", prefix, num)
	occurredAt := time.Now().Format("2006-01-02")

	markdown := RenderCardMarkdown(
		cardID,
		input.Title,
		occurredAt,
		input.Context,
		input.Problem,
		input.Cause,
		input.Solution,
		input.Prevention,
		input.Refs,
		input.ViolationPatterns,
		input.Category,
	)
	if err := AppendCardToFile(filePath, markdown); err != nil {
		return nil, fmt.Errorf("save KB card: %w", err)
	}

	return &ports.KBCardRecord{
		CardID:     cardID,
		Prefix:     prefix,
		Number:     num,
		Title:      input.Title,
		OccurredAt: occurredAt,
		FilePath:   filePath,
		Kind:       "category",
	}, nil
}

// ListCards returns cards matching the filter conditions.
//
// Adds Subcategory/Prefix/Last/Limit on top of FilterCards
// (category/keyword/since), applying post-filter + sort + truncate.
func (KBFSStore) ListCards(filter ports.KBListFilter) ([]ports.KBCardRecord, error) {
	projectRoot := GetProjectRootForKB()
	all, err := ScanAllCards(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("ScanAllCards: %w", err)
	}
	filtered := FilterCards(all, projectRoot, filter.Category, filter.Keyword, filter.Since)

	// Extra filters (subcategory/prefix).
	if filter.Subcategory != "" || filter.Prefix != "" {
		kept := filtered[:0:0]
		for _, c := range filtered {
			if filter.Subcategory != "" {
				// FilePath filename (without extension) must equal subcategory.
				base := filepath.Base(c.FilePath)
				base = strings.TrimSuffix(base, ".md")
				if base != filter.Subcategory {
					continue
				}
			}
			if filter.Prefix != "" && c.Prefix != filter.Prefix {
				continue
			}
			kept = append(kept, c)
		}
		filtered = kept
	}

	// --last (time-desc) or --limit (truncate ignoring sort).
	if filter.Last > 0 {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].OccurredAt > filtered[j].OccurredAt
		})
		if len(filtered) > filter.Last {
			filtered = filtered[:filter.Last]
		}
	} else if filter.Limit > 0 && len(filtered) > filter.Limit {
		filtered = filtered[:filter.Limit]
	}

	result := make([]ports.KBCardRecord, 0, len(filtered))
	for _, c := range filtered {
		result = append(result, ports.KBCardRecord{
			CardID:     c.CardID,
			Prefix:     c.Prefix,
			Number:     c.Number,
			Title:      c.Title,
			OccurredAt: c.OccurredAt,
			Context:    c.Context,
			FilePath:   c.FilePath,
			Kind:       c.Kind,
		})
	}
	return result, nil
}

// SearchCards is the full-text search backing the kb search subcommand.
// Searches title and body. matchAll=true requires every keyword,
// matchAll=false requires at least one.
// Returns: KBCardRecord + matched_snippet (~50 chars around the first body match).
func (KBFSStore) SearchCards(keywords []string, matchAll bool, filter ports.KBListFilter) ([]KBCardSearchResult, error) {
	projectRoot := GetProjectRootForKB()
	all, err := ScanAllCards(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("ScanAllCards: %w", err)
	}
	// Apply category/since/prefix/subcategory as common filters first.
	filtered := FilterCards(all, projectRoot, filter.Category, "", filter.Since)

	var results []KBCardSearchResult
	for _, c := range filtered {
		if filter.Subcategory != "" {
			base := filepath.Base(c.FilePath)
			base = strings.TrimSuffix(base, ".md")
			if base != filter.Subcategory {
				continue
			}
		}
		if filter.Prefix != "" && c.Prefix != filter.Prefix {
			continue
		}

		// Title + body full-text.
		haystack := c.Title + "\n" + c.Context
		if body, bErr := os.ReadFile(c.FilePath); bErr == nil {
			haystack += "\n" + string(body)
		}
		haystackLower := strings.ToLower(haystack)

		matched := 0
		var firstMatch string
		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			idx := strings.Index(haystackLower, kwLower)
			if idx >= 0 {
				matched++
				if firstMatch == "" {
					start := idx - 40
					if start < 0 {
						start = 0
					}
					end := idx + len(kw) + 40
					if end > len(haystack) {
						end = len(haystack)
					}
					firstMatch = strings.ReplaceAll(haystack[start:end], "\n", " ")
				}
			}
		}

		ok := false
		if matchAll {
			ok = matched == len(keywords)
		} else {
			ok = matched > 0
		}
		if !ok {
			continue
		}
		results = append(results, KBCardSearchResult{
			Card:    c,
			Snippet: firstMatch,
			Matched: matched,
		})
	}

	// Sort (OccurredAt desc) + limit/last.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Card.OccurredAt > results[j].Card.OccurredAt
	})
	if filter.Last > 0 && len(results) > filter.Last {
		results = results[:filter.Last]
	} else if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}
	return results, nil
}

// KBCardSearchResult — return type for kb search.
type KBCardSearchResult struct {
	Card    CardHeader `json:"-"`
	Snippet string     `json:"matched_snippet"`
	Matched int        `json:"matched_keywords"`
}

// RebuildIndex regenerates docs/07-knowledge/INDEX.md.
func (KBFSStore) RebuildIndex() (*ports.KBRebuildResult, error) {
	projectRoot := GetProjectRootForKB()
	result, err := RecountIndex(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("RecountIndex: %w", err)
	}
	return &ports.KBRebuildResult{
		TotalCards:    result.TotalCards,
		FilesScanned:  result.FilesScanned,
		IndexPath:     GetIndexPath(projectRoot),
		WriteOccurred: len(result.FilesUpdated) > 0 || len(result.AddedToIndex) > 0,
	}, nil
}

// Compile-time check — KBFSStore satisfies the ports.KBStore contract.
var _ ports.KBStore = (*KBFSStore)(nil)
