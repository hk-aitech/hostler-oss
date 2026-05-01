package domain

import "fmt"

// KBCategory enumerates KB categories.
type KBCategory string

const (
	KBCategoryMistakes     KBCategory = "mistakes"
	KBCategoryOperations   KBCategory = "operations"
	KBCategoryArchitecture KBCategory = "architecture"
	KBCategoryAPI          KBCategory = "api"
	KBCategoryDomain       KBCategory = "domain"
	KBCategoryMigration    KBCategory = "migration"
)

// KBCard — domain type for a KB knowledge card.
type KBCard struct {
	ID          string
	Title       string
	Category    KBCategory
	Subcategory string
	Problem     string
	Solution    string
	Context     string
	Cause       string
	Prevention  string
	CreatedAt   string
	FilePath    string
}

// IsValid reports whether every required field (ID, Title, Category) is populated.
func (c KBCard) IsValid() bool {
	return c.ID != "" && c.Title != "" && c.Category != ""
}

// Validate is the error-returning variant of IsValid. Returns the specific
// missing field by name so pkg/kb can compose user-facing messages.
func (c KBCard) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("KBCard.ID is empty")
	}
	if c.Title == "" {
		return fmt.Errorf("KBCard.Title is empty (ID=%s)", c.ID)
	}
	if c.Category == "" {
		return fmt.Errorf("KBCard.Category is empty (ID=%s)", c.ID)
	}
	return nil
}
