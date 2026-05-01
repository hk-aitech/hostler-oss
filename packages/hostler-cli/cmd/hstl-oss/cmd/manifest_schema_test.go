package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --schema and persistent-flag recursion tests.

// TestBuildLeafSchema_Structure builds a LeafSchema from a single cobra
// command and verifies it follows the draft-07 shape (type/properties/required).
func TestBuildLeafSchema_Structure(t *testing.T) {
	c := &cobra.Command{
		Use:   "start <id>",
		Short: "Test command",
		Args:  cobra.ExactArgs(1),
	}
	c.Flags().Bool("with-ceremony", false, "--with-ceremony flag")
	c.Flags().String("reason", "", "reason")
	_ = c.MarkFlagRequired("reason")

	leaf := buildLeafSchema(c)

	if leaf.Name == "" {
		t.Errorf("Name is empty")
	}
	if leaf.InputSchema == nil {
		t.Fatalf("InputSchema is nil")
	}
	if leaf.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", leaf.InputSchema.Type, "object")
	}
	if _, ok := leaf.InputSchema.Properties["with-ceremony"]; !ok {
		t.Errorf("with-ceremony missing from Properties")
	}
	if leaf.InputSchema.Properties["with-ceremony"].Type != "boolean" {
		t.Errorf("with-ceremony type = %q, want boolean", leaf.InputSchema.Properties["with-ceremony"].Type)
	}

	// MarkFlagRequired is Annotations-based in cobra — reason should be in required.
	foundReason := false
	for _, r := range leaf.InputSchema.Required {
		if r == "reason" {
			foundReason = true
			break
		}
	}
	if !foundReason {
		t.Errorf("reason missing from required: %v", leaf.InputSchema.Required)
	}

	if leaf.OutputSchema == nil {
		t.Fatalf("OutputSchema is nil")
	}
	if leaf.OutputSchema.Properties["status"].Type != "string" {
		t.Errorf("status type mismatch")
	}
}

// TestBuildLeafSchema_FlagTypeMapping verifies the six cobra Flag → JSON
// Schema type mappings (manifest-schema-contract.md §4).
func TestBuildLeafSchema_FlagTypeMapping(t *testing.T) {
	c := &cobra.Command{Use: "test"}
	c.Flags().String("s", "", "string")
	c.Flags().Bool("b", false, "bool")
	c.Flags().Int("i", 0, "int")
	c.Flags().Float64("f", 0, "float")
	c.Flags().StringSlice("ss", nil, "string slice")
	c.Flags().Duration("d", 0, "duration")

	leaf := buildLeafSchema(c)
	tests := []struct {
		flag     string
		wantType string
	}{
		{"s", "string"},
		{"b", "boolean"},
		{"i", "integer"},
		{"f", "number"},
		{"ss", "array"},
		{"d", "string"}, // duration -> string + format: duration
	}
	for _, tt := range tests {
		prop, ok := leaf.InputSchema.Properties[tt.flag]
		if !ok {
			t.Errorf("flag %q not found in properties", tt.flag)
			continue
		}
		if prop.Type != tt.wantType {
			t.Errorf("flag %q type = %q, want %q", tt.flag, prop.Type, tt.wantType)
		}
	}
}

// TestLeafSchema_JSON_HasSchema verifies that JSON serialisation of a leaf
// (when invoked standalone) includes the `$schema` field — required for
// draft-07 validation.
func TestLeafSchema_JSON_HasSchema(t *testing.T) {
	c := &cobra.Command{Use: "test", Short: "desc"}
	leaf := buildLeafSchema(c)
	leaf.Schema = manifestSchemaURI // simulate standalone leaf invocation

	data, err := json.Marshal(leaf)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	schemaURI, ok := parsed["$schema"].(string)
	if !ok {
		t.Fatalf("$schema field missing or type mismatch")
	}
	if !strings.HasPrefix(schemaURI, "http://json-schema.org/draft-07") {
		t.Errorf("$schema = %q, want draft-07 URI", schemaURI)
	}
}

