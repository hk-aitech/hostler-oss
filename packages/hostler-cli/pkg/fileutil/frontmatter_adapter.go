// Package fileutil — FrontmatterCodec adapter.
//
// ADR-001 §4.1 Phase C. Wraps the existing `parseFrontmatter` /
// `dumpFrontmatter` / `UpdateTaskFrontmatter` so the result satisfies
// ports.FrontmatterCodec. In Phase D, services accept the adapter via
// their constructor and route calls through it.
package fileutil

import "github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"

// YAMLCodec is the stateless YAML frontmatter codec adapter.
type YAMLCodec struct{}

// NewYAMLCodec constructs the default YAML codec.
func NewYAMLCodec() *YAMLCodec { return &YAMLCodec{} }

// Decode delegates to parseFrontmatter.
func (YAMLCodec) Decode(content []byte) (map[string]any, string, error) {
	return parseFrontmatter(content)
}

// Encode delegates to dumpFrontmatter.
func (YAMLCodec) Encode(fm map[string]any, body string) ([]byte, error) {
	s, err := dumpFrontmatter(fm, body)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// UpdateField updates a single frontmatter field in content.
// In-memory only — use when no file path is involved. For file-based
// updates prefer the existing `UpdateTaskFrontmatter(filePath, fields)`.
func (c YAMLCodec) UpdateField(content []byte, field string, value any) ([]byte, error) {
	fm, body, err := c.Decode(content)
	if err != nil {
		return nil, err
	}
	fm[field] = value
	return c.Encode(fm, body)
}

// Compile-time check — YAMLCodec satisfies the ports.FrontmatterCodec contract.
var _ ports.FrontmatterCodec = (*YAMLCodec)(nil)

// FSFrontmatterFile is the file-path-based frontmatter operation adapter.
// Satisfies internal/app.FrontmatterFileOps and delegates to the existing
// ReadTaskFrontmatter / UpdateTaskFrontmatter globals.
type FSFrontmatterFile struct{}

// NewFSFrontmatterFile constructs the file-based FM adapter.
func NewFSFrontmatterFile() *FSFrontmatterFile { return &FSFrontmatterFile{} }

// ReadTask delegates to ReadTaskFrontmatter.
func (FSFrontmatterFile) ReadTask(filePath string) (map[string]any, error) {
	return ReadTaskFrontmatter(filePath)
}

// UpdateTask delegates to UpdateTaskFrontmatter.
func (FSFrontmatterFile) UpdateTask(filePath string, fields map[string]any) error {
	return UpdateTaskFrontmatter(filePath, fields)
}
