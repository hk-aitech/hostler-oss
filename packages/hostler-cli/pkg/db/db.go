// Package db manages the SQLite connection and schema initialisation.
// Storage location: ~/.hostler/data/{project_key}/hstl.db. WAL mode keeps
// concurrent worktree access safe.
package db

import (
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/envalias"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/harnessdefaults"
	_ "modernc.org/sqlite"
)

// projectConfigDirs is the search order for project-config.yaml.
// brand.ProjectDirName is the canonical primary location; any legacy
// alternates listed in brand.LegacyProjectDirNames are consulted as
// read-only fallbacks.
var projectConfigDirs = append([]string{brand.ProjectDirName}, brand.LegacyProjectDirNames...)

//go:embed schemas/tables.sql
var tablesDDL string

// Global DB instance.
var _db *sql.DB

const (
	// maxOpenConnsDefault — maximum concurrent connections in WAL mode.
	// Override with HSTL_DB_MAX_CONNS. modernc.org/sqlite has an issue
	// where the busy handler does not fire on concurrent in-process
	// writes, so the default stays at 1. Parallel readers (e.g.
	// project_status) should use NewReadOnlyConn() for a separate
	// connection.
	maxOpenConnsDefault = 1
	// busyTimeoutDefault — SQLite busy_timeout in ms. Override with
	// HSTL_DB_BUSY_TIMEOUT_MS.
	busyTimeoutDefault = 5000
)