// TestSchemaOutput_CommandsArray verifies the SchemaOutput shape when invoked
// against a branch (title/type/commands array).
func TestSchemaOutput_CommandsArray(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	sub1 := &cobra.Command{Use: "sub1", Short: "s1"}
	sub2 := &cobra.Command{Use: "sub2", Short: "s2"}
	parent.AddCommand(sub1, sub2)

	out := SchemaOutput{
		Schema:   manifestSchemaURI,
		Title:    "hstl parent",
		Type:     "object",
		Version:  "1.0.0",
		Commands: []LeafSchema{buildLeafSchema(sub1), buildLeafSchema(sub2)},
	}

	if out.Schema != manifestSchemaURI {
		t.Errorf("$schema mismatch")
	}
	if out.Type != "object" {
		t.Errorf("Type = %q, want object", out.Type)
	}
	if len(out.Commands) != 2 {
		t.Errorf("Commands len = %d, want 2", len(out.Commands))
	}

	// Confirm Name is populated on every leaf.
	for i, cmd := range out.Commands {
		if cmd.Name == "" {
			t.Errorf("Commands[%d].Name is empty", i)
		}
	}
}

// TestCollectLeafCommands_Scope verifies that collectLeafCommands only walks
// the subtree under the given command (the foundation for persistent-flag
// scope recursion).
func TestCollectLeafCommands_Scope(t *testing.T) {
	// Tree shape: root - task - (start, complete)
	//                  \ sprint - (start)
	root := &cobra.Command{Use: "root"}
	task := &cobra.Command{Use: "task"}
	taskStart := &cobra.Command{Use: "start", Run: func(*cobra.Command, []string) {}}
	taskComplete := &cobra.Command{Use: "complete", Run: func(*cobra.Command, []string) {}}
	task.AddCommand(taskStart, taskComplete)

	sprint := &cobra.Command{Use: "sprint"}
	sprintStart := &cobra.Command{Use: "start", Run: func(*cobra.Command, []string) {}}
	sprint.AddCommand(sprintStart)

	root.AddCommand(task, sprint)

	// root scope → 3 leaves
	if got := len(collectLeafCommands(root)); got != 3 {
		t.Errorf("root scope leaf count = %d, want 3", got)
	}

	// task scope → 2 leaves
	if got := len(collectLeafCommands(task)); got != 2 {
		t.Errorf("task scope leaf count = %d, want 2", got)
	}

	// leaf scope (taskStart) → 1 leaf (itself)
	if got := len(collectLeafCommands(taskStart)); got != 1 {
		t.Errorf("taskStart scope leaf count = %d, want 1", got)
	}
}

// TestInstallManifestSchemaSupport_ArgsWrap verifies that the Args wrapper
// bypasses positional-arg validation when --manifest/--schema is set.
func TestInstallManifestSchemaSupport_ArgsWrap(t *testing.T) {
	// Command requires 2 positional args.
	c := &cobra.Command{
		Use:  "test",
		Args: cobra.ExactArgs(2),
		Run:  func(*cobra.Command, []string) {},
	}
	parent := &cobra.Command{Use: "parent"}
	parent.AddCommand(c)

	installManifestSchemaSupport(parent)

	// Flag not set: the original Args runs and must fail.
	manifestFlag = false
	schemaFlag = false
	err := c.Args(c, []string{"a"})
	if err == nil {
		t.Errorf("flag not set + insufficient args -> expected error, got nil")
	}

	// Flag set: Args bypass.
	manifestFlag = true
	err = c.Args(c, []string{"a"})
	if err != nil {
		t.Errorf("flag set -> expected args bypass, got err=%v", err)
	}

	// cleanup
	manifestFlag = false
	schemaFlag = false
}

// TestInstallManifestSchemaSupport_GroupRunE verifies that a default RunE is
// injected on group commands that don't already define one.
func TestInstallManifestSchemaSupport_GroupRunE(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	group := &cobra.Command{Use: "group"} // no RunE
	leaf := &cobra.Command{Use: "leaf", Run: func(*cobra.Command, []string) {}}
	group.AddCommand(leaf)
	parent.AddCommand(group)

	// Before injection: RunE must be nil.
	if group.RunE != nil {
		t.Errorf("pre-install: group.RunE should be nil")
	}

	installManifestSchemaSupport(parent)

	// After injection: RunE is populated.
	if group.RunE == nil {
		t.Errorf("post-install: group.RunE should be non-nil")
	}
}
