// Package ports — FrontmatterCodec port.
//
// Isolates YAML frontmatter parsing and serialization behind a single
// interface. Today similar logic is duplicated in the Task / Sprint /
// KB / Mailbox packages — this declaration leads to a single shared
// adapter in a follow-up refactor.
//
// Verification pattern:
//
//	var _ ports.FrontmatterCodec = (*YAMLCodec)(nil)
//
// The first adapter is YAMLCodec, which wraps fileutil's existing
// parseFrontmatter / dumpFrontmatter helpers. Duplicate implementations
// in other packages will be consolidated in a later refactor.
package ports

// FrontmatterCodec is the port for parsing and serializing `---` YAML
// frontmatter.
// ADR-001 §4.1 — shared across Task / Sprint / KB / Mailbox files.
//
// Implementation contract:
//   - Decode: split `---`\nfields\n`---`\nbody into (frontmatter map, body).
//     If no frontmatter is present, returns (empty map, original content)
//     without an error.
//   - Encode: frontmatter map + body → complete markdown string.
//   - UpdateField: keep content intact while updating a single field.
//     Supports add and replace. Deletion is expressed as value=nil or via a
//     separate method.
type FrontmatterCodec interface {
	// Decode splits markdown content into a frontmatter map and body.
	// When no frontmatter is present, returns (empty map, original content)
	// without an error.
	Decode(content []byte) (frontmatter map[string]any, body string, err error)

	// Encode serializes frontmatter + body into a complete markdown document.
	Encode(frontmatter map[string]any, body string) ([]byte, error)

	// UpdateField updates a single frontmatter field within content.
	// Adds the field when missing. Body and other fields stay intact.
	UpdateField(content []byte, field string, value any) ([]byte, error)
}
