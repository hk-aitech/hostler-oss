// Package ports — KBStore port.
//
// hstl-specific port. The KB (Knowledge Base) is the knowledge asset
// under `docs/07-knowledge/`. It must travel with Hostler, so declaring
// the port up front leaves room for swapping adapters later.
//
// Today `cli/pkg/kb/` performs file-based CRUD and INDEX rebuilds. This
// port is an extracted declaration that the CLI `hstl kb` commands can
// route through later.
//
// Verification pattern:
//
//	var _ ports.KBStore = (*KBFSStore)(nil)
//
// HMAC seal/verify cooperates with a separate port (HMACSigner) and is out of
// scope here.
package ports

// KBCardRecord is the metadata for a single KB card.
// 1:1 with fileutil's CardHeader — uses domain-neutral names at the port layer.
type KBCardRecord struct {
	CardID     string // "M27" or "KB-LS-CATALOG-2026-04"
	Prefix     string // "M" (category) or "KB" (single-file)
	Number     int    // sequence number for category cards; 0 for single-file
	Title      string
	OccurredAt string
	Context    string
	FilePath   string
	Kind       string // "category" | "single_file"
}

// KBListFilter is the parameter struct for KBStore.ListCards.
// The zero value means "no filter" — return everything.
//
// Extended fields: Subcategory / Prefix / Last / Limit.
// Semantic SSOT: docs/08-references/standards/cli-filter-schema.md.
type KBListFilter struct {
	Category    string // "mistakes" / "architecture" / "operations" etc.
	Keyword     string // title substring match (list --keyword). Full-text search uses the search subcommand.
	Since       string // cards on or after "YYYY-MM-DD"
	Subcategory string // file-name based (e.g. "persistence", "methodology")
	Prefix      string // card ID prefix (A|M|O|I)
	Last        int    // top N entries by OccurredAt time-desc (forces sorting)
	Limit       int    // truncation (sort-agnostic)
}

// KBCardInput is the input set for KBStore.CreateCard.
// Accepts all 10 flags of CLI `hstl kb create`. Category / Subcategory
// / Title / Problem / Solution are required. The rest are optional —
// zero values are omitted from the card sections.
type KBCardInput struct {
	Category          string
	Subcategory       string
	Title             string
	Problem           string
	Solution          string
	Context           string
	Cause             string
	Prevention        string
	Refs              []string
	ViolationPatterns []string
}

// KBRebuildResult summarises a KBStore.RebuildIndex call.
type KBRebuildResult struct {
	TotalCards    int
	FilesScanned  int
	IndexPath     string
	WriteOccurred bool
}

// KBStore is the port for KB card CRUD and aggregation.
// Currently implemented by `pkg/kb`.
type KBStore interface {
	// CreateCard appends a new card to the matching category/subcategory file.
	// Returns: the issued CardID and the actual storage path.
	// KBCardInput accepts the full argument set.
	CreateCard(input KBCardInput) (*KBCardRecord, error)

	// ListCards returns the cards matching filter.
	// A zero-value filter returns every card.
	ListCards(filter KBListFilter) ([]KBCardRecord, error)

	// RebuildIndex regenerates docs/07-knowledge/INDEX.md.
	// Updates per-category counts and the summary section.
	RebuildIndex() (*KBRebuildResult, error)
}