func getMaxOpenConns() int {
	if s := envalias.Lookup("DB_MAX_CONNS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}
	return maxOpenConnsDefault
}

func getBusyTimeout() int {
	if s := envalias.Lookup("DB_BUSY_TIMEOUT_MS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}
	return busyTimeoutDefault
}

// buildDSN returns a SQLite DSN that includes _pragma parameters so each
// new pooled connection re-applies them automatically. db.Exec("PRAGMA
// ...") only affects the connection it ran on, so settings are lost when
// the pool recreates a connection.
func buildDSN(dbPath string) string {
	return fmt.Sprintf(
		"%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(%d)",
		dbPath, getBusyTimeout(),
	)
}

// DetectProjectKey resolves the project key.
//
// Priority order (drift-prevention update):
//  1. Environment variable HSTL_PROJECT (explicit override, test isolation, etc.)
//  2. project.key in project-config.yaml (stable SSOT)
//  3. git remote origin URL → first 8 hex chars of SHA256 (new project without yaml)
//  4. Current directory basename → first 8 hex chars of SHA256 (final fallback)
//
// Before the YAML step existed, changing the git remote URL silently
// shifted the DB path and made existing data look "lost". With YAML as
// SSOT, remote changes no longer affect resolution.
//
// When falling through to step 3 or 4, the dynamically computed key
// shifts whenever a binary version changes the hash algorithm/format,
// causing silent DB relocation. To prevent that, the computed key is
// pinned into project-config.yaml so subsequent runs resolve at step 2.
func DetectProjectKey() string {
	// 1. Environment variable wins.
	if envKey := strings.TrimSpace(envalias.Lookup("PROJECT")); envKey != "" {
		return envKey
	}

	// 2. project.key from project-config.yaml.
	if key := readProjectKeyFromYAML(); key != "" {
		return key
	}

	// 3. Hash of git remote origin URL.
	var key string
	if url := gitRemoteURL(); url != "" {
		key = hashStr(url)
	} else {
		// 4. Fallback: hash of current directory basename.
		cwd, _ := os.Getwd()
		key = hashStr(filepath.Base(cwd))
	}

	// Persist the fallback-computed key so subsequent runs stay stable.
	// Write failures are silent (the build/test environment may not have
	// write permission — fall back to dynamic computation).
	pinProjectKeyToYAML(key)

	return key
}

// pinProjectKeyToYAML persists the computed projectKey to the
// `project.key` field of `{ProjectDirName}/project-config.yaml`.
// If the YAML already exists with a key set, this is a no-op. The
// defensive write prevents DB loss caused by hash-format drift across
// binary versions.
//
// Minimal contract:
//   - No project root, or write failure → silent no-op.
//   - Existing file (canonical or legacy path) → leave alone (idempotent).
//   - No existing file → create a minimal yaml at the canonical path
//     (brand.ProjectDirName).
//
// brand.ProjectDirName is canonical; any path listed in
// brand.LegacyProjectDirNames is a read-only fallback. If any of them
// already holds the file, this function is a no-op (avoids duplicates).
func pinProjectKeyToYAML(key string) {
	if key == "" {
		return
	}
	root := strings.TrimSpace(envalias.Lookup("PROJECT_ROOT"))
	if root == "" {
		// Detect the root via git rev-parse (importing pkg/fileutil here
		// would create a circular dependency, so the logic is inlined).
		out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err == nil {
			root = strings.TrimSpace(string(out))
		}
	}
	if root == "" {
		if cwd, err := os.Getwd(); err == nil {
			root = cwd
		}
	}
	if root == "" {
		return
	}
	// Bail out if any canonical or legacy YAML already exists.
	for _, dir := range projectConfigDirs {
		p := filepath.Join(root, dir, "project-config.yaml")
		if _, err := os.Stat(p); err == nil {
			return
		}
	}
	// Create a fresh file at the canonical path.
	yamlDir := filepath.Join(root, brand.ProjectDirName)
	yamlPath := filepath.Join(yamlDir, "project-config.yaml")
	if err := os.MkdirAll(yamlDir, 0o755); err != nil {
		return
	}
	content := fmt.Sprintf("# Auto-generated to prevent DB-path drift across binary upgrades.\nproject:\n  key: %q\n", key)
	_ = os.WriteFile(yamlPath, []byte(content), 0o644)
}

// readProjectKeyFromYAML extracts project.key from project-config.yaml.
// To avoid a yaml.v3 dependency, it uses a simple line scan.
//
// Order of operations:
//  1. Walk upward from the current directory looking for
//     project-config.yaml under the canonical / legacy directories.
//  2. Once found, perform a line-based parse — return the value of
//     `key:` inside the `project:` block.
//  3. If the file is missing, parsing fails, or no key is present,
//     return an empty string (the caller falls through to the next step).
//
// Avoids importing pkg/config to keep the dependency graph acyclic.
func readProjectKeyFromYAML() string {
	path := findProjectYAML()
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return parseProjectKey(string(data))
}

// findProjectYAML walks upward from the current directory looking for
// project-config.yaml. brand.ProjectDirName is preferred; entries in
// brand.LegacyProjectDirNames are checked as fallbacks. Returns an empty
// string if `/` is reached without finding any.
func findProjectYAML() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < maxParentSearchDepth; i++ {
		for _, d := range projectConfigDirs {
			p := filepath.Join(dir, d, "project-config.yaml")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}

// maxParentSearchDepth caps the upward search in findProjectYAML — avoids
// infinite loops while remaining a practical bound.
const maxParentSearchDepth = 20

// parseProjectKey extracts the `key:` value from the `project:` block of
// a YAML string. Both 2-space and 4-space nested indents are accepted.
// Comments are ignored.
func parseProjectKey(yaml string) string {
	lines := strings.Split(yaml, "\n")
	inProject := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Detect a top-level key (no leading indent).
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			if strings.HasPrefix(trimmed, "project:") {
				inProject = true
				continue
			}
			// Entered a different top-level key → exit the project block.
			inProject = false
			continue
		}
		if !inProject {
			continue
		}
		// Inside the project block — look for "key: value".
		if strings.HasPrefix(trimmed, "key:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "key:"))
			// Strip surrounding quotes.
			val = strings.Trim(val, `"'`)
			return val
		}
	}
	return ""
}

// hashStr returns the first 8 hex chars of SHA256(s).
func hashStr(s string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return fmt.Sprintf("%x", h[:4])
}

// gitRemoteURL returns the git remote origin URL.
// Returns an empty string on failure.
func gitRemoteURL() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GetDBPath returns the SQLite DB path keyed by project.
// Path layout: ~/.hostler/data/{project_key}/hstl.db.
//
// HSTL_DB_PATH overrides the default location.
func GetDBPath() string {
	if override := strings.TrimSpace(envalias.Lookup("DB_PATH")); override != "" {
		return override
	}
	projectKey := DetectProjectKey()
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hostler", "data", projectKey, "hstl.db")
}

// InitDB creates the DB directory, opens the SQLite file, configures WAL
// mode, and applies the schema. Idempotent for already-initialised
// databases (CREATE TABLE IF NOT EXISTS).
func InitDB() error {
	dbPath := GetDBPath()

	// Create the directory tree.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("failed to create DB directory: %w", err)
	}

	// DSN _pragma sets WAL + foreign_keys + busy_timeout — preserved
	// across pool re-creation.
	db, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		return fmt.Errorf("SQLite open failed: %w", err)
	}
	db.SetMaxOpenConns(getMaxOpenConns())

	// Apply the schema (idempotent).
	if _, err := db.Exec(tablesDDL); err != nil {
		db.Close()
		return fmt.Errorf("schema apply failed: %w", err)
	}

	// Idempotent ADD COLUMN for tasks.work_ticket.
	if err := ensureColumn(db, "tasks", "work_ticket", "TEXT"); err != nil {
		db.Close()
		return fmt.Errorf("failed to ensure tasks.work_ticket column: %w", err)
	}

	// Idempotent ADD COLUMN for the audit_events hash chain.
	if err := ensureColumn(db, "audit_events", "prev_hash", "TEXT"); err != nil {
		db.Close()
		return fmt.Errorf("failed to ensure audit_events.prev_hash column: %w", err)
	}
	if err := ensureColumn(db, "audit_events", "event_hash", "TEXT"); err != nil {
		db.Close()
		return fmt.Errorf("failed to ensure audit_events.event_hash column: %w", err)
	}

	// Load default Harness templates.
	if err := loadHarnessDefaults(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to load harness defaults: %w", err)
	}

	_db = db
	return nil
}

