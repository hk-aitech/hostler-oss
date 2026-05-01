package domain

import "testing"

// T835 ADR-001 A2 — KBCard invariant unit tests.

func TestKBCard_IsValid_True(t *testing.T) {
	c := KBCard{ID: "M001", Title: "test", Category: KBCategoryMistakes}
	if !c.IsValid() {
		t.Error("a fully populated card must report IsValid true")
	}
}

func TestKBCard_IsValid_MissingID(t *testing.T) {
	c := KBCard{Title: "test", Category: KBCategoryMistakes}
	if c.IsValid() {
		t.Error("a missing ID must report IsValid false")
	}
}

func TestKBCard_IsValid_MissingTitle(t *testing.T) {
	c := KBCard{ID: "M001", Category: KBCategoryMistakes}
	if c.IsValid() {
		t.Error("a missing Title must report IsValid false")
	}
}

func TestKBCard_IsValid_MissingCategory(t *testing.T) {
	c := KBCard{ID: "M001", Title: "test"}
	if c.IsValid() {
		t.Error("a missing Category must report IsValid false")
	}
}

func TestKBCard_Validate_NoError(t *testing.T) {
	c := KBCard{ID: "A001", Title: "architecture", Category: KBCategoryArchitecture}
	if err := c.Validate(); err != nil {
		t.Errorf("fully populated card: err=%v", err)
	}
}

func TestKBCard_Validate_MissingIDMessage(t *testing.T) {
	c := KBCard{Title: "t", Category: KBCategoryOperations}
	err := c.Validate()
	if err == nil {
		t.Fatal("missing ID must produce an error")
	}
	if !contains(err.Error(), "ID") {
		t.Errorf("error message must mention ID: %s", err.Error())
	}
}

func TestKBCard_Validate_MissingTitleMessage(t *testing.T) {
	c := KBCard{ID: "M002", Category: KBCategoryMistakes}
	err := c.Validate()
	if err == nil {
		t.Fatal("missing Title must produce an error")
	}
	if !contains(err.Error(), "Title") || !contains(err.Error(), "M002") {
		t.Errorf("error message must mention Title and ID: %s", err.Error())
	}
}

func TestKBCard_Validate_MissingCategoryMessage(t *testing.T) {
	c := KBCard{ID: "O003", Title: "operations"}
	err := c.Validate()
	if err == nil {
		t.Fatal("missing Category must produce an error")
	}
	if !contains(err.Error(), "Category") {
		t.Errorf("error message must mention Category: %s", err.Error())
	}
}
