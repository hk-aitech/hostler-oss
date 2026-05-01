// cmd/gen-schema
//
// JSON Schema generator triggered automatically by `go generate ./pkg/config/...`.
// Uses google/jsonschema-go's reflect-based schema generator to convert
// config.ProjectConfig into a JSON Schema Draft 2020-12 document and writes it
// to {repo_root}/schemas/project-config/v2.0.0.schema.json.
//
// - idempotent: two consecutive runs leave schemas/ unchanged (pretty-print JSON ordering guaranteed)
// - CI drift guard: `go generate ./... && git diff --exit-code schemas/`
//
// Invocation: `go run ./cmd/gen-schema` (workspace root = cli).
// The //go:generate directive in types.go uses a path relative to pkg/config.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/config"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/fileutil"
	pkglog "github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/log"
)

// logger is the gen-schema-specific slog-based logger.
var logger = pkglog.NewDefault()

// fatalf logs at Error level and exits the process. Replaces stdlib log.Fatalf.
func fatalf(format string, args ...any) {
	logger.Error(fmt.Sprintf("[gen-schema] "+format, args...))
	os.Exit(1)
}

// schemaVersion is the current schema version. Distinct from
// ProjectConfig.SchemaVersion. Bump to v2.1.0, v3.0.0, etc. on BREAKING changes
// and handle the migration in the migration runner.
const schemaVersion = "v2.0.0"

// schemaID is the public URL used as $id. Actual hosting is out of scope here,
// but a consistent URL is published so that VSCode / JetBrains pick it up.
const schemaID = "https://hostler.dev/schemas/project-config/" + schemaVersion + ".json"

// jsonIndent controls the indentation of the generated schema. Pinned to
// guarantee idempotency.
const jsonIndent = "  "

// outputRelPath is the generated file path relative to the repo root.
// Unified folder convention: emitted under the primary configuration folder
// (`.hstl/schemas/`). The repo-root `schemas/` directory is a legacy layout
// artefact and gen-schema no longer writes there.
var outputRelPath = filepath.Join(".hstl", "schemas", "project-config", schemaVersion+".schema.json")

func main() {
	// Use pkg/log (slog-based) instead of stdlib log. The "[gen-schema] "
	// prefix is added at fatalf / logger call sites.

	// 1. Generate JSON Schema from ProjectConfig (includes enum injection).
	// Use the single entry point SchemaForProjectConfig so that gen-schema and
	// the runtime validator share the same schema.
	schema, err := config.SchemaForProjectConfig()
	if err != nil {
		fatalf("schema generation failed: %v", err)
	}

	// 2. Inject metadata ($id, $schema, title, description).
	schema.ID = schemaID
	schema.Schema = "https://json-schema.org/draft/2020-12/schema"
	schema.Title = "hostler project-config.yaml"
	schema.Description = "Official schema for project-config.yaml, the per-project configuration managed by the hostler plugin. Auto-generated from types.go (Go SSOT)."

	// 3. Resolve the output path (relative to cli/).
	outPath, err := resolveOutputPath()
	if err != nil {
		fatalf("failed to resolve output path: %v", err)
	}

	// 4. Serialize to JSON (idempotent: indent + sorted keys).
	buf, err := json.MarshalIndent(schema, "", jsonIndent)
	if err != nil {
		fatalf("JSON serialization failed: %v", err)
	}
	buf = append(buf, '\n') // POSIX EOL

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(outPath, buf, 0o644); err != nil {
		fatalf("failed to write file: %v", err)
	}

	fmt.Printf("%s (%d bytes)\n", outPath, len(buf))
}

// resolveOutputPath returns the absolute path of
// schemas/project-config/v2.0.0.schema.json under the repo root.
//
// findRepoRoot logic lives in pkg/fileutil.FindRepoRoot. The shared helper
// covers both the monorepo (packages/hostler-cli/) and the legacy
// (cli/) layouts.
func resolveOutputPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := fileutil.FindRepoRoot(cwd)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, outputRelPath), nil
}