// loadHarnessDefaults loads default templates from harness_defaults.json
// into the harness_templates table. Templates refresh on every start
// (INSERT OR REPLACE).
func loadHarnessDefaults(db *sql.DB) error {
	var defaults map[string]json.RawMessage
	if err := json.Unmarshal(harnessdefaults.DefaultsJSON(), &defaults); err != nil {
		return fmt.Errorf("failed to parse harness_defaults.json: %w", err)
	}

	for templateKey, items := range defaults {
		configJSON := string(items)
		_, err := db.Exec(
			"INSERT OR REPLACE INTO harness_templates (template_key, config) VALUES (?, ?)",
			templateKey, configJSON,
		)
		if err != nil {
			return fmt.Errorf("harness_template insert failed (key=%s): %w", templateKey, err)
		}
	}
	return nil
}

// GetDB returns the initialised global DB instance.
// InitDB() must have been called first.
func GetDB() *sql.DB {
	return _db
}

// LoadSprintPhaseNames returns the sprint:default phase names from
// harness_defaults.json, in order. Sprint Phase SSOT.
//
// Deprecated: call pkg/harnessdefaults.LoadSprintPhaseNames() directly.
// Retained for backward compatibility.
func LoadSprintPhaseNames() ([]string, error) {
	return harnessdefaults.LoadSprintPhaseNames()
}

// NewConnection opens a fresh WAL-mode SQLite connection. The caller is
// responsible for Close().
func NewConnection() (*sql.DB, error) {
	dbPath := GetDBPath()
	// DSN _pragma applies the PRAGMAs — preserved across pool recreation.
	db, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("SQLite open failed: %w", err)
	}
	db.SetMaxOpenConns(getMaxOpenConns())
	return db, nil
}

// Close closes the global DB connection.
func Close() {
	if _db != nil {
		_db.Close()
		_db = nil
	}
}

// NowUTC returns a UTC timestamp string suitable for SQLite storage.
func NowUTC() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

// ensureColumn idempotently ADD COLUMNs into a SQLite table when the
// column is absent. SQLite has no ADD COLUMN IF NOT EXISTS, so a
// table_info lookup substitutes for it.
func ensureColumn(db *sql.DB, table, column, typeDecl string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, typeDecl))
	return err
}
